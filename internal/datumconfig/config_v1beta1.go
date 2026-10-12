package datumconfig

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/yaml"
)

const (
	V1Beta1APIVersion = "datumctl.config.datum.net/v1beta1"
)

// ConfigV1Beta1 is the v1beta1 config format with session-based auth and the
// user's selected contexts.
//
// Contexts holds only selections: the current context and each session's last
// context. The orgs and projects a user can reach are always fetched live, so
// configs written by older releases that also stored every discovered context
// and a display-name cache still load; the extra entries are dropped on save.
type ConfigV1Beta1 struct {
	APIVersion     string              `json:"apiVersion" yaml:"apiVersion"`
	Kind           string              `json:"kind" yaml:"kind"`
	Sessions       []Session           `json:"sessions,omitempty" yaml:"sessions,omitempty"`
	Contexts       []DiscoveredContext `json:"contexts,omitempty" yaml:"contexts,omitempty"`
	CurrentContext string              `json:"current-context,omitempty" yaml:"current-context,omitempty"`
	ActiveSession  string              `json:"active-session,omitempty" yaml:"active-session,omitempty"`
	AutoUpdate     bool                `json:"auto-update,omitempty" yaml:"auto-update,omitempty"`
}

// Session represents one authenticated login. Each login to an endpoint creates
// a session. This replaces the cluster + user entries from v1alpha1.
type Session struct {
	Name        string   `json:"name" yaml:"name"`
	UserKey     string   `json:"user-key" yaml:"user-key"`
	UserEmail   string   `json:"user-email" yaml:"user-email"`
	UserName    string   `json:"user-name,omitempty" yaml:"user-name,omitempty"`
	Endpoint    Endpoint `json:"endpoint" yaml:"endpoint"`
	LastContext string   `json:"last-context,omitempty" yaml:"last-context,omitempty"`
}

// Endpoint holds connection details for an API server, bound to a login session.
type Endpoint struct {
	Server                   string `json:"server" yaml:"server"`
	AuthHostname             string `json:"auth-hostname" yaml:"auth-hostname"`
	TLSServerName            string `json:"tls-server-name,omitempty" yaml:"tls-server-name,omitempty"`
	InsecureSkipTLSVerify    bool   `json:"insecure-skip-tls-verify,omitempty" yaml:"insecure-skip-tls-verify,omitempty"`
	CertificateAuthorityData string `json:"certificate-authority-data,omitempty" yaml:"certificate-authority-data,omitempty"`
}

// DiscoveredContext is an org or project context a user can select.
//
// Name is a session-qualified unique key of the form "session/ref" (see
// QualifiedContextName). Because org and project IDs overlap across
// environments — staging and production can both expose "datum/datum-cloud" —
// the bare Ref() is only unique within a single session, so the stored Name
// carries the owning session to keep entries from different logins distinct.
type DiscoveredContext struct {
	Name           string `json:"name" yaml:"name"`
	Session        string `json:"session" yaml:"session"`
	OrganizationID string `json:"organization-id" yaml:"organization-id"`
	ProjectID      string `json:"project-id,omitempty" yaml:"project-id,omitempty"`
	Namespace      string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
}

// Ref returns the canonical, session-relative reference string for this context
// — "orgID" for org contexts or "orgID/projectID" for project contexts. This is
// the value users pass to "datumctl ctx use". It is unique only within a
// session; use Name (or QualifiedContextName) for a globally-unique key.
func (c *DiscoveredContext) Ref() string {
	if c.ProjectID != "" {
		return c.OrganizationID + "/" + c.ProjectID
	}
	return c.OrganizationID
}

// QualifiedContextName builds the session-qualified unique key for a context
// from its owning session name and its session-relative ref. Session names
// never contain "/", so the first segment always recovers the session.
func QualifiedContextName(sessionName, ref string) string {
	return sessionName + "/" + ref
}

// QualifiedName returns the session-qualified unique key for this context.
func (c *DiscoveredContext) QualifiedName() string {
	return QualifiedContextName(c.Session, c.Ref())
}

// FormatWithID returns "displayName (resourceID)" when the display name differs
// from the resource ID, or just the resource ID when they match. Used for
// consistent human-friendly output across commands.
func FormatWithID(displayName, resourceID string) string {
	if displayName != "" && displayName != resourceID {
		return fmt.Sprintf("%s (%s)", displayName, resourceID)
	}
	return resourceID
}

func NewV1Beta1() *ConfigV1Beta1 {
	return &ConfigV1Beta1{
		APIVersion: V1Beta1APIVersion,
		Kind:       DefaultKind,
	}
}

func (c *ConfigV1Beta1) ensureDefaults() {
	if c.APIVersion == "" {
		c.APIVersion = V1Beta1APIVersion
	}
	if c.Kind == "" {
		c.Kind = DefaultKind
	}
}

