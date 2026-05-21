package ai

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"go.datum.net/datumctl/internal/ai/llm"
)

const (
	// historyWindowSize is the maximum number of messages sent to the LLM in a
	// single Chat call. Older messages are dropped from the window (not from
	// the full history slice) to avoid hitting provider token limits in long
	// interactive sessions.
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

	// IsTerminal controls spinner behaviour.
	IsTerminal bool

	// Gate is the confirmation gate for destructive tool calls.
	// If nil, StdinGate{In, ErrOut} is used when IsTerminal is true,
	// and AutoDeclineGate is used otherwise.
	Gate ConfirmGate
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
	if opts.Gate == nil {
		if opts.IsTerminal {
			opts.Gate = StdinGate{In: opts.In, Out: opts.ErrOut}
		} else {
			opts.Gate = AutoDeclineGate{ErrOut: opts.ErrOut}
		}
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

// TurnResult is the outcome of one agent turn.
type TurnResult struct {
	Response string
	Err      error
}

// RunTurn executes one full user→LLM→tools→response cycle without looping for
// more user input. Safe to call from a goroutine (e.g. a Bubbletea tea.Cmd).
// The user message is appended to the shared history before the LLM call.
func (a *Agent) RunTurn(ctx context.Context, userMessage string) TurnResult {
	a.history = append(a.history, llm.Message{Role: llm.RoleUser, Content: userMessage})

	// Collect the response into a strings.Builder instead of streaming to Out,
	// so the TUI pane can append it all at once.
	var buf strings.Builder
	savedOut := a.opts.Out
	a.opts.Out = &buf
	err := a.runOnce(ctx)
	a.opts.Out = savedOut

	return TurnResult{Response: buf.String(), Err: err}
}

// ClearHistory resets the conversation history. Used by the TUI's ctrl+l keybind.
func (a *Agent) ClearHistory() {
	a.history = nil
}

// runOnce executes one question→tool-calls→answer cycle, up to MaxIterations.
func (a *Agent) runOnce(ctx context.Context) error {
	toolDefs := a.opts.Registry.Defs()

	for iter := 0; iter < a.opts.MaxIterations; iter++ {
		spinner := NewSpinner(a.opts.ErrOut, a.opts.IsTerminal)
		spinner.Run()

		window := a.historyWindow()

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
			fmt.Fprintln(a.opts.Out)
			return nil
		}

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
// stops the spinner so streamed text begins on a clean line.
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

// executeToolCall finds the tool, handles confirmation, and runs it.
func (a *Agent) executeToolCall(ctx context.Context, tc llm.ToolCall) (string, bool) {
	tool, ok := a.opts.Registry.Find(tc.ToolName)
	if !ok {
		return fmt.Sprintf("unknown tool %q", tc.ToolName), true
	}

	if tool.RequiresConfirm {
		if !a.opts.Gate.Confirm(tc) {
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
// historyWindowSize messages.
func (a *Agent) historyWindow() []llm.Message {
	if len(a.history) <= historyWindowSize {
		return a.history
	}
	fmt.Fprintf(a.opts.ErrOut,
		"[ai] warning: conversation history (%d messages) exceeds window (%d); oldest messages dropped\n",
		len(a.history), historyWindowSize)
	return a.history[len(a.history)-historyWindowSize:]
}
