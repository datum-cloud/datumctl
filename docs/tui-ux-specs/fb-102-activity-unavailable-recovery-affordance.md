# FB-102 — "activity unavailable" has no recovery affordance when `!isCRDAbsent`

**Status:** PENDING ENGINEER
**Priority:** P3
**Brief source:** `docs/tui-backlog.md` — search `### FB-102`
**Dependencies:** FB-082 ACCEPTED (activityFetchFailed state machine + S3 teaser copy), FB-076 ACCEPTED (`[r]` dispatch)
**Coordinates with:** FB-100 (stale-rows retained on refresh error — complementary surfaces, no bundling needed; see §2)

---

## 1. Problem statement

`resourcetable.go:304–305` renders `"activity unavailable"` in muted style whenever
`m.activityFetchFailed=true`. Two sub-cases merge into this one surface:

1. **CRD-absent** (`isCRDAbsent=true` at `model.go:464`): genuinely permanent — the activity
   CRD is not installed on this cluster. Retrying won't help; a retry hint would mislead.
2. **Transient error** (`isCRDAbsent=false`): network blip, API timeout, rate-limit. Pressing
   `[r]` will retry and may succeed. The operator has no on-screen signal this is recoverable.

Currently `ResourceTableModel` stores only `activityFetchFailed bool` — no field distinguishes
the two sub-cases. Both render identical copy. The gap: transient-error operators see a
permanent-feeling message with no affordance for the action that recovers them.

---

## 2. Designer decision

**Option D selected — compact parenthetical inline in S3 body, transient-only.**

### Options considered

| Option | Description | Decision |
|--------|-------------|----------|
| A | Inline `"activity unavailable  [r] to retry"` append | Rejected: longer copy (35 chars) with two-space separator is awkward; parenthetical style (D) is more terminal-conventional. |
| B | Second-line muted hint below the unavailable label | Rejected: two-line footprint expands S3 height budget; not worth it for a P3 error state that most operators won't see often. |
| C | Status-bar hint `"Activity unavailable — [r] to retry"` | Rejected: discoverability depends on operator scanning the status-bar, not S3; collision risk with FB-079/FB-097 quota hints; no inline indication on the surface that raised the error. |
| **D** | **Inline `"activity unavailable (press [r])"` on transient-only** | **Selected.** Compact; terminal-conventional parenthetical; stays within single-line S3 body (no height cost); well within minimum `contentW ≥ 50` gate (32 chars rendered text). CRD-absent keeps clean copy distinguished by absence of parenthetical. |
| E | Dismiss | Considered; P3 harm is concrete — `[r]` is the recovery gesture and the inline state renders without acknowledging it. Option D is cheap. |

### FB-100 coordination

FB-100 (stale-rows retained on refresh error) handles the **populated-rows** case: rows exist,
`[r]` fails, rows remain visible + optional status-bar hint `"Activity refresh failed —
showing cached data"`. FB-102 handles the **empty-rows** case: no rows, fetch failed,
`"activity unavailable"` body.

These are complementary, non-overlapping surfaces. No bundling is needed. The Option D
inline copy works for the empty-rows case regardless of whether FB-100 ships. If FB-100's
optional status-bar hint ships, the two hints co-exist without conflict (one in S3 body, one
in status-bar, different trigger conditions).

### Styling

`[r]` key label: accent+bold (consistent with `[3]`/`[4]` key labels in FB-088 and keybind
strip). Surrounding parenthetical `"(press "` and `")"`: muted.

---

## 3. Implementation

### 3a. `ResourceTableModel` — add CRD-absent flag

**File:** `internal/tui/components/resourcetable.go`

Add field alongside existing `activityFetchFailed`:
```go
activityFetchFailed  bool              // FB-082: true when last fetch returned an error
activityCRDAbsent    bool              // FB-102: true when error is CRD-absent (permanent)
```

Add setter:
```go
func (m *ResourceTableModel) SetActivityCRDAbsent(v bool) { m.activityCRDAbsent = v }
```

In `SetActivityRows()` (currently line ~805), clear alongside `activityFetchFailed`:
```go
func (m *ResourceTableModel) SetActivityRows(rows []data.ActivityRow) {
    m.activityFetchFailed = false // FB-082: successful data arrival clears error state
    m.activityCRDAbsent = false   // FB-102: clear CRD-absent flag on successful data
    // ... existing rows assignment
}
```

This ensures context switch (`SetActivityRows(nil)`) also resets `activityCRDAbsent` —
no separate context-switch wire-up needed.

### 3b. `renderActivitySection()` — gated recovery hint

**File:** `internal/tui/components/resourcetable.go`, `renderActivitySection()` (line ~289).

`accentBold` is already declared in this function scope. Use it for `[r]`.

**Current (`line 304–305`):**
```go
case m.activityFetchFailed:
    body = muted.Render("activity unavailable")
```

**New:**
```go
case m.activityFetchFailed:
    if m.activityCRDAbsent {
        body = muted.Render("activity unavailable")
    } else {
        body = muted.Render("activity unavailable") +
            " " +
            muted.Render("(press ") + accentBold.Render("[r]") + muted.Render(")")
    }
```

The combined rendered copy is 32 chars at the text level (ANSI-stripped). The S3 outer gate
(`contentW >= 50`) ensures this always fits in the available width.

