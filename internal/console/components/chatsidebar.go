package components

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"go.datum.net/datumctl/internal/console/styles"
)

// ConvEntry is a lightweight entry shown in the sidebar.
type ConvEntry struct {
	ID        string
	UpdatedAt time.Time
	Preview   string
}

// ChatSidebarModel shows a scrollable list of past conversations.
type ChatSidebarModel struct {
	width, height int
	focused       bool

	history []ConvEntry
	histCur int
}

// NewChatSidebarModel creates a ChatSidebarModel sized to the given inner dimensions.
func NewChatSidebarModel(w, h int) ChatSidebarModel {
	return ChatSidebarModel{width: w, height: h}
}

// SetSize resizes the sidebar.
func (m *ChatSidebarModel) SetSize(w, h int) { m.width = w; m.height = h }

// SetFocused sets whether this sidebar has keyboard focus.
func (m *ChatSidebarModel) SetFocused(f bool) { m.focused = f }

// SetHistory replaces the conversation history list.
func (m *ChatSidebarModel) SetHistory(entries []ConvEntry) {
	m.history = entries
	if m.histCur >= len(entries) {
		m.histCur = max(0, len(entries)-1)
	}
}

// HistoryCursor returns the selected history entry index.
func (m ChatSidebarModel) HistoryCursor() int { return m.histCur }

// SelectedHistoryEntry returns the currently highlighted history entry, if any.
func (m ChatSidebarModel) SelectedHistoryEntry() (ConvEntry, bool) {
	if len(m.history) == 0 {
		return ConvEntry{}, false
	}
	return m.history[m.histCur], true
}

// CursorUp moves the cursor toward earlier conversations.
func (m *ChatSidebarModel) CursorUp() {
	if m.histCur > 0 {
		m.histCur--
	}
}

// CursorDown moves the cursor toward later conversations.
func (m *ChatSidebarModel) CursorDown() {
	if m.histCur < len(m.history)-1 {
		m.histCur++
	}
}

// Init satisfies the tea.Model interface.
func (m ChatSidebarModel) Init() tea.Cmd { return nil }

// Update satisfies the tea.Model interface (sidebar has no independent message handling).
func (m ChatSidebarModel) Update(_ tea.Msg) (ChatSidebarModel, tea.Cmd) { return m, nil }

// View renders the sidebar.
func (m ChatSidebarModel) View() string {
	accentBold := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Primary).Bold(true)
	muted := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Muted)
	titleStyle := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Secondary).Bold(true)
	dimStyle := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Muted).Italic(true)

	var sb strings.Builder

	title := titleStyle.Render("Conversations")
	rule := muted.Render(strings.Repeat("─", m.width))
	sb.WriteString(title + "\n")
	sb.WriteString(rule + "\n")

	contentH := max(0, m.height-4) // leave room for hint
	start := 0
	if m.histCur >= contentH/2 {
		start = m.histCur - contentH/2
	}
	end := min(start+contentH/2, len(m.history))

	for i := start; i < end; i++ {
		e := m.history[i]
		preview := e.Preview
		if preview == "" {
			preview = "(empty)"
		}
		maxW := max(4, m.width-4)
		runes := []rune(preview)
		if len(runes) > maxW {
			preview = string(runes[:maxW-1]) + "…"
		}
		ts := e.UpdatedAt.Local().Format("01/02 15:04")
		if i == m.histCur && m.focused {
			sb.WriteString(accentBold.Render(" ▸ "+ts) + "\n")
			sb.WriteString(accentBold.Render("   "+preview) + "\n")
		} else {
			sb.WriteString(muted.Render("   "+ts) + "\n")
			sb.WriteString(dimStyle.Render("   "+preview) + "\n")
		}
	}

	if len(m.history) == 0 {
		sb.WriteString(muted.Render("  (no saved chats)") + "\n")
	}

	sb.WriteString("\n" + muted.Render("[Enter] load  [Shift+N] new"))

	content := sb.String()
	return styles.PaneBorder(m.focused).Render(styles.SurfaceFill(content, m.width, m.height))
}
