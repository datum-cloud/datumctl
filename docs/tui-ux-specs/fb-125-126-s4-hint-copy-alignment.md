# FB-125 + FB-126 — S4 hint copy alignment (combined)

**Feature briefs:** FB-125, FB-126 (`docs/tui-backlog.md`)
**Status:** SPEC AUTHORED 2026-04-20 — ready for engineer routing
**Date:** 2026-04-20
**Depends on:** FB-124 PENDING ENGINEER (implements the string being changed here)

---

## 0. Design decisions

### D1 — FB-125: align to "next pane" (Option A)

Select **Option A**: change the S4 hint from `([Tab] to focus)` to `([Tab] next pane)`.

**Why:** The help overlay and status bar both describe the same Tab key as `[Tab] next pane` and `Tab  next pane` respectively (see `helpoverlay.go:28`, `resourcetable.go:732`, `resourcetable.go:750`). Operator muscle-memory cross-references these surfaces. Inconsistent verbs ("focus" vs "next pane") for the same key create brief confusion when a user presses `?` to look up a key they just encountered. Aligning to "next pane" removes that lookup friction.

Semantic note: "to focus" is marginally more goal-descriptive (tells the operator *why* to press Tab). "next pane" is the TUI's canonical label for Tab. Consistency wins over marginal semantic precision for a P3 discoverability hint.

### D2 — FB-126: resolved as side effect of D1 (Option D from FB-126 perspective)

The FB-126 concern was that `):` reads as "the sentence wants to end twice" — two closers back to back. This friction arises when the parenthetical contains a verb phrase (`to focus`), which makes the paren-close feel like a sentence-end before `:` introduces the list.

Changing to `([Tab] next pane)` replaces the verb phrase with a noun phrase. The pattern `(noun phrase):` is standard English — "Coffee options (choose one): espresso, latte" — where `)` closes the annotation and `:` opens the list. The double-closer reads naturally once the content inside the parens is a noun phrase, not an action verb. No structural change needed; no separate brief to implement.

### D3 — FB-127: close as WONTFIX

FB-127 flags that the longer hint prefix clips more destinations at narrow widths (~65 cols). This was documented as a known tradeoff in FB-124 spec §2.4. The copy change here saves 1 char (28 → 27 rendered chars), which is negligible. Adding width-conditional copy branches (shorter hint or suppressed hint below a threshold) introduces a new test axis for a minority case that does not affect any actionable workflow — operators in the hint state have dormant keys regardless of how many destinations are visible. Close as Option C (known tradeoff).

---

## 1. Code change

**One string, one line.**

**File:** `internal/tui/components/resourcetable.go`  
**Current (line 432):**
```go
prefixText = "jump to ([Tab] to focus):  "
```

**After:**
```go
prefixText = "jump to ([Tab] next pane):  "
```

No other changes. The `navPaneFocused` gate, setter, and `updatePaneFocus()` call site from FB-124 are untouched.

---

## 2. Layout diagrams

### 2.1 NavPane focused — before (FB-124 copy)

```
jump to ([Tab] to focus):  [b] backends  [n] networks  [w] workloads  …
```

### 2.2 NavPane focused — after (this spec)

```
jump to ([Tab] next pane):  [b] backends  [n] networks  [w] workloads  …
```

One char shorter. All entries shift left by 1 rendered char — no visible difference at normal widths.

### 2.3 TablePane focused — unchanged

```
jump to:  [b] backends  [n] networks  [w] workloads  [p] policies  …
```

### 2.4 Help overlay Tab row — unchanged

```
[Tab] next pane
```

The S4 hint now matches this established phrasing.

---

## 3. Styling

No style changes. `prefixText` continues to be rendered with the existing `muted` style. The `[Tab]` annotation remains part of the muted string — not accent-bold — preserving the visual distinction between the instructional token and the actionable `[b]`, `[n]` entries.

---

## 4. Acceptance criteria

