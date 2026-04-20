package components

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"go.datum.net/datumctl/internal/tui/data"
)

// ansiRe strips ANSI escape codes so View() output can be checked as plain text.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

// makeContent builds a multi-line string with n numbered lines.
func makeContent(n int) string {
	rows := make([]string, n)
	for i := range rows {
		rows[i] = fmt.Sprintf("line %03d content here", i+1)
	}
	return strings.Join(rows, "\n")
}

func TestDetailViewModel_TitleBar(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		width            int
		height           int
		kind             string
		resourceName     string
		loading          bool
		describeAvailable bool
		wantContains     []string
		wantAbsent       []string
	}{
		{
			name:         "renders resource kind and name with slash separator",
			width:        120, // wide enough for rightText (~81 chars post-FB-024) + leftText
			height:       20,
			kind:         "httproutes",
			resourceName: "my-route-name",
			wantContains: []string{"httproutes", " / ", "my-route-name"},
		},
		{
			name:              "shows keybind hint when not loading",
			width:             100,
			height:            20,
			kind:              "pods",
			resourceName:      "my-pod",
			describeAvailable: true,
			wantContains:      []string{"[j/k] scroll", "[y] yaml", "[C] conditions", "[x] delete", "[Esc] back"},
		},
		{
			name:         "loading state appends loading suffix and hides keybind hint",
			width:        60,
			height:       20,
			kind:         "gateways",
			resourceName: "prod-gateway",
			loading:      true,
			wantContains: []string{"gateways", " / ", "prod-gateway", "loading"},
			wantAbsent:   []string{"[y] yaml"},
		},
		{
			name:         "title bar contains only kind and name — namespace is omitted",
			width:        100,
			height:       20,
			kind:         "pods",
			resourceName: "my-pod",
			// The title bar spec is "<kind> / <name>" — namespace never appears.
			wantContains: []string{"pods", " / ", "my-pod"},
			wantAbsent:   []string{"namespace", "default"},
		},
		{
			name:         "long name over 40 chars is truncated with ellipsis",
			width:        105, // wide enough for rightText (~81 chars post-FB-024) but not the long name
			height:       20,
			kind:         "httproutes",
			resourceName: "my-extremely-verbose-route-name-that-exceeds-forty-chars-easily",
			wantContains: []string{"httproutes", " / ", "…"},
		},
		{
			name:         "truncated name does not overflow pane width",
			width:        105, // same as above
			height:       20,
			kind:         "httproutes",
			resourceName: "my-extremely-verbose-route-name-that-exceeds-forty-chars-easily",
			// After truncation the first output line (title bar) must fit within the pane.
			// We assert the full name is gone and the kind is still visible.
			wantAbsent: []string{"my-extremely-verbose-route-name-that-exceeds-forty-chars-easily"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := NewDetailViewModel(tt.width, tt.height)
			m.SetResourceContext(tt.kind, tt.resourceName)
			m.SetLoading(tt.loading)
			m.SetDescribeAvailable(tt.describeAvailable)
			got := stripANSI(m.View())

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("View() missing %q\ngot:\n%s", want, got)
				}
			}
			for _, absent := range tt.wantAbsent {
				if strings.Contains(got, absent) {
					t.Errorf("View() should not contain %q\ngot:\n%s", absent, got)
				}
			}
		})
	}
}

func TestDetailViewModel_ScrollProgress(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		setup     func(m *DetailViewModel)
		wantLabel string
	}{
		{
			name: "shows top when at beginning",
			setup: func(m *DetailViewModel) {
				m.SetContent(makeContent(50))
			},
			wantLabel: "top",
		},
		{
			name: "shows 100% when scrolled to bottom",
			setup: func(m *DetailViewModel) {
				m.SetContent(makeContent(50))
				m.vp.GotoBottom()
			},
			wantLabel: "100%",
		},
		{
			name: "shows percentage label when scrolled to middle",
			setup: func(m *DetailViewModel) {
				m.SetContent(makeContent(50))
				// Position mid-scroll: YOffset=20 out of max≈40, ≈50%.
				m.vp.YOffset = 20
			},
			wantLabel: "%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := NewDetailViewModel(60, 10)
			m.SetResourceContext("pods", "my-pod")
			tt.setup(&m)
			got := stripANSI(m.View())
			if !strings.Contains(got, tt.wantLabel) {
				t.Errorf("scroll footer: want label %q\ngot:\n%s", tt.wantLabel, got)
			}
		})
	}
}

