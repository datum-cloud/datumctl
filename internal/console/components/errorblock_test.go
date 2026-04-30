package components

import (
	"context"
	"errors"
	"strings"
	"testing"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"go.datum.net/datumctl/internal/console/data"
)

// severityStub satisfies data.ResourceClient for ErrorSeverityOf tests.
type severityStub struct {
	unauthorized bool
	forbidden    bool
	notFound     bool
}

func (s severityStub) ListResourceTypes(_ context.Context) ([]data.ResourceType, error) {
	return nil, nil
}
func (s severityStub) ListResources(_ context.Context, _ data.ResourceType, _ string) ([]data.ResourceRow, []string, error) {
	return nil, nil, nil
}
func (s severityStub) DescribeResource(_ context.Context, _ data.ResourceType, _, _ string) (data.DescribeResult, error) {
	return data.DescribeResult{}, nil
}
func (s severityStub) DeleteResource(_ context.Context, _ data.ResourceType, _, _ string) error {
	return nil
}
func (s severityStub) IsForbidden(err error) bool    { return s.forbidden && err != nil }
func (s severityStub) IsNotFound(err error) bool     { return s.notFound && err != nil }
func (s severityStub) IsConflict(_ error) bool       { return false }
func (s severityStub) IsUnauthorized(err error) bool { return s.unauthorized && err != nil }
func (s severityStub) ListEvents(_ context.Context, _, _, _ string) ([]data.EventRow, error) {
	return nil, nil
}
func (s severityStub) InvalidateResourceListCache(_ string) {}

// --- RenderErrorBlock ---

func TestRenderErrorBlock_WidthBands(t *testing.T) {
	t.Parallel()
	block := ErrorBlock{
		Title:    "An error",
		Detail:   "extra detail text",
		Actions:  []ActionHint{{Key: "r", Label: "retry"}, {Key: "Esc", Label: "back"}},
		Severity: data.ErrorSeverityWarning,
	}

	tests := []struct {
		name              string
		width             int
		wantDetail        bool
		wantRetry         bool
		wantBlankSections bool // wide only: blank lines between sections
		wantSingleLine    bool // collapsed: no newline at all
	}{
		{
			name:              "wide (w=80)",
			width:             80,
			wantDetail:        true,
			wantRetry:         true,
			wantBlankSections: true,
		},
		{
			name:              "narrow (w=50)",
			width:             50,
			wantDetail:        true,
			wantRetry:         true,
			wantBlankSections: false,
		},
		{
			name:              "unusable (w=30)",
			width:             30,
			wantDetail:        false,
			wantRetry:         true,
			wantBlankSections: false,
		},
		{
			name:           "collapsed (w=10)",
			width:          10,
			wantDetail:     false,
			wantRetry:      false,
			wantSingleLine: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			b := block
			b.Width = tt.width
			plain := stripANSI(RenderErrorBlock(b))

			if !strings.Contains(plain, "An error") {
				t.Errorf("width=%d: want title 'An error' in output, got %q", tt.width, plain)
			}
			if tt.wantDetail && !strings.Contains(plain, "extra detail text") {
				t.Errorf("width=%d: want detail 'extra detail text' in output, got %q", tt.width, plain)
			}
			if !tt.wantDetail && strings.Contains(plain, "extra detail text") {
				t.Errorf("width=%d: detail must NOT appear at this width, got %q", tt.width, plain)
			}
			if tt.wantRetry && !strings.Contains(plain, "[r]") {
				t.Errorf("width=%d: want '[r]' action hint, got %q", tt.width, plain)
			}
			if !tt.wantRetry && strings.Contains(plain, "[r]") {
				t.Errorf("width=%d: '[r]' action must NOT appear at this width, got %q", tt.width, plain)
			}
			if tt.wantBlankSections && !strings.Contains(plain, "\n\n") {
				t.Errorf("width=%d (wide): want blank separator lines (\\n\\n), got %q", tt.width, plain)
			}
			if !tt.wantBlankSections && !tt.wantSingleLine && strings.Contains(plain, "\n\n") {
				t.Errorf("width=%d (narrow/unusable): must NOT have blank separator lines, got %q", tt.width, plain)
			}
			if tt.wantSingleLine && strings.Contains(plain, "\n") {
				t.Errorf("width=%d (collapsed): want single-line output (no \\n), got %q", tt.width, plain)
			}
		})
	}
}

