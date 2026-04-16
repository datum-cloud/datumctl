package discovery

import (
	"context"
	"fmt"
	"time"

	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/datumconfig"
)

// DefaultStaleness is the cache age after which a warning is shown.
const DefaultStaleness = 24 * time.Hour

// UpdateConfigCache is a convenience wrapper that performs a full refresh for
// a session: merges discovery into the cache, regenerates the session's
// contexts, and garbage-collects stale entries.
func UpdateConfigCache(
	cfg *datumconfig.ConfigV1Beta1,
	sessionName string,
	orgs []DiscoveredOrg,
	projects []DiscoveredProject,
) {
	now := time.Now().UTC()
	cfg.Cache.LastRefreshed = &now

	MergeCacheFromDiscovery(cfg, orgs, projects)
	SyncContextsForSession(cfg, sessionName, orgs, projects)
	GCCache(cfg)
}

// MergeCacheFromDiscovery updates the cache with newly discovered orgs and
// projects. Existing entries are updated; unmentioned entries are preserved.
func MergeCacheFromDiscovery(
	cfg *datumconfig.ConfigV1Beta1,
	orgs []DiscoveredOrg,
	projects []DiscoveredProject,
) {
	for _, o := range orgs {
		upsertCachedOrg(&cfg.Cache, datumconfig.CachedOrg{
			ID:          o.Name,
			DisplayName: o.DisplayName,
		})
	}
	for _, p := range projects {
		upsertCachedProject(&cfg.Cache, datumconfig.CachedProject{
			ID:          p.Name,
			DisplayName: p.DisplayName,
			OrgID:       p.OrgName,
		})
	}
}

// SyncContextsForSession replaces all DiscoveredContext entries for the given
// session with fresh ones derived from the discovered orgs and projects.
// Contexts belonging to other sessions are preserved.
func SyncContextsForSession(
	cfg *datumconfig.ConfigV1Beta1,
	sessionName string,
	orgs []DiscoveredOrg,
	projects []DiscoveredProject,
) {
	remaining := make([]datumconfig.DiscoveredContext, 0, len(cfg.Contexts))
	for _, ctx := range cfg.Contexts {
		if ctx.Session != sessionName {
			remaining = append(remaining, ctx)
		}
	}
	cfg.Contexts = remaining

	for _, o := range orgs {
		cfg.UpsertContext(datumconfig.DiscoveredContext{
			Name:           o.Name,
			Session:        sessionName,
			OrganizationID: o.Name,
		})
	}

	for _, p := range projects {
		cfg.UpsertContext(datumconfig.DiscoveredContext{
			Name:           fmt.Sprintf("%s/%s", p.OrgName, p.Name),
			Session:        sessionName,
			OrganizationID: p.OrgName,
			ProjectID:      p.Name,
			Namespace:      datumconfig.DefaultNamespace,
		})
	}
}

// GCCache removes cached orgs and projects that are no longer referenced by
// any DiscoveredContext in the config. This is safe to call at any time and
// correctly preserves entries referenced by other sessions' contexts.
func GCCache(cfg *datumconfig.ConfigV1Beta1) {
	referencedOrgs := make(map[string]bool)
	referencedProjects := make(map[string]bool)
	for _, ctx := range cfg.Contexts {
		if ctx.OrganizationID != "" {
			referencedOrgs[ctx.OrganizationID] = true
		}
		if ctx.ProjectID != "" {
			referencedProjects[ctx.ProjectID] = true
		}
	}

	keptOrgs := make([]datumconfig.CachedOrg, 0, len(cfg.Cache.Organizations))
	for _, o := range cfg.Cache.Organizations {
		if referencedOrgs[o.ID] {
			keptOrgs = append(keptOrgs, o)
		}
	}
	cfg.Cache.Organizations = keptOrgs

	keptProjects := make([]datumconfig.CachedProject, 0, len(cfg.Cache.Projects))
	for _, p := range cfg.Cache.Projects {
		if referencedProjects[p.ID] {
			keptProjects = append(keptProjects, p)
		}
	}
	cfg.Cache.Projects = keptProjects
}

// RefreshSession re-runs API discovery for the given session and updates the
// config cache. Does not require re-authentication — uses the existing session
// credentials. Returns the number of contexts discovered.
func RefreshSession(ctx context.Context, cfg *datumconfig.ConfigV1Beta1, session *datumconfig.Session) (int, error) {
	tknSrc, err := authutil.GetTokenSourceForUser(ctx, session.UserKey)
	if err != nil {
		return 0, fmt.Errorf("get token source: %w", err)
	}

	userID, err := authutil.GetUserIDFromTokenForUser(session.UserKey)
	if err != nil {
		return 0, fmt.Errorf("get user ID: %w", err)
	}

	apiHostname := datumconfig.StripScheme(session.Endpoint.Server)

	orgs, projects, err := FetchOrgsAndProjects(ctx, apiHostname, tknSrc, userID)
	if err != nil {
		return 0, fmt.Errorf("discover contexts: %w", err)
	}

	UpdateConfigCache(cfg, session.Name, orgs, projects)

	return len(cfg.ContextsForSession(session.Name)), nil
}

// IsCacheStale returns true if the cache has not been refreshed within the
// given duration, or if it has never been refreshed.
func IsCacheStale(cfg *datumconfig.ConfigV1Beta1, maxAge time.Duration) bool {
	if cfg.Cache.LastRefreshed == nil {
		return true
	}
	return time.Since(*cfg.Cache.LastRefreshed) > maxAge
}

func upsertCachedOrg(cache *datumconfig.ContextCache, org datumconfig.CachedOrg) {
	for i := range cache.Organizations {
		if cache.Organizations[i].ID == org.ID {
			cache.Organizations[i] = org
			return
		}
	}
	cache.Organizations = append(cache.Organizations, org)
}

func upsertCachedProject(cache *datumconfig.ContextCache, proj datumconfig.CachedProject) {
	for i := range cache.Projects {
		if cache.Projects[i].ID == proj.ID {
			cache.Projects[i] = proj
			return
		}
	}
	cache.Projects = append(cache.Projects, proj)
}
