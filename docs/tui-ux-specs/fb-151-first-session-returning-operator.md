# FB-151 — First-session vs returning-operator detection on welcome surface

**Status:** PENDING ENGINEER
**Priority:** P2
**Brief source:** `docs/tui-backlog.md` — search `### FB-151`
**Dependencies:** FB-105 ACCEPTED + PERSONA-EVAL-COMPLETE

---

## 1. Problem statement

The welcome panel renders identically for a first-time operator and someone on their
hundredth launch. First-timers lack the `[?]` help-overlay hint; returning operators
continue to see orientation copy they already know. Two audiences, one copy.

---

## 2. Option selection: Question 1 — What constitutes "first session"?

### Chosen: Option D — runtime-only heuristic, scoped to within-session distinction

**Why D:**
Options A and B (file marker / config counter) both require I/O mutations on every TUI
launch — new files, new config fields, new write paths. For a P2 copy-polish brief with
strict anti-regression requirements (FB-105/042/083 tests), introducing a persistence
layer is disproportionate. A silent write failure would leave the wrong copy in
perpetuity.

Option C generalizes into the approach described as D: use existing `m.typeName != ""`
(already tracked in `ResourceTableModel`) as the returning-operator proxy. When
`typeName == ""`, the operator has no cached resource-type navigation from a prior
session — treat as first-session for copy purposes. When `typeName != ""`, the operator
has navigated before — treat as returning. This is already the signal that drives the
`[Tab] resume` hint, so it carries existing semantics.

**Limitation acknowledged:** An operator who clears their config appears first-session.
This is acceptable — the copy difference is orientation copy, not critical information.
On re-authentication, the welcome panel re-shows first-session copy until they navigate
again.

**No new fields required.** `m.typeName` already exists in `ResourceTableModel`.

---

## 3. Option selection: Question 2 — Which surfaces personalize?

Two surfaces, both within existing regions (no new S1–S6 sections):

### Surface 1 — S1 orientation hint (line 3 in `renderHeaderBand`)

The existing hint fires at `renderHeaderBand` when:
`!m.forceDashboard && m.tuiCtx.ActiveCtx != nil && m.tuiCtx.ActiveCtx.ProjectID != "" && len(m.registrations) == 0`

This is already the first-session / cold-start state. The hint is naturally absent when
`m.forceDashboard=true` (returning operator who navigated: sees `[Tab] resume` instead).

Change: add `[?]` keybind inline to the hint — same line, same position. This is the
most impactful first-session affordance: the first-timer's one missing signal.

### Surface 2 — S4 all-clear block (in `welcomePanel`)

The all-clear line fires when `len(m.attentionItems) == 0 && !m.activityLoading && len(m.activityRows) == 0`.

A first-session operator sees the all-clear and has no commands yet. A returning operator
has muscle memory. Add a sub-line under the all-clear that is gated on `m.typeName == ""`
(first-session). This sub-line is the input-changed discriminator for AC#3.

---

## 4. Ratified copy

### 4.1 S1 orientation hint

| State | Condition | Copy |
|---|---|---|
| Before (current) | any | `"→  select a resource type from the sidebar to get started"` |
| First-session | `m.typeName == ""` | `"→  select a resource type from the sidebar, or press "` + `accentBold("[?]")` + `" for help"` |
| Returning (implicit) | `m.typeName != ""` | line3 branch not reached (either `[Tab] resume` shows, or registrations are loaded and line3 is empty) |

**FB-105 AC preservation:** New copy still contains `"select a resource type"` — the
AC#1 substring anchor is preserved. The trailing `"to get started"` changes to
`", or press [?] for help"`. If FB-105 tests assert the FULL line3 string (not just
the substring), test-engineer must update those assertions. The spec flags this — see
§6.

Render segments (line 3):
```
muted("→  select a resource type from the sidebar, or press ")
+ accentBold("[?]")
+ muted(" for help")
```

Width: this line is ~55 chars. The line3 branch already has no width gate (it renders
the full hint or drops to a shortened form). No new gate needed; the existing line3
truncation fallback applies.

### 4.2 S4 first-session sub-line

```
// After the existing "all clear · no issues detected" line:
if m.typeName == "" {
    → append: muted("→  press ") + accentBold("[?]") + muted(" to see available commands")
}
```

`stripANSI` output: `"→  press [?] to see available commands"`.

The sub-line renders only when S4 renders (`contentH >= 18 && contentW >= 50`). No
additional width gate — S4's existing gate is sufficient.