| AC | Axis | Test target | Expected |
| --- | --- | --- | --- |
| **AC1** | `[Observable]` | New copy present when `navPaneFocused=true` | `newS4WelcomeModel()` + `SetNavPaneFocused(true)`. `stripANSI(m.View())` contains `"jump to ([Tab] next pane):"`. |
| **AC2** | `[Observable]` | Old copy absent when `navPaneFocused=true` | Same model. `stripANSI(m.View())` does NOT contain `"to focus"`. |
| **AC3** | `[Observable]` | Plain prefix unchanged when `navPaneFocused=false` | `SetNavPaneFocused(false)`. `stripANSI(m.View())` contains `"jump to:"` and does NOT contain `"next pane"` in the S4 prefix (help overlay may still contain it). |
| **AC4** | `[Input-changed]` | Toggle `navPaneFocused` true→false changes View() | Record `v1` with `navPaneFocused=true`, `v2` with `false`. `v1 != v2`. `v1` contains `"next pane"` in S4; `v2` does not. |
| **AC5** | `[Anti-regression]` | FB-124 conditional gate preserved — hint appears only when NavPane focused | AC1+AC3 together cover this; they verify the conditional is respected. |
| **AC6** | `[Anti-regression]` | Entry body unchanged — `[b] backends`, `[n] networks` etc. present regardless of `navPaneFocused` | Both true and false states: `stripANSI(m.View())` contains `"[b] backends"` and `"[n] networks"`. |
| **AC7** | `[Anti-regression]` | FB-073 tests still green | `TestFB073_AC1_NavPane_QuickJump_NoFire` and `TestFB073_AC3_TablePane_QuickJump_StillFires` pass. |
| **AC8** | `[Integration]` | AppModel: Tab pane-switch removes hint; new copy shows pre-Tab | `newFB124AppModel()` (NavPane, welcome, S4 visible). `stripANSI(appM.View())` contains `"next pane"`. Tab to TablePane. `stripANSI(appM.View())` does NOT contain `"next pane"` in S4 prefix. |
| **AC9** | `[Integration]` | Build + full test suite green | `go install ./...` clean. `go test ./internal/tui/...` exit 0. |

**Axis summary:** Observable × 3, Input-changed × 1, Anti-regression × 3, Integration × 2.

**Existing tests to update:** Any FB-124 test that asserts `"[Tab] to focus"` must be updated to assert `"[Tab] next pane"`. Search: `grep -n "to focus" internal/tui/` — update all such assertions. No logic changes; string assertions only.

---

## 5. Non-goals

- Not changing help overlay `[Tab] next pane` row (it already uses the correct phrasing).
- Not changing status bar `Tab  next pane` (already consistent).
- Not addressing narrow-width clipping (FB-127 closed as WONTFIX — see D3).
- Not changing FB-073 gate logic or FB-124 conditional-rendering logic.

---

## 6. Hand-off checklist

**Engineer:**
- [ ] `components/resourcetable.go:432` — change `"jump to ([Tab] to focus):  "` to `"jump to ([Tab] next pane):  "`
- [ ] Search and update any test assertions containing `"to focus"` in the S4 hint context
- [ ] Run `go install ./...` and `go test ./internal/tui/...`

**Test-engineer:**
- [ ] AC1 — new copy `"[Tab] next pane"` present when `navPaneFocused=true`
- [ ] AC2 — old copy `"to focus"` absent when `navPaneFocused=true`
- [ ] AC3 — plain prefix when `navPaneFocused=false`
- [ ] AC4 — Input-changed: toggle changes View()
- [ ] AC5 — FB-124 conditional gate preserved (covered by AC1+AC3)
- [ ] AC6 — Entry body unchanged regardless of focus state
- [ ] AC7 — FB-073 anti-regressions green
- [ ] AC8 — AppModel integration: Tab removes hint
- [ ] AC9 — Build + full suite green
- [ ] Axis-coverage table (Observable × 3, Input-changed × 1, Anti-regression × 3, Integration × 2) in submission message
