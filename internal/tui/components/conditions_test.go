package components

// Axis-coverage table (FB-018, 23 ACs) — component-level tests
//
// This file covers the RenderConditionsTable rendering unit tests.
// Model-level tests (keybind routing, mode resets, lifecycle) live in
// internal/tui/model_test.go under the FB-018 section.
//
// Axis              | ACs covered here
// ------------------|------------------
// Happy/First-press | 1 (RenderConditionsTable called, returns content)
// Repeat-press      | 3 — (model_test.go)
// Input-changed     | 4, 5, 20 — (model_test.go)
// Anti-behavior     | 6, 7, 8, 9, 10 — (model_test.go)
// Failure/Edge      | 11, 12, 13, 18, 19
// Observable        | 14, 15, 16, 17, 19, 22
// Integration       | 21, 23 — (model_test.go)

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// TestMain sets a stable color profile for the entire components package test suite
// so that ANSI-color assertions in conditions tests produce deterministic output.
func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	m.Run()
}

// ── helpers ──────────────────────────────────────────────────────────────────

// seedObj builds an unstructured object whose .status.conditions field is set
// to the provided slice.
func seedObj(conditions []interface{}) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{
				"conditions": conditions,
			},
		},
	}
}

// condEntry builds a normal condition map with all five fields populated.
func condEntry(typ, status, reason, message, ltt string) interface{} {
	return map[string]interface{}{
		"type":               typ,
		"status":             status,
		"reason":             reason,
		"message":            message,
		"lastTransitionTime": ltt,
	}
}

// ── AC#11 — absent .status.conditions renders "No conditions reported" ────────

// TestRenderConditionsTable_NoConditions_Placeholder verifies AC#11:
// when .status.conditions is absent (key not present), the render returns the
// muted placeholder and does NOT render any column headers.
func TestRenderConditionsTable_NoConditions_Placeholder(t *testing.T) {
	t.Parallel()
	// .status exists but has no conditions key.
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{},
		},
	}
	out := stripANSI(RenderConditionsTable(obj, 80))
	if !strings.Contains(out, "No conditions reported") {
		t.Errorf("absent conditions: want placeholder, got: %q", out)
	}
	if strings.Contains(out, "Type") {
		t.Errorf("absent conditions: column header 'Type' must not appear, got: %q", out)
	}
}

// ── AC#12 — empty .status.conditions slice renders the same placeholder ───────

// TestRenderConditionsTable_EmptySlice_Placeholder verifies AC#12:
// when .status.conditions is an empty array, the render is identical to the
// absent case — same placeholder text, no column headers.
func TestRenderConditionsTable_EmptySlice_Placeholder(t *testing.T) {
	t.Parallel()
	obj := seedObj([]interface{}{})
	out := stripANSI(RenderConditionsTable(obj, 80))
	if !strings.Contains(out, "No conditions reported") {
		t.Errorf("empty conditions: want placeholder, got: %q", out)
	}
	if strings.Contains(out, "Type") {
		t.Errorf("empty conditions: column header 'Type' must not appear, got: %q", out)
	}
}

// ── AC#13 — malformed rows never panic; best-effort render ───────────────────

