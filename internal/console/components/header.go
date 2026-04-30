package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	tuictx "go.datum.net/datumctl/internal/console/context"
	"go.datum.net/datumctl/internal/console/styles"
)

type HeaderModel struct {
	Ctx   tuictx.TUIContext
	Width int
}

func NewHeaderModel(ctx tuictx.TUIContext) HeaderModel {
	return HeaderModel{Ctx: ctx}
}

func (m HeaderModel) View() string {
	w := m.Width
	if w <= 0 {
		w = 80
	}
	// Content width: HeaderStyle has Padding(0,1) so each line's text area is w-2.
	contentW := w - 2

	// Render each line individually at full terminal width so the blue background
	// fills the entire row — a pre-rendered inner block with its own ANSI resets
	// would fight the outer wrapper and leave the wordmark area unstyled.
	base := styles.HeaderStyle.Width(w)
	bold := base.Bold(true)

	sep := strings.Repeat("─", contentW)

	infoLeft := "user: " + m.Ctx.UserEmail
	if m.Ctx.OrgName != "" {
		infoLeft += "   org: " + m.Ctx.OrgName
	}
	if m.Ctx.ProjectName != "" {
		infoLeft += "   project: " + m.Ctx.ProjectName
	}

	infoLine := infoLeft
	if m.Ctx.ReadOnly {
		badge := lipgloss.NewStyle().
			Foreground(styles.Warning).
			Bold(true).
			Render("READ-ONLY")
		leftW := lipgloss.Width(infoLeft)
		badgeW := lipgloss.Width(badge)
		pad := contentW - leftW - badgeW
		if pad < 1 {
			pad = 1
		}
		infoLine = infoLeft + fmt.Sprintf("%*s", pad, "") + badge
	}

	ns := ""
	if m.Ctx.Namespace != "" {
		ns = "ns: " + m.Ctx.Namespace
	}

	paneLabel := m.Ctx.ActivePaneLabel
	if paneLabel != "" && m.Ctx.ResourceCount > 0 {
		paneLabel += fmt.Sprintf(" (%d)", m.Ctx.ResourceCount)
	}

	var refresh string
	switch {
	case m.Ctx.Refreshing:
		frame := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Warning).Bold(true).Render(m.Ctx.SpinnerFrame)
		label := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Warning).Render(" refreshing…")
		refresh = frame + label
	case !m.Ctx.LastRefresh.IsZero():
		refresh = "updated " + HumanizeSince(m.Ctx.LastRefresh)
	}

	nsW := lipgloss.Width(ns)
	paneW := lipgloss.Width(paneLabel)
	rightW := lipgloss.Width(refresh)

	centerStart := contentW/2 - paneW/2
	if centerStart < nsW+1 {
		centerStart = nsW + 1
	}
	leftPad := centerStart - nsW
	if leftPad < 0 {
		leftPad = 0
	}
	rightPad := contentW - centerStart - paneW - rightW
	if rightPad < 0 {
		rightPad = 0
	}
	ctxLine := ns + fmt.Sprintf("%*s", leftPad, "") + paneLabel + fmt.Sprintf("%*s", rightPad, "") + refresh

	var lines []string
	for _, wl := range strings.Split(styles.DatumWordmark, "\n") {
		lines = append(lines, bold.Render(wl))
	}
	lines = append(lines, base.Render(sep))
	lines = append(lines, base.Render(infoLine))
	lines = append(lines, base.Render(ctxLine))

	return strings.Join(lines, "\n")
}

func HumanizeSince(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < 5*time.Second:
		return "just now"
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	default:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
}
