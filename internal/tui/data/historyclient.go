package data

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pmezard/go-difflib/difflib"
	authnv1 "k8s.io/api/authentication/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	activityv1alpha1 "go.miloapis.com/activity/pkg/apis/activity/v1alpha1"
	activityclientset "go.miloapis.com/activity/pkg/client/clientset/versioned"
	"go.datum.net/datumctl/internal/client"
)

// HistoryRow is a single row in the revision list.
type HistoryRow struct {
	Rev       int
	Timestamp time.Time
	User      string // raw username
	UserDisp  string // compressed display form (e.g., system:sa:name)
	Source    string // "human" | "system"
	Verb      string // create/update/patch/delete
	Status    int32  // HTTP response code
	Summary   string // synthesized
	Parseable bool   // false when ResponseObject.Raw was nil or unparseable
}

type historyCacheKey struct {
	apiGroup  string
	kind      string
	name      string
	namespace string
}

type historyCacheEntry struct {
	fetchedAt time.Time
	rows      []HistoryRow
	manifests []map[string]any
	truncated bool
}

const historyCacheTTL = 60 * time.Second

// HistoryClient wraps the activity clientset with a per-resource TTL cache for
// AuditLogQuery history fetches.
type HistoryClient struct {
	factory *client.DatumCloudFactory
	mu      sync.Mutex
	cache   map[historyCacheKey]historyCacheEntry
}

func NewHistoryClient(factory *client.DatumCloudFactory) *HistoryClient {
	return &HistoryClient{
		factory: factory,
		cache:   make(map[historyCacheKey]historyCacheEntry),
	}
}

// LoadHistory fetches (or returns cached) revision history for the given resource.
// Returns (rows, manifests, truncated, err). rows and manifests are parallel slices
// indexed 0..N-1 where index 0 is the oldest revision (REV 1).
func (c *HistoryClient) LoadHistory(
	ctx context.Context,
	rt ResourceType,
	name, namespace string,
) ([]HistoryRow, []map[string]any, bool, error) {
	key := historyCacheKey{
		apiGroup:  rt.Group,
		kind:      rt.Kind,
		name:      name,
		namespace: namespace,
	}

	c.mu.Lock()
	if entry, ok := c.cache[key]; ok && time.Since(entry.fetchedAt) < historyCacheTTL {
		rows := entry.rows
		manifests := entry.manifests
		truncated := entry.truncated
		c.mu.Unlock()
		return rows, manifests, truncated, nil
	}
	c.mu.Unlock()

	restConfig, err := c.factory.ConfigFlags.ToRESTConfig()
	if err != nil {
		return nil, nil, false, fmt.Errorf("rest config: %w", err)
	}

	cs, err := activityclientset.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, false, fmt.Errorf("activity clientset: %w", err)
	}

	filter := buildHistoryFilter(rt, name, namespace)
	query := &activityv1alpha1.AuditLogQuery{
		Spec: activityv1alpha1.AuditLogQuerySpec{
			StartTime: "now-30d",
			EndTime:   "now",
			Filter:    filter,
			Limit:     100,
		},
	}

	result, err := cs.ActivityV1alpha1().AuditLogQueries().Create(ctx, query, metav1.CreateOptions{})
	if err != nil {
		return nil, nil, false, err
	}

	truncated := result.Status.Continue != "" && len(result.Status.Results) >= 100

	// Results are newest-first; reverse to oldest-first so REV 1 = oldest.
	events := result.Status.Results
	for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
		events[i], events[j] = events[j], events[i]
	}

	rows := make([]HistoryRow, 0, len(events))
	manifests := make([]map[string]any, 0, len(events))

	for i, ev := range events {
		rev := i + 1

		ts := ev.StageTimestamp.Time
		if ts.IsZero() {
			ts = ev.RequestReceivedTimestamp.Time
		}

		var statusCode int32
		if ev.ResponseStatus != nil {
			statusCode = ev.ResponseStatus.Code
		}

		var manifest map[string]any
		parseable := false
		if ev.ResponseObject != nil && len(ev.ResponseObject.Raw) > 0 {
			m, parseErr := cleanObjectForDiff(ev.ResponseObject.Raw)
			if parseErr == nil {
				manifest = m
				parseable = true
			}
		}
		manifests = append(manifests, manifest)

		source := classifySource(ev.User)
		summary := summarizeFields(ev.Verb, prevManifest(manifests, i), manifest, parseable)

		rows = append(rows, HistoryRow{
			Rev:       rev,
			Timestamp: ts,
			User:      ev.User.Username,
			UserDisp:  compressUsername(ev.User.Username),
			Source:    source,
			Verb:      ev.Verb,
			Status:    statusCode,
			Summary:   summary,
			Parseable: parseable,
		})
	}

	c.mu.Lock()
	c.cache[key] = historyCacheEntry{
		fetchedAt: time.Now(),
		rows:      rows,
		manifests: manifests,
		truncated: truncated,
	}
	c.mu.Unlock()

	return rows, manifests, truncated, nil
}

