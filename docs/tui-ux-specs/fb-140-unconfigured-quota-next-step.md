# FB-140 — Next-step affordance for unconfigured quota state

**Status:** PENDING UX-DESIGNER
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

## 2. Design question for UX

UX-designer picks one of:

- **A. Muted sub-line below the state message.** Example inline prose for the unconfigured branch: `"Run \`datumctl quota setup\` to configure."` (or similar — exact copy designer-pinned). Same treatment for the unauthorized branch with different wording (e.g., `"Contact your project admin for access."`).
- **B. Keybind hint on the row.** Add `[?] setup` or `[?] help` that opens the existing HelpOverlay scoped to a new "Quota setup" section. Relies on HelpOverlay as the canonical destination (per FB-065/FB-132 convention). Requires HelpOverlay content addition.
- **C. Muted sub-line with docs URL.** Static link prose: `"See docs.datum.net/quota/setup"` for unconfigured; different URL for unauthorized. Introduces first docs URL in the TUI rendering — sets precedent.
- **D. Do nothing for unconfigured; do something for unauthorized only.** On the grounds that unconfigured is a one-time setup concern (operator fixes once, doesn't re-encounter) while unauthorized may recur (access revoked mid-session, cross-project switching) and deserves the more prominent affordance.
- **E. A + B together.** Sub-line prose plus `[?]` keybind. Maximum discoverability; maximum vertical real estate cost in S2.
- **F. Do nothing.** Welcome S2 is a quick-glance status surface; remedy belongs in a future `datumctl config` wizard or a dedicated first-run onboarding surface. Current copy is sufficient to name the problem.

UX-preference signal: **A or B** is the minimum viable change. C sets docs-URL-in-TUI precedent that may want a broader audit. D treats the two branches asymmetrically — viable if the designer judges recurrence frequency. E adds noise for a P3 discoverability lift. F defers to a future surface.

**Flag to UX-designer:** the S2 region is already vertically tight on the welcome landing — budget for added sub-lines should be verified against narrow-width rendering (≤50 contentW branch at `resourcetable.go:291`, which currently drops to a different compact layout).

---

## 3. File locations

- **Render site:** `internal/tui/components/resourcetable.go`
  - L273 — unauthorized branch: `return leftHeader + "\n\n" + muted.Render("Platform health unavailable")`
  - L280 — unconfigured branch: `return leftHeader + "\n\n" + muted.Render("Quota service not configured")`
  - L291 — narrow-width branch (contentW < 50): separate compact layout; designer should specify narrow-width behavior explicitly
- **HelpOverlay content (if option B or E chosen):** likely `internal/tui/components/helpoverlay.go` — designer to confirm section anchor mechanism
- **Keybind dispatch (if option B or E chosen):** welcome-panel key handler in `internal/tui/model.go` — `[?]` is already bound globally for HelpOverlay; scoping to a quota-setup section is the new work

---

## 4. Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable]** In the designer's chosen option, when `!m.bucketConfigured`, `stripANSI(welcomePanel().View())` contains the designer-pinned next-step affordance (sub-line text, keybind hint, or both). Test: render with `bc == nil`; assert pinned substring present.
2. **[Observable]** (Conditional on designer's treatment of unauthorized branch) When `m.bucketErr != nil && m.bucketUnauthorized`, `stripANSI(welcomePanel().View())` contains the designer-pinned unauthorized-branch affordance OR asserts absence (option D's explicit scope). Test: render with forbidden error; assert per designer's choice.
3. **[Input-changed]** `!bucketConfigured` vs `bucketConfigured && populated` vs `bucketErr unauthorized` produce three visibly distinct View() outputs — new affordance does not collapse the three-way distinction FB-139 established. Test: render each of the three states, assert pairwise inequality of the stripped View() output.
4. **[Anti-regression]** FB-139 AC1–AC3 tests green unmodified: `TestFB139_AC1_Observable_NewCopy`, `TestFB139_AC2_InputChanged_OldPhraseAbsent`, `TestFB139_AC3_AntiRegression_UnauthorizedUnchanged`. Test: full FB-139 suite runs green.
5. **[Anti-regression]** FB-074 suite green: `TestFB074_AC1/AC2/AC12`. Test: full FB-074 suite runs green.
6. **[Anti-regression]** Narrow-width branch (`contentW < 50`) rendering unchanged OR designer-pinned compact variant explicitly asserted. Test: render with narrow width, assert per designer's pinned narrow behavior.
7. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

---

## 5. Non-goals

- Not wiring a `datumctl quota setup` CLI command (if option A or E references it, copy must match actual command OR designer pins placeholder prose that is forward-compatible).
- Not creating a new help-overlay section unless option B or E is chosen.
- Not auditing other "terminal state with no next-step" surfaces elsewhere in the TUI — separate audit brief if the pattern recurs.
- Not redesigning the S2 Platform health region's layout beyond adding the affordance.
- Not changing the unauthorized branch copy at L273 unless option A/C/E for the unauthorized branch requires it.
