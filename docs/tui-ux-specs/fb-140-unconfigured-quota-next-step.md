# FB-140 — Next-step affordance for unconfigured quota state

**Status:** PENDING ENGINEER
**Priority:** P3
**Brief source:** `docs/tui-backlog.md` — search `### FB-140`
**Dependencies:** FB-139 ACCEPTED

---

## 1. Problem statement

FB-139 shipped clean disambiguation between the unauthorized (`"Platform health unavailable"` at `resourcetable.go:273`) and unconfigured (`"Quota service not configured"` at `resourcetable.go:280`) branches of the welcome panel's S2 Platform health region. Persona eval confirmed the copy reads correctly for a DevOps operator.

Persona surfaced one P3 finding (out-of-scope for FB-139, filed forward): when the operator lands on the welcome screen and sees `"Quota service not configured"`, the TUI names the problem but offers no next-step affordance. The operator understands the failure mode but has no in-TUI pointer toward configuration — they fall back to `datumctl quota --help`, docs site navigation, or team-internal runbooks.

The parallel question applies to the unauthorized branch at L273: an operator reading `"Platform health unavailable"` during an access-denied event has no in-TUI signal on who to escalate to or what to check (IAM policy? bucket credential? project scope?). These are two distinct remedy paths (configure service vs. request access) and may warrant distinct affordances.

This brief is intentionally scope-limited to the S2 Platform health region. It does not audit other "terminal state with no next-step" surfaces elsewhere in the TUI.

---

## 2. Pinned design: Option A — symmetric muted sub-line

**Both branches get a muted sub-line below the state message. Different prose per branch. No new keybinds, no HelpOverlay changes.**

### Rationale for Option A

- Option B/E (keybind hint opening scoped HelpOverlay) rejected: `helpoverlay.go` has no section-anchor mechanism — it's a flat 4-column layout. Adding scoped-section-open would be substantial new capability for a P3 discoverability fix. Cost exceeds benefit.
- Option C (docs URL) rejected: sets first-docs-URL-in-TUI precedent that warrants a separate audit brief before adoption. Not appropriate for a P3 lift.
- Option D (asymmetric) considered but rejected: both branches benefit from a sub-line. Unconfigured operators may encounter this state multiple times while setting up environments (especially in CI or multi-project scenarios). Symmetric treatment is simpler and more consistent.
- Option F (defer) rejected: the copy already names the problem; a single sub-line that names the remedy is minimal and proportionate.
- Option E (A+B) rejected: same HelpOverlay concern as B; vertical cost unjustified for P3.

### Pinned copy

**Unconfigured branch** (`!m.bucketConfigured`, L280):

```
Quota service not configured
Ask your platform admin to enable quota.
```

Both lines rendered with `muted` style (same `lipgloss.NewStyle().Foreground(styles.Muted)`). Second line is the new addition.

Note: does not reference `datumctl quota setup` — that command does not exist in the current CLI. The sub-line is forward-compatible prose that doesn't depend on any specific command name.

**Unauthorized branch** (`m.bucketUnauthorized`, L273):

```
Platform health unavailable
Contact your project admin for access.
```

Same muted style for both lines. Second line is the new addition.

---

## 3. Narrow-width behavior

The unconfigured (L280) and unauthorized (L273) branches both return before the existing `contentW < 50` narrow check at L291. That narrow check applies only to the populated/healthy path and is irrelevant here.

For the new sub-lines, specify narrow suppression at the site of each early-return:

- **`contentW >= 40`** → render two-line output (state message + sub-line)
- **`contentW < 40`** → render current single-line output only (no sub-line); avoids wrapping artifacts at very narrow pane widths

This threshold is independent of the `wideEnough` (≥50) gate used for S3/S4/S5 regions.

---

## 4. Render site — exact changes

File: `internal/tui/components/resourcetable.go`

### Unauthorized branch (currently L273)

**Before:**
```go
if m.bucketUnauthorized {
    return leftHeader + "\n\n" + muted.Render("Platform health unavailable")
}
```

**After:**
```go
if m.bucketUnauthorized {
    line := muted.Render("Platform health unavailable")
    if contentW >= 40 {
        line += "\n" + muted.Render("Contact your project admin for access.")
    }
    return leftHeader + "\n\n" + line
}
```

### Unconfigured branch (currently L280)

**Before:**
```go
if !m.bucketConfigured {
    return leftHeader + "\n\n" + muted.Render("Quota service not configured")
}
```

**After:**
```go
if !m.bucketConfigured {
    line := muted.Render("Quota service not configured")
    if contentW >= 40 {
        line += "\n" + muted.Render("Ask your platform admin to enable quota.")
    }
    return leftHeader + "\n\n" + line
}
```

---

## 5. File locations

- **Render site:** `internal/tui/components/resourcetable.go`
  - L273 — unauthorized branch
  - L280 — unconfigured branch
  - L291 — narrow healthy-path branch (unchanged; not involved in this brief)
- **HelpOverlay:** `internal/tui/components/helpoverlay.go` — **no changes required**
- **Key handler:** `internal/tui/model.go` — **no changes required**

---

## 6. Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable]** When `!m.bucketConfigured` and `contentW >= 40`, `stripANSI(welcomePanel().View())` contains `"Ask your platform admin to enable quota."`. Test: render with `bc == nil` and sufficient width; assert pinned substring present.

2. **[Observable]** When `m.bucketErr != nil && m.bucketUnauthorized` and `contentW >= 40`, `stripANSI(welcomePanel().View())` contains `"Contact your project admin for access."`. Test: render with forbidden error and sufficient width; assert pinned substring present.

3. **[Observable — narrow suppression]** When `!m.bucketConfigured` and `contentW < 40`, `stripANSI(welcomePanel().View())` does NOT contain `"Ask your platform admin to enable quota."` (sub-line suppressed). Test: render with narrow width; assert substring absent.

4. **[Input-changed]** `!bucketConfigured` vs `bucketConfigured && populated` vs `bucketErr unauthorized` produce three visibly distinct `View()` outputs — new sub-lines do not collapse the three-way distinction FB-139 established. Test: render each of the three states at `contentW >= 40`; assert pairwise inequality of stripped View() output.

5. **[Anti-regression]** FB-139 AC1–AC3 tests green unmodified: `TestFB139_AC1_Observable_NewCopy`, `TestFB139_AC2_InputChanged_OldPhraseAbsent`, `TestFB139_AC3_AntiRegression_UnauthorizedUnchanged`. Test: full FB-139 suite runs green.

6. **[Anti-regression]** FB-074 suite green: `TestFB074_AC1/AC2/AC12`. Test: full FB-074 suite runs green.

7. **[Anti-regression]** Narrow healthy-path branch (contentW < 50, populated quota data) rendering unchanged. Test: render with `contentW = 45` and populated buckets; assert no sub-line affordance text appears (that branch is unaffected).

8. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

---

## 7. Non-goals

- Not wiring a `datumctl quota setup` CLI command.
- Not creating a new help-overlay section.
- Not auditing other "terminal state with no next-step" surfaces elsewhere in the TUI — separate audit brief if the pattern recurs.
- Not redesigning the S2 Platform health region's layout beyond the two sub-lines.
- Not changing the existing unauthorized or unconfigured primary copy (L273, L280 first-line text unchanged).
