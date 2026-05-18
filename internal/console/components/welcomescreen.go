package components

import (
	"strings"

	"charm.land/lipgloss/v2"
	"go.datum.net/datumctl/internal/console/styles"
)

// WelcomeScreenModel renders a full-screen welcome page shown to unauthenticated
// users. It owns its own layout so the normal header/sidebar/table chrome is
// skipped entirely, giving a clean centered presentation.
type WelcomeScreenModel struct {
	Width  int
	Height int
}

func NewWelcomeScreenModel() WelcomeScreenModel {
	return WelcomeScreenModel{}
}

func renderServicesTiers(accent, muted, _ lipgloss.Style) string {
	tiers := []struct{ name, subtitle string }{
		{"DELIVER", "route users, agents, and traffic"},
		{"BUILD", "run apps and AI workloads"},
		{"CONNECT", "private networking and reachability"},
		{"MANAGE", "observe, secure, and operate"},
	}
	var rows []string
	for _, t := range tiers {
		rows = append(rows, accent.Render(t.name)+muted.Render("  "+t.subtitle))
	}
	return strings.Join(rows, "\n")
}

func (m WelcomeScreenModel) View() string {
	accent := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Accent).Bold(true)
	muted := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Muted)
	secondary := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Secondary)
	success := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Success).Bold(true)

	w := m.Width
	if w <= 0 {
		w = 80
	}
	h := m.Height
	if h <= 0 {
		h = 40
	}

	var blocks []string

	// Datum wordmark — 5-line ASCII art, accent-colored
	wordmarkLines := strings.Split(styles.DatumWordmark, "\n")
	styledWordmark := make([]string, len(wordmarkLines))
	for i, line := range wordmarkLines {
		styledWordmark[i] = accent.Render(line)
	}
	blocks = append(blocks, strings.Join(styledWordmark, "\n"))
	blocks = append(blocks, "")

	// Title
	blocks = append(blocks, accent.Render("Welcome to Datum Cloud"))
	blocks = append(blocks, "")

	// Tagline
	if w >= 54 {
		blocks = append(blocks, secondary.Render("Connect infrastructure. Ship faster. Sleep better."))
		blocks = append(blocks, "")
	}

	// Four tiers — name + subtitle only, no service details
	blocks = append(blocks, renderServicesTiers(accent, muted, secondary))
	blocks = append(blocks, "")

	// Action prompts
	loginLine := success.Render("▸") + muted.Render("  ") + accent.Render("[l]") + muted.Render("  ") + secondary.Render("log in to get started")
	blocks = append(blocks, loginLine)
	blocks = append(blocks, accent.Render("[q]")+muted.Render("  quit"))

	inner := lipgloss.JoinVertical(lipgloss.Center, blocks...)

	return lipgloss.NewStyle().Background(styles.Surface).Render(
		lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, inner,
			lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Background(styles.Surface)),
		),
	)
}
