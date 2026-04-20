package components

import (
	"errors"
	"strings"
	"testing"
	"time"

	"go.datum.net/datumctl/internal/tui/data"
)

// --- formatActivityTime ---

func TestFormatActivityTime_TodayShowsHHMMSS(t *testing.T) {
	t.Parallel()
	now := time.Now()
	got := formatActivityTime(now)
	// Expect HH:MM:SS format (8 chars).
	if len(got) != 8 || got[2] != ':' || got[5] != ':' {
		t.Errorf("formatActivityTime today = %q, want HH:MM:SS", got)
	}
}

func TestFormatActivityTime_ThisYearShowsMonDDHHMM(t *testing.T) {
	t.Parallel()
	// A date in this year but not today.
	t1 := time.Now().AddDate(0, -1, 0).Truncate(24 * time.Hour)
	if t1.Month() == time.Now().Month() && t1.Year() == time.Now().Year() {
		t.Skip("cannot construct a date in a different month for this test")
	}
	got := formatActivityTime(t1)
	// Expect "Jan 02 15:04" format — 13 chars, space between day and time.
	if len(got) < 12 {
		t.Errorf("formatActivityTime this-year = %q, want Jan DD HH:MM (len>=12)", got)
	}
}

func TestFormatActivityTime_PastYearShowsDate(t *testing.T) {
	t.Parallel()
	t1 := time.Date(2020, 6, 15, 12, 0, 0, 0, time.UTC)
	got := formatActivityTime(t1)
	if got != "2020-06-15" {
		t.Errorf("formatActivityTime past year = %q, want %q", got, "2020-06-15")
	}
}

// --- padRight ---

func TestPadRight_ShortString(t *testing.T) {
	t.Parallel()
	got := padRight("hi", 5)
	if got != "hi   " {
		t.Errorf("padRight short = %q, want %q", got, "hi   ")
	}
}

func TestPadRight_ExactLength(t *testing.T) {
	t.Parallel()
	got := padRight("hello", 5)
	if got != "hello" {
		t.Errorf("padRight exact = %q, want %q", got, "hello")
	}
}

func TestPadRight_TooLong(t *testing.T) {
	t.Parallel()
	got := padRight("toolong", 4)
	if got != "tool" {
		t.Errorf("padRight truncate = %q, want %q", got, "tool")
	}
}

func TestPadRight_ZeroWidth(t *testing.T) {
	t.Parallel()
	got := padRight("hi", 0)
	if got != "" {
		t.Errorf("padRight zero = %q, want %q", got, "")
	}
}

// --- truncate ---

func TestTruncate_FitsExact(t *testing.T) {
	t.Parallel()
	got := truncate("hello", 5)
	if got != "hello" {
		t.Errorf("truncate exact = %q, want %q", got, "hello")
	}
}

func TestTruncate_TooLong(t *testing.T) {
	t.Parallel()
	got := truncate("hello world", 8)
	if got != "hello w…" {
		t.Errorf("truncate = %q, want %q", got, "hello w…")
	}
}

func TestTruncate_NOne(t *testing.T) {
	t.Parallel()
	got := truncate("hello", 1)
	if got != "…" {
		t.Errorf("truncate n=1 = %q, want %q", got, "…")
	}
}

func TestTruncate_NZero(t *testing.T) {
	t.Parallel()
	got := truncate("hello", 0)
	if got != "…" {
		t.Errorf("truncate n=0 = %q, want %q", got, "…")
	}
}

// --- ActivityViewModel state helpers ---

func newTestActivityViewModel() ActivityViewModel {
	return NewActivityViewModel(80, 24)
}

func TestActivityViewModel_InitialState_NotLoading(t *testing.T) {
	t.Parallel()
	m := newTestActivityViewModel()
	if m.HasRows() {
		t.Error("HasRows() = true on fresh model, want false")
	}
	if m.NextContinue() != "" {
		t.Errorf("NextContinue() = %q, want empty", m.NextContinue())
	}
}