func TestRenderErrorBlock_EmptyTitle_DefaultsToError(t *testing.T) {
	t.Parallel()
	plain := stripANSI(RenderErrorBlock(ErrorBlock{Title: "", Width: 80}))
	if !strings.Contains(plain, "Error") {
		t.Errorf("empty title: want fallback 'Error', got %q", plain)
	}
}

func TestRenderErrorBlock_MoreThan4Actions_TruncatesWithEllipsis(t *testing.T) {
	t.Parallel()
	actions := []ActionHint{
		{Key: "1", Label: "one"},
		{Key: "2", Label: "two"},
		{Key: "3", Label: "three"},
		{Key: "4", Label: "four"},
		{Key: "5", Label: "five"}, // 5th — must be dropped, "…" appended
	}
	plain := stripANSI(RenderErrorBlock(ErrorBlock{
		Title: "Too many", Actions: actions, Width: 80,
	}))
	if strings.Contains(plain, "five") {
		t.Error(">4 actions: 5th action label must NOT appear in output")
	}
	if !strings.Contains(plain, "…") {
		t.Error(">4 actions: want '…' ellipsis marker, not found")
	}
	// First 4 must appear.
	for _, key := range []string{"[1]", "[2]", "[3]", "[4]"} {
		if !strings.Contains(plain, key) {
			t.Errorf(">4 actions: key %q missing from output: %q", key, plain)
		}
	}
}

func TestRenderErrorBlock_NegativeWidth_CollapsedOutput(t *testing.T) {
	t.Parallel()
	plain := stripANSI(RenderErrorBlock(ErrorBlock{Title: "bad", Width: -5}))
	if strings.Contains(plain, "\n") {
		t.Errorf("negative width: want single-line collapsed output, got %q", plain)
	}
	if !strings.Contains(plain, "bad") {
		t.Errorf("negative width: want title 'bad' in output, got %q", plain)
	}
}

func TestRenderErrorBlock_GlyphPerSeverity(t *testing.T) {
	t.Parallel()
	warnPlain := stripANSI(RenderErrorBlock(ErrorBlock{Title: "warn", Severity: data.ErrorSeverityWarning, Width: 80}))
	errPlain := stripANSI(RenderErrorBlock(ErrorBlock{Title: "err", Severity: data.ErrorSeverityError, Width: 80}))

	if !strings.Contains(warnPlain, "⚠") {
		t.Errorf("Warning severity: want '⚠' glyph, got %q", warnPlain)
	}
	if !strings.Contains(errPlain, "✕") {
		t.Errorf("Error severity: want '✕' glyph, got %q", errPlain)
	}
	if strings.Contains(warnPlain, "✕") {
		t.Error("Warning severity: must NOT contain '✕' glyph")
	}
	if strings.Contains(errPlain, "⚠") {
		t.Error("Error severity: must NOT contain '⚠' glyph")
	}
}

func TestRenderErrorBlock_NoActionsNoDetail_TitleOnly(t *testing.T) {
	t.Parallel()
	plain := stripANSI(RenderErrorBlock(ErrorBlock{Title: "Oops", Width: 80}))
	if !strings.Contains(plain, "Oops") {
		t.Errorf("want 'Oops', got %q", plain)
	}
	// No blank-line separators when there's no detail or actions.
	if strings.Contains(plain, "\n\n") {
		t.Errorf("no detail/actions: want no blank separators, got %q", plain)
	}
}

// --- SanitizeErrMsg ---

func TestSanitizeErrMsg(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "nil error returns empty string",
			err:  nil,
			want: "",
		},
		{
			name: "first line only when newline present",
			err:  errors.New("first line\nsecond line"),
			want: "first line",
		},
		{
			name: "JSON Status body extracts message field",
			err:  errors.New(`{"kind":"Status","apiVersion":"v1","message":"decoded message"}`),
			want: "decoded message",
		},
		{
			name: "JSON without message field left as-is",
			err:  errors.New(`{"kind":"Status","code":500}`),
			want: `{"kind":"Status","code":500}`,
		},
		{
			name: "ANSI escapes stripped",
			err:  errors.New("\x1b[31mred error\x1b[0m"),
			want: "red error",
		},
		{
			name: "exactly 80 chars preserved",
			err:  errors.New(strings.Repeat("x", 80)),
			want: strings.Repeat("x", 80),
		},
		{
			name: "81 chars truncated to 77 + ellipsis",
			err:  errors.New(strings.Repeat("a", 81)),
			want: strings.Repeat("a", 77) + "…",
		},
		{
			name: "plain error with no special chars preserved",
			err:  errors.New("connection refused"),
			want: "connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := SanitizeErrMsg(tt.err)
			if got != tt.want {
				t.Errorf("SanitizeErrMsg(%v) = %q, want %q", tt.err, got, tt.want)
			}
		})
	}
}

