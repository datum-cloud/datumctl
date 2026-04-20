# FB-128 — S3 activity error hint: add action verb

**Feature brief:** FB-128 (`docs/tui-backlog.md` — search `### FB-128`)
**Status:** SPEC AUTHORED 2026-04-20 — ready for engineer routing
**Date:** 2026-04-20
**Depends on:** FB-102 ACCEPTED. FB-129 companion spec at `fb-129-error-body-narrow-width-guard.md` — implement together.

---

## 0. Design decision

**Select Option B:** `activity unavailable ([r] to retry)` — 35 visible chars.

**Why B over A (`press [r] to retry`, 40 chars):**  
`[r]` in bracket notation already implies keypress by TUI convention — every key in S6, S4, and the help overlay uses `[key]` without a preceding "press." The word "press" is redundant. Removing it saves 5 chars, which matters for FB-129 coordination (see below).

**Why B over D (status quo):**  
The verb "to retry" makes the affordance self-contained. Operators encountering this screen for the first time should not need to cross-reference the events status bar or help overlay to understand what `[r]` does here.

**Why not C (em-dash):**  
`activity unavailable — [r] retry` drops the parenthetical, collapsing the visual distinction from CRD-absent (`activity unavailable` with no suffix). FB-102's visual contract — parenthetical = transient/retryable, no parenthetical = permanent — is worth preserving.

**FB-129 coordination:** Option B's 35-char full form fits exactly at contentW ≥ 35 — aligning with FB-082's existing Tier 3 boundary. Below 35 cols the narrow-width guard (FB-129) drops the parenthetical. No new breakpoint introduced.

---

## 1. Code change

**File:** `internal/tui/components/resourcetable.go:339-341`

**Before:**
```go
body = muted.Render("activity unavailable") +
    " " +
    muted.Render("(press ") + accentBold.Render("[r]") + muted.Render(")")
```

**After:**
```go
body = muted.Render("activity unavailable") +
    " " +
    muted.Render("(") + accentBold.Render("[r]") + muted.Render(" to retry)")
```

Width guard from FB-129 wraps this branch — see FB-129 spec for the outer `if contentW >= 35` gate.

---

## 2. Layout diagrams

### Transient error, contentW ≥ 35 (normal)
```
Recent activity                           [4] full dashboard
──────────────────────────────────────────────────────────────
activity unavailable ([r] to retry)
```

### Transient error, contentW < 35 (narrow — FB-129 path)
```
Recent activity
───────────────
activity unavailable
```

### CRD-absent error (all widths — unchanged)
```
Recent activity
───────────────────────────────────────────
activity unavailable
```

---

## 3. Acceptance criteria

| AC | Axis | Test target | Expected |
|----|------|-------------|---------|
| **AC1** | `[Observable]` | Transient error at normal width | `ResourceTableModel` with `activityFetchFailed=true`, `activityCRDAbsent=false`, `contentW=80`. `stripANSI(m.View())` contains `"([r] to retry)"`. |
| **AC2** | `[Observable]` | Old copy absent | Same model. `stripANSI(m.View())` does NOT contain `"(press [r])"`. |
| **AC3** | `[Anti-regression]` | CRD-absent has no parenthetical | `activityCRDAbsent=true`. `stripANSI(m.View())` contains `"activity unavailable"` and does NOT contain `"to retry"` or `"press"`. |
| **AC4** | `[Anti-regression]` | FB-102 tests updated | Existing FB-102 test assertions for `"(press [r])"` updated to `"([r] to retry)"`. All pass. |
| **AC5** | `[Integration]` | Build + suite green | `go install ./...` clean. `go test ./internal/tui/...` exit 0. |

**Axis summary:** Observable × 2, Anti-regression × 2, Integration × 1.

---

## 4. Non-goals

- Not changing CRD-absent copy (`"activity unavailable"` stays as-is).
- Not changing help overlay or status bar `[r] refresh` copy.
- Narrow-width fallback behavior is specified in FB-129 — implement alongside.
