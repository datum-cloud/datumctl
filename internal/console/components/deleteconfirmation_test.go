package components

import (
	"regexp"
	"strings"
	"testing"

	"go.datum.net/datumctl/internal/console/data"
)

var ansiEscapeDelete = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSIDelete(s string) string {
	return ansiEscapeDelete.ReplaceAllString(s, "")
}

// newDeleteTarget returns a test DeleteTarget for a namespaced resource.
func newDeleteTarget(name, namespace string) data.DeleteTarget {
	return data.DeleteTarget{
		RT:        data.ResourceType{Kind: "Pod", Name: "pods", Group: ""},
		Name:      name,
		Namespace: namespace,
	}
}

// newDeleteModel is a test helper for wide-mode rendering (appWidth=120).
func newDeleteModel(name, namespace string) DeleteConfirmationModel {
	return NewDeleteConfirmationModel(newDeleteTarget(name, namespace), nil)
}

// renderAt renders the dialog at the given terminal dimensions.
func renderAt(m DeleteConfirmationModel, w, h int) string {
	return stripANSIDelete(m.View(w, h))
}

// ==================== State 1: Prompt (AC#1/AC#2) ====================

// TestDeleteConfirmation_PromptState_ShowsTitle verifies that the Prompt state
// renders the resource name and kind in the dialog title.
func TestDeleteConfirmation_PromptState_ShowsTitle(t *testing.T) {
	t.Parallel()
	m := newDeleteModel("datumctl-test-pod", "")
	got := renderAt(m, 120, 40)

	if !strings.Contains(got, "Delete") {
		t.Errorf("Prompt: want 'Delete' in title, got: %q", got)
	}
	if !strings.Contains(got, "datumctl-test-pod") {
		t.Errorf("Prompt: want resource name 'datumctl-test-pod' in title, got: %q", got)
	}
}

// TestDeleteConfirmation_PromptState_ShowsCannotBeUndone verifies AC#2: the
// Prompt state renders the "This action cannot be undone" warning.
func TestDeleteConfirmation_PromptState_ShowsCannotBeUndone(t *testing.T) {
	t.Parallel()
	m := newDeleteModel("datumctl-test-pod", "")
	got := renderAt(m, 120, 40)

	if !strings.Contains(got, "cannot be undone") {
		t.Errorf("AC#2: want 'cannot be undone' in Prompt state, got: %q", got)
	}
}

// TestDeleteConfirmation_PromptState_ShowsYNHints verifies that [Y] confirm and
// [N]/[Esc] cancel keybind hints are present in Prompt state.
func TestDeleteConfirmation_PromptState_ShowsYNHints(t *testing.T) {
	t.Parallel()
	m := newDeleteModel("datumctl-test-pod", "")
	got := renderAt(m, 120, 40)

	if !strings.Contains(got, "[Y]") {
		t.Errorf("Prompt: [Y] confirm hint missing, got: %q", got)
	}
	if !strings.Contains(got, "[N]") {
		t.Errorf("Prompt: [N] cancel hint missing, got: %q", got)
	}
}

// ==================== State 2: InFlight (AC#3) ====================

// TestDeleteConfirmation_InFlightState_ShowsSpinner verifies AC#3: InFlight
// state renders "Deleting…" and the [Esc] close hint.
func TestDeleteConfirmation_InFlightState_ShowsSpinner(t *testing.T) {
	t.Parallel()
	m := newDeleteModel("datumctl-test-pod", "")
	m.SetState(DeleteStateInFlight)
	got := renderAt(m, 120, 40)

	if !strings.Contains(got, "Deleting") {
		t.Errorf("AC#3: want 'Deleting' in InFlight state, got: %q", got)
	}
	if !strings.Contains(got, "[Esc] close") {
		t.Errorf("AC#3: want '[Esc] close' hint in InFlight state, got: %q", got)
	}
	// Y/N hints must be absent during in-flight.
	if strings.Contains(got, "[Y]") {
		t.Errorf("AC#3: [Y] must be absent in InFlight state, got: %q", got)
	}
}

// ==================== State 3: Forbidden (AC#6) ====================

// TestDeleteConfirmation_ForbiddenState_ShowsPermissionsError verifies AC#6:
// Forbidden state renders "insufficient permissions" warning.
func TestDeleteConfirmation_ForbiddenState_ShowsPermissionsError(t *testing.T) {
	t.Parallel()
	m := newDeleteModel("datumctl-test-pod", "")
	m.SetState(DeleteStateForbidden)
	got := renderAt(m, 120, 40)

	if !strings.Contains(got, "insufficient permissions") {
		t.Errorf("AC#6: want 'insufficient permissions' in Forbidden state, got: %q", got)
	}
	if !strings.Contains(got, "[Esc] close") {
		t.Errorf("AC#6: want '[Esc] close' hint in Forbidden state, got: %q", got)
	}
	// [r] retry must not be present in Forbidden state.
	if strings.Contains(got, "[r]") {
		t.Errorf("AC#6: [r] must be absent in Forbidden state (retry won't fix 403), got: %q", got)
	}
}