### 3c. `model.go` — wire `SetActivityCRDAbsent` at the error handler

**File:** `internal/tui/model.go`, `case data.ProjectActivityErrorMsg:` (~line 462).

**Current:**
```go
case data.ProjectActivityErrorMsg:
    m.activityDashboard.SetLoading(false)
    isCRDAbsent := errors.Is(msg.Err, data.ErrActivityCRDAbsent) || errors.Is(msg.Err, data.ErrActivityCRDPartial)
    if isCRDAbsent {
        m.activityCRDAbsentThisSession = true
    }
    m.activityDashboard.SetLoadErr(msg.Err, msg.Unauthorized, isCRDAbsent)
    m.table.SetActivityLoading(false)
    m.table.SetActivityFetchFailed(true)
```

**New (add one line after `SetActivityFetchFailed`):**
```go
    m.table.SetActivityLoading(false)
    m.table.SetActivityFetchFailed(true)
    m.table.SetActivityCRDAbsent(isCRDAbsent) // FB-102: gate recovery hint on transient-only
```

No other wiring needed — `SetActivityRows()` clears the flag on success, and context switch
calls `SetActivityRows(nil)` which clears it automatically.

---

## 4. ASCII layouts

### S3 — transient error (no rows, `activityCRDAbsent=false`)

```
  Recent activity                              [4] full dashboard
  ──────────────────────────────────────────────────────────────
  activity unavailable (press [r])
```

(`[r]` renders in accent+bold; parenthetical in muted)

### S3 — CRD-absent (`activityCRDAbsent=true`) — UNCHANGED

```
  Recent activity                              [4] full dashboard
  ──────────────────────────────────────────────────────────────
  activity unavailable
```

### S3 — narrow width (`contentW = 50`, minimum gate)

```
  Recent activity         [4] full dashboard
  ──────────────────────────────────────────
  activity unavailable (press [r])
```

32-char body comfortably within 50-char `contentW`.

---

## 5. Acceptance criteria

| # | Axis | Criterion |
|---|------|-----------|
| AC1 | Observable | `activityFetchFailed=true`, `activityCRDAbsent=false`: `stripANSI(table.View())` contains `"activity unavailable"` AND `"(press"` AND `"[r]"`. Test: call `SetActivityFetchFailed(true)` + `SetActivityCRDAbsent(false)` on a table model; assert View() substrings. |
| AC2 | Observable | `activityFetchFailed=true`, `activityCRDAbsent=true`: View() contains `"activity unavailable"` AND does NOT contain `"(press"` or `"[r]"`. Test: `SetActivityFetchFailed(true)` + `SetActivityCRDAbsent(true)`; assert presence of unavailable copy + absence of parenthetical. |
| AC3 | Input-changed | Same `activityFetchFailed=true`, `activityCRDAbsent=false` vs `activityCRDAbsent=true` → different View() output. Test: render both states; assert View() with `crdAbsent=false` != View() with `crdAbsent=true`. |
| AC4 | Anti-behavior | `activityFetchFailed=false` (normal state): View() does NOT contain `"activity unavailable"` or `"(press [r])"`. Test: default model; assert substrings absent. |
| AC5 | Anti-behavior | After `SetActivityRows(someRows)` following a transient error: `activityCRDAbsent` and `activityFetchFailed` both clear → recovery hint absent, rows rendered. Test: set error state → call `SetActivityRows` with a row fixture → assert `"(press [r])"` absent + row content present. |
| AC6 | Anti-regression | FB-082 CRD-absent path: `TestFB082_*` tests for `"activity unavailable"` copy still pass (CRD-absent gating preserves existing copy). |
| AC7 | Anti-regression | FB-082 error-recovery path: `TestFB082_ErrorRecovery_SetActivityRows_ClearsFailedFlag` still passes. |
| AC8 | Anti-regression | FB-076 `[r]` dispatch tests green. |
| AC9 | Integration | `go install ./...` compiles; `go test ./internal/tui/...` green. |

---

## 6. Axis-coverage table

| Axis | Covered by | Notes |
|------|-----------|-------|
| Observable | AC1, AC2 | transient shows parenthetical; CRD-absent does not |
| Input-changed | AC3 | same fetch-failed state, different CRD-absent flag → different View() |
| Anti-behavior | AC4, AC5 | no hint in normal state; hint clears on successful rows |
| Anti-regression | AC6, AC7, AC8 | FB-082 CRD-absent copy + error recovery + FB-076 dispatch |
| Integration | AC9 | compile + full suite |

---

## 7. Non-goals

- Not changing `isCRDAbsent` detection logic in `model.go` — it's correct as shipped.
- Not bundling with FB-100 (stale-rows retained path is a distinct surface; each has its own fix).
- Not adding a status-bar hint for the transient case — inline copy in S3 body is more
  discoverable and avoids collision with quota hints (FB-079/FB-097).
- Not changing the `ActivityDashboardModel.crdAbsent` flag or dashboard error rendering
  (that surface has its own multi-state error block at `activitydashboard.go:190–197`).
- Not gating on `activityCRDAbsent` in the keybind strip's `!m.crdAbsent` check
  (`activitydashboard.go:314`) — that's the dashboard component's own guard, unchanged.
