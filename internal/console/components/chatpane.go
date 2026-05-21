package components

import (
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"go.datum.net/datumctl/internal/ai/llm"
	"go.datum.net/datumctl/internal/console/styles"
)

type chatMsg struct {
	role    string // "user" or "assistant"
	content string
}

// ChatPaneModel is the interactive AI chat pane component.
type ChatPaneModel struct {
	width, height int
	focused       bool

	processing     bool // true while RunTurn goroutine is running
	agentReady     bool
	agentErr       string
	confirmPending bool
	confirmCall    llm.ToolCall // non-zero when confirmPending

	streaming    bool // true while chunks are being appended to the streaming slot
	streamingIdx int  // index into messages[] of the current assistant streaming slot

	messages           []chatMsg
	messageLineOffsets []int // viewport line where each user msg starts (for sidebar sync)

	orgName     string
	projectName string

	inputReady bool // true once textarea has been initialized via New()
	input      textarea.Model
	vp         viewport.Model
	sp         spinner.Model
}

// NewChatPaneModel creates a ChatPaneModel sized to the given inner dimensions.
func NewChatPaneModel(w, h int) ChatPaneModel {
	ta := textarea.New()
	ta.Placeholder = "Ask about your Datum Cloud resources…"
	ta.ShowLineNumbers = false
	ta.CharLimit = 4000
	ta.SetHeight(2)
	ta.MaxHeight = 6 // grow up to 6 rows, then scroll
	taStyles := ta.Styles()
	taStyles.Focused.Text = lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Primary)
	taStyles.Blurred.Text = lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Primary)
	taStyles.Focused.Placeholder = lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Muted)
	taStyles.Blurred.Placeholder = lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Muted)
	taStyles.Focused.Prompt = lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Accent)
	taStyles.Blurred.Prompt = lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Muted)
	taStyles.Focused.EndOfBuffer = lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Primary)
	taStyles.Blurred.EndOfBuffer = lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Muted)
	// Base foreground must be set explicitly; otherwise typed text inherits the
	// terminal default which may be invisible against the Surface background.
	taStyles.Focused.Base = lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Primary)
	taStyles.Blurred.Base = lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Muted)
	ta.SetStyles(taStyles)
	focusCmd := ta.Focus()
	_ = focusCmd // focus blink cmd; Init() will handle startup

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	m := ChatPaneModel{input: ta, sp: sp, inputReady: true}
	m.SetSize(w, h)
	return m
}

// SetSize resizes the chat pane and rebuilds the viewport.
func (m *ChatPaneModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	if !m.inputReady {
		// Zero-value model (e.g. uninitialized AppModel in tests) — skip textarea
		// sizing; the internal viewport pointer is nil and would panic.
		vpH := max(1, h-4)
		m.vp.SetWidth(w)
		m.vp.SetHeight(vpH)
		return
	}
	m.input.SetWidth(max(1, w-2))
	// viewport gets everything minus: title(1) + rule(1) + sep(1) + input area
	inputH := max(2, m.input.Height())
	vpH := max(1, h-3-inputH)
	m.vp.SetWidth(w)
	m.vp.SetHeight(vpH)
	m.rebuildContent()
}

// Width returns the current pane width.
func (m ChatPaneModel) Width() int { return m.width }

// Height returns the current pane height.
func (m ChatPaneModel) Height() int { return m.height }

// SetFocused sets whether this pane has keyboard focus.
func (m *ChatPaneModel) SetFocused(f bool) { m.focused = f }

// SetProcessing sets the spinner state.
func (m *ChatPaneModel) SetProcessing(b bool) { m.processing = b }

// SetAgentReady marks the agent as initialised and clears any prior error.
func (m *ChatPaneModel) SetAgentReady() { m.agentReady = true; m.agentErr = "" }

// SetAgentError records an agent initialisation error and stops the spinner.
func (m *ChatPaneModel) SetAgentError(s string) {
	m.agentErr = s
	m.processing = false
	m.rebuildContent()
}

// SetConfirmPending records a pending tool-call confirmation request.
func (m *ChatPaneModel) SetConfirmPending(call llm.ToolCall) {
	m.confirmPending = true
	m.confirmCall = call
}

// ClearConfirmPending clears any pending confirmation request.
func (m *ChatPaneModel) ClearConfirmPending() {
	m.confirmPending = false
	m.confirmCall = llm.ToolCall{}
}

