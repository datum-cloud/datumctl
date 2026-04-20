# FB-124 — S4 quick-jump focus-activation affordance

**Feature brief:** FB-124 (`docs/tui-backlog.md` — search `### FB-124`)
**Status:** SPEC AUTHORED 2026-04-20 — ready for engineer routing
**Date:** 2026-04-20
**Maps to:** Amendment to FB-042 spec §7 (S4 section) and FB-073 spec §4 (affordance surface)
**Depends on:** FB-073 ACCEPTED (`7c042aa`).

---

## 0. Design decisions

### D1 — Option A: conditional prefix hint when `navPaneFocused=true`

**Selected: Option A (conditional variant).** Options B, C, D are rejected.

**Why conditional, not static:**
The brief frames Option A as a static copy change — always showing the hint. Static copy produces a wrong affordance when `activePane == TablePane`: "jump to ([Tab] to focus):" tells an operator who already has focus that they need to tab. Conditional rendering shows the hint precisely when it's true and hides it when it isn't. This is the same gate-when-applicable principle used throughout the TUI hint system (FB-118, FB-119, FB-123).

**Why not Option B (dim-when-dormant):**
Dimmed bracket tokens would introduce a "visual dim = key disabled" convention with no existing precedent in the TUI. Other keys that are genuinely disabled are absent, not muted. A new visual convention for a P3 brief creates debt: future readers of the codebase would need to understand what dimmed-accent-bold means in S4 vs accent-bold.

**Why not Option C (footer line below S4):**
A separate line below S4 is spatially disconnected from the keys. An operator scanning S4 would read the keys first; a footer line registers as a caption, not an instruction for the keys just above it. Option A's inline placement is more directly associated with the key strip.

**Why not Option D (status-bar hint):**
The status bar is at the bottom of the screen; S4 is mid-panel. Spatial distance between signal and remedy is too large for a first-press discovery gap.

### D2 — Copy: `"jump to ([Tab] to focus):  "`

The hint uses `[Tab]` in bracket notation — matching the key-notation convention already established by S4's own entry format (`[b] backends`, `[n] networks`). The parenthetical `([Tab] to focus)` modifies the header without replacing it. Plain English reading: "jump to (you need to [Tab] to focus first)."

**Width cost:** `"jump to ([Tab] to focus):  "` = 28 rendered chars vs `"jump to:  "` = 10 rendered chars (+18 chars). At the minimum S4 content width (50 cols), 28 chars leaves 22 chars for entries. The existing trim-from-right mechanism in `renderQuickJumpSection` handles overflow gracefully — it already clips entries when the line is too long. No new width handling needed.

### D3 — One field, one call site

`ResourceTableModel.renderQuickJumpSection()` has no access to `activePane`. The same `navPaneFocused bool` field pattern used by `SetFocused(bool)` (which already exists on `ResourceTableModel`) applies here. AppModel sets it once, in `updatePaneFocus()` — the single canonical focus-distribution point already called in 30+ locations in model.go.

---

## 1. Code changes

### 1.1 `components/resourcetable.go` — struct field and setter

Add to `ResourceTableModel` struct:
```go
navPaneFocused bool
```

Add setter:
```go
func (m *ResourceTableModel) SetNavPaneFocused(focused bool) {
    m.navPaneFocused = focused
}
```

### 1.2 `components/resourcetable.go` — `renderQuickJumpSection()` prefix

**Before (line 420):**
```go
prefix := muted.Render("jump to:  ")
```

**After:**
```go
prefixText := "jump to:  "
if m.navPaneFocused {
    prefixText = "jump to ([Tab] to focus):  "
}
prefix := muted.Render(prefixText)
```

**That is the entire `renderQuickJumpSection()` change.** Three new lines; trim-from-right logic is unchanged.

### 1.3 `model.go` — `updatePaneFocus()` at line 855

Add one line inside `updatePaneFocus()`, alongside the existing `SetFocused` calls:

```go
m.table.SetNavPaneFocused(m.activePane == NavPane)
```

**One line. One call site.** Because every pane transition calls `updatePaneFocus()`, no other model sites need changes.

---

## 2. State diagrams

Width = 80 col welcome panel. S4 shown (contentH ≥ 18, contentW ≥ 50).

### 2.1 NavPane focused — default welcome-panel state (BEFORE fix)

```
jump to:  [b] backends  [n] networks  [w] workloads  [p] policies  [g] gateways  …
```

S4 keys look active (accent-bold brackets). Operator presses `[b]` → silent no-op.

### 2.2 NavPane focused — default welcome-panel state (AFTER fix)

```
jump to ([Tab] to focus):  [b] backends  [n] networks  [w] workloads  …
```

Hint is inline with the keys. The activation step is stated before the keys are listed.

### 2.3 TablePane focused (AFTER fix)

```
jump to:  [b] backends  [n] networks  [w] workloads  [p] policies  [g] gateways  …
```

Hint absent. Keys are active; no instruction needed.

### 2.4 Narrow width (contentW=50), NavPane focused

```
jump to ([Tab] to focus):  [b] backends  [n] …
```

