package styles

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ColorizeDiff takes a raw unified-diff string and returns an ANSI-colorized string.
// Lines prefixed with "+++"/"---" → Accent bold; "@@ " → Muted; "+" → Success; "-" → Error.
// Context lines (space prefix or blank) are rendered without color.
func ColorizeDiff(raw string) string {
	if raw == "" {
		return ""
	}
	lines := strings.Split(raw, "\n")
	// Remove trailing empty element from Split on a newline-terminated string.
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	accent := lipgloss.NewStyle().Foreground(Accent).Bold(true)
	muted := lipgloss.NewStyle().Foreground(Muted)
	success := lipgloss.NewStyle().Foreground(Success)
	errStyle := lipgloss.NewStyle().Foreground(Error)

	var sb strings.Builder
	for i, line := range lines {
		var rendered string
		switch {
		case strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++"):
			rendered = accent.Render(line)
		case strings.HasPrefix(line, "@@"):
			rendered = muted.Render(line)
		case len(line) > 0 && line[0] == '+':
			rendered = success.Render(line)
		case len(line) > 0 && line[0] == '-':
			rendered = errStyle.Render(line)
		default:
			rendered = line
		}
		sb.WriteString(rendered)
		if i < len(lines)-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}
