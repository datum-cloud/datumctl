package components

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"go.datum.net/datumctl/internal/console/styles"
)

// loginSuccessGlyph and loginFailGlyph are the consistent status indicators
// used in the login overlay. Keeping them as named constants ensures every
// state line references the same character rather than scattered literals.
const (
	loginSuccessGlyph = "✓"
	loginFailGlyph    = "✗"
)

// LoginOverlayState tracks which phase the device auth flow is in.
type LoginOverlayState int

const (
	LoginOverlayInitializing LoginOverlayState = iota // waiting for device auth to start
	LoginOverlayPending                               // device code shown, waiting for user
	LoginOverlayComplete                              // auth succeeded (transient)
	LoginOverlayFailed                                // auth failed
)

// LoginOverlayModel is an overlay that walks the user through the device code
// authentication flow without leaving the TUI.
type LoginOverlayModel struct {
	Width  int
	Height int

	State           LoginOverlayState
	AuthHostname    string // hostname shown in the initializing state line
	VerificationURI string
	UserCode        string
	ErrMsg          string
	SpinnerFrame    string
}

func NewLoginOverlayModel(authHostname string) LoginOverlayModel {
	return LoginOverlayModel{
		State:        LoginOverlayInitializing,
		AuthHostname: authHostname,
	}
}

func (m LoginOverlayModel) View() string {
	accent := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Accent).Bold(true)
	muted := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Muted)
	secondary := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Secondary)
	success := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Success).Bold(true)
	errStyle := lipgloss.NewStyle().Background(styles.Surface).Foreground(styles.Error)

	var lines []string

	lines = append(lines, accent.Render("Log in to Datum Cloud"))
	lines = append(lines, "")

	host := m.AuthHostname
	if host == "" {
		host = "auth.datum.net"
	}

	switch m.State {
	case LoginOverlayInitializing:
		lines = append(lines, muted.Render(m.SpinnerFrame+" Connecting to "+host+"..."))

	case LoginOverlayPending:
		lines = append(lines, secondary.Render("Open this URL in your browser:"))
		lines = append(lines, "")
		lines = append(lines, accent.Render("  "+m.VerificationURI))
		lines = append(lines, "")
		if m.UserCode != "" {
			lines = append(lines, secondary.Render("Enter code:")+muted.Render("  ")+accent.Render(m.UserCode))
			lines = append(lines, "")
		}
		lines = append(lines, muted.Render(m.SpinnerFrame+" Waiting for authentication..."))
		lines = append(lines, "")
		lines = append(lines, muted.Render("[b] open in browser"))

	case LoginOverlayComplete:
		lines = append(lines, success.Render(loginSuccessGlyph+" Authentication successful"))
		lines = append(lines, "")
		lines = append(lines, muted.Render("Loading your resources..."))

	case LoginOverlayFailed:
		lines = append(lines, errStyle.Render(loginFailGlyph+" Authentication failed"))
		lines = append(lines, "")
		lines = append(lines, muted.Render(m.ErrMsg))
		lines = append(lines, "")
		lines = append(lines, muted.Render("[Esc] dismiss   [l] try again"))
	}

	// Determine overlay width — minimum 80 so a typical verification URL fits without
	// wrapping, then expand to fit any longer content up to terminal width minus chrome.
	maxW := 80
	for _, l := range lines {
		if w := lipgloss.Width(l); w > maxW {
			maxW = w
		}
	}
	if m.Width > 0 && maxW > m.Width-8 {
		maxW = m.Width - 8
	}

	body := strings.Join(lines, "\n")
	return styles.OverlayStyle.Width(maxW).Render(body)
}

// StatusLine returns a one-line summary for embedding in the status bar.
func (m LoginOverlayModel) StatusLine() string {
	switch m.State {
	case LoginOverlayInitializing:
		return fmt.Sprintf("%s Connecting...", m.SpinnerFrame)
	case LoginOverlayPending:
		return fmt.Sprintf("%s Waiting for authentication — [b] open browser", m.SpinnerFrame)
	case LoginOverlayFailed:
		return "Login failed — [Esc] dismiss  [l] retry"
	default:
		return ""
	}
}