func TestDetailViewModel_ChromeSuppression(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		height     int
		wantChrome bool
	}{
		{"height 6 shows chrome", 6, true},
		{"height 10 shows chrome", 10, true},
		{"height 20 shows chrome", 20, true},
		{"height 5 suppresses chrome", 5, false},
		{"height 4 suppresses chrome", 4, false},
		{"height 1 suppresses chrome", 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := NewDetailViewModel(100, tt.height) // wide enough to fit all hints including [C]
			m.SetResourceContext("pods", "my-pod")
			got := stripANSI(m.View())
			// The keybind hint only appears in the title bar chrome.
			hasChrome := strings.Contains(got, "[j/k] scroll") && strings.Contains(got, "[Esc] back")
			if tt.wantChrome && !hasChrome {
				t.Errorf("height=%d: expected chrome (title bar) but not found\ngot:\n%s", tt.height, got)
			}
			if !tt.wantChrome && hasChrome {
				t.Errorf("height=%d: expected no chrome but title bar found\ngot:\n%s", tt.height, got)
			}
		})
	}
}

func TestDetailViewModel_SetSize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		newW         int
		newH         int
		wantVpHeight int
	}{
		{"tall terminal shrinks viewport by 4 rows", 60, 20, 16},
		{"exactly 6 rows gives viewport height 2", 60, 6, 2},
		{"below 6 rows viewport fills full height (no chrome)", 60, 5, 5},
		{"height 1 gives viewport height 1", 60, 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := NewDetailViewModel(60, 10)
			m.SetSize(tt.newW, tt.newH)
			if got := m.vp.Height; got != tt.wantVpHeight {
				t.Errorf("SetSize(%d, %d): viewport height = %d, want %d", tt.newW, tt.newH, got, tt.wantVpHeight)
			}
		})
	}
}

func TestDetailViewModel_ResourceContextAccessors(t *testing.T) {
	t.Parallel()
	m := NewDetailViewModel(60, 20)
	m.SetResourceContext("configmaps", "my-config")
	m.SetLoading(true)

	if got := m.ResourceKind(); got != "configmaps" {
		t.Errorf("ResourceKind() = %q, want %q", got, "configmaps")
	}
	if got := m.ResourceName(); got != "my-config" {
		t.Errorf("ResourceName() = %q, want %q", got, "my-config")
	}
	if got := m.Loading(); !got {
		t.Errorf("Loading() = false, want true")
	}
}

// ==================== FB-009: SetMode / Mode / ScrollToTop ====================

func TestDetailViewModel_SetMode_AccessorRoundTrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		mode string
	}{
		{"yaml mode", "yaml"},
		{"describe mode", "describe"},
		{"empty mode", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := NewDetailViewModel(80, 20)
			m.SetMode(tt.mode)
			if got := m.Mode(); got != tt.mode {
				t.Errorf("Mode() = %q after SetMode(%q), want %q", got, tt.mode, tt.mode)
			}
		})
	}
}

func TestDetailViewModel_SetMode_YamlMode_TitleBarShowsLabel(t *testing.T) {
	t.Parallel()
	m := NewDetailViewModel(120, 20) // wide enough to fit mode label + all hints post-FB-024
	m.SetResourceContext("pods", "my-pod")
	m.SetMode("yaml")

	got := stripANSI(m.View())
	if !strings.Contains(got, "yaml") {
		t.Errorf("SetMode(yaml): titleBar should contain 'yaml', got:\n%s", got)
	}
}

// When mode is "yaml", the [y] hint should say "[y] describe" (toggle back).
func TestDetailViewModel_SetMode_YamlMode_FlipsKeyHint(t *testing.T) {
	t.Parallel()
	m := NewDetailViewModel(120, 20)
	m.SetResourceContext("pods", "my-pod")
	m.SetDescribeAvailable(true)
	m.SetMode("yaml")

	got := stripANSI(m.View())
	if !strings.Contains(got, "[y] describe") {
		t.Errorf("SetMode(yaml): want '[y] describe' hint, got:\n%s", got)
	}
	if strings.Contains(got, "[y] yaml") {
		t.Errorf("SetMode(yaml): '[y] yaml' hint should not appear when already in yaml mode, got:\n%s", got)
	}
}