// ConfirmPending reports whether a tool-call confirmation is waiting for input.
func (m ChatPaneModel) ConfirmPending() bool { return m.confirmPending }

// Processing reports whether an agent turn is in flight.
func (m ChatPaneModel) Processing() bool { return m.processing }

// SetContext sets the active org and project names shown in the pane header.
func (m *ChatPaneModel) SetContext(org, project string) {
	m.orgName = org
	m.projectName = project
}

// LastAssistantMessage returns the content of the most recent assistant message, or "".
func (m ChatPaneModel) LastAssistantMessage() string {
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].role == "assistant" {
			return m.messages[i].content
		}
	}
	return ""
}

// ScrollUp scrolls the message viewport up by 3 lines.
func (m *ChatPaneModel) ScrollUp() { m.vp.ScrollUp(3) }

// ScrollDown scrolls the message viewport down by 3 lines.
func (m *ChatPaneModel) ScrollDown() { m.vp.ScrollDown(3) }

// ScrollToLine jumps the viewport to the given line offset.
func (m *ChatPaneModel) ScrollToLine(line int) { m.vp.SetYOffset(line) }

// InputValue returns the current text-input value.
func (m ChatPaneModel) InputValue() string { return m.input.Value() }

// ClearInput resets the text input.
func (m *ChatPaneModel) ClearInput() { m.input.SetValue("") }

// AppendUserMessage appends a user turn to the chat and scrolls to the bottom.
func (m *ChatPaneModel) AppendUserMessage(text string) {
	m.messages = append(m.messages, chatMsg{role: "user", content: text})
	m.rebuildContent()
	m.vp.GotoBottom()
}

// AppendToolEvent appends an ephemeral tool-call status line to the chat and
// scrolls to the bottom. Tool events are rendered as dimmed italic lines and
// are not included in messageLineOffsets (they are not user messages).
func (m *ChatPaneModel) AppendToolEvent(toolName string) {
	m.messages = append(m.messages, chatMsg{role: "tool", content: toolName})
	m.rebuildContent()
	m.vp.GotoBottom()
}

// AppendAssistantMessage appends an assistant turn to the chat and scrolls to the bottom.
func (m *ChatPaneModel) AppendAssistantMessage(text string) {
	m.messages = append(m.messages, chatMsg{role: "assistant", content: text})
	m.rebuildContent()
	m.vp.GotoBottom()
}

// StartAssistantStream opens an empty assistant message slot that will be
// filled chunk-by-chunk via AppendToStream.
func (m *ChatPaneModel) StartAssistantStream() {
	m.streamingIdx = len(m.messages)
	m.messages = append(m.messages, chatMsg{role: "assistant", content: ""})
	m.streaming = true
	m.rebuildContent()
}

// AppendToStream appends a text chunk to the current streaming assistant message.
func (m *ChatPaneModel) AppendToStream(chunk string) {
	if !m.streaming || m.streamingIdx >= len(m.messages) {
		return
	}
	m.messages[m.streamingIdx].content += chunk
	m.rebuildContent()
	m.vp.GotoBottom()
}

// FinalizeStream marks the in-progress stream as complete. The viewport is
// rebuilt once more so the final markdown render is applied.
func (m *ChatPaneModel) FinalizeStream() {
	m.streaming = false
	m.rebuildContent()
}

// StreamContent returns the accumulated content of the current streaming
// message (empty string when not streaming).
func (m ChatPaneModel) StreamContent() string {
	if !m.streaming || m.streamingIdx >= len(m.messages) {
		return ""
	}
	return m.messages[m.streamingIdx].content
}

// LineOffsetForMessage returns the viewport line offset for the nth user message (0-based).
func (m ChatPaneModel) LineOffsetForMessage(idx int) (int, bool) {
	if idx < 0 || idx >= len(m.messageLineOffsets) {
		return 0, false
	}
	return m.messageLineOffsets[idx], true
}

// Init starts the spinner tick and the textarea cursor blink.
func (m ChatPaneModel) Init() tea.Cmd {
	focusCmd := m.input.Focus()
	return tea.Batch(func() tea.Msg { return m.sp.Tick() }, focusCmd)
}

