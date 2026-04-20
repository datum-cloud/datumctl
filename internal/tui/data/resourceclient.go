package data

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/openapi"
	"k8s.io/client-go/rest"

	"go.datum.net/datumctl/internal/client"
)

var skipGroups = map[string]bool{
	"events.k8s.io":         true,
	"authentication.k8s.io": true,
	"authorization.k8s.io":  true,
	"coordination.k8s.io":   true,
}

// DescribeResult carries both the formatted describe text and the raw object.
type DescribeResult struct {
	Content string
	Raw     *unstructured.Unstructured
}

type ResourceClient interface {
	ListResourceTypes(ctx context.Context) ([]ResourceType, error)
	ListResources(ctx context.Context, rt ResourceType, ns string) (rows []ResourceRow, columns []string, err error)
	DescribeResource(ctx context.Context, rt ResourceType, name, ns string) (DescribeResult, error)

	// DeleteResource deletes the named resource using the Kubernetes dynamic client.
	// Callers use the classification helpers to route to the right render branch.
	DeleteResource(ctx context.Context, rt ResourceType, name, namespace string) error

	// Error classifiers — thin wrappers so callers avoid importing k8serrors directly.
	IsForbidden(err error) bool
	IsNotFound(err error) bool
	IsConflict(err error) bool
	IsUnauthorized(err error) bool

	// ListEvents fetches Kubernetes events whose involvedObject references the named
	// resource in the given namespace. For cluster-scoped resources, involvedObjectNamespace
	// is empty and the query uses the cluster-wide events endpoint.
	// Returns [] (not nil) on empty. Errors propagate; callers classify via IsForbidden / IsNotFound. // AC#27
	ListEvents(
		ctx                     context.Context,
		involvedObjectKind      string,
		involvedObjectName      string,
		involvedObjectNamespace string,
	) ([]EventRow, error)

	// InvalidateResourceListCache purges any cached list for the given resource kind.
	// No-op on clients that perform live fetches with no cache.
	InvalidateResourceListCache(kind string)
}

// tableResponse holds the subset of the Kubernetes Table API response we care about.
type tableResponse struct {
	ColumnDefinitions []struct {
		Name     string `json:"name"`
		Priority int32  `json:"priority"`
	} `json:"columnDefinitions"`
	Rows []struct {
		Cells  []any `json:"cells"`
		Object struct {
			Metadata struct {
				Name      string `json:"name"`
				Namespace string `json:"namespace"`
			} `json:"metadata"`
		} `json:"object"`
	} `json:"rows"`
}

type KubeResourceClient struct {
	factory *client.DatumCloudFactory
	dc      dynamic.Interface // non-nil overrides factory.DynamicClient() — for testing only
}

func NewKubeResourceClient(factory *client.DatumCloudFactory) *KubeResourceClient {
	return &KubeResourceClient{factory: factory}
}

// dynamicClient returns the dynamic client to use, preferring the injected
// test client over the factory-derived one.
func (k *KubeResourceClient) dynamicClient() (dynamic.Interface, error) {
	if k.dc != nil {
		return k.dc, nil
	}
	return k.factory.DynamicClient()
}

func (k *KubeResourceClient) ListResourceTypes(ctx context.Context) ([]ResourceType, error) {
	dc, err := k.factory.ToDiscoveryClient()
	if err != nil {
		return nil, fmt.Errorf("discovery client: %w", err)
	}

	lists, err := dc.ServerPreferredResources()
	if err != nil {
		// ServerPreferredResources may return partial results alongside an error.
		// We continue with whatever was returned.
		if lists == nil {
			return nil, fmt.Errorf("server preferred resources: %w", err)
		}
	}

	var types []ResourceType
	for _, list := range lists {
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			continue
		}
		if skipGroups[gv.Group] {
			continue
		}
		for _, r := range list.APIResources {
			if !hasVerb(r.Verbs, "list") {
				continue
			}
			types = append(types, ResourceType{
				Name:       r.Name,
				Kind:       r.Kind,
				Group:      gv.Group,
				Version:    gv.Version,
				Namespaced: r.Namespaced,
			})
		}
	}
	attachDescriptions(ctx, dc.OpenAPIV3(), types)
	return types, nil
}