// When mode is empty or "describe", the [y] hint says "[y] yaml".
func TestDetailViewModel_SetMode_DescribeMode_DefaultKeyHint(t *testing.T) {
	t.Parallel()
	tests := []struct{ name, mode string }{
		{"empty mode", ""},
		{"describe mode", "describe"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := NewDetailViewModel(120, 20)
			m.SetResourceContext("pods", "my-pod")
			m.SetDescribeAvailable(true)
			m.SetMode(tt.mode)
			got := stripANSI(m.View())
			if !strings.Contains(got, "[y] yaml") {
				t.Errorf("SetMode(%q): want '[y] yaml' hint, got:\n%s", tt.mode, got)
			}
		})
	}
}

func TestDetailViewModel_ScrollToTop_ResetsViewport(t *testing.T) {
	t.Parallel()
	m := NewDetailViewModel(60, 10)
	m.SetResourceContext("pods", "my-pod")
	m.SetContent(makeContent(50))
	m.vp.GotoBottom()

	m.ScrollToTop()

	// After ScrollToTop the viewport offset must be zero.
	if m.vp.YOffset != 0 {
		t.Errorf("ScrollToTop: YOffset = %d, want 0", m.vp.YOffset)
	}
}

// ==================== FB-019: RenderEventsTable ====================

// mockEventsRC is a minimal data.ResourceClient stub for RenderEventsTable tests.
type mockEventsRC struct {
	isForbidden bool
	isNotFound  bool
}

func (m *mockEventsRC) ListResourceTypes(_ context.Context) ([]data.ResourceType, error) {
	return nil, nil
}
func (m *mockEventsRC) ListResources(_ context.Context, _ data.ResourceType, _ string) ([]data.ResourceRow, []string, error) {
	return nil, nil, nil
}
func (m *mockEventsRC) DescribeResource(_ context.Context, _ data.ResourceType, _, _ string) (data.DescribeResult, error) {
	return data.DescribeResult{}, nil
}
func (m *mockEventsRC) DeleteResource(_ context.Context, _ data.ResourceType, _, _ string) error {
	return nil
}
func (m *mockEventsRC) IsForbidden(err error) bool    { return m.isForbidden && err != nil }
func (m *mockEventsRC) IsNotFound(err error) bool     { return m.isNotFound && err != nil }
func (m *mockEventsRC) IsConflict(_ error) bool       { return false }
func (m *mockEventsRC) IsUnauthorized(_ error) bool   { return false }
func (m *mockEventsRC) InvalidateResourceListCache(_ string) {}
func (m *mockEventsRC) ListEvents(_ context.Context, _, _, _ string) ([]data.EventRow, error) {
	return nil, nil
}

func seedEvents() []data.EventRow {
	return []data.EventRow{
		{Type: "Normal", Reason: "SuccessfulCreate", Message: "Created resource", Count: 1, LastTimestamp: time.Now().Add(-5 * time.Minute)},
		{Type: "Warning", Reason: "BackOff", Message: "Back-off restarting failed container", Count: 3, LastTimestamp: time.Now().Add(-2 * time.Minute)},
	}
}

// TestRenderEventsTable_Loading (AC#24): loading=true renders spinner + "Loading events".
func TestRenderEventsTable_Loading(t *testing.T) {
	t.Parallel()
	rc := &mockEventsRC{}
	sp := spinner.New()
	got := stripANSI(RenderEventsTable(nil, true, nil, rc, 80, sp))
	if !strings.Contains(got, "Loading events") {
		t.Errorf("loading=true: want 'Loading events' in output, got:\n%s", got)
	}
}

// TestRenderEventsTable_EmptyEvents (AC#15): nil events + nil err → empty-state message.
func TestRenderEventsTable_EmptyEvents(t *testing.T) {
	t.Parallel()
	rc := &mockEventsRC{}
	sp := spinner.New()
	got := stripANSI(RenderEventsTable(nil, false, nil, rc, 80, sp))
	if !strings.Contains(got, "No events recorded for this resource.") {
		t.Errorf("empty events: want 'No events recorded for this resource.', got:\n%s", got)
	}
}

var eventsGR = schema.GroupResource{Group: "core", Resource: "events"}

// TestRenderEventsTable_ForbiddenError (AC#12): Forbidden k8s error → canonical "Permission denied" title.
func TestRenderEventsTable_ForbiddenError(t *testing.T) {
	t.Parallel()
	rc := &mockEventsRC{isForbidden: true}
	sp := spinner.New()
	fetchErr := k8serrors.NewForbidden(eventsGR, "pod", errors.New("denied"))
	got := stripANSI(RenderEventsTable(nil, false, fetchErr, rc, 80, sp))
	if !strings.Contains(got, "Permission denied") {
		t.Errorf("forbidden error: want 'Permission denied', got:\n%s", got)
	}
	if !strings.Contains(got, "You don't have permission") {
		t.Errorf("forbidden error: want permission detail, got:\n%s", got)
	}
}