// TestRenderConditionsTable_MalformedRow_NoPanic verifies AC#13:
// adversarial condition entries (nil, non-map, partial maps, wrong-type fields,
// unparseable timestamps) MUST NOT panic and MUST return a non-empty string.
func TestRenderConditionsTable_MalformedRow_NoPanic(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		conditions []interface{}
	}{
		{
			name:       "nil entry",
			conditions: []interface{}{nil},
		},
		{
			name:       "non-map string entry",
			conditions: []interface{}{"some string"},
		},
		{
			// JSON-deserialized numbers are float64, never int. float64 is a
			// valid JSON type and exercises the "not a map" path in parseConditionRow.
			name:       "non-map float64 entry",
			conditions: []interface{}{float64(42)},
		},
		{
			name:       "slice-instead-of-map entry",
			conditions: []interface{}{[]interface{}{"a", "b"}},
		},
		{
			name: "partial map — only type field",
			conditions: []interface{}{
				map[string]interface{}{"type": "Ready"},
			},
		},
		{
			// JSON-deserialized numbers are float64, not int; int panics in NestedSlice deep-copy.
			name: "non-string status (float64)",
			conditions: []interface{}{
				map[string]interface{}{"type": "Ready", "status": float64(42)},
			},
		},
		{
			name: "unparseable lastTransitionTime",
			conditions: []interface{}{
				map[string]interface{}{
					"type":               "Ready",
					"status":             "True",
					"lastTransitionTime": "not-a-date",
				},
			},
		},
		{
			name: "mixed valid and nil entries",
			conditions: []interface{}{
				condEntry("Accepted", "True", "OK", "fine", "2026-04-18T14:22:03Z"),
				nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			obj := seedObj(tt.conditions)
			// If parseConditionRow panics, the test fails automatically.
			out := RenderConditionsTable(obj, 80)
			if out == "" {
				t.Errorf("%q: want non-empty output, got empty string", tt.name)
			}
		})
	}
}

// ── AC#14 — non-Ready rows rendered in styles.Warning ANSI color ─────────────

// TestRenderConditionsTable_NonReadyRow_WarningStyle verifies AC#14:
// rows where status != "True" are rendered with styles.Warning ANSI color.
// Uses the TrueColor profile set in TestMain so lipgloss emits ANSI codes.
func TestRenderConditionsTable_NonReadyRow_WarningStyle(t *testing.T) {
	t.Parallel()
	conditions := []interface{}{
		condEntry("Accepted", "True", "Accepted", "All good", "2026-04-18T14:22:03Z"),
		condEntry("Ready", "False", "NotReady", "Reconcile pending", "2026-04-19T02:15:00Z"),
		condEntry("ResolvedRefs", "Unknown", "ResolveTimeout", "Timed out", "2026-04-19T02:16:00Z"),
	}
	obj := seedObj(conditions)
	out := RenderConditionsTable(obj, 80)

	// Raw output must contain ANSI codes (from TrueColor-profiled warningStyle).
	if !strings.Contains(out, "\x1b[") {
		t.Error("want ANSI escape codes in output for non-Ready rows; got none")
	}

	// Stripped output must still contain all condition types.
	stripped := stripANSI(out)
	for _, want := range []string{"Accepted", "Ready", "ResolvedRefs"} {
		if !strings.Contains(stripped, want) {
			t.Errorf("stripped output missing %q\ngot:\n%s", want, stripped)
		}
	}
}

// ── AC#15/16/17/18/19 — width-band column-drop + off-by-one boundaries ────────

