package datumconfig

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"sigs.k8s.io/yaml"
)

const (
	V1Beta1APIVersion = "datumctl.config.datum.net/v1beta1"
)

// ConfigV1Beta1 is the v1beta1 config format with session-based auth and
// API-discovered contexts.
type ConfigV1Beta1 struct {
	APIVersion     string              `json:"apiVersion" yaml:"apiVersion"`
	Kind           string              `json:"kind" yaml:"kind"`
	Sessions       []Session           `json:"sessions,omitempty" yaml:"sessions,omitempty"`
	Contexts       []DiscoveredContext `json:"contexts,omitempty" yaml:"contexts,omitempty"`
	CurrentContext string              `json:"current-context,omitempty" yaml:"current-context,omitempty"`
	ActiveSession  string              `json:"active-session,omitempty" yaml:"active-session,omitempty"`
	Cache          ContextCache        `json:"cache" yaml:"cache,omitempty"`
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

// DiscoveredContext is a context entry derived from the API. Names follow the
// format "orgID" for org-scoped or "orgID/projectID" for project-scoped.
type DiscoveredContext struct {
	Name           string `json:"name" yaml:"name"`
	Session        string `json:"session" yaml:"session"`
	OrganizationID string `json:"organization-id" yaml:"organization-id"`
	ProjectID      string `json:"project-id,omitempty" yaml:"project-id,omitempty"`
	Namespace      string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
}

// Ref returns the canonical reference string for this context — "orgID" for org
// contexts or "orgID/projectID" for project contexts. This is the value users
// pass to "datumctl ctx use".
func (c *DiscoveredContext) Ref() string {
	if c.ProjectID != "" {
		return c.OrganizationID + "/" + c.ProjectID
	}
	return c.OrganizationID
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

// DisplayRef returns a human-friendly label for this context, using cached
// display names where available. Falls back to Ref() when no display names
// are cached.
func (c *ConfigV1Beta1) DisplayRef(ctx *DiscoveredContext) string {
	orgLabel := ctx.OrganizationID
	for _, o := range c.Cache.Organizations {
		if o.ID == ctx.OrganizationID && o.DisplayName != "" {
			orgLabel = o.DisplayName
			break
		}
	}
	if ctx.ProjectID == "" {
		return orgLabel
	}
	projLabel := ctx.ProjectID
	for _, p := range c.Cache.Projects {
		if p.ID == ctx.ProjectID && p.DisplayName != "" {
			projLabel = p.DisplayName
			break
		}
	}
	return orgLabel + "/" + projLabel
}

// OrgDisplayName returns the cached display name for an org, or the ID if none.
func (c *ConfigV1Beta1) OrgDisplayName(orgID string) string {
	for _, o := range c.Cache.Organizations {
		if o.ID == orgID && o.DisplayName != "" {
			return o.DisplayName
		}
	}
	return orgID
}

// ProjectDisplayName returns the cached display name for a project, or the ID if none.
func (c *ConfigV1Beta1) ProjectDisplayName(projectID string) string {
	for _, p := range c.Cache.Projects {
		if p.ID == projectID && p.DisplayName != "" {
			return p.DisplayName
		}
	}
	return projectID
}

// ContextCache stores API-discovered orgs and projects with a staleness timestamp.
type ContextCache struct {
	Organizations []CachedOrg     `json:"organizations,omitempty" yaml:"organizations,omitempty"`
	Projects      []CachedProject `json:"projects,omitempty" yaml:"projects,omitempty"`
	LastRefreshed *time.Time      `json:"last-refreshed,omitempty" yaml:"last-refreshed,omitempty"`
}

// CachedOrg is an API-discovered organization.
type CachedOrg struct {
	ID          string `json:"id" yaml:"id"`
	DisplayName string `json:"display-name,omitempty" yaml:"display-name,omitempty"`
}

// CachedProject is an API-discovered project under an org.
type CachedProject struct {
	ID          string `json:"id" yaml:"id"`
	DisplayName string `json:"display-name,omitempty" yaml:"display-name,omitempty"`
	OrgID       string `json:"org-id" yaml:"org-id"`
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

// ContextByName returns the context with the given name, or nil.
func (c *ConfigV1Beta1) ContextByName(name string) *DiscoveredContext {
	for i := range c.Contexts {
		if c.Contexts[i].Name == name {
			return &c.Contexts[i]
		}
	}
	return nil
}

// ResolveContext finds a context by flexible matching. Resource IDs always
// take precedence over display names. It tries, in order:
//
//  1. Exact context name match
//  2. orgID/projectID match (for "org/project" queries)
//  3. orgID-only match for org-level contexts
//  4. projectID-only match if unambiguous
//  5. Display-name match on org + project (scoped together, only if unambiguous)
//  6. Display-name-only org or project match (unambiguous)
//
// Returns nil if no match, or if a display-name match is ambiguous.
func (c *ConfigV1Beta1) ResolveContext(query string) *DiscoveredContext {
	// 1. Exact name match.
	if ctx := c.ContextByName(query); ctx != nil {
		return ctx
	}

	orgPart, projPart, hasSlash := strings.Cut(query, "/")

	if hasSlash {
		// 2. orgID/projectID match.
		for i := range c.Contexts {
			ctx := &c.Contexts[i]
			if ctx.OrganizationID == orgPart && ctx.ProjectID == projPart {
				return ctx
			}
		}

		// 5. Display-name match, scoped: resolve orgPart to an org ID first,
		// then scope project display-name resolution to that org.
		resolvedOrgIDs := c.resolveOrgIDs(orgPart)
		if len(resolvedOrgIDs) == 0 {
			// orgPart might already be a resource ID even though the slash path
			// didn't match — allow it through as a search scope.
			resolvedOrgIDs = []string{orgPart}
		}

		var match *DiscoveredContext
		for _, orgID := range resolvedOrgIDs {
			projIDs := c.resolveProjectIDsInOrg(projPart, orgID)
			// Also include projPart as a literal ID candidate within this org.
			projIDs = appendUnique(projIDs, projPart)
			for _, projID := range projIDs {
				for i := range c.Contexts {
					ctx := &c.Contexts[i]
					if ctx.OrganizationID == orgID && ctx.ProjectID == projID {
						if match != nil && match != ctx {
							return nil // ambiguous
						}
						match = ctx
					}
				}
			}
		}
		return match
	}

	// 3. orgID-only match (org-level contexts).
	for i := range c.Contexts {
		ctx := &c.Contexts[i]
		if ctx.OrganizationID == query && ctx.ProjectID == "" {
			return ctx
		}
	}

	// 4. projectID-only match if unambiguous (resource IDs only, no display names).
	var idMatch *DiscoveredContext
	for i := range c.Contexts {
		ctx := &c.Contexts[i]
		if ctx.ProjectID == query {
			if idMatch != nil {
				return nil // ambiguous on resource ID
			}
			idMatch = ctx
		}
	}
	if idMatch != nil {
		return idMatch
	}

	// 6a. Display-name-only org match (unambiguous).
	resolvedOrgIDs := c.resolveOrgIDs(query)
	if len(resolvedOrgIDs) == 1 {
		for i := range c.Contexts {
			ctx := &c.Contexts[i]
			if ctx.OrganizationID == resolvedOrgIDs[0] && ctx.ProjectID == "" {
				return ctx
			}
		}
	} else if len(resolvedOrgIDs) > 1 {
		return nil // ambiguous display name
	}

	// 6b. Display-name-only project match (unambiguous).
	resolvedProjIDs := c.resolveProjectIDs(query)
	if len(resolvedProjIDs) == 1 {
		for i := range c.Contexts {
			ctx := &c.Contexts[i]
			if ctx.ProjectID == resolvedProjIDs[0] {
				return ctx
			}
		}
	}
	return nil
}

// resolveOrgIDs returns all org resource IDs whose display name matches.
func (c *ConfigV1Beta1) resolveOrgIDs(displayName string) []string {
	var ids []string
	for _, o := range c.Cache.Organizations {
		if o.DisplayName == displayName && o.DisplayName != o.ID {
			ids = append(ids, o.ID)
		}
	}
	return ids
}

// resolveProjectIDs returns all project resource IDs whose display name matches.
func (c *ConfigV1Beta1) resolveProjectIDs(displayName string) []string {
	var ids []string
	for _, p := range c.Cache.Projects {
		if p.DisplayName == displayName && p.DisplayName != p.ID {
			ids = append(ids, p.ID)
		}
	}
	return ids
}

// resolveProjectIDsInOrg returns project resource IDs whose display name
// matches and which belong to the given org.
func (c *ConfigV1Beta1) resolveProjectIDsInOrg(displayName, orgID string) []string {
	var ids []string
	for _, p := range c.Cache.Projects {
		if p.OrgID == orgID && p.DisplayName == displayName && p.DisplayName != p.ID {
			ids = append(ids, p.ID)
		}
	}
	return ids
}

func appendUnique(s []string, v string) []string {
	if slices.Contains(s, v) {
		return s
	}
	return append(s, v)
}

// CurrentContextEntry returns the active context, or nil if none is set.
func (c *ConfigV1Beta1) CurrentContextEntry() *DiscoveredContext {
	if c.CurrentContext == "" {
		return nil
	}
	return c.ContextByName(c.CurrentContext)
}

// ActiveSessionEntry returns the active session. It first checks ActiveSession,
// then falls back to the session referenced by the current context.
func (c *ConfigV1Beta1) ActiveSessionEntry() *Session {
	if c.ActiveSession != "" {
		if s := c.SessionByName(c.ActiveSession); s != nil {
			return s
		}
	}
	ctx := c.CurrentContextEntry()
	if ctx != nil {
		return c.SessionByName(ctx.Session)
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

// ContextsForSession returns all contexts belonging to a session.
func (c *ConfigV1Beta1) ContextsForSession(sessionName string) []DiscoveredContext {
	var result []DiscoveredContext
	for _, ctx := range c.Contexts {
		if ctx.Session == sessionName {
			result = append(result, ctx)
		}
	}
	return result
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
	return cfg, nil
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
