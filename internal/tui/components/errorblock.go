package components

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/charmbracelet/lipgloss"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"go.datum.net/datumctl/internal/tui/data"
	"go.datum.net/datumctl/internal/tui/styles"
)

// ActionHint is a single keybinding hint rendered in the action row.
type ActionHint struct {
	Key   string // e.g. "r", "Esc"
	Label string // e.g. "retry", "back to navigation"
}

// ErrorBlock is the structured input to RenderErrorBlock.
type ErrorBlock struct {
	Title    string
	Detail   string
	Actions  []ActionHint
	Severity data.ErrorSeverity
	Width    int // render width (pane-local inner width)
}

// RenderErrorBlock renders a standardized error card per the FB-022 spec.
func RenderErrorBlock(b ErrorBlock) string {
	w := b.Width
	if w < 0 {
		w = 0
	}

	g := errorGlyph(b.Severity)
	color := errorColor(b.Severity)

	title := b.Title
	if title == "" {
		title = "Error"
	}

	titleStyle := lipgloss.NewStyle().Foreground(color).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(styles.Muted)
	accentStyle := lipgloss.NewStyle().Foreground(styles.Accent).Bold(true)

	// Collapsed band (<20): single-line fallback, no actions or detail.
	if w < 20 {
		return titleStyle.Render(g + " " + title)
	}

	// Build action row string (cap at 4 hints + ellipsis).
	actions := b.Actions
	addEllipsis := len(actions) > 4
	if addEllipsis {
		actions = actions[:4]
	}
	var actionParts []string
	for _, a := range actions {
		bracket := accentStyle.Render("[" + a.Key + "]")
		label := mutedStyle.Render(a.Label)
		actionParts = append(actionParts, bracket+" "+label)
	}
	if addEllipsis {
		actionParts = append(actionParts, mutedStyle.Render("…"))
	}
	actionRow := strings.Join(actionParts, "   ")

	hasDetail := b.Detail != ""
	hasActions := len(b.Actions) > 0

	var lines []string
	lines = append(lines, "") // top-pad
	lines = append(lines, "  "+titleStyle.Render(g+" "+title))

	switch {
	case w >= 60:
		// Wide/Standard: blank separators between title / detail / actions.
		if hasDetail || hasActions {
			lines = append(lines, "")
		}
		if hasDetail {
			lines = append(lines, "      "+mutedStyle.Render(b.Detail))
		}
		if hasDetail && hasActions {
			lines = append(lines, "")
		}
		if hasActions {
			lines = append(lines, "  "+actionRow)
		}
	case w >= 40:
		// Narrow (40–59): no blank separators, outer blank lines retained.
		if hasDetail {
			lines = append(lines, "      "+mutedStyle.Render(b.Detail))
		}
		if hasActions {
			lines = append(lines, "  "+actionRow)
		}
	default:
		// Unusable (20–39): title + actions only; detail dropped.
		if hasActions {
			lines = append(lines, "  "+actionRow)
		}
	}

	lines = append(lines, "") // bot-pad
	return strings.Join(lines, "\n")
}

// SanitizeErrMsg returns a single-line, ANSI-stripped, ≤80-char string from err.
func SanitizeErrMsg(err error) string {
	if err == nil {
		return ""
	}
	s := err.Error()

	// First line only.
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		s = s[:idx]
	}

	// Extract message from Kubernetes Status JSON body when present.
	if strings.Contains(s, `"kind"`) {
		if start := strings.Index(s, "{"); start >= 0 {
			var status struct {
				Message string `json:"message"`
			}
			if jsonErr := json.Unmarshal([]byte(s[start:]), &status); jsonErr == nil && status.Message != "" {
				s = status.Message
			}
		}
	}

	// Strip ANSI escapes.
	s = data.StripANSI(s)

	// Hard-cap at 80 chars.
	if len(s) > 80 {
		return s[:77] + "…"
	}
	return s
}

// ErrorSeverityOf classifies err using rc's classifier methods.
// Delegates to data.SeverityOf; kept in components for caller ergonomics.
func ErrorSeverityOf(err error, rc data.ResourceClient) data.ErrorSeverity {
	return data.SeverityOf(err, rc)
}

// sanitizedTitleForError returns the canonical title for known Kubernetes error
// types, falling back to fallback when no classifier matches.
func sanitizedTitleForError(err error, fallback string) string {
	if err == nil {
		return fallback
	}
	switch {
	case k8serrors.IsUnauthorized(err):
		return "Session expired"
	case k8serrors.IsForbidden(err):
		return "Permission denied"
	case k8serrors.IsNotFound(err):
		return "Resource not found"
	case errors.Is(err, context.DeadlineExceeded):
		return "Request timed out"
	}
	return fallback
}

// titleAndDetailForError returns the canonical (title, detail) pair without
// redundancy between rows. Classifier-matched errors use §8 canonical strings
// for both fields; generic errors use fallbackTitle + SanitizeErrMsg.
func titleAndDetailForError(err error, fallbackTitle string) (title, detail string) {
	if err == nil {
		return fallbackTitle, ""
	}
	switch {
	case k8serrors.IsUnauthorized(err):
		return "Session expired", "Run `datumctl login` and try again."
	case k8serrors.IsForbidden(err):
		return "Permission denied", "You don't have permission to perform this action."
	case k8serrors.IsNotFound(err):
		return "Resource not found", "The resource has been removed or renamed."
	case errors.Is(err, context.DeadlineExceeded):
		return "Request timed out", "Server did not respond in time."
	}
	return fallbackTitle, SanitizeErrMsg(err)
}

// actionsForSeverity returns the canonical action row for the given severity.
// Warning → [r] retry + [Esc] back; Error → [Esc] back only.
func actionsForSeverity(sev data.ErrorSeverity, backLabel string) []ActionHint {
	if sev == data.ErrorSeverityWarning {
		return []ActionHint{{Key: "r", Label: "retry"}, {Key: "Esc", Label: backLabel}}
	}
	return []ActionHint{{Key: "Esc", Label: backLabel}}
}

// SanitizedTitleForError is the exported form of sanitizedTitleForError for
// model-layer consumers (package tui) that cannot access unexported helpers.
func SanitizedTitleForError(err error, fallback string) string {
	return sanitizedTitleForError(err, fallback)
}

// ActionsForSeverity is the exported form of actionsForSeverity for
// model-layer consumers (package tui) that cannot access unexported helpers.
func ActionsForSeverity(sev data.ErrorSeverity, backLabel string) []ActionHint {
	return actionsForSeverity(sev, backLabel)
}

func errorGlyph(sev data.ErrorSeverity) string {
	if sev == data.ErrorSeverityError {
		return "✕"
	}
	return "⚠"
}

func errorColor(sev data.ErrorSeverity) lipgloss.TerminalColor {
	if sev == data.ErrorSeverityError {
		return styles.Error
	}
	return styles.Warning
}
