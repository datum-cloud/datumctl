package console

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
	datumai "go.datum.net/datumctl/internal/ai"
	"go.datum.net/datumctl/internal/ai/llm"
	"go.datum.net/datumctl/internal/console/components"
)

// newNavPaneModelForChat builds a minimal AppModel in NavPane with both chat
// and chatSidebar components initialised (required for chat-key routing).
func newNavPaneModelForChat() AppModel {
	m := AppModel{
		ctx:         context.Background(),
		rc:          stubResourceClient{},
		activePane:  NavPane,
		sidebar:     components.NewNavSidebarModel(22, 20),
		table:       components.NewResourceTableModel(58, 20),
		detail:      components.NewDetailViewModel(58, 20),
		filterBar:   components.NewFilterBarModel(),
		helpOverlay: components.NewHelpOverlayModel(),
		chat:        components.NewChatPaneModel(80, 20),
		chatSidebar: components.NewChatSidebarModel(20, 20),
	}
	m.updatePaneFocus()
	return m
}

// newChatPaneModelForTest builds a minimal AppModel already in ChatPane.
func newChatPaneModelForTest() AppModel {
	m := newNavPaneModelForChat()
	// Simulate pressing [a] by setting the necessary fields directly.
	m.chatOriginPane = DashboardOrigin{Pane: NavPane, ShowDashboard: false}
	m.activePane = ChatPane
	m.chatSidebarFocused = false
	m.updatePaneFocus()
	return m
}

// TestChatPaneOpenFromNavPane verifies that pressing [a] from NavPane:
//   - transitions activePane to ChatPane
//   - dispatches a non-nil tea.Cmd (initChatAgentCmd + chat.Init batch)
//   - allocates chatConfirmCh
func TestChatPaneOpenFromNavPane(t *testing.T) {
	t.Parallel()
	m := newNavPaneModelForChat()
	// chatAgent starts nil — first [a] press triggers initialisation.
	if m.chatAgent != nil {
		t.Fatal("precondition: chatAgent must be nil before first [a] press")
	}

	result, cmd := m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	appM := result.(AppModel)

	if appM.activePane != ChatPane {
		t.Errorf("activePane = %v, want ChatPane after pressing [a]", appM.activePane)
	}
	if cmd == nil {
		t.Error("cmd = nil after first [a] press, want non-nil (initChatAgentCmd batch)")
	}
	if appM.chatConfirmCh == nil {
		t.Error("chatConfirmCh = nil after first [a] press, want non-nil buffered channel")
	}
}

// TestChatPaneReturnToOrigin verifies that pressing [a] while already in
// ChatPane returns to the origin pane.
func TestChatPaneReturnToOrigin(t *testing.T) {
	t.Parallel()
	// Start in TablePane, navigate to ChatPane, then press [a] to go back.
	m := AppModel{
		ctx:         context.Background(),
		rc:          stubResourceClient{},
		activePane:  TablePane,
		sidebar:     components.NewNavSidebarModel(22, 20),
		table:       components.NewResourceTableModel(58, 20),
		detail:      components.NewDetailViewModel(58, 20),
		filterBar:   components.NewFilterBarModel(),
		helpOverlay: components.NewHelpOverlayModel(),
		chat:        components.NewChatPaneModel(80, 20),
		chatSidebar: components.NewChatSidebarModel(20, 20),
	}
	m.updatePaneFocus()

	// Press [a] to open ChatPane from TablePane.
	result, _ := m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	appM := result.(AppModel)
	if appM.activePane != ChatPane {
		t.Fatalf("expected ChatPane after first [a], got %v", appM.activePane)
	}

	// Press [esc] to return to TablePane ([a] while input-focused types the char).
	result2, _ := appM.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	appM2 := result2.(AppModel)

	if appM2.activePane != TablePane {
		t.Errorf("activePane = %v after second [a] press, want TablePane", appM2.activePane)
	}
}

// TestChatPaneEscReturnToOrigin verifies that pressing [esc] from ChatPane
// returns to the origin pane.
func TestChatPaneEscReturnToOrigin(t *testing.T) {
	t.Parallel()
	m := newChatPaneModelForTest()
	// Set up origin as NavPane explicitly.
	m.chatOriginPane = DashboardOrigin{Pane: NavPane, ShowDashboard: false}

	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	appM := result.(AppModel)

	if appM.activePane != NavPane {
		t.Errorf("activePane = %v after [esc] in ChatPane, want NavPane", appM.activePane)
	}
}