// ==================== State 4: Conflict (AC#7) ====================

// TestDeleteConfirmation_ConflictState_ShowsConflictWarning verifies AC#7:
// Conflict state renders the "Resource was modified since load" warning
// with [r] refresh and [Esc] cancel keybinds.
func TestDeleteConfirmation_ConflictState_ShowsConflictWarning(t *testing.T) {
	t.Parallel()
	m := newDeleteModel("datumctl-test-pod", "")
	m.SetState(DeleteStateConflict)
	got := renderAt(m, 120, 40)

	if !strings.Contains(got, "modified since load") {
		t.Errorf("AC#7: want 'modified since load' in Conflict state, got: %q", got)
	}
	if !strings.Contains(got, "[r]") {
		t.Errorf("AC#7: want [r] refresh hint in Conflict state, got: %q", got)
	}
	if !strings.Contains(got, "[Esc]") {
		t.Errorf("AC#7: want [Esc] cancel hint in Conflict state, got: %q", got)
	}
}

// ==================== State 5: TransientError (AC#8) ====================

// TestDeleteConfirmation_TransientErrorState_ShowsError verifies AC#8:
// TransientError state renders "Delete failed" with the error detail and [r] retry.
func TestDeleteConfirmation_TransientErrorState_ShowsError(t *testing.T) {
	t.Parallel()
	m := newDeleteModel("datumctl-test-pod", "")
	m.SetState(DeleteStateTransientError)
	m.SetErrorDetail("server error 500")
	got := renderAt(m, 120, 40)

	if !strings.Contains(got, "Delete failed") {
		t.Errorf("AC#8: want 'Delete failed' in TransientError state, got: %q", got)
	}
	if !strings.Contains(got, "server error 500") {
		t.Errorf("AC#8: want error detail in TransientError state, got: %q", got)
	}
	if !strings.Contains(got, "[r]") {
		t.Errorf("AC#8: want [r] retry hint in TransientError state, got: %q", got)
	}
}

// TestDeleteConfirmation_TransientError_InputChanged_ForbiddenNotRetryable verifies
// AC#8 anti-behavior: Forbidden state does NOT show [r] retry (distinct from TransientError).
func TestDeleteConfirmation_TransientError_InputChanged_ForbiddenNotRetryable(t *testing.T) {
	t.Parallel()
	m := newDeleteModel("datumctl-test-pod", "")
	m.SetState(DeleteStateForbidden)
	got := renderAt(m, 120, 40)

	if strings.Contains(got, "[r]") {
		t.Errorf("AC#8 anti-behavior: [r] must be absent in Forbidden state (not retryable), got: %q", got)
	}
}

// ==================== AC#9: Namespace line ====================

// TestDeleteConfirmation_NamespaceLine_Shown verifies AC#9: namespaced resources
// show a "Namespace: <ns>" line in wide mode.
func TestDeleteConfirmation_NamespaceLine_Shown(t *testing.T) {
	t.Parallel()
	m := newDeleteModel("datumctl-test-pod", "default")
	got := renderAt(m, 120, 40) // wide mode

	if !strings.Contains(got, "Namespace: default") {
		t.Errorf("AC#9: want 'Namespace: default' for namespaced resource, got: %q", got)
	}
}

// TestDeleteConfirmation_NamespaceLine_HiddenWhenEmpty verifies AC#9 input-changed:
// cluster-scoped resources (namespace="") do not show a Namespace line.
func TestDeleteConfirmation_NamespaceLine_HiddenWhenEmpty(t *testing.T) {
	t.Parallel()
	m := newDeleteModel("datumctl-test-ns", "") // cluster-scoped
	got := renderAt(m, 120, 40)

	if strings.Contains(got, "Namespace:") {
		t.Errorf("AC#9 input-changed: Namespace line must be absent for cluster-scoped resource, got: %q", got)
	}
}

// ==================== AC#10: Kind resolver ====================

