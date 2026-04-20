# FB-059 — Quota freshness gap-guard threshold over-triggers at split-pane widths

**Status:** PENDING ENGINEER
**Priority:** P3
**Brief source:** `docs/tui-backlog.md` — search `### FB-059`
**Dependencies:** FB-043 ACCEPTED

---

## 1. Problem statement

FB-043 added `"updated Xs ago"` freshness to the QuotaDashboard title bar. The guard drops freshness when `w - W(baseLeft+freshness) - W(hint) < 2`. The full hint string `"[↑/↓] move  [t] table  [s] group  [r] refresh"` is ~45 chars wide. With a typical baseLeft of ~11–36 chars and freshness of ~16–18 chars, freshness only appears when `w >= ~74–101`. An 80-col terminal with a split sidebar (22 cols) gives paneWidth ≈ 58 — below threshold in all practical configurations.

Root cause: the threshold check is `>= 2` (correct), but the hint string alone (~45 chars) consumes most of the available 58 cols before freshness is even considered. The threshold value is not the problem; the hint width is.

---

## 2. Pinned design: narrow-mode hint + compact freshness prefix

Two changes activate at `w < 80`. No changes at `w >= 80` (wide mode behavior unchanged).

### 2.1 Narrow-mode hint (`w < 80`)

Replace the full hint with a condensed form that drops the verbose labels:

| Width | Hint string | Approx width |
|---|---|---|
| `w >= 80` (wide) | `"[↑/↓] move  [t] table  [s] group  [r] refresh"` | ~45 chars |
| `w < 80` (narrow) | `"[↑/↓] [t] [s] [r]"` | ~18 chars |

The condensed hint preserves all four active keybinds. Their meaning is established by muscle memory and the HelpOverlay (`[?]`); verbose labels are redundant at narrow widths.

When `m.originLabel != ""` at narrow widths: the back-hint `"[3] back to <label>"` is preserved but the right-side nav block is condensed:
```
[3] back to <label>  [↑/↓] [t] [s] [r]
```

### 2.2 Compact freshness prefix (`w < 80`)

Replace the verbose `"  updated "` prefix (10 chars) with a compact separator `" · "` (3 chars):

| Width | Freshness string | Example | Approx width |
|---|---|---|---|
| `w >= 80` (wide) | `"  updated " + HumanizeSince(...)` | `"  updated 5m ago"` | ~16–18 chars |
| `w < 80` (narrow) | `" · " + HumanizeSince(...)` | `" · 5m ago"` | ~9–11 chars |

The `" · "` middle-dot separator is a common TUI annotation convention. In context (immediately after `"quota usage"` or `"quota usage — <label>"`), the age string is self-explanatory without the `"updated"` prefix.

The refreshing indicator at narrow widths: replace `"  ⟳ refreshing…"` with `" ↻"` (3 chars) — matches the compact prefix pattern.

### 2.3 Gate threshold: unchanged at `>= 2`

The `>= 2` gate is preserved. The narrow-mode changes free enough space that the gate is met at the representative split-pane width:

**Verification — paneWidth=58, no ctxLabel:**
- narrow hint = 18, compact freshness (`" · 5m ago"`) = 9
- candidate = baseLeft(11) + fresh(9) = 20
- gap = 58 − 20 − 18 = **20 ≥ 2** ✓

**Verification — paneWidth=58, short ctxLabel (8 chars, e.g. `"dev-proj"`):**
- baseLeft = 11+3+8 = 22, candidate = 22+9 = 31
- gap = 58 − 31 − 18 = **9 ≥ 2** ✓

**Verification — paneWidth=58, long ctxLabel (~22 chars):**
- baseLeft = 36, candidate = 36+9 = 45
- gap = 58 − 45 − 18 = **−5 < 2** — freshness still drops with very long labels
- This is acceptable: the AC test uses no ctxLabel; the long-label case is an acknowledged limitation, not a regression (it was always dropped before this fix)

**Verification — wide mode unchanged (w=100, no ctxLabel):**
- full hint = 45, full freshness = 16, candidate = 27
- gap = 100 − 27 − 45 = **28 ≥ 2** ✓ (same as today)

---

## 3. Render site — code shape

File: `internal/tui/components/quotadashboard.go` — `titleBar()` function.

