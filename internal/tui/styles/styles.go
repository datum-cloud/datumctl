package styles

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const SidebarWidth = 22

// DatumWordmark is a 5-line slant-style ASCII wordmark rendered in the header.
const DatumWordmark = "    ____        __                \n   / __ \\____ _/ /___  ______ ___ \n  / / / / __ `/ __/ / / / __ `__ \\\n / /_/ / /_/ / /_/ /_/ / / / / / /\n/_____/\\__,_/\\__/\\__,_/_/ /_/ /_/"

var (
	Primary   = lipgloss.AdaptiveColor{Light: "#5C6BC0", Dark: "#7986CB"}
	Secondary = lipgloss.AdaptiveColor{Light: "#546E7A", Dark: "#90A4AE"}
	Muted     = lipgloss.AdaptiveColor{Light: "#757575", Dark: "#9E9E9E"}
	Error     = lipgloss.AdaptiveColor{Light: "#C62828", Dark: "#EF9A9A"}
	Success   = lipgloss.AdaptiveColor{Light: "#2E7D32", Dark: "#A5D6A7"}
	Accent    = lipgloss.AdaptiveColor{Light: "#00897B", Dark: "#4DD0E1"}
	Warning   = lipgloss.AdaptiveColor{Light: "#EF6C00", Dark: "#FFB74D"}

	ActiveBorderColor   = Accent
	InactiveBorderColor = lipgloss.AdaptiveColor{Light: "#BDBDBD", Dark: "#424242"}

	OverlayBackdrop = lipgloss.AdaptiveColor{Light: "#9E9E9E", Dark: "#212121"}

	HeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#FFFFFF"}).
			Background(lipgloss.AdaptiveColor{Light: "#3949AB", Dark: "#283593"}).
			Padding(0, 1)

	FooterStyle = lipgloss.NewStyle().
			Foreground(Secondary).
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(InactiveBorderColor).
			Padding(0, 1)

	SidebarStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(InactiveBorderColor).
			Padding(0, 1)

	TableStyle = lipgloss.NewStyle().
			Padding(0, 1)

	SelectedRowStyle = lipgloss.NewStyle().
				Background(Accent).
				Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#0B1D1F"}).
				Bold(true)

	OverlayStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.AdaptiveColor{Light: "#5C6BC0", Dark: "#7986CB"}).
			Padding(1, 2)
)

// PaneBorder returns a sidebar-shaped border style with active or inactive color.
func PaneBorder(focused bool) lipgloss.Style {
	color := lipgloss.TerminalColor(InactiveBorderColor)
	if focused {
		color = ActiveBorderColor
	}
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(color).
		Padding(0, 1)
}

// StatusColorFor returns a semantic color for a Kubernetes status string.
func StatusColorFor(status string) lipgloss.TerminalColor {
	switch strings.ToLower(status) {
	case "running", "ready", "active", "succeeded", "available", "true", "healthy":
		return Success
	case "pending", "updating", "progressing", "creating", "reconciling":
		return Warning
	case "failed", "error", "crashloopbackoff", "unhealthy", "false", "degraded":
		return Error
	case "terminating", "deleting":
		return Muted
	default:
		return Secondary
	}
}

// ComputeSidebarWidth returns a sidebar width clamped to [18, 28] based on the longest type name.
func ComputeSidebarWidth(typeNames []string) int {
	longest := 0
	for _, n := range typeNames {
		if len(n) > longest {
			longest = len(n)
		}
	}
	w := longest + 2
	if w < 18 {
		return 18
	}
	if w > 28 {
		return 28
	}
	return w
}
