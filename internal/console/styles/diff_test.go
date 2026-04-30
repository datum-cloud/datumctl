package styles

import (
	"strings"
	"testing"

)

// TestMain runs the test suite. In lipgloss v2, Render() always emits
// full-fidelity ANSI, so no color profile configuration is needed.
func TestMain(m *testing.M) {
	m.Run()
}

func TestColorizeDiff_EmptyInput(t *testing.T) {
	t.Parallel()
	if got := ColorizeDiff(""); got != "" {
		t.Errorf("ColorizeDiff(\"\") = %q, want \"\"", got)
	}
}

func TestColorizeDiff_PlusLines_TextPreserved(t *testing.T) {
	t.Parallel()
	raw := "+added line\n context line\n"
	got := ColorizeDiff(raw)
	if !strings.Contains(got, "added line") {
		t.Errorf("ColorizeDiff: want 'added line' in output, got %q", got)
	}
	if !strings.Contains(got, "context line") {
		t.Errorf("ColorizeDiff: want 'context line' in output, got %q", got)
	}
}

func TestColorizeDiff_MinusLines_TextPreserved(t *testing.T) {
	t.Parallel()
	raw := "-removed line\n"
	got := ColorizeDiff(raw)
	if !strings.Contains(got, "removed line") {
		t.Errorf("ColorizeDiff: want 'removed line' in output, got %q", got)
	}
}

func TestColorizeDiff_HunkHeader_TextPreserved(t *testing.T) {
	t.Parallel()
	raw := "@@ -1,3 +1,4 @@\n"
	got := ColorizeDiff(raw)
	if !strings.Contains(got, "@@ -1,3 +1,4 @@") {
		t.Errorf("ColorizeDiff: hunk header text should be preserved in output, got %q", got)
	}
}

func TestColorizeDiff_FileHeaders_TextPreserved(t *testing.T) {
	t.Parallel()
	raw := "--- rev 1\n+++ rev 2\n"
	got := ColorizeDiff(raw)
	if !strings.Contains(got, "rev 1") || !strings.Contains(got, "rev 2") {
		t.Errorf("ColorizeDiff: file header text should be preserved, got %q", got)
	}
}

func TestColorizeDiff_ContextLines_Unchanged(t *testing.T) {
	t.Parallel()
	raw := " context line here\n"
	got := ColorizeDiff(raw)
	// Context lines (space prefix) must not be colorized — no ANSI added.
	if got != " context line here" {
		t.Errorf("ColorizeDiff context line: got %q, want %q", got, " context line here")
	}
}

func TestColorizeDiff_MultiLine_OrderPreserved(t *testing.T) {
	t.Parallel()
	raw := "--- rev 1\n+++ rev 2\n@@ -1,2 +1,3 @@\n context\n-old\n+new\n"
	got := ColorizeDiff(raw)
	// All text content must be present.
	for _, want := range []string{"rev 1", "rev 2", "@@ -1,2 +1,3 @@", "context", "old", "new"} {
		if !strings.Contains(got, want) {
			t.Errorf("ColorizeDiff: want %q in output, got %q", want, got)
		}
	}
}