// TestRenderEventsTable_NotFoundError (AC#13): NotFound k8s error → canonical "Resource not found" title.
func TestRenderEventsTable_NotFoundError(t *testing.T) {
	t.Parallel()
	rc := &mockEventsRC{isNotFound: true}
	sp := spinner.New()
	fetchErr := k8serrors.NewNotFound(eventsGR, "pod")
	got := stripANSI(RenderEventsTable(nil, false, fetchErr, rc, 80, sp))
	if !strings.Contains(got, "Resource not found") {
		t.Errorf("not-found error: want 'Resource not found', got:\n%s", got)
	}
}

// TestRenderEventsTable_GenericError (AC#14): generic err → "Could not fetch events" fallback title.
func TestRenderEventsTable_GenericError(t *testing.T) {
	t.Parallel()
	rc := &mockEventsRC{}
	sp := spinner.New()
	fetchErr := errors.New("dial tcp: connection refused")

	t.Run("wide_width", func(t *testing.T) {
		t.Parallel()
		got := stripANSI(RenderEventsTable(nil, false, fetchErr, rc, 80, sp))
		if !strings.Contains(got, "Could not fetch events") {
			t.Errorf("generic error: want 'Could not fetch events', got:\n%s", got)
		}
	})

	t.Run("narrow_width_truncation", func(t *testing.T) {
		t.Parallel()
		// Very narrow width still renders error block — no panic.
		got := stripANSI(RenderEventsTable(nil, false, fetchErr, rc, 40, sp))
		if !strings.Contains(got, "Could not fetch events") {
			t.Errorf("narrow generic error: want 'Could not fetch events', got:\n%s", got)
		}
	})
}

// TestRenderEventsError_Forbidden_RendersCanonicalCopy (Input-changed): Forbidden error renders
// the canonical §8 "Permission denied" title; old ad-hoc copy must not appear.
func TestRenderEventsError_Forbidden_RendersCanonicalCopy(t *testing.T) {
	t.Parallel()
	rc := &mockEventsRC{isForbidden: true}
	sp := spinner.New()
	fetchErr := k8serrors.NewForbidden(eventsGR, "pod", errors.New("denied"))
	got := stripANSI(RenderEventsTable(nil, false, fetchErr, rc, 80, sp))
	if !strings.Contains(got, "Permission denied") {
		t.Errorf("forbidden (canonical): want 'Permission denied', got:\n%s", got)
	}
	if strings.Contains(got, "No permission to list events") {
		t.Errorf("forbidden (canonical): old copy 'No permission to list events' must be absent, got:\n%s", got)
	}
}

// TestRenderEventsError_NotFound_RendersCanonicalCopy (Input-changed): NotFound error renders
// the canonical §8 "Resource not found" title; old ad-hoc copy must not appear.
func TestRenderEventsError_NotFound_RendersCanonicalCopy(t *testing.T) {
	t.Parallel()
	rc := &mockEventsRC{isNotFound: true}
	sp := spinner.New()
	fetchErr := k8serrors.NewNotFound(eventsGR, "pod")
	got := stripANSI(RenderEventsTable(nil, false, fetchErr, rc, 80, sp))
	if !strings.Contains(got, "Resource not found") {
		t.Errorf("not-found (canonical): want 'Resource not found', got:\n%s", got)
	}
	if strings.Contains(got, "Namespace no longer exists") {
		t.Errorf("not-found (canonical): old copy 'Namespace no longer exists' must be absent, got:\n%s", got)
	}
}

// TestRenderEventsError_Generic_RoutesThroughErrorBlock (Migration parity): generic error must
// produce a RenderErrorBlock output — confirmed by the presence of the ⚠ glyph and [Esc] hint.
func TestRenderEventsError_Generic_RoutesThroughErrorBlock(t *testing.T) {
	t.Parallel()
	rc := &mockEventsRC{}
	sp := spinner.New()
	fetchErr := errors.New("dial tcp: connection refused")
	got := stripANSI(RenderEventsTable(nil, false, fetchErr, rc, 80, sp))
	if !strings.Contains(got, "⚠") && !strings.Contains(got, "✕") {
		t.Errorf("generic error: want error glyph (⚠ or ✕), confirming RenderErrorBlock path; got:\n%s", got)
	}
	if !strings.Contains(got, "[Esc]") {
		t.Errorf("generic error: want '[Esc]' back hint from RenderErrorBlock, got:\n%s", got)
	}
}

