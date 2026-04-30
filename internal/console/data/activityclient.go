package data

import (
	"context"
	stderrors "errors"
	"fmt"
	"strings"
	"sync"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	activityv1alpha1 "go.miloapis.com/activity/pkg/apis/activity/v1alpha1"
	activityclientset "go.miloapis.com/activity/pkg/client/clientset/versioned"
	"go.datum.net/datumctl/internal/client"
)

// ErrActivityCRDAbsent is returned when the ActivityQuery CRD is not installed on the cluster.
var ErrActivityCRDAbsent = stderrors.New("activity CRD not installed on cluster")

// ErrActivityCRDPartial is returned when the CRD is present but does not support filter-less project-scope queries.
var ErrActivityCRDPartial = stderrors.New("activity CRD present but does not support filter-less project-scope queries")

// ResourceRef identifies a Kubernetes resource referenced in an activity row.
type ResourceRef struct {
	APIGroup  string
	Kind      string
	Name      string
	Namespace string
}

// ActivityRow is a single display row from the activity timeline.
type ActivityRow struct {
	Timestamp    time.Time
	Origin       string // "audit" | "event"
	ActorDisplay string // email or name; "" for event rows
	ChangeSource string // "human" | "system"
	Summary      string
	ResourceRef  *ResourceRef // non-nil only for ListRecentProjectActivity rows
}

// activityCacheKey identifies a resource for activity caching.
type activityCacheKey struct {
	apiGroup  string
	kind      string
	name      string
	namespace string
}

type activityCacheEntry struct {
	rows      []ActivityRow
	cont      string
	fetchedAt time.Time
}

// projectActivityCacheKey identifies a project-scope activity cache entry.
type projectActivityCacheKey struct {
	window time.Duration
	limit  int
}

type projectActivityCacheEntry struct {
	rows      []ActivityRow
	fetchedAt time.Time
}

// ActivityClient wraps the activity clientset with a per-resource TTL cache.
type ActivityClient struct {
	factory      *client.DatumCloudFactory
	mu           sync.Mutex
	cache        map[activityCacheKey]activityCacheEntry
	projectCache map[projectActivityCacheKey]projectActivityCacheEntry
}

// NewActivityClient constructs an ActivityClient using the factory for REST config.
func NewActivityClient(factory *client.DatumCloudFactory) *ActivityClient {
	return &ActivityClient{
		factory:      factory,
		cache:        make(map[activityCacheKey]activityCacheEntry),
		projectCache: make(map[projectActivityCacheKey]projectActivityCacheEntry),
	}
}

const activityCacheTTL = 60 * time.Second

// ListActivity fetches activity for the given resource, using the cache when
// possible.  continueToken empty = first page; non-empty = next page (cache
// bypass, results appended to caller's buffer).
func (c *ActivityClient) ListActivity(
	ctx context.Context,
	apiGroup, kind, name, namespace, continueToken string,
) ([]ActivityRow, string, error) {
	key := activityCacheKey{apiGroup: apiGroup, kind: kind, name: name, namespace: namespace}

	// Only cache the first page.
	if continueToken == "" {
		c.mu.Lock()
		if entry, ok := c.cache[key]; ok && time.Since(entry.fetchedAt) < activityCacheTTL {
			rows := entry.rows
			cont := entry.cont
			c.mu.Unlock()
			return rows, cont, nil
		}
		c.mu.Unlock()
	}

	restConfig, err := c.factory.ConfigFlags.ToRESTConfig()
	if err != nil {
		return nil, "", fmt.Errorf("rest config: %w", err)
	}

	cs, err := activityclientset.NewForConfig(restConfig)
	if err != nil {
		return nil, "", fmt.Errorf("activity clientset: %w", err)
	}

	filter := buildActivityFilter(apiGroup, kind, name, namespace)
	query := &activityv1alpha1.ActivityQuery{
		Spec: activityv1alpha1.ActivityQuerySpec{
			StartTime: "now-30d",
			EndTime:   "now",
			Filter:    filter,
			Limit:     50,
			Continue:  continueToken,
		},
	}

	result, err := cs.ActivityV1alpha1().ActivityQueries().Create(ctx, query, metav1.CreateOptions{})
	if err != nil {
		return nil, "", err
	}

	rows := make([]ActivityRow, 0, len(result.Status.Results))
	for _, a := range result.Status.Results {
		rows = append(rows, activityToRow(a))
	}

	nextCont := result.Status.Continue

	// Only cache first-page results.
	if continueToken == "" {
		c.mu.Lock()
		c.cache[key] = activityCacheEntry{
			rows:      rows,
			cont:      nextCont,
			fetchedAt: time.Now(),
		}
		c.mu.Unlock()
	}

	return rows, nextCont, nil
}

// ForceRefresh invalidates the cache entry for a specific resource so the next
// ListActivity call fetches fresh data.
func (c *ActivityClient) ForceRefresh(apiGroup, kind, name, namespace string) {
	key := activityCacheKey{apiGroup: apiGroup, kind: kind, name: name, namespace: namespace}
	c.mu.Lock()
	delete(c.cache, key)
	c.mu.Unlock()
}

// Invalidate drops all cached entries (used on context switch).
func (c *ActivityClient) Invalidate() {
	c.mu.Lock()
	c.cache = make(map[activityCacheKey]activityCacheEntry)
	c.projectCache = make(map[projectActivityCacheKey]projectActivityCacheEntry)
	c.mu.Unlock()
}