// --- ErrorSeverityOf ---

func TestErrorSeverityOf(t *testing.T) {
	t.Parallel()
	someErr := errors.New("some error")

	tests := []struct {
		name string
		err  error
		rc   severityStub
		want data.ErrorSeverity
	}{
		{
			name: "nil error returns Warning",
			err:  nil,
			rc:   severityStub{},
			want: data.ErrorSeverityWarning,
		},
		{
			name: "Unauthorized → Error",
			err:  someErr,
			rc:   severityStub{unauthorized: true},
			want: data.ErrorSeverityError,
		},
		{
			name: "Forbidden → Error",
			err:  someErr,
			rc:   severityStub{forbidden: true},
			want: data.ErrorSeverityError,
		},
		{
			name: "NotFound → Error",
			err:  someErr,
			rc:   severityStub{notFound: true},
			want: data.ErrorSeverityError,
		},
		{
			name: "DeadlineExceeded → Warning",
			err:  context.DeadlineExceeded,
			rc:   severityStub{},
			want: data.ErrorSeverityWarning,
		},
		{
			name: "generic error → Warning",
			err:  someErr,
			rc:   severityStub{},
			want: data.ErrorSeverityWarning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ErrorSeverityOf(tt.err, tt.rc)
			if got != tt.want {
				t.Errorf("ErrorSeverityOf() = %v, want %v", got, tt.want)
			}
		})
	}
}

// --- sanitizedTitleForError (unexported) ---

var testGR = schema.GroupResource{Group: "test.io", Resource: "things"}

