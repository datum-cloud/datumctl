package ai

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"

	"go.datum.net/datumctl/internal/ai/llm"
)

const (
	// historyWindowSize is the maximum number of messages sent to the LLM in a
	// single Chat call. Older messages are dropped from the window (not from
	// the full history slice) to avoid hitting provider token limits in long
	// interactive sessions. A warning is printed to stderr when truncation occurs.
	historyWindowSize = 40
)

// AgentOptions configures an Agent.
type AgentOptions struct {
	LLM           llm.LLMClient
	Registry      *Registry
	SystemPrompt  string
	MaxIterations int

	// In/Out/ErrOut allow callers to substitute streams for testing.
	// When nil, os.Stdin/os.Stdout/os.Stderr are used.
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer

	// Interactive, when true, starts a REPL after the initial query completes.
	Interactive bool

	// IsTerminal controls spinner and pipe-mode behaviour. When false,
	// mutations are auto-declined and a clear message is shown.
	IsTerminal bool
}

// Agent runs the agentic loop.
type Agent struct {
	opts    AgentOptions
	history []llm.Message
}

// NewAgent creates an Agent from the given options.
func NewAgent(opts AgentOptions) *Agent {
	if opts.In == nil {
		opts.In = os.Stdin
	}
	if opts.Out == nil {
		opts.Out = os.Stdout
	}
	if opts.ErrOut == nil {
		opts.ErrOut = os.Stderr
	}
	if opts.MaxIterations <= 0 {
		opts.MaxIterations = 20
	}
	return &Agent{opts: opts}
}

// Run executes the agentic loop starting with initialQuery. In interactive
// mode it loops, reading subsequent queries from stdin after each response.
func (a *Agent) Run(ctx context.Context, initialQuery string) error {
	a.history = append(a.history, llm.Message{Role: llm.RoleUser, Content: initialQuery})

	for {
		if err := a.runOnce(ctx); err != nil {
			return err
		}
		if !a.opts.Interactive {
			return nil
		}
		// Read the next user input for the interactive REPL.
		fmt.Fprintf(a.opts.Out, "\n> ")
		sc := bufio.NewScanner(a.opts.In)
		if !sc.Scan() {
			return nil // EOF
		}
		line := sc.Text()
		if line == "" || line == "exit" || line == "quit" {
			return nil
		}
		a.history = append(a.history, llm.Message{Role: llm.RoleUser, Content: line})
	}
}

// runOnce executes one question→tool-calls→answer cycle, up to MaxIterations.
func (a *Agent) runOnce(ctx context.Context) error {
	toolDefs := a.opts.Registry.Defs()

	for iter := 0; iter < a.opts.MaxIterations; iter++ {
		spinner := NewSpinner(a.opts.ErrOut, a.opts.IsTerminal)
		spinner.Run()

		window := a.historyWindow()

		// For the final text response (no tool calls expected or on last-leg
		// iterations) we stream directly to Out so the user sees tokens as they
		// arrive. For intermediate tool-calling turns we collect the full
		// response first, since tool call arguments must be complete before we
		// can execute them.
		resp, err := a.opts.LLM.StreamChat(ctx, a.opts.SystemPrompt, window, toolDefs, &spinnerClearWriter{
			spinner: spinner,
			out:     a.opts.Out,
		})
		spinner.Stop()
		if err != nil {
			return fmt.Errorf("LLM error: %w", err)
		}
		a.history = append(a.history, resp)

		if len(resp.ToolCalls) == 0 {
			// Text was already streamed; just add a trailing newline.
			fmt.Fprintln(a.opts.Out)
			return nil
		}

		// Process all tool calls in the batch before the next Chat call.
		for _, tc := range resp.ToolCalls {
			result, isErr := a.executeToolCall(ctx, tc)
			a.history = append(a.history, llm.Message{
				Role: llm.RoleToolResult,
				ToolResult: &llm.ToolResult{
					CallID:  tc.ID,
					Content: result,
					IsError: isErr,
				},
			})
		}
	}

	return fmt.Errorf("agentic loop exceeded %d iterations without completing", a.opts.MaxIterations)
}

// spinnerClearWriter wraps the real output writer. On the very first Write it
// stops the spinner (clearing the spinner line) so streamed text begins on a
// clean line. Subsequent writes go straight through.
type spinnerClearWriter struct {
	spinner *Spinner
	out     io.Writer
	cleared bool
}

func (w *spinnerClearWriter) Write(p []byte) (int, error) {
	if !w.cleared {
		w.spinner.Stop()
		w.cleared = true
	}
	return w.out.Write(p)
}

// executeToolCall finds the tool, handles confirmation gates, and runs it.
// Returns the result string and whether it represents an error.
func (a *Agent) executeToolCall(ctx context.Context, tc llm.ToolCall) (string, bool) {
	tool, ok := a.opts.Registry.Find(tc.ToolName)
	if !ok {
		return fmt.Sprintf("unknown tool %q", tc.ToolName), true
	}

	if tool.RequiresConfirm {
		if !a.opts.IsTerminal {
			fmt.Fprintf(a.opts.ErrOut,
				"[ai] mutation skipped: %s requires interactive mode (not a terminal)\n", tc.ToolName)
			return `{"skipped":true,"reason":"mutations require interactive mode — run without piping to apply changes"}`, false
		}
		PrintPreview(a.opts.ErrOut, tc)
		if !Ask(a.opts.In, a.opts.ErrOut) {
			return `{"skipped":true,"reason":"user declined"}`, false
		}
	}

	result, err := tool.Execute(ctx, tc.Arguments)
	if err != nil {
		fmt.Fprintf(a.opts.ErrOut, "[ai] tool %s error: %v\n", tc.ToolName, err)
		return err.Error(), true
	}
	return result, false
}

// historyWindow returns the slice of history to send to Chat, capped at
// historyWindowSize messages. A stderr warning is printed when truncation
// occurs to signal the user that context is being dropped.
func (a *Agent) historyWindow() []llm.Message {
	if len(a.history) <= historyWindowSize {
		return a.history
	}
	fmt.Fprintf(a.opts.ErrOut,
		"[ai] warning: conversation history (%d messages) exceeds window (%d); oldest messages dropped\n",
		len(a.history), historyWindowSize)
	return a.history[len(a.history)-historyWindowSize:]
}