func TestActivityViewModel_SetRows_ClearsError(t *testing.T) {
	t.Parallel()
	m := newTestActivityViewModel()
	m.SetError(errors.New("oops"), false)
	m.SetRows([]data.ActivityRow{
		{Timestamp: time.Now(), Origin: "audit", Summary: "created"},
	}, "tok1")

	if !m.HasRows() {
		t.Error("HasRows() = false after SetRows, want true")
	}
	if m.NextContinue() != "tok1" {
		t.Errorf("NextContinue() = %q, want %q", m.NextContinue(), "tok1")
	}
}

func TestActivityViewModel_AppendRows_Accumulates(t *testing.T) {
	t.Parallel()
	m := newTestActivityViewModel()
	m.SetRows([]data.ActivityRow{
		{Timestamp: time.Now(), Origin: "audit", Summary: "first"},
	}, "tok1")
	m.AppendRows([]data.ActivityRow{
		{Timestamp: time.Now(), Origin: "event", Summary: "second"},
	}, "")

	if !m.HasRows() {
		t.Error("HasRows() = false after AppendRows, want true")
	}
	if m.NextContinue() != "" {
		t.Errorf("NextContinue() = %q after AppendRows with empty cont, want empty", m.NextContinue())
	}
}

func TestActivityViewModel_Reset_ClearsState(t *testing.T) {
	t.Parallel()
	m := newTestActivityViewModel()
	m.SetRows([]data.ActivityRow{
		{Timestamp: time.Now(), Origin: "audit", Summary: "created"},
	}, "tok1")
	m.Reset()

	if m.HasRows() {
		t.Error("HasRows() = true after Reset, want false")
	}
	if m.NextContinue() != "" {
		t.Errorf("NextContinue() = %q after Reset, want empty", m.NextContinue())
	}
}

// --- buildContent state routing ---

func TestActivityViewModel_BuildContent_LoadingState(t *testing.T) {
	t.Parallel()
	m := newTestActivityViewModel()
	m.SetLoading(true)
	got := stripANSI(m.buildContent())
	if !strings.Contains(got, "loading activity") {
		t.Errorf("loading state content = %q, want 'loading activity'", got)
	}
}

func TestActivityViewModel_BuildContent_UnauthorizedState(t *testing.T) {
	t.Parallel()
	m := newTestActivityViewModel()
	m.SetError(errors.New("forbidden"), true)
	got := stripANSI(m.buildContent())
	if !strings.Contains(got, "Activity is not enabled") {
		t.Errorf("unauthorized state = %q, want 'Activity is not enabled'", got)
	}
	// Must not show retry hint.
	if strings.Contains(got, "[r] retry") {
		t.Errorf("unauthorized state must not show [r] retry, got %q", got)
	}
}

func TestActivityViewModel_BuildContent_ErrorState(t *testing.T) {
	t.Parallel()
	m := newTestActivityViewModel()
	m.SetError(errors.New("internal server error"), false)
	got := stripANSI(m.buildContent())
	if !strings.Contains(got, "Could not load activity") {
		t.Errorf("error state = %q, want 'Could not load activity'", got)
	}
	if !strings.Contains(got, "internal server error") {
		t.Errorf("error state = %q, want error message text", got)
	}
	if !strings.Contains(got, "[r] retry") {
		t.Errorf("error state = %q, want '[r] retry'", got)
	}
}

func TestActivityViewModel_BuildContent_EmptyState(t *testing.T) {
	t.Parallel()
	m := newTestActivityViewModel()
	got := stripANSI(m.buildContent())
	if !strings.Contains(got, "No activity recorded") {
		t.Errorf("empty state = %q, want 'No activity recorded'", got)
	}
	if !strings.Contains(got, "30 days") {
		t.Errorf("empty state = %q, want '30 days' caveat", got)
	}
}