**Voice alignment:** Both lines use: muted style, lowercase, no exclamation, `[?]`
styled `accentBold`. Matches FB-105 register. Matches FB-152 (`"quota looks healthy"`)
register.

---

## 5. Render sites

- **S1 line 3:** `internal/tui/components/resourcetable.go` — `renderHeaderBand()`, the
  `else if m.tuiCtx.ActiveCtx != nil && ... len(m.registrations) == 0` branch (currently
  line 682).
- **S4 sub-line:** `internal/tui/components/resourcetable.go` — `welcomePanel()`, the
  `if showS4 { ... all-clear block }` (currently around line 227–234).
- **No other files.** `model.go`, `detailview.go`, keybind handlers: no changes.

---

## 6. FB-105 coordination note for test-engineer

FB-105 AC#1 asserts `"select a resource type"` as a substring — this passes unchanged.

If FB-105 tests assert the FULL current string `"→  select a resource type from the
sidebar to get started"` (exact match), those assertions must update to:
`"→  select a resource type from the sidebar, or press [?] for help"`.

Test-engineer should grep for the old string in `resourcetable_test.go` and update
any exact-match assertions. The spec designates this as a coordinated AC update, not
a regression.

---

## 7. Interaction with FB-042 / FB-083

- **FB-042 section ordering (S1–S6):** No new section added. S1 line3 is an intra-S1
  copy change. S4 sub-line is intra-S4 (appended within the existing S4 block, not a
  new block). Section indices and separator positions unchanged.
- **FB-083 attention-list:** The S5 attention-list rendering is unchanged. The FB-151
  sub-line in S4 renders only when `attentionItems` is empty — mutually exclusive with
  attention-list content by definition.

---

## 8. Interaction with FB-152

FB-151 (S1/S4 copy) and FB-152 (S2 header healthy copy) are independent ACs. The
shared register constraint: both use lowercase, no exclamation, muted/accentBold
styling. FB-152's `"quota looks healthy"` and FB-151's `"all clear · no issues detected"
+ first-session sub-line read coherently on the same welcome panel — S2 scopes to
quota, S4 scopes to overall platform. No copy collision.

---

## 9. Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable — first-session S1]** With `typeName=""`, `ActiveCtx.ProjectID!=""`,
   `registrations=nil`, `stripANSI(m.renderHeaderBand(80))` contains `"select a resource
   type"` AND `"[?]"`. Test: construct ResourceTableModel with empty typeName; assert
   both substrings.

2. **[Observable — first-session S4]** With `typeName=""`, `attentionItems=nil`,
   `activityLoading=false`, `activityRows=nil`, `contentH >= 18`, `contentW >= 50`:
   `stripANSI(m.welcomePanel())` contains `"press [?] to see available commands"`.
   Test: construct matching model; assert.

3. **[Input-changed — returning vs first-session at S4]** State A: `typeName=""` →
   welcome panel contains `"press [?] to see available commands"`. State B: same model
   with `typeName="projects"` (simulate post-navigation) → panel does NOT contain
   `"press [?] to see available commands"`. Two View() outputs differ at this substring.

4. **[Anti-regression — FB-105 substring preserved]** `"select a resource type"`
   substring still present in first-session state. If FB-105 tests assert the full
   old line3 string, test-engineer updates those assertions per §6. All other FB-105
   AC substrings (`"all clear"`, `"jump to:"`) unaffected.

5. **[Anti-regression — FB-042 section ordering]** Section indices and separator
   positions intact. Test: existing FB-042 S1–S6 ordering tests green.

6. **[Anti-regression — FB-083 attention-list]** Attention-list substring assertions
   unchanged. Test: existing FB-083 tests green.

7. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

---

## 10. Axis-coverage table (engineer fills at submission)

| AC | Axis | Test function |
|---|---|---|
| 1 | Observable | |
| 2 | Observable | |
| 3 | Input-changed | |
| 3 | Anti-behavior (returning state excludes hint) | |
| 4 | Anti-regression | |
| 5 | Anti-regression | |
| 6 | Anti-regression | |
| 7 | Integration | |

---

## 11. Non-goals

- Not adding a file-based or config-based persistent first-run marker.
- Not building a first-run walkthrough overlay.
- Not introducing analytics / telemetry.
- Not changing existing S1–S6 section ordering (FB-042 invariant).
- Not redesigning `jump to:` label (FB-105 shipped that).
- Not changing `[Tab] resume` behavior (already handles returning-within-session).
- Not coordinating with FB-152's S2 copy except for voice consistency (§8).