func TestSanitizedTitleForError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		err      error
		fallback string
		want     string
	}{
		{
			name:     "nil error returns fallback",
			err:      nil,
			fallback: "Could not load",
			want:     "Could not load",
		},
		{
			name:     "Unauthorized → Session expired",
			err:      k8serrors.NewUnauthorized("token expired"),
			fallback: "Could not load",
			want:     "Session expired",
		},
		{
			name:     "Forbidden → Permission denied",
			err:      k8serrors.NewForbidden(testGR, "myobj", errors.New("denied")),
			fallback: "Could not load",
			want:     "Permission denied",
		},
		{
			name:     "NotFound → Resource not found",
			err:      k8serrors.NewNotFound(testGR, "myobj"),
			fallback: "Could not load",
			want:     "Resource not found",
		},
		{
			name:     "DeadlineExceeded → Request timed out",
			err:      context.DeadlineExceeded,
			fallback: "Could not load",
			want:     "Request timed out",
		},
		{
			name:     "generic error returns fallback",
			err:      errors.New("something went wrong"),
			fallback: "Could not load history",
			want:     "Could not load history",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := sanitizedTitleForError(tt.err, tt.fallback)
			if got != tt.want {
				t.Errorf("sanitizedTitleForError() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- actionsForSeverity (unexported) ---

func TestActionsForSeverity(t *testing.T) {
	t.Parallel()
	backLabel := "back to navigation"

	t.Run("Warning returns retry + back", func(t *testing.T) {
		t.Parallel()
		got := actionsForSeverity(data.ErrorSeverityWarning, backLabel)
		if len(got) != 2 {
			t.Fatalf("Warning: want 2 actions, got %d", len(got))
		}
		if got[0].Key != "r" || got[0].Label != "retry" {
			t.Errorf("Warning[0]: want {r, retry}, got {%s, %s}", got[0].Key, got[0].Label)
		}
		if got[1].Key != "Esc" || got[1].Label != backLabel {
			t.Errorf("Warning[1]: want {Esc, %s}, got {%s, %s}", backLabel, got[1].Key, got[1].Label)
		}
	})

	t.Run("Error returns back only", func(t *testing.T) {
		t.Parallel()
		got := actionsForSeverity(data.ErrorSeverityError, backLabel)
		if len(got) != 1 {
			t.Fatalf("Error: want 1 action, got %d", len(got))
		}
		if got[0].Key != "Esc" || got[0].Label != backLabel {
			t.Errorf("Error[0]: want {Esc, %s}, got {%s, %s}", backLabel, got[0].Key, got[0].Label)
		}
	})

	t.Run("Warning never returns Error-only result", func(t *testing.T) {
		t.Parallel()
		got := actionsForSeverity(data.ErrorSeverityWarning, backLabel)
		hasRetry := false
		for _, a := range got {
			if a.Key == "r" {
				hasRetry = true
			}
		}
		if !hasRetry {
			t.Error("Warning actions must include retry key 'r'")
		}
	})

	t.Run("Error never includes retry hint", func(t *testing.T) {
		t.Parallel()
		got := actionsForSeverity(data.ErrorSeverityError, backLabel)
		for _, a := range got {
			if a.Key == "r" {
				t.Error("Error actions must NOT include retry key 'r'")
			}
		}
	})
}

// --- StatusBarModel.View() severity integration ---

func TestStatusBarModel_View_Severity(t *testing.T) {
	t.Parallel()
	someErr := errors.New("network timeout")

	tests := []struct {
		name        string
		severity    data.ErrorSeverity
		wantGlyph   string
		forbidGlyph string
	}{
		{
			name:        "Warning severity shows ⚠ glyph",
			severity:    data.ErrorSeverityWarning,
			wantGlyph:   "⚠",
			forbidGlyph: "✕",
		},
		{
			name:        "Error severity shows ✕ glyph",
			severity:    data.ErrorSeverityError,
			wantGlyph:   "✕",
			forbidGlyph: "⚠",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := StatusBarModel{
				Width:       80,
				Err:         someErr,
				ErrSeverity: tt.severity,
			}
			plain := stripANSI(m.View())
			if !strings.Contains(plain, tt.wantGlyph) {
				t.Errorf("severity=%v: want glyph %q in view, got %q", tt.severity, tt.wantGlyph, plain)
			}
			if strings.Contains(plain, tt.forbidGlyph) {
				t.Errorf("severity=%v: must NOT contain glyph %q, got %q", tt.severity, tt.forbidGlyph, plain)
			}
		})
	}
}

func TestStatusBarModel_View_ErrorMsgSanitized(t *testing.T) {
	t.Parallel()
	// SanitizeErrMsg is used — ANSI in the error must not bleed into the view.
	m := StatusBarModel{
		Width:       80,
		Err:         errors.New("\x1b[31mcolored error\x1b[0m"),
		ErrSeverity: data.ErrorSeverityWarning,
	}
	plain := stripANSI(m.View())
	if !strings.Contains(plain, "colored error") {
		t.Errorf("want sanitized message 'colored error' in view, got %q", plain)
	}
}

func TestStatusBarModel_View_NoError_NoErrBlock(t *testing.T) {
	t.Parallel()
	m := StatusBarModel{Width: 80, Err: nil}
	plain := stripANSI(m.View())
	if strings.Contains(plain, "⚠") || strings.Contains(plain, "✕") {
		t.Errorf("no error: must not show error glyph, got %q", plain)
	}
}

// --- Migration parity: error paths in historyview, activityview, quotadashboard, activitydashboard ---

func TestHistoryViewModel_ErrorPath_ContainsRetryAndEscHints(t *testing.T) {
	t.Parallel()
	m := NewHistoryViewModel(80, 20)
	m.SetResourceContext("Pod", "my-pod")
	m.SetError(errors.New("network error"), false)
	plain := stripANSI(m.View())
	if !strings.Contains(plain, "[r]") {
		t.Errorf("historyview error: want '[r]' retry hint, got %q", plain)
	}
	if !strings.Contains(plain, "[Esc]") {
		t.Errorf("historyview error: want '[Esc]' back hint, got %q", plain)
	}
}

func TestActivityViewModel_ErrorPath_ContainsRetryAndEscHints(t *testing.T) {
	t.Parallel()
	m := NewActivityViewModel(80, 20)
	m.SetError(errors.New("network error"), false)
	plain := stripANSI(m.View())
	if !strings.Contains(plain, "[r]") {
		t.Errorf("activityview error: want '[r]' retry hint, got %q", plain)
	}
	if !strings.Contains(plain, "[Esc]") {
		t.Errorf("activityview error: want '[Esc]' back hint, got %q", plain)
	}
}

func TestQuotaDashboardModel_ErrorPath_ContainsRetryAndEscHints(t *testing.T) {
	t.Parallel()
	// buildMainContent() is the canonical error rendering path for QuotaDashboard.
	m := NewQuotaDashboardModel(80, 20, "test-project")
	m.SetLoadErr(errors.New("quota load failed"))
	plain := stripANSI(m.buildMainContent())
	if !strings.Contains(plain, "[r]") {
		t.Errorf("quotadashboard error: want '[r]' retry hint, got %q", plain)
	}
	if !strings.Contains(plain, "[Esc]") {
		t.Errorf("quotadashboard error: want '[Esc]' back hint, got %q", plain)
	}
}

func TestActivityDashboardModel_ErrorPath_ContainsRetryAndEscHints(t *testing.T) {
	t.Parallel()
	m := NewActivityDashboardModel(80, 20, "test-project (proj)")
	m.SetLoadErr(errors.New("activity load failed"), false, false)
	plain := stripANSI(m.View())
	if !strings.Contains(plain, "[r]") {
		t.Errorf("activitydashboard error: want '[r]' retry hint, got %q", plain)
	}
	if !strings.Contains(plain, "[Esc]") {
		t.Errorf("activitydashboard error: want '[Esc]' back hint, got %q", plain)
	}
}

// Anti-behavior: YAML detail error (Actions==nil) must NOT show [r] retry.
// This is tested via model_test.go's YAML marshal error test; here we verify
// that RenderErrorBlock with nil Actions produces no retry hint.
func TestRenderErrorBlock_NilActions_NoRetryHint(t *testing.T) {
	t.Parallel()
	plain := stripANSI(RenderErrorBlock(ErrorBlock{
		Title:    "Could not render YAML",
		Detail:   "yaml: some error",
		Actions:  nil,
		Severity: data.ErrorSeverityError,
		Width:    80,
	}))
	if strings.Contains(plain, "[r]") {
		t.Errorf("nil Actions: must NOT contain '[r]' retry hint, got %q", plain)
	}
	if !strings.Contains(plain, "Could not render YAML") {
		t.Errorf("nil Actions: want title in output, got %q", plain)
	}
}

// --- titleAndDetailForError (unexported) ---

func TestTitleAndDetailForError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		err          error
		fallback     string
		wantTitle    string
		wantDetail   string
		detailIsExpr bool   // true when detail == SanitizeErrMsg(err) rather than a fixed string
	}{
		{
			name:       "nil error returns fallback title and empty detail",
			err:        nil,
			fallback:   "Could not load",
			wantTitle:  "Could not load",
			wantDetail: "",
		},
		{
			name:       "Unauthorized → Session expired + login hint",
			err:        k8serrors.NewUnauthorized("token expired"),
			fallback:   "fallback",
			wantTitle:  "Session expired",
			wantDetail: "Run `datumctl login` and try again.",
		},
		{
			name:       "Forbidden → Permission denied + action hint",
			err:        k8serrors.NewForbidden(testGR, "obj", errors.New("denied")),
			fallback:   "fallback",
			wantTitle:  "Permission denied",
			wantDetail: "You don't have permission to perform this action.",
		},
		{
			name:       "NotFound → Resource not found + rename hint",
			err:        k8serrors.NewNotFound(testGR, "obj"),
			fallback:   "fallback",
			wantTitle:  "Resource not found",
			wantDetail: "The resource has been removed or renamed.",
		},
		{
			name:       "DeadlineExceeded → Request timed out + server hint",
			err:        context.DeadlineExceeded,
			fallback:   "fallback",
			wantTitle:  "Request timed out",
			wantDetail: "Server did not respond in time.",
		},
		{
			name:         "generic error → fallback title + SanitizeErrMsg detail",
			err:          errors.New("connection refused"),
			fallback:     "Could not load history",
			wantTitle:    "Could not load history",
			detailIsExpr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotTitle, gotDetail := titleAndDetailForError(tt.err, tt.fallback)
			if gotTitle != tt.wantTitle {
				t.Errorf("title: got %q, want %q", gotTitle, tt.wantTitle)
			}
			if tt.detailIsExpr {
				// Generic path: detail must equal SanitizeErrMsg(err).
				wantDetail := SanitizeErrMsg(tt.err)
				if gotDetail != wantDetail {
					t.Errorf("detail (generic): got %q, want SanitizeErrMsg=%q", gotDetail, wantDetail)
				}
			} else if gotDetail != tt.wantDetail {
				t.Errorf("detail: got %q, want %q", gotDetail, tt.wantDetail)
			}
		})
	}
}