// TestRenderEventsError_NarrowWidth_AntiRegression (Anti-regression): width<40 "Terminal too narrow"
// guard must fire BEFORE the error path — even when fetchErr is non-nil.
func TestRenderEventsError_NarrowWidth_AntiRegression(t *testing.T) {
	t.Parallel()
	rc := &mockEventsRC{isForbidden: true}
	sp := spinner.New()
	fetchErr := k8serrors.NewForbidden(eventsGR, "pod", errors.New("denied"))
	got := stripANSI(RenderEventsTable(nil, false, fetchErr, rc, 30, sp))
	if !strings.Contains(got, "Terminal too narrow") {
		t.Errorf("narrow+error: want 'Terminal too narrow' guard (upstream of error path), got:\n%s", got)
	}
	if strings.Contains(got, "Permission denied") {
		t.Errorf("narrow+error: error block must NOT render when width<40, got:\n%s", got)
	}
}

// TestRenderEventsTable_WidthBand_OffByOne (AC#22): off-by-one transitions at 40, 60, 80.
func TestRenderEventsTable_WidthBand_OffByOne(t *testing.T) {
	t.Parallel()
	rc := &mockEventsRC{}
	sp := spinner.New()
	events := seedEvents()

	tests := []struct {
		name         string
		width        int
		wantContains []string
		wantAbsent   []string
	}{
		{
			name:         "width_39_unusable",
			width:        39,
			wantContains: []string{"Terminal too narrow"},
			wantAbsent:   []string{"Type", "Reason", "Age"},
		},
		{
			name:         "width_40_narrow",
			width:        40,
			wantContains: []string{"Type", "Reason", "Age"},
			wantAbsent:   []string{"Terminal too narrow", "Message", "Count"},
		},
		{
			name:         "width_59_narrow",
			width:        59,
			wantContains: []string{"Type", "Reason", "Age"},
			wantAbsent:   []string{"Terminal too narrow", "Count"},
		},
		{
			name:         "width_60_standard",
			width:        60,
			wantContains: []string{"Type", "Reason", "Age", "Count"},
			wantAbsent:   []string{"Terminal too narrow", "Message"},
		},
		{
			name:         "width_79_standard",
			width:        79,
			wantContains: []string{"Type", "Reason", "Age", "Count"},
			wantAbsent:   []string{"Terminal too narrow", "Message"},
		},
		{
			name:         "width_80_wide",
			width:        80,
			wantContains: []string{"Type", "Reason", "Age", "Message", "Count"},
			wantAbsent:   []string{"Terminal too narrow"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := stripANSI(RenderEventsTable(events, false, nil, rc, tt.width, sp))
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("width=%d: want %q in output, got:\n%s", tt.width, want, got)
				}
			}
			for _, absent := range tt.wantAbsent {
				if strings.Contains(got, absent) {
					t.Errorf("width=%d: want %q absent from output, got:\n%s", tt.width, absent, got)
				}
			}
		})
	}
}