// TestRenderConditionsTable_WidthBands verifies AC#15–19:
// the correct column set is rendered at each width-band boundary.
// Off-by-one fixtures {39, 40, 59, 60, 79, 80} probe inclusive-lower /
// exclusive-upper semantics matching the FB-016 convention.
func TestRenderConditionsTable_WidthBands(t *testing.T) {
	t.Parallel()
	conditions := []interface{}{
		condEntry("Accepted", "True", "Accepted", "Route accepted", "2026-04-18T14:22:03Z"),
		condEntry("Ready", "False", "NotReady", "Pending", "2026-04-19T02:15:00Z"),
	}
	obj := seedObj(conditions)

	tests := []struct {
		name         string
		width        int
		wantContains []string
		wantAbsent   []string
	}{
		{
			name:         "w=39 unusable — placeholder only", // AC#18
			width:        39,
			wantContains: []string{"too narrow"},
			wantAbsent:   []string{"Type", "Status", "Reason", "Message", "LastTransitionTime"},
		},
		{
			name:         "w=40 narrow — Type/Status/Reason only", // AC#17
			width:        40,
			wantContains: []string{"Type", "Status", "Reason"},
			wantAbsent:   []string{"Message", "LastTransitionTime"},
		},
		{
			name:         "w=59 narrow — same as w=40", // AC#17
			width:        59,
			wantContains: []string{"Type", "Status", "Reason"},
			wantAbsent:   []string{"Message", "LastTransitionTime"},
		},
		{
			name:         "w=60 standard — Type/Status/Reason/LastTransitionTime", // AC#16
			width:        60,
			wantContains: []string{"Type", "Status", "Reason", "LastTransitionTime"},
			wantAbsent:   []string{"Message"},
		},
		{
			name:         "w=79 standard — same as w=60", // AC#16
			width:        79,
			wantContains: []string{"Type", "Status", "Reason", "LastTransitionTime"},
			wantAbsent:   []string{"Message"},
		},
		{
			name:         "w=80 wide — all 5 columns", // AC#15
			width:        80,
			wantContains: []string{"Type", "Status", "Reason", "Message", "LastTransitionTime"},
			wantAbsent:   nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out := stripANSI(RenderConditionsTable(obj, tt.width))
			for _, want := range tt.wantContains {
				if !strings.Contains(out, want) {
					t.Errorf("w=%d: want %q in output\ngot:\n%s", tt.width, want, out)
				}
			}
			for _, absent := range tt.wantAbsent {
				if strings.Contains(out, absent) {
					t.Errorf("w=%d: must NOT contain %q\ngot:\n%s", tt.width, absent, out)
				}
			}
		})
	}
}

// ── AC#13 extra — malformed .status structure → "Conditions unavailable" placeholder ──

// TestRenderConditionsTable_MalformedStructure_Placeholder verifies AC#13 §7a.1:
// when unstructured.NestedSlice returns an error (e.g., .status.conditions is a string,
// not a slice), the render returns the distinct "Conditions unavailable" placeholder —
// NOT "No conditions reported". This distinguishes benign-absence from malformed-shape.
func TestRenderConditionsTable_MalformedStructure_Placeholder(t *testing.T) {
	t.Parallel()
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{
				"conditions": "not-a-slice", // triggers NestedSlice error
			},
		},
	}
	out := stripANSI(RenderConditionsTable(obj, 80))
	if !strings.Contains(out, "Conditions unavailable") {
		t.Errorf("malformed .status.conditions: want 'Conditions unavailable' placeholder, got:\n%q", out)
	}
	if strings.Contains(out, "No conditions reported") {
		t.Errorf("malformed .status.conditions: got wrong placeholder ('No conditions reported'); must be 'Conditions unavailable'")
	}
}

// ── AC#13 em-dash — missing fields render as "—" at the cell layer ────────────

// TestRenderConditionsTable_MissingFields_EmDash verifies AC#13 §7c em-dash rule:
// when all condition fields are absent, each cell renders as em-dash "—",
// not as empty string or whitespace.
func TestRenderConditionsTable_MissingFields_EmDash(t *testing.T) {
	t.Parallel()
	obj := seedObj([]interface{}{
		map[string]interface{}{}, // all fields absent
	})
	out := stripANSI(RenderConditionsTable(obj, 80))
	if !strings.Contains(out, "—") {
		t.Errorf("all-missing fields: want em-dash '—' in rendered cells, got:\n%q", out)
	}
}

// ── AC#14 conservative — all 7 §6a classification rows ───────────────────────

