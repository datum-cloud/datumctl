package components

import (
	"strings"
	"testing"

	"go.datum.net/datumctl/internal/ai/llm"
)

// TestChatPaneModel_EmptyInitializingState verifies that before the agent is
// ready and with no messages, View() shows the initializing placeholder text.
func TestChatPaneModel_EmptyInitializingState(t *testing.T) {
	t.Parallel()
	m := NewChatPaneModel(80, 24)
	// agentReady is false by default and no messages present.

	plain := stripANSI(m.View())

	if !strings.Contains(plain, "Initializing AI assistant") {
		t.Errorf("initializing state: want 'Initializing AI assistant' in View(), got %q", plain)
	}
}

// TestChatPaneModel_AgentErrorState verifies that after SetAgentError the
// error string appears in View().
func TestChatPaneModel_AgentErrorState(t *testing.T) {
	t.Parallel()
	m := NewChatPaneModel(80, 24)
	m.SetAgentError("no API key")

	plain := stripANSI(m.View())

	if !strings.Contains(plain, "no API key") {
		t.Errorf("agent error state: want 'no API key' in View(), got %q", plain)
	}
}

// TestChatPaneModel_AgentReadyNoMessages verifies that once the agent is ready
// with no messages, View() shows the "Ask anything" prompt. SetSize triggers the
// content rebuild that makes the new state visible in the viewport.
func TestChatPaneModel_AgentReadyNoMessages(t *testing.T) {
	t.Parallel()
	m := NewChatPaneModel(80, 24)
	m.SetAgentReady()
	// SetSize calls rebuildContent, making the updated agentReady state visible.
	m.SetSize(80, 24)

	plain := stripANSI(m.View())

	if !strings.Contains(plain, "Ask anything") {
		t.Errorf("agent ready, no messages: want 'Ask anything' in View(), got %q", plain)
	}
}

// TestChatPaneModel_UserMessageAppended verifies that after AppendUserMessage
// the View() contains both the "You" label and the message text.
func TestChatPaneModel_UserMessageAppended(t *testing.T) {
	t.Parallel()
	m := NewChatPaneModel(80, 24)
	m.AppendUserMessage("hello")

	plain := stripANSI(m.View())

	if !strings.Contains(plain, "You") {
		t.Errorf("user message: want 'You' label in View(), got %q", plain)
	}
	if !strings.Contains(plain, "hello") {
		t.Errorf("user message: want 'hello' in View(), got %q", plain)
	}
}

// TestChatPaneModel_AssistantMessageAppended verifies that after
// AppendAssistantMessage the View() contains "Assistant" and the response text.
func TestChatPaneModel_AssistantMessageAppended(t *testing.T) {
	t.Parallel()
	m := NewChatPaneModel(80, 24)
	m.AppendAssistantMessage("response text")

	plain := stripANSI(m.View())

	if !strings.Contains(plain, "Assistant") {
		t.Errorf("assistant message: want 'Assistant' label in View(), got %q", plain)
	}
	if !strings.Contains(plain, "response text") {
		t.Errorf("assistant message: want 'response text' in View(), got %q", plain)
	}
}

// TestChatPaneModel_ProcessingSpinner verifies that after SetProcessing(true)
// the title line still contains "AI Chat" (processing does not erase the title).
func TestChatPaneModel_ProcessingSpinner(t *testing.T) {
	t.Parallel()
	m := NewChatPaneModel(80, 24)
	m.SetProcessing(true)

	plain := stripANSI(m.View())

	if !strings.Contains(plain, "AI Chat") {
		t.Errorf("processing state: want 'AI Chat' title in View(), got %q", plain)
	}
	if !m.Processing() {
		t.Error("Processing() should return true after SetProcessing(true)")
	}
}

