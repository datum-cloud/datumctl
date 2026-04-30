package styles

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const SidebarWidth = 22

// DatumWordmark is a 5-line slant-style ASCII wordmark rendered in the header.
const DatumWordmark = "    ____        __                \n   / __ \\____ _/ /___  ______ ___ \n  / / / / __ `/ __/ / / / __ `__ \\\n / /_/ / /_/ / /_/ /_/ / / / / / /\n/_____/\\__,_/\\__/\\__,_/_/ /_/ /_/"

// Dark-mode palette derived from the Datum brand guidelines
// (datum.net/brand/color/). The console assumes a dark terminal background:
// Aurora Moss is the signature accent, Midnight Fjord anchors bounded surfaces,
// and Glacier Mist 900 carries body text. Pine Forge and Canyon Clay tones
// provide support.
var (
	// Foreground tones are tuned for high contrast against the dark Midnight
	// Fjord surface while staying inside the brand's sage / cream family.
	Primary   = lipgloss.Color("#F6F6F5") // Glacier Mist 700 — body text
	Secondary = lipgloss.Color("#B8CFBE") // lifted Pine Forge — supporting text
	Muted     = lipgloss.Color("#9AA89D") // mid sage — de-emphasized but legible
	Accent    = lipgloss.Color("#E6F59F") // Aurora Moss — signature highlight

	// Semantic colors are not defined in the brand guidelines; these harmonize
	// with the earthy brand palette while preserving stoplight legibility.
	Success = lipgloss.Color("#B8DCBB") // brightened sage green
	Warning = lipgloss.Color("#F0C89A") // warm amber bridging Canyon Clay and Aurora Moss
	Error   = lipgloss.Color("#F0B8B8") // lifted Canyon Clay

	// Surface is the dark-mode panel background — Midnight Fjord — painted on
	// every bounded surface so the console appears dark independent of the
	// user's terminal background color.
	Surface = lipgloss.Color("#0C1D31")

	ActiveBorderColor   = Accent
	InactiveBorderColor = lipgloss.Color("#5A6D62") // lifted Pine Forge — visible but subdued

	OverlayBackdrop = Surface // Midnight Fjord

	// Header uses the signature Midnight Fjord + Aurora Moss pairing.
	HeaderStyle = lipgloss.NewStyle().
			Foreground(Accent).
			Background(Surface).
			Padding(0, 1)

	FooterStyle = lipgloss.NewStyle().
			Foreground(Secondary).
			Background(Surface).
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(InactiveBorderColor).
			BorderBackground(Surface).
			Padding(0, 1)

	SidebarStyle = lipgloss.NewStyle().
			Background(Surface).
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(InactiveBorderColor).
			BorderBackground(Surface).
			Padding(0, 1)

	TableStyle = lipgloss.NewStyle().
			Background(Surface).
			Padding(0, 1)

	// Selected row uses a dark forest tone — clearly distinct from the Midnight
	// Fjord surface but soft enough not to overpower the content.
	SelectedBg       = lipgloss.Color("#1E3A2A")
	SelectedRowStyle = lipgloss.NewStyle().
				Background(SelectedBg).
				Foreground(Accent).
				Bold(true)

	OverlayStyle = lipgloss.NewStyle().
			Background(Surface).
			Foreground(Primary).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Accent).
			BorderBackground(Surface).
			Padding(1, 2)
)

// PaneBorder returns a sidebar-shaped border style with active or inactive color.
// The Surface background is painted across the bordered area so main panels
// render on the dark-mode Midnight Fjord surface rather than inheriting the
// user's terminal background.
func PaneBorder(focused bool) lipgloss.Style {
	color := lipgloss.TerminalColor(InactiveBorderColor)
	if focused {
		color = ActiveBorderColor
	}
	return lipgloss.NewStyle().
		Background(Surface).
		Foreground(Primary).
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(color).
		BorderBackground(Surface).
		Padding(0, 1)
}

// PaneInnerSize returns the inner content size for a PaneBorder-wrapped
// surface allocated (width, height). PaneBorder adds 1 column of padding on
// each side plus a 1-column right border, so the inner surface must be
// (width - 3) cells wide for the rendered pane to fit exactly within the
// allocated width. Height is unaffected (no vertical padding/border).
func PaneInnerSize(width, height int) (innerW, innerH int) {
	innerW = width - 3
	if innerW < 0 {
		innerW = 0
	}
	innerH = height
	if innerH < 0 {
		innerH = 0
	}
	return innerW, innerH
}

// SurfaceFill paints Surface as the background across every cell of every
// line, pads each line to width, and pads vertically to height.
//
// Unlike wrapping content in a lipgloss Style with Background(Surface), which
// loses the bg after every inner `\x1b[0m` reset emitted by nested styled
// spans, SurfaceFill re-asserts the active background immediately after each
// reset. For lines that open with a non-Surface background (e.g. the selected
// table row), that background is preserved through resets instead of Surface,
// so the highlight spans the full row width.
func SurfaceFill(s string, width, height int) string {
	surfaceOpen := lipgloss.NewStyle().Background(Surface).Render("")
	surfaceOpen = strings.TrimSuffix(surfaceOpen, "\x1b[0m")
	const reset = "\x1b[0m"

	var lines []string
	if s == "" {
		lines = []string{""}
	} else {
		lines = strings.Split(s, "\n")
	}

	blank := surfaceOpen + strings.Repeat(" ", max(width, 0)) + reset

	out := make([]string, 0, max(len(lines), height))
	for _, ln := range lines {
		vw := lipgloss.Width(ln)
		padded := ln
		if vw < width {
			padded += strings.Repeat(" ", width-vw)
		}
		// Determine which background to keep sticky for this line. If the line
		// opens with ANSI codes other than Surface's (e.g. the selected-row
		// highlight), preserve those codes through resets so the background
		// spans the full width rather than reverting to Surface mid-row.
		bgOpen := surfaceOpen
		if prefix := ansiPrefix(ln); prefix != "" && prefix != surfaceOpen {
			bgOpen = prefix
		}
		padded = strings.ReplaceAll(padded, reset, reset+bgOpen)
		out = append(out, bgOpen+padded+reset)
	}
	for len(out) < height {
		out = append(out, blank)
	}
	return strings.Join(out, "\n")
}

// ansiPrefix returns all leading ANSI escape sequences from s (everything
// before the first non-escape byte), or "" if s does not start with an escape.
func ansiPrefix(s string) string {
	i := 0
	for i < len(s) {
		if s[i] != '\x1b' || i+1 >= len(s) || s[i+1] != '[' {
			break
		}
		j := i + 2
		for j < len(s) && (s[j] < 'A' || s[j] > 'Z') && (s[j] < 'a' || s[j] > 'z') {
			j++
		}
		if j < len(s) {
			j++
		}
		i = j
	}
	return s[:i]
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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