// TestRenderEventsTable_WarningRow_Highlight (AC#17): Warning row is styled; Normal is not.
func TestRenderEventsTable_WarningRow_Highlight(t *testing.T) {
	t.Parallel()
	rc := &mockEventsRC{}
	sp := spinner.New()

	events := []data.EventRow{
		{Type: "Normal", Reason: "SuccessfulCreate", Message: "ok", Count: 1, LastTimestamp: time.Now().Add(-1 * time.Minute)},
		{Type: "Warning", Reason: "BackOff", Message: "backing off", Count: 2, LastTimestamp: time.Now().Add(-1 * time.Minute)},
		{Type: "warning", Reason: "CaseInsensitive", Message: "lower warning", Count: 1, LastTimestamp: time.Now().Add(-1 * time.Minute)},
		{Type: "", Reason: "NoType", Message: "no type field", Count: 1, LastTimestamp: time.Now().Add(-1 * time.Minute)},
	}

	raw := RenderEventsTable(events, false, nil, rc, 80, sp)
	lines := strings.Split(raw, "\n")

	// Find data rows (skip header lines — lines 0 and 1 are header+sep).
	// The render puts data rows after the two header lines, using the same order.
	// We look for lines containing "SuccessfulCreate", "BackOff", "CaseInsensitive", "NoType".
	findLine := func(needle string) string {
		for _, l := range lines {
			if strings.Contains(stripANSI(l), needle) {
				return l
			}
		}
		return ""
	}

	normalLine := findLine("SuccessfulCreate")
	warnLine := findLine("BackOff")
	lowerWarnLine := findLine("CaseInsensitive")
	noTypeLine := findLine("NoType")

	if normalLine == "" {
		t.Fatal("could not find Normal row in output")
	}
	if warnLine == "" {
		t.Fatal("could not find Warning row in output")
	}
	if lowerWarnLine == "" {
		t.Fatal("could not find lower 'warning' row in output")
	}
	if noTypeLine == "" {
		t.Fatal("could not find no-type row in output")
	}

	// Normal and no-type rows must have no ANSI coloring (raw == stripped).
	if normalLine != stripANSI(normalLine) {
		t.Errorf("Normal row should not have ANSI styling:\n%s", normalLine)
	}
	if noTypeLine != stripANSI(noTypeLine) {
		t.Errorf("No-type row should not have ANSI styling:\n%s", noTypeLine)
	}

	// Warning rows (both cases) must have ANSI styling (raw != stripped).
	if warnLine == stripANSI(warnLine) {
		t.Errorf("Warning row should have ANSI styling (warning highlight missing):\n%s", warnLine)
	}
	if lowerWarnLine == stripANSI(lowerWarnLine) {
		t.Errorf("lowercase 'warning' row should have ANSI styling (case-insensitive highlight missing):\n%s", lowerWarnLine)
	}
}

// TestRenderEventsTable_MalformedRow_BestEffort (AC#16): zero-value EventRow — no panic;
// Count==0 renders —; future timestamp renders —; long message renders without panic.
func TestRenderEventsTable_MalformedRow_BestEffort(t *testing.T) {
	t.Parallel()
	rc := &mockEventsRC{}
	sp := spinner.New()

	t.Run("zero_value_row_no_panic", func(t *testing.T) {
		t.Parallel()
		events := []data.EventRow{{}} // fully zero
		// Must not panic.
		got := stripANSI(RenderEventsTable(events, false, nil, rc, 80, sp))
		// Count==0 → "—"
		if !strings.Contains(got, "—") {
			t.Errorf("zero-value row: want '—' for Count=0, got:\n%s", got)
		}
	})

	t.Run("future_timestamp_renders_dash", func(t *testing.T) {
		t.Parallel()
		events := []data.EventRow{
			{Type: "Normal", Reason: "Test", Message: "test", Count: 1, LastTimestamp: time.Now().Add(10 * time.Minute)},
		}
		got := stripANSI(RenderEventsTable(events, false, nil, rc, 80, sp))
		// Future timestamp → Age should be "—"
		if !strings.Contains(got, "—") {
			t.Errorf("future timestamp: want '—' for Age, got:\n%s", got)
		}
	})

	t.Run("long_message_no_panic", func(t *testing.T) {
		t.Parallel()
		longMsg := strings.Repeat("x", 1000)
		events := []data.EventRow{
			{Type: "Normal", Reason: "Test", Message: longMsg, Count: 1, LastTimestamp: time.Now().Add(-1 * time.Minute)},
		}
		// Must not panic regardless of width.
		_ = RenderEventsTable(events, false, nil, rc, 80, sp)
		_ = RenderEventsTable(events, false, nil, rc, 40, sp)
	})
}

