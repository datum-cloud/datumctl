package data

import (
	"strings"
	"testing"
	"time"

	activityv1alpha1 "go.miloapis.com/activity/pkg/apis/activity/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// newFakeActivity builds a minimal activityv1alpha1.Activity for testing activityToRow.
func newFakeActivity(email, summary string) activityv1alpha1.Activity {
	return activityv1alpha1.Activity{
		ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.Now()},
		Spec: activityv1alpha1.ActivitySpec{
			Summary:      summary,
			ChangeSource: "human",
			Actor:        activityv1alpha1.ActivityActor{Email: email, Name: email},
			Origin:       activityv1alpha1.ActivityOrigin{Type: "audit", ID: "test-id"},
			Resource:     activityv1alpha1.ActivityResource{Kind: "Project", Name: "my-project"},
			Tenant:       activityv1alpha1.ActivityTenant{Type: "project", Name: "test"},
		},
	}
}

// newFakeActivityWithResource is like newFakeActivity but with explicit resource fields.
func newFakeActivityWithResource(email, summary, apiGroup, kind, name, namespace string) activityv1alpha1.Activity {
	a := newFakeActivity(email, summary)
	a.Spec.Resource = activityv1alpha1.ActivityResource{
		APIGroup:  apiGroup,
		Kind:      kind,
		Name:      name,
		Namespace: namespace,
	}
	return a
}

// --- buildActivityFilter ---

func TestBuildActivityFilter_FullFields(t *testing.T) {
	t.Parallel()
	got := buildActivityFilter("resourcemanager.miloapis.com", "Project", "my-project", "default")
	want := `spec.resource.apiGroup == "resourcemanager.miloapis.com" && spec.resource.kind == "Project" && spec.resource.name == "my-project" && spec.resource.namespace == "default"`
	if got != want {
		t.Errorf("buildActivityFilter full fields:\ngot:  %s\nwant: %s", got, want)
	}
}

func TestBuildActivityFilter_EmptyGroup(t *testing.T) {
	t.Parallel()
	got := buildActivityFilter("", "Pod", "nginx", "kube-system")
	if !strings.Contains(got, `spec.resource.apiGroup == ""`) {
		t.Errorf("buildActivityFilter empty group: want apiGroup == %q in %q", "", got)
	}
}

func TestBuildActivityFilter_CELInjectionPrevented(t *testing.T) {
	t.Parallel()
	// A resource name containing CEL special characters must be safely quoted.
	maliciousName := `foo" || "1" == "1`
	got := buildActivityFilter("example.com", "Widget", maliciousName, "ns")
	// The raw injection string should appear only inside Go-quoted form, not raw.
	if strings.Contains(got, `|| "1" == "1"`) {
		t.Errorf("buildActivityFilter: CEL injection not escaped; got: %s", got)
	}
	// The %q-quoted form must be present.
	if !strings.Contains(got, `"foo\" || \"1\" == \"1"`) {
		t.Errorf("buildActivityFilter: expected escaped name not found; got: %s", got)
	}
}

func TestBuildActivityFilter_AllFieldsQuoted(t *testing.T) {
	t.Parallel()
	// Verify all four fields are present in the filter.
	got := buildActivityFilter("g", "K", "n", "ns")
	for _, field := range []string{
		`spec.resource.apiGroup`,
		`spec.resource.kind`,
		`spec.resource.name`,
		`spec.resource.namespace`,
	} {
		if !strings.Contains(got, field) {
			t.Errorf("buildActivityFilter: missing field %q in %q", field, got)
		}
	}
}

// --- sanitizeSummary ---

func TestSanitizeSummary_PlainString(t *testing.T) {
	t.Parallel()
	got := sanitizeSummary("hello world")
	if got != "hello world" {
		t.Errorf("sanitizeSummary plain = %q, want %q", got, "hello world")
	}
}

func TestSanitizeSummary_StripsSingleANSI(t *testing.T) {
	t.Parallel()
	got := sanitizeSummary("\x1b[31mred text\x1b[0m")
	if got != "red text" {
		t.Errorf("sanitizeSummary ANSI = %q, want %q", got, "red text")
	}
}

func TestSanitizeSummary_StripsMultipleANSI(t *testing.T) {
	t.Parallel()
	got := sanitizeSummary("\x1b[1m\x1b[32mbold green\x1b[0m normal")
	if got != "bold green normal" {
		t.Errorf("sanitizeSummary multi-ANSI = %q, want %q", got, "bold green normal")
	}
}

func TestSanitizeSummary_CollapseNewline(t *testing.T) {
	t.Parallel()
	got := sanitizeSummary("first line\nsecond line")
	if got != "first line…" {
		t.Errorf("sanitizeSummary newline = %q, want %q", got, "first line…")
	}
}

func TestSanitizeSummary_CollapseCarriageReturn(t *testing.T) {
	t.Parallel()
	got := sanitizeSummary("first\rsecond")
	if got != "first…" {
		t.Errorf("sanitizeSummary CR = %q, want %q", got, "first…")
	}
}

func TestSanitizeSummary_NewlineAtStart(t *testing.T) {
	t.Parallel()
	got := sanitizeSummary("\nonly newline")
	if got != "…" {
		t.Errorf("sanitizeSummary leading newline = %q, want %q", got, "…")
	}
}

