package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"go.datum.net/datumctl/internal/console/data"
	"go.datum.net/datumctl/internal/console/styles"
)

// DeleteConfirmationState represents the dialog's state machine step.
type DeleteConfirmationState int

const (
	DeleteStatePrompt DeleteConfirmationState = iota
	DeleteStateInFlight
	DeleteStateForbidden
	DeleteStateConflict
	DeleteStateTransientError
)

// DeleteConfirmationModel is a pure state+view component for the delete
// confirmation dialog. All key routing lives in AppModel.handleOverlayKey.
type DeleteConfirmationModel struct {
	target        data.DeleteTarget
	sanitizedName string
	kindDisplay   string
	state         DeleteConfirmationState
	errorDetail   string
	spinner       spinner.Model
}

// NewDeleteConfirmationModel constructs the dialog for the given target.
// kindDisplay is resolved from FB-014 registrations; fallback is target.RT.Kind.
func NewDeleteConfirmationModel(target data.DeleteTarget, registrations []data.ResourceRegistration) DeleteConfirmationModel {
	kind := data.ResolveDescription(registrations, target.RT.Group, target.RT.Name)
	if kind == "" {
		kind = target.RT.Kind
	}
	s := spinner.New()
	s.Spinner = spinner.Dot
	return DeleteConfirmationModel{
		target:        target,
		sanitizedName: data.SanitizeResourceName(target.Name),
		kindDisplay:   kind,
		state:         DeleteStatePrompt,
		spinner:       s,
	}
}

func (m DeleteConfirmationModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m DeleteConfirmationModel) Update(msg tea.Msg) (DeleteConfirmationModel, tea.Cmd) {
	if m.state == DeleteStateInFlight {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

// State returns the current dialog state.
func (m DeleteConfirmationModel) State() DeleteConfirmationState { return m.state }

// Target returns the delete target.
func (m DeleteConfirmationModel) Target() data.DeleteTarget { return m.target }

// InFlight reports whether a delete call is in progress.
func (m DeleteConfirmationModel) InFlight() bool { return m.state == DeleteStateInFlight }

// SetState transitions the dialog to the given state.
func (m *DeleteConfirmationModel) SetState(s DeleteConfirmationState) { m.state = s }

// SetErrorDetail sets the sanitized error string for TransientError rendering.
func (m *DeleteConfirmationModel) SetErrorDetail(s string) { m.errorDetail = s }

// View renders the dialog centered over the terminal. appWidth and appHeight
// are the full terminal dimensions (passed from AppModel.View).
func (m DeleteConfirmationModel) View(appWidth, appHeight int) string {
	dialogW := dialogWidth(appWidth)
	inner := m.renderInner(dialogW)

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Accent).
		Padding(1, 2).
		Width(dialogW)

	box := border.Render(inner)

	return lipgloss.Place(appWidth, appHeight,
		lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceBackground(styles.OverlayBackdrop),
	)
}

// dialogWidth computes the dialog content width from the terminal width,
// implementing the three-band spec (§6a).
func dialogWidth(appWidth int) int {
	switch {
	case appWidth >= 120:
		return 70
	case appWidth >= 100:
		return 60
	case appWidth >= 80:
		return 50
	case appWidth >= 44:
		return appWidth - 30
	default:
		if appWidth-4 > 0 {
			return appWidth - 4
		}
		return 20
	}
}

// wideMode returns true when the dialog content width supports single-line hints.
func wideMode(dialogW int) bool { return dialogW >= 60 }

// narrowMode returns true when the dialog is too narrow for Kind display.
func narrowMode(dialogW int) bool { return dialogW < 40 }

func (m DeleteConfirmationModel) renderInner(dialogW int) string {
	accent := lipgloss.NewStyle().Foreground(styles.Accent).Bold(true)
	muted := lipgloss.NewStyle().Foreground(styles.Muted)
	warn := lipgloss.NewStyle().Foreground(styles.Warning)

	var lines []string

	// Title line.
	lines = append(lines, m.renderTitle(dialogW, accent, muted))
	lines = append(lines, "")

	// Namespace line (namespaced resources only).
	if m.target.Namespace != "" && !narrowMode(dialogW) {
		lines = append(lines, muted.Render("Namespace: ")+m.target.Namespace)
		lines = append(lines, "")
	}

	// State-specific body + keybind footer.
	switch m.state {
	case DeleteStatePrompt:
		lines = append(lines, muted.Render("This action cannot be undone."))
		lines = append(lines, "")
		lines = append(lines, m.renderPromptHints(dialogW, accent, muted))

	case DeleteStateInFlight:
		lines = append(lines, muted.Render(m.spinner.View()+" Deleting…"))
		lines = append(lines, "")
		lines = append(lines, muted.Render("[Esc] close (delete continues)"))

	case DeleteStateForbidden:
		lines = append(lines, warn.Render("⚠ Cannot delete — insufficient permissions"))
		lines = append(lines, "")
		lines = append(lines, muted.Render("[Esc] close"))

	case DeleteStateConflict:
		lines = append(lines, warn.Render("⚠ Resource was modified since load"))
		lines = append(lines, "")
		lines = append(lines, muted.Render("[r] refresh   [Esc] cancel"))

	case DeleteStateTransientError:
		errLine := fmt.Sprintf("⚠ Delete failed — %s", m.truncatedError(dialogW))
		lines = append(lines, warn.Render(errLine))
		lines = append(lines, "")
		lines = append(lines, muted.Render("[r] retry   [Esc] cancel"))
	}

	return strings.Join(lines, "\n")
}

func (m DeleteConfirmationModel) renderTitle(dialogW int, accent, muted lipgloss.Style) string {
	if narrowMode(dialogW) {
		// Narrow: "Delete <ns>/<name>?" or "Delete <name>?"
		if m.target.Namespace != "" {
			return accent.Render("Delete ") + muted.Render(`"`+m.target.Namespace+"/"+m.sanitizedName+`"?`)
		}
		return accent.Render("Delete ") + muted.Render(`"`+m.sanitizedName+`"?`)
	}
	return accent.Render("Delete ") + m.kindDisplay + " " + muted.Render(`"`+m.sanitizedName+`"?`)
}

func (m DeleteConfirmationModel) renderPromptHints(dialogW int, accent, muted lipgloss.Style) string {
	confirm := accent.Render("[Y]") + muted.Render(" confirm")
	cancel := accent.Render("[N]") + muted.Render("/") + accent.Render("[Esc]") + muted.Render(" cancel")
	if wideMode(dialogW) {
		return confirm + muted.Render("   ") + cancel
	}
	return confirm + "\n" + cancel
}

func (m DeleteConfirmationModel) truncatedError(dialogW int) string {
	s := data.SanitizeResourceName(m.errorDetail)
	maxLen := dialogW - 20
	if maxLen < 10 {
		maxLen = 10
	}
	if len(s) > maxLen {
		return s[:maxLen] + "…"
	}
	return s
}