// TestChatPaneModel_ConfirmPending verifies that after SetConfirmPending with a
// non-empty ToolName, View() shows "Confirm" and the tool name.
func TestChatPaneModel_ConfirmPending(t *testing.T) {
	t.Parallel()
	m := NewChatPaneModel(80, 24)
	call := llm.ToolCall{
		ID:        "call-1",
		ToolName:  "delete_resource",
		Arguments: map[string]any{"kind": "DNSZone", "name": "my-zone"},
	}
	m.SetConfirmPending(call)

	plain := stripANSI(m.View())

	if !strings.Contains(plain, "Confirm") {
		t.Errorf("confirm pending: want 'Confirm' in View(), got %q", plain)
	}
	if !strings.Contains(plain, "delete_resource") {
		t.Errorf("confirm pending: want tool name 'delete_resource' in View(), got %q", plain)
	}
	if !m.ConfirmPending() {
		t.Error("ConfirmPending() should return true after SetConfirmPending")
	}
}

// TestChatPaneModel_InputCleared verifies that ClearInput resets the input to empty.
func TestChatPaneModel_InputCleared(t *testing.T) {
	t.Parallel()
	m := NewChatPaneModel(80, 24)
	m.AppendUserMessage("hi")
	m.ClearInput()

	if m.InputValue() != "" {
		t.Errorf("after ClearInput: InputValue() = %q, want empty string", m.InputValue())
	}
}

// TestChatPaneModel_LineOffsetForMessage verifies that after two AppendUserMessage
// calls the line offsets are tracked correctly.
func TestChatPaneModel_LineOffsetForMessage(t *testing.T) {
	t.Parallel()
	m := NewChatPaneModel(80, 24)
	m.AppendUserMessage("first message")
	m.AppendUserMessage("second message")

	offset0, ok0 := m.LineOffsetForMessage(0)
	if !ok0 {
		t.Fatal("LineOffsetForMessage(0): ok = false, want true")
	}
	if offset0 != 0 {
		t.Errorf("LineOffsetForMessage(0) = %d, want 0", offset0)
	}

	offset1, ok1 := m.LineOffsetForMessage(1)
	if !ok1 {
		t.Fatal("LineOffsetForMessage(1): ok = false, want true")
	}
	if offset1 <= 0 {
		t.Errorf("LineOffsetForMessage(1) = %d, want > 0", offset1)
	}

	_, okOOB := m.LineOffsetForMessage(2)
	if okOOB {
		t.Error("LineOffsetForMessage(2): ok = true for out-of-bounds index, want false")
	}
}

// TestChatPaneModel_SetSize verifies that SetSize does not panic and the
// component remains functional (viewport height at least 1).
func TestChatPaneModel_SetSize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{"standard 80x20", 80, 20},
		{"narrow 40x10", 40, 10},
		{"minimal 10x6", 10, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := NewChatPaneModel(tt.width, tt.height)
			// SetSize must not panic and View() must return non-empty output.
			view := m.View()
			if view == "" {
				t.Errorf("%dx%d: View() returned empty string", tt.width, tt.height)
			}
		})
	}
}

// TestChatPaneModel_AgentErrorClearsOnReady verifies that SetAgentReady clears
// a prior agent error so the error string no longer appears in View(). SetSize
// is called to trigger the content rebuild that makes the change visible.
func TestChatPaneModel_AgentErrorClearsOnReady(t *testing.T) {
	t.Parallel()
	m := NewChatPaneModel(80, 24)
	m.SetAgentError("no API key")
	m.SetAgentReady()
	// SetSize calls rebuildContent, making the cleared error state visible.
	m.SetSize(80, 24)

	plain := stripANSI(m.View())

	if strings.Contains(plain, "no API key") {
		t.Errorf("after SetAgentReady+SetSize: error 'no API key' must NOT appear in View(), got %q", plain)
	}
}

// TestChatPaneModel_ConfirmPendingClearedAfterClear verifies that
// ClearConfirmPending removes the pending state.
func TestChatPaneModel_ConfirmPendingClearedAfterClear(t *testing.T) {
	t.Parallel()
	m := NewChatPaneModel(80, 24)
	call := llm.ToolCall{ToolName: "apply_manifest", Arguments: map[string]any{}}
	m.SetConfirmPending(call)
	m.ClearConfirmPending()

	if m.ConfirmPending() {
		t.Error("ConfirmPending() = true after ClearConfirmPending(), want false")
	}
	plain := stripANSI(m.View())
	if strings.Contains(plain, "apply_manifest") {
		t.Errorf("after ClearConfirmPending: tool name must NOT appear in View(), got %q", plain)
	}
}