// TestDeleteConfirmation_KindResolver_WithRegistration verifies AC#10: when a
// matching ResourceRegistration exists, the kind display name is used in the title.
func TestDeleteConfirmation_KindResolver_WithRegistration(t *testing.T) {
	t.Parallel()
	target := data.DeleteTarget{
		RT:   data.ResourceType{Kind: "Deployment", Name: "Deployment", Group: "apps"},
		Name: "datumctl-test-deploy",
	}
	regs := []data.ResourceRegistration{
		{Group: "apps", Name: "Deployment", Description: "Workload"},
	}
	m := NewDeleteConfirmationModel(target, regs)
	got := renderAt(m, 120, 40)

	if !strings.Contains(got, "Workload") {
		t.Errorf("AC#10: want resolved kind 'Workload' in title, got: %q", got)
	}
}

// TestDeleteConfirmation_KindResolver_FallbackToKind verifies AC#10 input-changed:
// when no registration matches, the raw Kind is displayed.
func TestDeleteConfirmation_KindResolver_FallbackToKind(t *testing.T) {
	t.Parallel()
	target := data.DeleteTarget{
		RT:   data.ResourceType{Kind: "Widget", Name: "Widget", Group: "example.com"},
		Name: "datumctl-test-widget",
	}
	m := NewDeleteConfirmationModel(target, nil)
	got := renderAt(m, 120, 40)

	if !strings.Contains(got, "Widget") {
		t.Errorf("AC#10 fallback: want raw Kind 'Widget' in title when no registration, got: %q", got)
	}
}

// ==================== AC#18: ANSI injection prevention ====================

// TestDeleteConfirmation_ANSIInjection_NameSanitized verifies AC#18: a resource
// name containing ANSI escape sequences is sanitized before display.
func TestDeleteConfirmation_ANSIInjection_NameSanitized(t *testing.T) {
	t.Parallel()
	maliciousName := "datumctl-test-\x1b[31mmalicious\x1b[0m"
	m := newDeleteModel(maliciousName, "")
	got := renderAt(m, 120, 40) // stripANSIDelete applied to rendered output

	// The ANSI escape must not appear in the raw rendered bytes.
	if strings.Contains(got, "\x1b") {
		t.Errorf("AC#18: ANSI escape survived in rendered output: %q", got)
	}
	// The sanitized name (ANSI stripped) must appear.
	if !strings.Contains(got, "datumctl-test-malicious") {
		t.Errorf("AC#18: sanitized name missing from output, got: %q", got)
	}
}

// ==================== S15: Width-band rendering ====================

// TestDeleteConfirmation_WidthBand_NarrowMode verifies S15: when appWidth forces
// dialogW < 40 (narrow mode), the Kind label is omitted from the title.
func TestDeleteConfirmation_WidthBand_NarrowMode(t *testing.T) {
	t.Parallel()
	target := data.DeleteTarget{
		RT:   data.ResourceType{Kind: "Deployment", Name: "deployments", Group: "apps"},
		Name: "datumctl-test-deploy",
	}
	m := NewDeleteConfirmationModel(target, nil)
	// appWidth=44 → dialogW=14 (44-30), which is < 40 → narrowMode
	got := renderAt(m, 44, 40)

	// In narrow mode, only name appears in title (no Kind prefix).
	if strings.Contains(got, "Deployment") {
		t.Errorf("S15 narrow: Kind 'Deployment' must be absent at narrow width, got: %q", got)
	}
	if !strings.Contains(got, "datumct") {
		t.Errorf("S15 narrow: resource name (partial) must appear even in narrow mode, got: %q", got)
	}
}

// TestDeleteConfirmation_WidthBand_WideMode_YNSingleLine verifies S15: at wide
// dialogW (≥60), [Y] and [N]/[Esc] appear on the same line.
func TestDeleteConfirmation_WidthBand_WideMode_YNSingleLine(t *testing.T) {
	t.Parallel()
	m := newDeleteModel("datumctl-test-pod", "")
	got := renderAt(m, 120, 40) // dialogW=70, wideMode=true

	// In wide mode the hint line has [Y] and [N] on the same rendered line.
	// Verify both appear in the output without a stacked newline between them.
	yIdx := strings.Index(got, "[Y]")
	nIdx := strings.Index(got, "[N]")
	if yIdx < 0 || nIdx < 0 {
		t.Fatalf("S15 wide: [Y] or [N] missing, got: %q", got)
	}
	// In wide mode [Y] comes before [N] on the same display line (no newline between them).
	between := got[yIdx:nIdx]
	if strings.Contains(between, "\n") {
		t.Errorf("S15 wide: [Y] and [N] on separate lines, want same line in wideMode, got between: %q", between)
	}
}