// SessionByName returns the session with the given name, or nil.
func (c *ConfigV1Beta1) SessionByName(name string) *Session {
	for i := range c.Sessions {
		if c.Sessions[i].Name == name {
			return &c.Sessions[i]
		}
	}
	return nil
}

// SessionByUserKey returns the first session matching the given user key.
func (c *ConfigV1Beta1) SessionByUserKey(userKey string) *Session {
	for i := range c.Sessions {
		if c.Sessions[i].UserKey == userKey {
			return &c.Sessions[i]
		}
	}
	return nil
}

// SessionByEmail returns all sessions matching the given email.
func (c *ConfigV1Beta1) SessionByEmail(email string) []*Session {
	var sessions []*Session
	for i := range c.Sessions {
		if c.Sessions[i].UserEmail == email {
			sessions = append(sessions, &c.Sessions[i])
		}
	}
	return sessions
}

// SwitchArgs returns the arguments that select session s with
// "datumctl auth switch": the email alone when it is unique, or the email plus
// "--endpoint <host>" when the same email is signed in on several endpoints.
func (c *ConfigV1Beta1) SwitchArgs(s *Session) string {
	if len(c.SessionByEmail(s.UserEmail)) > 1 {
		return fmt.Sprintf("%s --endpoint %s", s.UserEmail, StripScheme(s.Endpoint.Server))
	}
	return s.UserEmail
}

// ContextByName returns the context with the given name, or nil.
func (c *ConfigV1Beta1) ContextByName(name string) *DiscoveredContext {
	for i := range c.Contexts {
		if c.Contexts[i].Name == name {
			return &c.Contexts[i]
		}
	}
	return nil
}

// CurrentContextEntry returns the active context, or nil if none is set.
// Under a session override (see SetSessionOverride) it returns a context owned
// by the overriding session, falling back to that session's last-used context,
// so a command never pairs one session's credentials with another's scope.
func (c *ConfigV1Beta1) CurrentContextEntry() *DiscoveredContext {
	if name, _ := SessionOverride(); name != "" {
		return c.overrideContextEntry(name)
	}
	if c.CurrentContext == "" {
		return nil
	}
	return c.ContextByName(c.CurrentContext)
}

// ActiveSessionEntry returns the active session. The current context's session
// is authoritative: whatever context is selected determines which environment
// is active, so whoami and every request agree. The stored ActiveSession is
// only a fallback for when no current context resolves (e.g. right after login
// before a context is picked). A session override (--session or DATUM_SESSION)
// takes precedence over both for the life of the process.
func (c *ConfigV1Beta1) ActiveSessionEntry() *Session {
	if name, _ := SessionOverride(); name != "" {
		return c.SessionByName(name)
	}
	if ctx := c.CurrentContextEntry(); ctx != nil {
		if s := c.SessionByName(ctx.Session); s != nil {
			return s
		}
	}
	if c.ActiveSession != "" {
		if s := c.SessionByName(c.ActiveSession); s != nil {
			return s
		}
	}
	return nil
}

// UpsertSession creates or updates a session by name.
func (c *ConfigV1Beta1) UpsertSession(s Session) {
	for i := range c.Sessions {
		if c.Sessions[i].Name == s.Name {
			c.Sessions[i] = s
			return
		}
	}
	c.Sessions = append(c.Sessions, s)
}

// SelectContext makes ctx the current context, records it as its session's last
// context, and makes that session active. A namespace already recorded for the
// same context is kept.
func (c *ConfigV1Beta1) SelectContext(ctx DiscoveredContext) {
	if existing := c.ContextByName(ctx.Name); existing != nil && existing.Namespace != "" {
		ctx.Namespace = existing.Namespace
	}
	c.UpsertContext(ctx)
	c.CurrentContext = ctx.Name
	// The active session is always the current context's session; keep the
	// stored fallback in lockstep so whoami and requests never diverge.
	c.ActiveSession = ctx.Session
	if s := c.SessionByName(ctx.Session); s != nil {
		s.LastContext = ctx.Name
	}
}

// pruneUnselectedContexts drops contexts that are neither current nor some
// session's last context.
func (c *ConfigV1Beta1) pruneUnselectedContexts() {
	keep := map[string]bool{c.CurrentContext: true}
	for _, s := range c.Sessions {
		keep[s.LastContext] = true
	}
	contexts := make([]DiscoveredContext, 0, len(c.Contexts))
	for _, ctx := range c.Contexts {
		if keep[ctx.Name] {
			contexts = append(contexts, ctx)
		}
	}
	c.Contexts = contexts
}

// UpsertContext creates or updates a context by name.
func (c *ConfigV1Beta1) UpsertContext(ctx DiscoveredContext) {
	for i := range c.Contexts {
		if c.Contexts[i].Name == ctx.Name {
			c.Contexts[i] = ctx
			return
		}
	}
	c.Contexts = append(c.Contexts, ctx)
}