// TestRenderConditionsTable_ConservativeWarningClassification verifies AC#14 §6a:
// ANY status value other than the exact string "True" (capital T, lowercase rest)
// is classified as non-Ready and rendered with warning color. This includes
// lowercase "true", empty string, missing field, and non-string values.
func TestRenderConditionsTable_ConservativeWarningClassification(t *testing.T) {
	t.Parallel()

	// findRowLine returns the raw (ANSI-inclusive) output line containing the given
	// type marker, skipping the header row (identified by "Type" AND "Reason" together)
	// and separator rows (containing the box-drawing character "─").
	findRowLine := func(out, typ string) string {
		for _, l := range strings.Split(out, "\n") {
			plain := stripANSI(l)
			isHeader := strings.Contains(plain, "Type") && strings.Contains(plain, "Reason")
			isSep := strings.Contains(plain, "─")
			if strings.Contains(plain, typ) && !isHeader && !isSep {
				return l
			}
		}
		return ""
	}

	tests := []struct {
		name     string
		cond     map[string]interface{}
		wantWarn bool
	}{
		{
			name:     "True — Ready (no warning)",
			cond:     map[string]interface{}{"type": "Accepted", "status": "True", "reason": "OK"},
			wantWarn: false,
		},
		{
			name:     "False — non-Ready (warning)",
			cond:     map[string]interface{}{"type": "Ready", "status": "False", "reason": "NotReady"},
			wantWarn: true,
		},
		{
			name:     "Unknown — non-Ready (warning)",
			cond:     map[string]interface{}{"type": "Sync", "status": "Unknown", "reason": "Pending"},
			wantWarn: true,
		},
		{
			name:     "empty string — non-Ready (conservative warning)",
			cond:     map[string]interface{}{"type": "Programmed", "status": "", "reason": "Empty"},
			wantWarn: true,
		},
		{
			name:     "lowercase true — non-Ready (conservative: Kubernetes emits capital True)",
			cond:     map[string]interface{}{"type": "Resolved", "status": "true", "reason": "lowercase"},
			wantWarn: true,
		},
		{
			name:     "missing status key — non-Ready (parsed as empty string)",
			cond:     map[string]interface{}{"type": "Bound", "reason": "NoStatusKey"},
			wantWarn: true,
		},
		{
			// JSON-deserialized numbers are float64; int panics in NestedSlice deep-copy.
			name:     "non-string status (float64) — non-Ready (parsed as empty string)",
			cond:     map[string]interface{}{"type": "Attached", "status": float64(42), "reason": "NonString"},
			wantWarn: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			obj := seedObj([]interface{}{tt.cond})
			out := RenderConditionsTable(obj, 80)
			typ, _ := tt.cond["type"].(string)
			line := findRowLine(out, typ)
			if line == "" {
				t.Fatalf("could not find data row for type=%q in output:\n%s", typ, stripANSI(out))
			}
			hasANSI := strings.Contains(line, "\x1b")
			if tt.wantWarn && !hasANSI {
				t.Errorf("%s: want warning ANSI on row, got none.\nRow: %q", tt.name, line)
			}
			if !tt.wantWarn && hasANSI {
				t.Errorf("%s: want no ANSI on True row, got ANSI.\nRow: %q", tt.name, line)
			}
		})
	}
}

// ── AC#22 (component side) — ShowConditionsHint renders [C] conditions entry ──

// TestHelpOverlayModel_ShowConditionsHint_Renders verifies AC#22 (component side):
// when ShowConditionsHint == true, the HelpOverlay View() contains "[C]    conditions";
// when false, that entry is absent.
func TestHelpOverlayModel_ShowConditionsHint_Renders(t *testing.T) {
	t.Parallel()

	t.Run("ShowConditionsHint=true — entry present", func(t *testing.T) {
		t.Parallel()
		m := NewHelpOverlayModel()
		m.ShowConditionsHint = true
		m.Width = 120
		m.Height = 40
		got := stripANSI(m.View())
		if !strings.Contains(got, "[C]    conditions") {
			t.Errorf("ShowConditionsHint=true: want '[C]    conditions' in View(), got:\n%s", got)
		}
		if !strings.Contains(got, "conditions") {
			t.Errorf("ShowConditionsHint=true: want 'conditions' in View(), got:\n%s", got)
		}
	})

	t.Run("ShowConditionsHint=false — entry absent", func(t *testing.T) {
		t.Parallel()
		m := NewHelpOverlayModel()
		m.ShowConditionsHint = false
		m.Width = 120
		m.Height = 40
		got := stripANSI(m.View())
		if strings.Contains(got, "[C]    conditions") {
			t.Errorf("ShowConditionsHint=false: '[C]    conditions' must not appear in View(), got:\n%s", got)
		}
	})
}
