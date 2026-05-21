package ai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"go.datum.net/datumctl/internal/ai/llm"
)

// ConfirmGate decides whether a destructive tool call should proceed.
// The implementation is responsible for I/O (stdin prompt or TUI dialog).
type ConfirmGate interface {
	// Confirm displays the proposed action and returns true if approved.
	Confirm(call llm.ToolCall) bool
}

// PrintPreview writes a human-readable description of a proposed mutating
// action to w. Called before prompting for confirmation.
func PrintPreview(w io.Writer, call llm.ToolCall) {
	fmt.Fprintf(w, "\n--- Proposed action ---\n")
	fmt.Fprintf(w, "Tool:    %s\n", call.ToolName)
	b, _ := json.MarshalIndent(call.Arguments, "", "  ")
	fmt.Fprintf(w, "Details:\n%s\n", string(b))
	fmt.Fprintf(w, "-----------------------\n")
}

// StdinGate is the CLI confirmation gate. It prints a preview to out and reads
// y/n from in. Returns true only if the user types "y".
type StdinGate struct {
	In  io.Reader
	Out io.Writer
}

func (g StdinGate) Confirm(call llm.ToolCall) bool {
	PrintPreview(g.Out, call)
	fmt.Fprint(g.Out, "Apply changes? [y/N]: ")
	sc := bufio.NewScanner(g.In)
	if !sc.Scan() {
		return false
	}
	return strings.ToLower(strings.TrimSpace(sc.Text())) == "y"
}

// TUIGate is the Bubbletea confirmation gate. It sends the tool call to a
// channel for the TUI to display, then blocks waiting for the user's decision.
// Ctx is the per-turn context; cancelling it auto-declines and unblocks Confirm.
type TUIGate struct {
	// RequestCh receives confirmation requests from the agent goroutine.
	RequestCh chan<- ConfirmRequest
	// Ctx is the per-turn context. Cancel it to auto-decline any pending request.
	Ctx context.Context
}

// ConfirmRequest carries a tool call and a channel to send the decision back on.
type ConfirmRequest struct {
	Call    llm.ToolCall
	ReplyCh chan bool
}

func (g TUIGate) Confirm(call llm.ToolCall) bool {
	replyCh := make(chan bool, 1)
	select {
	case g.RequestCh <- ConfirmRequest{Call: call, ReplyCh: replyCh}:
		select {
		case reply := <-replyCh:
			return reply
		case <-g.Ctx.Done():
			return false
		}
	case <-g.Ctx.Done():
		return false
	}
}

// AutoDeclineGate always declines mutations. Used in pipe/non-terminal mode.
type AutoDeclineGate struct {
	ErrOut io.Writer
}

func (g AutoDeclineGate) Confirm(call llm.ToolCall) bool {
	fmt.Fprintf(g.ErrOut,
		"[ai] mutation skipped: %s requires interactive mode (not a terminal)\n", call.ToolName)
	return false
}
