# FB-143 — Welcome S2 transient platform-health error has no sub-line affordance

**Status:** PENDING ENGINEER
**Priority:** P3
**Brief source:** `docs/tui-backlog.md` — search `### FB-143`
**Dependencies:** FB-140 ACCEPTED

---

## 1. Problem statement

FB-140 added muted sub-lines to the unauthorized and unconfigured Platform health branches, making the transient non-unauthorized error branch at `resourcetable.go:279` the only static branch without a sub-line. After FB-140 the asymmetry is more visible: two branches name a remedy; the transient branch names a state and stops.

The transient branch has different operator semantics than the other two:
- Unauthorized → go-ask-someone (admin/IAM remedy).
- Unconfigured → go-ask-someone (platform-admin remedy).
- Transient → wait-it-out OR retry (self-remedy possible).

The brief is scope-limited to the single transient branch at L279. No other Platform health branches are touched.

---

## 2. Pinned design: Option A — `"Refresh to retry."`

**Single muted sub-line below `"Platform health temporarily unavailable"`, reading: `"Refresh to retry."`**

### Rationale

- The primary copy `"Platform health temporarily unavailable"` already carries the **wait signal** via "temporarily." Option B (`"This may resolve on its own."`) would redundantly restate what "temporarily" already conveys — no new information for the operator.
- Option A (`"Refresh to retry."`) adds **new information**: that the operator has an immediate action available (`[r]` refresh, already in the keybind strip). Two distinct signals — primary says "this is transient," sub-line says "here's what you can do right now."
- Option C (combined) over-explains: "temporarily" + "may resolve" + "refresh to retry" is three signals where two suffice. Vertical cost not justified for a P3 lift.
- Option D (silence) rejected: asymmetry is now plainly visible after FB-140 populated the neighboring branches.
- Option E (contact-someone variant) is semantically wrong for a transient error — there is no specific person to contact.
- No HelpOverlay changes (no section-anchor mechanism — same constraint that ruled out B/E in FB-140).
- No docs URL (sets first-docs-URL-in-TUI precedent — out of scope per FB-140 spec rationale).

---

## 3. Render site — exact change

File: `internal/tui/components/resourcetable.go`

**Before (L279):**
```go
return leftHeader + "\n\n" + muted.Render("Platform health temporarily unavailable")
```

**After:**
```go
line := muted.Render("Platform health temporarily unavailable")
if contentW >= 40 {
    line += "\n" + muted.Render("Refresh to retry.")
}
return leftHeader + "\n\n" + line
```

This is the only change required. Pattern is identical to the FB-140 pattern at L274 (unauthorized) and L285 (unconfigured).

---

## 4. Narrow-width behavior

Same `contentW >= 40` gate used by FB-140 at L274 and L285. Sub-line suppressed when `contentW < 40` to avoid wrapping artifacts at very narrow pane widths.

This is a carry-forward constraint — using a different threshold would create inconsistency across the three sub-line branches.

---

## 5. File locations

- **Render site:** `internal/tui/components/resourcetable.go` L279 — transient error branch
- **HelpOverlay:** `internal/tui/components/helpoverlay.go` — **no changes required**
- **Key handler:** `internal/tui/model.go` — **no changes required** (`[r]` refresh already exists)

---

## 6. Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable]** When `m.bucketErr != nil && !m.bucketUnauthorized` and `contentW >= 40`, `stripANSI(welcomePanel().View())` contains `"Refresh to retry."`. Test: render with non-unauthorized error and sufficient width; assert pinned substring present.

2. **[Observable — narrow suppression]** When `m.bucketErr != nil && !m.bucketUnauthorized` and `contentW < 40`, `stripANSI(welcomePanel().View())` does NOT contain `"Refresh to retry."`. Test: render with narrow width; assert substring absent.

3. **[Input-changed]** Unauthorized, unconfigured, and transient branches produce three visibly distinct `View()` outputs at `contentW >= 40` — the new sub-line does not collapse the three-way distinction. Test: render each of the three states; assert pairwise inequality of stripped View() output.

4. **[Anti-regression]** FB-140 AC1–AC4 green unmodified. Test: full FB-140 suite runs green.

5. **[Anti-regression]** FB-139 AC1–AC3 green unmodified. Test: full FB-139 suite runs green.

6. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

---

## 7. Non-goals

- Not redesigning the S2 Platform health region's layout beyond the new sub-line.
- Not changing the existing transient error primary copy (`"Platform health temporarily unavailable"` unchanged).
- Not auditing other "terminal state with no next-step" surfaces elsewhere in the TUI.
- Not wiring a new auto-retry mechanism — `[r]` refresh is the existing operator gesture; sub-line just surfaces it.
- Not adding a HelpOverlay section (no section-anchor mechanism; same constraint as FB-140).