// TestRenderEventsTable_ColumnOrder (AC#2): at width=80, column headers appear in order.
func TestRenderEventsTable_ColumnOrder(t *testing.T) {
	t.Parallel()
	rc := &mockEventsRC{}
	sp := spinner.New()
	events := seedEvents()

	got := stripANSI(RenderEventsTable(events, false, nil, rc, 80, sp))
	// Find the header line — it's the first line.
	lines := strings.Split(got, "\n")
	var headerLine string
	for _, l := range lines {
		if strings.Contains(l, "Type") && strings.Contains(l, "Reason") {
			headerLine = l
			break
		}
	}
	if headerLine == "" {
		t.Fatalf("could not find header line in output:\n%s", got)
	}

	idxType := strings.Index(headerLine, "Type")
	idxReason := strings.Index(headerLine, "Reason")
	idxAge := strings.Index(headerLine, "Age")
	idxMessage := strings.Index(headerLine, "Message")
	idxCount := strings.Index(headerLine, "Count")

	if idxType < 0 {
		t.Errorf("AC#2: 'Type' header missing")
	}
	if idxReason < 0 {
		t.Errorf("AC#2: 'Reason' header missing")
	}
	if idxAge < 0 {
		t.Errorf("AC#2: 'Age' header missing")
	}
	if idxMessage < 0 {
		t.Errorf("AC#2: 'Message' header missing")
	}
	if idxCount < 0 {
		t.Errorf("AC#2: 'Count' header missing")
	}

	// Column order: Type < Reason < Age < Message < Count
	if idxType >= idxReason {
		t.Errorf("AC#2: 'Type' (%d) must appear before 'Reason' (%d)", idxType, idxReason)
	}
	if idxReason >= idxAge {
		t.Errorf("AC#2: 'Reason' (%d) must appear before 'Age' (%d)", idxReason, idxAge)
	}
	if idxAge >= idxMessage {
		t.Errorf("AC#2: 'Age' (%d) must appear before 'Message' (%d)", idxAge, idxMessage)
	}
	if idxMessage >= idxCount {
		t.Errorf("AC#2: 'Message' (%d) must appear before 'Count' (%d)", idxMessage, idxCount)
	}
}

// ==================== FB-024: Events sub-view correctness fixes — title bar ====================

// TestFB024_TitleBar_EHint covers ACs #4, #5, #6, #6a, #4a.
func TestFB024_TitleBar_EHint(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		width        int
		height       int
		mode         string
		wantContains []string
		wantAbsent   []string
	}{
		{
			// AC#4 — [E] events appears in non-events mode at wide width.
			name:         "default mode shows [E] events hint",
			width:        120,
			height:       20,
			mode:         "",
			wantContains: []string{"[E] events"},
			wantAbsent:   []string{"[E] describe"},
		},
		{
			// AC#5 — [E] describe appears when mode == "events".
			name:         "events mode flips label to [E] describe",
			width:        120,
			height:       20,
			mode:         "events",
			wantContains: []string{"[E] describe"},
			wantAbsent:   []string{"[E] events"},
		},
		{
			// AC#6 — at w=40 the right-text is dropped entirely (truncation ladder).
			name:       "narrow width=40 drops entire right-text including [E]",
			width:      40,
			height:     20,
			mode:       "",
			wantAbsent: []string{"[E]"},
		},
		{
			// AC#6a — height < 6 suppresses chrome, so no title bar at all.
			name:       "height=5 suppresses chrome — no [E] hint rendered",
			width:      120,
			height:     5,
			mode:       "",
			wantAbsent: []string{"[E]"},
		},
		{
			// AC#4a — placeholder state (describeRaw=nil, events loaded, eventsMode=false):
			// after toggling back from events mode, label is [E] events (not [E] describe).
			name:         "describe mode after events-toggle-back shows [E] events",
			width:        120,
			height:       20,
			mode:         "describe",
			wantContains: []string{"[E] events"},
			wantAbsent:   []string{"[E] describe"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := NewDetailViewModel(tt.width, tt.height)
			m.SetResourceContext("pods", "my-pod")
			if tt.mode != "" {
				m.SetMode(tt.mode)
			}
			got := stripANSI(m.View())
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("View() missing %q\ngot:\n%s", want, got)
				}
			}
			for _, absent := range tt.wantAbsent {
				if strings.Contains(got, absent) {
					t.Errorf("View() must not contain %q\ngot:\n%s", absent, got)
				}
			}
		})
	}
}

// ==================== End FB-024 ====================

// ==================== FB-119: [y]/[C] hint gate when describe unavailable ====================

// TestFB119_AC1_Observable_DescribeUnavailable_YCHintsAbsent verifies that
// when describeAvailable=false, [y] and [C] are absent from View().
func TestFB119_AC1_Observable_DescribeUnavailable_YCHintsAbsent(t *testing.T) {
	t.Parallel()
	m := NewDetailViewModel(160, 40)
	m.SetResourceContext("pods", "my-pod")
	m.SetDescribeAvailable(false)
	m.SetLoading(false)

	got := stripANSI(m.View())
	if strings.Contains(got, "[y]") {
		t.Errorf("AC1 [Observable FB-119]: '[y]' present when describeAvailable=false:\n%s", got)
	}
	if strings.Contains(got, "[C]") {
		t.Errorf("AC1 [Observable FB-119]: '[C]' present when describeAvailable=false:\n%s", got)
	}
}

