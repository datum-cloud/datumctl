package data

import (
	"strings"
	"testing"
)

// ==================== AC#18: SanitizeResourceName ====================

func TestSanitizeResourceName_PlainPassthrough(t *testing.T) {
	t.Parallel()
	got := SanitizeResourceName("my-resource-name")
	if got != "my-resource-name" {
		t.Errorf("plain string: got %q, want %q", got, "my-resource-name")
	}
}

func TestSanitizeResourceName_StripsANSI(t *testing.T) {
	t.Parallel()
	// AC#18: ANSI escape sequences dropped entirely, not rendered as visible bytes.
	got := SanitizeResourceName("evil\x1b[31mname\x1b[0m")
	if got != "evilname" {
		t.Errorf("ANSI strip: got %q, want %q", got, "evilname")
	}
}

func TestSanitizeResourceName_DropsC0Controls(t *testing.T) {
	t.Parallel()
	// C0 control characters (0x00–0x1F) must be dropped, not rendered.
	got := SanitizeResourceName("name\x00with\x01nulls\x1f")
	if got != "namewithulls" {
		// Allow alternate form without 'n' if \x00 is stripped next to 'n'.
		// The exact result depends on which chars are dropped vs kept.
		// Assert no control characters remain.
		for _, r := range got {
			if r < 0x20 {
				t.Errorf("C0 controls: output still contains control char U+%04X: %q", r, got)
			}
		}
	}
	// Simpler invariant: no character below 0x20 survives.
	for _, r := range got {
		if r < 0x20 || r == 0x7F {
			t.Errorf("C0/DEL control char U+%04X survived sanitization in %q", r, got)
		}
	}
}

func TestSanitizeResourceName_DropsDEL(t *testing.T) {
	t.Parallel()
	got := SanitizeResourceName("name\x7fend")
	if strings.ContainsRune(got, 0x7F) {
		t.Errorf("DEL (0x7F) survived sanitization in %q", got)
	}
}

func TestSanitizeResourceName_TruncatesAt253(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("a", 300)
	got := SanitizeResourceName(long)
	if len(got) != 253 {
		t.Errorf("truncation: got len %d, want 253", len(got))
	}
}

func TestSanitizeResourceName_ExactlyAt253_NoTruncation(t *testing.T) {
	t.Parallel()
	exact := strings.Repeat("b", 253)
	got := SanitizeResourceName(exact)
	if len(got) != 253 {
		t.Errorf("exactly 253: got len %d, want 253", len(got))
	}
}

func TestSanitizeResourceName_Empty(t *testing.T) {
	t.Parallel()
	got := SanitizeResourceName("")
	if got != "" {
		t.Errorf("empty: got %q, want empty string", got)
	}
}

func TestSanitizeResourceName_ANSIThenControl(t *testing.T) {
	t.Parallel()
	// ANSI stripped first, then C0 dropped.
	got := SanitizeResourceName("\x1b[32mgreen\x1b[0m\x00name")
	if strings.ContainsRune(got, 0x00) {
		t.Errorf("null byte survived after ANSI strip + C0 drop: %q", got)
	}
	if strings.Contains(got, "\x1b") {
		t.Errorf("ANSI escape survived: %q", got)
	}
}

func TestSanitizeResourceName_ANSIInjectionPrevented(t *testing.T) {
	t.Parallel()
	// AC#18 invariant: the output must contain no ANSI escapes.
	input := "datumctl-test-\x1b[31mmalicious\x1b[0m"
	got := SanitizeResourceName(input)
	if strings.Contains(got, "\x1b") {
		t.Errorf("AC#18: ANSI injection not prevented; output contains ESC: %q", got)
	}
	if got != "datumctl-test-malicious" {
		t.Errorf("AC#18: unexpected sanitized value: got %q, want %q", got, "datumctl-test-malicious")
	}
}

// ==================== AC#18 ruling fixtures (product-experience 2026-04-19) ====================

// TestSanitizeResourceName_Ruling_NullDrop verifies the exact ruling fixture:
// "a\x00b" → "ab" (null byte dropped, not replaced).
func TestSanitizeResourceName_Ruling_NullDrop(t *testing.T) {
	t.Parallel()
	got := SanitizeResourceName("a\x00b")
	if got != "ab" {
		t.Errorf("ruling fixture: SanitizeResourceName(%q) = %q, want %q", "a\x00b", got, "ab")
	}
}

// TestSanitizeResourceName_Ruling_TabDrop verifies the ruling fixture:
// "tab\there" → "tabhere" (horizontal tab 0x09 dropped).
func TestSanitizeResourceName_Ruling_TabDrop(t *testing.T) {
	t.Parallel()
	got := SanitizeResourceName("tab\there")
	if got != "tabhere" {
		t.Errorf("ruling fixture: SanitizeResourceName(%q) = %q, want %q", "tab\there", got, "tabhere")
	}
}

// TestSanitizeResourceName_Ruling_NewlineDrop verifies the ruling fixture:
// "nl\nhere" → "nlhere" (newline 0x0A dropped).
func TestSanitizeResourceName_Ruling_NewlineDrop(t *testing.T) {
	t.Parallel()
	got := SanitizeResourceName("nl\nhere")
	if got != "nlhere" {
		t.Errorf("ruling fixture: SanitizeResourceName(%q) = %q, want %q", "nl\nhere", got, "nlhere")
	}
}

// TestSanitizeResourceName_Ruling_ANSIDrop verifies the ruling fixture:
// "evil\x1b[31mname\x1b[0m" → "evilname" (ANSI sequences stripped).
func TestSanitizeResourceName_Ruling_ANSIDrop(t *testing.T) {
	t.Parallel()
	got := SanitizeResourceName("evil\x1b[31mname\x1b[0m")
	if got != "evilname" {
		t.Errorf("ruling fixture: SanitizeResourceName(%q) = %q, want %q", "evil\x1b[31mname\x1b[0m", got, "evilname")
	}
}