// Update handles spinner ticks and delegates input/viewport updates.
func (m ChatPaneModel) Update(msg tea.Msg) (ChatPaneModel, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.sp, cmd = m.sp.Update(msg)
		cmds = append(cmds, cmd)
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
		var vpCmd tea.Cmd
		m.vp, vpCmd = m.vp.Update(msg)
		cmds = append(cmds, vpCmd)
	}
	m.rebuildContent()
	return m, tea.Batch(cmds...)
}

// View renders the chat pane.
func (m ChatPaneModel) View() string {
	accentBold := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Accent).Bold(true)
	muted := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Muted)

	title := accentBold.Render("AI Chat")
	if m.orgName != "" || m.projectName != "" {
		ctx := m.orgName
		if m.projectName != "" {
			if ctx != "" {
				ctx += " / "
			}
			ctx += m.projectName
		}
		title = accentBold.Render("AI Chat") + "  " + muted.Render(ctx)
	}
	if m.processing {
		title = title + "  " + m.sp.View()
	}
	rule := muted.Render(strings.Repeat("─", m.width))
	sep := muted.Render(strings.Repeat("─", m.width))

	var inputRow string
	switch {
	case m.confirmPending:
		b, _ := json.MarshalIndent(m.confirmCall.Arguments, "", "  ")
		inputRow = lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Warning).
			Render(fmt.Sprintf("  Confirm %s?  [y] approve  [n] cancel\n%s", m.confirmCall.ToolName, string(b)))
	case m.agentErr != "":
		inputRow = lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Error).Render("  " + m.agentErr)
	default:
		inputRow = m.input.View()
	}

	// The static section (title, rule, viewport, separator) is passed through
	// SurfaceFill to ensure consistent background painting. The input row is
	// joined afterward so SurfaceFill does not interfere with the textarea's
	// own ANSI cursor and text escape codes.
	inputRowH := strings.Count(inputRow, "\n") + 1
	staticH := max(1, m.height-inputRowH)
	staticContent := lipgloss.JoinVertical(lipgloss.Left,
		title,
		rule,
		m.vp.View(),
		sep,
	)
	content := lipgloss.JoinVertical(lipgloss.Left,
		styles.SurfaceFill(staticContent, m.width, staticH),
		inputRow,
	)
	return styles.PaneBorder(m.focused).Render(content)
}

// rebuildContent regenerates the viewport content from the current message list.
func (m *ChatPaneModel) rebuildContent() {
	var sb strings.Builder
	accentBold := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Accent).Bold(true)
	def := lipgloss.NewStyle().Background(styles.Surface)
	muted := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Muted)

	m.messageLineOffsets = nil
	lineCount := 0

	for i, msg := range m.messages {
		switch msg.role {
		case "user":
			m.messageLineOffsets = append(m.messageLineOffsets, lineCount)
			header := accentBold.Render("You")
			sb.WriteString(header + "\n")
			sb.WriteString(def.Render(msg.content) + "\n\n")
			lineCount += strings.Count(msg.content, "\n") + 3
		case "assistant":
			header := muted.Render("Assistant")
			sb.WriteString(header + "\n")
			content := msg.content
			// Close any unclosed fenced code block so the renderer doesn't
			// style all subsequent text as code while the stream is in flight.
			if m.streaming && i == m.streamingIdx {
				if strings.Count(content, "```")%2 != 0 {
					content += "\n```"
				}
			}
			rendered := renderMarkdown(content, max(1, m.width-2))
			sb.WriteString(rendered + "\n\n")
			lineCount += strings.Count(rendered, "\n") + 3
		case "tool":
			// Ephemeral status line: dimmed italic, no header, not tracked in messageLineOffsets.
			toolStyle := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Muted).Italic(true)
			line := toolStyle.Render("  → " + msg.content)
			sb.WriteString(line + "\n")
			lineCount += 1
		}
	}

	if len(m.messages) == 0 {
		if m.agentErr != "" {
			sb.WriteString(muted.Render("  No API key configured. Run:\n  datumctl ai config set anthropic_api_key sk-ant-…\n"))
		} else if !m.agentReady {
			sb.WriteString(muted.Render("  Initializing AI assistant…\n"))
		} else {
			sb.WriteString(muted.Render("  Ask anything about your Datum Cloud resources.\n"))
		}
	}

	m.vp.SetContent(sb.String())
}
