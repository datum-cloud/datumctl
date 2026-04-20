# FB-152 — S2 platform-health section personality copy

**Status:** PENDING ENGINEER
**Priority:** P2
**Brief source:** `docs/tui-backlog.md` — search `### FB-152`
**Dependencies:** FB-105 ACCEPTED + PERSONA-EVAL-COMPLETE

---

## 1. Problem statement

The S2 "Platform health" section reads as an admin status board on the welcome surface.
When quota is healthy, the header row shows `"Platform health         ✓ All clear"`.
Generic. It doesn't carry the direct-confident-dry Datum voice established by FB-105.
FB-105 deferred S2 personality work — this brief ships it.

---

## 2. Question 1 — Which S2 render paths get personality?

### Decision: healthy path only. All other paths stay as-is.

| State | Decision | Rationale |
|---|---|---|
| **Healthy (all in bounds)** | **Personalize** | Strongest candidate: healthy state has no affordances to crowd out, and a warm signal is genuinely missing |
| Degraded (some at threshold) | Data-only, unchanged | Per-row warning glyphs already carry the signal; adding copy would duplicate it |
| Loading | Unchanged | Too ephemeral — spinner is the active signal |
| Error / transient error | Unchanged | FB-071/135/143/144/145 own this copy; don't overlap |
| Unconfigured | Unchanged | FB-140 owns next-step affordance; don't overlap |
| Zero governed types | Unchanged | "No governed resource types" message is already correct |

The healthy path is `summary.ConstrainedTypes == 0` in `renderPlatformHealthSection()`
at `resourcetable.go:310`. This is the only branch this brief touches.

---

## 3. Question 2 — What does "healthy" copy say?

### Chosen: `"✓ quota looks healthy"`

| Candidate | Decision |
|---|---|
| `"quota looks healthy · N buckets in bounds"` | Rejected — adding bucket count creates a width budget risk; the count is visible in the bucket rows below |
| **`"quota looks healthy"`** | **Accepted** — concise, honest ("looks" avoids "is"), Datum-voice |
| `"all quota in bounds"` | Rejected — "in bounds" is technical jargon; "quota looks healthy" is more readable |
| `"everything's in shape"` | Rejected — informality drift from the FB-105 register |
| `"your quota is healthy"` | Rejected — possessive "your" feels over-familiar; "is" is overconfident |

The `"looks"` qualifier is the FB-105 persona-validated register anchor (`"looks healthy"`
was accepted by persona in FB-105 for S4 copy). Keeping it here creates lexical
consistency across S2 and S4.

The leading `"✓"` glyph is preserved from the current `"✓ All clear"` — it's the visual
anchor operators scan for before reading the text.

Styled: `styles.Success` + Bold — identical to the current `"✓ All clear"` style. No
style-system change.

---

## 4. Question 3 — Where in S2 does the line render?

### Chosen: Option C — inline replace of the right-aligned status in the header row

The current healthy header row:
```
Platform health         ✓ All clear
```
Becomes:
```
Platform health         ✓ quota looks healthy
```

**Why Option C over Option B (sub-line below header):**
Option C has zero vertical-budget impact. Option B adds a line — at smaller terminal
heights S2 can already be tight. Option C is a single string replacement with no
structural change to the render logic.

**Why Option C over Option A (replace header):**
Option A loses `"Platform health"` as the section anchor, breaking screen-reader
semantics and any test that asserts the header substring. Option C keeps the header
intact.

Width check at `contentW = 50` (minimum wide-mode threshold):
- `"Platform health"` = 15 plain chars
- `"✓ quota looks healthy"` = 21 plain chars (styled Success+Bold)
- 15 + gap + 21 = 50 → gap = 14. Fits. ✓

Width check at `contentW = 80`:
- gap = 80 − 15 − 21 = 44. Comfortable. ✓

Narrow path (`contentW < 50`): renders the summary text variant (`"%d of %d governed
types ≥80%% allocated"`) — this branch is unchanged. ✓

---

## 4. Question 4 — Coexistence with FB-043/059/060/063 title-bar chrome

No collision. FB-043/059/060/063 all write into the `titleBar()` method of
`QuotaDashboardModel` (`quotadashboard.go`) — a completely separate component rendered
only when the QuotaDashboard pane is active. FB-152 writes into
`renderPlatformHealthSection()` inside `welcomePanel()` of `ResourceTableModel`
(`resourcetable.go`). These are different components, different methods, different panes.
The spec explicitly disclaims: **FB-152 does not touch the QuotaDashboard title bar or
any of its state-machine fields.**