func hasVerb(verbs []string, verb string) bool {
	return slices.Contains(verbs, verb)
}

// attachDescriptions fetches OpenAPI v3 schemas in parallel and populates
// Description on each ResourceType. Errors are silently ignored so that a
// missing or slow OpenAPI endpoint never blocks the TUI from starting.
func attachDescriptions(_ context.Context, oc openapi.Client, types []ResourceType) {
	// Build path → indices map so we fetch each group/version only once.
	pathIdx := map[string][]int{}
	for i, rt := range types {
		var p string
		if rt.Group == "" {
			p = "api/" + rt.Version
		} else {
			p = "apis/" + rt.Group + "/" + rt.Version
		}
		pathIdx[p] = append(pathIdx[p], i)
	}

	paths, err := oc.Paths()
	if err != nil {
		return
	}

	var mu sync.Mutex
	descsByPath := map[string]map[string]string{}
	var wg sync.WaitGroup

	for path, gv := range paths {
		if _, needed := pathIdx[path]; !needed {
			continue
		}
		wg.Add(1)
		go func(p string, gv openapi.GroupVersion) {
			defer wg.Done()
			data, err := gv.Schema("application/json")
			if err != nil {
				return
			}
			descs := parseOpenAPIDescriptions(data)
			mu.Lock()
			descsByPath[p] = descs
			mu.Unlock()
		}(path, gv)
	}
	wg.Wait()

	for path, indices := range pathIdx {
		descs := descsByPath[path]
		for _, i := range indices {
			if d, ok := descs[types[i].Kind]; ok {
				types[i].Description = d
			}
		}
	}
}

