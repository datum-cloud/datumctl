package components

import (
	"charm.land/lipgloss/v2"
	"go.datum.net/datumctl/internal/console/styles"
)

type HelpOverlayModel struct {
	Width              int
	Height             int
	ShowDeleteHint     bool // true when activePane ∈ {TablePane, DetailPane}
	ShowConditionsHint bool // true when activePane == DetailPane (FB-018)
	ShowEventsHint     bool // true when activePane == DetailPane (FB-019) // AC#26
}

func NewHelpOverlayModel() HelpOverlayModel {
	return HelpOverlayModel{}
}

func (m HelpOverlayModel) View() string {
	accent := lipgloss.NewStyle().Foreground(styles.Accent).Bold(true)
	row := lipgloss.NewStyle().Foreground(styles.Secondary)
	muted := lipgloss.NewStyle().Foreground(styles.Muted).Italic(true)

	nav := lipgloss.JoinVertical(lipgloss.Left,
		accent.Render("NAVIGATION"),
		row.Render("[j/k] move down/up"),
		row.Render("[Tab] next pane"),
		muted.Render("      resume (cached)"),
		row.Render("[↑↓]  arrow keys"),
	)

	actionLines := []string{
		accent.Render("ACTIONS"),
		row.Render("[Enter] select"),
		row.Render("[/]    filter"),
		row.Render("[Esc]  back / home"),
	}
	if m.ShowConditionsHint {
		actionLines = append(actionLines, row.Render("[C]    conditions")) // AC#22
	}
	if m.ShowEventsHint { // AC#26
		actionLines = append(actionLines, row.Render("[E]    events"))
	}
	if m.ShowDeleteHint {
		actionLines = append(actionLines, row.Render("[x]    delete resource"))
	}
	actions := lipgloss.JoinVertical(lipgloss.Left, actionLines...)

	view := lipgloss.JoinVertical(lipgloss.Left,
		accent.Render("VIEW"),
		row.Render("[d]  describe"),
		row.Render("[r]  refresh"),
		row.Render("[c]  switch context"),
		row.Render("[3]  quota dashboard"),
		row.Render("[t]  quota table"),
		row.Render("[4]  activity dashboard"),
	)

	global := lipgloss.JoinVertical(lipgloss.Left,
		accent.Render("GLOBAL"),
		row.Render("[?]  help"),
		row.Render("[q]  quit"),
		row.Render("[^C] force quit"),
	)

	cols := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(22).Render(nav),
		lipgloss.NewStyle().Width(22).Render(actions),
		lipgloss.NewStyle().Width(24).Render(view),
		lipgloss.NewStyle().Width(22).Render(global),
	)

	footer := muted.Render("? or Esc to close")

	body := lipgloss.JoinVertical(lipgloss.Center, cols, "", footer)
	modal := styles.OverlayStyle.Width(90).Render(body)

	return lipgloss.Place(m.Width, m.Height,
		lipgloss.Center, lipgloss.Center, modal,
		lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Background(styles.OverlayBackdrop)),
	)
}