func TestTitleAndDetailForError_NoRedundancy(t *testing.T) {
	t.Parallel()
	// Classifier-matched errors must not repeat the title verbatim in the detail.
	// "Permission denied ... permission denied" is redundant; §8 canonical strings are distinct.
	cases := []struct {
		name string
		err  error
	}{
		{"Unauthorized", k8serrors.NewUnauthorized("x")},
		{"Forbidden", k8serrors.NewForbidden(testGR, "x", errors.New("denied"))},
		{"NotFound", k8serrors.NewNotFound(testGR, "x")},
		{"DeadlineExceeded", context.DeadlineExceeded},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			title, detail := titleAndDetailForError(c.err, "fallback")
			if strings.EqualFold(title, detail) {
				t.Errorf("%s: title and detail are identical (%q) — redundant", c.name, title)
			}
			if strings.Contains(strings.ToLower(detail), strings.ToLower(title)) {
				t.Errorf("%s: detail %q contains title %q — redundant pair", c.name, detail, title)
			}
		})
	}
}

// --- FB-022 hotfix: severity-driven anti-behavior (5 consumers × 2 variants) ---

// historyview

func TestHistoryViewModel_ErrorPath_Unauthorized_NoRetry(t *testing.T) {
	t.Parallel()
	m := NewHistoryViewModel(80, 20)
	m.SetResourceContext("Pod", "my-pod")
	m.SetError(k8serrors.NewUnauthorized("session expired"), false)
	plain := stripANSI(m.View())
	if !strings.Contains(plain, "Session expired") {
		t.Errorf("historyview Unauthorized: want 'Session expired' title, got %q", plain)
	}
	if strings.Contains(plain, "[r]") {
		t.Errorf("historyview Unauthorized (Error sev): must NOT contain '[r]' retry hint, got %q", plain)
	}
	if strings.Contains(strings.ToLower(plain), "retry") {
		t.Errorf("historyview Unauthorized (Error sev): must NOT contain 'retry', got %q", plain)
	}
	if !strings.Contains(plain, "[Esc]") {
		t.Errorf("historyview Unauthorized: want '[Esc]' back hint, got %q", plain)
	}
}