// parseOpenAPIDescriptions extracts kind → description from an OpenAPI v3
// schema JSON blob. Schema keys use the format "…​.Kind" (last dot-segment).
func parseOpenAPIDescriptions(data []byte) map[string]string {
	var schema struct {
		Components struct {
			Schemas map[string]struct {
				Description string `json:"description"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil
	}
	out := make(map[string]string, len(schema.Components.Schemas))
	for key, s := range schema.Components.Schemas {
		if s.Description == "" {
			continue
		}
		parts := strings.Split(key, ".")
		out[parts[len(parts)-1]] = s.Description
	}
	return out
}

// listAsTable fetches the resource list using the Kubernetes Table API
// (Accept: application/json;as=Table;v=v1;g=meta.k8s.io). This is the same
// mechanism kubectl uses so the API server returns properly formatted printer
// columns for every resource type automatically.
func (k *KubeResourceClient) listAsTable(ctx context.Context, rt ResourceType, ns string) (*tableResponse, error) {
	cfg, err := k.factory.ConfigFlags.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("rest config: %w", err)
	}

	transport, err := rest.TransportFor(cfg)
	if err != nil {
		return nil, fmt.Errorf("transport: %w", err)
	}

	// Build the resource URL path.
	var path string
	if rt.Group == "" {
		// Core API group: /api/{version}/...
		if ns != "" {
			path = fmt.Sprintf("/api/%s/namespaces/%s/%s", rt.Version, ns, rt.Name)
		} else {
			path = fmt.Sprintf("/api/%s/%s", rt.Version, rt.Name)
		}
	} else {
		// Named API group: /apis/{group}/{version}/...
		if ns != "" {
			path = fmt.Sprintf("/apis/%s/%s/namespaces/%s/%s", rt.Group, rt.Version, ns, rt.Name)
		} else {
			path = fmt.Sprintf("/apis/%s/%s/%s", rt.Group, rt.Version, rt.Name)
		}
	}

	url := strings.TrimRight(cfg.Host, "/") + path + "?includeObject=Metadata"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json;as=Table;v=v1;g=meta.k8s.io,application/json")

	httpClient := &http.Client{Transport: transport}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("table request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("table request returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var tbl tableResponse
	if err := json.Unmarshal(body, &tbl); err != nil {
		return nil, fmt.Errorf("decode table response: %w", err)
	}

	return &tbl, nil
}

func (k *KubeResourceClient) ListResources(ctx context.Context, rt ResourceType, ns string) ([]ResourceRow, []string, error) {
	tbl, err := k.listAsTable(ctx, rt, ns)
	if err != nil {
		return nil, nil, fmt.Errorf("list %s: %w", rt.Name, err)
	}

	// Collect priority-0 column indices and names.
	type colInfo struct {
		index int
		name  string
	}
	var cols []colInfo
	for i, cd := range tbl.ColumnDefinitions {
		if cd.Priority == 0 {
			cols = append(cols, colInfo{index: i, name: cd.Name})
		}
	}

	colNames := make([]string, len(cols))
	for i, c := range cols {
		colNames[i] = c.name
	}

	rows := make([]ResourceRow, 0, len(tbl.Rows))
	for _, r := range tbl.Rows {
		name := r.Object.Metadata.Name
		if name == "" && len(r.Cells) > 0 {
			// Fallback: first cell often contains the name.
			name = cellString(r.Cells[0])
		}

		cells := make([]string, len(cols))
		for i, c := range cols {
			if c.index < len(r.Cells) {
				cells[i] = cellString(r.Cells[c.index])
			}
		}

		rows = append(rows, ResourceRow{
			Name:      name,
			Namespace: r.Object.Metadata.Namespace,
			Cells:     cells,
		})
	}

	return rows, colNames, nil
}

// cellString converts a table cell value to a display string. nil becomes "".
func cellString(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func (k *KubeResourceClient) DescribeResource(ctx context.Context, rt ResourceType, name, ns string) (DescribeResult, error) {
	dc, err := k.factory.DynamicClient()
	if err != nil {
		return DescribeResult{}, fmt.Errorf("dynamic client: %w", err)
	}

	gvr := schema.GroupVersionResource{
		Group:    rt.Group,
		Version:  rt.Version,
		Resource: rt.Name,
	}

	obj, err := dc.Resource(gvr).Namespace(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return DescribeResult{}, fmt.Errorf("get %s/%s: %w", rt.Name, name, err)
	}

	raw := obj.Object

	var sb strings.Builder
	metaMap, _ := raw["metadata"].(map[string]any)

	// ── Metadata ────────────────────────────────────────────────────────────
	writeSectionHeader(&sb, "metadata")
	fmt.Fprintf(&sb, "  Name:       %s\n", stringField(metaMap, "name"))
	if ns := stringField(metaMap, "namespace"); ns != "" {
		fmt.Fprintf(&sb, "  Namespace:  %s\n", ns)
	}
	if ts := stringField(metaMap, "creationTimestamp"); ts != "" {
		if t, err2 := time.Parse(time.RFC3339, ts); err2 == nil {
			fmt.Fprintf(&sb, "  Created:    %s  (%s ago)\n",
				t.UTC().Format("2006-01-02 15:04:05 UTC"), formatAge(time.Since(t)))
		}
	}
	if labels, ok := metaMap["labels"].(map[string]any); ok && len(labels) > 0 {
		sb.WriteString("  Labels:\n")
		for _, k := range sortedKeys(labels) {
			fmt.Fprintf(&sb, "    %s: %v\n", k, labels[k])
		}
	}
	if annotations, ok := metaMap["annotations"].(map[string]any); ok && len(annotations) > 0 {
		sb.WriteString("  Annotations:\n")
		for _, k := range sortedKeys(annotations) {
			if k == "kubectl.kubernetes.io/last-applied-configuration" {
				continue
			}
			v := fmt.Sprintf("%v", annotations[k])
			if len(v) > 120 {
				v = v[:117] + "..."
			}
			fmt.Fprintf(&sb, "    %s: %s\n", k, v)
		}
	}
	sb.WriteString("\n")

	// ── Spec ────────────────────────────────────────────────────────────────
	if spec, ok := raw["spec"]; ok {
		writeSectionHeader(&sb, "spec")
		renderValue(&sb, spec, "  ")
		sb.WriteString("\n")
	}

	// ── Status ──────────────────────────────────────────────────────────────
	if statusRaw, ok := raw["status"]; ok {
		writeSectionHeader(&sb, "status")
		statusMap, _ := statusRaw.(map[string]any)

		if conditions, ok := statusMap["conditions"].([]any); ok && len(conditions) > 0 {
			sb.WriteString("  Conditions:\n")
			fmt.Fprintf(&sb, "    %-24s %-8s %-28s %s\n", "Type", "Status", "Reason", "Age")
			fmt.Fprintf(&sb, "    %s\n", strings.Repeat("─", 70))
			for _, c := range conditions {
				cond, ok := c.(map[string]any)
				if !ok {
					continue
				}
				condType, _ := cond["type"].(string)
				condStatus, _ := cond["status"].(string)
				reason, _ := cond["reason"].(string)
				age := ""
				if lt, _ := cond["lastTransitionTime"].(string); lt != "" {
					if t, err2 := time.Parse(time.RFC3339, lt); err2 == nil {
						age = formatAge(time.Since(t))
					}
				}
				fmt.Fprintf(&sb, "    %-24s %-8s %-28s %s\n", condType, condStatus, reason, age)
				if msg, _ := cond["message"].(string); msg != "" {
					fmt.Fprintf(&sb, "      Message: %s\n", msg)
				}
			}
			sb.WriteString("\n")
		}

		for _, k := range sortedKeys(statusMap) {
			if k == "conditions" {
				continue
			}
			v := statusMap[k]
			switch v.(type) {
			case map[string]any, []any:
				fmt.Fprintf(&sb, "  %s:\n", k)
				renderValue(&sb, v, "    ")
			default:
				fmt.Fprintf(&sb, "  %s: %v\n", k, v)
			}
		}
	}

	return DescribeResult{Content: sb.String(), Raw: obj}, nil
}

// ── Quota / AllowanceBucket support ─────────────────────────────────────────

type bucketCacheEntry struct {
	buckets   []AllowanceBucket
	fetchedAt time.Time
}

var (
	bucketCacheMu sync.Mutex
	bucketCache   = map[string]bucketCacheEntry{}
)

func (k *KubeResourceClient) ListAllowanceBuckets(ctx context.Context) ([]AllowanceBucket, error) {
	cfg, err := k.factory.ConfigFlags.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("rest config: %w", err)
	}
	key := cfg.Host

	bucketCacheMu.Lock()
	entry, ok := bucketCache[key]
	bucketCacheMu.Unlock()
	if ok && time.Since(entry.fetchedAt) < 30*time.Second {
		return entry.buckets, nil
	}

	gvr, err := k.findAllowanceBucketGVR()
	if err != nil {
		// CRD not installed — return empty, not an error.
		return nil, nil
	}

	dc, err := k.factory.DynamicClient()
	if err != nil {
		return nil, fmt.Errorf("dynamic client: %w", err)
	}

	list, err := dc.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list allowancebuckets: %w", err)
	}

	buckets := make([]AllowanceBucket, 0, len(list.Items))
	for _, item := range list.Items {
		buckets = append(buckets, parseAllowanceBucket(item))
	}

	bucketCacheMu.Lock()
	bucketCache[key] = bucketCacheEntry{buckets: buckets, fetchedAt: time.Now()}
	bucketCacheMu.Unlock()

	return buckets, nil
}

// DeleteResource deletes the named resource with explicit background propagation.
// The caller is responsible for checking the error type via the classifier methods.
func (k *KubeResourceClient) DeleteResource(ctx context.Context, rt ResourceType, name, namespace string) error {
	dc, err := k.dynamicClient()
	if err != nil {
		return fmt.Errorf("dynamic client: %w", err)
	}
	gvr := schema.GroupVersionResource{
		Group:    rt.Group,
		Version:  rt.Version,
		Resource: rt.Name,
	}
	policy := metav1.DeletePropagationBackground
	opts := metav1.DeleteOptions{PropagationPolicy: &policy}
	if namespace != "" {
		return dc.Resource(gvr).Namespace(namespace).Delete(ctx, name, opts)
	}
	return dc.Resource(gvr).Delete(ctx, name, opts)
}

// ListEvents fetches core/v1 Events with involvedObject field selectors for the given resource. // AC#27
func (k *KubeResourceClient) ListEvents(ctx context.Context, involvedObjectKind, involvedObjectName, involvedObjectNamespace string) ([]EventRow, error) {
	dc, err := k.dynamicClient()
	if err != nil {
		return nil, fmt.Errorf("dynamic client: %w", err)
	}

	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "events"}

	fieldSelector := "involvedObject.name=" + involvedObjectName
	if involvedObjectNamespace != "" {
		fieldSelector += ",involvedObject.namespace=" + involvedObjectNamespace
	}

	var list *unstructured.UnstructuredList
	if involvedObjectNamespace != "" {
		list, err = dc.Resource(gvr).Namespace(involvedObjectNamespace).List(ctx, metav1.ListOptions{FieldSelector: fieldSelector})
	} else {
		list, err = dc.Resource(gvr).List(ctx, metav1.ListOptions{FieldSelector: fieldSelector})
	}
	if err != nil {
		return nil, err
	}

	rows := make([]EventRow, 0, len(list.Items))
	for _, item := range list.Items {
		rows = append(rows, parseEventRowFromUnstructured(item))
	}
	return rows, nil
}

func parseEventRowFromUnstructured(item unstructured.Unstructured) EventRow {
	raw := item.Object
	r := EventRow{}

	r.Type, _ = raw["type"].(string)
	r.Reason, _ = raw["reason"].(string)
	r.Message, _ = raw["message"].(string)

	if count, ok := raw["count"]; ok {
		switch v := count.(type) {
		case int64:
			r.Count = int32(v)
		case float64:
			r.Count = int32(v)
		case int32:
			r.Count = v
		}
	}

	if ts, ok := raw["lastTimestamp"].(string); ok && ts != "" {
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			r.LastTimestamp = t
		}
	}

	if series, ok := raw["series"].(map[string]any); ok {
		if ts, ok2 := series["lastObservedTime"].(string); ok2 && ts != "" {
			if t, err := time.Parse(time.RFC3339, ts); err == nil {
				if r.LastTimestamp.IsZero() {
					r.LastTimestamp = t
				}
			}
		}
	}

	meta, _ := raw["metadata"].(map[string]any)
	if ts := stringField(meta, "creationTimestamp"); ts != "" && r.LastTimestamp.IsZero() {
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			r.EventTime = t
		}
	}

	return r
}

func (k *KubeResourceClient) IsForbidden(err error) bool    { return k8serrors.IsForbidden(err) }
func (k *KubeResourceClient) IsNotFound(err error) bool     { return k8serrors.IsNotFound(err) }
func (k *KubeResourceClient) IsConflict(err error) bool     { return k8serrors.IsConflict(err) }
func (k *KubeResourceClient) IsUnauthorized(err error) bool { return k8serrors.IsForbidden(err) || k8serrors.IsUnauthorized(err) }

// InvalidateResourceListCache is a no-op on KubeResourceClient because list calls
// are always live HTTP requests with no in-memory cache.
func (k *KubeResourceClient) InvalidateResourceListCache(_ string) {}

func (k *KubeResourceClient) InvalidateBucketCache() {
	cfg, err := k.factory.ConfigFlags.ToRESTConfig()
	if err != nil {
		return
	}
	key := cfg.Host
	bucketCacheMu.Lock()
	delete(bucketCache, key)
	bucketCacheMu.Unlock()
}

func (k *KubeResourceClient) findAllowanceBucketGVR() (schema.GroupVersionResource, error) {
	dc, err := k.factory.ToDiscoveryClient()
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("discovery client: %w", err)
	}
	lists, err := dc.ServerPreferredResources()
	if err != nil && lists == nil {
		return schema.GroupVersionResource{}, fmt.Errorf("server preferred resources: %w", err)
	}
	for _, list := range lists {
		gv, parseErr := schema.ParseGroupVersion(list.GroupVersion)
		if parseErr != nil {
			continue
		}
		if gv.Group != "quota.miloapis.com" {
			continue
		}
		for _, r := range list.APIResources {
			if r.Name == "allowancebuckets" {
				return schema.GroupVersionResource{
					Group:    gv.Group,
					Version:  gv.Version,
					Resource: "allowancebuckets",
				}, nil
			}
		}
	}
	return schema.GroupVersionResource{}, fmt.Errorf("allowancebuckets not found in discovery")
}

func parseAllowanceBucket(item unstructured.Unstructured) AllowanceBucket {
	raw := item.Object
	meta, _ := raw["metadata"].(map[string]any)
	spec, _ := raw["spec"].(map[string]any)
	status, _ := raw["status"].(map[string]any)

	var labels map[string]any
	if meta != nil {
		labels, _ = meta["labels"].(map[string]any)
	}

	b := AllowanceBucket{
		Name:         stringField(meta, "name"),
		ConsumerKind: labelVal(labels, "quota.miloapis.com/consumer-kind"),
		ConsumerName: labelVal(labels, "quota.miloapis.com/consumer-name"),
		ResourceType: stringField(spec, "resourceType"),
		Limit:        int64AnyField(status, "limit"),
		Allocated:    int64AnyField(status, "allocated"),
		Available:    int64AnyField(status, "available"),
		ClaimCount:   intAnyField(status, "claimCount"),
	}

	if ts := stringField(status, "lastReconciliationTime"); ts != "" {
		if t, parseErr := time.Parse(time.RFC3339, ts); parseErr == nil {
			b.LastReconciliation = t
		}
	}

	if refs, ok := status["contributingGrantRefs"].([]any); ok {
		for _, r := range refs {
			if s, ok2 := r.(string); ok2 {
				b.ContributingGrantRefs = append(b.ContributingGrantRefs, s)
			}
		}
	}

	return b
}

func labelVal(labels map[string]any, key string) string {
	if labels == nil {
		return ""
	}
	v, _ := labels[key].(string)
	return v
}

func int64AnyField(m map[string]any, keys ...string) int64 {
	if m == nil {
		return 0
	}
	for _, key := range keys {
		if v, ok := m[key]; ok {
			switch n := v.(type) {
			case int64:
				return n
			case float64:
				return int64(n)
			case int:
				return int64(n)
			case string:
				// k8s resource.Quantity strings (e.g. "50", "1k") — try plain
				// integer parse first, then fall back to Quantity parsing.
				if i, err := strconv.ParseInt(n, 10, 64); err == nil {
					return i
				}
				if q, err := resource.ParseQuantity(n); err == nil {
					return q.Value()
				}
			}
		}
	}
	return 0
}

func intAnyField(m map[string]any, keys ...string) int {
	if m == nil {
		return 0
	}
	for _, key := range keys {
		if v, ok := m[key]; ok {
			switch n := v.(type) {
			case int:
				return n
			case float64:
				return int(n)
			case int64:
				return int(n)
			case string:
				if i, err := strconv.ParseInt(n, 10, 64); err == nil {
					return int(i)
				}
			}
		}
	}
	return 0
}

func writeSectionHeader(sb *strings.Builder, name string) {
	pad := 40 - len(name) - 4
	if pad < 2 {
		pad = 2
	}
	fmt.Fprintf(sb, "── %s %s\n", name, strings.Repeat("─", pad))
}

func renderValue(sb *strings.Builder, v any, indent string) {
	switch val := v.(type) {
	case map[string]any:
		for _, k := range sortedKeys(val) {
			child := val[k]
			switch child.(type) {
			case map[string]any, []any:
				fmt.Fprintf(sb, "%s%s:\n", indent, k)
				renderValue(sb, child, indent+"  ")
			default:
				fmt.Fprintf(sb, "%s%s: %v\n", indent, k, child)
			}
		}
	case []any:
		for _, item := range val {
			switch item.(type) {
			case map[string]any:
				fmt.Fprintf(sb, "%s-\n", indent)
				renderValue(sb, item, indent+"  ")
			default:
				fmt.Fprintf(sb, "%s- %v\n", indent, item)
			}
		}
	default:
		fmt.Fprintf(sb, "%s%v\n", indent, val)
	}
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func stringField(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, _ := m[key].(string)
	return v
}

func formatAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