func TestActivityViewModel_BuildContent_RowsState(t *testing.T) {
	t.Parallel()
	m := newTestActivityViewModel()
	m.SetRows([]data.ActivityRow{
		{Timestamp: time.Now(), Origin: "audit", ActorDisplay: "user@example.com", ChangeSource: "human", Summary: "created widget"},
	}, "")
	got := stripANSI(m.buildContent())
	if !strings.Contains(got, "created widget") {
		t.Errorf("rows state = %q, want summary text", got)
	}
	// End marker must be present when nextContinue is empty.
	if !strings.Contains(got, "end of activity") {
		t.Errorf("rows state end marker = %q, want '— end of activity —'", got)
	}
}

func TestActivityViewModel_BuildContent_MoreMarker_WithNextContinue(t *testing.T) {
	t.Parallel()
	m := newTestActivityViewModel()
	m.SetRows([]data.ActivityRow{
		{Timestamp: time.Now(), Origin: "audit", Summary: "created"},
	}, "tok1")
	got := stripANSI(m.buildContent())
	if !strings.Contains(got, "more") {
		t.Errorf("more marker = %q, want 'more — scroll ↓' or similar", got)
	}
}

func TestActivityViewModel_BuildContent_DayBoundary_Separator(t *testing.T) {
	t.Parallel()
	m := newTestActivityViewModel()
	day1 := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	day2 := time.Date(2025, 1, 2, 11, 0, 0, 0, time.UTC)
	m.SetRows([]data.ActivityRow{
		{Timestamp: day1, Origin: "audit", Summary: "first day"},
		{Timestamp: day2, Origin: "audit", Summary: "second day"},
	}, "")
	got := stripANSI(m.buildContent())
	// Day boundary separator should appear between the two dates.
	if !strings.Contains(got, "2025-01-02") {
		t.Errorf("day boundary: want '2025-01-02' separator in %q", got)
	}
}

// --- View chrome at various heights ---

func TestActivityViewModel_View_TallEnough_HasChrome(t *testing.T) {
	t.Parallel()
	m := NewActivityViewModel(80, 24)
	m.SetResourceContext("Project", "my-project")
	m.SetRows([]data.ActivityRow{
		{Timestamp: time.Now(), Origin: "audit", Summary: "created"},
	}, "")
	got := stripANSI(m.View())
	if !strings.Contains(got, "my-project") {
		t.Errorf("View tall: want resource name in chrome, got %q", got)
	}
	if !strings.Contains(got, "TIME") {
		t.Errorf("View tall: want 'TIME' column header, got %q", got)
	}
}

func TestActivityViewModel_View_TooShort_MinimalContent(t *testing.T) {
	t.Parallel()
	// h < 6: no title bar chrome
	m := NewActivityViewModel(80, 4)
	m.SetRows([]data.ActivityRow{
		{Timestamp: time.Now(), Origin: "audit", Summary: "created"},
	}, "")
	got := stripANSI(m.View())
	// Should still render something (viewport only).
	if got == "" {
		t.Error("View short: expected non-empty output")
	}
	// Should NOT have the title bar with resource name (chrome suppressed).
	if strings.Contains(got, "TIME") {
		t.Errorf("View short height: should not render column header, got %q", got)
	}
}

// --- SetSize ---

func TestActivityViewModel_SetSize_ViewportHeightClamped(t *testing.T) {
	t.Parallel()
	m := NewActivityViewModel(80, 10)
	// After SetSize(80, 10), vpH = 10 - 5 = 5.
	m.SetSize(80, 10)
	if m.vp.Height != 5 {
		t.Errorf("vp.Height = %d, want %d (h-5 chrome)", m.vp.Height, 5)
	}
}

func TestActivityViewModel_SetSize_SmallHeight_MinViewport(t *testing.T) {
	t.Parallel()
	m := NewActivityViewModel(80, 3)
	m.SetSize(80, 3)
	// h < 6: no chrome deduction, vpH = h = 3.
	if m.vp.Height != 3 {
		t.Errorf("vp.Height = %d for h=3, want %d", m.vp.Height, 3)
	}
}