func TestHistoryViewModel_ErrorPath_Generic_HasRetry(t *testing.T) {
	t.Parallel()
	m := NewHistoryViewModel(80, 20)
	m.SetResourceContext("Pod", "my-pod")
	m.SetError(errors.New("connection refused"), false)
	plain := stripANSI(m.View())
	if !strings.Contains(plain, "[r]") {
		t.Errorf("historyview generic (Warning sev): want '[r]' retry hint, got %q", plain)
	}
	if !strings.Contains(plain, "[Esc]") {
		t.Errorf("historyview generic (Warning sev): want '[Esc]' back hint, got %q", plain)
	}
}

// activityview

func TestActivityViewModel_ErrorPath_Unauthorized_NoRetry(t *testing.T) {
	t.Parallel()
	m := NewActivityViewModel(80, 20)
	m.SetError(k8serrors.NewUnauthorized("session expired"), false)
	plain := stripANSI(m.View())
	if !strings.Contains(plain, "Session expired") {
		t.Errorf("activityview Unauthorized: want 'Session expired' title, got %q", plain)
	}
	if strings.Contains(plain, "[r]") {
		t.Errorf("activityview Unauthorized (Error sev): must NOT contain '[r]' retry hint, got %q", plain)
	}
	if strings.Contains(strings.ToLower(plain), "retry") {
		t.Errorf("activityview Unauthorized (Error sev): must NOT contain 'retry', got %q", plain)
	}
	if !strings.Contains(plain, "[Esc]") {
		t.Errorf("activityview Unauthorized: want '[Esc]' back hint, got %q", plain)
	}
}