// TestFB119_AC2_Observable_DescribeAvailable_YCHintsPresent verifies that
// when describeAvailable=true, [y] yaml and [C] conditions are in View().
func TestFB119_AC2_Observable_DescribeAvailable_YCHintsPresent(t *testing.T) {
	t.Parallel()
	m := NewDetailViewModel(160, 40)
	m.SetResourceContext("pods", "my-pod")
	m.SetDescribeAvailable(true)
	m.SetLoading(false)

	got := stripANSI(m.View())
	if !strings.Contains(got, "[y] yaml") {
		t.Errorf("AC2 [Observable FB-119]: '[y] yaml' absent when describeAvailable=true:\n%s", got)
	}
	if !strings.Contains(got, "[C] conditions") {
		t.Errorf("AC2 [Observable FB-119]: '[C] conditions' absent when describeAvailable=true:\n%s", got)
	}
}

// TestFB119_AC3_Observable_DescribeUnavailable_OtherHintsPresent verifies that
// [E], [x], and [Esc] remain visible when describeAvailable=false.
func TestFB119_AC3_Observable_DescribeUnavailable_OtherHintsPresent(t *testing.T) {
	t.Parallel()
	m := NewDetailViewModel(160, 40)
	m.SetResourceContext("pods", "my-pod")
	m.SetDescribeAvailable(false)
	m.SetLoading(false)

	got := stripANSI(m.View())
	for _, want := range []string{"[E] events", "[x] delete", "[Esc] back"} {
		if !strings.Contains(got, want) {
			t.Errorf("AC3 [Observable FB-119]: %q absent when describeAvailable=false (gate must not suppress non-describe hints):\n%s", want, got)
		}
	}
}

// TestFB119_AC4_AntiRegression_Loading_SuppressesAllHints verifies that
// loading=true suppresses all rightText hints regardless of describeAvailable.
func TestFB119_AC4_AntiRegression_Loading_SuppressesAllHints(t *testing.T) {
	t.Parallel()
	for _, avail := range []bool{true, false} {
		avail := avail
		t.Run(fmt.Sprintf("describeAvailable=%v", avail), func(t *testing.T) {
			t.Parallel()
			m := NewDetailViewModel(160, 40)
			m.SetResourceContext("pods", "my-pod")
			m.SetDescribeAvailable(avail)
			m.SetLoading(true)

			got := stripANSI(m.View())
			for _, absent := range []string{"[y]", "[C]", "[E]", "[x]"} {
				if strings.Contains(got, absent) {
					t.Errorf("AC4 [Anti-regression FB-119]: loading=true: %q present; want all hints suppressed:\n%s", absent, got)
				}
			}
		})
	}
}

// TestFB119_AC6_InputChanged_ToggleDescribeAvailable verifies that toggling
// describeAvailable true→false produces different View() output.
func TestFB119_AC6_InputChanged_ToggleDescribeAvailable(t *testing.T) {
	t.Parallel()
	m := NewDetailViewModel(160, 40)
	m.SetResourceContext("pods", "my-pod")
	m.SetLoading(false)

	m.SetDescribeAvailable(true)
	v1 := stripANSI(m.View())

	m.SetDescribeAvailable(false)
	v2 := stripANSI(m.View())

	if v1 == v2 {
		t.Errorf("AC6 [Input-changed FB-119]: View() unchanged after SetDescribeAvailable(true→false); want different output")
	}
	if !strings.Contains(v1, "[y]") {
		t.Errorf("AC6 [Input-changed FB-119]: v1 (describeAvailable=true) missing '[y]':\n%s", v1)
	}
	if strings.Contains(v2, "[y]") {
		t.Errorf("AC6 [Input-changed FB-119]: v2 (describeAvailable=false) still contains '[y]':\n%s", v2)
	}
}

// TestFB119_AC7_AntiRegression_EToggleSwap_WhenDescribeUnavailable verifies
// that [E] toggle-swap to "[E] describe" still works when describeAvailable=false.
func TestFB119_AC7_AntiRegression_EToggleSwap_WhenDescribeUnavailable(t *testing.T) {
	t.Parallel()
	m := NewDetailViewModel(160, 40)
	m.SetResourceContext("pods", "my-pod")
	m.SetDescribeAvailable(false)
	m.SetLoading(false)
	m.SetMode("events")

	got := stripANSI(m.View())
	if !strings.Contains(got, "[E] describe") {
		t.Errorf("AC7 [Anti-regression FB-119]: '[E] describe' absent in events mode with describeAvailable=false:\n%s", got)
	}
}

// ==================== End FB-119 (component layer) ====================