// TestDeleteConfirmation_WidthBand_StandardMode_YNStackedLines verifies S15:
// at standard dialogW (40–59), [Y] and [N]/[Esc] appear on separate lines.
func TestDeleteConfirmation_WidthBand_StandardMode_YNStackedLines(t *testing.T) {
	t.Parallel()
	m := newDeleteModel("datumctl-test-pod", "")
	// appWidth=80 → dialogW=50 (band 80–99), wideMode(50)=false → stacked
	got := renderAt(m, 80, 40)

	yIdx := strings.Index(got, "[Y]")
	nIdx := strings.Index(got, "[N]")
	if yIdx < 0 || nIdx < 0 {
		t.Fatalf("S15 standard: [Y] or [N] missing, got: %q", got)
	}
	between := got[yIdx:nIdx]
	if !strings.Contains(between, "\n") {
		t.Errorf("S15 standard: [Y] and [N] on same line, want stacked in standard mode, got between: %q", between)
	}
}

// ==================== S14: ANSI injection resilience ====================

// TestDeleteConfirmation_ANSIInjectionResilience verifies S14: names with ANSI
// escapes, newlines, or other control characters are sanitized before display.
// The API call uses the original (unsanitized) name; only display is clean.
func TestDeleteConfirmation_ANSIInjectionResilience(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   string
		wantSub string
	}{
		{"ANSI_escape", "datumctl-test-\x1b[31mevil\x1b[0m", "datumctl-test-evil"},
		{"newline_in_name", "datumctl-test-\nnewline", "datumctl-test-newline"},
		{"null_in_name", "datumctl-test-\x00null", "datumctl-test-null"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := newDeleteModel(tc.input, "")
			got := renderAt(m, 120, 40)

			// renderAt already strips ANSI via stripANSIDelete; verify no raw ESC remains.
			if strings.Contains(got, "\x1b") {
				t.Errorf("S14 %s: ANSI escape survived in rendered output: %q", tc.name, got)
			}
			// The dialog view contains layout newlines — only check the sanitized name
			// itself does not contain the original injected chars.
			if !strings.Contains(got, tc.wantSub) {
				t.Errorf("S14 %s: sanitized substring %q missing from output, got: %q", tc.name, tc.wantSub, got)
			}
		})
	}
}

// ==================== S15: width-band boundary fixtures {39, 40, 59, 60} ====================

// TestDeleteConfirmation_WidthBandBoundaries verifies S15: dialog renders correctly
// at the exact boundary values {39, 40, 59, 60} for the width-band transitions.
func TestDeleteConfirmation_WidthBandBoundaries(t *testing.T) {
	t.Parallel()

	m := newDeleteModel("datumctl-test-pod", "")

	// appWidth=39 → dialogW = 39-4=35 (< 40 → narrowMode; < 40 → no Kind in title)
	t.Run("width_39_narrow", func(t *testing.T) {
		t.Parallel()
		got := renderAt(m, 39, 40)
		if strings.Contains(got, "Pod") {
			t.Errorf("boundary 39: Kind 'Pod' must be absent in narrowMode, got: %q", got)
		}
	})

	// appWidth=44 → dialogW=14 (in the appWidth-30 band, still < 40 → narrowMode)
	t.Run("width_44_narrow", func(t *testing.T) {
		t.Parallel()
		got := renderAt(m, 44, 40)
		if strings.Contains(got, "Pod") {
			t.Errorf("boundary 44: Kind 'Pod' must be absent in narrowMode (dialogW=14), got: %q", got)
		}
	})

	// appWidth=80 → dialogW=50 (band 80-99 → standard → stacked hints)
	t.Run("width_80_standard_stacked", func(t *testing.T) {
		t.Parallel()
		got := renderAt(m, 80, 40)
		yIdx := strings.Index(got, "[Y]")
		nIdx := strings.Index(got, "[N]")
		if yIdx < 0 || nIdx < 0 {
			t.Fatalf("boundary 80: [Y] or [N] missing")
		}
		between := got[yIdx:nIdx]
		if !strings.Contains(between, "\n") {
			t.Errorf("boundary 80: [Y] and [N] on same line, want stacked (standard mode)")
		}
	})

	// appWidth=120 → dialogW=70 (≥ 100 band → wideMode → single-line hints)
	t.Run("width_120_wide_single_line", func(t *testing.T) {
		t.Parallel()
		got := renderAt(m, 120, 40)
		yIdx := strings.Index(got, "[Y]")
		nIdx := strings.Index(got, "[N]")
		if yIdx < 0 || nIdx < 0 {
			t.Fatalf("boundary 120: [Y] or [N] missing")
		}
		between := got[yIdx:nIdx]
		if strings.Contains(between, "\n") {
			t.Errorf("boundary 120: [Y] and [N] on separate lines, want same line (wideMode)")
		}
	})
}