func TestActivityViewModel_ErrorPath_Generic_HasRetry(t *testing.T) {
	t.Parallel()
	m := NewActivityViewModel(80, 20)
	m.SetError(errors.New("connection refused"), false)
	plain := stripANSI(m.View())
	if !strings.Contains(plain, "[r]") {
		t.Errorf("activityview generic (Warning sev): want '[r]' retry hint, got %q", plain)
	}
	if !strings.Contains(plain, "[Esc]") {
		t.Errorf("activityview generic (Warning sev): want '[Esc]' back hint, got %q", plain)
	}
}

// quotadashboard

func TestQuotaDashboardModel_ErrorPath_Unauthorized_NoRetry(t *testing.T) {
	t.Parallel()
	m := NewQuotaDashboardModel(80, 20, "test-project")
	m.SetLoadErr(k8serrors.NewUnauthorized("session expired"))
	plain := stripANSI(m.buildMainContent())
	if !strings.Contains(plain, "Session expired") {
		t.Errorf("quotadashboard Unauthorized: want 'Session expired' title, got %q", plain)
	}
	if strings.Contains(plain, "[r]") {
		t.Errorf("quotadashboard Unauthorized (Error sev): must NOT contain '[r]' retry hint, got %q", plain)
	}
	if strings.Contains(strings.ToLower(plain), "retry") {
		t.Errorf("quotadashboard Unauthorized (Error sev): must NOT contain 'retry', got %q", plain)
	}
	if !strings.Contains(plain, "[Esc]") {
		t.Errorf("quotadashboard Unauthorized: want '[Esc]' back hint, got %q", plain)
	}
}

func TestQuotaDashboardModel_ErrorPath_Generic_HasRetry(t *testing.T) {
	t.Parallel()
	m := NewQuotaDashboardModel(80, 20, "test-project")
	m.SetLoadErr(errors.New("connection refused"))
	plain := stripANSI(m.buildMainContent())
	if !strings.Contains(plain, "[r]") {
		t.Errorf("quotadashboard generic (Warning sev): want '[r]' retry hint, got %q", plain)
	}
	if !strings.Contains(plain, "[Esc]") {
		t.Errorf("quotadashboard generic (Warning sev): want '[Esc]' back hint, got %q", plain)
	}
}

// activitydashboard

func TestActivityDashboardModel_ErrorPath_Unauthorized_NoRetry(t *testing.T) {
	t.Parallel()
	// activitydashboard hardcodes its card title ("Recent activity temporarily
	// unavailable"), so we assert ✕ glyph and detail text instead of "Session expired".
	// The keybind strip always shows "[r] refresh" — we check "retry" absent (not "[r]")
	// since "retry" is the error card retry action label, while "refresh" is the footer.
	m := NewActivityDashboardModel(80, 20, "test-project (proj)")
	m.SetLoadErr(k8serrors.NewUnauthorized("session expired"), false, false)
	plain := stripANSI(m.View())
	if !strings.Contains(plain, "✕") {
		t.Errorf("activitydashboard Unauthorized (Error sev): want '✕' glyph, got %q", plain)
	}
	if strings.Contains(strings.ToLower(plain), "retry") {
		t.Errorf("activitydashboard Unauthorized (Error sev): must NOT contain 'retry' action, got %q", plain)
	}
	if !strings.Contains(plain, "[Esc]") {
		t.Errorf("activitydashboard Unauthorized: want '[Esc]' back hint, got %q", plain)
	}
}

func TestActivityDashboardModel_ErrorPath_Generic_HasRetry(t *testing.T) {
	t.Parallel()
	m := NewActivityDashboardModel(80, 20, "test-project (proj)")
	m.SetLoadErr(errors.New("connection refused"), false, false)
	plain := stripANSI(m.View())
	if !strings.Contains(plain, "[r]") {
		t.Errorf("activitydashboard generic (Warning sev): want '[r]' retry hint, got %q", plain)
	}
	if !strings.Contains(plain, "[Esc]") {
		t.Errorf("activitydashboard generic (Warning sev): want '[Esc]' back hint, got %q", plain)
	}
}

// ctxswitcher — produces LoadErrorMsg rather than rendering an inline card.
// Tests verify that data.SeverityOfClassified (used at ctxswitcher.go:116) routes
// Unauthorized to ErrorSeverityError (no retry) and generic errors to Warning (retry).