The engineer should restructure `titleBar()` to compute `hint` and `freshPrefix` based on the `w` threshold before the freshness gate check:

```go
func (m QuotaDashboardModel) titleBar() string {
    w := m.width
    accentBold := lipgloss.NewStyle().Foreground(styles.Accent).Bold(true)
    muted := lipgloss.NewStyle().Foreground(styles.Muted)

    baseLeft := accentBold.Render("quota usage")
    if m.ctxLabel != "" {
        baseLeft += muted.Render(" — " + m.ctxLabel)
    }

    // Narrow-mode: condense hint and freshness prefix when width is constrained.
    var hint string
    var freshPrefix string
    var refreshingLabel string
    if w < 80 {
        if m.originLabel != "" {
            backHint := accentBold.Render("[3]") + muted.Render(" back to "+m.originLabel+"  ")
            hint = backHint + muted.Render("[↑/↓] [t] [s] [r]")
        } else {
            hint = muted.Render("[↑/↓] [t] [s] [r]")
        }
        freshPrefix = " · "
        refreshingLabel = " ↻"
    } else {
        if m.originLabel != "" {
            backHint := accentBold.Render("[3]") + muted.Render(" back to "+m.originLabel+"  ")
            hint = backHint + muted.Render("[↑/↓] move  [t] table  [s] group  [r] refresh")
        } else {
            hint = muted.Render("[↑/↓] move  [t] table  [s] group  [r] refresh")
        }
        freshPrefix = "  updated "
        refreshingLabel = "  ⟳ refreshing…"
    }

    left := baseLeft
    if m.refreshing {
        refresh := muted.Render(refreshingLabel)
        candidate := baseLeft + refresh
        if w-lipgloss.Width(candidate)-lipgloss.Width(hint) >= 2 {
            left = candidate
        }
    } else if !m.fetchedAt.IsZero() {
        fresh := muted.Render(freshPrefix + HumanizeSince(m.fetchedAt))
        candidate := baseLeft + fresh
        if w-lipgloss.Width(candidate)-lipgloss.Width(hint) >= 2 {
            left = candidate
        }
    }

    gap := max(1, w-lipgloss.Width(left)-lipgloss.Width(hint))
    return left + strings.Repeat(" ", gap) + hint
}
```

This is the intended shape — engineer may adjust style variable reuse as needed.

---

## 4. File locations

- **Primary change:** `internal/tui/components/quotadashboard.go` — `titleBar()` function
- **No changes:** `internal/tui/model.go`, `internal/tui/components/detailview.go` (DetailPane freshness handled separately; its `w < 60` drop threshold is independent and unchanged)
- **No changes:** HelpOverlay, key handlers, status bar

---

## 5. Acceptance criteria

Axis tags: `[Observable]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable]** At `paneWidth = 58` (no ctxLabel, `fetchedAt` populated), `stripANSI(QuotaDashboardModel.View())` contains the freshness age string (the output of `HumanizeSince`). Test: inject `width=58`, `fetchedAt=time.Now().Add(-5*time.Minute)`; assert `"5m ago"` (or equivalent HumanizeSince output) present.

2. **[Observable — compact prefix]** At `paneWidth = 58`, the wide-mode prefix `"updated"` is absent; the compact separator `" · "` is present adjacent to the age string. Test: assert `"updated"` absent, `"· "` present.

3. **[Observable — wide mode unchanged]** At `paneWidth = 100`, the full hint `"[↑/↓] move  [t] table  [s] group  [r] refresh"` is present and the full freshness prefix `"updated"` is present. Test: inject `width=100`, `fetchedAt` populated; assert both substrings present.

4. **[Anti-regression]** Extreme narrow (paneWidth ≤ 40): freshness drops cleanly (no overflow artifacts). The `>= 2` gate still fires; no new behavior at extreme widths.

5. **[Anti-regression]** FB-043 existing freshness tests green at wide widths.

6. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

---

## 6. Non-goals

- Not reverting FB-043 behavior at wide widths.
- Not changing the `>= 2` gate threshold.
- Not changing the DetailPane inline quota block freshness (its `w < 60` threshold is a separate concern — `detailview.go`).
- Not fixing the long-ctxLabel case at paneWidth=58 (freshness may still drop; acknowledged limitation).
- Not redesigning the hint row at wide widths.
- Not changing the `HumanizeSince` output format.
