# FB-039 — DetailPane placeholder copy + title-bar mode coherence

**Status:** PENDING ENGINEER
**Priority:** P3
**Brief source:** `docs/tui-backlog.md` — search `### FB-039`
**Dependencies:** FB-024 ACCEPTED, FB-051 ACCEPTED

---

## 1. Problem statement

Two sub-problems with the same thesis: when describe is unavailable and events are
loaded, the DetailPane body and title-bar header should tell a single coherent story
that directs the operator to action.

**Sub-problem A (copy):** `buildDetailContent()` (model.go:2128) currently renders the
first body line as `"Describe unavailable — only events loaded."` — state-descriptive,
no action directive inline. The `[E]` affordance is in the `placeholderActionRow` below
and in the title-bar hint row, but the hint row may be suppressed during loading (FB-040)
or collapsed at narrow widths. The body's first line should be self-sufficient.

**Sub-problem B (header):** When the placeholder renders, the title bar shows
`"describe [unavailable]"` as the mode label (set by `detailModeLabel()`, `model.go:2086`
via FB-051). The body line says `"Describe unavailable"`. Both communicate the same fact;
neither shows a path forward.

---

## 2. Option selection

### Sub-problem B: Option B2 selected (keep mode label, update body copy)

**Why B2:**
FB-051 already changed the mode label from `"describe"` to `"describe [unavailable]"`.
The bare `"describe"` vs. `"Describe unavailable"` contradiction the brief identifies
is therefore already partially resolved in the codebase. The remaining coherence gap
is that both the header and the body communicate the state without directing action.

B2 resolves this by making the body copy action-directive. The header stays
`"describe [unavailable]"` — it identifies the mode and its state. The body becomes
`"Describe unavailable. Press [E] to view loaded events."` — it confirms the state and
provides the action path. These two lines are now complementary: mode identity in the
header, recovery instruction in the body.

**Why not B1 (suppress mode label):**
B1 would suppress the mode label entirely, requiring a change to `detailModeLabel()`
return value and introducing a new state branch (`mode = ""` while placeholder is
active). This is a state-machine change that needs its own axis coverage and
test-engineer round-trip. The benefit (eliminating visual redundancy between header
and body) is marginal given that FB-051's `"describe [unavailable]"` label is already
non-contradictory. B2 achieves the user goal — action-directive body — with a single
string change.

---

## 3. Ratified copy

### Sub-problem A: first body line

| | Text |
|---|---|
| **Before** | `"Describe unavailable — only events loaded."` |
| **After** | `"Describe unavailable. Press [E] to view loaded events."` |

`[E]` must be styled with `accentBold` (same style used in `placeholderActionRow`'s
`eKey`). The rest of the sentence uses `muted`. Render as three segments:

```
muted.Render("Describe unavailable. Press ") + accentBold.Render("[E]") + muted.Render(" to view loaded events.")
```

After ANSI stripping, this produces: `"Describe unavailable. Press [E] to view loaded events."`
The AC#1 substring `"Press [E]"` is present.

### Sub-problem B: mode label (unchanged)

`detailModeLabel()` continues to return `"describe [unavailable]"` when the placeholder
condition is met. **No change to `detailModeLabel()`.**

---

## 4. Render site

**Primary change:** `internal/tui/model.go` — `buildDetailContent()`, line 2128.

Replace:
```go
lines := []string{muted.Render("Describe unavailable — only events loaded.")}
```

With:
```go
lines := []string{
    muted.Render("Describe unavailable. Press ") + accentBold.Render("[E]") + muted.Render(" to view loaded events."),
}
```

**No other changes.** `placeholderActionRow`, `detailModeLabel()`, all key handlers,
and all other `buildDetailContent()` branches are untouched.

---

## 5. Interaction notes

### 5.1 FB-024 placeholder-wins-over-FB-022 precedence unchanged

The guard condition activating the placeholder —
`m.describeRaw == nil && m.events != nil && !m.yamlMode && !m.conditionsMode && !m.eventsMode`
— is not modified. The placeholder branch still runs before the FB-022 error-block
branch at model.go:2137. No change to ordering or precedence.

### 5.2 FB-018/FB-019 mode labels unaffected

`detailModeLabel()` only intercepts when the placeholder condition is met. When
`conditionsMode` or `eventsMode` is active, `m.detail.Mode()` returns the actual mode
unchanged. `"conditions"` and `"events"` mode labels are unaffected.

### 5.3 FB-026 notation mismatch — not in scope

FB-039's placeholder body uses `[E]` (title-bar convention). The HelpOverlay uses
`[Shift+E]` (a pre-existing mismatch tracked as FB-026). The engineer must not
conflate FB-039 with FB-026: the placeholder body intentionally uses `[E]`, and
`[E]` must not be changed to `[Shift+E]` here.

### 5.4 FB-040 loading suppression — no interaction

The placeholder activates only when `m.events != nil`. When `m.detail.Loading()` is
true, `detailModeLabel()` returns `""` (loading-mode suppression path). These two states
are mutually exclusive at the point where FB-039's copy change lives. FB-040 does not
block or conflict with FB-039.

### 5.5 `placeholderActionRow` — intentional redundancy

The `placeholderActionRow` already renders `[E] events` as a separate scannable action
line. With the new body copy, `[E]` appears in both the prose sentence and the action
row. This redundancy is intentional: the prose is for first-read comprehension; the
action row is for quick scannable reference. **No change to `placeholderActionRow`.**

---

## 6. Narrow-width behavior

The new first-line copy is 53 render chars
(`"Describe unavailable. Press [E] to view loaded events."`) vs. the old 42 chars.
The body renders inside the viewport (not the title bar), so it wraps naturally at the
terminal width. No width gate applies to this line. No truncation rule needed. The
`placeholderActionRow` width gate (`contentW < 40`) is unaffected.

---

## 7. Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable]** When placeholder is active (`describeRaw=nil`, `events!=nil`, not in
   yaml/conditions/events mode), `stripANSI(buildDetailContent())` contains `"Press [E]"`.
   Test: construct AppModel in placeholder state; call `buildDetailContent()`; assert
   substring present.

2. **[Anti-regression — old copy absent]** Same state: `stripANSI(buildDetailContent())`
   does NOT contain `"only events loaded"`. Test: same construct; assert old substring
   absent.

3. **[Observable / Input-changed — header-body coherence]** When placeholder is active,
   `stripANSI(detail.View())` contains both `"describe [unavailable]"` (mode label) AND
   `"Press [E]"` (action-directive body). Both are present simultaneously; no
   contradiction. Test: construct full model, call `detail.View()`, assert both substrings
   present.

4. **[Anti-regression]** FB-024 placeholder-wins-over-FB-022 ordering unchanged: when
   `describeRaw=nil`, `events!=nil`, and `loadErr!=nil` (errMode), the placeholder renders
   (not the FB-022 error block). The `TestFB024_*` suite stays green.

5. **[Anti-regression]** FB-018 conditions-mode and FB-019 events-mode title-bar labels
   render correctly. Existing detailview tests green.

6. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

---

## 8. Non-goals

- Not changing `detailModeLabel()` or its return values.
- Not changing `placeholderActionRow` (action row copy, keys, or logic).
- Not addressing the FB-026 `[Shift+E]` vs `[E]` notation mismatch in the HelpOverlay.
- Not changing when the placeholder triggers (that is FB-037's scope).
- Not changing the FB-025 `[r] refresh` affordance.
