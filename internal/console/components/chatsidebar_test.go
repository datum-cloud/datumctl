package components

import (
	"strings"
	"testing"
	"time"
)

func convEntries(n int) []ConvEntry {
	entries := make([]ConvEntry, n)
	for i := range entries {
		entries[i] = ConvEntry{
			ID:        "id",
			UpdatedAt: time.Now(),
			Preview:   strings.Repeat("x", 20),
		}
	}
	return entries
}

// TestChatSidebarModel_EmptyState verifies that an empty sidebar shows
// "(no saved chats)" in View().
func TestChatSidebarModel_EmptyState(t *testing.T) {
	t.Parallel()
	m := NewChatSidebarModel(30, 20)

	plain := stripANSI(m.View())

	if !strings.Contains(plain, "(no saved chats)") {
		t.Errorf("empty state: want '(no saved chats)' in View(), got %q", plain)
	}
}

// TestChatSidebarModel_PreviewTruncation verifies that a long preview is
// truncated and does not overflow the visible area.
func TestChatSidebarModel_PreviewTruncation(t *testing.T) {
	t.Parallel()
	width := 20
	m := NewChatSidebarModel(width, 20)
	m.SetHistory([]ConvEntry{{
		ID:        "id1",
		UpdatedAt: time.Now(),
		Preview:   strings.Repeat("x", width*3),
	}})

	plain := stripANSI(m.View())

	// PaneBorder adds ~10 chars of chrome (border + padding + trailing spaces).
	const borderOverhead = 12
	for _, line := range strings.Split(plain, "\n") {
		visLen := len([]rune(line))
		if visLen > width+borderOverhead {
			t.Errorf("line too wide (%d chars, want <= %d): %q", visLen, width+borderOverhead, line)
		}
	}
}

// TestChatSidebarModel_CursorDownPastEndIsNoOp verifies that CursorDown() at
// the last item does not advance the cursor further.
func TestChatSidebarModel_CursorDownPastEndIsNoOp(t *testing.T) {
	t.Parallel()
	m := NewChatSidebarModel(30, 20)
	m.SetHistory(convEntries(1))

	m.CursorDown()

	if m.HistoryCursor() != 0 {
		t.Errorf("CursorDown past end: HistoryCursor() = %d, want 0", m.HistoryCursor())
	}
}

// TestChatSidebarModel_CursorUpPastStartIsNoOp verifies that CursorUp() at the
// first item does not go negative.
func TestChatSidebarModel_CursorUpPastStartIsNoOp(t *testing.T) {
	t.Parallel()
	m := NewChatSidebarModel(30, 20)
	m.SetHistory(convEntries(2))
	m.CursorDown() // moves to 1
	m.CursorUp()   // moves to 0

	m.CursorUp() // should be a no-op at 0

	if m.HistoryCursor() != 0 {
		t.Errorf("CursorUp past start: HistoryCursor() = %d, want 0", m.HistoryCursor())
	}
}

// TestChatSidebarModel_FocusedCursorGlyph verifies that when focused the
// selected entry renders with the "▸" glyph.
func TestChatSidebarModel_FocusedCursorGlyph(t *testing.T) {
	t.Parallel()
	m := NewChatSidebarModel(30, 20)
	m.SetFocused(true)
	m.SetHistory([]ConvEntry{{
		ID:        "id1",
		UpdatedAt: time.Now(),
		Preview:   "test message",
	}})

	plain := stripANSI(m.View())

	if !strings.Contains(plain, "▸") {
		t.Errorf("focused sidebar: want '▸' glyph in View(), got %q", plain)
	}
	if !strings.Contains(plain, "test message") {
		t.Errorf("focused sidebar: want 'test message' in View(), got %q", plain)
	}
}

// TestChatSidebarModel_UnfocusedNoAccentGlyph verifies that when unfocused the
// selected item is rendered without the "▸" glyph.
func TestChatSidebarModel_UnfocusedNoAccentGlyph(t *testing.T) {
	t.Parallel()
	m := NewChatSidebarModel(30, 20)
	m.SetFocused(false)
	m.SetHistory([]ConvEntry{{
		ID:        "id1",
		UpdatedAt: time.Now(),
		Preview:   "my item",
	}})

	plain := stripANSI(m.View())

	if strings.Contains(plain, "▸") {
		t.Errorf("unfocused sidebar: want no '▸' glyph in View(), got %q", plain)
	}
	if !strings.Contains(plain, "my item") {
		t.Errorf("unfocused sidebar: want 'my item' in View(), got %q", plain)
	}
}

// TestChatSidebarModel_CursorNavigation verifies that CursorUp/Down move
// through history entries correctly.
func TestChatSidebarModel_CursorNavigation(t *testing.T) {
	t.Parallel()
	m := NewChatSidebarModel(30, 20)
	m.SetHistory(convEntries(3))

	m.CursorDown()
	if m.HistoryCursor() != 1 {
		t.Errorf("after CursorDown: HistoryCursor() = %d, want 1", m.HistoryCursor())
	}

	m.CursorDown()
	if m.HistoryCursor() != 2 {
		t.Errorf("after second CursorDown: HistoryCursor() = %d, want 2", m.HistoryCursor())
	}

	m.CursorUp()
	if m.HistoryCursor() != 1 {
		t.Errorf("after CursorUp: HistoryCursor() = %d, want 1", m.HistoryCursor())
	}

	m.CursorUp()
	if m.HistoryCursor() != 0 {
		t.Errorf("after second CursorUp: HistoryCursor() = %d, want 0", m.HistoryCursor())
	}
}
