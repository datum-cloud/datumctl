package data

import (
	"encoding/json"
	"strings"
	"testing"

	authnv1 "k8s.io/api/authentication/v1"
)

// --- classifySource ---

func TestClassifySource_SystemPrefix_ReturnsSystem(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		username string
		groups   []string
		want     string
	}{
		{"system: prefix", "system:controller:ns", nil, "system"},
		{"system:serviceaccount prefix", "system:serviceaccount:ns:sa", nil, "system"},
		{"serviceaccounts group", "some-bot", []string{"system:serviceaccounts"}, "system"},
		{"serviceaccounts:ns group", "ci-bot", []string{"system:serviceaccounts:ci"}, "system"},
		{"human user", "alice@example.com", nil, "human"},
		{"human with unrelated group", "bob@example.com", []string{"developers"}, "human"},
		{"empty username", "", nil, "human"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			u := authnv1.UserInfo{Username: tt.username, Groups: tt.groups}
			got := classifySource(u)
			if got != tt.want {
				t.Errorf("classifySource(%q, %v) = %q, want %q", tt.username, tt.groups, got, tt.want)
			}
		})
	}
}

// --- compressUsername ---

func TestCompressUsername(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		want string
	}{
		{"system:serviceaccount:default:my-sa", "system:sa:my-sa"},
		{"system:serviceaccount:kube-system:coredns", "system:sa:coredns"},
		{"alice@example.com", "alice@example.com"},
		{"system:controller:ns", "system:controller:ns"},
		{"", "(anonymous)"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			got := compressUsername(tt.in)
			if got != tt.want {
				t.Errorf("compressUsername(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// --- cleanObjectForDiff ---

func TestCleanObjectForDiff_StripsNoisyFields(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"apiVersion": "v1",
		"kind": "Pod",
		"metadata": {
			"name": "my-pod",
			"namespace": "default",
			"labels": {"app": "api"},
			"managedFields": [{"manager": "kubectl"}],
			"resourceVersion": "12345",
			"generation": 3,
			"uid": "abc-123"
		},
		"spec": {"nodeName": "node-1"},
		"status": {"phase": "Running"}
	}`)

	got, err := cleanObjectForDiff(raw)
	if err != nil {
		t.Fatalf("cleanObjectForDiff: unexpected error: %v", err)
	}

	// status must be stripped.
	if _, ok := got["status"]; ok {
		t.Error("cleanObjectForDiff: 'status' key present, want stripped")
	}

	meta, ok := got["metadata"].(map[string]any)
	if !ok {
		t.Fatal("cleanObjectForDiff: metadata missing or wrong type")
	}
	for _, noisy := range []string{"managedFields", "resourceVersion", "generation", "uid"} {
		if _, ok := meta[noisy]; ok {
			t.Errorf("cleanObjectForDiff: metadata.%s present, want stripped", noisy)
		}
	}

	// Preserved fields must remain.
	if meta["name"] != "my-pod" {
		t.Errorf("cleanObjectForDiff: metadata.name = %v, want 'my-pod'", meta["name"])
	}
	if got["apiVersion"] != "v1" {
		t.Errorf("cleanObjectForDiff: apiVersion = %v, want 'v1'", got["apiVersion"])
	}
	if got["kind"] != "Pod" {
		t.Errorf("cleanObjectForDiff: kind = %v, want 'Pod'", got["kind"])
	}
	if got["spec"] == nil {
		t.Error("cleanObjectForDiff: spec should be preserved")
	}
}

func TestCleanObjectForDiff_InvalidJSON_ReturnsError(t *testing.T) {
	t.Parallel()
	_, err := cleanObjectForDiff([]byte(`{not valid json`))
	if err == nil {
		t.Error("cleanObjectForDiff: expected error for invalid JSON, got nil")
	}
}

// --- summarizeFields ---

func TestSummarizeFields(t *testing.T) {
	t.Parallel()
	prev := map[string]any{"spec": map[string]any{"nodeName": "node-1"}}
	curr := map[string]any{"spec": map[string]any{"nodeName": "node-2"}}

	tests := []struct {
		name      string
		verb      string
		prev      map[string]any
		curr      map[string]any
		parseable bool
		want      string
	}{
		{"create verb", "create", nil, curr, true, "Created"},
		{"delete verb", "delete", prev, nil, true, "Deleted"},
		{"update with changes", "update", prev, curr, true, "spec.nodeName"},
		{"update no changes (metadata-only)", "update", curr, curr, true, "metadata only"},
		{"unparseable", "update", prev, curr, false, "— unparseable —"},
		{"patch with changes", "patch", prev, curr, true, "spec.nodeName"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := summarizeFields(tt.verb, tt.prev, tt.curr, tt.parseable)
			if got != tt.want {
				t.Errorf("summarizeFields(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestSummarizeFields_MoreThanThreeChanges_ShowsPlusMore(t *testing.T) {
	t.Parallel()
	prev := map[string]any{"a": 1, "b": 2, "c": 3, "d": 4}
	curr := map[string]any{"a": 10, "b": 20, "c": 30, "d": 40}
	got := summarizeFields("update", prev, curr, true)
	if !strings.Contains(got, "+1 more") {
		t.Errorf("summarizeFields >3 changes: want '+1 more' in %q", got)
	}
}

// --- buildHistoryFilter ---

func TestBuildHistoryFilter_WithNamespace(t *testing.T) {
	t.Parallel()
	rt := ResourceType{Name: "pods", Kind: "Pod", Group: ""}
	got := buildHistoryFilter(rt, "my-pod", "default")
	if !strings.Contains(got, `objectRef.resource == "pods"`) {
		t.Errorf("filter missing resource clause: %q", got)
	}
	if !strings.Contains(got, `objectRef.name == "my-pod"`) {
		t.Errorf("filter missing name clause: %q", got)
	}
	if !strings.Contains(got, `objectRef.namespace == "default"`) {
		t.Errorf("filter missing namespace clause: %q", got)
	}
}

func TestBuildHistoryFilter_WithoutNamespace(t *testing.T) {
	t.Parallel()
	rt := ResourceType{Name: "nodes", Kind: "Node", Group: ""}
	got := buildHistoryFilter(rt, "node-1", "")
	if strings.Contains(got, "namespace") {
		t.Errorf("filter should not contain namespace clause when namespace is empty: %q", got)
	}
}

func TestBuildHistoryFilter_SpecialCharsInName_Quoted(t *testing.T) {
	t.Parallel()
	rt := ResourceType{Name: "httproutes", Kind: "HTTPRoute", Group: "gateway.networking.k8s.io"}
	got := buildHistoryFilter(rt, `my-route"name`, "")
	// %q encoding should escape the quote.
	if strings.Contains(got, `"my-route"name"`) {
		t.Errorf("filter contains unescaped quote in name: %q", got)
	}
}

// --- ComputeDiff ---

func TestComputeDiff_Rev0_IsCreation(t *testing.T) {
	t.Parallel()
	hc := NewHistoryClient(nil)
	m1 := map[string]any{"apiVersion": "v1", "kind": "Pod", "spec": map[string]any{"nodeName": "node-1"}}
	manifests := []map[string]any{m1}

	body, isCreation, predMissing, err := hc.ComputeDiff(manifests, 0)
	if err != nil {
		t.Fatalf("ComputeDiff(k=0): unexpected error: %v", err)
	}
	if !isCreation {
		t.Error("ComputeDiff(k=0): isCreation = false, want true")
	}
	if predMissing {
		t.Error("ComputeDiff(k=0): predMissing = true, want false")
	}
	if !strings.Contains(body, "node-1") {
		t.Errorf("ComputeDiff(k=0): body should contain manifest content, got %q", body)
	}
}

func TestComputeDiff_Rev1_ProducesDiff(t *testing.T) {
	t.Parallel()
	hc := NewHistoryClient(nil)
	m1 := map[string]any{"spec": map[string]any{"nodeName": "node-1"}}
	m2 := map[string]any{"spec": map[string]any{"nodeName": "node-2"}}
	manifests := []map[string]any{m1, m2}

	body, isCreation, predMissing, err := hc.ComputeDiff(manifests, 1)
	if err != nil {
		t.Fatalf("ComputeDiff(k=1): unexpected error: %v", err)
	}
	if isCreation {
		t.Error("ComputeDiff(k=1): isCreation = true, want false")
	}
	if predMissing {
		t.Error("ComputeDiff(k=1): predMissing = true, want false")
	}
	// Unified diff must contain removal of node-1 and addition of node-2.
	if !strings.Contains(body, "node-1") || !strings.Contains(body, "node-2") {
		t.Errorf("ComputeDiff(k=1): diff body should contain both old and new values, got %q", body)
	}
	if !strings.Contains(body, "-") || !strings.Contains(body, "+") {
		t.Errorf("ComputeDiff(k=1): diff body should contain +/- lines, got %q", body)
	}
}

func TestComputeDiff_PredecessorNil_PredMissingTrue(t *testing.T) {
	t.Parallel()
	hc := NewHistoryClient(nil)
	m2 := map[string]any{"spec": map[string]any{"nodeName": "node-2"}}
	manifests := []map[string]any{nil, m2} // index 0 is nil (window truncated)

	body, isCreation, predMissing, err := hc.ComputeDiff(manifests, 1)
	if err != nil {
		t.Fatalf("ComputeDiff(predNil): unexpected error: %v", err)
	}
	if isCreation {
		t.Error("ComputeDiff(predNil): isCreation = true, want false")
	}
	if !predMissing {
		t.Error("ComputeDiff(predNil): predMissing = false, want true")
	}
	if !strings.Contains(body, "node-2") {
		t.Errorf("ComputeDiff(predNil): body should render curr manifest, got %q", body)
	}
}

func TestComputeDiff_OutOfRange_ReturnsError(t *testing.T) {
	t.Parallel()
	hc := NewHistoryClient(nil)
	_, _, _, err := hc.ComputeDiff([]map[string]any{}, 0)
	if err == nil {
		t.Error("ComputeDiff(empty, k=0): expected error, got nil")
	}
	_, _, _, err = hc.ComputeDiff([]map[string]any{{}}, -1)
	if err == nil {
		t.Error("ComputeDiff(k=-1): expected error, got nil")
	}
}

func TestComputeDiff_CurrNil_ReturnsUnparseableMsg(t *testing.T) {
	t.Parallel()
	hc := NewHistoryClient(nil)
	m1 := map[string]any{"spec": "v1"}
	manifests := []map[string]any{m1, nil} // curr is nil (unparseable)

	body, isCreation, _, err := hc.ComputeDiff(manifests, 1)
	if err != nil {
		t.Fatalf("ComputeDiff(currNil): unexpected error: %v", err)
	}
	if isCreation {
		t.Error("ComputeDiff(currNil): isCreation = true, want false")
	}
	if !strings.Contains(body, "could not be parsed") {
		t.Errorf("ComputeDiff(currNil): want 'could not be parsed' message, got %q", body)
	}
}

// --- ForceRefresh / Invalidate ---

func TestHistoryClient_ForceRefresh_DropsCacheEntry(t *testing.T) {
	t.Parallel()
	hc := NewHistoryClient(nil)
	rt := ResourceType{Name: "pods", Kind: "Pod", Group: ""}
	key := historyCacheKey{kind: "Pod", name: "my-pod"}
	// Inject a cache entry directly.
	hc.mu.Lock()
	hc.cache[key] = historyCacheEntry{rows: []HistoryRow{{Rev: 1}}}
	hc.mu.Unlock()

	hc.ForceRefresh(rt, "my-pod", "")

	hc.mu.Lock()
	_, ok := hc.cache[key]
	hc.mu.Unlock()
	if ok {
		t.Error("ForceRefresh: cache entry still present, want deleted")
	}
}

func TestHistoryClient_Invalidate_DropsAllEntries(t *testing.T) {
	t.Parallel()
	hc := NewHistoryClient(nil)
	hc.mu.Lock()
	hc.cache[historyCacheKey{kind: "Pod", name: "a"}] = historyCacheEntry{}
	hc.cache[historyCacheKey{kind: "Pod", name: "b"}] = historyCacheEntry{}
	hc.mu.Unlock()

	hc.Invalidate()

	hc.mu.Lock()
	n := len(hc.cache)
	hc.mu.Unlock()
	if n != 0 {
		t.Errorf("Invalidate: cache has %d entries, want 0", n)
	}
}

// --- findChangedFields ---

func TestFindChangedFields_NilPrev_ReturnsAllCurrentKeys(t *testing.T) {
	t.Parallel()
	curr := map[string]any{"a": 1, "b": 2}
	got := findChangedFields(nil, curr)
	// Both keys must appear.
	gotSet := make(map[string]bool)
	for _, k := range got {
		gotSet[k] = true
	}
	for _, want := range []string{"a", "b"} {
		if !gotSet[want] {
			t.Errorf("findChangedFields(nil, curr): missing key %q in %v", want, got)
		}
	}
}

func TestFindChangedFields_UnchangedTopLevel_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	m := map[string]any{"spec": map[string]any{"x": 1}}
	got := findChangedFields(m, m)
	if len(got) != 0 {
		t.Errorf("findChangedFields(same, same) = %v, want empty", got)
	}
}

func TestFindChangedFields_OneLevelDeep_ReturnsPath(t *testing.T) {
	t.Parallel()
	prev := map[string]any{"spec": map[string]any{"x": 1}}
	curr := map[string]any{"spec": map[string]any{"x": 2}}
	got := findChangedFields(prev, curr)
	if len(got) != 1 || got[0] != "spec.x" {
		t.Errorf("findChangedFields: got %v, want [spec.x]", got)
	}
}

// --- manifestLines ---

func TestManifestLines_ProducesNewlineTerminatedLines(t *testing.T) {
	t.Parallel()
	m := map[string]any{"key": "value"}
	lines := manifestLines(m)
	if len(lines) == 0 {
		t.Fatal("manifestLines: empty result")
	}
	// JSON marshal of {"key":"value"} produces 3 lines: { "key": "value" }
	for i, line := range lines {
		if !strings.HasSuffix(line, "\n") {
			t.Errorf("manifestLines[%d] = %q, want newline-terminated", i, line)
		}
	}
	combined := strings.Join(lines, "")
	var roundtrip map[string]any
	if err := json.Unmarshal([]byte(combined), &roundtrip); err != nil {
		t.Errorf("manifestLines: round-trip JSON parse failed: %v", err)
	}
}
