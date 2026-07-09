package serviceactivation

// Exit codes for the shared entitlement-state contract. They live at 10+,
// clear of ipam's documented domain codes at 2–8; 0, 1, and 9 mean the same
// thing across schemes. Plugins are exec'd binaries, so the plugin's entry
// point maps these onto the process exit status.
const (
	// ExitOK is a command that ran, an access request submitted via the explicit
	// request verb (even if pending), or a status query.
	ExitOK = 0

	// ExitError is a generic, transport, or API error.
	ExitError = 1

	// ExitDeclined means the user declined the interactive request prompt.
	ExitDeclined = 9

	// ExitNotEnabled means the service is not enabled and no request was made —
	// the non-interactive gate path. Run the access request verb.
	ExitNotEnabled = 10

	// ExitDeniedOrRevoked means access was denied or revoked; terminal without
	// an explicit renew.
	ExitDeniedOrRevoked = 11

	// ExitPending means a gated command was blocked awaiting processing or
	// approval (including immediately after an interactive submit), or a --wait
	// timed out while still pending. Safe to re-run.
	ExitPending = 12

	// ExitUnavailable means the service is not available on this platform
	// environment (unpublished, or the enablement API is absent).
	ExitUnavailable = 13
)

// Error carries a preformatted, user-facing message together with the process
// exit code the plugin should surface and the classified State that produced it.
type Error struct {
	// Code is the intended process exit status.
	Code int
	// State is the activation state that produced this error, when applicable.
	State State
	// Message is the complete, user-facing text (may span multiple lines). It is
	// written to stderr by the caller; it must not be prefixed with "Error:" for
	// states that are not failures.
	Message string
	// Cause is an optional wrapped error for logging/unwrapping.
	Cause error
}

func (e *Error) Error() string { return e.Message }

func (e *Error) Unwrap() error { return e.Cause }

// ExitCode returns the process exit status a caller should use.
func (e *Error) ExitCode() int { return e.Code }

// ExitCodeOf returns the exit code carried by err, defaulting to ExitError for
// any non-nil error that is not an *Error, and ExitOK for nil.
func ExitCodeOf(err error) int {
	if err == nil {
		return ExitOK
	}
	if e, ok := err.(*Error); ok {
		return e.Code
	}
	return ExitError
}

// newError builds an *Error.
func newError(code int, state State, message string, cause error) *Error {
	return &Error{Code: code, State: state, Message: message, Cause: cause}
}