func TestSanitizeSummary_ANSIThenNewline(t *testing.T) {
	t.Parallel()
	// ANSI is stripped first, then newline collapse.
	got := sanitizeSummary("\x1b[32mgreen\x1b[0m\nmore text")
	if got != "green…" {
		t.Errorf("sanitizeSummary ANSI+newline = %q, want %q", got, "green…")
	}
}

func TestSanitizeSummary_Empty(t *testing.T) {
	t.Parallel()
	got := sanitizeSummary("")
	if got != "" {
		t.Errorf("sanitizeSummary empty = %q, want %q", got, "")
	}
}

// ==================== FB-016: AC#16/17/18 — CEL filter format ====================

// TestBuildProjectActivityFilter_ContainsHumanFilter verifies AC#16: the CEL filter
// for ListRecentProjectActivity contains the 'human' changeSource clause.
func TestBuildProjectActivityFilter_ContainsHumanFilter(t *testing.T) {
	t.Parallel()
	windowStart := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	got := buildProjectActivityFilter(windowStart)
	if !strings.Contains(got, "spec.changeSource == 'human'") {
		t.Errorf("AC#16: filter missing changeSource clause, got: %s", got)
	}
}

// TestBuildProjectActivityFilter_ContainsTimestampClause verifies AC#17: the CEL filter
// contains a spec.timestamp clause bounding the time window.
func TestBuildProjectActivityFilter_ContainsTimestampClause(t *testing.T) {
	t.Parallel()
	windowStart := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	got := buildProjectActivityFilter(windowStart)
	if !strings.Contains(got, "spec.timestamp > timestamp(") {
		t.Errorf("AC#17: filter missing timestamp clause, got: %s", got)
	}
}

// TestBuildProjectActivityFilter_ExactCELString verifies AC#18: the exact CEL string
// format — starts with the changeSource clause, includes the RFC3339 timestamp.
func TestBuildProjectActivityFilter_ExactCELString(t *testing.T) {
	t.Parallel()
	windowStart := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	got := buildProjectActivityFilter(windowStart)

	wantPrefix := "spec.changeSource == 'human' && spec.timestamp > timestamp("
	if !strings.HasPrefix(got, wantPrefix) {
		t.Errorf("AC#18: filter prefix mismatch:\ngot:  %s\nwant: starts with %s", got, wantPrefix)
	}
	// RFC3339 timestamp of the window start must appear in the filter.
	wantTS := windowStart.UTC().Format("2006-01-02T15:04:05Z07:00")
	if !strings.Contains(got, wantTS) {
		t.Errorf("AC#18: filter missing RFC3339 timestamp %q, got: %s", wantTS, got)
	}
}

// ==================== FB-016: AC#20 — ResourceRef additivity ====================

// TestActivityToRow_ResourceRefNil verifies AC#20 (first half): activityToRow
// (FB-006 path used by ListActivity) leaves ResourceRef nil — no breaking change.
func TestActivityToRow_ResourceRefNil(t *testing.T) {
	t.Parallel()
	a := newFakeActivity("alice@example.com", "updated spec")
	row := activityToRow(a)
	if row.ResourceRef != nil {
		t.Errorf("AC#20: activityToRow ResourceRef = non-nil, want nil (FB-006 path must not set ResourceRef)")
	}
}

// TestActivityToProjectRow_ResourceRefPopulated verifies AC#20 (second half):
// activityToProjectRow (FB-016 path) sets ResourceRef from spec.resource.
func TestActivityToProjectRow_ResourceRefPopulated(t *testing.T) {
	t.Parallel()
	a := newFakeActivityWithResource("alice@example.com", "created resource", "apps", "Deployment", "checkout-api", "default")
	row := activityToProjectRow(a)
	if row.ResourceRef == nil {
		t.Fatal("AC#20: activityToProjectRow ResourceRef = nil, want non-nil")
	}
	if row.ResourceRef.Kind != "Deployment" {
		t.Errorf("AC#20: ResourceRef.Kind = %q, want %q", row.ResourceRef.Kind, "Deployment")
	}
	if row.ResourceRef.Name != "checkout-api" {
		t.Errorf("AC#20: ResourceRef.Name = %q, want %q", row.ResourceRef.Name, "checkout-api")
	}
	if row.ResourceRef.APIGroup != "apps" {
		t.Errorf("AC#20: ResourceRef.APIGroup = %q, want %q", row.ResourceRef.APIGroup, "apps")
	}
}

// TestActivityToProjectRow_ResourceRefNil_WhenNoNameOrKind verifies AC#20 edge case:
// when the activity's resource subfield has no Name or Kind, ResourceRef stays nil.
func TestActivityToProjectRow_ResourceRefNil_WhenNoNameOrKind(t *testing.T) {
	t.Parallel()
	a := newFakeActivityWithResource("actor", "action", "", "", "", "")
	row := activityToProjectRow(a)
	if row.ResourceRef != nil {
		t.Errorf("AC#20: activityToProjectRow with empty name+kind: ResourceRef = non-nil, want nil")
	}
}