// RemoveSession removes a session and all contexts referencing it.
func (c *ConfigV1Beta1) RemoveSession(name string) {
	sessions := make([]Session, 0, len(c.Sessions))
	for _, s := range c.Sessions {
		if s.Name != name {
			sessions = append(sessions, s)
		}
	}
	c.Sessions = sessions

	contexts := make([]DiscoveredContext, 0, len(c.Contexts))
	for _, ctx := range c.Contexts {
		if ctx.Session != name {
			contexts = append(contexts, ctx)
		}
	}
	c.Contexts = contexts

	if c.ActiveSession == name {
		c.ActiveSession = ""
	}
}

// RemoveSessionsByEmail removes all sessions (and their contexts) matching the email.
func (c *ConfigV1Beta1) RemoveSessionsByEmail(email string) {
	sessionNames := make(map[string]bool)
	sessions := make([]Session, 0, len(c.Sessions))
	for _, s := range c.Sessions {
		if s.UserEmail == email {
			sessionNames[s.Name] = true
		} else {
			sessions = append(sessions, s)
		}
	}
	c.Sessions = sessions

	contexts := make([]DiscoveredContext, 0, len(c.Contexts))
	for _, ctx := range c.Contexts {
		if !sessionNames[ctx.Session] {
			contexts = append(contexts, ctx)
		}
	}
	c.Contexts = contexts

	if sessionNames[c.ActiveSession] {
		c.ActiveSession = ""
	}

	// Clear current context if it belonged to a removed session.
	if c.CurrentContext != "" {
		if ctx := c.ContextByName(c.CurrentContext); ctx == nil {
			c.CurrentContext = ""
		}
	}
}

// HasMultipleEndpoints returns true if sessions span more than one endpoint server.
func (c *ConfigV1Beta1) HasMultipleEndpoints() bool {
	servers := make(map[string]bool)
	for _, s := range c.Sessions {
		servers[s.Endpoint.Server] = true
	}
	return len(servers) > 1
}

// LoadAuto loads the v1beta1 config from the default path. The name is kept
// for compatibility with earlier code that anticipated multi-version handling;
// today only v1beta1 is supported.
func LoadAuto() (*ConfigV1Beta1, error) {
	return LoadV1Beta1()
}

// LoadAutoFromPath loads the v1beta1 config from the given path.
func LoadAutoFromPath(path string) (*ConfigV1Beta1, error) {
	return LoadV1Beta1FromPath(path)
}

// LoadV1Beta1 loads a v1beta1 config from the default path.
func LoadV1Beta1() (*ConfigV1Beta1, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	return LoadV1Beta1FromPath(path)
}

// LoadV1Beta1FromPath loads a v1beta1 config from the given path.
func LoadV1Beta1FromPath(path string) (*ConfigV1Beta1, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return NewV1Beta1(), nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	if len(strings.TrimSpace(string(data))) == 0 {
		return NewV1Beta1(), nil
	}

	cfg := NewV1Beta1()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	cfg.ensureDefaults()
	cfg.migrateSessionScoping()
	return cfg, nil
}

// migrateSessionScoping upgrades a config written before contexts were
// session-qualified. It rewrites bare-ref context names to the session-qualified
// form. It is idempotent and safe to run repeatedly,
// including on partially-migrated configs, so no re-login is required.
func (c *ConfigV1Beta1) migrateSessionScoping() {
	for i := range c.Contexts {
		ctx := &c.Contexts[i]
		if ctx.Session == "" {
			continue
		}
		want := ctx.QualifiedName()
		if ctx.Name == want {
			continue
		}
		old := ctx.Name
		ctx.Name = want
		// Repoint the pointers that addressed the old bare-ref name.
		if c.CurrentContext == old {
			c.CurrentContext = want
		}
		for j := range c.Sessions {
			if c.Sessions[j].LastContext == old {
				c.Sessions[j].LastContext = want
			}
		}
	}

}

// SaveV1Beta1 saves a v1beta1 config to the default path.
func SaveV1Beta1(cfg *ConfigV1Beta1) error {
	path, err := DefaultPath()
	if err != nil {
		return err
	}
	return SaveV1Beta1ToPath(cfg, path)
}

// SaveV1Beta1ToPath saves a v1beta1 config to the given path.
func SaveV1Beta1ToPath(cfg *ConfigV1Beta1, path string) error {
	if cfg == nil {
		return errors.New("config is nil")
	}
	cfg.ensureDefaults()
	cfg.pruneUnselectedContexts()

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("ensure config dir: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write temp config: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}

	return nil
}

// SessionName generates a canonical session name from email and API hostname.
func SessionName(email, apiHostname string) string {
	return fmt.Sprintf("%s@%s", email, StripScheme(apiHostname))
}
