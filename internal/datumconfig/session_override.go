package datumconfig

import (
	"fmt"
	"strings"
	"sync"

	customerrors "go.datum.net/datumctl/internal/errors"
)

// SessionEnvVar names the environment variable that runs one datumctl process
// as a given session. The --session flag takes precedence over it.
const SessionEnvVar = "DATUM_SESSION"

// SessionOverrideAnnotation marks a command that changes the active session or
// its login. Such commands reject --session and ignore DATUM_SESSION, since
// acting "as another session" while changing which session is active is
// contradictory.
const SessionOverrideAnnotation = "datumctl.session-override"

// SessionOverrideRejected is the SessionOverrideAnnotation value for commands
// that reject a session override.
const SessionOverrideRejected = "reject"

// SessionOverrideSource records where a session override came from, so output
// can tell the user why the command is not running as the active session.
type SessionOverrideSource string

const (
	SessionOverrideFromFlag SessionOverrideSource = "--session"
	SessionOverrideFromEnv  SessionOverrideSource = SessionEnvVar
)

// sessionOverride is the process-wide session override. It lives only in
// memory: ActiveSessionEntry and CurrentContextEntry consult it, but it is
// never a field of ConfigV1Beta1, so saving the config can never persist it.
var sessionOverride struct {
	mu      sync.RWMutex
	name    string
	source  SessionOverrideSource
	context string // context picked in this process (console switcher)
}

// SetSessionOverride makes every lookup of the active session in this process
// return the named session, without changing the stored active session or
// current context. name must be a session name; resolve user input first with
// ResolveSessionSelector.
func SetSessionOverride(name string, source SessionOverrideSource) {
	sessionOverride.mu.Lock()
	defer sessionOverride.mu.Unlock()
	sessionOverride.name = name
	sessionOverride.source = source
	sessionOverride.context = ""
}

// ClearSessionOverride removes the process-wide session override.
func ClearSessionOverride() {
	SetSessionOverride("", "")
}

// SessionOverride returns the overriding session name and where it came from,
// or ("", "") when the process runs as the stored active session.
func SessionOverride() (string, SessionOverrideSource) {
	sessionOverride.mu.RLock()
	defer sessionOverride.mu.RUnlock()
	return sessionOverride.name, sessionOverride.source
}

// HasSessionOverride reports whether this process runs as an overriding session.
func HasSessionOverride() bool {
	name, _ := SessionOverride()
	return name != ""
}

// SetOverrideContext selects a context for the rest of this process while a
// session override is active, in place of writing current-context to disk. It
// is ignored when no override is active or the context belongs to another
// session.
func SetOverrideContext(name string) {
	sessionOverride.mu.Lock()
	defer sessionOverride.mu.Unlock()
	if sessionOverride.name != "" {
		sessionOverride.context = name
	}
}

func overrideContext() string {
	sessionOverride.mu.RLock()
	defer sessionOverride.mu.RUnlock()
	return sessionOverride.context
}

// ActiveSessionName returns the name of the session this process acts as: the
// session override when one is set, else the stored active session.
func (c *ConfigV1Beta1) ActiveSessionName() string {
	if s := c.ActiveSessionEntry(); s != nil {
		return s.Name
	}
	if name, _ := SessionOverride(); name != "" {
		return name
	}
	return c.ActiveSession
}

// CurrentContextName returns the name of the context this process acts in, or
// "" when there is none. Callers comparing against the current context use it
// instead of the stored CurrentContext so a session override is honored.
func (c *ConfigV1Beta1) CurrentContextName() string {
	if ctx := c.CurrentContextEntry(); ctx != nil {
		return ctx.Name
	}
	return ""
}

// overrideContextEntry returns the context for the overriding session: a
// context picked earlier in this process, else the stored current context when
// it belongs to that session, else the session's last-used context, else nil.
func (c *ConfigV1Beta1) overrideContextEntry(sessionName string) *DiscoveredContext {
	candidates := []string{overrideContext(), c.CurrentContext}
	if s := c.SessionByName(sessionName); s != nil {
		candidates = append(candidates, s.LastContext)
	}
	for _, name := range candidates {
		if name == "" {
			continue
		}
		if ctx := c.ContextByName(name); ctx != nil && ctx.Session == sessionName {
			return ctx
		}
	}
	return nil
}

// ResolveSessionSelector finds the session a --session flag or DATUM_SESSION
// value names: an exact session name (email@api-host), or an email signed in
// on exactly one endpoint. An email signed in on several endpoints, or a value
// matching nothing, returns a UserError listing the valid choices.
func (c *ConfigV1Beta1) ResolveSessionSelector(value string, source SessionOverrideSource) (*Session, error) {
	value = strings.TrimSpace(value)
	if s := c.SessionByName(value); s != nil {
		return s, nil
	}
	matches := c.SessionByEmail(value)
	if len(matches) == 1 {
		return matches[0], nil
	}

	if len(matches) > 1 {
		var b strings.Builder
		b.WriteString("Name the session instead:")
		for _, s := range matches {
			fmt.Fprintf(&b, "\n  --session %s", s.Name)
		}
		return nil, customerrors.NewUserErrorWithHint(
			fmt.Sprintf("%s is signed in on more than one endpoint, so %s %s is ambiguous.", value, source, value),
			b.String(),
		)
	}

	if len(c.Sessions) == 0 {
		return nil, customerrors.NewUserErrorWithHint(
			fmt.Sprintf("No session matches %s %s: no accounts are signed in.", source, value),
			"Run 'datumctl login' to authenticate.",
		)
	}
	var b strings.Builder
	b.WriteString("Valid sessions:")
	for _, s := range c.Sessions {
		fmt.Fprintf(&b, "\n  %s", s.Name)
	}
	b.WriteString("\nPass a session name, or an email signed in on one endpoint. Run 'datumctl auth list' to see accounts.")
	return nil, customerrors.NewUserErrorWithHint(
		fmt.Sprintf("No session matches %s %s.", source, value),
		b.String(),
	)
}

// ApplySessionOverride resolves value against the config at the default path
// and, when it names a session, installs it as the process-wide override. An
// empty value clears any override. It returns the selected session.
func ApplySessionOverride(value string, source SessionOverrideSource) (*Session, error) {
	if strings.TrimSpace(value) == "" {
		ClearSessionOverride()
		return nil, nil
	}
	cfg, err := LoadAuto()
	if err != nil {
		return nil, err
	}
	s, err := cfg.ResolveSessionSelector(value, source)
	if err != nil {
		return nil, err
	}
	SetSessionOverride(s.Name, source)
	return s, nil
}
