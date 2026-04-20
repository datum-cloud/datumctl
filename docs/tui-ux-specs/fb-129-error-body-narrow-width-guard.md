# FB-129 — S3 error body narrow-width guard

**Feature brief:** FB-129 (`docs/tui-backlog.md` — search `### FB-129`)
**Status:** SPEC AUTHORED 2026-04-20 — ready for engineer routing
**Date:** 2026-04-20
**Depends on:** FB-102 ACCEPTED. FB-128 companion spec at `fb-128-error-hint-retry-verb.md` — implement together.

---

## 0. Design decision

**Select Option A:** drop parenthetical below contentW < 35. Threshold = 35.

**Why 35 (not 40, not a new breakpoint):**  
FB-082's existing tier contract uses 65 and 45 as column-drop breakpoints for data rows. The lower tier boundary implicitly covers the "barely usable" range. The FB-128 full-form copy (`activity unavailable ([r] to retry)`) is 35 visible chars — it fits exactly at contentW = 35. Guarding at 35 means the full form is always single-line at any width where it renders, and the guard aligns with FB-082's existing Tier 3 floor. No new breakpoint is introduced.

**Why Option A over B (compact `"unavail. [r]"`):**  
Abbreviations introduce a new truncation pattern not present elsewhere in S3/S4 body copy. Option A's narrow fallback is `"activity unavailable"` — 20 chars, identical to CRD-absent at all widths. It's a clean, consistent narrow-mode: at contentW < 35, both transient and CRD-absent look the same. The operator loses the retry hint at ultra-narrow widths, but contentW < 35 is a near-unusable terminal size where the entire TUI is degraded.

**Why Option A over C (status-bar PostHint):**  
Status-bar hint slots have collision risk with FB-079/FB-097 hint infrastructure. A 2-line conditional in the existing branch is lower risk.

**Why Option A over D (accept wrap):**  
A 2-line conditional is a cheap fix that closes the gap in FB-082's width contract.

---

## 1. Code change

**File:** `internal/tui/components/resourcetable.go:335-342`

**Before:**
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

**After** (incorporates FB-128 copy change):
```go
case m.activityFetchFailed:
    if m.activityCRDAbsent || contentW < 35 {
        body = muted.Render("activity unavailable")
    } else {
        body = muted.Render("activity unavailable") +
            " " +
            muted.Render("(") + accentBold.Render("[r]") + muted.Render(" to retry)")
    }
```

`contentW` is already the first parameter of `renderActivitySection(contentW int)` — no new field or setter needed.

**Note:** The `activityCRDAbsent || contentW < 35` condition means narrow-transient and CRD-absent both render `"activity unavailable"`. The visual distinction between transient and CRD-absent is intentionally sacrificed at widths where the full form would wrap — below 35 cols the TUI is near-unusable anyway.

---

## 2. Layout diagrams

### contentW = 80, transient error (full form — FB-128 copy)
```
Recent activity                           [4] full dashboard
──────────────────────────────────────────────────────────────
activity unavailable ([r] to retry)
```

### contentW = 34, transient error (narrow fallback)
```
Recent activity
──────────────────────────────────
activity unavailable
```

### CRD-absent, any width (unchanged)
```
Recent activity
──────────────────────────────────────────
activity unavailable
```

---

## 3. Acceptance criteria

| AC | Axis | Test target | Expected |
|----|------|-------------|---------|
| **AC1** | `[Observable]` | Narrow transient: parenthetical dropped below threshold | `activityFetchFailed=true`, `activityCRDAbsent=false`, `contentW=34`. `stripANSI(m.View())` contains `"activity unavailable"` and does NOT contain `"to retry"`. |
| **AC2** | `[Observable]` | Wide transient: full copy renders above threshold | Same flags, `contentW=35`. `stripANSI(m.View())` contains `"([r] to retry)"`. |
| **AC3** | `[Input-changed]` | Width transition 34→35 changes View() | Record `v1` at contentW=34, `v2` at contentW=35. `v1 != v2`. `v2` contains `"to retry"`; `v1` does not. |
| **AC4** | `[Anti-regression]` | CRD-absent unchanged at all widths | `activityCRDAbsent=true`, contentW=80. `stripANSI(m.View())` contains `"activity unavailable"` and does NOT contain `"to retry"`. |
| **AC5** | `[Anti-regression]` | FB-082 data-row tier contract unaffected | Wide model with `activityRows` set: tier column-drop thresholds (65/45) still apply. |
| **AC6** | `[Integration]` | Build + suite green | `go install ./...` clean. `go test ./internal/tui/...` exit 0. |

**Axis summary:** Observable × 2, Input-changed × 1, Anti-regression × 2, Integration × 1.

---

## 4. Non-goals

- Not changing CRD-absent copy or behavior at any width.
- Not introducing new breakpoint values — 35 aligns with existing FB-082 Tier 3 floor.
- Not routing the retry hint to the status bar.
- Not guarding at any width above 35 (contentW ≥ 35 → full form; that is the contract).