func TestCtxSwitcherModel_Unauthorized_SeverityIsError_NoRetry(t *testing.T) {
	t.Parallel()
	err := k8serrors.NewUnauthorized("session expired")
	sev := data.SeverityOfClassified(err)
	if sev != data.ErrorSeverityError {
		t.Errorf("SeverityOfClassified(Unauthorized) = %v, want ErrorSeverityError", sev)
	}
	// Simulate the card AppModel renders from ctxswitcher's LoadErrorMsg.
	plain := stripANSI(RenderErrorBlock(ErrorBlock{
		Title:    SanitizedTitleForError(err, "Save failed"),
		Detail:   SanitizeErrMsg(err),
		Actions:  ActionsForSeverity(sev, "back"),
		Severity: sev,
		Width:    80,
	}))
	if !strings.Contains(plain, "Session expired") {
		t.Errorf("ctxswitcher Unauthorized: want 'Session expired' title, got %q", plain)
	}
	if strings.Contains(plain, "[r]") {
		t.Errorf("ctxswitcher Unauthorized (Error sev): must NOT contain '[r]' retry hint, got %q", plain)
	}
	if strings.Contains(strings.ToLower(plain), "retry") {
		t.Errorf("ctxswitcher Unauthorized (Error sev): must NOT contain 'retry', got %q", plain)
	}
	if !strings.Contains(plain, "[Esc]") {
		t.Errorf("ctxswitcher Unauthorized: want '[Esc]' back hint, got %q", plain)
	}
}

func TestCtxSwitcherModel_Generic_SeverityIsWarning_HasRetry(t *testing.T) {
	t.Parallel()
	err := errors.New("connection refused")
	sev := data.SeverityOfClassified(err)
	if sev != data.ErrorSeverityWarning {
		t.Errorf("SeverityOfClassified(generic) = %v, want ErrorSeverityWarning", sev)
	}
	plain := stripANSI(RenderErrorBlock(ErrorBlock{
		Title:    SanitizedTitleForError(err, "Save failed"),
		Detail:   SanitizeErrMsg(err),
		Actions:  ActionsForSeverity(sev, "back"),
		Severity: sev,
		Width:    80,
	}))
	if !strings.Contains(plain, "[r]") {
		t.Errorf("ctxswitcher generic (Warning sev): want '[r]' retry hint, got %q", plain)
	}
	if !strings.Contains(plain, "[Esc]") {
		t.Errorf("ctxswitcher generic (Warning sev): want '[Esc]' back hint, got %q", plain)
	}
}

// --- Test C: Cross-consumer glyph parity (persona P2 #1) ---

// TestCrossConsumer_ForbiddenError_GlyphParity verifies that a Forbidden error
// produces the "✕" glyph on BOTH the status-bar surface and an in-pane card
// (historyview), proving same-error-same-glyph contract across surfaces.
func TestCrossConsumer_ForbiddenError_GlyphParity(t *testing.T) {
	t.Parallel()
	forbiddenErr := k8serrors.NewForbidden(testGR, "res-name", errors.New("denied"))
	sev := data.SeverityOfClassified(forbiddenErr)

	// Surface 1: status bar.
	sb := StatusBarModel{
		Width:       80,
		Err:         forbiddenErr,
		ErrSeverity: sev,
		Pane:        "TABLE",
	}
	sbPlain := stripANSI(sb.View())
	if !strings.Contains(sbPlain, "✕") {
		t.Errorf("status bar Forbidden: want '✕' glyph, got %q", sbPlain)
	}
	if strings.Contains(sbPlain, "⚠") {
		t.Errorf("status bar Forbidden: must NOT show '⚠' glyph, got %q", sbPlain)
	}

	// Surface 2: in-pane card (historyview).
	hv := NewHistoryViewModel(80, 20)
	hv.SetResourceContext("Pod", "my-pod")
	hv.SetError(forbiddenErr, false)
	hvPlain := stripANSI(hv.View())
	if !strings.Contains(hvPlain, "✕") {
		t.Errorf("historyview Forbidden: want '✕' glyph, got %q", hvPlain)
	}
	if strings.Contains(hvPlain, "⚠") {
		t.Errorf("historyview Forbidden: must NOT show '⚠' glyph, got %q", hvPlain)
	}
}