// ForceRefreshProject drops the project-scope cache so the next call to
// ListRecentProjectActivity fetches fresh data.
func (c *ActivityClient) ForceRefreshProject(window time.Duration, limit int) {
	key := projectActivityCacheKey{window: window, limit: limit}
	c.mu.Lock()
	delete(c.projectCache, key)
	c.mu.Unlock()
}

// ListRecentProjectActivity fetches recent human-authored activity for the active project.
// The CEL filter pins changeSource to 'human' and limits to the given time window.
func (c *ActivityClient) ListRecentProjectActivity(
	ctx context.Context,
	window time.Duration,
	limit int,
) ([]ActivityRow, error) {
	key := projectActivityCacheKey{window: window, limit: limit}

	c.mu.Lock()
	if entry, ok := c.projectCache[key]; ok && time.Since(entry.fetchedAt) < activityCacheTTL {
		rows := entry.rows
		c.mu.Unlock()
		return rows, nil
	}
	c.mu.Unlock()

	restConfig, err := c.factory.ConfigFlags.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("rest config: %w", err)
	}

	cs, err := activityclientset.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("activity clientset: %w", err)
	}

	windowEnd := time.Now()
	windowStart := windowEnd.Add(-window)
	filter := buildProjectActivityFilter(windowStart)

	query := &activityv1alpha1.ActivityQuery{
		Spec: activityv1alpha1.ActivityQuerySpec{
			StartTime: windowStart.UTC().Format(time.RFC3339),
			EndTime:   windowEnd.UTC().Format(time.RFC3339),
			Filter:    filter,
			Limit:     int32(limit),
			Continue:  "",
		},
	}

	result, err := cs.ActivityV1alpha1().ActivityQueries().Create(ctx, query, metav1.CreateOptions{})
	if err != nil {
		if meta.IsNoMatchError(err) {
			return nil, ErrActivityCRDAbsent
		}
		if k8serrors.IsBadRequest(err) {
			msg := err.Error()
			if strings.Contains(msg, "filter: resource selector required") || strings.Contains(msg, "filter-less") {
				return nil, ErrActivityCRDPartial
			}
		}
		return nil, err
	}

	rows := make([]ActivityRow, 0, len(result.Status.Results))
	for _, a := range result.Status.Results {
		rows = append(rows, activityToProjectRow(a))
	}

	c.mu.Lock()
	c.projectCache[key] = projectActivityCacheEntry{
		rows:      rows,
		fetchedAt: time.Now(),
	}
	c.mu.Unlock()

	return rows, nil
}

// IsUnauthorized reports whether err is a 403 Forbidden response.
func (c *ActivityClient) IsUnauthorized(err error) bool {
	return k8serrors.IsForbidden(err) || k8serrors.IsUnauthorized(err)
}

// buildProjectActivityFilter returns the CEL filter used by ListRecentProjectActivity.
// The filter pins changeSource to 'human' and restricts to events after windowStart.
func buildProjectActivityFilter(windowStart time.Time) string {
	return fmt.Sprintf("spec.changeSource == 'human' && spec.timestamp > timestamp(%q)", windowStart.UTC().Format(time.RFC3339))
}

// buildActivityFilter returns a CEL filter expression for an ActivityQuery.
// Each component is quoted with %q to prevent CEL injection from resource names.
func buildActivityFilter(apiGroup, kind, name, namespace string) string {
	return fmt.Sprintf(
		"spec.resource.apiGroup == %q && spec.resource.kind == %q && spec.resource.name == %q && spec.resource.namespace == %q",
		apiGroup, kind, name, namespace,
	)
}

func activityToRow(a activityv1alpha1.Activity) ActivityRow {
	actor := a.Spec.Actor.Email
	if actor == "" {
		actor = a.Spec.Actor.Name
	}
	// Event rows have no meaningful actor.
	if a.Spec.Origin.Type == "event" {
		actor = ""
	}
	return ActivityRow{
		Timestamp:    a.CreationTimestamp.Time,
		Origin:       a.Spec.Origin.Type,
		ActorDisplay: actor,
		ChangeSource: a.Spec.ChangeSource,
		Summary:      sanitizeSummary(a.Spec.Summary),
	}
}

// activityToProjectRow is like activityToRow but also populates ResourceRef
// from the activity's spec.resource subfield.
func activityToProjectRow(a activityv1alpha1.Activity) ActivityRow {
	row := activityToRow(a)
	ref := &ResourceRef{
		APIGroup:  a.Spec.Resource.APIGroup,
		Kind:      a.Spec.Resource.Kind,
		Name:      a.Spec.Resource.Name,
		Namespace: a.Spec.Resource.Namespace,
	}
	// Only set ResourceRef when at least Name or Kind is populated.
	if ref.Name != "" || ref.Kind != "" {
		row.ResourceRef = ref
	}
	return row
}

// sanitizeSummary strips ANSI escapes and collapses newlines to a single line.
func sanitizeSummary(s string) string {
	// Strip ANSI escape sequences.
	out := make([]rune, 0, len(s))
	inEsc := false
	for _, r := range s {
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		if r == '\x1b' {
			inEsc = true
			continue
		}
		out = append(out, r)
	}
	result := string(out)
	// Collapse newlines: keep only up to the first newline boundary.
	for i, c := range result {
		if c == '\n' || c == '\r' {
			if i > 0 {
				return result[:i] + "…"
			}
			return "…"
		}
	}
	return result
}