---

## 5. Render site

**Primary change:** `internal/tui/components/resourcetable.go` —
`renderPlatformHealthSection()`, the healthy status line at approximately line 311.

Replace:
```go
statusLine = success.Render("✓ All clear")
```
With:
```go
statusLine = success.Render("✓ quota looks healthy")
```

**That is the only change.** No structural changes. No new fields. No new conditions.
No changes to `welcomePanel()`, `model.go`, key handlers, or any other file.

---

## 6. Anti-regression note: `"✓ All clear"` substring in S2

The current string `"✓ All clear"` may be asserted in `resourcetable_test.go` (FB-042
or related tests). The engineer must grep for `"✓ All clear"` and `"All clear"` in
`internal/tui/components/resourcetable_test.go` and `internal/tui/model_test.go`.

Any test asserting the S2 status line must update to `"quota looks healthy"`. This is
a coordinated AC update, not a regression. Test-engineer confirms the update.

Note: the S4 all-clear `"all clear · no issues detected"` (in `welcomePanel()`) is a
**different string** — it is NOT changed by FB-152. Only the S2 header status line
changes.

---

## 7. Interaction with FB-151

Both briefs share the Datum voice. On a healthy welcome panel, an operator sees:
- S2: `Platform health    ✓ quota looks healthy` (FB-152)  
- S4: `all clear · no issues detected` + first-session sub-line if applicable (FB-151)

S2 scopes to quota specifically; S4 scopes to overall platform absence of issues.
Complementary, not redundant. No copy collision. Independent ACs, independent
implementation.

---

## 8. Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable — healthy state]** With `summary.ConstrainedTypes == 0` and
   `contentW >= 50`: `stripANSI(m.renderPlatformHealthSection(80, false, false))`
   contains `"quota looks healthy"`. Test: construct with healthy bucket fixture;
   assert substring present.

2. **[Anti-regression — old copy absent in healthy state]** Same state:
   `stripANSI(...)` does NOT contain `"✓ All clear"` (in the S2 status position).
   Test: same fixture; assert old substring absent.

3. **[Input-changed — healthy → degraded]** State A: `ConstrainedTypes=0` →
   `"quota looks healthy"` present, `"need attention"` absent. State B:
   `ConstrainedTypes=1` → `"quota looks healthy"` absent, `"need attention"` present.
   Two View() outputs differ at the status substring.

4. **[Anti-regression — Platform health header preserved]** `stripANSI(...)` still
   contains `"Platform health"` in healthy state. Test: assert header anchor present.

5. **[Anti-regression — FB-042 S2 structure]** Section ordering, bucket rows, separator
   positions unchanged. Existing FB-042 tests green (with `"✓ All clear"` assertions
   updated per §6).

6. **[Anti-regression — error/loading/unconfigured paths unchanged]** Transient error,
   unauthorized, unconfigured, loading, zero-governed-types branches produce unchanged
   View() output. Test: existing tests for these paths green.

7. **[Anti-regression — S4 all-clear unchanged]** `"all clear · no issues detected"`
   (S4, different string) still present when S4 renders. Test: existing FB-105 S4
   tests green.

8. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

---

## 9. Axis-coverage table (engineer fills at submission)

| AC | Axis | Test function |
|---|---|---|
| 1 | Observable | |
| 2 | Anti-regression (old copy absent) | |
| 3 | Input-changed | |
| 3 | Anti-behavior (degraded excludes healthy copy) | |
| 4 | Anti-regression | |
| 5 | Anti-regression | |
| 6 | Anti-regression | |
| 7 | Anti-regression | |
| 8 | Integration | |

---

## 10. Non-goals

- Not changing any other S2 render path (error, loading, unconfigured, degraded).
- Not changing S2 bucket row rendering, percentages, or kind labels.
- Not changing S4 all-clear copy (`"all clear · no issues detected"`).
- Not touching the QuotaDashboard title-bar chrome (FB-043/059/060/063 own that).
- Not reordering S1–S6 sections (FB-042 invariant).
- Not changing FB-071 error co-location or FB-135 persistence behavior.
- Not changing FB-140 unconfigured-state affordance.
- Not changing FB-143/144/145 transient-error sub-line copy.