// ForceRefresh drops the cache entry for the given resource so the next
// LoadHistory call fetches fresh data.
func (c *HistoryClient) ForceRefresh(rt ResourceType, name, namespace string) {
	key := historyCacheKey{apiGroup: rt.Group, kind: rt.Kind, name: name, namespace: namespace}
	c.mu.Lock()
	delete(c.cache, key)
	c.mu.Unlock()
}

// Invalidate drops all cached entries (used on context switch).
func (c *HistoryClient) Invalidate() {
	c.mu.Lock()
	c.cache = make(map[historyCacheKey]historyCacheEntry)
	c.mu.Unlock()
}

// IsUnauthorized returns true when err is a 403/401 response.
func (c *HistoryClient) IsUnauthorized(err error) bool {
	return k8serrors.IsForbidden(err) || k8serrors.IsUnauthorized(err)
}

// ComputeDiff returns the raw (uncolorized) unified-diff body between revision
// k (0-indexed) and its predecessor.
// isCreation is true when k == 0.
// predMissing is true when k > 0 but manifests[k-1] is nil (window truncated).
func (c *HistoryClient) ComputeDiff(
	manifests []map[string]any,
	k int,
) (body string, isCreation bool, predMissing bool, err error) {
	if k < 0 || k >= len(manifests) {
		return "", false, false, fmt.Errorf("revision index %d out of range [0, %d)", k, len(manifests))
	}

	curr := manifests[k]

	if k == 0 {
		if curr == nil {
			return "— manifest could not be parsed —", true, false, nil
		}
		rendered, e := renderManifest(curr)
		if e != nil {
			return "— manifest could not be parsed —", true, false, nil
		}
		return rendered, true, false, nil
	}

	prev := manifests[k-1]

	if curr == nil {
		return "— manifest for this revision could not be parsed —", false, false, nil
	}

	if prev == nil {
		// Predecessor not available (window truncated or parse error).
		rendered, e := renderManifest(curr)
		if e != nil {
			return "— manifest could not be parsed —", false, true, nil
		}
		return rendered, false, true, nil
	}

	prevLines := manifestLines(prev)
	currLines := manifestLines(curr)

	// FromFile = predecessor REV label; ToFile = current REV label.
	// Index k is 0-based: REV = k+1, predecessor REV = k.
	var buf bytes.Buffer
	ud := difflib.UnifiedDiff{
		A:        prevLines,
		B:        currLines,
		FromFile: fmt.Sprintf("rev %d", k),
		ToFile:   fmt.Sprintf("rev %d", k+1),
		Context:  3,
	}
	if e := difflib.WriteUnifiedDiff(&buf, ud); e != nil {
		return "", false, false, e
	}

	return buf.String(), false, false, nil
}

// buildHistoryFilter returns a CEL filter expression for an AuditLogQuery.
func buildHistoryFilter(rt ResourceType, name, namespace string) string {
	parts := []string{
		fmt.Sprintf("objectRef.resource == %q", rt.Name),
		fmt.Sprintf("objectRef.name == %q", name),
		"verb in ['create', 'update', 'patch', 'delete']",
	}
	if namespace != "" {
		parts = append(parts, fmt.Sprintf("objectRef.namespace == %q", namespace))
	}
	return strings.Join(parts, " && ")
}