Trim-from-right clips entries at 50 cols — same behavior as pre-fix narrow layout, but now prefix is longer. The `[b]` entry still renders; subsequent entries clip.

---

## 3. Acceptance criteria

| AC | Axis | Test target | Expected |
| --- | --- | --- | --- |
| **AC1** | `[Observable]` | Hint present when `navPaneFocused=true` and S4 visible | Construct `ResourceTableModel` at welcome-panel size (width=100, height=22), add matching registrations, call `SetNavPaneFocused(true)`. `stripANSI(m.View())` contains `"jump to ([Tab] to focus):"`. |
| **AC2** | `[Observable]` | Hint absent when `navPaneFocused=false` | Same construct, `SetNavPaneFocused(false)`. `stripANSI(m.View())` contains `"jump to:"` and does NOT contain `"[Tab] to focus"`. |
| **AC3** | `[Input-changed]` | Toggling `navPaneFocused` true→false changes View() | Record `v1 = stripANSI(m.View())` with `navPaneFocused=true`. Call `SetNavPaneFocused(false)`, record `v2`. Assert `v1 != v2`. `v1` contains `"[Tab] to focus"`; `v2` does not. |
| **AC4** | `[Anti-regression]` | FB-073 gate unchanged — `[b]` from NavPane is still a no-op | `TestFB073_AC1_NavPane_QuickJump_NoFire` still passes. AppModel with `activePane==NavPane`: pressing `b` produces nil cmd and no pane change. |
| **AC5** | `[Anti-regression]` | FB-073 anti-regression — `[b]` from TablePane still fires | `TestFB073_AC3_TablePane_QuickJump_StillFires` still passes. AppModel with `activePane==TablePane`: pressing `b` produces LoadResourcesCmd. |
| **AC6** | `[Anti-regression]` | S4 entry body unchanged — `[letter] type-name` format unaffected | `stripANSI(m.View())` still contains the key-label pairs (e.g., `"[b] backends"`, `"[n] networks"`) regardless of `navPaneFocused` value. Only the prefix changes; entries are untouched. |
| **AC7** | `[Anti-regression]` | S4 absent when no matching registrations — hint does not appear on empty S4 | `ResourceTableModel` with no matching registrations: `stripANSI(m.View())` contains neither `"jump to:"` nor `"[Tab] to focus"`. |
| **AC8** | `[Integration]` | AppModel integration — `updatePaneFocus()` propagates `navPaneFocused` correctly | Construct `AppModel` in welcome-panel state (`showDashboard=true`) with `activePane==NavPane`. `stripANSI(appM.View())` contains `"[Tab] to focus"`. Tab to `TablePane`. `stripANSI(appM.View())` does NOT contain `"[Tab] to focus"`. |
| **AC9** | `[Integration]` | Build + full test suite green | `go install ./...` clean. `go test ./internal/tui/...` green. |

---

## 4. Styling

No new tokens. `prefixText` is rendered with the existing `muted` style (`lipgloss.NewStyle().Foreground(styles.Muted)`). The bracket `[Tab]` inside the prefix is part of the muted string — it is NOT styled accent-bold. This intentionally distinguishes the instructional `[Tab]` annotation from the actionable `[b]`, `[n]` etc. entries that follow in accent-bold. The visual contrast reinforces: muted = instruction, accent-bold = live keys.

---

## 5. Non-goals

- Not changing FB-073's `activePane != NavPane` gate logic.
- Not dimming S4 bracket tokens (Option B rejected — introduces new visual convention).
- Not adding a help-overlay copy change (HelpOverlay already lists `[Tab] next pane` — the gap is welcome-panel specific).
- Not changing the letter→type mapping or letter set.
- Not addressing the hint at narrow widths beyond what the existing trim-from-right mechanism already handles.

---

## 6. Hand-off checklist

**Engineer:**
- [ ] `components/resourcetable.go` — add `navPaneFocused bool` field to `ResourceTableModel` struct (§1.1)
- [ ] `components/resourcetable.go` — add `SetNavPaneFocused(bool)` setter (§1.1)
- [ ] `components/resourcetable.go:420` — replace static prefix with conditional per §1.2
- [ ] `model.go:855` — add `m.table.SetNavPaneFocused(m.activePane == NavPane)` inside `updatePaneFocus()` (§1.3)
- [ ] Run `go install ./...` and `go test ./internal/tui/...`

**Test-engineer:**
- [ ] AC1 — `navPaneFocused=true`: hint present
- [ ] AC2 — `navPaneFocused=false`: hint absent, plain prefix
- [ ] AC3 — Input-changed: toggle true→false changes View()
- [ ] AC4 — FB-073 NavPane no-fire anti-regression
- [ ] AC5 — FB-073 TablePane still-fires anti-regression
- [ ] AC6 — Entry body unchanged regardless of `navPaneFocused`
- [ ] AC7 — No-registrations: S4 section (including hint) absent
- [ ] AC8 — AppModel integration: Tab pane-switch removes hint
- [ ] AC9 — `go install` + full suite green
- [ ] Axis-coverage table (Observable × 2, Input-changed × 1, Anti-regression × 4, Integration × 2) in submission message
