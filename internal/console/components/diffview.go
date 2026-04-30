package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.datum.net/datumctl/internal/console/data"
	"go.datum.net/datumctl/internal/console/styles"
)

// DiffViewModel renders a unified diff (or creation manifest) for a single revision.
type DiffViewModel struct {
	width, height int
	focused       bool
	vp            viewport.Model

	rev             data.HistoryRow // revision being viewed
	prev            *data.HistoryRow // nil if rev 1 or predecessor not loaded
	content         string           // pre-rendered diff body (already colorized)
	isCreation      bool
	predecessorMissing bool
}

func NewDiffViewModel(width, height int) DiffViewModel {
	m := DiffViewModel{}
	m.SetSize(width, height)
	return m
}

func (m DiffViewModel) Init() tea.Cmd { return nil }

func (m DiffViewModel) Update(msg tea.Msg) (DiffViewModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down", "k", "up", "pgup", "pgdown", "g", "G":
			m.vp, cmd = m.vp.Update(msg)
		}
	default:
		m.vp, cmd = m.vp.Update(msg)
	}
	return m, cmd
}

func (m DiffViewModel) View() string {
	var content string
	if m.height < 6 {
		content = m.vp.View()
	} else {
		content = lipgloss.JoinVertical(lipgloss.Left,
			m.titleBar(),
			m.titleRule(),
			m.metaBanner(),
			m.metaRule(),
			m.vp.View(),
			m.footerRule(),
			m.scrollFooter(),
		)
	}
	return styles.PaneBorder(m.focused).Render(styles.SurfaceFill(content, m.width, m.height))
}

func (m *DiffViewModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	vpH := h
	if h >= 6 {
		// titleBar + titleRule + metaBanner + metaRule + footerRule + scrollFooter = 6 lines
		vpH = max(h-6, 1)
	}
	m.vp.Width = w
	m.vp.Height = vpH
}

func (m *DiffViewModel) SetFocused(focused bool) { m.focused = focused }

// SetRevision updates the diff view with a new revision and pre-colorized diff body.
// Call Reset() on the viewport before setting a new revision.
func (m *DiffViewModel) SetRevision(
	rev data.HistoryRow,
	prev *data.HistoryRow,
	colorizedBody string,
	isCreation, predMissing bool,
) {
	m.rev = rev
	m.prev = prev
	m.content = colorizedBody
	m.isCreation = isCreation
	m.predecessorMissing = predMissing
	m.vp.SetContent(m.buildContent())
	m.vp.GotoTop()
}

// Reset clears the diff view.
func (m *DiffViewModel) Reset() {
	m.rev = data.HistoryRow{}
	m.prev = nil
	m.content = ""
	m.isCreation = false
	m.predecessorMissing = false
	m.vp.SetContent("")
	m.vp.GotoTop()
}

func (m DiffViewModel) buildContent() string {
	var sb strings.Builder

	switch {
	case m.isCreation:
		success := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Success)
		sb.WriteString(success.Render("✨ Created resource"))
		sb.WriteString("\n\n")
		sb.WriteString(m.content)
	case m.predecessorMissing:
		muted := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Muted)
		sb.WriteString(muted.Render("📸 Initial state (oldest available change)"))
		sb.WriteString("\n\n")
		sb.WriteString(m.content)
	case m.content == "":
		muted := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Muted)
		sb.WriteString(muted.Render("  No visible changes in this revision (metadata-only)."))
	default:
		sb.WriteString(m.content)
	}

	return sb.String()
}

func (m DiffViewModel) metaBanner() string {
	muted := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Muted)
	accent := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Accent)
	secondary := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Secondary)

	var revLabel string
	if m.isCreation {
		revLabel = accent.Render(fmt.Sprintf("Rev %d (creation)", m.rev.Rev))
	} else if m.predecessorMissing {
		revLabel = accent.Render(fmt.Sprintf("Rev %d", m.rev.Rev)) +
			muted.Render(" ← (not loaded)")
	} else if m.prev != nil {
		revLabel = accent.Render(fmt.Sprintf("Rev %d", m.rev.Rev)) +
			muted.Render(" ← ") +
			accent.Render(fmt.Sprintf("Rev %d", m.prev.Rev))
	} else {
		revLabel = accent.Render(fmt.Sprintf("Rev %d", m.rev.Rev))
	}

	ts := m.rev.Timestamp.Local().Format("2006-01-02 15:04:05")
	verb := secondary.Render(m.rev.Verb)
	user := muted.Render(m.rev.User)
	statusStr := m.coloredStatus(m.rev.Status)

	parts := []string{revLabel, verb, ts, user}
	if statusStr != "" {
		parts = append(parts, statusStr)
	}

	return strings.Join(parts, muted.Render("    "))
}

func (m DiffViewModel) coloredStatus(code int32) string {
	if code == 0 {
		return ""
	}
	label := fmt.Sprintf("[%d]", code)
	switch {
	case code >= 200 && code < 300:
		return lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Success).Render(label)
	case code >= 400:
		return lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Error).Render(label)
	default:
		return lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Muted).Render(label)
	}
}

func (m DiffViewModel) titleBar() string {
	w := m.width
	accentBold := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Accent).Bold(true)
	muted := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Muted)

	leftText := accentBold.Render(m.rev.Verb) // will be empty for zero value
	if m.rev.Rev > 0 {
		leftText = accentBold.Render(fmt.Sprintf("Rev %d", m.rev.Rev))
	}

	rightText := muted.Render("[H] describe  [Esc] to list  [[]prev  []]next")

	gap := max(1, w-lipgloss.Width(leftText)-lipgloss.Width(rightText))
	return leftText + strings.Repeat(" ", gap) + rightText
}

func (m DiffViewModel) titleRule() string {
	return lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.InactiveBorderColor).
		Render(strings.Repeat("─", m.width))
}

func (m DiffViewModel) metaRule() string {
	return lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.InactiveBorderColor).
		Render(strings.Repeat("─", m.width))
}

func (m DiffViewModel) footerRule() string {
	return lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.InactiveBorderColor).
		Render(strings.Repeat("─", m.width))
}

func (m DiffViewModel) scrollFooter() string {
	pct := m.vp.ScrollPercent()
	var label string
	switch pct {
	case 0.0:
		label = "top"
	case 1.0:
		label = "100%"
	default:
		label = fmt.Sprintf("%d%%", int(pct*100))
	}

	secondary := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Secondary).Bold(true)
	muted := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Muted)

	styledLabel := secondary.Render(label)
	labelWidth := lipgloss.Width(styledLabel)
	ruleWidth := max(m.width-labelWidth-2, 0)

	return muted.Render(strings.Repeat("─", ruleWidth)) + "  " + styledLabel
}