// classifySource determines whether an audit event was human- or system-initiated.
func classifySource(user authnv1.UserInfo) string {
	if strings.HasPrefix(user.Username, "system:") {
		return "system"
	}
	for _, g := range user.Groups {
		if g == "system:serviceaccounts" || strings.HasPrefix(g, "system:serviceaccounts:") {
			return "system"
		}
	}
	return "human"
}

// compressUsername shortens system:serviceaccount:ns:name → system:sa:name.
func compressUsername(username string) string {
	const prefix = "system:serviceaccount:"
	if after, ok := strings.CutPrefix(username, prefix); ok {
		if _, name, found := strings.Cut(after, ":"); found {
			return "system:sa:" + name
		}
	}
	if username == "" {
		return "(anonymous)"
	}
	return username
}

// cleanObjectForDiff unmarshals raw JSON and strips noisy metadata fields.
func cleanObjectForDiff(raw []byte) (map[string]any, error) {
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}
	delete(obj, "status")
	if meta, ok := obj["metadata"].(map[string]any); ok {
		delete(meta, "managedFields")
		delete(meta, "resourceVersion")
		delete(meta, "generation")
		delete(meta, "uid")
	}
	return obj, nil
}

// prevManifest returns manifests[i-1] if i > 0, otherwise nil.
func prevManifest(manifests []map[string]any, i int) map[string]any {
	if i == 0 {
		return nil
	}
	return manifests[i-1]
}

// summarizeFields produces a one-line summary of what changed between prev and curr.
func summarizeFields(verb string, prev, curr map[string]any, parseable bool) string {
	switch verb {
	case "create":
		return "Created"
	case "delete":
		return "Deleted"
	}

	if !parseable || curr == nil {
		return "— unparseable —"
	}

	changed := findChangedFields(prev, curr)
	if len(changed) == 0 {
		return "metadata only"
	}
	if len(changed) <= 3 {
		return strings.Join(changed, ", ")
	}
	return strings.Join(changed[:3], ", ") + fmt.Sprintf(" +%d more", len(changed)-3)
}

// findChangedFields returns top-level or one-deep field paths that differ.
func findChangedFields(prev, curr map[string]any) []string {
	if prev == nil {
		var keys []string
		for k := range curr {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return keys
	}

	keySet := make(map[string]struct{})
	for k := range prev {
		keySet[k] = struct{}{}
	}
	for k := range curr {
		keySet[k] = struct{}{}
	}

	var changed []string
	for k := range keySet {
		prevV := prev[k]
		currV := curr[k]

		prevJ, _ := json.Marshal(prevV)
		currJ, _ := json.Marshal(currV)
		if string(prevJ) == string(currJ) {
			continue
		}

		prevMap, prevIsMap := prevV.(map[string]any)
		currMap, currIsMap := currV.(map[string]any)
		if prevIsMap && currIsMap {
			subSet := make(map[string]struct{})
			for sk := range prevMap {
				subSet[sk] = struct{}{}
			}
			for sk := range currMap {
				subSet[sk] = struct{}{}
			}
			for sk := range subSet {
				pj, _ := json.Marshal(prevMap[sk])
				cj, _ := json.Marshal(currMap[sk])
				if string(pj) != string(cj) {
					changed = append(changed, k+"."+sk)
				}
			}
		} else {
			changed = append(changed, k)
		}
	}

	sort.Strings(changed)
	return changed
}

// manifestLines serialises a cleaned manifest to JSON lines for difflib.
func manifestLines(m map[string]any) []string {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil
	}
	lines := strings.Split(string(b), "\n")
	for i := range lines {
		lines[i] += "\n"
	}
	return lines
}

// renderManifest serialises a manifest as indented JSON for the creation view.
func renderManifest(m map[string]any) (string, error) {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