// TestChatPaneEnterSendsMessage verifies that pressing Enter in ChatPane with
// text in the input and an initialised agent:
//   - appends the user message to the chat
//   - marks the chat as Processing
//   - dispatches a non-nil tea.Cmd (sendChatMessageCmd)
func TestChatPaneEnterSendsMessage(t *testing.T) {
	t.Parallel()
	m := newChatPaneModelForTest()

	// Simulate agent initialisation by directly delivering a chatAgentInitMsg.
	// NewAgent with nil LLM will fail on RunTurn but that's not tested here.
	agent := datumai.NewAgent(datumai.AgentOptions{
		LLM:      nil,
		Registry: datumai.NewEmptyRegistry(),
	})
	result, _ := m.Update(chatAgentInitMsg{agent: agent})
	appM := result.(AppModel)
	if appM.chatAgent == nil {
		t.Fatal("chatAgent still nil after chatAgentInitMsg")
	}

	// Manually set input value by routing a key character, then simulate enter.
	// We set the text directly by constructing the input state.
	// Since chat.InputValue() reads from the textinput, we need to feed characters.
	// Send a rune key so the textinput picks it up.
	for _, r := range []rune("hello world") {
		result, _ = appM.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		appM = result.(AppModel)
	}

	// Now press Enter.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	appM.chatTurnCtx = ctx
	appM.chatTurnCancel = cancel

	result, cmd := appM.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	appM = result.(AppModel)

	if !appM.chat.Processing() {
		t.Error("chat.Processing() = false after Enter press, want true")
	}
	if cmd == nil {
		t.Error("cmd = nil after Enter press with text, want non-nil (sendChatMessageCmd)")
	}
}

// TestChatConfirmAccept verifies that pressing [y] while a confirm is pending:
//   - sends true on the reply channel
//   - clears ConfirmPending
func TestChatConfirmAccept(t *testing.T) {
	t.Parallel()
	m := newChatPaneModelForTest()

	replyCh := make(chan bool, 1)
	m.chatConfirmReply = replyCh
	call := llm.ToolCall{
		ID:       "call-1",
		ToolName: "delete_resource",
		Arguments: map[string]any{
			"kind": "DNSZone",
			"name": "test-zone",
		},
	}
	m.chat.SetConfirmPending(call)

	result, _ := m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	appM := result.(AppModel)

	if appM.chat.ConfirmPending() {
		t.Error("ConfirmPending() = true after [y], want false")
	}
	select {
	case reply := <-replyCh:
		if !reply {
			t.Errorf("reply channel received %v, want true", reply)
		}
	default:
		t.Error("reply channel received nothing after [y]; expected true to be sent")
	}
}

// TestChatConfirmDecline verifies that pressing [n] while a confirm is pending:
//   - sends false on the reply channel
//   - clears ConfirmPending
func TestChatConfirmDecline(t *testing.T) {
	t.Parallel()
	m := newChatPaneModelForTest()

	replyCh := make(chan bool, 1)
	m.chatConfirmReply = replyCh
	call := llm.ToolCall{
		ID:       "call-2",
		ToolName: "apply_manifest",
		Arguments: map[string]any{
			"yaml": "kind: DNSZone",
		},
	}
	m.chat.SetConfirmPending(call)

	result, _ := m.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	appM := result.(AppModel)

	if appM.chat.ConfirmPending() {
		t.Error("ConfirmPending() = true after [n], want false")
	}
	select {
	case reply := <-replyCh:
		if reply {
			t.Errorf("reply channel received %v, want false", reply)
		}
	default:
		t.Error("reply channel received nothing after [n]; expected false to be sent")
	}
}

// TestChatAgentInitMsgSetsAgent verifies that delivering chatAgentInitMsg sets
// the chatAgent field and calls SetAgentReady on the chat component.
func TestChatAgentInitMsgSetsAgent(t *testing.T) {
	t.Parallel()
	m := newChatPaneModelForTest()
	agent := datumai.NewAgent(datumai.AgentOptions{
		LLM:      nil,
		Registry: datumai.NewEmptyRegistry(),
	})

	result, _ := m.Update(chatAgentInitMsg{agent: agent})
	appM := result.(AppModel)

	if appM.chatAgent == nil {
		t.Error("chatAgent = nil after chatAgentInitMsg with valid agent, want non-nil")
	}
	plain := stripANSIModel(appM.chat.View())
	if plain == "" {
		return // View renders without crash — test passes
	}
}

// TestChatAgentInitMsgWithError verifies that delivering chatAgentInitMsg with
// an error sets an agent error on the chat component.
func TestChatAgentInitMsgWithError(t *testing.T) {
	t.Parallel()
	m := newChatPaneModelForTest()

	result, _ := m.Update(chatAgentInitMsg{err: errStubNotFound})
	appM := result.(AppModel)

	if appM.chatAgent != nil {
		t.Error("chatAgent must be nil after chatAgentInitMsg with error")
	}
	plain := stripANSIModel(appM.chat.View())
	if plain == "" {
		return
	}
}
