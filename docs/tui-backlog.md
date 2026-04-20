# TUI Feature Backlog

Product owner: TUI product-experience agent
Last updated: 2026-04-20 (**FB-080 ACCEPTED + FB-080 PERSONA-EVAL-COMPLETE + FB-079 ACCEPTED + FB-094 PERSONA-EVAL-COMPLETE + FB-096/097 specs delivered + FB-088 spec validated post-rebuild + FB-099 filed + FB-095/FB-096 scope amendments + FB-096 amended + FB-099 spec delivered + FB-063 PENDING TEST-ENGINEER + FB-079 PERSONA-EVAL-COMPLETE + FB-082 ACCEPTED + FB-082 PERSONA-EVAL-COMPLETE + FB-100/101/102/103 filed + FB-084 ACCEPTED + FB-102 spec delivered + FB-104/105 filed from user feedback + FB-084 PERSONA-EVAL-COMPLETE + FB-106 filed + FB-063 ACCEPTED + FB-083 spec delivered + FB-089/090 bundled PENDING TEST-ENGINEER + FB-064 ACCEPTED + FB-089/090 ACCEPTED + FB-083 PENDING TEST-ENGINEER + FB-092 PENDING TEST-ENGINEER + FB-063 PERSONA-EVAL-COMPLETE + FB-064 PERSONA-EVAL-COMPLETE + FB-104/105 specs delivered + FB-107/108/109 filed + FB-083 ACCEPTED + FB-089+090 PERSONA-EVAL-COMPLETE + FB-110 filed + FB-092 ACCEPTED + FB-093 PENDING TEST-ENGINEER + FB-108/109 specs delivered + FB-093 ACCEPTED + FB-083 PERSONA-EVAL-COMPLETE + FB-111 filed + FB-092 PERSONA-EVAL-COMPLETE + FB-095 PENDING TEST-ENGINEER + FB-093 PERSONA-EVAL-COMPLETE + FB-095 ACCEPTED (gate-check retro: false-blocker claim from test-engineer, product-experience direct-verified suite green) + FB-095 PERSONA-EVAL-COMPLETE + FB-107 PENDING TEST-ENGINEER + FB-111 PENDING TEST-ENGINEER + FB-107 ACCEPTED + FB-111 ACCEPTED + FB-107 PERSONA-EVAL-COMPLETE + FB-111 PERSONA-EVAL-COMPLETE + FB-096 PENDING TEST-ENGINEER + FB-096 REWORK (view-output assertion push-back — evidence-verified) + FB-099 PENDING TEST-ENGINEER (Option C strip-only, 8 wire-up sites, 6 ACs, component View() assertion pattern clean) + FB-099 ACCEPTED (gate-check clean, cross-feature FB-093 + FB-054 anti-regression suites green) + FB-096 RE-GATE PENDING TEST-ENGINEER (engineer rework: `statusBarNorm` helper, Width=120, View() substring; AC2 renamed `_HintField` → `_HintShown`) + FB-099 PERSONA-EVAL-COMPLETE (7 positives, 0 findings, considered-and-dismissed discipline clean) + FB-097 PENDING ENGINEER (routed per team-lead cadence while FB-096 re-gates) + FB-096 ACCEPTED (re-gate clean: 11 ACs PASS, FB-055/079/080/099 anti-regression green, statusBarNorm helper + AC2 rename confirmed) + FB-097 PENDING TEST-ENGINEER (engineer delivered Option A 1-site change, 8 ACs PASS with View() discipline) — **cancel-path trilogy CLOSED (FB-095/FB-096/FB-099)**; team rebuild triggered per cadence + FB-104 PENDING TEST-ENGINEER (engineer S7 hoveredType section, 9 ACs with brief-AC-indexed table; flagged: AC2 truncation-overflow coverage gap, AC5 state-transition pattern, FB-082/083 named anti-regression) + FB-096 PERSONA-EVAL-COMPLETE (clean: 0 P1/P2, 2 P3 dismissed — cosmetic dead code + intentional design) + FB-097 ACCEPTED (test-engineer gate-check clean; authored `TestFB097_BriefAC4_ConfirmClearsReadyPrompt` to fill engineer's AC-indexing gap; all 9 ACs PASS with statusBarNorm View() discipline) + FB-105 PENDING TEST-ENGINEER (three additive changes: S1 orientation hint, S4 all-clear flavor, quick-jump prefix "Quick jump:" → "jump to:"; 11 ACs PASS with View() discipline; brief-AC-indexed table clean) + FB-072 PENDING TEST-ENGINEER (Option A `lastEntryViaQuickJump` flag; 5 ACs PASS incl. AC2 state-transition + AC3 FB-041 anti-regression named; FB-042 round-trip test updated to new single-Esc behavior) + FB-100 PENDING TEST-ENGINEER (rows-empty gate on SetActivityFetchFailed; new ActivityRowCount() getter; 7 ACs PASS incl. AC5 named FB-082 anti-regression tests + AC6 FB-076 `[r]` dispatch tests green) + FB-101 PENDING TEST-ENGINEER (one-line `max(1, min(16, contentW-22))` defensive fix; 5 ACs PASS incl. AC1/AC2 panic-recovery tests + AC3 truncation-unchanged at Tier 3 boundary contentW=44)** — FB-079 shipped Option D copy with 3 new tests, [Input-changed] row N/A accepted with brief-grounded rationale (one-site copy swap, AC1 vs AC2 already pair the two hint phases). FB-094 persona delivered 0 P1 + 2 P2 + 3 P3; all 5 findings triaged; **0 new briefs filed** — findings #1–#4 validate FB-088's origin-label thesis (priority elevated P3 → P2); finding #5 (help overlay stale) verified non-issue. FB-096 spec Option B (both cancel paths post acknowledgment, `"Quota dashboard cancelled"`) + FB-097 spec Option A (persistent ready-prompt, 1-char change at `model.go:401–402`) delivered; **FB-098 DISMISSED** by ux-designer (FB-079+FB-097 jointly close the uncertainty window). **FB-124 ACCEPTED + FB-124 PERSONA-EVAL-COMPLETE + FB-125/126/127 filed from FB-124 persona + FB-125/126 combined spec delivered by ux-designer + FB-127 WONTFIX + FB-102 ACCEPTED (commit `b476a23`, transient vs CRD-absent distinction shipped) + FB-102 PERSONA-EVAL-COMPLETE (1 P2 + 3 P3 findings; P2-1 and P3-2 fold into FB-103 which is elevated P3 → P2 with AC7-9 added for error/CRD-absent coverage; P3-3 filed as FB-128 (copy verb); P3-4 filed as FB-129 (narrow-width guard)) + FB-128 spec delivered (Option B `([r] to retry)`, 35 chars) + FB-129 spec delivered (Option A, threshold 35 aligned with FB-082 Tier 3) + FB-128/FB-129 PENDING ENGINEER as a combined 4-line change + FB-125 ACCEPTED (commit `de407a5`) + FB-126 RESOLVED (via FB-125 noun-phrase literal, persona grep-verified cross-surface consistency) + FB-125 PERSONA-EVAL-COMPLETE (1 P3 dismissed as persona-acknowledged trade-off without evidence of teaching failure; 0 new briefs) + FB-103 ACCEPTED (commit `4c9dab1`, one-site `ActivityRowCount()==0` gate, 9 tests + Integration AC) + FB-103 PERSONA-EVAL-COMPLETE (0 P1/P2, 1 P3-1 filed as FB-130 render-gate-misses-empty-but-loaded narrower-site) + FB-130 filed (P3, engineer-direct, 1-char render gate widen `activityRows == nil` → `len(activityRows) == 0` at `resourcetable.go:333`) + FB-128 ACCEPTED + FB-129 ACCEPTED (combined 4-line change at `resourcetable.go:335-342`, Option B copy + Option A threshold=35 width guard; 3 FB-128 tests + 5 FB-129 tests + TestFB102_AC1 cross-feature update all PASS; submitter-produced axis-coverage tables pre-review verified) + FB-128 PERSONA-EVAL-COMPLETE + FB-129 PERSONA-EVAL-COMPLETE (bundled eval: copy confirmed correct matching `[4] full dashboard` key-first pattern, `"retry"` verb precise for error recovery joining delete-dialog + describe consistency; 35-char boundary math verified exact; 1 P3-1 DISMISSED as considered-and-dismissed per FB-129 §0 rationale; 1 P3-2 filed as FB-131) + FB-131 filed (P3, help overlay `[r]  refresh` vs error-state `[r] retry` verb discordance; pre-existing widened by FB-128; cross-surface copy alignment, PENDING UX-DESIGNER) + FB-131 WONTFIX (ux-designer Option C: two-verb convention pre-exists FB-128 in `deleteconfirmation.go:166/172`; `refresh` = normal, `retry` = error-state; Options A/B/D rejected on layout/coupling/regression grounds; spec §2 documents convention for future designers) + FB-130 gate-check HELD (engineer's tests have backwards setup order — `SetActivityRows` clears `activityLoading`; tests never reach fix-target state `loading=true + rows=[]`; rework + revert-re-run evidence requested) + FB-130 ACCEPTED (rework clean: setup order corrected to `SetActivityRows([])` → `SetActivityLoading(true)` reaching fix-target state; 4 FB-130 tests + AC5 reuse all PASS; AC1 + AC2 analytically proven to fail under reverted `== nil` gate; full suite + `go install` clean). **Next:** FB-130 user-persona eval; no other briefs in flight.)

**Active queue:** FB-073 *(quick-jump fires from NavPane, P2, spec-ready Option A NavPane gate)* → FB-059 *(gap-guard threshold, P3)* → FB-060 *(failed-refresh signal)* → FB-065 *(copy mismatch, P3)* → FB-071 *(BucketsErrorMsg propagation, P3)* → FB-074 *(S2 unconfigured-quota copy, P3, engineer-direct)* → FB-075 *(S5 attention-kind separator, P3, spec-ready Option A blank-row)* → FB-085 *(P3, title bar [unavailable]+spinner contradiction, spec-ready merged with FB-086 Option A)* → FB-086 *(P3, [E] silent-block on double-failure, spec-ready merged with FB-085 Option A)* → FB-088 *(**P2** elevated 2026-04-20 from FB-094 persona 2 P2 findings, origin-label affordance, impl-block LIFTED — FB-094 ACCEPTED, spec-ready Option A, PENDING ENGINEER)* → **FB-102** *(P3, "activity unavailable" recovery affordance, FB-082 persona P3-2, **ACCEPTED 2026-04-20** commit `b476a23` on `feat/console`, **PERSONA-EVAL-COMPLETE** — 1 P2 folded into FB-103, 3 P3 findings triaged as FB-103/128/129)* → **FB-103** *(**P2** elevated 2026-04-20 from FB-102 persona P2-1; `[r]` refresh path doesn't set activityLoading=true → no in-flight signal on empty-rows OR error-state refresh; engineer-direct, FB-082 persona P3-3 + FB-102 persona P2-1/P3-2, **ACCEPTED 2026-04-20** commit `4c9dab1` on `feat/console`, **PERSONA-EVAL-COMPLETE** — 0 P1/P2, 1 P3-1 filed as FB-130 narrower site)* → **FB-130** *(P3, spinner render gate `activityLoading && activityRows == nil` at `resourcetable.go:333` misses empty-but-loaded case (project with genuinely 0 activity rows) — FB-103 persona P3-1, engineer-direct 1-char fix widening to `len(activityRows) == 0`, **ACCEPTED 2026-04-20** — PERSONA-EVAL PENDING)* → **FB-106** *(P3, placeholder action row `[r] retry describe` overflows at contentW < 40 — narrow-width threshold gate Option A, engineer-direct, FB-084 persona P3-2, **PENDING ENGINEER**)* → **FB-108** *(P3, ticker-driven quota refresh intentionally silent post-FB-063 — spec Option B shipped, 2-line comment doc-only at `model.go:547`, no behavior change, **PENDING ENGINEER**)* → **FB-109** *(P3, `"quota dashboard"` top vs `"full dashboard"` bottom copy alignment — Option A clean variant `"── [3] quota dashboard ──"` drops "full" + "Quota" prefix, 2-line `buildQuotaSectionHeader()` change + prefixWidth=24 constant, **PENDING ENGINEER**)* → **FB-110** *(P3, help overlay "resume cached" sub-label doesn't match S1 "(cached)" parenthetical idiom after FB-089 shipped — engineer-direct 1-line fix at `helpoverlay.go:29`, FB-089+090 persona P3-1, **PENDING ENGINEER**)* → FB-038 → FB-039 → FB-040 → FB-026 → FB-025 → *(deferred: FB-027–034)* → FB-020 → FB-021 → FB-023 → *(deferred: FB-007/008/009)*

*Removed from queue 2026-04-20: **FB-107 (ACCEPTED 2026-04-20** — P2 engineer-direct three-site quota refresh state-cleanup (Site A `BucketsErrorMsg`/`LoadErrorMsg` drop `activePane` gate; Site B `ContextSwitchedMsg` adds `SetRefreshing(false)`; new `IsRefreshing()` + `HasBuckets()` accessors on `QuotaDashboardModel`); 5 FB-107 tests AC1–AC5 PASS; anti-regression `TestFB063_*` 7 tests + `TestFB082_*` / `TestFB083_*` 26 tests executed directly; full suite green; `bucketErr` vs `statusBar.Err` path distinction correctly flagged by engineer + verified by test-engineer; user-persona dispatch pending); **FB-111 (ACCEPTED 2026-04-20** — P2 engineer-direct one-line render-layer gate update at `resourcetable.go:310` adding `&& !m.activityFetchFailed` to FB-083 rows gate; 3 FB-111 tests AC1–AC3 PASS with Observable ACs using `stripANSI(m.View())` substring checks; AC4 covered by existing `TestFB083_AC6_AntiBehavior_SuppressedHint_4KeyStillNavigates`; anti-regression `TestFB083_*` 6 tests + `TestFB082_ErrorRecovery_*` executed directly; full suite green; defense-in-depth with pending FB-100 state-layer fix preserved; user-persona dispatch pending); **FB-095 (ACCEPTED 2026-04-20** — P3 engineer-direct two-site cancel-path stash clear (nav-cancel + second-press cancel); 4 FB-095 tests + cross-feature `TestFB080_AC3` migration (necessary-consequence); `go install ./...` + full `internal/tui` suite + FB-078/FB-080/FB-087 anti-regression all green; test-engineer's initial compile-failure blocker claim not reproducible — product-experience ran gate-check directly, verdict PASS on all counts; user-persona dispatch pending); FB-093 (ACCEPTED 2026-04-20** — Option A + Option B single `renderKeybindStrip()` rewrite shipped; table branch gains `3 quota` + `4 activity` between `/ filter` and `c ctx`; dashboard branch computes `hasCachedTable := m.forceDashboard && m.typeName != ""` and conditionally drops Tab entry when cached-state hint is visible in S1 (band owns Tab copy); 6 FB-093 brief-AC-indexed tests (AC1–AC6) using `renderKeybindStrip()` directly to avoid false-matching `[Tab]` from S1 band; AC4 uses `x delete` absence as table-branch marker; test-engineer gate-check ran anti-regression suites AC7/AC8/AC9 directly rather than hand-waving (FB-054 + FB-056 + FB-041 all green); pre-existing test failures TestFB093_AC2/AC4/AC5 + TestFB089_AC4 all now PASS confirming FB-093 is the feature they were ahead of; cross-feature `TestFB089_AC4_AntiRegression_S6Strip_TabNextPaneUnchanged` update reviewed per `feedback_scope_creep_bonus_bugs` necessary-consequence pattern (FB-089's semantic invariant of cached-state Tab affordance preserved; assertion migrated S6 → S1 as mandated by FB-093 Sub-problem 2 spec); user-persona dispatch pending per cadence rule); FB-092 (ACCEPTED 2026-04-20** — Option A one-line copy swap at `model.go:1521` → `"Returned to welcome panel"` drops CTA + "dashboard" term; 6 `TestFB055_*` test updates with `strings.Fields` normalization for lipgloss narrow-width wrapping; 9 brief-AC-indexed axes green (Observable x4, Input-changed HintClearMsg, Anti-behavior x2, Anti-regression x2 FB-054+FB-041, Integration); pre-existing failures in `components/` (TestFB093_* + TestFB089_AC4) flagged as unrelated to model.go one-liner; user-persona dispatch pending); FB-083 (ACCEPTED 2026-04-20 — Option C rows-only hint suppression shipped; `renderActivitySection()` gates on `len(m.activityRows) > 0`, `lipgloss.Width("")==0` absorbs gap; test-engineer added `TestFB083_AC6_AntiBehavior_SuppressedHint_4KeyStillNavigates` to close AC6 gap, proving suppression is display-only; 9 brief-AC-indexed tests total covering all empty/loading/fetchFailed/populated body states + Input-changed pair + `[4]` navigation anti-behavior + FB-082/FB-067 anti-regression chains; user-persona dispatch pending); FB-055 (ACCEPTED 2026-04-20); FB-056 (ACCEPTED 2026-04-20); FB-057 (ACCEPTED 2026-04-20); FB-063 (ACCEPTED 2026-04-20 — Option A SetRefreshing shipped; 7 brief-AC-indexed tests green including AC3 Input-changed pair for refreshing→updated-ago transition; `buildMainContent()` used for height-sensitive AC4/AC6 per existing precedent, `m.View()` used for AC1/AC2/AC3 end-to-end); **FB-064 (ACCEPTED 2026-04-20** — Option B prepend hint shipped via `buildQuotaTopHint()` + prepend in `buildDetailContent()`, 2 sites model.go only; 7 brief-AC-indexed tests incl. `TestFB064_InputChanged_YamlModeToggle_TopHintChanges` Input-changed pair; AC3/AC4/AC5 re-tagged Anti-behavior per pre-submission guidance; `buildDetailContent()` assertions consistent with existing FB-044 precedent + FB-063 buildMainContent convention; `go install` clean + suite green); **FB-089 (ACCEPTED 2026-04-20** — Option A Tab hint copy swap bundled with FB-090; 5 brief-AC-indexed tests covering 3 width tiers + AC4 re-tagged Anti-regression per pre-submission feedback + AC5 Input-changed `typeName=""` vs `"backends"` View() diff; FB-054 AC1 copy-assertion audit documented, behavioral assertion preserved); **FB-090 (ACCEPTED 2026-04-20** — Option A `quickJumpLabel()` helper bundled with FB-089; 7 tests + 4-subtest AC4 block (helper unit + View()-level pair addresses gate flag for user-visible Input-changed); AC8 combined-result test proves FB-089+FB-090 compose correctly (`"resume dns (cached)"` for `dnsrecordsets`); all Observable assertions View()-level); FB-078 (ACCEPTED 2026-04-20 — PERSONA-EVAL-COMPLETE); FB-079 (ACCEPTED 2026-04-20 + PERSONA-EVAL-COMPLETE 2026-04-20 — 0 P1/P2/P3 new briefs, P3-1 separator inconsistency DISMISSED as defensible tonal distinction, P3-2 "when ready" over-instruction already tracked in FB-099 upgrade path, P3-3 positive cross-feature copy coordination finding); **FB-084 (ACCEPTED 2026-04-20** — Options A+D shipped at `model.go:1950/1976`, `placeholderActionRow(errMode, retryable)` signature gates `[r]` on severity, copy `[r] retry describe` when shown; 6 tests green; **scope-creep process flag logged** — engineer bundled cross-feature test-assertion modifications into FB-064 impl (View() → buildDetailContent() narrowing for AC1/AC4/AC5); end-state internally consistent with submitted axis-coverage table, no working behavior ripped out, accepted with rule-reinforcement note to engineer; user-persona dispatch pending); **FB-082 (ACCEPTED 2026-04-20** — Option B activity-unavailable + 3-tier width truncation + `activityFetchFailed` state field; 15 new tests + 2 existing anti-regression anchors green, `go install ./...` clean; AC4 dual-duty anti-regression+stuck-flag-anti-behavior; bonus state-priority + empty-rows-clears component tests in file); **FB-080 (ACCEPTED 2026-04-20** — Option C second-press cancel shipped; `!m.pendingQuotaOpen` stash guard at `model.go:1165` + else-branch cancel at `model.go:1184` clears `pendingQuotaOpen` + hint; 6 new brief-AC-indexed tests + updated FB-047 regression; [Input-changed] pair + FB-095 cross-ref stash-preservation test both present; all `stripANSIModel(appM.View())` observable assertions; `go install ./...` clean); FB-087 (ACCEPTED 2026-04-20); FB-091 (DISSOLVED 2026-04-20); **FB-094 (ACCEPTED 2026-04-20 + PERSONA-EVAL-COMPLETE 2026-04-20** — P1 HOTFIX, Option A dashboard-as-origin guard; unblocks FB-088 impl + FB-095 engineer-direct fix; **0 new briefs from persona — findings all fold into FB-088 priority elevation**); **FB-098 (DISMISSED 2026-04-20** — FB-079 persistent loading-hint + FB-097 Option A persistent ready-prompt jointly resolve the uncertainty window; refile with evidence if confusion recurs). Added to queue 2026-04-20: FB-090 PENDING ENGINEER; FB-093 PENDING UX-DESIGNER; FB-095 P3 PENDING ENGINEER; **FB-096 P2 PENDING ENGINEER** (spec Option B delivered — both Esc-cancel + nav-cancel paths post acknowledgment); **FB-097 P2 PENDING ENGINEER** (spec Option A delivered — persistent ready-prompt). Priority changes 2026-04-20: **FB-088 P3 → P2** (FB-094 persona 2 P2 findings converge on its origin-label thesis; impl-block LIFTED with FB-094 acceptance).*

**Recent decisions:** 2026-04-20 **FB-092 ACCEPTED + FB-093 PENDING TEST-ENGINEER + FB-108/109 specs delivered.** FB-092 test-engineer gate-check accepted: one-line copy swap at `model.go:1521` with 6 `TestFB055_*` test updates; `strings.Fields` normalization warranted for lipgloss narrow-width wrapping (25-char new vs 38-char old copy); normalization is assertion-scoped, not model-scoped. All 9 brief-AC-indexed axes green; AC4 Input-changed covers `HintClearMsg` decay transition. Pre-existing failures in `components/` (TestFB093_* written ahead of unshipped feature + TestFB089_AC4 regression) flagged as unrelated to this model.go one-liner — confirmed post-FB-093 submit (engineer's FB-093 submission resolves the TestFB089_AC4 concurrence by intentionally updating that test). User-persona dispatch pending for FB-092 per cadence rule. **FB-093 PENDING TEST-ENGINEER** — engineer shipped Option A + Option B single-function rewrite of `renderKeybindStrip()`: table branch gains `pair("3", "quota")` + `pair("4", "activity")` between `/ filter` and `c ctx`; dashboard branch computes `hasCachedTable := m.forceDashboard && m.typeName != ""` and conditionally appends Tab only when `!hasCachedTable`. 6 new FB-093 tests (AC1–AC6) using `renderKeybindStrip()` directly (not `m.View()`) to avoid false-matching `[Tab]` from the S1 band. AC4 uses `x delete` absence as the table-branch distinguishing marker (dashboard branch always had 3/4, so that's the correct observable Input-changed pair for dashboard→table transition). **Cross-feature test update reviewed — NOT scope creep:** engineer flagged `TestFB089_AC4_AntiRegression_S6Strip_TabNextPaneUnchanged` update in submission message (per `feedback_scope_creep_bonus_bugs` proactive-flagging discipline); reviewed per the "necessary consequence" clause — FB-089 AC4 was asserting the pre-FB-093 S6-Tab invariant (`"Tab next pane"` present when `forceDashboard=true + typeName="backends"`), and FB-093 Sub-problem 2 brief *explicitly changes* that S6 behavior. The cached-state Tab affordance invariant FB-089 was really protecting (Tab-affordance exists for cached state) is preserved by the updated assertion (S1 band's `"resume backends"` still fires). This is the same pattern as FB-084 bundling FB-064 test-assertion narrowing — invariant semantically preserved, literal assertion updated for new surface. Accepted. Routed to test-engineer for gate verification. **FB-108 PENDING ENGINEER** — ux-designer Option B: silent ticker is intentional; 2-line comment at `model.go:547` explaining the principle (`SetRefreshing(true)` = operator-initiated only; ambient cadence is served by `"updated X ago"` timestamp). No code change; no new tests. Answers the persona-flagged ambiguity (intent vs oversight) with explicit intent codification. **FB-109 PENDING ENGINEER** — ux-designer Option A clean variant: bottom separator becomes `"── [3] quota dashboard ──"` (drops `"full"` qualifier + leading `"Quota "` prefix); exact vocabulary match with FB-064 top hint at `model.go:1919`; 2-line change in `buildQuotaSectionHeader()` at `model.go:1942` + `const prefixWidth = 24` (was 29). Ux-designer designer-call queue now empty. 2026-04-20 **FB-083 ACCEPTED + FB-089+090 PERSONA-EVAL-COMPLETE + FB-110 filed.** FB-083 test-engineer gate-check accepted: engineer's 5 tests covered AC1–AC5; test-engineer identified AC6 Anti-behavior gap (engineer's original hand-wave "existing tests green" for `[4]`-still-navigates invariant was insufficient) and added explicit `TestFB083_AC6_AntiBehavior_SuppressedHint_4KeyStillNavigates` in `model_test.go` — sets up nil activityRows with hint-absent precondition via `stripANSIModel(m.table.View())` NOT containing `"[4] full dashboard"`, presses `[4]`, asserts `appM.activePane == ActivityDashboardPane`. Proves hint suppression is display-only; key handler unaffected. All 9 brief-AC-indexed axes covered (Observable x4, Input-changed, Anti-behavior, Anti-regression x2, Integration). `go install ./...` + full suite green. User-persona dispatch pending per cadence rule. Strong submitter-owned table discipline from test-engineer — caught Anti-behavior gap without being prompted. **FB-089+090 PERSONA-EVAL-COMPLETE** — user-persona delivered 0 P1 + 0 P2 + 3 P3, all code-verified against `resourcetable.go:33–57/543–581/396–408/644–662` + `helpoverlay.go:25–31`. Positive findings (4): vocabulary consistency S1↔S4 end-to-end (same `quickJumpLabel` source); non-listed type fallback acceptable (identity pass-through); mental-model contradiction substantially closed via generic→specific layering (S6 generic "next pane" vs S1 state-specific "resume X (cached)"); three-tier truncation graceful at narrow widths. Triage disposition: **1 new brief filed (FB-110), 2 dismissals**. **P3-1** (help overlay `"resume cached"` sub-label at `helpoverlay.go:29` doesn't match S1 `"(cached)"` parenthetical idiom — FB-091 was DISSOLVED with FB-089 absorbing the `(cached)` pattern; help overlay wasn't updated when FB-089 shipped) → **FB-110 P3 engineer-direct** (1-line sub-label change; candidate replacements: `"resume (cached)"` parenthetical-aligned or dropping sub-label entirely given S1 already carries the state-specific qualifier). Concrete cross-surface consistency gap introduced by FB-089 rework, clear surface fix. **P3-2** (`(cached)` drops at medium widths for long unlisted type names like `"certificaterequests"` at 19 chars) **DISMISSED** — persona explicitly frames as "intentional three-tier design decision" and "worth noting as a data point for any future unlisted-type audit"; no user evidence of confusion; three-tier truncation is a deliberate width-survivability contract; same precedent as FB-063 P3-3 (informational width-safety) and FB-064 P3-3 (pre-existing narrow-width). Refile with operator reports if surface. **P3-3** (residual S1 `"resume X"` vs S6 `"next pane"` verb mismatch) **DISMISSED** — persona self-limits: "it doesn't contradict, but it requires one more mental step than pure consistency"; "substantially better than before"; "synthesis step is invisible" is the strongest claim but has no concrete operator-confusion evidence; persona itself calls it "contextual layering, not contradiction." Unifying the verbs would collapse the intentional generic→state-specific layering that FB-089 just validated. Refile with first-time-operator evidence if confusion surfaces. **Net:** 1 new P3 brief (FB-110), 2 P3 dismissals. FB-089 + FB-090 status blocks updated to PERSONA-EVAL-COMPLETE. 2026-04-20 **FB-089/090 bundled ACCEPTED + FB-083 PENDING TEST-ENGINEER.** FB-089 + FB-090 test-engineer submission accepted: all pre-submission gate flags I raised in the routing message addressed. (1) FB-089 AC4 correctly re-tagged from Observable → Anti-regression (S6 strip invariant check, not FB-089 observable) — test-engineer confirmed in submission table. (2) FB-090 AC4 dual-coverage: original helper-unit test (4 subtests for dnsrecordsets/dnszones/backends/certificates) + new `TestFB090_AC4_InputChanged_ViewLevel_LabelDiffers` (pair A `dnsrecordsets→"resume dns"` vs pair B `backends→"resume backends"`, same fixture, typeName varies, View() differs) — user-visible consequence anchored. (3) FB-054 AC1 copy-assertion audit: test-engineer explicitly documented "literal string updated; behavioral assertion (hint present ↔ cache active) unchanged" — meaningful coverage preserved, not vacuous passing. (4) Combined-result test `TestFB090_AC8_AntiRegression_CombinedResult_DNSCached` proves FB-089 copy + FB-090 label compose correctly for DNS case (`typeName="dnsrecordsets"` renders `"resume dns (cached)"`) — bundled-impl verification. All Observable assertions View()-level. `go install ./...` clean + full suite green. Bundled per Option 2 coordination (shared `quickJumpLabel()` helper); bundling was the right call as evidenced by the need for AC8 composition test. User-persona dispatch pending per cadence rule (covers bundled ship with FB-089 user-problem front-and-center). **FB-083 PENDING TEST-ENGINEER** — engineer shipped Option C rows-only conditional per spec: `renderActivitySection()` hint made conditional on `len(m.activityRows) > 0`; `lipgloss.Width("")==0` absorbs gap cleanly. Engineer produced 5 brief-AC-indexed tests in the FB-083 block covering all four S3 body states (empty/loading/fetchFailed/populated) + Input-changed pair. Routed to test-engineer for gate verification per standard pipeline (engineer's axis table is submitter-owned per `feedback_axis_coverage_table_ownership`; test-engineer will re-audit against brief ACs and run full suite). 2026-04-20 **FB-064 ACCEPTED + engineer next-work re-routed to FB-092 per team-lead directive.** FB-064 test-engineer submission accepted: 7 brief-AC-indexed tests green, `go install ./...` + `go test ./internal/tui/... -count=1` clean, Option B prepend hint ships as specified. `newFB064DetailModel()` helper uses 80-line describe + matching compute.example.io/cpus bucket (innerW=77 ≥ 30). AC3/AC4/AC5 re-tagged as Anti-behavior (feature-guard axis, not regression-guard) per pre-submission guidance — correct disambiguation. [Input-changed] covered by `TestFB064_InputChanged_YamlModeToggle_TopHintChanges` — same fixture, yamlMode toggles, `buildDetailContent()` output differs (hint present vs absent). AC1 Observable dual-presence check (`"[3]"` AND `"quota dashboard"`) + AC2 Anti-regression proves additive coexistence with bottom separator (`"── Quota"` + `"full dashboard"` still present). Assertions use `m.buildDetailContent()` directly, consistent with FB-044 precedent in same file and mirrors the `buildMainContent()` convention accepted for FB-063 yesterday (viewport height gates View() substring reliability, same rationale). User-persona dispatch pending per cadence rule. **Engineer re-routed to FB-092** (one-line change at `model.go:1495`, spec-ready, P2 hint dedup) per team-lead directive — supersedes my FB-083 routing from prior hour; FB-083 falls to following work. Rationale: FB-092 is a faster-close-to-ship P2 single-line change with spec ready; FB-083 remains P3 and has equivalent spec-ready state. 2026-04-20 **FB-063 ACCEPTED + FB-083 PENDING ENGINEER + FB-089/090 bundled PENDING TEST-ENGINEER.** FB-063 test-engineer submission accepted: 7 brief-AC-indexed tests green, `go install ./...` clean, Option A `SetRefreshing` pattern ships as specified. AC1 dual-mapping pair (Observable `"dnszones"`/`"backends"` present during refresh + Anti-behavior `"Loading quota data"` absent) acceptable as the brief AC combined observation and anti-behavior. AC3 Input-changed simulates the exact `BucketsLoadedMsg` handler sequence (`SetLoading(false)` → `SetBuckets()` → `SetBucketFetchedAt()`), verifying "refreshing" → "updated Xs ago" transition end-to-end. Height-sensitive AC4 + AC6 use `buildMainContent()` directly (consistent with existing `TestQuotaDashboardModel_BuildMainContent_Loading` precedent — viewport wrapping at `height≥6` swallows spinner text from `View()`); precedent-following is appropriate here. All other assertions use `m.View()` end-to-end. User-persona dispatch pending per cadence rule. **FB-083 PENDING ENGINEER** — ux-designer delivered Option C (rows-only hint) with clean single-invariant framing: `[4] hint = data-availability affordance` truthful in exactly one state (`len(activityRows) > 0`). Automatically covers loading, FB-082 `activityFetchFailed`, and empty-state without conditional branching. FB-100 forward-compat noted (stale-row retention will still show hint correctly since `len > 0` includes cached rows). 3-line `renderActivitySection()` change; `lipgloss.Width("")` == 0 handles gap math cleanly. 9 ACs. **FB-089 + FB-090 bundled PENDING TEST-ENGINEER** — engineer bundled per Option 2 coordination protocol (sequential impl would briefly ship `"resume dnsrecordsets (cached)"` — the exact vocabulary mismatch FB-090 exists to fix). New `quickJumpLabel(typeName string) string` helper in `resourcetable.go:~35` iterates `quickJumpTable.matchSubstrs` (same logic as `hasRegistrationMatch`), returns `e.label` on DNS match or `typeName` fallback. S1 block rewritten: `displayName := quickJumpLabel(m.typeName)` used across all three width branches + new `"resume X (cached)"` idiom at full width. S6 strip unchanged. FB-054 AC1/AC2a/AC2b copy assertions updated (FB-089 explicitly replaces the asserted copy — AC1 was failing pre-update, AC2a/AC2b were vacuously passing). Engineer delivered two separate axis-tables (FB-089: 8 rows; FB-090: 9 rows) despite bundled PR — clean separation of brief-AC coverage. 2026-04-20 **FB-084 PERSONA-EVAL-COMPLETE** — user-persona delivered 0 P1 + 0 P2 + 4 P3, all code-verified against `model.go:1964–1994` (placeholderActionRow + call site) + `model.go:1239–1255` (case "r" handler) + `components/errorblock.go:147–151` (ErrorSeverityOf). Positive findings: all three primary surfaces verified correct (permission-denied `[r]` absent + informative hint on `r`-press; transient-error `[r] retry describe` present + redispatch on `r`-press; non-errMode `retryable=false` suppresses `[r]` correctly). Triage disposition: **1 new brief filed, 3 dismissals**. **P3-2** (retryable action row `~43 chars` overflows at `contentW < 40` SSH; FB-084 brief called out 40-col SSH as target but Option D uniform qualifier doesn't fit there) → **FB-106 P3** (engineer-direct, threshold gate drops qualifier below cutoff: `placeholderActionRow(errMode, retryable, contentW, ...)` passes `m.detail.Width()`; above threshold → `[r] retry describe`; below → `[r] retry`). **P3-1** (action row order `[E]` before `[r] retry describe`; persona argues recovery-primary should lead) **DISMISSED** — speculative scanning claim ("operators scanning left-to-right may not reach"), no user evidence; analogous to FB-079 P3-1 separator-inconsistency dismissal precedent; ordering both defensible (standard left-to-right stability vs recovery-primary elevation) and FB-084 ACCEPTED with current order; copy churn without evidence. Refile with operator reports if confusion surfaces. **P3-3** (`r`-press in non-errMode placeholder silently fires table refresh; persona argues action-row-absence implies no-op) **DISMISSED** — premise contradicts TUI convention (`r` is universal refresh key, surfaces in S6 keybind strip, not exhaustive in contextual action rows); `m.tuiCtx.Refreshing = true` at `model.go:1276` renders in header → operator gets visible feedback, not silent; finding itself acknowledges "The `Refreshing` indicator appears in the header." Guard would be scope-creep removing convention-aligned behavior. **P3-4** (non-retryable state has no affordance signaling `r` produces "No retry available" hint) **DISMISSED** — finding self-limits severity ("for a platform operator who knows `r` is a standard TUI key this is low-severity"); proposed fix (muted footnote `"retry not available for this error type"`) directly reverses FB-084 Option A's rationale (suppress `[r]` when broken to avoid broken-promise UX); restoring a visible hint reintroduces the failure-to-act signal the spec intentionally hid; discoverability pathway already exists via S6 + universal `r` convention. Refile with first-time-operator evidence if confusion surfaces. **Net:** 1 new P3 brief (FB-106), 3 P3 dismissals. FB-084 status → PERSONA-EVAL-COMPLETE. Per `feedback_scope_addition_fold_vs_new_brief`, FB-106 filed as distinct user-problem (narrow-width overflow) rather than FB-084 amendment (which is ACCEPTED). 2026-04-20 **FB-102 spec delivered → PENDING ENGINEER.** Ux-designer shipped Option D: compact parenthetical inline in S3 body, transient-only — `"activity unavailable (press [r])"` with `[r]` accent+bold, parenthetical muted. CRD-absent keeps the clean `"activity unavailable"` (distinguished by absence of parenthetical — no copy bloat on the permanent state). Options A/B/C rejected with coherent trade-off reasoning (A less idiomatic than D, B two-line height budget cost, C status-bar collision risk with FB-079/FB-097 quota hints). Implementation adds `activityCRDAbsent bool` field + `SetActivityCRDAbsent(bool)` setter to `ResourceTableModel` with auto-clear in `SetActivityRows()` (so rows-arrival auto-resets without extra wire-up) + 2-line call-site change at `model.go:471`. Clean coordination with FB-100: complementary surfaces (FB-100 = populated-rows retention; FB-102 = empty-rows inline copy), no bundling needed. Ux-designer queue now includes only FB-083 (blocked on FB-082 ship, now unblocked post-acceptance — next designer-call candidate). 2026-04-20 **FB-084 ACCEPTED + FB-064 PENDING TEST-ENGINEER + scope-creep process flag.** FB-084 implementation matches ux-designer spec (Options A+D): `placeholderActionRow(errMode, retryable)` suppresses `[r]` when `ErrorSeverityOf(m.loadErr, m.rc) == ErrorSeverityError`; copy `[r] retry describe` when shown; `case "r":` handler untouched (AC3 key-stays-live pin). 6 brief-AC-indexed tests green (AC1 Observable non-retryable/`[r]` absent + "Describe unavailable"; AC2 Observable retryable/`[r]` + "retry describe"; AC3 Anti-behavior `r`-press posts "No retry available" hint; AC4 Input-changed Error↔Warning severity transition pair; AC5 Anti-regression errMode=false unchanged). `go install ./...` clean, `go test ./internal/tui/... -count=1` green. **Scope-creep process flag:** engineer bundled FB-084 test-assertion modifications into FB-064 implementation — changed AC1/AC4/AC5 from `m.View()` substring checks to `m.buildDetailContent()` substring checks to isolate the placeholder action row from unrelated status-bar `[r] refresh` noise. Investigation disposition: end-state verified consistent with test-engineer's originally submitted axis-coverage table description (table already described assertions against placeholder action row content, not full View()); no working behavior was ripped out; modifications were mechanical (View → buildDetailContent narrowing to same assertion text); all 6 tests remain green. Accepted per `feedback_scope_creep_bonus_bugs` "don't rip out working code" clause, with rule-reinforcement note to engineer: future cross-feature test-infra changes must surface as a separate item (follow-up micro-brief or pre-flight product-experience approval), never silent bundling into another feature's PR. **FB-064** engineer-complete (Option B prepend hint shipped) → routed to test-engineer; axis-table gap noted for guidance (engineer roadmap missing explicit `[Input-changed]` row; AC3/AC4/AC5 mislabeled as Anti-regression when they're actually Anti-behavior — test-engineer must reshape the axis-coverage table to brief-AC indexing with Input-changed pair and correct axis tags). User-persona dispatch for FB-084 pending per `project_persona_agent_pipeline` cadence rule. 2026-04-20 **FB-082 PERSONA-EVAL-COMPLETE** — user-persona delivered 0 P1 + 1 P2 + 4 P3, all code-verified against `model.go:190–193/462–471/643–664/1260–1264` + `resourcetable.go:161–175/300–374`. Positive finding: **all 4 original FB-067 user-problems verified closed** (FB-067 P2 #1 init unconditional-loading fixed at `model.go:190`; FB-067 P2 #2 CtxSwitchedMsg unconditional fixed at `model.go:658`; FB-067 P3 #3 ProjectActivityErrorMsg stuck-spinner fixed at `model.go:470`; FB-067 P3 #5 "no recent activity" silent failure fixed at `model.go:471` + `resourcetable.go:304–305`). Triage disposition: **4 new briefs filed, 1 dismissal**. **P2-1** (`activityFetchFailed` clobbers valid stale rows on `[r]` refresh failure — operator presses `[r]` with populated activity, network blip fires `ProjectActivityErrorMsg`, `SetActivityFetchFailed(true)` unconditionally set at `model.go:471`, S3 snaps from valid rows to "activity unavailable" even though stale data is still in memory) → **FB-100 P2** (engineer-direct, gate `SetActivityFetchFailed(true)` on `ActivityRowCount() == 0`, reserve "activity unavailable" for never-loaded case). **P3-1** (latent panic in Tier 3 width math at `resourcetable.go:322–324`: `actorW = min(16, contentW-22)` goes 0 or negative at `contentW ≤ 22`, subsequent `actorRunes[:actorW-1]` slice panics; dead code today because S3 gated at `contentW >= 50` but unsafe if gate relaxes) → **FB-101 P3** (engineer-direct, 2-line `actorW = max(1, min(16, contentW-22))`). **P3-2** (`"activity unavailable"` reads as permanent — no `[r]` retry affordance; CRD-absent is genuinely permanent and recovery hint would mislead there; transient-error case needs recovery affordance gated on `!isCRDAbsent`) → **FB-102 P3** (designer-call: copy + placement options). **P3-3** (`[r]` refresh handler at `model.go:1262–1264` doesn't call `SetActivityLoading(true)` — empty-rows refresh gives no spinner feedback; FB-076 (ACCEPTED 2026-04-19) addressed dispatch existence, not visual feedback; distinct user-problem) → **FB-103 P3** (engineer-direct, add `SetActivityLoading(true)` in `[r]` handler when rows empty + project-scope + `ac != nil`; consider `SetActivityRefreshing` analog to FB-063 as follow-up if stale-rows populated case becomes a finding). **P3-4** (simultaneous S2 + S3 spinners on project-scope context switch — `⟳` glyph in adjacent blocks reads as busy on slow connections) **DISMISSED** — persona acknowledges "No finding here blocks shipping; it's a P3 polish note"; evidence is speculative ("on a slow connection"); proposed mitigation (stagger visual feedback by changing S3 glyph pattern) introduces its own complexity trading uniformity for busy-ness-reduction without concrete operator evidence. Refile with operator reports if noise is observed in practice. **Net:** 1 P2 new brief (FB-100), 3 P3 new briefs (FB-101/102/103), 1 P3 dismissal. FB-082 status → PERSONA-EVAL-COMPLETE. All 4 findings folded as new briefs rather than scope amendments because FB-082 is ACCEPTED and its shipped scope remains valid — new refinements are distinct user-problems (stale-rows error-policy nuance, defensive hygiene, recovery affordance, refresh-feedback), not diffusions of FB-082's state-machine thesis. 2026-04-20 **FB-082 ACCEPTED** — Activity teaser state machine + 3-tier width truncation shipped per spec. Engineer applied all 4 mechanical fixes (Init-fix with SetActivityLoading inside ac!=nil gate at model.go:186; ContextSwitchedMsg unconditional-loading removed line 636, moved inside dispatch gate at 649, org-scope else-branch resets loading+empty rows; ProjectActivityErrorMsg adds SetActivityLoading(false) + new SetActivityFetchFailed(true); S3 switch gains `case m.activityFetchFailed:` rendering Option B "activity unavailable" copy). Width truncation 3-tier contract (≥65 / 45–64 / <45) shipped. Test-engineer delivered 10 component-level + 5 model-level + 2 existing-anchor anti-regression tests. Axis-coverage table brief-AC-indexed with Observable (AC1 `⟳ loading…`), Input-changed (AC2 ContextSwitchedMsg project vs org scope → loading vs no-recent-activity; AC3 loading → error transition), Anti-regression (AC5 FB-067 chain, AC6 CRDAbsent flag), Integration (AC7). **AC4 dual duty**: happy-path successful fetch renders rows (anti-regression) + `SetActivityRows` clearing `activityFetchFailed` (anti-behavior stuck-flag guard) — both mapped tests green, acceptable overload given both tests assert distinct behavior. **Bonus unmapped anti-behavior coverage in the test file**: `TestFB082_Activity_LoadingTakesPriority_OverFetchFailed` (state-priority ordering when both flags set) + `TestFB082_ErrorRecovery_EmptyRows_ShowsNoActivity` (clearing flag with empty rows surfaces "no recent activity", not stuck "unavailable"). Sharp test design: the `"⟳ loading…"` ellipsis-immediately substring disambiguates S3 activity spinner from S2 `"⟳ loading platform health…"` during bucket-load overlap — test-engineer read the coupled surfaces, not just the brief. Repeat-axis implicitly N/A (async-message-driven state machine, no key-press repeat surface). Full test suite green (15.5s), `go install ./...` clean. User-persona dispatched for FB-082 eval per cadence rule. 2026-04-20 **FB-079 PERSONA-EVAL-COMPLETE** — user-persona delivered 0 P1 + 0 P2 + 3 P3, all code-verified against `model.go:1181` (loading hint) + `model.go:404` (ready-prompt) + `model.go:407–424/518–522` (cancel paths). Implementation verified clean against FB-079 Option D spec — both copy sites match exactly. Triage disposition: **0 new briefs**. **P3-1** (separator inconsistency: ellipsis `…` in loading hint vs em-dash ` — ` in ready-prompt) **DISMISSED** — persona's own framing acknowledges the tonal distinction is "defensible" (ellipsis signals ongoing state, em-dash signals a pivot announcement); no observed user-problem signal, only speculative "operator reading both sequentially may notice"; FB-079 Option D was accepted with this exact separator choice; filing a copy-polish brief without evidence of actual confusion would be churn. Refile with evidence if operators surface the inconsistency downstream. **P3-2** (the "press [3] when ready" tail becomes over-instruction after FB-097 persistent ready-prompt ships — operators no longer need to "remember to wait and press") **already tracked** — FB-099 spec §2 explicitly documents this as the "FB-097 upgrade path": if FB-097 Option A ships and "when ready" becomes redundant, revisit hint copy swap to `"press [3] to cancel"` at that point. Persona finding confirms FB-099's deferred decision was correctly scoped. No action needed beyond acknowledging the confirmation. **P3-3** is a **positive finding**: FB-096 acknowledgment copy (`"Quota dashboard cancelled"`) does not reuse "loading", "ready", or "press [3]" substrings from FB-079; FB-097 inherits the ready-prompt string directly — cross-feature copy coordination verified clean, no collision. FB-079 status → PERSONA-EVAL-COMPLETE. **Net:** 0 new briefs filed, 1 dismissal (P3-1), 1 already-tracked (P3-2 → FB-099 upgrade path), 1 positive confirmation (P3-3). 2026-04-20 **FB-063 PENDING TEST-ENGINEER** — engineer shipped Option A SetRefreshing per spec. Code sites: `quotadashboard.go` adds `refreshing bool` field + `SetRefreshing(b bool)` setter; `SetLoading(loading bool)` extended to auto-clear refreshing on `loading=false` (makes `BucketsLoadedMsg` → `SetLoading(false)` chain clear both flags, no new model.go call site); `buildMainContent()` spinner-branch guard tightened `case m.loading:` → `case m.loading && len(m.buckets) == 0:` (refresh with prior data skips spinner); `refreshViewport()` guard parallel `(m.loading && len(m.buckets) == 0) || m.loadErr != nil`; `titleBar()` adds `⟳ refreshing…` branch pre-empting gap-guarded "updated Xs ago" path. `model.go:~1305` `r`-press handler switches from `SetLoading(true)` → `SetRefreshing(true)`; initial-load sites (`ContextSwitchedMsg`, 3-key entry) retain `SetLoading(true)` — correct per spec (no prior data at those sites). Engineer's axis-table preview tracks brief ACs 1–7 with `[Observable]` x2 (bucket labels retained + `⟳ refreshing…` rendered), `[Input-changed]` AC3 stale→fresh (refreshing→updated Xs ago transition), `[Anti-regression]` x3 (AC4 zero-state spinner intact, AC5 cursor preserved across refresh, AC6 FB-035 green), `[Integration]` AC7 `go install ./...` clean. Note: this is the engineer's roadmap; test-engineer will produce the canonical submitter-owned axis-coverage table at acceptance. Engineer routed to FB-064 (next in queue). 2026-04-20 **FB-096 amendment + FB-099 spec delivered** — ux-designer completed both designer-calls. FB-096 scope amended: Site 3 added at `model.go:1192–1196` (second-press cancel branch); transience policy codified as "explicit cancels = 3s transient via postHint; implicit cancel (nav) = persistent" — Site 3 gets transient symmetric with Esc (both are explicit operator gestures; both return from a handler that CAN return Cmd). 3 new ACs (AC9 Observable View() substring, AC10 Input-changed first-vs-second-press View() diff, AC11 Anti-behavior third-press re-queue). FB-099 Option C selected (strip-only, no hint-copy churn): new `ResourceTableModel.pendingQuotaOpen` field + `SetPendingQuotaOpen(bool)` setter; `renderKeybindStrip()` welcome-branch substitutes `pair("3", "cancel")` during pending; 7 wire-up sites in model.go (~lines 402/421/519/576/863/1190/1194); typed-table branch unchanged. Rationale preserves FB-079 "when ready" copy as load-bearing (FB-097 ready-prompt not yet guaranteed to ship); FB-097 upgrade path documented for post-ship re-evaluation. Coordination note: FB-096 Esc-cancel site adds `m.table.SetPendingQuotaOpen(false)` when shipping FB-096+FB-099 together. **Net:** ux-designer designer-call queue now empty; engineer queue grows by FB-096 (P2, amended) and FB-099 (P3). 2026-04-20 **FB-080 PERSONA-EVAL-COMPLETE** — user-persona delivered 0 P1 + 1 P2 + 4 P3, all code-traced against `model.go:1165/1184/1192–1196` + `model.go:852–855` + keybind-strip site. Triage disposition: **P2-1** (silent second-press cancel, third cancel path not covered by FB-096 scope) + **P3-4** (FB-096 ACs don't cover line 1184) **both fold into FB-096 scope amendment** — FB-096 re-routed to ux-designer (PENDING UX-DESIGNER SCOPE AMENDMENT) to add Site 3 at `model.go:1192–1196` with same `"Quota dashboard cancelled"` hint + new AC for third-path Observable + Input-changed coverage; without amendment, 2/3 cancel paths would post acknowledgment and 1/3 (second-press) would remain silent post-ship. **P3-2** (second-press cancel doesn't clear `quotaOriginPane`) **folds into FB-095 scope expansion** — FB-095 brief amended with Site B at `model.go:1192–1196` alongside existing Site A at `model.go:850–852`; both sites must land in same PR. AC count grows from 4 → 6 with Site B Anti-behavior + Input-changed rows. **P3-1** (loading-hint copy says "press [3] when ready" but [3] now cancels — affordance-copy gap) + **P3-3** (keybind strip doesn't surface `[3] cancel` during pending state — discoverability gap) **both fold into new brief FB-099 P3** — loading-hint + keybind-strip affordance-discoverability gap; designer-call (Option A copy swap / B parenthetical / C strip-only / D A+C / E dismiss). Triage rationale: P3-1 + P3-3 share "pre-press discoverability" thesis (what can I do?) which is distinct from FB-096's "post-press acknowledgment" thesis (what just happened?) — per `feedback_scope_split_pushback`, new brief keeps FB-096's thesis clean. FB-099 priority P3 acknowledges FB-078 auto-transition lowers severity (operators who wait never hit the cancel affordance); reconsider if FB-096 amendment elevates cancel as a reachable gesture. **Net:** 0 new P2 briefs (P2-1 folded), 1 new P3 brief (FB-099 bundles P3-1+P3-3), 2 scope amendments (FB-095, FB-096). FB-080 status → PERSONA-EVAL-COMPLETE. 2026-04-20 **FB-079 ACCEPTED** — Option D hint copy shipped at `model.go:1180`: loading hint `"Quota dashboard loading… press [3] when ready"`; ready hint `"Quota dashboard ready — press [3]"` unchanged. 3 new tests (AC1/AC2 Observable both phases, AC3 old-copy Anti-regression absent); AC4 via existing FB-047 error-clear anchors; AC5 Integration. Anti-regression suites verified green: FB-078 (7), FB-094 (12), FB-055 (6). Axis-coverage table brief-AC-indexed; [Input-changed] row N/A with explicit rationale accepted — FB-079's surface is a one-site copy swap with no AC that varies on input; AC1 vs AC2 already pair the two hint phases (loading vs ready). Forcing a synthetic input-changed test would test a non-existent variant. Test-engineer submitter-owned framing was clean. 2026-04-20 **FB-096 spec delivered** — Option B both Esc-cancel + nav-cancel paths post `"Quota dashboard cancelled"` acknowledgment (no collision with FB-079/FB-078 copy); 2 sites — NavPane Esc handler transient 3s via `m.postHint()` + `updatePaneFocus()` nav-cancel persistent via `m.statusBar.PostHint(...)`; 8 ACs. Option B framing captures "nav-cancel is actually more confusing than Esc case" — operator didn't intend to cancel, just navigated — strong thesis-fit. **FB-097 spec delivered** — Option A persistent ready-prompt; 1-char change at `model.go:401–402` (`m.postHint(...)` → `m.statusBar.PostHint(...)` + `return m, nil`); mirrors FB-079 loading-hint pattern exactly. Clear sites validated (context-switch + FB-096 cancel paths + `[3]` confirm). Options C (glyph badge) + D (FB-089 coupling) rejected with correct scope rationale. **FB-098 DISMISSED** per ux-designer §7 — FB-079+FB-097 joint persistence closes the uncertainty window; refile with evidence if confusion recurs. Ux-designer designer-call queue now empty (FB-096/097 routed to engineer; FB-098 dismissed; FB-088 pending engineer impl at P2). 2026-04-20 **FB-094 PERSONA-EVAL-COMPLETE** — user-persona delivered 0 P1 + 2 P2 + 3 P3, all code-traced. P1 loop verified cleared (4 anti-regression chains traced: `3→4→3→Esc`, `4→3→4→Esc`, `3→4→Esc→Esc`, `TablePane→3→Esc→TablePane`). **0 new briefs filed.** Triage disposition: P2 #1 (single-Esc collapse violates stack-based back-nav mental model) + P2 #2 (Esc destination is history-dependent, not position-dependent) + P3 #3 (no status-bar hint — explicit FB-088 validation) + P3 #4 (ready-prompt copy no Esc-destination signal) **all fold into FB-088 origin-label work** — persona explicitly flagged: "may be addressable by the FB-088 origin-label work rather than requiring new briefs." **FB-088 priority elevated P3 → P2** to reflect 2 P2 findings converging on its thesis. P3 #5 (help overlay `[3]`/`[4]` descriptions stale) verified **non-issue** against `components/helpoverlay.go:56,58` — `"[3] quota (toggle)"` is accurate for the primary single-press use case; cross-dashboard collapse is edge behavior not explainable in 2 words, and the word "toggle" correctly describes origin-return semantics. Triage rationale: findings #1–#4 are all manifestations of the same user-problem (Esc destination is invisible), which is exactly FB-088's thesis — filing duplicates would violate `feedback_scope_split_pushback` (diffuse thesis). FB-088 spec unchanged — `SetOriginLabel(string)` + `dashboardOriginLabel(DashboardOrigin)` contract is architecture-compatible with FB-094 Option A (signatures preserved; call sites inside guards at line 1165 + line 1196 already marked). Team rebuild now requested per cadence (P1 hotfix close + FB-094 thematic batch complete). 2026-04-20 **FB-094 ACCEPTED** — P1 HOTFIX closed. Option A dashboard-as-origin guard shipped: `case "3":` line 1164 wrapped in `if m.activePane != ActivityDashboardPane`; `case "4":` line 1195 wrapped in `if m.activePane != QuotaDashboardPane`; `model.go:46–52` docstring replaced with cross-dashboard skip-stash explanation. 8 new FB-094 tests mapping all 10 brief ACs + dedicated [Input-changed] pair (`TestFB094_InputChanged_3Key_NavPane_StashWritten` + `TestFB094_InputChanged_3Key_ActivityDash_StashPreserved` — same `3` key, activePane varies, different stash outcome — plus symmetric 4Key pair). AC4 tests renamed from `EscLandsOnNavPane` → `QuotaOriginIsNavPane` for spec-faithful direct field assertion. AC9 anchor `TestFB087_AC5_Chain_3_4_Esc_CollapsesToNavPane` verified green targeted. Full suite + `go install ./...` green. **Gate cycle:** initial submit compressed brief AC6+AC8+AC9 into single row and renumbered AC10 as "AC8" → pushed back at pre-submission gate without producing replacement table (per `feedback_axis_coverage_table_ownership`); test-engineer reshaped + added new coverage + renamed AC4. Acceptance verified via direct code read + targeted test-run given P1 priority and substantive completeness (all 10 brief ACs covered, Input-changed present, Anti-regression anchors named) rather than waiting for resubmit ping. Unblocks FB-088 impl (`SetOriginLabel` contract preserved — call sites move inside guard blocks, signatures unchanged) and FB-095 engineer-direct fix (cancel-path stale `quotaOriginPane` clear). **Behavioral delta pinned in acceptance:** `3→4→Esc` now collapses to NavPane on first Esc (FB-087 AC5 intermediate-state updated; AC1+AC2 final-state NavPane preserved). 2026-04-20 FB-078 PERSONA-EVAL-COMPLETE → user-persona delivered 0 P1 + 2 P2 + 2 P3, all code-verified against `model.go:400/798–800/850–852/1173/1177/1495–1501`. **3 new briefs filed:** FB-096 P2 (Esc from NavPane with `pendingQuotaOpen=true` is a pure no-op — natural terminal-conventional cancel gesture is dead; also nav-cancel at line 850–852 silently clears hint with no acknowledgment. Designer-call: Option A Esc-only-cancel vs Option B Esc + unified nav-cancel acknowledgment (folds P3 #3). Option C discoverability-only rejected — hinting at an inert key doesn't fix the premise); FB-097 P2 (ready-prompt hint uses `postHint` → 3s `HintClearCmd` decay, contradicting intentionally persistent loading hint at line 1173. Asymmetry is the user-problem. Designer-call: Option A full persist / Option C glyph badge / Option D FB-089 `(cached)`-style Tab-hint integration. Option B extended decay rejected as tuning-knob that doesn't close the gap); FB-098 P3 (second `[3]` press during loading is silent no-op per line 1177; reassurance-hint Option A is minimum-change fix; may DISMISS under FB-097 Option A if persistent ready-prompt closes the uncertainty window). Triage record added after FB-087 record. Queue order preserves priority: FB-094 (P1) → existing P2/P3 backlog → FB-095 (P3, FB-078 follow-up) → FB-096/097/098 (FB-078 follow-ups clustered thematically) → FB-072/073. FB-078 status block updated with PERSONA-EVAL-COMPLETE marker. No regressions flagged against pre-FB-078 happy path. 2026-04-20 FB-094 P1 spec delivered by ux-designer → PENDING ENGINEER (Option A dashboard-as-origin guard, ~6 lines across `case "3":` and `case "4":` handlers in `model.go`; Option B back-stack rejected as too wide for P1 hotfix; FB-088 `SetOriginLabel` contract preserved — call sites move inside guard blocks, signatures unchanged). 10 ACs. **Critical behavioral delta:** guard fires on the **2nd** cross-dashboard press (not just 3rd), so `3→4→Esc` now lands on NavPane on first Esc (previously QuotaDashboardPane intermediate). Spec AC9 pins the `TestFB087_AC5_Chain_3_4_Esc_IntermediateIsQuotaDash` assertion flip (QuotaDashboardPane → NavPane). FB-087 AC1 + AC2 (final-state after full chain) unaffected. Docstring replacement for `model.go:48` pinned verbatim in spec. Engineer routed; test-engineer on deck for coverage once engineer finishes. 2026-04-20 FB-078 ACCEPTED (FB-047 auto-transition Option B+D — ready-prompt + cancel-on-nav). 7 tests mapping all 5 brief ACs + [Input-changed] gate-axis pair. First submit had 4 tests + prose-shaped table (test-internal-indexed, no [Input-changed] row, [Integration] not tabulated, [Anti-regression] anchor unclear) → pushed back per pre-submission gate feedback-memory. Second submit reshaped table to brief-AC indexing + added the explicit input-changed pair (`TestFB078_InputChanged_a` + `_b`, same `BucketsLoadedMsg{}`, activePane varies) + AC4 anti-regression test covering happy-path `[3]` at ready-prompt → QuotaDashboardPane transition. Observable tests all use `stripANSIModel(appM.View())` substring assertions on "Quota dashboard ready" (per `feedback_observable_acs_assert_view_output`) rather than model-field-only inspection. User-persona dispatched for FB-078 eval. FB-095 dependency now satisfied but held in queue behind FB-094 to preserve engineer P1 sequencing. 2026-04-20 FB-087 user-persona re-evaluation complete (re-derived post-rebuild after prior findings were lost) → 5 findings (1 P1 + 2 P2 + 2 P3), all verified in code before triage. **FB-094 P1 HOTFIX filed** — `3→4→3` inescapable-loop gap: neither `case "3":` (model.go:1152) nor `case "4":` (model.go:1180) guards the "on OTHER dashboard" branch; 3rd keypress clobbers both stash slots with dashboard panes; Esc chain bounces between line 1435 and line 1489 forever. Designer-call at ux-designer: hybrid A+C (dashboard-as-origin guard layered on FB-087 two-slot) vs Option B (back-stack supersedes two-slot). Option C (silently eat keypress) rejected as user-hostile; Option D (Esc always → NavPane) rejected as FB-048 regression. 7 ACs anchored in actual code behavior. FB-088 impl-block shifted FB-087 → FB-094 (stash architecture may change under B). **FB-095 P3 filed** — FB-078 cancel block at model.go:850–852 clears `pendingQuotaOpen` + hint but not `quotaOriginPane`; engineer-direct fix, 1-line addition, sequences after FB-078 acceptance. Folds: stash-comment rewrite (P2-a) → FB-094 AC7; severity note (P2-b) → FB-094 priority rationale. Cross-ref: Esc-destination affordance (P3-a) → FB-088 already pins. FB-078 test coverage in-flight with test-engineer (engineer shipped Option B+D just before rebuild; axis-coverage table required at acceptance submit per feedback-memory). Queue hold: FB-079/080/082/084/063/064/089/090/092 parked until FB-094 cycle closes. 2026-04-20 FB-093 spec delivered by ux-designer → PENDING ENGINEER (Sub-problem 1 Option A: insert `3 quota` + `4 activity` into table branch between `/ filter` and `c ctx` — strategic ordering for narrow-width survivability, `bareParts` static-slice fallback unaffected; Sub-problem 2 Option B: drop `Tab` from dashboard branch when `forceDashboard && typeName != ""` — S1 band's FB-054+089 hint is the richer Tab affordance, double-render is redundant; Tab `next pane` stays when `typeName == ""`). Single `renderKeybindStrip()` rewrite, 18 lines. Ux-designer queue now empty — standing by (FB-088 spec already delivered; FB-083 still blocked on FB-082 ship). 2026-04-20 FB-087 ACCEPTED (Cross-dashboard chaining two-slot stash — `dashboardOriginPane` split into `quotaOriginPane` + `activityOriginPane`; 7 code sites migrated per ux-designer Option A spec; 3 new tests + 3 existing-test references mapping all 7 spec ACs; axis-coverage table brief-AC-indexed on first submit, pre-submission gate passed first pass; `3→4→Esc→Esc` and symmetric `4→3→Esc→Esc` no longer stuck; FB-088 now engineer-ready). FB-090 spec delivered 2026-04-20 by ux-designer → PENDING ENGINEER (Option A — pure in-package `quickJumpLabel(typeName string) string` helper in `resourcetable.go`; called from S1 wherever `m.typeName` appears in Tab hint text; DNS case drives design: `typeName="dnsrecordsets"` currently yields `"dnsrecordsets"` in S1 vs `"dns"` in S4; post-fix aligns; Option C FB-014 resolver rejected as scope-exceeding a P3 copy fix; FB-089 coordination noted — FB-089 engineer must use `quickJumpLabel()` rather than raw `m.typeName` when writing `"resume X (cached)"` copy). Ux-designer **designer-call queue empty** after FB-090 delivery → routed to FB-093 P2. 2026-04-20 FB-056 user-persona evaluation complete → 4 findings (2 P2 + 2 P3); triaged → **FB-093 P2** filed (bundles P2-1 `[3]`/`[4]` cross-context keys disappear in table context + P2-2 `Tab next pane` strip label contradicts FB-054's `[Tab] to resume` band hint — both touch `renderKeybindStrip()`, both ladder from cross-context-mental-model thesis). 2 P3 DISMISSED: P3-1 (`Enter select` on dashboard) — "select" is contextually accurate for sidebar-item-select on dashboard, same verb legitimately covers both sidebar-select and row-select, filing would be copy churn without a user-problem signal. P3-2 (`[3]` destination has three labels across surfaces — `quota` / `quota (toggle)` / `full dashboard`) — each label serves a distinct contextual purpose (strip=memory-prompt, help=behavior-note, S3 CTA=mini-vs-full qualifier); unifying would lose the S3 contrast; FB-092's "dashboard"→"welcome panel" pivot already defuses the most-confusing term collision. Both P2 claims verified against `resourcetable.go:590–644` before filing. 2026-04-20 FB-056 ACCEPTED (Dashboard keybind strip and NavPane status reflect `showDashboard` context — welcomePanel strip drops `x delete`/`/ filter` when `showDashboard=true || typeName==""`; NAV_DASHBOARD statusBar gains `[3]`/`[4]` hints; 7 tests mapping brief ACs 1–7 with full axis coverage; initial-submit axis-coverage table used test-renumbering instead of brief-AC-indexing + missed AC2 `/ filter`-absent assertion → pushed back per pre-submission gate feedback-memory; resubmit added two-line assertion + re-indexed table → accepted). FB-092 spec delivered 2026-04-20 by ux-designer → PENDING ENGINEER (Option A `"Returned to welcome panel"`, one-line change at `model.go:1495`, drops CTA + "dashboard" term, FB-089-compatible, 9 ACs). Ux-designer routed to FB-088 (P3 designer-call, spec-writeable with impl-block on FB-087). Engineer dispatched to FB-057 (help overlay FB-041 semantics) after FB-056 resubmit loop closed. 2026-04-20 FB-055 ACCEPTED (Visible signal on NavPane Esc-to-dashboard — `postHint("Returned to dashboard — Tab to resume")` at model.go:1491, 3s `HintClearCmd` decay; 11 sub-tests with full axis-coverage table; initial submit missing ShiftTab sub-test was pushed back per feedback-memory "State-transition bugs slip through pass-through tests" and resubmitted with fix). FB-055 user-persona evaluation completed 2026-04-20 → 6 findings (2 P2 + 4 P3); triaged → **FB-092 P2** filed (bundles P2-1 redundant-with-FB-054 persistent hint + P2-2 "dashboard" term collision with QuotaDashboardPane/ActivityDashboardPane — both resolve with one hint-copy change). 4 P3 findings DISMISSED with rationale: P3-1 asymmetric Tab-hint polish lacks user-problem evidence (filed as dismissal, revisit if operator reports emerge); P3-2 right-side status-bar placement + P3-3 3s duration + P3-4 ⚡ glyph urgency signal are persona-acknowledged system-wide concerns not isolated to FB-055. FB-091 DISSOLVED 2026-04-20 — subsumed by FB-089 Option A's `(cached)` suffix (ux-designer recommendation accepted; FB-089 AC6 already asserts suffix render). FB-089 spec delivered 2026-04-20 by ux-designer → PENDING ENGINEER (Option A — `[Tab] resume <typeName> (cached)` at all widths, drops "or select a different type", S6 unchanged; "or select a different type" dropped as redundant with sidebar visibility; 4-line change in `resourcetable.go:513–531` width-gate block). FB-085+086 merged spec delivered 2026-04-20 by ux-designer → PENDING ENGINEER (both detail-error-state fixes, one spec `fb-085-086-detail-error-state-fixes.md`). **Pre-FB-055 post-rebuild:** engineer dispatched on FB-055 (queue: 055→056→057→087→080→079→078→082→084). Ux-designer redirected from FB-075 (P3) to FB-063 (P2) after priority correction; FB-075 spec already delivered at `docs/tui-ux-specs/fb-075-s5-attention-kind-separator.md` (Option A blank-row) before redirect landed — accepted as delivered, → PENDING ENGINEER. Ux-designer revised queue: FB-063 → FB-064 → FB-085+086 merged → FB-088 (spec writeable, impl blocked on FB-087) → FB-083 (blocked on FB-082 ship). FB-085+086 merge APPROVED (complementary DetailView error-state fixes, one spec `fb-085-086-detail-error-state-fixes.md` + one impl PR). FB-054 user-persona evaluation complete 2026-04-20 → 3 new briefs: FB-089 P2 (Tab hint copy cohesion, bundles P2-1+P3-1 — S1 verb-phrase vs S6 label contradiction + short-form "to"-drop style switch), FB-090 P3 (S1 raw typeName vs S4 display label vocabulary drift), FB-091 P3 (Tab cached vs quick-jump fresh copy distinction). All four findings verified against code (`resourcetable.go:514/520/529/367/607` and `model.go:1023/1035`). FB-073 spec delivered 2026-04-20 by ux-designer → PENDING ENGINEER (Option A — gate quick-jump on `m.activePane != NavPane` at model.go:981; single-condition diff; no inline glyphs in NavPane — S4 already serves as discovery surface; 3/4 keys unaffected as they route via separate code path). Spec at `docs/tui-ux-specs/fb-073-navpane-quickjump-gate.md`. **Team rebuild requested 2026-04-20** by team-lead per cadence rule (rebuild after each ACCEPTED feature). FB-054 ACCEPTED 2026-04-19 (Welcome panel Tab-to-restore hint when table cached — 7 tests covering Observable/Happy AC1+AC3 (View() contains "[Tab]"+resource type+"to resume") + Input-changed AC2a/AC2b/AC4 + Anti-behavior AC5 (Enter still fires load) + Anti-regression AC6 (FB-041 Esc chain); split component+model test files). FB-087 spec delivered 2026-04-19 by ux-designer — Option A two-slot stash (`quotaOriginPane` + `activityOriginPane`); 7 code sites documented; corrected trace `NavPane → 3 → 4 → Esc → Esc` returns Activity → Quota → NavPane. Option C rejected as it would break the "Esc undoes most recent navigation" invariant. **P2 designer queue is now clear.** FB-082 + FB-084 specs delivered 2026-04-19 by ux-designer (FB-082: Option B "activity unavailable" copy + new `activityFetchFailed` state field + 3-tier narrow-width truncation contract; FB-084: Option A+D — suppress `[r]` when severity is non-retryable + restore `[r] retry describe` qualifier). FB-053 ACCEPTED 2026-04-19 (transition hint when events swap error block for placeholder — 5 tests covering Observable hint+View()-diff + Anti-behavior normal-load+error-load + Anti-regression FB-037 placeholder; AC4 required engineer fix for `msg.Err == nil` guard). FB-048 + FB-050 PERSONA-EVAL-COMPLETE 2026-04-19 → 4 findings triaged: P2-1 → FB-087 (cross-dashboard chaining overwrites single-slot stash → stuck loop on dashboard pane); P2-2 → cross-ref FB-078 (already addressed by B+D); P3-1 → cross-ref FB-080 (impl note: cancel branch must skip stash overwrite); P3-2 → FB-088 (no on-screen affordance for `3`/`4` toggle-back return destination). FB-049 ACCEPTED 2026-04-19 (QuotaDashboardPane status-bar hint filter — 8 tests, axis table complete). FB-078 + FB-079 + FB-080 specs delivered 2026-04-19 (Option B+D / D / C). FB-081 DISSOLVED 2026-04-19. FB-051 + FB-052 PERSONA-EVAL-COMPLETE 2026-04-19 → FB-084 + FB-085 + FB-086 filed. FB-048 + FB-077 + FB-076 ACCEPTED 2026-04-19. FB-067 PERSONA-EVAL-COMPLETE 2026-04-19 → FB-082 + FB-083 filed. FB-051 + FB-052 ACCEPTED 2026-04-19. FB-054 + FB-055 + FB-056 + FB-057 specs delivered 2026-04-19. FB-050 ACCEPTED 2026-04-19. FB-047 user-persona evaluation complete 2026-04-19 → 5 NEW briefs FB-077 (P2 re-queue loop, engineer-direct) + FB-078 (P2 auto-transition cancel path, designer) + FB-079 (P3 hint copy, designer) + FB-080 (P3 silent-second-press, designer) + FB-081 (P3 auto-transition origin re-stash, follow-up to FB-048). FB-076 FILED 2026-04-19 (P3 follow-up for `r`-press dispatch from welcome — scope item #3 of FB-067 was added after engineer started; deferred to honor in-flight implementation). FB-047 ACCEPTED 2026-04-19 (`3` keypress queue while QuotaDashboard loads — 7 tests AC1–AC7 green, install clean). FB-042 user-persona evaluation reconciled 2026-04-19: prior triage table listed fabricated P-numbered findings; corrected against persona's actual delivery → FB-068/069/070 WITHDRAWN; FB-067 retained (independently code-verified) with P2-2 framing absorbed; 4 NEW briefs FB-072 (P2 Esc count) + FB-073 (P2 NavPane focus quick-jump) + FB-074 (P3 unconfigured-quota copy) + FB-075 (P3 attention-kind separator). FB-043 P3-4 (just-now boundary) DISMISSED — spinner tick re-renders View() at ~100ms; FB-043 P3-5 → FB-071 (legitimate). FB-066 ACCEPTED 2026-04-19 (prefixWidth constant pinned). FB-042 ACCEPTED 2026-04-19 (welcome dashboard). FB-058 ACCEPTED 2026-04-19 (sibling-data reword). FB-044 + FB-043 + FB-046 + FB-041 + FB-037 ACCEPTED. **FB-045 REJECTED** — premise wrong: `[t]` toggles pane, not flat/grouped; sibling-data reword salvaged as FB-058. FB-043 persona → FB-059 (gap-guard P3) + FB-060 (failed-refresh P3). FB-043+FB-044 persona re-run (2026-04-19) → FB-063 + FB-064 + FB-065 + FB-066. FB-061 + FB-062 WITHDRAWN. FB-068 + FB-069 + FB-070 WITHDRAWN 2026-04-19 (filed against fabricated FB-042 P-numbers; numbers reserved). FB-037 persona → FB-051/052/053; FB-041 persona → FB-054/055/056/057; FB-036 persona → FB-043/044/045/046; FB-035 persona → FB-047/048/049/050. Ux-designer reports FB-048+049+050 specs PENDING ENGINEER. **Next product-experience tasks:** trigger user-persona evaluation of FB-047 (per pipeline cadence); route FB-067 + FB-072 + FB-074 to engineer once they finish FB-067; route FB-073 + FB-075 to ux-designer queue.

Full acceptance history: `docs/tui-backlog-archive.md`. Far-queue full briefs (FB-007/008/009, FB-027–034): `docs/tui-backlog-deferred.md`.

---

## Current TUI State (audit summary)

### What is implemented

All Phase 1 files exist and compile. The core interaction loop is fully wired:

- **Entry point** — `datumctl tui` cobra command with `--read-only` flag, calls `tui.Run()`.
- **AppModel** — Init/Update/View TEA loop with proper pane state machine (NavPane → TablePane → DetailPane) and two overlay slots (CtxSwitcher, HelpOverlay).
- **Header** — 8-row blue banner with ASCII wordmark, user/org/project info line, ns/pane-label/refresh line, and conditional READ-ONLY badge.
- **NavSidebar** — bubbles/list with compact single-row delegate; shows `▸` cursor, resource-count annotation, name truncation. Focus coloring via active/inactive border.
- **ResourceTable** — bubbles/table using the Kubernetes Table API (printer columns, not hard-coded Name/Status/Age). Dynamic column widths with Name getting ≥35% of space. Status-dot color coding via `StatusColorFor`. Client-side filter via FilterBar.
- **Welcome panel** — shows Kind/Resource/Group/Scope/Description for the hovered sidebar item and a four-column keybind reference. Satisfies REQ-TUI-007 and REQ-TUI-008.
- **DetailView** — bubbles/viewport showing metadata, spec, and status-conditions table with formatted ages. Satisfies REQ-TUI-016.
- **FilterBar** — textinput with accent prompt; real-time filter applied on each keystroke; Esc clears and restores. Satisfies REQ-TUI-015.
- **StatusBar** — mode-aware label (NORMAL/FILTER/DETAIL/OVERLAY) with context-sensitive keybind hints and right-aligned error display.
- **HelpOverlay** — four-column modal. Satisfies REQ-TUI-017.
- **CtxSwitcher** — tree of org → project entries; Enter persists new context via `datumconfig.SaveV1Beta1` and fires `ContextSwitchedMsg`. Satisfies REQ-TUI-018.
- **Auto-refresh** — 15-second `TickCmd`; `r` forces immediate reload of resource types. Header shows humanized `updated Xs ago`. Satisfies REQ-TUI-019.
- **Quit** — `q` and `Ctrl+C` from any non-filter, non-overlay state. Satisfies REQ-TUI-020.
- **Layout** — `layout.SidebarWidth` scales with terminal (16–32 cols); `layout.TableWidth` takes the remainder. Resize events recalculate all pane dimensions.

### What is missing or broken

**Note (2026-04-19 kickoff audit):** Items 1, 2, 3, and 7 from the original audit are ACCEPTED (via FB-003, FB-009, FB-001, FB-002). Item 5 remains an engineering concern not a UX concern and has been addressed alongside each feature ship. Items 4 and 6 remain open. Items below updated to reflect 2026-04-19 state — only unshipped and newly-identified gaps listed.

1. **No error recovery UI** (elevated) — `LoadErrorMsg` writes the error to `statusBar.Err` and displays it right-aligned in the footer in red, but the app stays in LoadStateError indefinitely with no prompt to retry. Operators who encounter a transient API error have no visible call to action. FB-005 covers this; promoted from Medium → High on 2026-04-19 given the proliferation of new fetch paths across FB-010/012/013/015/016/017/018/019. See FB-005.

2. **No in-TUI namespace scope switch** — Header advertises `ns: <namespace>` per REQ-TUI-002 but no key opens a namespace selector. Operators debugging in multi-namespace projects must exit the TUI or re-select the entire project context via `c` (which doesn't offer a namespace drill-down). This is the most-requested workflow gap after error recovery. See FB-020 (new, 2026-04-19).

3. **No global resource search** — `/` filters rows within the currently-loaded resource type but nothing searches across types. Operators who know the name (`my-prod-api`) but not the type must guess-and-navigate. k9s-style `:` command bar is a well-established TUI pattern that fills this gap. See FB-021 (new, 2026-04-19).

4. **CtxSwitcher double-renders the backdrop** — `CtxSwitcherModel.View()` calls `lipgloss.Place` with `WithWhitespaceBackground` internally, and `AppModel.View()` calls `lipgloss.Place` again with the same backdrop color when rendering `CtxSwitcherOverlay`. This double-wrapping may produce rendering artifacts on some terminals. Tracked as a minor defect; not yet briefed pending reproduction on a non-iTerm terminal.

5. **FB-007 (change history diff) still PENDING** — `datumctl activity history --diff` is operationally valuable for "what changed in the last hour that broke things." FB-006 (activity) and FB-016 (activity rollup) ship activity events but not the per-revision diff surface. Remains PENDING Medium.

6. **FB-008 (multi-session switcher) still PENDING** — Operators with multiple sessions (personal + work) cannot switch sessions from the TUI. Low-Medium priority — workflow affects a narrow operator segment.

7. **Error rendering is fragmented** — `statusbar.go:88` dumps raw `err.Error()` (multi-line JSON blobs corrupt status-bar rendering); `historyview.go` / `activityview.go` copy-paste a two-line error pattern; `quotadashboard.go` runs title + detail + retry into one line; `activitydashboard.go` has a `sanitizeErrMsg` helper nobody else uses; `model.go:1598` renders YAML marshal errors with an inline `⚠` in the viewport body with no affordance. No severity hierarchy distinguishes hard errors (auth/RBAC) from transient (timeouts). See FB-022 (new, P1, 2026-04-19 from user input) — shared `ErrorBlock` component + universal `sanitizeErrMsg` + severity resolver. Underpins FB-005's inline retry card.

8. **Sidebar is a flat alphabetical list across API groups** — `ResourceType.Group` field exists but is never surfaced. As resource type count grows, scanning becomes harder. See FB-023 (new, P2, 2026-04-19 from user input) — section headers or collapsible tree or tab strip (ux-designer picks from three options).

---

## Feature Briefs

---

**Note:** ACCEPTED briefs (FB-001–006, FB-010–019, FB-022, FB-024) are in `docs/tui-backlog-archive.md`.

---

### FB-007 — Change history with diff in DETAIL pane

**Status: DEFERRED** — full brief in `docs/tui-backlog-deferred.md`
**Priority: Medium**

`H` key from DetailPane opens revision history list + unified diff (N vs N-1). Requires `ActivityClient` expansion with `GetHistory`/`GetRevision`. REQ-TUI-026.

---

### FB-008 — Multi-session switcher

**Status: DEFERRED** — full brief in `docs/tui-backlog-deferred.md`
**Priority: Low-Medium**

Extend CtxSwitcher to three-level tree (session → org → project). Selecting a project under a different session swaps sessions and re-fetches resource types. REQ-TUI-027.

---

### FB-009 — Raw YAML toggle in DETAIL pane

**Status: DEFERRED** — full brief in `docs/tui-backlog-deferred.md`
**Priority: Medium**

`y` key in DetailPane toggles between describe and raw YAML of the same resource. No new fetch — marshals the `Unstructured` already in memory. Per-resource state; resets on pane exit, row change, context switch. REQ-TUI-028.

### FB-020 — Namespace picker: in-TUI scope switch within the active context

**Status: PENDING** — written 2026-04-19 by product-experience during kickoff audit.

**Priority: High**

#### User problem

The TUI's working namespace is taken from `m.tuiCtx.Namespace` (set at startup from `datumconfig.LoadAuto()`'s active context). The header already advertises the namespace scope with `ns: <namespace>` on the info line (REQ-TUI-002), so operators *expect* to be able to change it — but there is no keybind or UI surface that does. The only path today is:

1. Press `c` to open CtxSwitcher
2. Re-select the same org / project
3. Observe the picker does not offer a namespace drill-down — namespace is derived from the project context, not selectable

Operators who need to scope a list to a single namespace (a common debugging pattern: "show me HTTPRoutes only in `team-alpha`" when the project contains 12 team namespaces) must quit the TUI and run `kubectl -n team-alpha get httproutes`. This is the most-requested workflow gap after error recovery — it defeats the TUI's primary purpose for any operator working in a multi-namespace project.

Relatedly: `ListResources` passes `ns` only when `rt.Namespaced` is true (`model.go:211, 229, 433, 777, 982, 1012, 1084`). The plumbing is already namespace-aware; only the selector UI is missing.

#### Proposed interaction

**New key `N` (Shift+n)** from NavPane or TablePane opens a **Namespace Switcher overlay** (new `OverlayID` value `NamespaceSwitcherOverlay`). Lowercase `n` is unclaimed and reserved for a hypothetical future "next result" navigation; `N` is the mnemonic.

**Overlay layout:**
```
 ┌─ Namespace ──────────────────────────────────┐
 │  (all)                                       │
 │▸ default                             ✓       │
 │  team-alpha                                  │
 │  team-beta                                   │
 │  kube-system                                 │
 │  datum-system                                │
 └──────────────────────────────────────────────┘
    [j/k] nav  [Enter] select  [Esc] cancel
```

- First item is always `(all)` — sets `m.tuiCtx.Namespace = ""`, which Kubernetes dynamic-client treats as all-namespaces for namespace-scoped resources.
- Current namespace is marked with `✓` and highlighted on open.
- List is fetched via `ResourceClient.ListNamespaces(ctx)` — new method that wraps `factory.DynamicClient().Resource(namespacesGVR).List()`. Cache for the CtxSwitcher session (no TTL; refetched on each overlay-open or context switch).
- Selecting a namespace: updates `m.tuiCtx.Namespace`, dismisses overlay, dispatches `LoadResourcesCmd` for the currently-selected type (if in TablePane) or remains idle (if in NavPane — the table will use the new namespace on next type selection). Header's `ns:` label updates.
- Selection does **not** persist to `~/.datum/config.yaml` — this is a transient in-TUI scope change, not a context mutation. Leaving the TUI reverts to the saved context's namespace. (Parallel to CtxSwitcher, which *does* persist; namespace picker deliberately does not — operators debugging often change namespace multiple times per session and don't want that bleeding into `kubectl`.)

**Failure modes:**
- `ListNamespaces` RBAC-forbidden: render the overlay with a single muted line *"No permission to list namespaces. Press Esc to cancel."* — no retry, no hint; the operator must have `list namespaces` RBAC in the project's control plane to use this feature.
- `ListNamespaces` error (network, timeout): same overlay shell with *"Could not list namespaces: <err>"* and `[Esc]` only.
- Namespace no longer exists (selected, then deleted out-of-band): first `ListResourcesCmd` returns a 404 scoped error; FB-005's inline error card handles presentation; operator re-opens `N` and selects a live namespace. FB-020 does not duplicate FB-005's recovery UX.

#### Design rules

- **Read-only operation.** No creation, deletion, or annotation of namespaces. REQ-TUI-037 (mutation test-safety) is NOT in scope — this is a pure scope selector.
- **Transient scope, not persisted.** See above. Matches operator mental model for debugging sessions.
- **Does not replace CtxSwitcher.** `c` still switches org / project context (and as a side-effect resets namespace to that context's saved default). `N` is the finer-grained scope switch *within* a project.
- **Cluster-scoped types ignore namespace.** Selecting `(all)` or any specific namespace has no effect on listing Projects, Organizations, CRDs, or anything where `rt.Namespaced == false`. Those lists render unchanged.
- **No filter inside the overlay.** If an operator has >20 namespaces and wants to type-to-filter, that's a v1.1 concern. v1 is a straight list.

#### Non-goals

- **Not a multi-select namespace filter.** Scope is single-namespace-or-all. Multi-select ("show httproutes in team-alpha AND team-beta") is a follow-up if requested.
- **Not a namespace-create affordance.** Per REQ-TUI-037 and the read-only mandate for this brief.
- **Not a namespace-events view.** FB-019 (events) remains per-resource; a namespace-scoped events rollup is its own brief if warranted.
- **Not a namespace-browser that lists workloads inside.** `N` changes scope; the existing sidebar + table mechanics show the workloads once scope is set.
- **No persistence to `~/.datum/config.yaml`.** Explicitly rejected — see Design rules.

#### Acceptance criteria

Axis tags: `[Happy]`, `[Repeat-press]`, `[Input-changed]`, `[Anti-behavior]`, `[Failure]`, `[Edge]`, `[Observable]`, `[Integration]`.

1. **[Happy / Observable]** Pressing `N` from NavPane opens the Namespace Switcher overlay. Overlay renders over the NavPane sidebar and right pane (similar placement to CtxSwitcher), status bar mode changes to `OVERLAY`. Overlay shows `(all)` as the first row and the list of namespaces returned by `ListNamespaces`, sorted alphabetically.
2. **[Happy / Observable]** Pressing `N` from TablePane opens the overlay with the current namespace pre-selected (highlighted + `✓` marker). `(all)` appears first with `✓` if `m.tuiCtx.Namespace == ""`.
3. **[Happy]** Selecting a namespace (Enter) dismisses the overlay, updates `m.tuiCtx.Namespace` to the selected value (empty string for `(all)`), and dispatches `LoadResourcesCmd` for `tableTypeName` if TablePane was the origin and a type was loaded.
4. **[Happy / Observable]** The header's `ns:` label updates to the new value within 100ms of selection. `(all)` renders as `ns: (all namespaces)` in the header per existing convention (or as documented by UX designer).
5. **[Repeat-press]** Pressing `N` a second time after selecting a namespace re-opens the overlay with the newly-selected namespace pre-highlighted. The overlay is idempotent; repeated opens do not stack or duplicate.
6. **[Repeat-press]** Re-selecting the *same* namespace that is already active (Enter on the currently-checked row) dismisses the overlay and does NOT dispatch a fresh `LoadResourcesCmd` (no-op optimization). Test: assert no `ResourcesLoadedMsg` fires after self-select.
7. **[Input-changed]** After selecting namespace `B` from state `A`, the table's rows reflect `B` — specifically: a seeded mock `ResourceClient.ListResources` is called with `ns == "B"` on the re-dispatch, and the table renders the B-returned rows. Test: seed per-ns mock, assert table body changes between selections.
8. **[Input-changed]** Selecting `(all)` from any prior namespace sets `m.tuiCtx.Namespace = ""` and the subsequent `LoadResourcesCmd` is called with `ns == ""`. Test: explicit empty-string assertion on the dispatched command's ns parameter.
9. **[Input-changed]** `ContextSwitchedMsg` resets `m.tuiCtx.Namespace` to the new context's saved default (not preserved across context switches). Test: open N-overlay, select team-alpha; `c` to switch project; assert `m.tuiCtx.Namespace` is now the new project's default, not `team-alpha`.
10. **[Anti-behavior]** Pressing `N` while the filter bar is focused types `N` into the filter input — filter consumes the key. Overlay does not open. Test: `/` → type some chars → `N` → assert `filterBar.Value()` contains `"N"` and `m.overlay == NoOverlay`.
11. **[Anti-behavior]** Pressing `N` while any other overlay is active (`CtxSwitcherOverlay`, `HelpOverlayID`, `DeleteConfirmationOverlay`) is consumed by the active overlay's key handler — no namespace overlay opens. Test matrix: one sub-test per overlay state.
12. **[Anti-behavior]** Pressing `N` in DetailPane, QuotaDashboardPane, ActivityPane, HistoryPane, ActivityDashboardPane, or DiffPane is a no-op — the namespace selector only opens from pane states where a subsequent list would apply the scope. Enumerated test matrix, one sub-test per pane.
13. **[Anti-behavior]** Lowercase `n` is NOT handled by FB-020 — no `case "n":` branch added. Reserved for future use. Test: on NavPane, press `n`, assert `m.overlay == NoOverlay` and no dispatched command.
14. **[Failure / Observable]** When `ListNamespaces` returns an RBAC-forbidden error (`IsForbidden(err) == true`), the overlay renders *"No permission to list namespaces. Press Esc to cancel."* — no retry, no list. Title stays `Namespace`. Esc dismisses cleanly.
15. **[Failure / Observable]** When `ListNamespaces` returns any other non-nil error, the overlay renders *"Could not list namespaces: <err>"* (truncated to overlay width) with `[Esc]` only. No retry UI inside FB-020 — FB-005's general retry path does not apply to overlay-scoped fetches.
16. **[Edge]** When `ListNamespaces` returns an empty slice (the project genuinely has zero namespaces — impossible for a live Datum project but testable), the overlay renders `(all)` as the only row. Selecting it is a no-op. No error.
17. **[Edge]** When the selected namespace is subsequently deleted out-of-band (external kubectl delete), the next `LoadResourcesCmd` returns a scoped 404; FB-005's error card handles presentation; pressing `N` again re-fetches the namespace list (without the now-deleted one). FB-020's overlay does not cache stale namespaces.
18. **[Observable]** Cluster-scoped resource types (`rt.Namespaced == false`) render unchanged after namespace selection — the header still shows the new `ns:` label (the scope is set, just unused), but the table body is identical to pre-selection. Test: load Projects (cluster-scoped), change namespace, assert table rows unchanged.
19. **[Observable]** The `?` HelpOverlay gains a `[Shift+N] namespace` entry in the NAVIGATION section, pane-gated on NavPane / TablePane (parallel to FB-017 `ShowDeleteHint` pattern). New `ShowNamespaceHint bool` on `HelpOverlayModel`, set true when `m.activePane ∈ {NavPane, TablePane}` at help-open time.
20. **[Integration]** `NamespacesLoadedMsg` and the overlay close both succeed cleanly under a concurrent `TickMsg` (auto-refresh) — a 15s tick fired mid-overlay does not dismiss the overlay, does not trigger a table reload that conflicts with the pending namespace change. Test: open overlay, inject `TickMsg`, assert overlay still active, no competing dispatch.
21. **[Integration / Anti-regression]** Existing FB-001 / FB-002 / FB-003 / FB-004 / FB-010 / FB-011 / FB-014 / FB-015 / FB-016 / FB-017 / FB-018 tests pass unchanged. The `m.tuiCtx.Namespace` mutation is additive on the Pane→Dispatch path; other message handlers already read `m.tuiCtx.Namespace`, so no call-site changes required. CI guarantee — `go test ./internal/tui/...` green after the change IS the proof. Parallel to FB-016 AC#21 / FB-019 AC#27 regression-guard pattern.

**Dependencies:**
- **Existing `ResourceClient` interface** — extended with `ListNamespaces(ctx) ([]string, error)`.
- **REQ-TUI-002** — the `ns:` header label is the observable target of AC#4.
- **FB-005** — soft dependency. Out-of-band namespace deletion is handled by FB-005's inline retry; FB-020 does not duplicate that path.

**Maps to:** REQ-TUI-018 (context switcher overlay) — `N` is its namespace-scope sibling. Propose new requirement: **REQ-TUI-040 — Namespace Switcher overlay (`N` key)**. Sub-requirements:
- 040.a `N` keybind pane-local to NavPane / TablePane; opens NamespaceSwitcherOverlay
- 040.b `ResourceClient.ListNamespaces` lists namespaces in the current project control plane
- 040.c `(all)` first-row sentinel maps to `m.tuiCtx.Namespace == ""`
- 040.d Selection updates `m.tuiCtx.Namespace` in-memory only; NOT persisted to `datumconfig`
- 040.e Context switch resets namespace scope to the new context's saved default
- 040.f Cluster-scoped types render unchanged regardless of namespace selection
- 040.g HelpOverlay `ShowNamespaceHint` pane-gating parallels FB-017 / FB-018 / FB-019 patterns

**Amends:**
- `AppModel` gains `namespaceOverlay components.NamespaceSwitcherModel` sub-model (parallels `ctxOverlay`).
- `OverlayID` gains `NamespaceSwitcherOverlay` constant.
- `handleNormalKey` gains a `case "N":` branch at the pane-gated sites.
- `ResourceClient` interface gains `ListNamespaces(ctx context.Context) ([]string, error)`.
- `internal/tui/data/` gains `LoadNamespacesCmd` / `NamespacesLoadedMsg` (folded into `resourceclient.go` per engineering discretion).
- `HeaderModel` rendering for the `ns:` slot updates — `(all namespaces)` label when `m.tuiCtx.Namespace == ""`.
- `?` HelpOverlay gains `[Shift+N] namespace` entry, pane-gated by new `ShowNamespaceHint bool`.

**Out of scope:**
- Multi-namespace selection (v1.1 if requested).
- Namespace creation / deletion (REQ-TUI-037 applies if ever briefed; deliberately out of scope here).
- Type-to-filter inside the overlay (v1.1).
- Persisting namespace scope across TUI sessions (deliberately transient).

**Next step:** route to ux-designer for overlay layout, `(all)` sentinel rendering, header `ns:` label variant, `[Shift+N]` HelpOverlay entry placement, failure-state messaging. Spec location: `docs/tui-ux-specs/fb-020-namespace-picker.md`.

---

### FB-021 — Global resource search: jump-to-resource by name (`:` command bar)

**Status: PENDING** — written 2026-04-19 by product-experience during kickoff audit.

**Priority: Medium**

#### User problem

Operators debugging an incident often know the *name* of the failing resource ("my-api-route", "prod-gateway") but not its *type*. Today they must:

1. Guess the type (HTTPRoute? Gateway? VirtualService?)
2. Navigate the sidebar to that type
3. Wait for the table to load
4. `/` filter by name
5. If no match, repeat with the next guess

This is a discoverability and workflow gap. In k9s the `:` command bar lets operators type `:httproute my-name` and jump directly. The datumctl TUI has `/` for *within-type* filtering but no *across-type* jump. For large projects (>20 resource types, hundreds of resources) this is a painful daily friction.

The sidebar is alphabetical; operators with an active debugging workflow want keystroke-efficient jumps, not mouse-and-scroll navigation.

#### Proposed interaction

**New key `:` (colon)** opens a **Command Bar** pinned above the status bar (parallel placement to FilterBar, slight visual distinction via prompt glyph). The command bar accepts a prefix-match query of the form:

```
: <name-fragment>              → search all loaded resource types by name
: <type> <name-fragment>       → scope search to a specific resource type
: <type>                       → select the type (equivalent to sidebar navigation)
```

**Layout (over status bar):**
```
 : httproute my-prod
   ┌─ 3 matches ──────────────────────────────────────┐
   │▸ httproutes / my-prod-api       (team-alpha)     │
   │  httproutes / my-prod-cdn       (team-alpha)     │
   │  httproutes / my-prod-edge      (team-beta)      │
   └──────────────────────────────────────────────────┘
    [j/k] nav  [Enter] open  [Esc] cancel
```

- **Prefix search on resource-type names** — as the operator types, the type token (first word) is matched against the cached `resourceTypes` slice. Exact match → scope to that type; otherwise fuzzy-match against type display names.
- **Name search** — second token (or only token if type token unrecognized) is matched case-insensitively against resource names. Up to 10 matches shown; overflow truncated with `(… N more)`.
- **Lazy per-type fetch** — fetching resources for every type on every keystroke is expensive. Strategy: search against types that have already been loaded this session (cached in `resources` only for the *currently-selected* type). Fallback: prompt operator with *"press Enter to search all types (slow)"* which triggers a parallel `ListResources` across all loaded types.
- **Enter on a match** — navigates the sidebar to that type, dispatches `LoadResourcesCmd`, awaits result, auto-selects the matching row, opens DetailPane (`d`-equivalent). Single keystroke → jump directly to inspecting the resource.
- **Tab completion** — pressing Tab on a partial type token completes to the unique match or offers a suggestion list.

**Failure modes:**
- Query matches nothing → matches list renders *"No matches. Press Esc to cancel."*
- Cross-type search slow path fails on one type → that type's row shows `<type> — error: <short>`; other types' results still render. Per-type failure does not block the whole search.
- Matching resource deleted out-of-band between search and Enter → FB-005 inline error handles presentation; search bar remains available.

#### Design rules

- **`:` is the command-bar prompt.** `/` remains the within-type name filter (REQ-TUI-015). The two are distinct: `/` filters rows, `:` searches types + navigates. Parallel conventions to vim.
- **Case-insensitive substring match.** Not regex, not fuzzy-lite, just substring. Simpler mental model.
- **Results are navigational, not actionable.** Enter opens DetailPane on the match. No delete, no edit — this is a jump, not a workspace.
- **Read-only operation.** REQ-TUI-037 (mutation test-safety) NOT in scope.
- **Esc cancels cleanly.** No state persists after Esc — `m.commandQuery = ""`, overlay dismissed, focus returns to prior pane.
- **Cache strategy: pragmatic, not exhaustive.** Search over the currently-loaded type's rows instantly; require an explicit Enter to trigger the across-type slow path. Avoid eager pre-warming all types (which would blow the k8s client's QPS budget on the session-open keystroke).

#### Non-goals

- **Not a full-text search over describe content.** Operators searching for a field value (`spec.rules[0].backendRefs[0].name`) need a follow-up brief if requested.
- **Not a cross-context search.** FB-021 searches within the current org/project/namespace scope only. Searching "this resource across all projects" is operationally complex and out of scope.
- **Not a bookmarks / history list.** `:` is stateless: no recall of prior queries. Session history would be a v1.1 feature.
- **Not a keyboard recorder / macros.** No `q` register, no replay. Simple command bar.
- **Not a YAML / describe content indexer.** Search is strictly on `metadata.name`. Annotations, labels, spec fields not indexed.
- **No auto-open-first-result.** Always show the match list; Enter on a specific row opens it. Operators with heavy workflows can `:httproute foo<Enter>` which will navigate the type and use the first match via FB-016-style cursor positioning — still not an auto-open, still shows one row in the list first.

#### Acceptance criteria

Axis tags: `[Happy]`, `[Repeat-press]`, `[Input-changed]`, `[Anti-behavior]`, `[Failure]`, `[Edge]`, `[Observable]`, `[Integration]`.

1. **[Happy / Observable]** Pressing `:` from NavPane or TablePane opens the command bar above the status bar. Prompt glyph `:` renders in accent color; `filterBar` (`/`) is not affected. Status bar mode changes to `COMMAND`.
2. **[Happy / Observable]** Typing `httproute` as the sole token matches the `httproutes` resource type; pressing Enter navigates the sidebar to `httproutes`, dispatches `LoadResourcesCmd`, and transitions focus to TablePane with no row-jump.
3. **[Happy / Observable]** Typing `httproute my-prod` queries `httproutes` for rows whose name contains `my-prod` (case-insensitive). Match list renders with up to 10 rows; pressing Enter on the highlighted row navigates to the resource and opens DetailPane.
4. **[Happy]** Typing only a name fragment (no recognized type prefix) with no slow-path confirmation triggers a search over the currently-loaded type only — matches from `m.resources` slice whose `Name` contains the query substring render as navigational hits.
5. **[Happy / Observable]** Pressing Tab on a partial type token (`httpro<Tab>`) completes to the unique match (`httproutes`) if unambiguous; otherwise renders the suggestion list of the top 5 prefix matches.
6. **[Repeat-press]** Pressing `:` a second time while the command bar is already active is a no-op — the bar stays focused, query preserved. Test: assert `m.commandQuery` unchanged after double-press.
7. **[Input-changed]** Editing the query progressively updates the match list in real time (each keystroke triggers a re-match against the cached type rows). Test: type `my`, then `my-p`, then `my-prod`, assert the match count decreases monotonically across the three rendered states.
8. **[Input-changed]** `ContextSwitchedMsg` mid-command closes the command bar and clears `m.commandQuery`. The newly-loaded context does NOT inherit the query.
9. **[Anti-behavior]** Pressing `:` while any overlay is active (`CtxSwitcherOverlay`, `HelpOverlayID`, `DeleteConfirmationOverlay`, `NamespaceSwitcherOverlay`) is consumed by the overlay's key handler — command bar does not open. Test matrix: one sub-test per overlay.
10. **[Anti-behavior]** Pressing `:` while the filter bar is focused types `:` into the filter input — filter consumes the key. Command bar does not open. Test: `/` → type some chars → `:` → assert `filterBar.Value()` contains `":"`.
11. **[Anti-behavior]** Pressing `:` in DetailPane, QuotaDashboardPane, ActivityPane, HistoryPane, ActivityDashboardPane, or DiffPane is a no-op. The command bar is a navigation tool; deep-pane states don't benefit from it. Enumerated test matrix.
12. **[Anti-behavior]** Any query starting with a mutation verb (`delete`, `rm`, `edit`) is explicitly rejected with an inline hint *"Command bar is read-only navigation. Use [x] on a row to delete."* — never dispatches a mutation command. Defense-in-depth: FB-021 introduces a parseable command surface, and operators with prior CLI muscle memory will type `:delete foo` if this isn't blocked. AC#12 is the guardrail.
13. **[Failure / Observable]** When the slow-path across-type search has one type fail (mock a single `ListResources` returning err), that type's matches row renders *"<type> — error: <short>"* in Warning color; other types' match rows render normally. The whole search does not fail.
14. **[Failure / Observable]** When `Enter` on a match triggers `LoadResourcesCmd` and the fetch fails, FB-005's inline error card shows in the TablePane; command bar remains usable (not auto-dismissed) so the operator can try a different match.
15. **[Edge]** Empty query: the match list shows *"Start typing a resource name or type"* — no matches, no error, no spinner.
16. **[Edge]** Query matches zero rows: match list renders *"No matches for `<query>`. Press Esc to cancel."* Title stays `Command`.
17. **[Edge]** Query matches >10 rows: top 10 render; last row is *"(… N more matches — narrow your search)"*. Test: seed 15 matching rows, assert truncation at 10 with the overflow hint rendered.
18. **[Observable]** Match rows render `<type-short> / <name>   (<namespace>)` if namespaced, `<type-short> / <name>` otherwise. The type-short is the singular form of the Kubernetes `kind` (e.g., `httproute`, not `httproutes`). Format pinned for consistency with FB-014 display-name conventions.
19. **[Observable]** The `?` HelpOverlay gains a `[:] command` entry in the NAVIGATION section, pane-gated on NavPane / TablePane (parallel to FB-020 `ShowNamespaceHint`). New `ShowCommandHint bool` on `HelpOverlayModel`.
20. **[Integration]** Full lifecycle: NavPane → `:` → type `gateway prod<Enter>` → sidebar navigates to `gateways`, TablePane loads, cursor lands on the match, DetailPane opens with describe content. One test exercises the complete chain with a seeded mock ResourceClient.
21. **[Integration / Anti-regression]** Existing FB-001 through FB-019 tests pass unchanged. Adding the `m.commandQuery` field and `case ":"` handler is additive; no existing key handler paths are modified. CI guarantee — `go test ./internal/tui/...` green. Parallel to FB-016 AC#21 / FB-019 AC#27 pattern.

**Dependencies:**
- **FB-003** (filter pane gating) — establishes the "key-pane-gated affordance" pattern FB-021 reuses.
- **FB-020** (namespace picker) — soft dependency. Both introduce navigation-scoped overlays; FB-021's command bar can be scoped to the active namespace, and the axis-coverage / overlay-precedence test patterns are shared. Target engineering sequencing: FB-020 ships first, FB-021 reuses its overlay-precedence tests as a template.
- **Existing `ResourceClient.ListResources`** — no new method required; FB-021 uses existing list operations and adds in-memory prefix matching. If operators report slow-path performance issues, a future brief could add a bulk-list optimization.

**Maps to:** new capability. Propose **REQ-TUI-041 — Global resource command bar (`:` key)**. Sub-requirements:
- 041.a `:` keybind pane-local to NavPane / TablePane; opens CommandBar
- 041.b Query grammar: `[<type-prefix>] <name-fragment>` (space-separated)
- 041.c Case-insensitive substring match on `metadata.name` within the scoped type
- 041.d Tab completion on the type token
- 041.e Slow-path cross-type search requires explicit Enter confirmation; per-type failures isolated
- 041.f Read-only: mutation verbs in the query are rejected with a hint
- 041.g HelpOverlay `ShowCommandHint` pane-gating

**Amends:**
- `AppModel` gains `commandBar components.CommandBarModel` sub-model, `commandQuery string` state field.
- `handleNormalKey` gains a `case ":":` branch at pane-gated sites.
- New `internal/tui/components/commandbar.go` component (parallel to `filterbar.go`).
- `StatusBarModel` gains `ModeCommand` (parallel to `ModeFilter`).
- `?` HelpOverlay gains `[:] command` entry.
- Mutation-verb rejection is enforced in `commandBar.Update` — no mutation paths added.

**Out of scope:**
- Per-query caching / search history (v1.1 if requested).
- Full-text describe-content search (separate brief if warranted).
- Regex or fuzzy matching (v1 is substring; operators can add wildcards later).
- Auto-open of the first match on Enter-with-unambiguous-query (explicit match-list UX preserved).

**Next step:** **Blocked on FB-005 and FB-020** shipping first, because (a) FB-005's inline error card is the dependency for AC#14, and (b) FB-020's overlay-precedence tests are the template for AC#9. Route to ux-designer after FB-020 ships and FB-005 is in test-engineer's queue. Spec location: `docs/tui-ux-specs/fb-021-command-bar.md`.

---


### FB-023 — Sidebar resource grouping by service/product

**Status: PENDING** — written 2026-04-19 by product-experience from team-lead user-forwarded input during kickoff audit.

**Priority: P2** — scannability improvement. Today's ~12–15 resource types are tractable as a flat alphabetical list; the value of grouping grows as the type count grows. Ship after FB-005 and FB-020 (higher-impact workflow fixes).

#### User problem

The sidebar renders `resourceTypes` as a flat alphabetical list. Each `ResourceType` already has a `Group` field (e.g., `networking.datumapis.com`, `iam.datumapis.com`, `compute.datumapis.com`) but the sidebar does not visually distinguish groups — operators scan the whole list to find a type, even when they know the resource lives in `networking`. As the resource type count grows (today ~12–15; projected 25+ as Datum Cloud adds services), the flat list becomes hard to scan and the alphabetical sort interleaves unrelated types (`deployments` → `gateways` → `httproutes` → `ipaddresspools` → `policies`).

This is a discoverability gap, not a correctness gap. Operators can still find types via scroll or via FB-021 (`:` command bar, once it ships). But the cognitive load of parsing "which group is this type in" at every sidebar scan is nontrivial.

#### Proposed interaction — three design options (ux-designer decides)

**Option 1 — Section headers in the flat list (lowest effort, recommended starting point).** Insert non-selectable divider items as group headers between clusters sharing the same API group. Visual:

```
  NETWORKING
    gateways
    httproutes
    ipaddresspools
  IAM
    policies
    roles
  COMPUTE
    deployments
    workloads
```

- Requires a thin wrapper `list.Item` type (e.g., `sidebarItem` union of `headerItem` and `resourceTypeItem`) and a Group → display name mapping table (e.g., `networking.datumapis.com` → `NETWORKING`).
- Cursor skips headers — a small Update interceptor in `NavSidebarModel` advances the cursor past header items when `j`/`k` would land on one.
- Headers are styled (bold, muted accent color, indented one character from resource type rows).
- Reuses existing `bubbles/list`; minimal structural change.

**Option 2 — Collapsible tree (highest value, most effort).** Replace the `bubbles/list` with a custom tree model. Top-level nodes are services (toggle open/close with Space or ←/→), children are resource types. Familiar from k9s, lazygit.

- Requires building a tree widget from scratch (no `bubbles/tree`).
- Cursor model: up/down traverses visible leaves + headers; headers toggle on Space/→/←.
- Initial state: all groups expanded by default.
- Group state persists via a `sidebarExpandedGroups map[string]bool` field on `AppModel`; sticks across context switches unless the new context has a different group set.

**Option 3 — Tab strip above the list (middle ground).** Horizontal `[Networking] [IAM] [Compute] [All]` strip filters the list below. Extra state: one `selectedGroup string` field on `NavSidebarModel`.

- Very k9s/lazygit-familiar.
- Operators can switch groups with `1`/`2`/`3`/`A` (hotkey per group) or `←`/`→` to rotate.
- Status bar shows `NAV (Networking)` mode label.
- Flat list below tab strip filters to only types in the selected group; `All` shows the full list.

**Recommendation:** start with Option 1. It ships fast, reuses existing components, and provides the primary scannability benefit. If resource-type count grows past ~20 types or operators report wanting collapse, revisit with Option 2 as a follow-up brief. Option 3 is appealing but adds keyboard-hotkey complexity (`1`/`2`/`3` conflicts with the existing pane-switcher keys from FB-016's `4` key convention).

Route to ux-designer with the three options; they decide. If ux-designer picks Option 2 or 3, the AC set below needs extension.

#### Design rules

- **Group → display name mapping is a curated table**, not derived from the API group string. `networking.datumapis.com` → `NETWORKING`; `iam.datumapis.com` → `IAM`. Unknown groups (e.g., CRDs from third-party operators) fall back to the raw group name in lowercase, rendered as-is.
- **Ungrouped resources go last.** Cluster-scoped kubernetes core types (Namespaces, Nodes) render under a `CORE` section at the bottom of the sidebar.
- **Cursor position preserved across re-renders.** If the resource type list is reloaded (manual `r` in NavPane or `TickMsg`), the cursor stays on the same resource type if it still exists; otherwise snaps to the nearest visible type (skipping headers).
- **REQ-TUI-037 (mutation test-safety) NOT in scope.** Pure display refactor; no new mutation surface.

#### Non-goals

- **Not a user-customizable grouping.** Group assignment follows the Kubernetes API group; operators cannot re-group types.
- **Not pinning / favorites.** Operators cannot elevate a type to the "top." If heavy workflow demands this, a follow-up brief (`FB-024`? — provisional) would add pinned types above the group list.
- **Not a search/filter on the sidebar itself.** `/` in NavPane still does nothing (per FB-003 — it's TABLE-pane-gated). `:` from FB-021 provides the global search surface.
- **Not a tab reorder / group reorder.** Group order is alphabetical by display name with `CORE` last. No operator control.
- **Not auto-collapse of groups with 0 types.** If a group has zero visible types in the current context, the header is omitted entirely — not rendered as empty.

#### Acceptance criteria (assumes Option 1 — section headers)

Axis tags: `[Happy]`, `[Repeat-press]`, `[Input-changed]`, `[Anti-behavior]`, `[Edge]`, `[Observable]`, `[Integration]`, `[Refactor-parity]`.

1. **[Happy / Observable]** Sidebar renders with group headers interleaved between resource-type rows. Headers are non-selectable, rendered in muted bold style, with a single-character indent distinguishing them from type rows. Test: render a 3-group fixture, assert header substrings appear in expected positions.
2. **[Happy / Observable]** Group headers use the curated display-name mapping: `networking.datumapis.com` → `NETWORKING`, `iam.datumapis.com` → `IAM`, `compute.datumapis.com` → `COMPUTE`, `core` → `CORE`. Test: per-mapping sub-tests with display-name assertion.
3. **[Happy]** Unknown API groups (e.g., `foo.bar.baz`) fall back to a `FOO.BAR.BAZ` header (uppercased raw group). Test: seed unknown-group fixture, assert uppercased header.
4. **[Happy]** Core kubernetes types (empty `Group` field) render under `CORE`, positioned last in the sidebar. Test: assert `CORE` header appears after all other groups.
5. **[Happy]** Within a group, resource types render alphabetically by `Name`. Test: seed out-of-order types in a group, assert rendered output is sorted.
6. **[Repeat-press]** Pressing `j` on a type row advances cursor to the next type row, skipping headers. Test: cursor at `networking/gateways`, press `j`, cursor should land on `networking/httproutes` (not on the `NETWORKING` header between `gateways` and `httproutes` — irrelevant because headers aren't between same-group items, but the skip logic must fire when moving across group boundaries).
7. **[Repeat-press]** Pressing `j` at the last type in a group advances cursor to the first type in the next group, skipping the group header. Test: cursor at last `networking` type, press `j`, cursor lands on first `iam` type (not on `IAM` header).
8. **[Repeat-press]** Pressing `k` at the first type in a group retreats cursor to the last type in the previous group, skipping the group header. Test: cursor at first `iam` type, press `k`, cursor lands on last `networking` type.
9. **[Repeat-press]** Pressing `j` at the last type of the last group does NOT advance past the end (no wrap-around). Symmetric for `k` at the first type. Test: cursor at last type; `j` is a no-op.
10. **[Input-changed]** `ResourceTypesLoadedMsg` re-renders the sidebar with new groupings. If the previously-selected type still exists in the new list, cursor stays on it. If it was removed (e.g., permission change), cursor snaps to the nearest visible type (skipping headers). Test: two re-render scenarios.
11. **[Input-changed]** `ContextSwitchedMsg` preserves no sidebar state across contexts — cursor resets to the first type, groups render fresh per the new context's resource type list.
12. **[Input-changed]** Pressing `j` repeatedly traverses every type in the sidebar (test: count of `j`-presses needed to reach end = total type count across all groups; cursor visits each type exactly once).
13. **[Anti-behavior]** Headers cannot be selected via `Enter` — pressing `Enter` while cursor is on a header position (only possible via explicit cursor-set call in a test; impossible from `j`/`k` given AC#6/7) is a no-op. Defensive test.
14. **[Anti-behavior]** Pressing digit keys (`1`–`9`) in NavPane does NOT jump to group N (reserved for pane-switcher per FB-016 convention). Test: press `1` in NavPane, assert no group jump, FB-016 pane-switch fires as expected.
15. **[Edge]** Single-group fixture (all types share one `Group`): sidebar renders with one header and the type list below. Test: assert exactly one header substring renders.
16. **[Edge]** Zero-group fixture (all types are `core`): sidebar renders with one `CORE` header. Test.
17. **[Edge]** Group with a single type renders with the header + one type row. Cursor behavior unchanged. Test.
18. **[Edge]** Terminal height too small to show all groups: sidebar scrolls — `bubbles/list` viewport handles it. Headers scroll with types (not sticky). Test: small-height render, assert scroll works and cursor stays visible.
19. **[Observable]** The `?` HelpOverlay's NAVIGATION section is unchanged by this brief — `[j/k] nav` remains the only sidebar-nav hint. No new hint for header-skipping (it's implicit). Intentional: fewer hints = cleaner overlay.
20. **[Observable]** The resource-type count annotation format (`<type> (N)` from REQ-TUI-006) is preserved inside each group — grouping wraps around type rows, does not alter per-row rendering. Test: assert `httproutes (3)` substring still appears under the `NETWORKING` header.
21. **[Refactor-parity]** Existing REQ-TUI-005 / REQ-TUI-006 tests pass unchanged — sidebar border/active-color behavior, cursor highlight, count annotations, name truncation. Test: run existing sidebar tests against the refactored impl; all green.
22. **[Integration / Anti-regression]** Existing FB-001 through FB-021 tests pass unchanged. `NavSidebarModel.SelectedType()` returns the same type regardless of whether headers are rendered between the cursor and other rows; all consumers of `SelectedType` see the same value. CI guarantee — `go test ./internal/tui/...` green. Parallel to prior anti-regression pattern.
23. **[Integration]** `go install ./...` compiles cleanly after the refactor.

**Dependencies:**
- **`NavSidebarModel`** (`internal/tui/components/navsidebar.go`) — refactored to wrap `list.Item` union (header + type).
- **`ResourceType.Group`** field (`internal/tui/data/types.go`) — already present; this brief consumes without modification.
- **Curated group → display-name map** — new file or constant in `internal/tui/styles/` or `internal/tui/data/`; enumerates the 6–8 known Datum API groups.

**Maps to:** REQ-TUI-006 (sidebar resource types listed) — extends with group scaffolding. Propose new **REQ-TUI-043 — Sidebar resource grouping by API group**. Sub-requirements:
- 043.a Group headers rendered between clusters of types sharing an API group
- 043.b Curated Group → display-name map with fallback to uppercased raw group
- 043.c Cursor skips headers during `j`/`k` navigation
- 043.d Alphabetical sort within groups; alphabetical sort of group display names; `CORE` always last
- 043.e Existing per-row rendering (count annotation, name truncation, cursor highlight) preserved
- 043.f No user-facing customization (group/order/pin)

**Amends:**
- `internal/tui/components/navsidebar.go` — refactored to render `sidebarItem` union (header + type).
- `internal/tui/data/types.go` or new `internal/tui/data/groups.go` — curated group → display-name map.
- `NavSidebarModel.Update` — cursor-skip logic for `j`/`k` on header positions.
- `NavSidebarModel.SelectedType()` — returns the resource-type at the cursor (unchanged contract; internal implementation skips header items).

**Out of scope:**
- Pinning / favorites (follow-up brief if requested).
- Collapsible tree (Option 2 — deferred until operator feedback demands it).
- Tab strip (Option 3 — deferred; digit keys conflict with FB-016 pane-switcher convention).
- User-reorderable groups.
- Group-level search / filter.

**Next step:** route to ux-designer for option selection (Option 1 recommended as starting point) and visual spec (header styling, indent depth, group→display-name map, CORE placement). If ux-designer selects Option 2 or 3, the AC set above needs extension. Spec location: `docs/tui-ux-specs/fb-023-sidebar-grouping.md`.

---

## Recommended starting point for the UX designer

**Start with FB-001 (Detail view chrome).**

Rationale:

1. It is the highest-visibility gap in the current UI. Every operator who inspects a resource encounters the bare viewport with no orientation cues. The fix is self-contained within `DetailViewModel` and `AppModel.recalcLayout` — it does not require changes to the data layer, keybindings, or other components.

2. The acceptance criteria are fully visual, making designer → engineer handoff clean: the designer can produce a mockup of the title bar and scroll footer at exact character widths, and the engineer has a clear specification to implement against.

3. It unblocks the REQ-TUI-016 automated test which currently cannot verify "detail pane is focused" (the border focus fix is part of this brief).

FB-002 (force-refresh) is equally important from an operator-workflow perspective but requires understanding the data loading state machine — better suited for the first engineer pass. FB-003 (filter guard) is a correctness fix that resolves a REQ violation; it should follow FB-002 in the engineering queue.

---


### FB-025 — Events freshness: in-place refresh + staleness indicator

**Status: ACCEPTED 2026-04-20** — spec at `docs/tui-ux-specs/fb-025-events-freshness-in-place-refresh.md`. Implementation: `detailview.go:29` (eventsFetchedAt field) + setter/getter at 115-116 + age label in titleBar at 169-170 + `RenderEventsTable` signature extended with fetchedAt at 335 (stale-empty recovery copy at ≥5m at 347-350). `model.go:367` sets timestamp on `EventsLoadedMsg` success; 3 reset sites at 623, 1469, 1482 clear on context/resource/pane transitions. D1 r-refresh dispatcher (r on DetailPane when `(eventsMode || events!=nil) && !eventsLoading`). Tests: 10 functions / 13 leaf cases (TestFB025_AC1/AC2/AC3 = r-key dispatch behavior; AC4 = SetFetchedAt on success; AC5 = error retains prior — third submission strengthened to pre/post `" · "` View() pair proving fetchedAt survives error reload; AC8 = reset sites × 5 sub-tests — third submission strengthened with `setFetchedAt` precondition + `assertNoAgeLabel` postcondition at identical width/mode so reset is the only suppressor; AC9 = refresh guard; Component AC4/AC6/AC7 = age label rendering + empty-state divergence). Test-engineer gate PASS on third submission after 3 violation rework rounds (AC4 model, AC5 labeling, AC8 helper) + 2 hardening rounds (second pass fixed bounded assertions, third pass eliminated vacuousness). Product-experience independent verification: all 13 leaf cases PASS, `go install ./...` clean, component hooks verified. Next: engineer commits FB-025 to `feat/console`, then user-persona evaluation. Prior: written 2026-04-19 by product-experience after user-persona's FB-019 evaluation.

**Priority: High** — in-flight incidents are the exact use case the events sub-view exists for. Current behavior (no `r`-refresh, no staleness signal) actively misleads the operator.

#### User problem

Persona (senior platform engineer debugging a live reconciliation incident) identified two compounding gaps that only appear under real operator workflows:

1. **(P2) No in-place refresh of events.** After the initial paired `LoadEventsCmd` fires on `d`, the events table is frozen until the operator exits DetailPane and re-enters. `case "E":` re-dispatches only when `m.events == nil || m.eventsErr != nil` (FB-019 §D8 cache-hit semantics); `case "r":` on DetailPane calls `LoadResourcesCmd` on the sidebar (`model.go:994`) and does NOT dispatch `LoadEventsCmd`. The operator's only path to fresh events is `Esc → d → E` — three keystrokes with a cursor round-trip through TablePane that breaks flow mid-triage.

2. **(P2) No staleness indicator.** Operator sees the same events table for as long as they stay in DetailPane. There is no "last fetched Nm ago" timestamp, no asterisk, no visual cue. During an incident, the operator believes they are seeing "what the control plane just did" when they are actually seeing "what the control plane did 12 minutes ago when I first loaded this DetailPane." This is an active misdirection, not a neutral gap. Combined with the no-refresh issue, the operator has no tool to correct it.

3. **(P3 — bundled) Empty-state is sterile and indistinguishable from pruned-state.** `"No events recorded for this resource."` accurately describes "zero events returned." But a resource whose `FailedSchedule` events got pruned 2 minutes ago (Kubernetes default events TTL is ~1h) and a resource that genuinely never had any events look identical to the operator. Combined with no staleness indicator, there is no signal that distinguishes "healthy steady state" from "just-missed-the-events window."

FB-019 explicitly deferred `r`-refreshes-events as engineer-discretion (§D8 last bullet). The operator cost of that deferral is significant during the exact workflow the feature was built for.

#### Proposed interaction (design decisions)

**D1 — `r` on DetailPane dispatches `LoadEventsCmd` when `m.eventsMode == true` OR when the events sub-view has been entered previously in this DetailPane session.** The "has been entered previously" branch catches the operator who pressed `E` once, toggled off to read describe, and now wants a fresh read without re-toggling. Semantically: if events are "relevant to this session" (eventsMode OR events slice non-nil), `r` refreshes them alongside the existing describe refresh. Engineering discretion for the eventsMode-only simpler variant if the "previously entered" branch adds state tracking complexity — document the call.

**Rejected:** `r` always dispatches `LoadEventsCmd`. Wastes API calls when the operator is only in describe mode. The trigger should be semantic (events are in scope for this session), not blanket.

**D2 — Title-bar mode indicator includes fetch-age when events mode is active.** When `m.mode == "events"` and `m.events != nil`, title bar reads `<kind> / <name>  events · 4m ago` (append " · Nm ago" / " · Ns ago" / " · Nh ago" / " · just now" suffix to the existing `events` label). Age computed from `m.eventsFetchedAt` — a new `time.Time` field on AppModel, set at `EventsLoadedMsg` handler (`model.go:302–305`) on success (not on error — error case keeps the previous successful fetch's age if any, else no age label). Re-fetch via `r` resets the age. Threshold: show label only if age > 1s; "just now" for age <= 15s.

**Rejected:** auto-refresh timer. Polls the control plane every Ns during an incident when the operator is already monitoring — wrong affordance. Operators want control over when a refresh happens, not a timer.

**D3 — Empty-state distinguishes "fresh empty" from "potentially-pruned" via age-since-first-observed-state.** When `m.events == nil && eventsErr == nil && !eventsLoading`, empty state renders:
- If `m.eventsFetchedAt` was set <5 minutes ago (fresh fetch): `"No events recorded for this resource."` (current copy, AC#15 unchanged).
- If `m.eventsFetchedAt` was set ≥5 minutes ago (stale empty): `"No events recorded as of Nm ago. Press [r] to refresh."` — directs operator to the newly-wired refresh affordance.

**Rejected:** always-append retry hint. When a resource is healthy and has been for hours, the "press [r] to refresh" hint is noise. The stale-empty case is where the hint earns its presence.

#### Acceptance criteria

Axis tags as before.

1. **[Happy / Observable]** Pressing `r` on DetailPane while `m.eventsMode == true` dispatches `LoadEventsCmd` alongside the existing describe refresh. `eventsLoading` sets true; existing events table remains visible until `EventsLoadedMsg` arrives. Test: enter events mode, press `r`, assert both `DescribeResourceCmd` and `LoadEventsCmd` in the batch.
2. **[Happy / Observable]** Pressing `r` on DetailPane while `m.eventsMode == false` but `m.events != nil` (operator entered events mode previously, then toggled back to describe) dispatches `LoadEventsCmd` in the refresh batch — per D1 "previously entered" branch. Test: `E`-on, `E`-off, press `r`, assert `LoadEventsCmd` dispatched.
3. **[Anti-behavior]** Pressing `r` on DetailPane while `m.events == nil` (events never fetched — should not happen given paired-fetch semantics, but defensive) does NOT dispatch a new `LoadEventsCmd`. Test: synthetic state with events=nil + eventsMode=false, assert no LoadEventsCmd in refresh batch.
4. **[Observable / Input-changed]** After a successful `EventsLoadedMsg`, `m.eventsFetchedAt` is set to the message arrival time. Title bar during events mode reads `<kind> / <name>  events · just now` when age <=15s. Age updates on subsequent renders. Test: seed a mock clock, assert substring changes from `just now` to `30s ago` to `2m ago`.
5. **[Observable]** After events load failure (`eventsErr != nil`), `m.eventsFetchedAt` is NOT updated — retains the previous successful fetch's timestamp (or stays zero if no prior success). Test: successful load followed by a failed reload; age label reflects the first load's time.
6. **[Edge]** `m.eventsFetchedAt` zero-value does NOT render an age label (avoids `events · 55y ago` on startup). Test pin: pre-fetch state, mode indicator reads `events` with no age suffix.
7. **[Observable]** Empty-state copy diverges by age: `age < 5m` → `"No events recorded for this resource."` (unchanged from FB-019 AC#15). `age >= 5m` → `"No events recorded as of Nm ago. Press [r] to refresh."` substring rendered. Test: two sub-tests pinning each branch.
8. **[Input-changed]** Context switch (`ContextSwitchedMsg`) and DetailPane exit (`Esc`) both clear `m.eventsFetchedAt` to zero alongside the existing four-invariant reset. Test: reset-site matrix extended with eventsFetchedAt assertion.
9. **[Repeat-press]** Multiple rapid `r` presses during in-flight refresh behave per FB-024 AC#7 — exactly one in-flight `LoadEventsCmd` at a time. Test: `r` during `eventsLoading=true`, assert no second dispatch.
10. **[Integration / Anti-regression]** FB-019 AC#8 (`E` no-op when `describeRaw == nil`), AC#23 (quad-state), AC#27 (existing tests green) preserved. Refresh behavior on other panes (NavPane reloads types, TablePane reloads rows) unchanged.

**Dependencies:**
- **FB-019 (ACCEPTED)** — AC set extension; no conflict.
- **FB-024 (PENDING UX-DESIGNER)** — shares the re-dispatch guard at `model.go:911`. FB-025 builds on FB-024's `!eventsLoading` guard. Sequence FB-024 first, FB-025 second. Confirmed non-blocking: if FB-024 ships after FB-025, merge resolution is mechanical.
- **`r` refresh handler at `model.go:964–`** — extends existing DetailPane branch to include events dispatch.

**Maps to:** amends REQ-TUI-039 (FB-019):
- 039.d1 `r` on DetailPane refreshes events when events are in session scope (eventsMode OR events non-nil)
- 039.e1 Title-bar mode indicator includes fetch-age when events mode is active (> 1s)
- 039.f1 Empty-state copy diverges by fetch age (fresh <5m vs stale >=5m)

**Next step:** ux-designer for the exact age-format + empty-state-copy-swap + title-bar composition. Then engineer + test-engineer in parallel.

---

### FB-026 — Keybind hint format consistency pass

**Status: ACCEPTED 2026-04-20** — spec at `docs/tui-ux-specs/fb-026-keybind-hint-format-consistency.md`. Implementation: `detailview.go:143-150` (toggle-swap on C) + helpoverlay format normalization. Tests: `TestFB026_AC1_TitleBar_HintMatrix` (4 sub-tests: empty/yaml/conditions/events modes), `TestFB026_AC2_HelpOverlay_CanonicalFormat`, `TestFB026_AC3_HelpOverlay_GlobalHelp_NoToggleVerb`, `TestFB026_AC4_ConditionsMode_ToggleSwap`, `TestFB026_AC5_PaneGating_Preserved` (4 sub-tests), `TestFB026_AC6_NarrowWidth_HintRowDropped`. Test-engineer gate PASS. Product-experience independent verification: 10 sub-tests PASS, `go install ./...` clean, code matches spec at `detailview.go:143-150`. Next: engineer commits FB-026 to `feat/console` with product-prose message. Prior: written 2026-04-19 by product-experience after user-persona FB-019 eval surfaced cross-surface inconsistency. **Update 2026-04-19:** user-persona FB-024 eval re-flagged this as P3-3 (title-bar uses `[E]`, HelpOverlay uses `[Shift+E]`); operator hit the notation mismatch immediately after FB-024 shipped the first `[E]` title-bar surface — confirming FB-026's priority is "address before more surfaces ship," not "defer." Carry-over: HelpOverlay `[Shift+E]` at `helpoverlay.go:43` resolved as part of the format-consistency pass.

**Priority: Medium** — cosmetic correctness, no operator-workflow blocker, but the accumulated inconsistency will calcify as more briefs ship.

#### User problem

Three discoverability surfaces render the same keybinds using three different conventions:

| Surface | Keybind | Format |
|---|---|---|
| DetailPane title-bar hint (`detailview.go:147`) | yaml | `[y] yaml` |
| DetailPane title-bar hint (`detailview.go:147`) | conditions | `[C] toggle conditions` |
| HelpOverlay (`helpoverlay.go:40,43`) | conditions / events | `[Shift+C] conditions` / `[Shift+E] events` |
| HelpOverlay (`helpoverlay.go:32–38`) | describe, filter, back, describe-again | `[Enter] select`, `[d] describe`, `[Esc] back/cancel` |
| Status-bar hint lines (various) | varies | Mixed |

Inconsistencies surfaced by persona finding #6:

1. **Lowercase-bracket `[y]` vs uppercase-with-modifier `[Shift+E]`.** `[C]` in the title bar and `[Shift+C]` in HelpOverlay are the SAME key rendered two ways. A user who reads the title bar and then searches the HelpOverlay for `[C]` will miss the match.
2. **Verb-bare `[y] yaml` vs verb-explicit `[C] toggle conditions`.** Mixed in the same hint row. Either the verb is semantic ("toggle") or it's absent ("yaml"); pick one.
3. **Dead copy-paste branch** (RW #3): `cHint := "[C] toggle conditions"` at `detailview.go:143` followed by `if m.mode == "conditions" { cHint = "[C] toggle conditions" }` at `:144–146` — identical re-assignment. Likely intended to parallel `yHint` where `m.mode == "yaml"` swaps to `"[y] describe"`. Symptom of the inconsistency that hardened into a bug.

#### Proposed direction (design decisions)

**D1 — Canonical format: `[<key>] <action-verb-or-noun>`.** Single lowercase bracket with the key name as typed; action word short and direct. Examples:
- `[y] yaml` / `[y] describe` (toggle-label swap applied consistently — eliminates dead branch)
- `[C] conditions` / `[C] describe` (drop "toggle" verb — context makes it obvious)
- `[E] events` / `[E] describe`
- `[x] delete`
- `[Esc] back`
- `[Shift+E] events` in HelpOverlay → **change to `[E] events`** (HelpOverlay follows the same single-bracket convention; the uppercase letter in the brackets IS the Shift+letter signal).

**Rationale:** `[Shift+E]` is verbose and pedagogical but redundant — operators recognize uppercase in brackets as Shift-modified. Every operator-facing TUI in the ecosystem (k9s, lazygit, htop) uses the single-bracket convention. Datum's HelpOverlay is the outlier.

**Rejected:** uppercase-the-key-name alternative (`[SHIFT+E]`). Louder but not clearer.

**Rejected:** explicit modifier prefix (`Shift+E`). Denser; doesn't buy anything over `[E]`.

**D2 — Verb choice: drop "toggle" — context implies it.** `[y] yaml` already implies toggle (press twice, you're back). `[C] conditions` / `[E] events` follow. The word "toggle" is noise in a hint row where width is at a premium.

**D3 — Toggle-label swap applied consistently where the key is a toggle.** For every mode-toggle keybind (`y`, `C`, `E`), title-bar hint swaps the action word when the mode is active. `[y] describe` / `[C] describe` / `[E] describe` when returning-to-describe is the semantic action. Eliminates the FB-018 cHint dead branch as a side-effect of applying this uniformly. The `[x] delete` keybind is NOT a toggle — no swap. `[Esc] back` is not a toggle — no swap.

**D4 — HelpOverlay sections and status-bar hints included in the pass.** Do not patch only the title bar. Apply the canonical format to `helpoverlay.go:27–62` (every `row.Render(...)` call) and to status-bar hint generation code. Goal: any surface that renders a keybind hint uses the exact same format.

#### Acceptance criteria

1. **[Observable]** DetailPane title-bar hint row renders `[y] yaml` / `[y] describe` / `[C] conditions` / `[C] describe` / `[E] events` / `[E] describe` based on mode. Single lowercase bracket, no "toggle" verb, toggle-label swap for each mode key. Test: all six substring assertions across `m.mode` values `""`, `"yaml"`, `"conditions"`, `"events"`.
2. **[Observable]** HelpOverlay renders `[E] events` and `[C] conditions` (NOT `[Shift+E] events` or `[Shift+C] conditions`). Test: HelpOverlay view assertions at the two substrings.
3. **[Observable]** All existing HelpOverlay hints conform to single-bracket format: `[Enter] select`, `[/] filter`, `[Esc] back/cancel`, `[d] describe`, `[r] refresh`, `[c] switch context`, `[4] activity dashboard`, `[?] toggle help`, `[q] quit`, `[^C] force quit`, `[x] delete resource`. Test: one substring assertion per hint.
4. **[Anti-regression]** FB-018 `cHint` dead branch at `detailview.go:143–146` is removed — replaced by a proper toggle-label swap mirroring `yHint`. Test: press `C` to enter conditions mode, assert title bar reads `[C] describe` (NOT `[C] conditions`).
5. **[Integration / Anti-regression]** FB-019 AC#26 HelpOverlay pane-gating preserved (ShowEventsHint still works). FB-018 AC#22 preserved. FB-017 ShowDeleteHint preserved. Test: existing HelpOverlay tests adjusted only for format, not gating.
6. **[Observable]** Width-narrow truncation path at `detailview.go:168–172` unchanged — title-bar hint row still drops when gap < 2.
7. **[Integration]** `go install ./...` clean; `go test ./internal/tui/...` green after hint-format changes.

**Dependencies:**
- **FB-018 (ACCEPTED), FB-017 (ACCEPTED), FB-019 (ACCEPTED)** — all three ship HelpOverlay hints; all three surfaces change format.
- **FB-024 (PENDING)** — adds `[E] events` hint to the title bar. Sequence FB-024 first (adds the hint), FB-026 second (normalizes format across all hints). Confirmed: merge-order tolerant; if FB-026 ships first, FB-024's new hint lands in the canonical format.

**Maps to:** new proposed **REQ-TUI-044 — Keybind hint format consistency** — every operator-facing keybind hint (title bar, HelpOverlay, status bar) uses `[<key>] <action>` format with no modifier-prefix verbosity, toggle-label swap where the key is a mode toggle.

**Non-goals:**
- **Not a keybind remap.** Keys stay the same; only hint rendering changes.
- **Not a styles/color refactor.** Color tokens unchanged; format is the thesis.
- **Not a status-bar layout redesign.** If status-bar hint density is a separate concern, it's a separate brief.

**Next step:** ux-designer confirms the canonical format copy and the hint-row position for the new `[E] events` entry (coordination with FB-024). Then engineer folds all three surfaces.

---


### FB-027 — Events sub-view polish: width-collapse, row-change staleness, clock-skew, selector-safety

**Status: DEFERRED** — full brief in `docs/tui-backlog-deferred.md`
**Priority: P3**

4 fixes: narrow-width reason-column collapse into `<Reason>: <Message>`, row-change re-entry skeleton placeholder, clock-skew `~now`/`in <dur>` age render, `fields.OneTermEqualSelector` refactor. Plus Count=0 core-v1 renders as `1` with `series.count` fallback.

---

### FB-028 — Events sub-view: deterministic sort (newest-first)

**Status: DEFERRED** — full brief in `docs/tui-backlog-deferred.md`
**Priority: P2**

Client-side sort by `LastTimestamp DESC` at `parseEventRows` construction. Tie-breakers: `FirstTimestamp DESC`, then `Reason` alpha. Unparseable timestamps sink to bottom.

---

### FB-029 — Events sub-view: ownerReferences-based fan-out for composite resources

**Status: DEFERRED** — full brief in `docs/tui-backlog-deferred.md`
**Priority: P2**

Fan out events to children via `ownerReferences` walk (depth cap 3, time-bound 1s, RBAC graceful). Scope indicator on descendant rows. Feature-flag gated.

---

### FB-030 — Events sub-view: First/Last timestamp signal pairing

**Status: DEFERRED** — full brief in `docs/tui-backlog-deferred.md`
**Priority: P2**

Age column shows `First → Last` at wide widths (≥100), e.g. `3d → 15s`. Falls back to Last-only at narrower widths.

---

### FB-031 — Events sub-view: within-table filter

**Status: DEFERRED** — full brief in `docs/tui-backlog-deferred.md`
**Priority: P3**

`/` in events mode activates events-scoped filter bar (with `(events)` label). Matches Type, Reason, Message, InvolvedObject.Name.

---

### FB-032 — SanitizeErrMsg friendly prefix rewrites (Appendix A spin-out)

**Status: DEFERRED** — full brief in `docs/tui-backlog-deferred.md`
**Priority: P3**

14-entry Appendix A rewrite table applied between newline-strip and length-cap stages. Unrecognized prefixes pass through unchanged. Full fixture table in deferred brief.

---

### FB-033 — Command-token accent styling in error card Detail/Title rows

**Status: DEFERRED** — full brief in `docs/tui-backlog-deferred.md`
**Priority: P3**

Tokenizer pass in `RenderErrorBlock` for `[key]` and `` `command` `` tokens in Detail/Title rows. Accent color applied; unmatched/ambiguous tokens stay muted.

---

### FB-034 — Error card preservation across terminal resize

**Status: DEFERRED** — full brief in `docs/tui-backlog-deferred.md`
**Priority: P2**

`recalcLayout` rebuilds `ResourceTableModel` but never replays `SetLoadErr` onto new instance. Fix: replay from `m.loadState`/`m.loadErr` when `lastFailedFetchKind == "list"`. Amends REQ-TUI-041.

---

### FB-035 — Wire `3` key to QuotaDashboard (dead affordance fix)

**Status: ACCEPTED** — test-engineer resubmitted 2026-04-19 with 12 tests (added `TestFB035_Key3_FromActivityDashboardPane_EntersDashboard` for AC#1c + `TestFB035_Key3_FromNavPane_ResourceListVisible_EntersDashboard` for AC#1b); axis-coverage verified by product-experience (Happy AC#1/#2/#1a/#1b/#1c, Repeat-press AC#3/#3a rapid-mash, Anti-behavior AC#4/#5 overlay + FilterBar, Observable AC#6/#6b HelpOverlay row + ordering, Edge AC#8a quota-loading, Anti-regression AC#7 `4`+`t` unchanged). `go test -count=1 -run "TestFB035_" ./internal/tui/...` green. Minor feedback logged (resubmit summary table undercounted actual tests, 10 claimed vs 12 present — non-blocking). User-persona evaluation queued per pipeline protocol. Previous state: **PUSH-BACK 1** — test-engineer submitted 2026-04-19 with 11 tests; product-experience rejected at pre-submission gate (axis gap): AC#1c (ActivityDashboardPane → QuotaDashboardPane) is REQUIRED per spec §8 but only AC#1a (DetailPane) was pinned. Pre-PUSH-BACK state: **IN-PROGRESS** — engineer implementation landed 2026-04-19; `case "3":` added app-global to `model.go` mirroring `4`'s pattern, HelpOverlay row added (`[3] quota dashboard`). Previous state: **PENDING ENGINEER** — ux-designer spec delivered 2026-04-19 at `docs/tui-ux-specs/fb-035-quota-dashboard-key-wiring.md`. Design decisions: pane-scope confirmed **app-global** (mirror `4`'s pattern at `model.go:1013–1024`; `3` transitions from any non-overlay/non-FilterBar pane to `QuotaDashboardPane`, toggle-back to NavPane); AC#1a pinned in spec §8 with explicit test cases for DetailPane / HistoryPane / DiffPane / ActivityDashboardPane; AC#6a/b pin HelpOverlay row placement + ordering; AC#1b pins NavPane-without-welcome-panel state; AC#3a pins rapid-toggle convergence. Previous state: **PENDING UX-DESIGNER** — filed 2026-04-19 after user reported pressing `3` from the welcome panel does nothing. Queued immediately after FB-024 (ahead of FB-025) per team-lead direction — High priority because the TUI advertises a feature that doesn't work.

**Priority: High** (broken affordance — the TUI advertises a feature that doesn't work)

#### User problem

The platform health welcome panel renders `(press 3 for full dashboard)` (`resourcetable.go:349`) whenever the top-3 quota summary is visible. Pressing `3` does nothing — there is no `case "3":` handler anywhere in `model.go`. The `QuotaDashboardPane` (`model.go:30`) exists and is fully implemented but is only reachable by navigating to the `allowancebuckets` resource type, not via a direct key shortcut. Operators see the call-to-action, press the advertised key, and get no response — a broken promise that erodes trust in the TUI's key hints.

#### Root cause

`resourcetable.go:349` was written when the `3` key was planned to navigate to `QuotaDashboardPane`, but the key handler was never wired in `model.go`. `4` navigates to `ActivityDashboardPane` (model.go:1013); `3` should navigate to `QuotaDashboardPane` symmetrically from the welcome panel and TABLE pane.

#### Proposed interaction

- `3` from the welcome panel (NavPane, no resource type selected) or from TablePane → navigate to `QuotaDashboardPane`, same as the existing `allowancebuckets` navigation path (`model.go:929-937`).
- `3` from `QuotaDashboardPane` → return to NavPane (toggle, matching the `4`/ActivityDashboard pattern at model.go:929/1013).
- `3` is a no-op in all overlay states (CtxSwitcher, HelpOverlay) and in FilterBar mode (matching the existing numeric-key guard).
- HelpOverlay and status bar hint strip should surface `[3] quota dashboard` alongside `[4] activity`.

#### Acceptance criteria

Axis tags: `[Happy]`, `[Repeat-press]`, `[Anti-behavior]`, `[Observable]`, `[Integration]`, `[Anti-regression]`.

1. **[Happy / Observable]** From the welcome panel, pressing `3` navigates to `QuotaDashboardPane`. Test: NavPane with welcome panel visible, press `3`, assert `activePane == QuotaDashboardPane`.
2. **[Happy / Observable]** From `TablePane`, pressing `3` navigates to `QuotaDashboardPane`. Test: TablePane active, press `3`, assert `activePane == QuotaDashboardPane`.
3. **[Repeat-press]** Pressing `3` from `QuotaDashboardPane` returns to NavPane (toggle). Test: enter QuotaDashboard via `3`, press `3` again, assert `activePane == NavPane`.
4. **[Anti-behavior]** `3` is a no-op when CtxSwitcher or HelpOverlay is open. Test: each overlay active, press `3`, assert `activePane` unchanged.
5. **[Anti-behavior]** `3` is a no-op in FilterBar mode. Test: FilterBar active, press `3`, assert pane unchanged.
6. **[Observable]** `(press 3 for full dashboard)` hint in welcome panel remains accurate after wiring (not removed). Test: welcome panel rendered with quota summary visible, assert hint substring present.
7. **[Integration / Anti-regression]** `4` / ActivityDashboard navigation unaffected. `allowancebuckets` navigation to QuotaDashboard unaffected. Test: existing `4`-key and allowancebuckets tests pass unchanged.
8. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** None — `QuotaDashboardPane` is fully implemented; this is key-wiring only.

**Maps to:** REQ-TUI-019 (keyboard navigation) — extends the existing numeric-key pane-navigation pattern.

**Non-goals:**
- Not changing the `QuotaDashboardPane` behavior or layout.
- Not adding `3` to panes other than NavPane/TablePane/QuotaDashboardPane.
- Not moving quota dashboard access away from the `allowancebuckets` navigation path.

---

### FB-036 — Remove recon age and claim count from quota views

**Status: ACCEPTED** — test-engineer resubmitted 2026-04-19 with `TestFB036_CompactFormTree_NoOutOfSync` at `quotabanner_test.go:547` (3 sub-tests: recent_recon, stale_recon_20m, zero_recon; width=40 forces `renderBannerTreeRowCompact` path; asserts absence of both `"out of sync"` and `"recon"`). Verified at line 547; `go test -count=1 -run "TestFB036_" ./internal/tui/components/...` green. Full axis-coverage: Observable (banner full w=200 no `"recon"`/`"stale"`; banner tree w=200 no `"recon"`; banner compact w=40 no `"out of sync"`/`"recon"`; block no `"claims:"`/`"reconciled"`; dashboard no `"recon"`); Width-band (full-form w=200 + compact-form w=40 both pinned); Anti-behavior (bar/counts/suffix present w=200; compact w=40 counts present); Edge (w=60 mode-flip stays full; zero LastRecon no em-dash; block height=2). Note on submitter/reviewer coordination: test-engineer claimed the compact test was "added in the previous round" but my grep at push-back 2 didn't show it — likely a between-edit grep race; moot since the test is correctly in place now. User-persona evaluation queued. Previous state: **PUSH-BACK 1** — test-engineer submitted 2026-04-19 with 11 tests; product-experience rejected at pre-submission gate (axis gap + stale note): (1) AC#5a negative assertion covers full-form tree only (`TestFB036_FullFormTree_NoOutOfSync`) — compact-form tree path missing a mirroring negative assertion; (2) submission note "pending product-experience confirmation" was stale. Previous state: **IN-PROGRESS** — engineer implementation landed 2026-04-19; recon age + claim count surfaces removed from QuotaBanner (`bannerReconCell` deleted), QuotaBlock (`renderStatsLine` deleted, `reconciliationRow` removed), QuotaDashboard (`buildReconCell` call removed); `(out of sync)` compact-path cleanup completed (`renderBannerTreeRowCompact` no longer emits marker at `quotabanner.go:269`; `treeOutOfSync` call sites removed; dead `buildReconCell` removed). **Engineer authorization (2026-04-19):** YES, delete `(out of sync)` from the compact banner tree row path — same operator-signal-vs-platform-signal rejection as the recon cell itself. The marker was only legible when visually coupled to recon; isolating it creates a cryptic stray label. Full sweep: remove compact-form render + `treeOutOfSync` call sites + now-dead `buildReconCell`. Previous state: **PENDING ENGINEER** — ux-designer spec delivered 2026-04-19 at `docs/tui-ux-specs/fb-036-quota-recon-claims-removal.md`. Design decisions: `(out of sync)` marker confirmed **deleted alongside recon cell** — same quota-system-health-signal rejection thesis (rationale: marker was only legible when visually coupled to recon; isolating it would create a cryptic stray label; if drift detection matters later, it gets its own brief with its own surface); width-band matrix (§4) pins w=200/120/80/60/40 behavior across all three surfaces (banner/block/dashboard); AC#7a (w=60 mode flip — highest-risk reflow), AC#4a (block height 3→2 collapse), AC#5a (out-of-sync removal absence), AC#7b/c, AC#1a (empty-state) added in §8. Previous state: **PENDING UX-DESIGNER** — filed 2026-04-19 from user feedback: "We don't need to show that or the number of claims." Queued after FB-035 (and ahead of FB-025) per team-lead direction.

**Priority: Medium** (visual clutter; operator-irrelevant internal system metadata)

#### User problem

Two pieces of quota system internals are surfaced in every quota view:

1. **Recon age** (`recon 2m`, `recon 18m (stale)`) — the time since the quota controller last aggregated this bucket. Operators don't act on this; it reflects quota system health, not resource quota health.
2. **Claim count** (`claims: 3`) — the number of granted `ResourceClaims` consuming quota. Operators care about allocated vs limit, not how many individual claims make up the allocation.

Both fields appear across three surfaces:
- **QuotaBanner** (`quotabanner.go`) — `bannerReconCell` appended to every banner line and tree row
- **QuotaBlock** (`quotablock.go`) — `buildReconCell` on tree rows + `reconciliationRow` footer (`claims: N   reconciled Xm ago`)
- **QuotaDashboard** (`quotadashboard.go`) — `buildReconCell` on every dashboard row

#### Proposed change

Remove both fields from all three surfaces. No replacement — the space reclaimed by removing recon + claims should be absorbed by the existing bar/counts/suffix layout (overhead calculations updated accordingly).

**Specific removal sites:**
- `quotabanner.go`: delete `bannerReconCell` function + all call sites (lines ~133, ~157, ~196–198, ~264, ~266); remove recon overhead from `renderBannerLineWithNames` and `renderBannerTree`
- `quotablock.go`: delete `buildReconCell` calls on parent/child rows (lines ~66, ~90); delete `reconciliationRow` function and its call site (line ~267–275)
- `quotadashboard.go`: delete `buildReconCell` call (line ~254–255); remove recon overhead from bar-width calculation (line ~233)

#### Acceptance criteria

Axis tags: `[Happy]`, `[Anti-behavior]`, `[Observable]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable]** QuotaBanner full-form View() contains no `"recon"` substring. Test: banner with `LastReconciliation` set, assert `stripANSI(View())` does not contain `"recon"`.
2. **[Observable]** QuotaBanner stale path (>15m) contains no `"stale"` substring. Test: bucket with `LastReconciliation` 20m ago, assert no `"stale"` in View().
3. **[Observable]** QuotaBanner tree-view rows contain no `"recon"` substring. Test: tree banner with parent+child, assert no `"recon"` in any line.
4. **[Observable]** QuotaBlock View() contains no `"claims:"` substring. Test: block with `ClaimCount > 0`, assert no `"claims:"`.
5. **[Observable]** QuotaBlock View() contains no `"reconciled"` substring. Test: block with `LastReconciliation` set, assert no `"reconciled"`.
6. **[Observable]** QuotaDashboard rows contain no `"recon"` substring. Test: dashboard with buckets, assert no `"recon"` in View().
7. **[Anti-behavior]** Existing bar/counts/suffix still render correctly after recon overhead removed. Test: banner at width=200, assert `"/ "` counts and bar glyphs present.
8. **[Anti-regression]** `go test ./internal/tui/...` green; `go install ./...` compiles.

**Dependencies:** None — pure removal, no new components.

**Maps to:** No new REQ — removes display of `AllowanceBucket.Status.LastReconciliation` and `AllowanceBucket.Status.ClaimCount` from all quota views.

---

### FB-037 — DetailPane error-first race: re-render on events-loaded + error-block action hints

**Status: ACCEPTED** — 2026-04-19 by product-experience. Single-line change at `internal/tui/model.go:321` (`if m.eventsMode` → `if m.eventsMode || m.loadState == data.LoadStateError || m.describeRaw == nil`). Spec at `docs/tui-ux-specs/fb-037-detail-pane-error-first-race.md` explicitly ratified the "delegate to FB-024 placeholder" alternative for Part B (§2 line 107: "This spec chooses the alternative") making AC#4/7-as-written vacuous-by-design; the Part B invariant (`loadState==error AND events!=nil always resolves to placeholder, never error block`) is asserted by `TestFB037_ErrorState_EventsLoaded_PlaceholderPreemptsErrorBlock`. 7 new tests at `model_test.go:8951–9068`: ErrorFirst_EventsArrive_ReRendersToPlaceholder (AC#1 input-changed, `detail.View()` assertion), EventsFirstRace_PlaceholderShows_AntiRegression (AC#2), BothFail_OrderInvariant_ErrorBlockShows (AC#3, `detail.View()`), ErrorState_EventsLoaded_PlaceholderPreemptsErrorBlock (AC#4 restated), ErrorBlock_EventsNil_NoEHint (AC#5 anti-behavior), EFromPlaceholder_EntersEventsMode (AC#6), RepeatE_FromPlaceholder_TogglesEventsMode (AC#7 substitution — adds repeat-press axis coverage). Anti-regression: all FB-024/FB-022 tests green; `go install ./...` + `go test ./internal/tui/...` pass. User-persona eval queued. Previously: filed 2026-04-19 from user-persona FB-024 follow-up (P2-1 + P2-2 bundled: same causal chain — error-first fetch order produces error-block state, then the error-block's action hints actively misdirect operators away from the events that would explain the describe failure).

**Priority: P2** (FB-024's stated P1 use-case — "operator needs events to explain describe failure" — silently breaks under network race; found by stress-testing the brief's core scenario)

#### User problem

FB-024 added a placeholder ("Describe unavailable — only events loaded.") that directs the operator to events when describe fails but events succeed. The fix works when `EventsLoadedMsg` arrives before `LoadErrorMsg`. It **silently fails** when `LoadErrorMsg` arrives first — a network-race order the operator has no control over.

**Symptom A (P2-1):** With describe-RBAC-partial resources, two operators debugging the same resource see different initial DetailPane content depending on fetch order:
- Events-first: FB-024 placeholder with `(press E)` affordance — "you have a path forward" signal.
- Error-first: FB-022 error block with `[Esc] back to table` action — "dead end, give up" signal.

The operator pressed `d` on the same resource in both cases. The only variable is the network race.

**Symptom B (P2-2):** In the error-first case, the FB-022 error block's action hints (`[Esc] back to table`) actively steer the operator away from events — which **are loaded and would contain the RBAC denial reason**. The only recovery path is the small `[E] events` in the title-bar hint row, which:
- Is a passive corner-of-screen signal competing with explicit action prose in the error block, AND
- Is dropped entirely at narrow widths (detailview.go:157–176 drops right-text when `gap < 2`).

The stated P1 of FB-024 — "operator needs events to explain describe failure" — is defeated on both symptoms because the UX surface pushes the operator toward "back to table" at the exact moment events would explain the failure.

#### Root cause

**Symptom A:** `model.go:311–322` gates re-render on `m.eventsMode`:
```go
case data.EventsLoadedMsg:
    m.eventsLoading = false
    // ... stash events / eventsErr ...
    if m.eventsMode {
        m.detail.SetContent(m.buildDetailContent())
    }
```
Operator hasn't pressed `E` yet → `eventsMode == false` → events are stashed in `m.events` but the viewport is never re-rendered. `buildDetailContent`'s FB-024 placeholder branch (`model.go:1706`) would match on the next render (`describeRaw == nil && events != nil && !eventsMode`), but that next render never comes until the operator presses a key.

**Symptom B:** FB-022's error block action hints are determined by `components.ActionsForSeverity(sev, "back to table")` at `model.go:1724`. The helper has no awareness of whether `events != nil`, so it renders the same actions regardless of whether events are a recoverable adjacent data source.

#### Proposed change

**Part A — re-render on EventsLoadedMsg when on DetailPane.** Remove the `m.eventsMode` gate at `model.go:320`. Re-render unconditionally when the operator is on DetailPane (`activePane == DetailPane`) so any `buildDetailContent` branch that benefits from events-now-loaded (placeholder, error-block-with-events-action) refreshes immediately.

Simpler alternative: re-render when `loadState == LoadStateError` OR `describeRaw == nil`. Covers the race without touching eventsMode-active branches.

**Part B — error-block actions include `[E] view events` when events are available.** Extend the `model.go:1710–1727` error-block render branch to pass an `eventsAvailable bool` flag into a new `components.ActionsForErrorWithEvents(sev, eventsAvailable, backLabel)` helper. When `eventsAvailable && sev == Warning` (the describe-RBAC-partial case), actions should surface `[E] view events` ahead of `[Esc] back to table`.

Alternative: have the error block recognize the events-loaded state and delegate to the FB-024 placeholder entirely (simpler, fewer moving parts — the placeholder already exists for this exact scenario).

#### Acceptance criteria

Axis tags: `[Happy]`, `[Repeat-press]`, `[Input-changed]`, `[Anti-behavior]`, `[Observable]`, `[Integration]`, `[Anti-regression]`.

1. **[Observable / Input-changed]** Error-first race: describe fails, operator on DetailPane, then events arrive → viewport re-renders to FB-024 placeholder (not stuck on error block). Test: set `loadState == LoadStateError`, `lastFailedFetchKind == "describe"`, `events == nil`, render → assert error-block title. Dispatch `EventsLoadedMsg{Events: [...]}`. Re-render → assert `stripANSI(detail.View())` contains `"Describe unavailable"` and does **not** contain error-block title.
2. **[Observable / Input-changed]** Events-first race: events arrive before describe error → placeholder renders (already covered by FB-024 AC#1, but pin as anti-regression here).
3. **[Observable]** Both-fail order-invariant: describe fails, then events fail → error block renders (events failure means nothing to switch to). Test: error block's title substring present; no FB-024 placeholder.
4. **[Observable / Input-changed]** When error block renders AND `events != nil`, action hints include `[E] view events` before `[Esc] back to table`. Test: set `loadState==error`, `events: [...]`, assert rendered actions contain substring `"[E]"` and `"view events"`.
5. **[Anti-behavior]** When error block renders AND `events == nil`, action hints do **not** include `[E]` anywhere. Test: `events == nil`, assert rendered actions omit `"[E]"`.
6. **[Integration]** Pressing `E` from the FB-024 placeholder (race recovery path) enters events mode and renders the events table. Test: after Part A re-render, press `E`, assert `eventsMode == true` and events table content renders.
7. **[Repeat-press]** Pressing `E` from error-block-with-events-action (when Part B lands) also enters events mode. Test: error block rendered with `events != nil`, press `E`, assert events mode.
8. **[Anti-regression]** FB-024 tests green: TestFB024_BothFailed_ErrorBlock_NotPlaceholder, TestFB024_RepeatToggle_CacheHit_NoRedispatch, TestFB024_RapidEPress_InFlight_NoExtraDispatch, TestFB024_EAfterError_Redispatch_OnceOnly, TestFB024_TitleBar_EHint/all sub-tests.
9. **[Anti-regression]** FB-022 tests green (error block precedence in both-failed case).
10. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-024 ACCEPTED (2026-04-19). FB-022 v2 ACCEPTED. No new infrastructure.

**Maps to:** Amendment to REQ-TUI-041 (DetailPane render reconciliation under fetch race) and REQ-TUI-039 (FB-024 describe-unavailable placeholder) — makes FB-024's guarantee order-invariant.

**Non-goals:**
- Not changing describe or events fetch timing/ordering (network-race is a given).
- Not expanding the placeholder to cover events-failed-describe-failed (that case correctly shows error block — see AC#3).
- Not refactoring the FB-022 `ErrorBlock` component — additive action hint only (or a wrapper-level swap to placeholder).
- Not addressing HelpOverlay `[Shift+E]` vs title-bar `[E]` (that's FB-026).

---

### FB-038 — DetailPane empty-viewport oscillation on rapid-E mash during in-flight fetch

**Status: ACCEPTED 2026-04-20** — Option A implementation verified at `model.go:2046` (3-line pre-check in `buildDetailContent()` returning muted `"Loading…"` placeholder when `describeRaw == nil && events == nil && !eventsMode && eventsLoading`). 3 AC-indexed tests green: AC1 [Observable] `TestFB038_AC1_InFlight_ViewContainsLoading` (asserts `stripANSI(m.View())` contains `"Loading"` during in-flight state); AC2 [Repeat-press] `TestFB038_AC2_RapidEPress_NoBlankBody` (4 rapid E presses — presses 2/4 at eventsMode=false are the regression check, both non-blank + contain `"Loading"`); AC3 [Anti-behavior] `TestFB038_AC3_PreCheck_Inert_AfterEventsLoaded` (after `EventsLoadedMsg`, View() does NOT contain `"Loading…"` AND DOES contain `"Describe unavailable"` — FB-024 placeholder correctly takes over). AC4 N/A per spec (Option B not chosen). AC5 [Anti-regression] FB-024 rapid-E dispatch guard preserved via full suite. AC6 [Integration] `go install ./...` clean + full `go test ./internal/tui/...` green. Test-engineer gate-check initial FAIL on AC1/AC3 (Observable/Anti-behavior asserting state-only); engineer rework added `stripANSI(appM.View())` assertions; re-gate PASS. Product-experience independent verification: `go test ./internal/tui/ -run 'TestFB038' -count=1 -v` all 3 PASS. **Resolves FB-024 follow-up P2-3.** FB-024 D1 relaxation preserved (operators can still pre-toggle to events mode during loading — render side is the defect, input side correct). **Next:** engineer commits FB-038 to `feat/console` with product-prose message describing the no-more-blank-body fix + push. **Prior Status: PENDING ENGINEER 2026-04-20** — spec delivered 2026-04-20 by ux-designer at `docs/tui-ux-specs/fb-038-detail-empty-viewport-oscillation.md` (Option A: 3-line render-layer pre-check; Option B rejected because it removes FB-024's D1 relaxation). **Original Status: PENDING UX-DESIGNER** — filed 2026-04-19 from user-persona FB-024 follow-up (P2-3: rapid-E pressed during paired-fetch in-flight produces alternating spinner/blank viewport; FB-024's re-dispatch guard test name implies the rapid-E case was considered, but coverage is for dispatch count, not rendered output).

**Priority: P2** (reads as a rendering bug; the body oscillates between spinner and blank while the title bar shows steady "loading…" — reinforces a "TUI is broken" impression on the very surface FB-024 was hardening)

#### User problem

An operator presses `d` on a resource and immediately mashes `E` while both describe and events are still in-flight (FB-024's D1 relaxation admits the E press because `eventsLoading == true`). The resulting body alternates between the events-mode spinner ("Loading events…") and a blank empty string on every press.

**Press-by-press:**
- Press 1 (`E`): `eventsMode = true` → `buildDetailContent` → `RenderEventsTable(events=nil, loading=true, err=nil, ...)` → "Loading events…" spinner.
- Press 2 (`E`): `eventsMode = false` → `buildDetailContent` falls through: `describeRaw == nil`, `events == nil`, `yamlMode/conditionsMode/eventsMode` all false, `loadState != error` (still loading), `content == ""` → returns `""`.
- Press 3: spinner again.
- Press 4: blank again.

The title bar remains steady ("loading…"), so it's not catastrophic, but the oscillation reads as a rendering bug.

#### Root cause

`model.go:1749–1752` in `buildDetailContent`:
```go
content := m.describeContent
if len(m.buckets) == 0 || content == "" {
    return content  // returns "" during initial in-flight
}
```

No branch in `buildDetailContent` handles the "nothing-rendered-yet but fetches in-flight" state when `eventsMode == false`. The FB-024 placeholder requires `events != nil`; FB-005 error block requires `loadState == error`; all mode flags are false during initial load. Falls through to empty string.

#### Proposed change

**Option A (defensive):** Add a pre-check branch at the top of `buildDetailContent`:
```go
if m.describeRaw == nil && m.events == nil && (m.loadState == LoadStateLoading || m.eventsLoading) {
    return lipgloss.NewStyle().Foreground(styles.Muted).Render("Loading…")
}
```
Or keep the viewport rendering the spinner output unconditionally when either fetch is in-flight and no data has arrived yet — regardless of `eventsMode`.

**Option B (input-gating):** Reject the outer E-guard when nothing is rendered yet. FB-024's D1 currently admits `E` when `m.eventsLoading == true`; tighten to admit only when `events != nil` OR `loadState == LoadStateError` (i.e., there is *something* to switch between).

Designer to pick; Option A is less restrictive to the operator (they can still press E), Option B prevents the flicker at the input layer but reduces the operator's ability to pre-toggle during load.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Repeat-press]`, `[Anti-behavior]`, `[Integration]`.

1. **[Observable]** During initial in-flight fetch (describeRaw=nil, events=nil, eventsLoading=true, loadState=loading), `buildDetailContent` never returns an empty string. Test: construct state, call `buildDetailContent()`, assert `len(result) > 0`.
2. **[Repeat-press]** Pressing `E` 1, 2, 3, 4 times in rapid succession during in-flight fetch produces stable body content (either always-spinner or always-loading-placeholder — implementation choice, but never blank). Test: programmatically inject 4 `E` keypresses, assert `stripANSI(detail.View())` contains non-empty content after each press.
3. **[Anti-behavior]** Once at least one fetch resolves (events or describe), the normal FB-024 placeholder / error-block / events-mode / describe-mode branches still apply. Test: dispatch `EventsLoadedMsg` after 4-press mash; assert subsequent E press enters proper events-mode render.
4. **[Observable]** If Option B (input-gating) is chosen: pressing `E` during in-flight-with-nothing-rendered is a no-op (`eventsMode` stays false). Test: press E four times, assert `eventsMode == false` throughout. (AC contingent on spec choice.)
5. **[Anti-regression]** FB-024 TestFB024_RapidEPress_InFlight_NoExtraDispatch still passes (dispatch count ≤ 1) with the body-rendering fix.
6. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-024 ACCEPTED. No new infrastructure.

**Maps to:** Amendment to REQ-TUI-040 (events mode render contract — include "events mode with no data yet during in-flight fetch" case).

**Non-goals:**
- Not re-opening FB-024's dispatch-guard behavior (that's the correct design; this brief addresses the *render* side only).
- Not changing the events table's own in-flight rendering.

---

### FB-039 — DetailPane placeholder copy + title-bar mode coherence for describe-unavailable state

**Status: PENDING UX-DESIGNER** — filed 2026-04-19 from user-persona FB-024 follow-up (P3-1 + P3-2 bundled: same thesis — when describe is unavailable, the DetailPane body and title-bar header should tell a single coherent story, and the body should be action-directive rather than state-descriptive).

**Priority: P3** (polish after the structural FB-037/FB-038 fixes; operator isn't blocked but reads the surface as incoherent)

#### User problem

**Sub-problem A (P3-1):** FB-024 placeholder copy reads "Describe unavailable — only events loaded." This is state-descriptive. It tells the operator what's true, not what to do. The `[E] events` affordance is only in the title-bar hint row, which may be truncated at narrow widths or suppressed during loading (FB-040). An operator who lands on the placeholder at a narrow terminal sees a dead-end message with no affordance.

**Sub-problem B (P3-2):** When the placeholder renders, the title-bar mode label still reads `describe` (accent+bold) — because the DetailPane's `mode` field is still `"describe"`. Header says "we're in describe mode"; body says "describe unavailable." For a half-second operators wonder if stale content flashed. The surface reads as "we're in describe mode but describe is unavailable, which mode am I in?"

#### Proposed change

**A (copy):** Update the placeholder copy to include the action:
- Current: `"Describe unavailable — only events loaded."`
- Proposed: `"Describe unavailable. Press [E] to view loaded events."`

Mirrors kubectl/k9s conventions (hint + keybind inline). Makes the message its own recovery path; operator doesn't depend on the title-bar hint row surviving width collapse.

**B (header):** When the placeholder branch renders, either:
- Option B1: suppress the `describe` mode label in the title bar (set mode to `""` or a distinct `"unavailable"` state).
- Option B2: keep the mode label but soften the copy to match ("Describe unavailable for this resource — press `E` for events").

Designer to pick. B1 avoids the apparent contradiction; B2 keeps mode-label invariant (simpler model state).

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`.

1. **[Observable]** Placeholder copy contains the keybind inline: `stripANSI(buildDetailContent())` contains `"Press [E]"` (or the chosen final phrasing — designer to ratify exact string).
2. **[Observable]** Placeholder copy does NOT contain the old state-descriptive-only phrasing: `stripANSI(...)` does not contain `"only events loaded"` without also containing `"[E]"` (anti-regression on drift back to bare state-describe).
3. **[Observable / Input-changed]** When placeholder renders (describeRaw=nil, events!=nil), title-bar contents match the body's story — either no `describe` mode label (B1) or softened body copy that acknowledges the unavailable state (B2). Test: assert `stripANSI(detail.View())` does not contain both `"describe"` mode label AND `"Describe unavailable"` body-absolutism simultaneously.
4. **[Anti-regression]** FB-024 placeholder still wins over FB-022 error block when `events != nil` and describe failed. Test: TestFB024_* suite stays green; placeholder render path precedence unchanged.
5. **[Anti-regression]** FB-018 conditions-mode label and FB-019 events-mode label still render correctly in the title bar. Test: existing detailview tests green.
6. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-024 ACCEPTED.

**Maps to:** Amendment to REQ-TUI-039 (describe-unavailable placeholder) — copy + header coherence rules.

**Non-goals:**
- Not changing when the placeholder triggers (that's FB-037's scope).
- Not addressing the HelpOverlay `[Shift+E]` vs title-bar `[E]` notation mismatch (FB-026).
- Not changing FB-025 "Press [r] to refresh" pattern (separate brief).

---

### FB-040 — Title-bar keybind hints suppressed during DetailPane loading (discoverability gap)

**Status: PENDING UX-DESIGNER** — filed 2026-04-19 from user-persona FB-024 follow-up (P3-4: first-time operators discovering `[E] events` during initial paired-fetch have no affordance because `detailview.go:138` gates right-text on `!m.loading`; HelpOverlay fallback has the `[Shift+E]` notation mismatch per FB-026).

**Priority: P3** (seasoned users already know the keybind; impacts first-time discoverability in the exact window FB-024 was designed to harden — incident-triage mid-load)

#### User problem

An operator presses `d` on a resource and — while waiting for describe/events to return — looks for the `[E]` affordance to learn the keymap. During the initial paired-fetch, `m.loading == true` and the title-bar right-text (`[y] yaml  [C] conditions  [E] events  [x] delete  [Esc] back`) is suppressed entirely per `detailview.go:138`. No affordance visible.

If the operator hits `?` to consult the HelpOverlay, the overlay shows `[Shift+E] events` (helpoverlay.go:43) — a different notation from the title-bar `[E]` convention (this is the FB-026 issue, separate brief, but it compounds the discoverability failure).

Result: the first-time operator learning FB-019/FB-024 in an incident-triage context has no affordance during the exact window the feature was designed for.

#### Root cause

`detailview.go:133–152`:
```go
if m.loading {
    leftText += muted.Render("  ") + m.spinner.View() + muted.Render(" loading…")
}
var rightText string
if !m.loading {  // <-- gates hints on loading == false
    // ... builds "[y] yaml  [C] conditions  [E] events  [x] delete  [Esc] back"
}
```

`m.loading` is true during the initial paired-fetch. Hints appear only after fetches resolve — too late to aid discoverability during the initial wait.

#### Proposed change

Render the keybind-hint right-text during loading too. The left-text `Kind / name  describe  ⟳ loading…` is short; at wide terminals (`w ≥ 90`) there is ample room for both. At narrow widths, existing width-collapse logic (`detailview.go:154–176`) already drops or truncates; no additional collapse rule needed.

Remove the `if !m.loading { ... }` guard at `detailview.go:138` (keep the text build, just unconditional). Width-collapse branches at `:154` onward already handle the narrow case.

Alternative: always render hints unconditionally; keep `loading…` on left-text side only. Same effect, simpler diff.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`.

1. **[Observable]** During initial fetch (loading=true, w=100), title bar includes `[E] events` substring. Test: construct DetailPane model with `loading=true`, width=100, assert `stripANSI(detailview.View())` contains `"[E] events"`.
2. **[Observable]** During initial fetch at narrow widths (w=40), hints drop entirely per existing collapse rule (no regression on narrow-width collapse). Test: w=40 + loading=true, assert no `"[E]"` (space-preserving fallback).
3. **[Input-changed]** Transition from `loading=true` → `loading=false` (fetch resolved) does not regress hint visibility at w=100. Test: render at loading=true, then loading=false, assert both contain `"[E] events"`.
4. **[Observable]** `loading…` indicator remains on left-text side during load. Test: assert `stripANSI(detailview.View())` contains `"loading"` during loading=true.
5. **[Anti-regression]** FB-024 TitleBar tests (TestFB024_TitleBar_EHint and sub-tests) green — existing `[E] events` / `[E] describe` assertions hold post-fetch.
6. **[Anti-regression]** FB-017 `[x] delete`, FB-018 `[C] conditions`, FB-019 hints all still render in both loading and loaded states.
7. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-024 ACCEPTED. Doesn't block on FB-026 (HelpOverlay notation fix) — orthogonal surface.

**Maps to:** Amendment to REQ-TUI-038 / REQ-TUI-040 title-bar hint contracts (hints visible during loading).

**Non-goals:**
- Not changing HelpOverlay `[Shift+E]` / `[E]` notation (FB-026).
- Not adding new keybinds to the hint row.
- Not changing the `loading…` spinner behavior on left-text.

---

### FB-041 — Escape always returns to welcome dashboard

**Status: ACCEPTED** — 2026-04-19 by product-experience. Two files: `resourcetable.go` (`forceDashboard bool` at line 44 + View() condition at line 104 `typeName == "" || forceDashboard` + `SetForceDashboard` at line 576) and `model.go` (`showDashboard bool` on AppModel; NavPane Esc handler; clear sites on Enter/Tab/Shift-Tab/"3"/"4"/ContextSwitchedMsg). 10 new tests at `model_test.go:9138–9364` covering all 8 brief ACs + all 5 spec §7 axis pins: `Esc_NavPane_WithTableLoaded_ShowsDashboard` (AC#1+2), `Esc_NavPane_SecondPress_IsNoop` (AC#3, **byte-identical View() comparison** — strong input-changed defense), `Esc_ThenEnter_ClearsDashboard` + `Esc_ThenTab_ClearsDashboard` + `JKey_DuringDashboard_UpdatesLeftBlock` + `Startup_Esc_IsNoop_NoTableTypeName` + `TableRows_PreservedInMemory_AfterEsc` (§7 pins), `TablePane_Esc_NoShowDashboard` (AC#5), `DetailPane_Esc_GoesToTablePane_NotNavPane` (AC#4), `Overlay_Esc_DismissesOverlay_NoShowDashboard` (AC#7); AC#6 (filter-mode Esc) satisfied by anti-regression statement (existing filter-escape tests green). AC#8 integration green. Submitter's axis-coverage table mapped all 8 ACs + exceeded brief with 5 §7 pins — no gaps. User-persona eval queued. **Unblocks FB-042** (welcome-dashboard enhancement depends on Esc-to-dashboard return path — now queued for ux-designer).
**Priority: High** — navigating back is fundamental; dead-end escape kills session flow

#### User problem

Pressing Escape from TablePane returns to NavPane (sidebar cursor focus), and pressing Escape from DetailPane returns to TablePane. There is no single keystroke that returns the operator to the welcome dashboard from anywhere in the navigation stack. Operators who want to re-orient to the platform-health overview must Tab back to the sidebar and then move focus away from the table — multiple keystrokes, non-obvious flow.

The welcome dashboard is the starting surface and should be recoverable at any depth with a single Escape from the sidebar, creating a clear mental model: sidebar is the anchor, the dashboard is always one Escape away.

#### Proposed interaction

When the cursor is in NavPane (sidebar is focused) and the operator presses Escape, the right-hand panel returns to the welcome/landing dashboard view — even if a resource type was previously selected and a table is loaded.

This means:

- **Esc in DetailPane** → TablePane (unchanged — existing behavior).
- **Esc in TablePane** → NavPane / sidebar (unchanged — existing behavior).
- **Esc in NavPane (sidebar focused)** → welcome dashboard rendered in right pane; no pane transition (sidebar retains focus); any loaded table is still cached.

The effect is: pressing Escape enough times always walks you back to the dashboard.

```
DetailPane  --[Esc]--> TablePane --[Esc]--> NavPane (sidebar)
NavPane (sidebar, was showing table) --[Esc]--> NavPane (sidebar, shows dashboard)
```

At the dashboard, Esc is a no-op (already at top level).

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`.

1. **[Observable]** From NavPane with a resource selected (table visible in right pane), pressing Escape renders the welcome dashboard in the right pane. Test: construct AppModel with `NavPane` active + resource selected, send Esc key, assert `stripANSI(model.View())` contains `"Welcome,"` and `"Platform health"`.
2. **[Observable]** After the Esc-to-dashboard transition, sidebar still has focus and cursor position is unchanged. Test: assert pane is still `NavPane`; sidebar cursor index unchanged.
3. **[Input-changed]** From NavPane already showing the dashboard (no resource selected), pressing Escape is a no-op. Test: send two consecutive Esc keys; second press does not change view output.
4. **[Anti-regression]** Esc from DetailPane still returns to TablePane (not dashboard). Test: assert pane transitions `DetailPane → TablePane`, not `DetailPane → NavPane`.
5. **[Anti-regression]** Esc from TablePane still returns to NavPane with table visible (not dashboard). Test: assert pane = `NavPane`, right pane still shows table (not welcome panel).
6. **[Anti-regression]** Esc in filter mode still clears filter and stays in TablePane (filter escape takes priority). Test: existing filter-escape tests green.
7. **[Anti-regression]** Overlays (CtxSwitcher, HelpOverlay) still dismiss on Esc and do not fall through to NavPane dashboard transition. Test: existing overlay-escape tests green.
8. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-015 ACCEPTED (welcome dashboard exists).

**Maps to:** REQ-TUI-007 (welcome panel) — Esc-to-dashboard is the return path.

**Non-goals:**
- Not adding a dedicated "home" key (Esc is sufficient).
- Not clearing sidebar filter or resetting cursor on dashboard return.
- Not changing the `Tab` pane-cycling behavior.

---

### FB-042 — Enhanced welcome dashboard: actionable, engaging, context-aware

**Status: ACCEPTED 2026-04-19 by product-experience** — test-engineer rework closed all three AC gaps with a complete axis-coverage table. Verified:
- AC1 (`"All clear"`) → `TestFB042_HealthSummary_AllClear` at `internal/tui/components/resourcetable_test.go:1043`
- AC6 (condition-kind AttentionItem with `⚠` icon) → `TestFB042_Attention_ConditionItem_RendersLabelAndDetail` at `resourcetable_test.go:1060`
- AC8 (quick-jump → Esc round-trip with `ResourcesLoadedMsg` injection to clear LoadStateLoading) → `TestFB042_QuickJump_RoundTrip_ReturnsToWelcome` at `model_test.go:9781`
- Full AC1–AC10 axis-coverage table delivered with submission, mapping multiple test rows per AC across happy / input-changed / repeat-press / anti-behavior / anti-regression axes
- `go install ./...` clean, `go test ./internal/tui/... -count=1` green

**Earlier rejection record (preserved for audit):** initial submission lacked direct tests for AC1 (positive `All clear` branch), AC6 (condition-kind branch), and AC8 (round-trip stability). Rework closed all three plus added supporting axis-coverage rows beyond the minimum.

**Priority: Medium** — dashboard is the first thing operators see; investing in it multiplies every session

#### User problem

The current welcome dashboard (FB-015) is informative but passive: it tells the operator their context and shows a static platform-health summary. An operator landing on it after a long absence doesn't know what happened since they were last active, doesn't know where to start, and has no affordance pointing them toward work in progress.

A great operator dashboard is a *launching pad*: it surfaces recent activity ("what happened while I was away"), highlights resources that need attention (high quota utilization, conditions not Ready), and provides quick-jump shortcuts so common tasks require fewer keystrokes.

#### Proposed enhancements

The welcome dashboard should grow to include the following sections, stacked vertically within the right pane. UX designer picks exact layout and decides which sections are collapsible or abbreviated at narrow widths.

**Section 1 — Identity + greeting (existing, keep)**
```
Welcome, Scot Wells
Datum Technology, Inc / datum-cloud
```
Keep as-is. The personalized greeting is already a bright spot.

**Section 2 — Platform health summary (existing, keep + enhance)**
Keep the top-N quota bar mini-chart. Add:
- An at-a-glance "all clear" or "N resources need attention" line above the bars.
- Color the summary line green (all clear) or amber/red (attention needed) to make status scannable without reading the bars.

**Section 3 — Recent activity teaser (new)**
Show the last 3–5 activity events across the project, pulled from the same source as the ActivityDashboard (FB-016). Format: relative timestamp + actor + resource name + action. A `[4]` hint directs to the full activity dashboard.

```
  Recent activity                             [4] full dashboard
  ──────────────────────────────────────────
  2m ago   swells@datum.net   dnszone/prod-api   Updated
  8m ago   ci@datum.net       backend/api-gw     Created
  1h ago   system             backend/api-gw     condition Ready
```

If FB-016 data is unavailable or not yet loaded, render a muted "loading…" or "no recent activity".

**Section 4 — Quick-jump shortcuts (new)**
A compact row of the most useful resource types for the current project, inferred from what the sidebar shows (or from governed quota types). Pressing the displayed key navigates directly to that resource type in the sidebar and loads its table.

```
  Quick jump:  [n] namespaces  [d] dnsrecordsets  [b] backends  [w] workloads
```

Keys are chosen to not conflict with existing global bindings. Designer to assign final key set.

**Section 5 — "What's needing attention" spotlight (new)**
Show up to 3 resources with non-Ready conditions or high quota utilization, surfaced as clickable rows:

```
  Needs attention
  ──────────────────────────────────────────
  ⚠  backend / api-gw           condition: Degraded  [Enter] view
  ▲  dnszones quota              91% allocated        [3] quota dashboard
```

If nothing needs attention: render a brief "Nothing needs attention right now" line in muted green.

**Section 6 — Keybind reference (existing, keep)**
Keep the four-column keybind strip. Consider updating it to reflect new dashboard keys from Section 4.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`.

1. **[Observable]** Dashboard renders Section 2 "all clear" line when no quota bucket is ≥80% utilized. Test: construct welcome panel with all buckets <80%, assert `stripANSI(view)` contains `"all clear"` (or designer-ratified copy).
2. **[Observable]** Dashboard renders Section 2 attention line ("N resources need attention") when ≥1 bucket is ≥80%. Test: inject bucket at 85%, assert count copy present.
3. **[Observable]** Section 3 recent-activity teaser renders up to 3 events when activity data is available. Test: inject 5 events, assert exactly 3 rendered rows (truncated, not all 5).
4. **[Observable]** Section 3 renders "no recent activity" muted placeholder when activity data is empty/nil. Test: nil events, assert placeholder copy present.
5. **[Observable]** Section 4 quick-jump row is visible and contains at least one resource type short name when sidebar resources are loaded. Test: inject resource list, assert `stripANSI(view)` contains at least one quick-jump key hint.
6. **[Observable]** Section 5 spotlight renders ≥1 attention row when a resource with non-Ready condition is present. Test: inject resource with `Ready=False` condition, assert spotlight row present.
7. **[Observable]** Section 5 renders "Nothing needs attention" placeholder when all resources are healthy. Test: inject all-healthy resources, assert placeholder present.
8. **[Input-changed]** Navigating from the dashboard to a resource type (via quick-jump or Enter) and back via Esc (FB-041) re-renders the dashboard with the same content. Test: round-trip; assert view unchanged.
9. **[Anti-regression]** FB-015 identity block, platform-health bars, and keybind strip all still present and unchanged after enhancements. Test: existing FB-015 substring assertions green.
10. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-015 ACCEPTED (welcome dashboard). FB-041 (Esc-to-dashboard) should ship first — operators need the return path before the dashboard becomes a hub. FB-016 data is reused (no new API calls needed for activity teaser).

**Maps to:** REQ-TUI-007 (welcome panel) + amendment to REQ-TUI-006 (platform health) — enhanced dashboard contract.

**Non-goals:**
- Not adding write operations (all data is read-only).
- Not adding a customizable dashboard (fixed layout for now).
- Not real-time push updates — existing 15s polling cycle is sufficient.
- Not surfacing events from other projects/orgs.

---

### FB-043 — Consumer-legible data freshness signal on quota surfaces

**Status: ACCEPTED 2026-04-19** — test-engineer validated with 11 new tests covering all six axes (Observable, Input-changed, Anti-behavior×5, Anti-regression, Integration). Canonical `stripANSI(*.View())` substring checks for both `titleBar()` (QuotaDashboard) and `buildQuotaSectionHeader()` (DetailPane inline). Input-changed axis uses two fixture times proving rendered value changes between fetches. Width-guard Anti-behavior tests confirm `gap < 2` (QuotaDashboard) and `innerW < 60` (DetailPane) drop freshness. `BucketsErrorMsg` Anti-behavior test confirms no freshness on error. `ContextSwitchedMsg` Anti-regression test confirms freshness cleared on context switch. AC#5/AC#6 (FB-036 removals hold + ActivityDashboardPane clean) covered implicitly via existing `TestFB036_*` tests green in full suite. Implementation: three files (`header.go` `HumanizeSince` export, `quotadashboard.go` `fetchedAt` field + `SetBucketFetchedAt` + title-bar freshness, `model.go` `bucketsFetchedAt` field + `buildQuotaSectionHeader()` method). **FB-044 now unblocked from FB-043 sequencing gate.** Dispatched user-persona for eval. Filed 2026-04-19 by product-experience from FB-036 user-persona P2-1 finding.
**Priority: P2** — decision signal gap: operators cannot confirm quota numbers reflect recent state after FB-036 removed the recon-age field that incidentally served as a freshness proxy.

#### User problem

FB-036 correctly removed `reconciled Xm ago` from the consumer quota surfaces because reconciliation age is a platform-health signal, not a consumer-decision signal. However, that same field incidentally served a second job: it was the operator's only proxy for *"how fresh is this number?"* — particularly after pressing `r` to refresh.

Post-FB-036, when a consumer sees `"7 / 100 (7%)"` on QuotaDashboard or inside the DetailPane inline quota block, they cannot tell whether that reflects state from 5 seconds ago or 5 minutes ago. When deciding "do I have room to create 10 more resources right now," data freshness is load-bearing.

The main header already shows `updated Xs ago` for resource-type fetch. The quota surfaces need an equivalent consumer-legible freshness affordance — **without** reintroducing the removed operator-signal fields.

#### Proposed interaction

Add a small, muted "fetched Xs ago" (or designer-ratified copy) freshness timestamp to:

1. **QuotaDashboard** — somewhere in the title-bar region, right-aligned, using the same humanized age pattern as the main header (`Xs / Xm / Xh ago`).
2. **DetailPane inline quota block** — appended to or near the `── Quota ──` section header, muted color.

Update happens when the quota fetch completes (new data landed), not on a polling tick. After `r` refresh, the timestamp resets to "fetched 0s ago" (or equivalent) proving the refresh took effect.

This is a *consumer* freshness signal tied to the client fetch, not a platform-health recon signal.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`.

1. **[Observable]** QuotaDashboard View() contains a freshness string (e.g., `"fetched"` substring) after an initial successful fetch. Test: inject fetch-completed state with a known timestamp, assert `stripANSI(dashboard.View())` contains freshness token.
2. **[Observable]** DetailPane inline quota block View() contains a freshness string after fetch completes. Test: same pattern, DetailPane view assertion.
3. **[Input-changed]** Freshness string updates after a subsequent fetch completes — the rendered age value differs between two successive fetches separated by a simulated clock advance. Test: inject t0 fetch (age="0s"); advance simulated clock by 30s + trigger second fetch; assert View() now contains "30s" or "0s" *different from previous value*. Must use View() content comparison, not model state.
4. **[Input-changed]** Pressing `r` refresh triggers freshness reset — post-`r` View() contains a smaller age value than pre-`r` View(). Test: render at age=90s; press `r` with fake clock; assert View() age value decreased.
5. **[Anti-regression]** FB-036 removals hold — `stripANSI(quotadashboard.View())` and inline block View() still do **not** contain `"out of sync"` or `"reconciled"` substrings. Test: FB-036's `TestFB036_CompactFormTree_NoOutOfSync` still green; extended to cover FB-043's new copy.
6. **[Anti-regression]** Freshness timestamp does not appear on operator/ActivityDashboardPane drift surfaces (scope is consumer quota surfaces only).
7. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-036 ACCEPTED (field removal). Reuses existing header humanized-age helper.

**Maps to:** REQ-TUI-019 (auto-refresh freshness) extended to quota surfaces.

**Non-goals:**
- Not reintroducing `reconciled Xm ago`, `claims: N`, or any operator-signal field (FB-036 thesis held).
- Not adding a millisecond-precision timestamp — same humanized pattern as main header (`Xs`, `Xm`, `Xh`).
- Not surfacing reconciliation age even indirectly — freshness = time-since-client-fetch only.

---

### FB-044 — Discoverable jump from DetailPane inline quota to full QuotaDashboard

**Status: ACCEPTED 2026-04-19** — test-engineer validated with 7 new tests. Axis coverage: Happy (`MatchingBuckets_Wide_Has3Affordance`), Anti-behavior×2 (`NoMatchingBuckets_No3Affordance` + `NarrowWidth_No3Affordance`), Input-changed (`WidthChange_3AffordanceAppearsDisappears`), Repeat (`RepeatCall_Idempotent`), Observable boundary (`BoundaryWidth33_Has3Affordance` at exact innerW=30 threshold), Anti-regression (`EmptyBuckets_NoBareQuotaSection`). All assertions use `stripANSIModel(buildDetailContent())` — no model-field inspection. AC#3 (`3` keypress → QuotaDashboard) and AC#4 (repeat press no-op) covered by existing `TestFB035_Key3_FromDetailPane_EntersDashboard`, `TestFB035_Key3_FromQuotaDashboard_TogglesBackToNav`, `TestFB035_Key3_RapidMash_DeterministicToggle` — FB-044 adds no new key handler so re-testing would be redundant. AC#6 (ActivityDashboardPane absence) covered structurally: `buildQuotaSectionHeader()` only called from `buildDetailContent()` with `len(matching) > 0`, not from ActivityDashboardPane code paths. Single file change: `model.go` — three-tier width guard in `buildQuotaSectionHeader()`. Next: user-persona dispatch. Filed 2026-04-19 by product-experience from FB-036 user-persona P2-2 finding.
**Priority: P2** — discoverability gap: operators viewing resource DetailPane with an inline quota block have no affordance to reach the full QuotaDashboard.

#### User problem

After FB-035 wired the `3` key to QuotaDashboard, the key works — but it is only discoverable from the welcome dashboard's keybind strip and from the HelpOverlay. When an operator is in DetailPane viewing a resource and sees the inline `── Quota ──` section, there is no inline hint that pressing `3` would take them to the fuller breakdown. Operators only learn the shortcut by accident.

This is a classic affordance placement gap: the trigger key (`3`) is global, but the *contextual moment the operator wants to use it* (viewing a quota block) has no hint at its point of attention.

#### Proposed interaction

When DetailPane contains an inline quota block, surface a compact affordance making the `3` key visible in-context. Designer picks the final placement; options include:

- **A.** Append `[3] full quota view` (or equivalent copy) to the `── Quota ──` section header text.
- **B.** Render a muted one-liner below the inline quota block: `[3] full quota dashboard`.
- **C.** Status-bar hint: when DetailPane is focused and its current viewport contains the quota section, the status bar shows a `[3] quota` chip in addition to its normal keybind list.

Only one option ships. Preference order for the designer: A > B > C (A is cheapest and closest to the attention point).

The affordance should **not** render on DetailPane surfaces without an inline quota block (non-quota-governed resource types), to avoid misleading operators into thinking `3` is contextual.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`.

1. **[Observable]** For a resource type with an inline quota block, `stripANSI(DetailPane.View())` contains the `[3]` affordance string (exact copy per ratified designer option).
2. **[Observable]** For a resource type **without** an inline quota block (e.g., an ungoverned type), `stripANSI(DetailPane.View())` does **not** contain the `[3]` affordance string. Test: render DetailPane with ungoverned type; assert absence.
3. **[Input-changed]** Pressing `3` from DetailPane (after the affordance is visible) opens QuotaDashboard. Test: construct DetailPane state with quota-governed resource; send `'3'` key; assert model transitions to QuotaDashboardPane AND `stripANSI(model.View())` contains QuotaDashboard title (distinguish actual pane transition from no-op).
4. **[Input-changed]** Pressing `3` repeatedly on DetailPane still opens QuotaDashboard on the first press, and is a no-op-or-close (designer-ratified) on the second press. Test: two consecutive `'3'` presses; assert first renders dashboard; assert second does not flicker / does not re-render dashboard state.
5. **[Anti-regression]** FB-035 global `3` binding still works from NavPane and welcome dashboard. Existing FB-035 tests green.
6. **[Anti-regression]** Affordance copy does not appear in any ActivityDashboardPane or operator-signal surface.
7. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-035 ACCEPTED (`3` key wired to QuotaDashboard). FB-036 ACCEPTED (inline quota block layout stable).

**Maps to:** REQ-TUI-008 (keybind discoverability) + REQ-TUI-017 (help/affordance surfacing).

**Non-goals:**
- Not adding a new key — only surfacing the existing `3` key contextually.
- Not making `3` behave differently inside DetailPane (global behavior unchanged).
- Not promoting other global keys to contextual affordances in the same brief (e.g., `4` for activity) — that's a separate discoverability pattern, handle per finding.

---

### FB-045 — Quota-surface consumer copy polish: `[t]` label + "sibling data unavailable" rewording

**Status: REJECTED 2026-04-19 — brief premise factually wrong; requires revision** — test-engineer delivered 9 tests all green, but the review surfaced a factually-wrong premise in the original brief that propagated through the spec and implementation, producing a label regression rather than an improvement. See `#### Rejection — brief premise error` below. **Next action:** revert the `[t]` label change + HelpOverlay VIEW entry; optionally keep the sibling-data reword (valid improvement) as a standalone brief (FB-058). The P3 finding that motivated FB-045's `[t]` portion (persona confusion about `[t]` semantics) remains unresolved and should be reopened via a new brief that *correctly* investigates which key the operator was actually pressing. Original brief filed 2026-04-19 by product-experience from FB-036 user-persona P3-1 + P3-2 findings bundled.
**Priority: P3** — copy legibility on quota consumer surfaces; two independent strings share the thematic "operator cannot reliably parse what the quota view is telling them."

#### Rejection — brief premise error

FB-045 rests on a factually-wrong claim: *"`[t] table` reads as 'go back to the resource table'; it actually toggles flat-list vs. grouped-tree view."*

Reading `internal/tui/model.go:940–953`:
- `[t]` toggles **between `QuotaDashboardPane` and `TablePane`** (for `allowancebuckets`). It literally means "go back to the table pane." The old label `[t] table` was accurate.
- `[s]` toggles **grouping** (`m.quota.ToggleGrouping()` at `model.go:957`). Flat/grouped is `[s]`'s job.

Test-engineer's own test name — `TestFB045_TKey_TogglesQuotaDashboardAndTablePane` — confirms the actual behavior (pane toggle), contradicting both the shipped label `[t] flat` and the HelpOverlay VIEW entry `[t]  flat / grouped`.

**Consequence of the shipped change:** the label has *less* accuracy than before. A user reading `[t] flat` now expects `[t]` to switch to flat-list view (a view-mode toggle, which is `[s]`). When they press it, they land in a completely different pane (the resource table showing proxy rows). Confusion increases, not decreases.

**Required reverts:**
1. `QuotaDashboard.titleBar()` — restore `[t] table` label.
2. HelpOverlay — remove the `[t]  flat / grouped` VIEW entry (or replace with an accurate entry if `[t]` is ever promoted to the overlay).
3. Tests — delete or update FB-045 assertions that pin `[t] flat` as the canonical label.

**Keep:** the sibling-data reword (`(sibling data unavailable)` → `(other projects' usage hidden)`) is a standalone legitimate improvement and should survive as a new standalone brief (**FB-058**) if test-engineer/ux-designer agree, or folded into a fresh FB-045-revision.

**Why this slipped through:** the persona finding flagged real confusion about `[t]`, but the brief author (me) accepted the persona's interpretation of *what the key does* without verifying against code. The ux-designer spec inherited the error. The engineer implemented literally. The test-engineer's test revealed the contradiction but passed because it asserted on the new label, not on label↔behavior alignment. Lesson: when a persona claim contradicts an existing label, read the key's handler before writing the brief. Memory update pending.

**Pipeline action:** test-engineer please revert the FB-045 copy change in a new commit (or request engineer revert). Sibling-data reword survives. Ux-designer: treat the `[t]`-label portion of the FB-045 spec as cancelled.

#### User problem

Two copy strings on the FB-036 post-cleanup quota surface read ambiguously to first-time and returning operators:

1. **`[t] table`** in the QuotaDashboard title-bar keybind strip reads as *"go back to the resource table"* (i.e., exit the dashboard). It actually toggles flat-list vs. grouped-tree view. Operators experimentally press it and get unexpected behavior.
2. **`(sibling data unavailable)`** appended to tree parent rows when sibling-project data is restricted — "sibling data" is a platform-internal term. Consumers don't know they have siblings or what siblings are.

Post-FB-036, with the recon/claim noise removed, these two strings are now more prominent by contrast and read as the remaining rough edges on an otherwise cleaner surface.

#### Proposed interaction

1. Rename `[t] table` → `[t] flat` or `[t] list` (designer picks the unambiguous label; must not conflict with an existing global or pane-local `[t]` or `[l]` binding). Corresponding HelpOverlay entry updated.
2. Rewrite `(sibling data unavailable)` → consumer-legible explanation. Designer-ratified copy, but seed examples: `"(other projects' usage hidden)"` or `"(peer projects restricted)"` — whatever survives designer review as clearest at ≤ 40 chars so it fits inline without re-wrapping tree rows.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`.

1. **[Observable]** `stripANSI(QuotaDashboard.View())` contains the new `[t]` label copy (per designer choice) and does **not** contain the string `"[t] table"`. Test: substring presence + substring absence.
2. **[Observable]** HelpOverlay View() contains the new `[t]` label copy aligned with the title-bar strip. Test: substring match.
3. **[Observable]** When rendering a tree with restricted sibling data, `stripANSI(QuotaDashboard.View())` contains the new rewritten note (per designer copy) and does **not** contain `"sibling data unavailable"`. Test: inject fixture with restricted-sibling state; substring assertions both ways.
4. **[Input-changed]** Pressing `[t]` from QuotaDashboard still toggles flat/grouped view (behavior unchanged — only label changes). Test: press `[t]` twice; assert view toggles between flat and grouped renderings (existing FB-036-era toggle test extended).
5. **[Anti-regression]** `[s] group` and `[r] refresh` labels unchanged. FB-036 compact-form tree test still green.
6. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-036 ACCEPTED (layout baseline stable).

**Maps to:** REQ-TUI-008 (keybind legibility).

**Non-goals:**
- Not changing the actual behavior of `[t]` — only its label.
- Not changing the sibling-data restriction semantics or fetch logic — only the rendered note.
- Not touching other potentially-confusing labels outside QuotaDashboard (out of scope; file separately if found).

---

### FB-046 — FB-036 removal-residue cleanup: dead helper, stale docstring, stale comment

**Status: ACCEPTED 2026-04-19** — test-engineer validated with grep-observable ACs: `bucketFormatAge` removed (0 matches), `"claims: N"` removed (0 matches), `"keep recon column stable"` removed (0 matches), full test suite green, `go install ./...` compiles. Axis-coverage table: Observable (AC#1/2/3 via grep), Anti-regression (AC#4 full suite), Integration (AC#5 install). No new test code needed — anti-regression is existing suite remaining green. **Skipped UX-DESIGNER** — no user-observable surface changes. **Skipped user-persona dispatch** — strict cleanup brief with no user-observable surface changes to evaluate; no persona gap to find. Closes FB-036 residue cleanly.
**Priority: P3** — developer-facing residue; closes out FB-036's thesis (clean removal of recon/claim fields).

#### Engineer-facing problem

FB-036 removed the recon-age and claim-count fields from the rendered quota surfaces but left three code-level artifacts that reference the removed functionality:

1. `internal/tui/components/quotablock.go:215–226` — `bucketFormatAge` helper is no longer called anywhere; was used exclusively for the removed recon line. Dead code.
2. `internal/tui/components/quotablock.go:138–139` — `RenderQuotaBlock` docstring still documents `claims: N   reconciled Xm ago` as the third output line (including a stale warning note). Misleading API contract.
3. `internal/tui/components/quotabanner.go:145` — `// keep recon column stable` comment references a column that no longer exists. Confusing for future layout-math readers.

This brief closes the residue so no future reader confuses dead artifacts with intentional preservation.

#### Proposed change

1. Delete `bucketFormatAge` at `quotablock.go:215–226` after a final grep confirms no callers (tests + production).
2. Rewrite the `RenderQuotaBlock` docstring at `quotablock.go:138–139` to describe the current two-line output (bar + count) and remove the stale third-line note.
3. Delete or rewrite the `// keep recon column stable` comment at `quotabanner.go:145` — if the padding constant it guards still serves a purpose, rewrite the comment to explain the real reason; otherwise delete the comment.

No user-observable behavior changes. This is a strict cleanup brief.

#### Acceptance criteria

Axis tags: `[Observable]` *(absence-based)*, `[Anti-regression]`.

1. **[Observable]** `grep -rn 'bucketFormatAge' internal/tui/` returns zero matches. Test: build-time; CI-grep verification.
2. **[Observable]** `RenderQuotaBlock` godoc no longer contains `"reconciled"` or `"claims:"` substrings. Test: inspect rendered godoc via `go doc ./internal/tui/components RenderQuotaBlock`; substring absence.
3. **[Observable]** `internal/tui/components/quotabanner.go` no longer contains the substring `"keep recon column stable"`. Test: CI-grep verification.
4. **[Anti-regression]** All existing FB-035/FB-036 View() substring tests green. No rendered-output change.
5. **[Anti-regression]** `go vet ./...` reports no new warnings. `go install ./...` compiles. `go test ./internal/tui/...` green.
6. **[Integration]** Engineer must verify no caller outside `internal/tui/` uses `bucketFormatAge` before deletion (run `grep -rn 'bucketFormatAge' .` at repo root).

**Dependencies:** FB-036 ACCEPTED.

**Maps to:** REQ-TUI-036-cleanup (non-normative — housekeeping).

**Non-goals:**
- Not refactoring `RenderQuotaBlock` itself — only docstring and dead-code removal.
- Not re-running FB-036 acceptance — FB-036's ACs were met; this is strictly post-ship tidy.
- Not auditing unrelated dead code in `internal/tui/components/` — scope pinned to the three FB-036 residue sites.

---

### FB-047 — `3` keypress swallowed silently while QuotaDashboard data is loading

**Status: ACCEPTED 2026-04-19** — implemented by engineer + tests by test-engineer. Verified: `go test ./internal/tui/... -run TestFB047_` all green; `go install ./...` clean. 7 tests cover AC1–AC7 across Observable/Repeat-press/Anti-behavior/Anti-regression/Error-path axes. AC8/AC9 (quotaBucketsLoaded affordance gating) dropped per spec trim.
**Priority: P2** — silent input failure teaches users the feature "doesn't exist from here."

> Note 2026-04-19: An earlier scope expansion absorbing FB-044 P3-1 was retracted when the source persona finding was withdrawn. Original brief restored.

#### User problem

On first launch, bucket data for QuotaDashboard fetches in the background. If the operator presses `3` during that fetch, the guard `if !m.quota.IsLoading()` silently discards the keypress — no visible feedback, no "loading…" hint, no spinner. The operator concludes either (a) `3` is unbound in the current pane, or (b) the feature doesn't exist yet. They give up and never try `3` again in this session.

This is a **discoverability-through-affordance** bug: the key IS bound, but the system's response is indistinguishable from "unbound key" to the operator.

#### Proposed interaction

When `3` is pressed AND `m.quota.IsLoading()` returns true, the operator must get *some* visible acknowledgement that the key was received and the dashboard is on the way. Designer picks one of:

- **A.** Queue the keypress: set `m.pendingQuotaOpen = true`; when `QuotaLoadedMsg` arrives, auto-transition to QuotaDashboardPane. Operator's single `3` press "just works" (with a perceived delay).
- **B.** Show a transient status-bar hint: `"Quota dashboard loading — opens in a moment"` with a token-based auto-clear (same pattern as FB-012 stale messages). Does NOT auto-transition; operator presses `3` again after load.
- **C.** Combination: show the hint AND queue the transition.

Engineer-preference signal: option A (auto-transition) is the least surprising to the operator but requires pending-key state plumbing. Option B is safest and requires no state changes.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-behavior]`, `[Anti-regression]`.

1. **[Observable]** When `3` is pressed during quota loading, `stripANSI(model.View())` contains a visible acknowledgement (hint copy per designer choice) OR the pane transitions to QuotaDashboardPane after load. Test: inject loading state; send `'3'`; assert View() either contains hint substring OR `pendingQuotaOpen` flag is set AND post-`QuotaLoadedMsg` View() shows QuotaDashboard title.
2. **[Input-changed]** Pressing `3` twice during loading (repeat-press axis) does not produce double-queued transitions or duplicate hints. Test: send `'3'` twice pre-load; post-load assert QuotaDashboard opened exactly once (no double-transition flicker).
3. **[Anti-behavior]** Pressing `3` when `!m.quota.IsLoading()` behaves exactly as today (FB-035 AC set green). Test: existing FB-035 tests untouched.
4. **[Anti-regression]** Silent discard no longer occurs — `stripANSI(model.View())` after pressing `3` during loading differs from View() before pressing `3` (either hint copy appears OR pane transitions). Test: before/after View() content comparison asserts change (per `feedback_input_changed_assertions.md`).
5. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-035 ACCEPTED.

**Maps to:** REQ-TUI-008 (keybind discoverability) — covers the "key pressed but no feedback" anti-pattern.

**Non-goals:**
- Not adding a loading spinner to the dashboard itself (out of scope; a separate loading-affordance brief if needed).
- Not changing bucket fetch timing or caching.
- Not applying the same queueing pattern to `4`, `d`, or other keys in the same brief — if the pattern generalizes, it ships in a later convergence brief.

---

### FB-048 — Dashboard panes preserve and restore origin-pane context on exit

**Status: ACCEPTED 2026-04-19** — engineer implementation shipped jointly with FB-050 (toggle-for-both); 7 tests green. AC1 + AC3 rework added `stripANSIModel(View())` substring assertions on `"my-pod"` (row rendered) + absence of `"Welcome"` (NavPane not re-rendered); AC2 View() yaml-content; AC4 Esc path restores state; AC5 zero-origin returns to NavPane; AC6 filter `"xyz-unique"` survives round-trip; AC7 showDashboard round-trip. Persona evaluation will bundle with FB-050 now that both are accepted (shared origin-restore + toggle UX surface).

Originally filed 2026-04-19 by product-experience from FB-035 user-persona P2-2 + P3-3 bundled.
**Priority: P2** — navigation cost loss penalizes "quick cross-reference" workflows during investigation.

#### User problem

Operators use QuotaDashboard (`3`) and ActivityDashboard (`4`) as *quick-reference lookups* during investigation — "let me check quota while browsing HTTPRoutes" or "let me scan recent activity before reading this resource's YAML." When they exit the dashboard (Esc or `3`/`4` again), they land at NavPane with the sidebar cursor position preserved but the resource table and row selection lost. Scroll position, filter state, and DetailPane YAML/events state are also discarded.

Compare to `d` (describe): pressing Esc from DetailPane returns to TablePane with the exact previous row selected — cost of the roundtrip is near-zero. The dashboard panes should behave similarly.

Concrete user-experience sequences from persona:

- **P2-2:** TablePane (HTTPRoute `prod-api` row selected, filter `prod-*` active) → `3` → QuotaDashboard → `3`/Esc → lands on NavPane sidebar (row, filter, scroll all gone).
- **P3-3:** DetailPane (resource YAML open, viewport scrolled to mid-body) → `3` → QuotaDashboard → `3`/Esc → NavPane. Investigator has to re-navigate: sidebar → type → find row → Enter → open detail → re-scroll.

#### Proposed interaction

When the operator opens a dashboard pane (`3` → QuotaDashboard, `4` → ActivityDashboard) from **any** pane, stash the origin pane + origin state. When the operator exits the dashboard via `3`/`4`/Esc, restore the stashed origin pane and state.

Origin state to preserve:
- Pane identity (NavPane / TablePane / DetailPane).
- For TablePane: `tableTypeName`, selected row index, filter text, scroll offset.
- For DetailPane: active mode (yaml / conditions / events), scroll offset, `describeRaw`/`events` caches.

Startup case: if dashboard was opened from NavPane (welcome state), exit returns to NavPane (no regression from today's behavior).

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`.

1. **[Observable / Input-changed]** Open QuotaDashboard via `3` from TablePane (row selected) → exit via `3` → `stripANSI(model.View())` shows the same TablePane with the same row selected. Test: inject TablePane state (rowIdx=5), press `3`, assert QuotaDashboard; press `3`, assert `model.tablePane.SelectedIndex == 5` AND `stripANSI(model.View())` contains the row's name at the same cursor position (View() substring includes cursor glyph near that name).
2. **[Observable / Input-changed]** Open QuotaDashboard from DetailPane (YAML mode, scrolled to offset N) → exit → DetailPane re-renders at the same mode and offset. Test: inject DetailPane(yaml=true, offset=10), press `3`, press `3`, assert View() contains same YAML region + scroll offset preserved.
3. **[Observable / Input-changed]** Same pair of tests for `4` → ActivityDashboard → return.
4. **[Observable]** Exit via Esc (not the toggle key) also restores origin. Test: enter dashboard, press Esc, assert origin pane + state restored identical to toggle-key exit.
5. **[Anti-regression]** When dashboard is opened from NavPane (welcome / no table), exit returns to NavPane (same as today). Test: existing FB-035 NavPane-return tests green.
6. **[Anti-regression]** Filter state on TablePane is preserved through dashboard round-trip. Test: inject filter "prod-*", `3`, `3`, assert filter still active + filtered rows.
7. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-035 ACCEPTED, FB-041 (Esc-to-dashboard) can coexist — origin-pane restore takes priority over Esc-to-welcome when the exit path is *from* a dashboard (i.e., dashboards are "modal lookups," welcome-dashboard is the home state). Designer resolves interaction precedence in §7 of the spec.

**Maps to:** REQ-TUI-018 (context switch) extended to intra-session dashboard round-trips.

**Non-goals:**
- Not adding a stack-based navigation history (single-level origin stash is enough — dashboards don't nest).
- Not changing the `Tab` pane-cycling behavior.
- Not preserving overlay state (HelpOverlay / CtxSwitcher) — those already dismiss on `3`/`4` today and continue to do so.

---

### FB-049 — QuotaDashboardPane status bar: surface pane-local keys, suppress inapplicable generic keys

**Status: ACCEPTED 2026-04-19** — 8 tests covering Observable hint absences/presences (AC1/AC2/AC3/AC6), Input-changed hint switches (AC4/AC5), Anti-regression NavPane+TablePane unchanged (AC7). Axis coverage table complete. All Observable ACs use View() substring assertions. Filed 2026-04-19 by product-experience from FB-035 user-persona P2-3 + P3-1 bundled.
**Priority: P2** — status bar actively misinforms operators about which keys work in QuotaDashboardPane.

#### User problem

In QuotaDashboardPane, the status bar falls through to the generic hint string used for NORMAL mode: `[j/k] move  [Enter] select  [/] filter  [d] describe  …`. Two failure modes:

1. **Misinformation (P2-3):** `[/] filter` is shown but pressing `/` from QuotaDashboard triggers a transient "Select a resource type first…" hint — the advertised key doesn't work here. Operators interpret this as a bug. Similarly, `[Enter] select` and `[d] describe` are non-applicable in quota context.
2. **Omission (P3-1):** The quota-local keys `[s] group/flat`, `[t] table view`, `[r] refresh`, `[3] back` are shown only in the dashboard's own title bar. On narrow terminals or tall viewports where the title bar scrolls out of natural view, the status bar — where operators reflexively look — shows none of these.

Net effect: operators cannot trust the status bar as a ground-truth affordance surface in QuotaDashboardPane.

#### Proposed interaction

The status bar must be pane-aware. When `activePane == QuotaDashboardPane`:
- **Suppress** generic NORMAL-mode hints that do not apply: `[/] filter`, `[Enter] select`, `[d] describe`, `[x] delete`, `[c] conditions` (and any similar).
- **Surface** quota-local hints: `[s]`, `[t]`, `[r]`, `[3]` back.
- Keep mode label (`QUOTA`) and global-across-all-panes hints (`[?]` help, `[q]` quit) as-is.

The dashboard title bar continues to show the same hints it does today (redundancy is fine — designer decides whether to drop from title bar once status bar has them, but the brief's thesis is status-bar correctness, not title-bar reduction).

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-behavior]`, `[Anti-regression]`.

1. **[Observable]** On QuotaDashboardPane, `stripANSI(statusBar.View())` does **not** contain `"[/] filter"`, `"[Enter] select"`, or `"[d] describe"`. Test: construct model with QuotaDashboardPane active; assert substring absences.
2. **[Observable]** On QuotaDashboardPane, `stripANSI(statusBar.View())` contains `"[s]"`, `"[t]"`, `"[r]"`, and `"[3]"` quota-local affordances. Test: substring presence.
3. **[Observable]** On QuotaDashboardPane, `stripANSI(statusBar.View())` still contains `"[?]"` and `"[q]"` global hints (anti-regression on global affordances). Test: substring presence.
4. **[Input-changed]** Transition NavPane → QuotaDashboardPane (press `3`) toggles the status-bar hint set from generic to quota-local. Test: before-View() has `"[/] filter"`; after-View() does not AND has `"[s]"`.
5. **[Input-changed]** Transition QuotaDashboardPane → NavPane (press `3` or Esc) restores generic hints. Test: before-View() has `"[s]"`; after-View() does not AND has `"[/] filter"`.
6. **[Anti-behavior]** Pressing `/` from QuotaDashboardPane is a no-op (silent, no transient hint). Test: send `'/'` on QuotaDashboardPane; assert View() unchanged (no hint copy appears). Related to FB-047 pattern but scope pinned to `/`.
7. **[Anti-regression]** Pane label reads `QUOTA` (unchanged). NavPane/TablePane/DetailPane status-bar hint sets are unchanged. Test: existing pane-label + generic-hint tests green.
8. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-035 ACCEPTED.

**Maps to:** REQ-TUI-008 (keybind discoverability) — pane-local affordance surfacing.

**Non-goals:**
- Not changing the dashboard's title-bar hint strip (unless designer chooses; default is keep).
- Not applying the same pane-aware status-bar treatment to ActivityDashboardPane in this brief — if the pattern generalizes, file a convergence brief for FB-050 territory.
- Not introducing a new `StatusBarHintSet` abstraction — simplest per-pane conditional is fine.

---

### FB-050 — Dashboard toggle-key behavior consistency: `3` and `4` share the same toggle-vs-no-op semantics

**Status: ACCEPTED 2026-04-19** — engineer implemented toggle-for-both jointly with FB-048 origin-restore; 5 tests green (`TestFB050_Key3_ToggleFromTablePane_RestoresTablePane`, `TestFB050_Key4_ToggleFromTablePane_RestoresTablePane`, `TestFB050_HelpOverlay_ContainsActivityToggleLabel`, `TestFB050_Key3_FirstPress_FromNavPane_StillEntersDashboard`, `TestFB050_Key4_FirstPress_FromNavPane_StillEntersDashboard`). Note: test-engineer's submitted axis-coverage table mislabeled AC1/AC2 as `[Observable]`; brief axis is `[Repeat-press]` (assert final pane after two presses) — tests do exercise repeat-press semantics, substantive coverage present. AC3 helpOverlay View() check ✓; AC4/AC5 first-press anti-regression ✓; AC6 install/test green. NavPane→QDash→NavPane double-press case covered transitively by FB-048 AC5 (`TestFB048_Key3_Exit_ZeroOrigin_ReturnsToNavPane`). Persona evaluation HELD until FB-048 also accepts (bundled origin-restore + toggle UX is the full operator surface).

Originally filed 2026-04-19 by product-experience from FB-035 user-persona P2-4 + P3-2 bundled.
**Priority: P2** — inconsistent dashboard toggle keys force operators to remember two mental models for visually identical keys.

#### User problem

Current behavior:
- Pressing `3` while on QuotaDashboardPane exits to NavPane (toggle behavior).
- Pressing `4` while on ActivityDashboardPane stays on ActivityDashboardPane (no-op / re-fire of transition).

Operators who use both dashboards report having to "remember which one toggles." This is a consistency bug, not a feature — there is no design rationale for the asymmetry.

Secondary observation (P3-2): the help overlay reads `[3] quota dashboard` with no indication the key is also the exit key. Similar implication for `[4] activity dashboard`. If the toggle-vs-no-op semantics are unified, the overlay copy should reflect it (e.g., `[3] quota dashboard (toggle)`).

#### Proposed interaction

Unify toggle semantics. Designer picks the canonical behavior; recommendation below.

- **Recommendation: toggle for both.** Pressing the dashboard's own key from that dashboard returns to the origin pane (per FB-048) or NavPane (pre-FB-048). Rationale: toggle matches the "modal lookup" mental model — press once to open, press same key to close. Matches Esc parity for dashboard panes.
- **Alternative: no-op for both.** Pressing the dashboard's own key is a no-op; exit is always via Esc. Rationale: consistent "use Esc to exit panes" convention.

The designer's spec must pin ONE choice and update the help overlay copy correspondingly.

**Interaction with FB-048:** If toggle-for-both is chosen, `3`/`4` exit paths must also preserve origin-pane context per FB-048. If no-op-for-both is chosen, FB-048's scope applies only to Esc.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Repeat-press]`, `[Anti-regression]`.

1. **[Repeat-press]** From NavPane, press `3` → QuotaDashboardPane; press `3` again → behavior matches the designer's chosen semantics (toggle-exit OR no-op-stay). Test: two consecutive `'3'` presses; assert final pane per spec decision.
2. **[Repeat-press]** From NavPane, press `4` → ActivityDashboardPane; press `4` again → **same** behavior as the `3` double-press (AC#1's outcome). Test: two consecutive `'4'` presses; assert final pane matches AC#1's final pane semantically.
3. **[Observable]** Help overlay contains copy reflecting the chosen semantics — e.g., if toggle: substring `"(toggle)"` or equivalent; if no-op: no toggle annotation. Test: `stripANSI(helpOverlay.View())` substring match per spec decision.
4. **[Anti-regression]** `3` from NavPane → QuotaDashboardPane on first press (unchanged from FB-035). Existing FB-035 tests green.
5. **[Anti-regression]** `4` from NavPane → ActivityDashboardPane on first press (unchanged from FB-016). Existing FB-016 tests green.
6. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-035 ACCEPTED, FB-016 ACCEPTED. **Interacts with FB-048** — toggle semantics + origin-pane-restore must ship in compatible order. Designer coordinates with FB-048 spec.

**Maps to:** REQ-TUI-008 (keybind consistency).

**Non-goals:**
- Not unifying `3`/`4` with other dashboard-equivalent keys (e.g., there's no `2` dashboard today).
- Not changing Esc behavior (Esc continues to exit dashboards; this brief only touches the dashboard's own key when pressed from within that dashboard).
- Not relitigating whether `3`/`4` should exist as global keys (they do, per FB-035/FB-016 ACCEPTED).

---

### FB-051 — Placeholder preserves error context and affordances on events-loaded transition

**Status: ACCEPTED 2026-04-19** — engineer shipped jointly with FB-052 (combined route, FB-052 §3 authoritative for inline placement). 8+1 tests green covering Observable error subline / title-bar mode label / [r] affordance + Input-changed recovery / r dispatches retry + Anti-regression loading variant + events=nil error-block precedence. All Observable ACs use `stripANSIModel(View())` substring assertions.

Originally filed 2026-04-19 by product-experience from FB-037 user-persona P2-1 + P3-1 + P3-2 bundled (shared thesis).
**Priority: P2** — triage context gap: when the error block is swapped for the FB-024 placeholder on events-loaded, the operator loses the error reason, the retry affordance, and gets a title-bar mode label that contradicts the content.

#### User problem

FB-037 fixes the error-first race by re-rendering DetailPane content on `EventsLoadedMsg` and giving the FB-024 placeholder priority over the error block. That restored the golden path (stuck error → actionable placeholder). But the placeholder throws away three pieces of information that were visible in the error block:

1. **Error reason** — before: `"⚠ Permission denied — You don't have permission to perform this action."`. After: `"Describe unavailable — only events loaded."` (muted, no severity, no reason). An operator doing triage on an RBAC failure reads the reason, then events arrive, and the reason silently vanishes. They can no longer tell whether describe is unavailable permanently (RBAC), transiently (timeout), or still loading.
2. **Title-bar mode label** — stays `describe` even though describe has failed and the viewport content is the placeholder. A user scanning title bar first sees "describe" and expects describe content, is confused by the placeholder.
3. **Retry affordance** — the error block's inline `[r] retry` hint is gone from the placeholder. `r` still works (FB-024 handler), but there's no hint. A user who saw the error block and was mid-reach for `r` loses the affordance.

All three share a single thesis: **the placeholder state does not inherit the error block's informational payload.** The placeholder is correct in *priority* but wrong in *content-preservation*.

#### Proposed interaction

When `loadState == error AND events != nil` (Part B priority-1 placeholder state), the placeholder should show:

1. **Error-reason subline (muted)** — below the primary placeholder copy, append a one-line muted summary of the last error reason (e.g., `(describe failed: permission denied)`). Designer picks exact copy; must be clearly secondary to the primary "Describe unavailable — only events loaded" line.
2. **Title-bar mode annotation** — change the title-bar mode label from `describe` to `describe (unavailable)` or `describe [failed]` (designer-ratified) when the placeholder is active. This resolves the content/label contradiction.
3. **`[r] retry` affordance** — surface the `[r] retry` hint in the placeholder state (designer picks placement: inline in placeholder body, in title-bar hint row, or both). Must work at all terminal widths (coordinates with FB-052).

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`.

1. **[Observable]** When the placeholder is active with a prior error, `stripANSI(DetailPane.View())` contains both the primary placeholder line AND the error-reason subline. Test: inject `loadState=error + lastError="permission denied" + events=[...]`; assert View() contains both `"Describe unavailable"` AND `"permission denied"` (or whatever copy the designer ratifies).
2. **[Observable]** The title-bar mode label contains an "unavailable" or equivalent failure annotation when the placeholder is active. Test: `stripANSI(DetailPane.View())` contains `"describe (unavailable)"` (designer-ratified exact copy).
3. **[Observable]** `stripANSI(DetailPane.View())` contains `[r]` affordance string when the placeholder is active.
4. **[Input-changed]** When `loadState` transitions from `error → ok` (describe recovers) and then events load, the placeholder no longer renders (priority 1 trigger no longer fires). Test: inject error + events → placeholder renders; then inject describe-ok → describe view renders; assert error-reason subline absent.
5. **[Input-changed]** Pressing `r` from the placeholder state re-dispatches describe fetch. Test: send `'r'` key; assert `DescribeFetchCmd` is dispatched (message-level assertion, not UI-level).
6. **[Anti-regression]** FB-037 placeholder priority holds — when `loadState=error AND events != nil`, the placeholder renders, not the error block. Existing FB-037 tests green.
7. **[Anti-regression]** Non-error-prior placeholder state (`describeRaw=nil AND loadState=ok AND events != nil`) does **not** contain the error-reason subline or the `(unavailable)` label annotation. Test: inject `describeRaw=nil + lastError="" + events=[...]`; assert error-reason subline absent.
8. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-037 ACCEPTED (re-render gate + placeholder priority). FB-024 ACCEPTED (placeholder exists).

**Maps to:** REQ-TUI-013 (error feedback) extended to placeholder-state inheritance.

**Non-goals:**
- Not changing placeholder priority ordering (Part B stays).
- Not adding a full error-overlay on top of the placeholder — one-line muted subline only.
- Not generalizing error-context preservation to other state transitions (e.g., table-pane errors) — scope pinned to DetailPane placeholder.

---

### FB-052 — Placeholder inline affordance at narrow terminal widths

**Status: ACCEPTED 2026-04-19** — engineer shipped jointly with FB-051 (combined route, FB-052 §3 authoritative for inline placement). 7 tests green covering Observable [E] inline at width=40 + width=120 + Input-changed E opens events mode (View() differs) + Input-changed loading vs error variant key sets + Anti-regression error-block-no-events has no [E] + FB-051 subline still present. All Observable ACs use `stripANSIModel(View())` substring assertions.

Originally filed 2026-04-19 by product-experience from FB-037 user-persona P2-2.
**Priority: P2** — narrow-terminal affordance gap: at ≤48 columns the title-bar hint row drops, leaving the placeholder state with zero visible path to events.

#### User problem

The FB-037 spec §3.5 acknowledged this gap. At terminal widths ≤48 columns, the title-bar hint row that shows `[E] events` (and `[E] describe`, `[r] retry`, etc.) is dropped per the existing header responsive-collapse. The placeholder body is a single muted line: `"Describe unavailable — only events loaded."` — and the `[E] events` keybinding hint disappears entirely. The key still works, but a user on a constrained terminal (SSH to jumpbox, split pane, small window) sees the placeholder and has no visible path to events.

Contrast with the error block: its inline action row `[r] retry  [Esc] back` renders independently of terminal width (it's in the body, not the title-bar hint row). The placeholder should match that pattern.

#### Proposed interaction

Surface the `[E] events` affordance (and any other placeholder-relevant keys) **inline in the placeholder body**, not only in the title-bar hint row. Designer picks exact placement:

- **A.** Append after the primary placeholder line: `"Describe unavailable — only events loaded. [E] view events"`.
- **B.** Render a second muted line below the primary: `"[E] view events"`.
- **C.** Reuse the error-block inline-action-row pattern: render `[E] events  [r] retry  [Esc] back` as a second line.

Preference order: C > B > A (C unifies with the error block's well-established pattern). Must render identically at narrow widths — no responsive collapse.

Coordinates with FB-051 (both modify the placeholder body). If FB-051 ships first, FB-052 extends FB-051's placeholder block. If FB-052 ships first, FB-051 will layer onto FB-052's inline-affordance pattern.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`.

1. **[Observable]** At terminal width = 40 (below the title-bar hint-row threshold), `stripANSI(DetailPane.View())` contains `[E]` in the placeholder body. Test: render placeholder at narrow width; assert substring.
2. **[Observable]** At terminal width = 120 (wide), the inline affordance is still present in the placeholder body — this is not a narrow-only affordance, it's an always-on inline replacement for the title-bar-only hint. Test: render placeholder at wide width; assert `[E]` appears inline AND in title-bar hint row (both OK).
3. **[Input-changed]** Pressing `E` from the placeholder at width=40 opens events mode. Test: send `'E'` key; assert `eventsMode == true` via View() assertion (title bar now shows `events` mode or similar).
4. **[Anti-regression]** Non-placeholder states (error block, describe view) do not gain a new inline affordance — scope is placeholder body only. Existing error-block `[r] retry  [Esc] back` inline row unchanged.
5. **[Anti-regression]** FB-037 placeholder priority holds (error + events → placeholder). Existing FB-037 tests green.
6. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-037 ACCEPTED (placeholder exists).

**Maps to:** REQ-TUI-017 (affordance surfacing) at narrow widths.

**Non-goals:**
- Not rewriting the title-bar hint-row responsive-collapse logic — the title-bar can still drop the hint at narrow widths; the inline affordance is a separate surface.
- Not adding inline affordances to non-placeholder DetailPane states.
- Not changing the `E` keybinding semantics.

---

### FB-053 — Transition signal when events arrive and swap error block for placeholder

**Status: ACCEPTED 2026-04-19** — 5 tests covering Observable hint appears on error→placeholder transition (AC1) + View() differs (AC2); Anti-behavior no hint on normal events load (AC3) + no hint on events-load error (AC4); Anti-regression FB-037 placeholder still renders (AC5). Axis-coverage table complete; all Observable ACs use View() substring assertions. AC4 required engineer fix (`msg.Err == nil` guard added) before submission. Filed 2026-04-19 by product-experience from FB-037 user-persona P3-3.
**Priority: P3** — minor-friction discoverability gap: the error→placeholder swap is silent; users who aren't watching the pane may miss the transition.

#### User problem

When the error block transitions to the FB-024 placeholder (events arrived while describe failed), the swap is silent. No toast, no "pane updated" signal, no flash. If the operator is reading the status bar or looking at another window, they return to a pane that looks empty-ish and assume it never loaded. The existing `SetContent → GotoTop` correctly positions the placeholder visibly, but there's no signal that *something changed*.

The main header's `r` key pattern already models a freshness/update signal (`updated Xs ago`). A brief equivalent for the error→placeholder transition would close this gap.

#### Proposed interaction

Designer picks one of:

- **A.** Brief status-bar hint: when the transition fires, show a muted `events loaded — press E` hint in the status bar for ~3s, then fade back to normal.
- **B.** Header-level annotation: add a transient freshness annotation to the DetailPane title bar for a few seconds after the transition (e.g., `events just loaded`).
- **C.** A subtle title-bar color pulse (lipgloss-supported) for one render cycle when the transition fires.

Preference order: A > B > C. Must not be intrusive — the placeholder's visible render is the primary signal; this is a secondary "something changed" cue.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`.

1. **[Observable]** Immediately after the error→placeholder transition fires, `stripANSI(model.View())` contains the transition-signal string (designer-ratified copy, e.g., `"events loaded"` or `"events just loaded"`). Test: inject error → error block renders; inject EventsLoadedMsg → assert View() contains the signal string.
2. **[Input-changed]** When the describe view is active (no placeholder), the transition-signal string does not appear. Test: inject describe-ok state; assert signal absent.
3. **[Anti-behavior]** The transition signal does not fire on first-page-load when `loadState=ok AND events=nil` transitions to `loadState=ok AND events=[...]` (normal describe + events load, no error). Test: assert signal absent.
4. **[Anti-regression]** FB-037 placeholder renders correctly whether or not the signal is present. Existing FB-037 tests green.
5. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-037 ACCEPTED. FB-051 (if signal wording references error context) — check before design.

**Maps to:** REQ-TUI-019 (freshness/update signals) extended to state-transition events.

**Non-goals:**
- Not adding a persistent status line — the signal is transient.
- Not generalizing transition-signals to every pane swap — this brief scopes only to error→placeholder in DetailPane.
- Not using color-pulse if lipgloss color animation is not already established (prefer text-based signal).

---

### FB-054 — Welcome panel surfaces Tab-to-restore hint when a table is cached

**Status: ACCEPTED 2026-04-19 / PERSONA-EVAL-COMPLETE 2026-04-20** — 7 tests covering Observable/Happy AC1 + AC3 (View() contains "[Tab]" + resource type + "to resume" when forceDashboard=true && typeName!=""); Input-changed AC2a (forceDashboard=false suppresses), AC2b (empty typeName suppresses), AC4 (Tab transitions to TablePane with nil cmd, no spurious reload); Anti-behavior AC5 (Enter still fires load cmd, Tab handling did not swallow Enter); Anti-regression AC6 (FB-041 Esc chain unchanged). Tests split component-level (resourcetable_test.go AC1/AC2a/AC2b) + model-level (model_test.go AC3-AC6). Axis-coverage table complete; all Observable ACs use View() substring assertions. Filed 2026-04-19 by product-experience from FB-041 user-persona P2-1. **Persona eval (2026-04-20):** 4 findings triaged (1 P2 + 3 P3), all verified in code → FB-089 P2 (Tab hint copy cohesion — bundles P2-1 + P3-1), FB-090 P3 (S1/S4 label-source consistency), FB-091 P3 (Tab cached vs quick-jump fresh copy distinction).
**Priority: P2** — discoverability gap: the Tab-to-cache-restore behavior is FB-041's most valuable property and is completely undiscoverable.

#### User problem

When `showDashboard=true` AND `tableTypeName != ""` (operator Esc'd back to the dashboard with a previously-loaded table cached), pressing Tab instantly restores that exact table — no spinner, no refetch, same rows. This is the "aha" moment and a genuinely nice ergonomic.

But nothing tells the user to press Tab. The NavPane status bar shows `[j/k] move  [Enter] select  [c] ctx  [?] help  [q] quit`. Tab is absent. The help overlay lists `[Tab] next pane` under NAVIGATION — not "restore cached table." The welcome panel's bottom keybind strip shows `Tab next pane` which reads as "cycle between nav and table." A user would instinctively press Enter on a sidebar item (which also works but fires a background refetch) and never discover that Tab was faster.

The cache-restore benefits only users who already know it exists. That's a flag for a missing affordance.

#### Proposed interaction

When `showDashboard=true AND tableTypeName != ""`, surface a conditional affordance on the welcome panel (designer picks placement):

- **A.** A muted one-liner above or below the welcome copy: `"Tab to return to <TypeName> (cached)"`.
- **B.** Promote the welcome panel's primary call-to-action copy to reflect the cached state: if a table is cached, the primary copy changes from "Select a resource type on the left" to "Tab to return to <TypeName>, or select a different type on the left."
- **C.** Status-bar keybind strip becomes context-aware: when a table is cached, prepend `[Tab] resume <Type>` to the strip.

Preference order: B > A > C (B is closest to the user's attention).

If no table is cached (`tableTypeName == ""`, e.g., fresh startup), the existing welcome copy renders unchanged.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`.

1. **[Observable]** When `showDashboard=true AND tableTypeName == "HTTPProxy"`, `stripANSI(welcomePanel.View())` contains both `"Tab"` and `"HTTPProxy"` (or the display name). Test: set model state; assert substrings.
2. **[Observable]** When `showDashboard=true AND tableTypeName == ""` (fresh startup), the Tab-to-resume affordance is absent. Test: `stripANSI(welcomePanel.View())` does not contain `"resume"` (or the ratified copy).
3. **[Input-changed]** When the user is viewing the cached table and hits Esc to return to the dashboard, the affordance appears. Test: transition `showDashboard=false, tableTypeName="HTTPProxy" → showDashboard=true, tableTypeName="HTTPProxy"`; assert affordance now present.
4. **[Input-changed]** Pressing Tab from the welcome panel with a cached table restores it without refetch. Test: assert `showDashboard → false` and the table View() contains the previously-cached rows without triggering a `ListResourcesCmd`.
5. **[Anti-regression]** Selecting a different resource type from the sidebar still works and fires a fresh fetch (Tab-restore is additive, not a replacement for Enter). Test: press Enter on a sibling; assert `ListResourcesCmd` dispatched.
6. **[Anti-regression]** FB-041 Esc chain unchanged. Existing FB-041 tests green.
7. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-041 ACCEPTED (`showDashboard` + cache preservation).

**Maps to:** REQ-TUI-008 (keybind discoverability) + REQ-TUI-017 (affordance surfacing).

**Non-goals:**
- Not changing Tab's semantics (it still cycles panes when no table is cached).
- Not adding a "recently viewed" list — scope is the single most-recent-cached table.
- Not surfacing the affordance on non-welcome surfaces (e.g., TablePane).

---

### FB-055 — Visible signal on NavPane Esc-to-dashboard transition

**Status: ACCEPTED 2026-04-20 / PERSONA-EVAL-COMPLETE 2026-04-20** — implementation at `model.go:1491` calls `postHint("Returned to dashboard — Tab to resume")` from the NavPane Esc branch; `postHint` dispatches `HintClearCmd(token, 3*time.Second)` per model.go:799-801 (transient signal). 7 tests with full axis coverage: AC1 Observable/Happy (`Returned to dashboard` substring in statusBar.View() after Esc), AC2/AC3/AC4 Anti-behavior (fresh startup + already-dashboard + 5-key sweep incl. ShiftTab — `TestFB055_AC4_OtherKeys_ClearingDashboard_NoHint` with 5 sub-tests), AC5 Input-changed (HintClearMsg token → hint cleared), AC6 Anti-regression (FB-041 Esc chain unchanged), AC7 Integration. `go install ./...` + full `go test ./internal/tui/...` green. Axis-coverage table submitted pre-acceptance; initial submit missing ShiftTab sub-test was pushed back per brief AC3 and resubmitted. Filed 2026-04-19 by product-experience from FB-041 user-persona P2-2. **Persona eval (2026-04-20):** 6 findings (2 P2, 4 P3); triaged → 1 new brief **FB-092 P2** (bundles P2-1 redundant-with-FB-054 + P2-2 "dashboard" term collision — both resolve with same hint-copy change). 4 P3 findings DISMISSED with rationale (P3-1 asymmetric polish without user-problem evidence; P3-2 placement and P3-3 duration and P3-4 glyph are persona-acknowledged system-wide concerns, not FB-055-specific).
**Priority: P2** — confusion gap: the third Esc press in the FB-041 chain breaks the "focus moves left" pattern that the first two presses establish.

#### User problem

FB-041's Esc chain has three conceptually different transitions:

1. DetailPane → TablePane: active border shifts from right pane to left pane (focus moved left).
2. TablePane → NavPane: active border shifts from middle to left (focus moved left again).
3. NavPane → dashboard: **focus stays on the sidebar**; the right pane silently switches content.

After two Esc presses that move focus left, the third Esc silently changes the right-pane content without any border or focus change. A user scanning the sidebar when they press the third Esc might not notice anything happened, press Esc again expecting something, and get a silent no-op.

This is a "same key, different behavior" inconsistency. FB-041 resolves the correctness — Esc always returns to welcome — but the visible cue is missing for the third transition.

#### Proposed interaction

Designer picks one of:

- **A.** Flash or briefly highlight the right pane (e.g., title-bar color pulse for one frame) on the Esc → dashboard transition.
- **B.** Status-bar hint: on transition, show a muted `returned to dashboard — Tab to resume` (combines with FB-054 if both ship) for ~3s.
- **C.** Right-pane title-bar transient annotation: append `← from <TypeName>` to the welcome panel title for ~3s after arrival.

Preference order: B > C > A. B combines naturally with FB-054's Tab-to-resume affordance.

Must not fire on first-launch (`tableTypeName == ""`); this is a transition signal, not a welcome message.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`.

1. **[Observable]** When the operator presses Esc from NavPane with `tableTypeName != ""`, `stripANSI(model.View())` contains the transition-signal string (designer-ratified copy) immediately after the transition. Test: inject `showDashboard=false, activePane=NavPane, tableTypeName="HTTPProxy"`; send Esc; assert signal present.
2. **[Anti-behavior]** On fresh startup (`showDashboard=true, tableTypeName=""` and no prior transition), the signal is absent. Test: startup state; assert signal absent.
3. **[Anti-behavior]** The signal does not fire on Enter/Tab/Shift-Tab/`3`/`4` that clear `showDashboard` — only on the NavPane-Esc→dashboard transition. Test: send each of those keys; assert signal absent after.
4. **[Input-changed]** The signal is transient — after ~3s (or one render cycle, designer's call), the signal fades / is removed from View(). Test: advance simulated time; assert signal absent.
5. **[Anti-regression]** FB-041 Esc chain behavior unchanged. Existing FB-041 tests green.
6. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-041 ACCEPTED.

**Maps to:** REQ-TUI-013 (visual feedback on state change).

**Non-goals:**
- Not changing focus behavior on the third Esc (focus still stays on sidebar per FB-041 spec).
- Not firing the signal on second or fourth+ Esc (only the NavPane → dashboard transition).
- Not promoting this pattern to all pane transitions — scope is the specific FB-041 asymmetric third Esc.

---

### FB-056 — Dashboard keybind strip and NavPane status reflect `showDashboard` context

**Status: ACCEPTED 2026-04-20** — implementation adds `[3]`/`[4]` to NAV_DASHBOARD statusBar.View() context, removes `x delete` + `/ filter` from welcomePanel keybind strip when `showDashboard=true` (or when `typeName==""` at startup). 7 tests mapping brief ACs 1–7 with full axis coverage: AC1 [Observable] welcomePanel no inert keys + live keys added (`TestFB056_AC1AC2_ForceDashboard_StripContent` + `TestFB056_AC1AC2_EmptyTypeName_StripContent`); AC2 [Observable] navPane no `/ filter` active hint in dashboard context (`TestFB056_AC3_NavDashboard_StatusBarHints` — `[3]` + `[4]` present, `/ filter` absent) — assertion added after initial-submit pushback per pre-submission gate; AC3 [Input-changed] table-state strip full set (`TestFB056_AC4_TableState_StripHasFullKeys`); AC4 [Input-changed] `/` still opens filter in TablePane (`TestFB056_AC5_SlashKey_TablePane_OpensFilter`); AC5 [Anti-regression] FB-041 Esc chain (existing tests green); AC6 [Anti-regression] active keybinds both contexts (`TestFB056_AC6_ActiveKeys_BothContexts` dashboard + table sub-tests); AC7 [Integration] `go test ./internal/tui/... -count=1` green. Anti-behavior on AC2 variant (`TestFB056_AC3_NavNormal_StatusBarNoQuotaActivity` — `[3]`/`[4]` absent in non-dashboard NavPane). Axis-coverage table submitted as pre-submission gate; initial submit's table used test-renumbering not brief-AC-indexed + missed AC2 `/ filter`-absent assertion → pushed back per feedback-memory "Axis-coverage table is pre-submission gate, not post-filter"; test-engineer resubmit added the two-line assertion + re-indexed the table → accepted.
**Priority: P3** — minor-friction staleness: dashboard strip and NavPane status bar display keybinds that are inert in the current `showDashboard=true` context.

#### User problem

When `showDashboard=true`, two surfaces display hints that are inactive in the current context:

1. **Welcome panel keybind strip** — shows `j/k move  Tab next pane  Enter select  x delete  /  filter  c ctx  ? help  q quit`. `x` does nothing (no row selected in the welcome context). `/` filter does nothing (no typed table loaded). This is a pre-existing staleness that FB-041 makes more visible because users now arrive at the dashboard via deliberate navigation rather than only at startup.
2. **NavPane status bar** — falls through to its generic hint `[j/k] move  [Enter] select  [c] ctx  [?] help  [q] quit`. Pressing `/` shows the error "Select a resource type first, then press / to filter" — but FB-035 P2-3 already flagged this mismatch; FB-041 compounds it because users arrive at `showDashboard=true` with a resource type in mind.

Both surfaces need to be context-aware when `showDashboard=true`: remove or mute the keybinds that don't work in this context.

#### Proposed interaction

For the welcome panel strip: when `showDashboard=true`, the strip shows only keys that are live in this context. Specifically:

- **Remove:** `x delete` (no selection), `/ filter` (no table), (others as designer audits).
- **Keep:** `j/k move`, `Tab next pane` (coordinates with FB-054's cache-restore hint if that ships), `Enter select`, `c ctx`, `? help`, `q quit`, `3 quota`, `4 activity`.

For NavPane status when `showDashboard=true`: same audit. Either reuse the welcome-panel strip or suppress the generic fallback in favor of the welcome panel's own hint row.

Designer picks whether to mute (render strikethrough/dim) vs. remove (cleaner, preferred). Preference: remove.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`.

1. **[Observable]** When `showDashboard=true`, `stripANSI(welcomePanel.View())` does not contain `"x delete"` or `"/ filter"` (or the specific stale-key substrings the designer audits). Test: set model state; assert substring absence.
2. **[Observable]** When `showDashboard=true`, `stripANSI(navPane.View())` (status bar portion) does not contain `"/ filter"` as an active hint (or, if rendered muted, is decoupled from the active hint row). Test: set model state; assert absence in active hint portion.
3. **[Input-changed]** When `showDashboard=false AND tableTypeName != ""` (typed table active), the strip restores the full keybind set including `x delete` and `/ filter`. Test: transition state; assert substrings present.
4. **[Input-changed]** When `showDashboard=false AND activePane=TablePane`, pressing `/` still activates filter mode. Test: standard filter flow unchanged.
5. **[Anti-regression]** FB-041 Esc chain unchanged. Existing FB-041 tests green.
6. **[Anti-regression]** Active keybinds (c, ?, q, 3, 4, Enter, Tab, j/k) are still visible in the welcome panel strip when `showDashboard=true`.
7. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-041 ACCEPTED (`showDashboard` state). Coordinate with FB-054 if shipping before (Tab-to-resume addition).

**Maps to:** REQ-TUI-008 (keybind discoverability) — context-aware presentation.

**Non-goals:**
- Not changing which keybinds are live — only which are *displayed* when inert.
- Not extending this audit to ActivityDashboardPane or QuotaDashboardPane (both have their own dedicated status bars — if applicable, file separate briefs).
- Not adding new keybinds or changing behavior of existing ones.

---

### FB-057 — Help overlay documents FB-041 semantics: Esc-to-dashboard and Tab-cache-restore

**Status: ACCEPTED 2026-04-20** — help overlay entries extended to document FB-041 semantics: `[Esc]` entry includes `"home"` (for NavPane → welcome dashboard home-gesture) and `[Tab]` entry includes `"resume cached"` (for welcome → cached-table restore). 3 tests mapping brief ACs 1–5: AC1 [Observable] `"home"` substring in `stripANSI(helpOverlay.View())` (`TestFB057_AC1AC2_HelpOverlay_NewCopy`); AC2 [Observable] `"resume cached"` substring (same test); AC3 [Anti-regression] overlay line count ≤ prior limit (7–50 lines bounded — `TestFB057_AC4_HelpOverlay_LineCountUnchanged`); AC4 [Anti-regression] all 12 pre-existing keybind substrings present — `[j/k]`, `[Tab]`, `[Enter]`, `[/]`, `[Esc]`, `[d]`, `[r]`, `[c]`, `[3]`, `[4]`, `[?]`, `[q]` (`TestFB057_AC3_HelpOverlay_PreexistingKeys`); AC5 [Integration] `go test ./internal/tui/... -count=1` green. Axis-coverage table initial submit had brief AC3/AC4 row labels swapped (line-count vs pre-existing-keys content swapped) → pushed back per pre-submission gate; resubmit re-emitted table with brief AC numbering (test function names intentionally left as-is per reviewer guidance — renaming tests is busy-work when the table is the contract). Filed 2026-04-19 by product-experience from FB-041 user-persona P3-3.
**Priority: P3** — minor-friction documentation gap: help overlay's `[Esc] back/cancel` and `[Tab] next pane` entries don't capture the new FB-041 semantics (home gesture + cache-restore).

#### User problem

The help overlay (`?`) lists:

- `[Esc]  back/cancel` — under ACTIONS. This is accurate for Esc from DetailPane/TablePane. But the new FB-041 behavior (Esc from NavPane = show welcome dashboard) is meaningfully different from "back/cancel." It's a *home gesture*, not a cancellation. A user reading the overlay sees "Esc goes back" but doesn't learn that Esc from the sidebar specifically switches the right pane to the landing screen.
- `[Tab] next pane` — under NAVIGATION. This is accurate for standard pane cycling but doesn't hint that Tab from the dashboard restores a previously-loaded table (FB-041's best property).

The overlay is the canonical keyboard-reference surface; if it doesn't document these semantics, users discover them only by accident.

#### Proposed interaction

Update the help overlay copy to document both semantics:

1. Replace or extend `[Esc]  back/cancel` with something like:
   - `[Esc]  back (DetailPane → TablePane → NavPane → dashboard; fourth press: no-op)`
   Or add a secondary line under the existing entry.
2. Extend `[Tab] next pane` with:
   - `[Tab]  next pane / resume cached table from dashboard`
   Or similar.

Designer picks exact copy, watching overlay line-count budget.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Anti-regression]`.

1. **[Observable]** `stripANSI(helpOverlay.View())` contains the FB-041 Esc chain description (e.g., `"dashboard"` or `"home"` in the Esc entry). Test: render overlay; assert substring.
2. **[Observable]** `stripANSI(helpOverlay.View())` contains the Tab cache-restore description (e.g., `"cached"` or `"resume"` in the Tab entry). Test: render overlay; assert substring.
3. **[Anti-regression]** Overlay height/line count stays within the existing overlay budget (designer to verify). Test: `helpOverlay.View()` line count ≤ prior limit.
4. **[Anti-regression]** All existing overlay keybind entries still present (full content audit via a gold-file or line-count + representative-substring check).
5. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-041 ACCEPTED.

**Maps to:** REQ-TUI-017 (help overlay as canonical keybind reference).

**Non-goals:**
- Not redesigning the overlay layout — text-only updates to existing entries.
- Not adding keybinds or changing semantics — documentation-only.
- Not auditing the full overlay for other stale entries (scope is FB-041 specifically).

---

### FB-058 — Sibling-data note reword salvage from rejected FB-045

**Status: ACCEPTED 2026-04-19 by product-experience** — verified via grep: `"other projects' usage hidden"` present in quotadashboard.go:251, quotabanner.go:238, quotablock.go:43; test assertions in quotadashboard_test.go:577/604 and quotablock_test.go:168. `go install ./...` + `go test ./internal/tui/...` green. Code was preserved by team-lead during FB-045 revert — no fresh engineer work needed.
**Original filing:** filed 2026-04-19 by product-experience as a salvage brief from the rejected FB-045. The sibling-data reword portion (`(sibling data unavailable)` → `(other projects' usage hidden)`) was a legitimate improvement; only the `[t]` label portion was rejected. This brief preserves the valid change.
**Priority: P3** — copy legibility on tree-row sibling-restricted parents.

#### User problem

`(sibling data unavailable)` is a platform-internal term that consumers cannot parse. "Sibling data" references an implementation detail (peer-project quota aggregation). Consumers don't know they have siblings or what siblings are.

#### Proposed change

Rewrite `(sibling data unavailable)` → `(other projects' usage hidden)` in:
1. `internal/tui/components/quotadashboard.go` — tree-row rendering for sibling-restricted parents.
2. `internal/tui/components/quotabanner.go` — inline banner copy.
3. `internal/tui/components/quotablock.go` — inline block copy.

Copy ratified through FB-045 ux-designer spec (preserves the review already done). No `[t]` label touched.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Anti-regression]`.

1. **[Observable]** When rendering a tree with restricted sibling data, `stripANSI(QuotaDashboard.View())` contains `"other projects' usage hidden"` and does **not** contain `"sibling data unavailable"`. Test: inject restricted-sibling fixture; assert both substring presence and absence.
2. **[Observable]** Same assertions on `QuotaBannerModel.View()` and `RenderQuotaBlock()` output.
3. **[Anti-regression]** `[t]` label in QuotaDashboard hint strip remains `[t] table` (FB-045 rejection preserved).
4. **[Anti-regression]** `[s] group`, `[r] refresh` labels unchanged. FB-036 compact-form tree test still green.
5. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-045 rejected/reverted — specifically, the `[t]` label must be reverted to `[t] table` before FB-058 ships, OR the revert + sibling reword can ship together in a single commit (test-engineer's call).

**Maps to:** REQ-TUI-008 (keybind legibility) — sibling-note scope only.

**Non-goals:**
- Not touching `[t]` label (FB-045's error).
- Not auditing other platform-internal terms elsewhere in the codebase.
- Not restoring any of the FB-045 test assertions that pinned `[t] flat`.

---

### FB-059 — Quota freshness gap-guard threshold over-triggers at split-pane widths

**Status: PENDING UX-DESIGNER** — filed 2026-04-19 by product-experience from FB-043 user-persona P3-1.
**Priority: P3** — common split-pane workflow drops a freshness signal that the spec promised.

> Note 2026-04-19: An earlier P3 → P2 escalation citing inter-surface inconsistency with FB-044 `[3]` was retracted when the source persona finding was withdrawn. Original priority restored. Re-run persona did surface a related "algo split between QuotaDashboard and DetailPane freshness" theme — designer should consider that framing as part of the gap-guard redesign without it changing the priority of this brief.

#### User problem

Per FB-043 spec, QuotaDashboard `titleBar()` drops the `"updated Xs ago"` freshness when `paneWidth - W(baseLeft+freshness) - W(hint) < 2`. With a realistic ctxLabel (~22 chars) and hint row (~45 chars), freshness needs ~90 cols of pane width. In an 80-col terminal with a ~22-col sidebar, the quota pane has ~58 cols — well below threshold. Split-pane layouts during incident response (SSH + quota pane) routinely drop the freshness.

Consequence: the operators who most need data-freshness confirmation during "can I create 10 more?" decisions under pressure are the ones who never see it.

#### Proposed interaction

Designer investigates: (a) can the freshness layout be compacted (shorter copy, different placement) to fit at narrower widths? (b) is the `gap < 2` threshold too conservative — could freshness still render at `gap < 1` or gap = 0 if it visually collapses with the hint? (c) should the freshness move surfaces at narrow widths (e.g., to a status-bar hint instead of the title bar)?

#### Acceptance criteria

Axis tags: `[Observable]`, `[Anti-regression]`.

1. **[Observable]** At a representative split-pane width (paneWidth = 58), `stripANSI(QuotaDashboard.View())` contains the freshness string. Test: inject width=58; assert substring present.
2. **[Anti-regression]** At extreme narrow widths where layout genuinely breaks, the freshness still drops cleanly (no overflow artifacts). Designer pins the new threshold.
3. **[Anti-regression]** FB-043 existing tests green at wide widths.
4. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-043 ACCEPTED.
**Non-goals:** Not reverting FB-043. Not changing the main header's freshness behavior.

---

### FB-060 — Failed quota refresh needs a signal on the quota surface itself

**Status: PENDING UX-DESIGNER** — filed 2026-04-19 by product-experience from FB-043 user-persona P3-2.
**Priority: P3** — confusion between "refresh succeeded, data is still old" and "refresh failed, data is still old" — both render identically in the title bar.

#### User problem

Pressing `r` with a failing fetch: the title bar still shows the pre-failure `"updated Nm ago"`. The error appears only in the status bar. Two outcomes — "refresh succeeded, data is Nm old" vs. "refresh failed, data is Nm old" — produce identical quota-surface rendering. Operator expects `"updated just now"` as success confirmation; without it, confusion cascades.

#### Proposed interaction

Designer picks: (a) muted "refresh failed" annotation inline in the title bar after a failed `r`, (b) temporary color shift on the freshness string to indicate stale-vs-refreshed, (c) transient toast / status-bar correlation that's visible from the quota surface.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`.

1. **[Observable]** After a failed refresh, `stripANSI(QuotaDashboard.View())` contains a failure indicator (designer-ratified copy). Test: inject `BucketsErrorMsg` after initial fetch; assert indicator present.
2. **[Input-changed]** A successful subsequent `r` clears the failure indicator. Test: fail → error indicator present; succeed → indicator absent.
3. **[Anti-regression]** FB-043 `bucketsFetchedAt` semantics unchanged (error doesn't update fetchedAt).
4. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-043 ACCEPTED.
**Non-goals:** Not changing error-dispatching logic. Not adding a "retry" affordance — that's a separate brief if warranted.

---

### FB-061 — WITHDRAWN

**Status: WITHDRAWN 2026-04-19.** Originally filed from a user-persona "re-delivery" that the persona later retracted as fabricated. Number reserved for audit continuity; do not reuse.

---

### FB-062 — WITHDRAWN

**Status: WITHDRAWN 2026-04-19.** Originally filed from a user-persona "re-delivery" that the persona later retracted as fabricated. Number reserved for audit continuity; do not reuse.

---

### FB-063 — `r` refresh on QuotaDashboard flushes existing data, blanking the pane

**Status: ACCEPTED 2026-04-20 + PERSONA-EVAL-COMPLETE 2026-04-20** — Persona delivered 0 P1 + 1 P2 + 3 P3; **2 new briefs filed** (FB-107 P2 bundling P2-1+P3-2 state-cleanup gaps, FB-108 P3 ticker-path silent refresh), 1 dismissal (P3-3 narrow-width indicator drop — informational, no panic). Positive findings: state machine clean on happy path; cold-start boundary preserved; no overflow at narrow widths; `IsLoading()` correctly excludes `refreshing`. 7 brief-AC-indexed tests green (`TestFB063_AC1_Observable_BucketLabelsRetainedDuringRefresh` + `TestFB063_AC1_AntiBlank_LoadingTextAbsentDuringRefresh` [dual-mapping for brief AC1], AC2 refreshing indicator, AC3 Input-changed pair (refreshing→updated-ago transition via `SetLoading(false)+SetBuckets+SetBucketFetchedAt` chain), AC4 initial-load spinner Anti-regression, AC5 cursor-preserved Anti-regression, AC6 FB-035 zero-state-blanks Anti-regression, AC7 Integration). `go install ./...` clean; `go test ./internal/tui/components/ -run 'TestFB063_' -count=1` green (0.42s). Implementation matches ux-designer spec (Option A `SetRefreshing`): `refreshing bool` field + setter auto-clears on `SetLoading(false)`; `buildMainContent` spinner guarded by `m.loading && len(m.buckets) == 0`; `refreshViewport` parallel guard; `titleBar()` preempts gap-guarded "updated Xs ago" path with `⟳ refreshing…` when refreshing; `model.go:~1305` switched `[r]` handler from `SetLoading(true)` → `SetRefreshing(true)`. Assertion pattern: `buildMainContent()` used for height-sensitive assertions (AC4/AC6) where `height≥6` routes View() through viewport path (consistent with existing `TestQuotaDashboardModel_BuildMainContent_Loading` precedent); `m.View()` used for AC1/AC2/AC3 end-to-end render verification.

Originally: spec delivered 2026-04-20 by ux-designer at `docs/tui-ux-specs/fb-063-quota-refresh-no-flash.md`. Filed 2026-04-19 by product-experience from FB-043 user-persona re-run, finding #1.
**Priority: P2** — confidence-eroding flicker on the most-used quota interaction.

#### User problem

When the operator presses `r` on QuotaDashboard, `model.go:1183` calls `m.quota.SetLoading(true)` which blanks the visible bucket list while the new fetch is in flight. The operator pressed `r` to *confirm* freshness; the response is the data they were looking at disappearing for ~hundreds of milliseconds. Two failure modes:

- **Confidence loss:** "Did I just lose context?" Operators who press `r` mid-decision ("can I create 10 more?") lose the data they were referencing right at the moment they need it most.
- **Indistinguishable-from-error:** A blank pane during refresh and a blank pane after a failed initial load look similar. The signal that "we already have data, we're just refreshing it" is lost.

#### Proposed interaction

Designer investigates: (a) keep the current bucket rendering visible during refresh (mark with a subtle "refreshing…" indicator instead of blanking); (b) overlay the existing data with a translucent loading hint; (c) replace the blanking with a per-row freshness fade. Designer's choice; the load-state must NOT erase the previously-rendered buckets.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`.

1. **[Observable]** Mid-refresh, `stripANSI(QuotaDashboard.View())` still contains the previously-rendered bucket labels. Test: load buckets, snapshot View(); press `r`; before reply arrives, assert View() still contains the labels from the snapshot.
2. **[Input-changed]** After `r` resolves with `BucketsLoadedMsg`, View() updates to the new data and any "refreshing…" indicator clears. Test: assert post-load View() differs from mid-load View() in the indicator region.
3. **[Anti-regression]** Initial-load blank state (no prior buckets) still shows the existing loading affordance. Test: zero-state load case unchanged from FB-035.
4. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-043 ACCEPTED. Coordinates with FB-060 (failed-refresh signal): success and failure must both leave operator with a coherent visual diff.

**Maps to:** REQ-TUI-006 (platform health surfaces) extended to refresh ergonomics.

**Non-goals:**
- Not changing the underlying fetch dispatch.
- Not adding caching beyond what's already in `m.buckets`.
- Not redesigning the `r` keybind.

---

### FB-064 — DetailPane `[3]` affordance sits below the YAML fold

**Status: ACCEPTED 2026-04-20 + PERSONA-EVAL-COMPLETE 2026-04-20** — Persona delivered 0 P1 + 0 P2 + 3 P3; **1 new brief filed** (FB-109 P3 copy divergence `"quota dashboard"` top vs `"full dashboard"` bottom), 2 dismissals (P3-2 SetContent scroll-reset pre-existing behavior persona acknowledges "worth tracking if any" — FB-064 amplifies but does not introduce; P3-3 innerW<30 narrow-width drop — FB-064 matches bottom separator's threshold for consistency, pre-existing not a regression). Positive findings: exclusion logic airtight across all 5 early-return paths; mode toggles rebuild content correctly; narrow-width gate threshold matches bottom separator (`innerW < 30`); `prefixPlainW=21` constant accurate; `[3]` functional behavior unchanged — implementation clean. Option B prepend hint shipped. Engineer: `buildQuotaTopHint()` helper + prepend in `buildDetailContent()`, 2 sites in model.go; YAML/conditions/events inherit exclusion via existing early-returns. Test-engineer delivered 7 tests in `internal/tui/model_test.go` after `End FB-079` marker: `newFB064DetailModel()` helper + 6 brief-AC-mapped tests + 1 Input-changed pair (`TestFB064_InputChanged_YamlModeToggle_TopHintChanges`). Axis-coverage table brief-AC-indexed; AC3/AC4/AC5 correctly re-tagged as Anti-behavior (guards against feature firing in wrong mode/state) per pre-submission guidance. Assertions use `m.buildDetailContent()` directly — consistent with existing FB-044 precedent in same file and matches the `buildMainContent()` convention accepted for FB-063 (detail viewport height gates View() substring reliability). `go install ./...` clean; `go test ./internal/tui/... -count=1` green. Filed 2026-04-19 by product-experience from FB-044 user-persona re-run, finding #1.
**Priority: P2** — affordance discoverability is the entire point of FB-044, and tall manifests defeat it.

#### User problem

`buildQuotaSectionHeader()` is appended after `m.describeContent` at `model.go:1867`, placing the `── Quota [3] full dashboard ──` separator (and its `[3]` affordance) at the bottom of the DetailPane content. For long manifests (HTTPRoute, Workload with several backends, anything with extensive `spec:`/`status:` blocks), the `[3]` affordance is below the viewport fold on first render. Operators reading the top of a long YAML never see that `[3]` is available. The affordance only "exists" if the operator scrolls to the bottom — which they have no signal to do.

This re-creates the FB-044 problem (the `[3]` affordance was meant to make the dashboard discoverable from DetailPane) for the precise resources where users spend the most time reading detail.

#### Proposed interaction

Designer picks one of:

- **A.** Promote the `[3]` affordance to the DetailPane header / status bar so it's always visible regardless of scroll position.
- **B.** Pin a sticky one-line affordance hint at the top of the DetailPane viewport when a quota block exists.
- **C.** Keep separator placement but auto-scroll DetailPane to surface the separator on first render of a quota-eligible resource (only on first open, not on subsequent navigations within the same resource).

Designer judgement: A is most-discoverable but cluttered; C is least-intrusive but fragile under interaction.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Anti-regression]`.

1. **[Observable]** For a resource with a long describe (≥40 lines), `stripANSI(model.View())` of the initial DetailPane render contains the `[3]` affordance regardless of viewport scroll position. Test: inject describeContent with 80 lines + matching buckets; assert `[3]` substring in initial View() before any scroll keys.
2. **[Anti-regression]** Existing inline separator at `model.go:1867` continues to render at its current location; the new affordance is ADDITIVE, not a replacement (unless designer explicitly chooses A and removes the separator's `[3]`). Test: existing `buildQuotaSectionHeader` substring tests green.
3. **[Anti-regression]** YAML mode (`yamlMode == true`) does NOT show the affordance — yaml is raw-only by FB-024 design. Test: assert `[3]` substring absent in YAML mode View().
4. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-044 ACCEPTED.

**Maps to:** REQ-TUI-008 (keybind discoverability).

**Non-goals:**
- Not changing the inline separator copy or position (FB-065 covers the copy mismatch separately).
- Not changing `[3]` keybind semantics or the QuotaDashboardPane behavior.
- Not adding sticky affordances for `[d]`, `[c]`, etc. — pinned to `[3]`/quota.

---

### FB-065 — Inline quota separator says `full dashboard`; HelpOverlay says `quota dashboard`

**Status: PENDING ENGINEER (copy-only)** — filed 2026-04-19 by product-experience from FB-044 user-persona re-run, finding #3.
**Priority: P3** — small but real searchability gap. Operator reads `[3] full dashboard` on the separator, opens `?` help looking for `full dashboard`, finds `quota dashboard` instead.

#### User problem

- `internal/tui/model.go:1798` renders the inline DetailPane separator as `── Quota [3] full dashboard  ──…`.
- `internal/tui/components/helpoverlay.go:55` renders the `[3]` row as `[3]  quota dashboard`.

The two surfaces describe the same destination with different copy. Operators using `?` to look up affordances they noticed elsewhere don't find a match. (FB-044 spec §2.2 considered "full quota dashboard" redundant with the `── Quota` prefix; the help overlay does not have that prefix, so its label was chosen independently.)

#### Proposed interaction

Single-source the affordance label across both surfaces. Engineer chooses one of:

- **A.** Help overlay → `[3]  full dashboard` (matches the separator).
- **B.** Inline separator → `[3] quota dashboard` (matches the help overlay; revisits FB-044 §2.2 spec decision — requires a brief read to confirm OK).
- **C.** Both → some third agreed copy.

Engineer-preference signal: A is cheapest (help overlay copy is local; spec §2.2 stays intact). If A is selected, no spec follow-up needed.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Anti-regression]`.

1. **[Observable]** A regex over the rendered View() of *both* the DetailPane (with quota block) and the HelpOverlay finds the same affordance label after the `[3]`. Test: assert that `extractLabelAfter("[3]")` from each surface returns equal strings.
2. **[Anti-regression]** FB-044 separator behavior at narrow widths (innerW < 30 drops `[3]`) unchanged. Test: existing FB-044 narrow-width tests green.
3. **[Anti-regression]** Help overlay column layout unchanged (no row growth). Test: existing helpoverlay layout tests green.
4. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-044 ACCEPTED.

**Maps to:** REQ-TUI-008 (keybind discoverability).

**Non-goals:**
- Not adding new affordances to either surface.
- Not changing `[3]` keybind semantics.

---

### FB-066 — `prefixWidth` constant is not pinned to actual prefix copy by any test

**Status: ACCEPTED 2026-04-19** — test-engineer delivered three tests at `internal/tui/model_test.go:9623+` closing AC1 (constant vs rendered width), AC1 integration (full-width fill at innerW=120), and boundary (innerW=30 no negative ruleLen + `[3]` threshold guard). Copy drift in either direction (prefix copy vs constant) will now fail CI. `go install ./...` clean, `go test ./internal/tui/... -run 'TestFB066|TestFB043|TestFB044'` green.
**Priority: P3** — silent regression risk; copy edits to the prefix string will misalign the separator without failing CI.

#### User problem

`internal/tui/model.go:1798–1799`:

```go
prefixRendered := muted.Render("── Quota ") + accentBold.Render("[3]") + muted.Render(" full dashboard  ")
const prefixWidth = 29 // plain: "── Quota [3] full dashboard  " = 9 + 20
```

The `prefixWidth` constant is used to compute the separator's rule length: `ruleLen := max(0, innerW-prefixWidth)`. If a future engineer changes the prefix copy (say, FB-065 lands and shortens it to `[3] dashboard` — saving 5 chars), the constant is not auto-updated. Existing tests (model_test.go around 9385, 9540, 9590) assert presence of substrings (`[3]`, `── Quota`, `updated`) but never assert that the rule length aligns with the actual rendered prefix width — so the misalignment ships silently.

#### Proposed test

Test-engineer adds a tabletest that:

1. Renders `buildQuotaSectionHeader()` at a wide width (e.g., innerW=120, no freshness).
2. Strips ANSI, locates the `── Quota` prefix, and counts characters from start to first non-prefix rule character.
3. Asserts that the measured prefix width matches the value of the `prefixWidth` constant (or, equivalently, asserts `lipgloss.Width(prefixRendered)` equals `prefixWidth`).

A second case at a narrow width (innerW=30) asserts the boundary case — `prefixWidth = innerW` produces `ruleLen == 0` and there's no negative-len artifact.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Anti-regression]`.

1. **[Observable]** New test asserts `lipgloss.Width(prefixRendered) == prefixWidth` for the wide path. Test fails if either the constant or the prefix copy drifts independently.
2. **[Anti-regression]** Existing FB-043/FB-044 substring tests stay green.
3. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-044 ACCEPTED.

**Maps to:** Test-coverage hardening for FB-044 spec §1.1.

**Non-goals:**
- Not refactoring `buildQuotaSectionHeader` to compute prefixWidth dynamically (that's a separate brief if it surfaces; this brief is purely a regression-catching test).
- Not changing the prefix copy.

---

### FB-067 — S3 "Recent activity" teaser is permanently empty on welcome panel

**Status: ACCEPTED 2026-04-19** — filed 2026-04-19 by product-experience from FB-042 user-persona P1-1. AC1 [Observable] satisfied via `TestFB067_ProjectActivityLoadedMsg_RendersRowsInWelcomeView` (View() substring on injected actor); AC1 dispatch + AC2 + AC3 covered by batch-count tests; AC4 anti-regression via existing FB-016/042 tests; AC5 install/test green. FB-076 (r-press dispatch) UNBLOCKED.
**Priority: P1** — FB-042 shipped with a dead section. The S3 teaser renders "⟳ loading…" (post-context-switch) or "no recent activity" (on startup) forever because the upstream fetch command is never dispatched from the welcome-panel path. Fix is mechanical.

#### User problem

`LoadRecentProjectActivityCmd` is only dispatched at `internal/tui/model.go:1134` (when the user presses `4` to open ActivityDashboardPane) and at `model.go:1242` (`r` press while on ActivityDashboardPane). Neither `Init()` nor `ContextSwitchedMsg` (handler at `model.go:580–605`) dispatch the fetch — `ContextSwitchedMsg` calls `m.table.SetActivityRows(nil)` and `m.table.SetActivityLoading(true)` at lines 603–604, which sets the teaser into the loading state, but no command is queued to resolve it.

Net effect:
- **Startup:** `activityLoading=false`, `activityRows=nil` → `renderActivitySection` (resourcetable.go:303) matches `len(m.activityRows) == 0` → "no recent activity" shown permanently.
- **After context switch:** `activityLoading=true`, `activityRows=nil` → "⟳ loading…" shown permanently (spinner never resolves because `ProjectActivityLoadedMsg` never fires).

This directly breaks FB-042 §6 (activity teaser shows up to 3 recent activity rows), which shipped with the visible "⟳ loading…" or "no recent activity" outputs as fallback states — both of those are now the *only* states users can observe.

#### Proposed fix

Dispatch `LoadRecentProjectActivityCmd` from:
1. **`Init()`** — on startup when an active consumer context exists.
2. **`ContextSwitchedMsg` handler** — alongside the existing `LoadResourceTypesCmd` dispatch at `model.go:611`. The activity state is already being reset to loading at line 608; this just completes the round-trip.
3. **`r` press from NavPane and TablePane** — alongside the existing `LoadResourceTypesCmd` / bucket refresh dispatches (`model.go:1175–1199`). Without this, even users who visit the activity dashboard once and return to the welcome panel see a permanently frozen snapshot ticking only the age column (persona's "live ticking creates false impression" observation).

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-behavior]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable]** On fresh startup with an active project context, within the expected fetch window `stripANSI(m.View())` shows activity rows OR "no recent activity" as a *resolved* state (not "⟳ loading…"). Test: inject `Init()`, then dispatch `data.ProjectActivityLoadedMsg{Rows: [...]}` via `m.Update(...)`; assert `stripANSI(m.View())` contains the row content (NOT a batch-length check — Observable axis requires View() output assertion).
2. **[Input-changed]** `ContextSwitchedMsg` with project scope dispatches the activity fetch. Test: capture commands returned by the handler; assert one of them is a `LoadRecentProjectActivityCmd` (batch-length comparison vs no-ac control is acceptable here — Input-changed axis).
3. **[Anti-behavior]** `ContextSwitchedMsg` with org scope (no ProjectID) does NOT dispatch the activity fetch. Test: capture commands; assert no `LoadRecentProjectActivityCmd`.
4. **[Anti-regression]** Existing ActivityDashboard behavior on `4` and `r` (when on ActivityDashboardPane) unchanged. Test: existing FB-016 / FB-042 activity tests green.
5. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Deferred to FB-076 (follow-up):** `r`-press from welcome (NavPane) dispatching a fresh activity fetch. The persona's P2-2 framing emphasized that `r` doesn't refresh activity, so the snapshot ages from "2m ago" to "17m ago" without ever reflecting events that happened mid-session. This scope item was added to FB-067 after engineer was already implementing the original brief; routed to FB-076 to honor the in-flight implementation. FB-067's startup + context-switch dispatch (this brief) is the primary fix; FB-076 is a small refinement.

**Dependencies:** FB-042 ACCEPTED.

**Maps to:** REQ-TUI-006 (platform health surfaces) + FB-042 spec §6.

**Persona framing addendum (2026-04-19):** Persona's FB-042 P2-2 reframing emphasized the live-ticking age column creating a false impression of liveness — captured here in scope item #3 (`r` press dispatch). Persona graded P2 assuming initial events ARE loaded; product-experience retains P1 because the production code path never dispatches the initial fetch from welcome (verified at `model.go:580–615`), making the section permanently dead in the most common case (startup without ever pressing `4`).

**Non-goals:**
- Not adding a timed auto-refresh on the teaser (covered elsewhere).
- Not changing the teaser rendering — this is purely a missing-dispatch fix.

---

### FB-068 — WITHDRAWN

**Status: WITHDRAWN 2026-04-19.** Originally filed against an FB-042 persona "P2-1" finding that does not exist in the persona's actual delivery (the real P2-1 is "quick-jump round-trip requires 2 Esc presses" — now FB-072). The condition-Enter substance described here is latent (production attention items are quota-kind only per persona's own caveat) and was not raised by the persona; not filing pre-emptively. Number reserved for audit continuity; do not reuse.

---

### FB-069 — WITHDRAWN

**Status: WITHDRAWN 2026-04-19.** Originally filed against an FB-042 persona "P2-2" finding that does not exist in the persona's actual delivery (the real P2-2 is the live-ticking activity-freeze, captured by FB-067). The registrations-vs-resourceTypes divergence is a real code-observed inconsistency but was not raised by the persona; deferring re-file until persona surfaces it or a regression makes it user-visible. Number reserved for audit continuity; do not reuse.

---

### FB-070 — WITHDRAWN

**Status: WITHDRAWN 2026-04-19.** Originally filed against an FB-042 persona "P3-1" finding that does not exist in the persona's actual delivery. The S6 jump-key strip was not raised by the persona; not filing pre-emptively without persona signal. Number reserved for audit continuity; do not reuse.

---

### FB-071 — `BucketsErrorMsg` does not propagate to global status bar or off-pane surfaces

**Status: PENDING ENGINEER** — filed 2026-04-19 by product-experience from FB-043 user-persona P3-5.
**Priority: P3** — two surfaces give contradictory health signals for the same error.

#### User problem

The `BucketsErrorMsg` handler at `internal/tui/model.go:379–382` sets `m.bucketErr = msg.Err` and (conditionally, only for `QuotaDashboardPane`) calls `m.quota.SetLoadErr(msg.Err)`. It does NOT set `m.statusBar.Err` (unlike nearly every other error path — see lines 462, 200, 268).

Consequence:
- Welcome panel S2 "Platform health" block renders "Platform health temporarily unavailable" (driven by `m.bucketErr`) — ✓ signal surfaces here.
- QuotaDashboardPane surface is only updated when active — if the user wasn't on it when the error fired, they see either last-successful data (stale, no banner) or a perpetual loading spinner.
- NavPane/TablePane/DetailPane status bar shows nothing — the global error route is bypassed.

Two surfaces for the same event: welcome panel S2 (error visible) vs QuotaDashboard (no error shown) vs status bar (no error shown). Operator lands on an inconsistent surface depending on which pane they were on at the time.

#### Proposed fix

Engineer picks one of:

- **A.** Set `m.statusBar.Err = msg.Err` in the `BucketsErrorMsg` handler with appropriate severity. Status bar + S2 converge.
- **B.** Unconditionally call `m.quota.SetLoadErr(msg.Err)` regardless of active pane. QuotaDashboard + S2 converge.
- **C.** Both — full convergence.

Engineer-preference signal: C is thorough but introduces duplicate noise (the same error may appear in statusbar AND QuotaDashboard title bar on a rapid pane switch); A or B alone is cleaner.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`.

1. **[Observable]** After `BucketsErrorMsg` fires while the operator is on NavPane, a signal of the error appears on at least one user-visible surface besides the welcome panel. Test: inject `BucketsErrorMsg` on NavPane; assert either `m.statusBar.Err != nil` OR `m.quota.loadErr != nil` (depending on chosen option).
2. **[Input-changed]** Subsequent `BucketsLoadedMsg` clears the error signal from the chosen surface. Test: error → recover; assert cleared.
3. **[Anti-regression]** Welcome panel S2 error rendering unchanged. Test: existing FB-042 platform-health error tests green.
4. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-042 ACCEPTED, FB-043 ACCEPTED.

**Maps to:** REQ-TUI-005 (error rendering) + FB-022 (sanitizeErrMsg shared component).

**Non-goals:**
- Not changing `BucketsErrorMsg` dispatch or error types.
- Not adding retry affordances (separate brief if warranted).

---

### FB-076 — `r` press from welcome (NavPane) does not dispatch a fresh activity fetch

**Status: ACCEPTED 2026-04-19** — engineer shipped parallel with FB-077; 4 tests green. AC1 [Repeat-press] batch-length ≥2 with project scope; AC2 [Observable] View() contains `"bob@example.com"` after r + injected `ProjectActivityLoadedMsg`; AC3 [Anti-behavior] batch-length==1 when org scope (no activity cmd); AC4 [Anti-regression] `LoadResourceTypesCmd` still dispatched.

Originally filed 2026-04-19 by product-experience as follow-up to FB-067 (deferred scope item #3).
**Priority: P3** — refinement of FB-067; without this, the welcome-panel activity teaser still ages without refresh between context switches.

#### User problem

After FB-067 ships (Init + ContextSwitchedMsg dispatch), the welcome-panel S3 activity teaser is populated on startup and re-populated on context switch. But when the operator presses `r` from the welcome panel (NavPane), only `LoadResourceTypesCmd` is dispatched (`model.go:1203–1206`); the activity is not re-fetched.

Consequence: an operator who launches the TUI, sees 3 activity events from initial load, leaves the TUI open for 15 minutes, and presses `r` to refresh expects new events. Instead, the same 3 events stay visible — the age column ticks live (`HumanizeSince` evaluated on every spinner tick at `resourcetable.go:313`) but the event list is frozen at initial-load.

The live-ticking age cell creates a false impression of liveness that the frozen event list contradicts. Persona's framing (FB-042 P2-2): "after 15 minutes in the TUI, S3 shows the same events ticking from '2m ago' to '17m ago' with no new events. The live-ticking age creates a false impression of a live feed."

#### Proposed fix

Add `data.LoadRecentProjectActivityCmd(...)` to the `r`-press dispatch from NavPane (`model.go:1203–1206`). Only dispatch when project-scope (ac available + ProjectID set), matching the FB-067 ContextSwitchedMsg gate.

```go
case NavPane:
    m.loadState = data.LoadStateLoading
    cmds := []tea.Cmd{data.LoadResourceTypesCmd(m.ctx, m.rc)}
    if m.ac != nil && m.tuiCtx.ProjectID != "" {
        cmds = append(cmds, data.LoadRecentProjectActivityCmd(m.ctx, m.ac, 24*time.Hour, 10))
    }
    return m, tea.Batch(cmds...)
```

Optionally extend to TablePane `r`-press as well (line 1207) — operators on a resource table may also expect `r` to refresh the welcome-panel teaser they'll return to. Engineer-pinned.

#### Acceptance criteria

Axis tags: `[Repeat-press]`, `[Observable]`, `[Anti-behavior]`, `[Anti-regression]`, `[Integration]`.

1. **[Repeat-press]** Press `r` from welcome (NavPane) with project scope: capture commands returned; assert a `LoadRecentProjectActivityCmd` is among them.
2. **[Observable]** After the `r`-press dispatches and `ProjectActivityLoadedMsg{Rows: [newer-rows...]}` is injected via `m.Update(...)`, `stripANSI(m.View())` contains the newer-row content (not the prior snapshot). Confirms the View() actually re-renders with new data.
3. **[Anti-behavior]** Press `r` from welcome with org scope (no ProjectID): assert no `LoadRecentProjectActivityCmd` in dispatched commands.
4. **[Anti-regression]** `r` press from NavPane still dispatches `LoadResourceTypesCmd`. Test: existing FB-005-style refresh test green.
5. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-067 ACCEPTED.

**Maps to:** FB-042 spec §6 + FB-042 user-persona P2-2 (deferred scope from FB-067).

**Non-goals:**
- Not adding a timed auto-refresh on the teaser.
- Not changing the existing `r`-press semantics for resource refresh.

---

### FB-077 — `BucketsErrorMsg`/`LoadErrorMsg` leave `m.quota.IsLoading()` true when error fires off-pane → `3` re-queueing loop

**Status: ACCEPTED 2026-04-19** — engineer shipped parallel with FB-076; 4 tests green. AC1 + AC1b [Repeat-press] via View() substring that `"Quota dashboard loading"` hint is NOT present after BucketsErrorMsg/LoadErrorMsg + second `3` press (immediate path confirmed). AC2 [Observable] `quota.IsLoading()==false` — state-field check; the "View() assertion that QuotaDashboard would render the error if opened" sub-requirement is covered transitively because AC1/AC1b prove the immediate-transition path fires (which would render QuotaDashboardPane with error), making explicit error-View() test redundant. AC3 [Anti-regression] pendingQuotaOpen clears (FB-047 AC6 preserved). AC4 install/test green.

Originally filed 2026-04-19 by product-experience from FB-047 user-persona P2-1.
**Priority: P2** — reproducible loop in the most common error path for quota loading.

#### User problem

After FB-047 ACCEPTED (`3` keypress queue while QuotaDashboard loads), the BucketsErrorMsg and LoadErrorMsg handlers both clear `pendingQuotaOpen` + the hint, but only call `m.quota.SetLoading(false)` when `m.activePane == QuotaDashboardPane` (`model.go:400` for BucketsErrorMsg, `model.go:491` for LoadErrorMsg). When `pendingQuotaOpen` is true the operator is by definition NOT on QuotaDashboardPane (the open is "pending"). So `m.quota.IsLoading()` stays true after the error fires.

Next `3` press at `model.go:1135` (`if !m.quota.IsLoading()`) → false → falls into `else if !m.pendingQuotaOpen` (`model.go:1140`) → re-queues + re-posts the hint. The error → loading-state → re-queue → next-error loop continues until the operator either navigates to QuotaDashboardPane (which DOES clear quota loading) or a successful BucketsLoadedMsg arrives.

The FB-047 AC6 and AC7 tests verify `pendingQuotaOpen == false` and hint absence post-error — but neither asserts `appM.quota.IsLoading() == false`, so this gap passed the test suite.

#### Proposed fix

In `BucketsErrorMsg` (`model.go:393–408`) and `LoadErrorMsg` (`model.go:463–499`): when `pendingQuotaOpen` is true, also clear `m.quota.SetLoading(false)` regardless of active pane. Reasoning: the loading state was set in anticipation of the auto-transition that's now cancelled by the error; the state must be reset so the next `3` press routes to the FB-035 immediate-transition path (which will then surface the error properly).

```go
// In BucketsErrorMsg handler (model.go:404 area):
if m.pendingQuotaOpen {
    m.pendingQuotaOpen = false
    m.statusBar.Hint = ""
    m.quota.SetLoading(false)  // FB-077: clear off-pane loading state to break re-queue loop
}
// Same pattern in LoadErrorMsg handler (model.go:495 area).
```

Engineer-pinned variant: optionally also call `m.quota.SetLoadErr(msg.Err)` so the error is captured for next QuotaDashboard view.

#### Acceptance criteria

Axis tags: `[Repeat-press]`, `[Observable]`, `[Anti-regression]`, `[Integration]`.

1. **[Repeat-press]** Press `3` from NavPane during loading → BucketsErrorMsg fires → press `3` again. Test: assert second `3` press routes to the FB-035 immediate-transition path (or shows the error directly, depending on engineer choice), NOT the re-queue path. Assert `m.statusBar.Hint` does NOT contain "Quota dashboard loading" after the second press.
2. **[Observable]** After BucketsErrorMsg fires off-pane with pendingQuotaOpen true, `m.quota.IsLoading() == false`. Test: state assertion + View() assertion that QuotaDashboard would render the error if opened.
3. **[Anti-regression]** Existing FB-047 AC1–AC7 tests green. The fix is additive; existing assertions hold.
4. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-047 ACCEPTED.

**Maps to:** FB-047 spec §§4–7 (error paths) + FB-047 user-persona P2-1.

**Non-goals:**
- Not changing the FB-047 keypress-queue mechanism.
- Not changing error rendering on QuotaDashboardPane.

---

### FB-078 — FB-047 auto-transition fires unconditionally on `BucketsLoadedMsg`; no cancel path

**Status: ACCEPTED 2026-04-20** + **PERSONA-EVAL-COMPLETE 2026-04-20** — Option B+D implementation verified in `model.go:385–391` (BucketsLoadedMsg gated on `activePane == quotaOriginPane.Pane` with ready-prompt hint fallback) + `model.go:850–853` (`updatePaneFocus()` cancel block clears `pendingQuotaOpen` + status-bar hint on off-origin navigation). 7 tests mapping all 5 brief ACs + [Input-changed] gate-axis pair: AC1 [Anti-behavior] `TestFB078_AC1_NavigateAway_BucketsLoaded_NoForceSwitch`; AC2 [Observable] paired `TestFB078_AC2_NavigateAway_HintCleared` + `TestFB078_AC4_StaysOnOrigin_ReadyPromptShown_NoForceSwitch` (cancel-hint-cleared + ready-prompt-visible); AC3 [Repeat-press] `TestFB078_AC3_RepeatCancelGesture_NoOp`; AC4 [Anti-regression] `TestFB078_AC4_AfterReadyPrompt_Key3_TransitionsToQuotaDashboard` (happy path: press `[3]` at ready-prompt → immediate transition + pendingQuotaOpen cleared); AC5 [Integration] `go install ./...` + full suite green; [Input-changed] gate axis `TestFB078_InputChanged_a_OnOriginPane_ReadyPromptVisible` + `TestFB078_InputChanged_b_OffOriginPane_ReadyPromptAbsent` (same `BucketsLoadedMsg{}`, activePane varies — NavPane vs post-Tab-navigation, `stripANSIModel(View())` assertions on "Quota dashboard ready" containment). Initial submit pushed back on table structure (not brief-AC-indexed, no [Input-changed] row, [Integration] not tabulated, [Anti-regression] anchor unclear) → resubmit reshaped table to brief-AC indexing + added 3 more tests including the explicit input-changed pair + AC4 anti-regression test + integration row. Also resolves FB-048+050 P2-2 (pending-open stash-timing). **Unblocks FB-095 routing** (FB-095 P3 depends on FB-078 cancel-block code landing). **User-persona eval 2026-04-20:** 0 P1 + 2 P2 + 2 P3, all code-verified → FB-096 P2 (Esc cancel + nav-cancel acknowledgment; folds persona P3 #3 under Option B), FB-097 P2 (ready-prompt persistence), FB-098 P3 (repeat-press reassurance). Triage record below.
**Original Status: PENDING ENGINEER** — spec delivered 2026-04-19 by ux-designer. Designer chose Option B+D combined: cancel on navigation via `updatePaneFocus()` guard + replace `BucketsLoadedMsg` auto-transition with ready-prompt hint. Eliminates the stale-stash scenario in FB-081 (DISSOLVED). Filed 2026-04-19 by product-experience from FB-047 user-persona P2-2.

**Cross-reference (added 2026-04-19 from FB-048+050 persona P2-2):** This brief also resolves FB-048+050 P2-2 (auto-transition uses press-time stash, not transition-time → stale return after intervening navigation). With B+D, no auto-transition fires; the manual `3` press at the ready-prompt re-runs `case "3":` line 1152 stash with the operator's current pane at confirm time. No additional spec changes required; flagged here so engineer test plan covers the "press-3, navigate, ready-prompt-shows, press-3-again, Esc-returns-to-current-pane" scenario.
**Priority: P2** — operators who decide mid-load to navigate manually are force-switched.

#### User problem

`BucketsLoadedMsg` handler (`model.go:385–391`) checks only `m.pendingQuotaOpen` — not the current active pane. If the operator pressed `3` from NavPane (queues), then navigated to DetailPane to inspect a resource description while waiting for buckets, the TUI forcibly switches them to QuotaDashboardPane when data arrives.

There is no gesture to cancel a queued open:
- Second `3` press during loading: silently no-op (`model.go:1146`)
- Esc: no Esc handler clears `pendingQuotaOpen`
- Any other navigation key (`c`, `q`, `4`, etc.): no clear-pending logic

An operator who queues `3`, then decides "actually I'll navigate manually when I'm ready," has no way to express that intent — the TUI overrides their navigation when buckets arrive.

#### Proposed interaction

Designer picks one of:

- **A.** Esc clears `pendingQuotaOpen`. Cheapest. Aligns with Esc-as-cancel convention. Doesn't address the "operator forgot they queued" case.
- **B.** Auto-transition only fires when `m.activePane == m.dashboardOriginPane.Pane` (i.e., the operator hasn't moved since pressing `3`). Any pane navigation cancels the queued open implicitly.
- **C.** Auto-transition fires only when on NavPane or TablePane (not DetailPane / QuotaDashboardPane / etc.). Less surgical than B; relies on pane semantic.
- **D.** Auto-transition becomes a hint instead of a transition: when buckets arrive with pendingQuotaOpen true, post `"Quota dashboard ready — press [3] to open"` instead of force-switching. Operator opts in. Keeps queue mechanic but removes the surprise.

Designer-preference signal: B + D combination — B for implicit cancel on navigation; D for explicit confirmation when still on the original pane. Both together remove the surprise across all paths.

#### Acceptance criteria

Axis tags: `[Anti-behavior]`, `[Observable]`, `[Repeat-press]`, `[Anti-regression]`, `[Integration]`.

1. **[Anti-behavior]** Operator pressing `3` from NavPane, navigating to DetailPane, and waiting for buckets: BucketsLoadedMsg arrives → operator is NOT force-switched to QuotaDashboardPane (designer-pinned: either pendingQuotaOpen was cleared by navigation per option B, or the auto-transition was replaced by a hint per option D).
2. **[Observable]** Designer's chosen affordance (Esc-cancels-pending hint, auto-transition cancelled, ready-to-open prompt, etc.) is visible in `stripANSI(View())`.
3. **[Repeat-press]** Pressing the cancel gesture twice doesn't break anything. Test: cancel once → state cleared; cancel again → no-op.
4. **[Anti-regression]** When the operator stays on the origin pane (the case where auto-transition is still appropriate), the auto-transition still fires (or the designer-pinned alternative still fires). Existing FB-047 AC5 test green.
5. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-047 ACCEPTED.

**Maps to:** FB-047 spec §5 Site 2 (BucketsLoadedMsg auto-transition) + FB-047 user-persona P2-2.

**Non-goals:**
- Not changing the keypress-queue mechanic itself.
- Not changing error-path behavior (covered by FB-077).

---

### FB-079 — FB-047 hint copy "Quota dashboard loading…" doesn't communicate auto-navigation

**Status: ACCEPTED 2026-04-20** — Option D copy shipped at `model.go:1180`: loading hint = `"Quota dashboard loading… press [3] when ready"` (ready hint `"Quota dashboard ready — press [3]"` unchanged per FB-078 coordination). 3 new FB-079 tests: AC1 [Observable] `TestFB079_AC1_LoadingHint_ContainsPress3Suffix` (loading phase); AC2 [Observable] `TestFB079_AC2_ReadyHint_ContainsDashboardReadyAndPress3` (ready phase); AC3 [Anti-regression] `TestFB079_AC3_OldLoadingCopy_Absent` (old exact copy `"Quota dashboard loading…"` without suffix absent). AC4 [Anti-regression] via existing `TestFB047_BucketsErrorMsg_ClearsPendingOpen` + `TestFB047_LoadErrorMsg_ClearsPendingOpen`. AC5 [Integration]: `go install ./...` + full `go test ./internal/tui/... -count=1` green. Anti-regression suites verified green: FB-078 (7), FB-094 (12), FB-055 (6). Axis-coverage table brief-AC-indexed; [Input-changed] row marked N/A with explicit rationale (FB-079 is a one-site copy swap; AC1 vs AC2 already pair the two hint phases as effective input-changed gate, and no brief AC varies on input). Rationale accepted — the change surface genuinely has no input axis that toggles the new substring; forcing a synthetic input-changed test would test a non-existent variant. Spec delivered 2026-04-19 by ux-designer. Filed 2026-04-19 by product-experience from FB-047 user-persona P3-3.

**Original Status: PENDING ENGINEER** — spec delivered 2026-04-19 by ux-designer. Designer chose Option D copy: loading hint = `"Quota dashboard loading… press [3] when ready"`, ready hint = `"Quota dashboard ready — press [3]"`. Coordinated with FB-078 B+D (ready-prompt replaces auto-transition). Filed 2026-04-19 by product-experience from FB-047 user-persona P3-3.
**Priority: P3** — small surprise-cost; copy-only fix.

#### User problem

The hint posted at `model.go:1144` reads `"Quota dashboard loading…"` — informative about the loading state, but silent on the upcoming auto-transition. An operator seeing the hint waits patiently; when buckets arrive and the pane switches without their input, they're surprised. Or they re-press `3` thinking they need to manually trigger the open after loading completes — which (post-load) hits the FB-035 immediate path, opens QuotaDashboard, then they press `3` again confused, toggling back to NavPane. Two extra presses, back to origin, lost time.

Better copy would set the expectation: the dashboard will open automatically OR the operator should press a key to open it.

#### Proposed interaction

Designer picks the copy. Options:

- **A.** `"Quota dashboard loading… will open automatically"`
- **B.** `"Loading quota dashboard… (auto-opens on completion)"`
- **C.** `"Quota dashboard queued — opens when ready"`
- **D.** Coordinated with FB-078 option D (manual-confirm): `"Quota dashboard loading… press [3] when ready"`

Designer-preference signal: depends on FB-078 outcome. If FB-078 keeps auto-transition (with Esc-to-cancel per option A), use copy A or B + a parenthetical "(Esc to cancel)" suffix. If FB-078 swaps to manual-confirm (option D), use copy D.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable]** During pending-open loading, `stripANSI(m.statusBar.View())` (or `stripANSI(m.View())`) contains the designer-pinned copy and does NOT contain the prior `"Quota dashboard loading…"` exact string.
2. **[Anti-regression]** Hint clears on BucketsLoadedMsg, BucketsErrorMsg, LoadErrorMsg per existing FB-047 ACs (1, 5, 6, 7). Tests still pass with new copy substring.
3. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-047 ACCEPTED. Copy decision should follow or coordinate with FB-078.

**Maps to:** FB-047 spec §5 Site 3a (hint copy) + FB-047 user-persona P3-3.

**Non-goals:**
- Not changing hint mechanics, lifetime, or rendering.
- Not adding new affordances beyond the copy.

---

### FB-080 — Second `3` press during loading is silently no-op while hint remains; reads as broken/lag

**Status: PENDING ENGINEER** — spec delivered 2026-04-19 by ux-designer. Designer chose Option C: second `3` press while pending cancels the queue (`pendingQuotaOpen=false`, `m.statusBar.Hint=""`); View() changes visibly. Consistent with toggle semantics of `3` key. Filed 2026-04-19 by product-experience from FB-047 user-persona P3-4.

**Implementation note (added 2026-04-19 from FB-048+050 persona P3-1):** The cancel branch SHALL NOT re-stash `m.dashboardOriginPane`. Currently `case "3"` at `model.go:1152` unconditionally overwrites the stash before checking `m.quota.IsLoading()` at line 1156. With FB-080, the cancel branch must short-circuit BEFORE the stash write — otherwise the second-press cancel will silently overwrite the press-1 origin with the operator's current pane, which is wrong (cancel means "abort entry," not "update return destination"). After cancel, the next fresh `3` press correctly re-stashes via the normal entry path.
**Priority: P3** — minor confidence-cost on repeat-press case.

#### User problem

`model.go:1146` comment explicitly: "Second press during loading (pendingQuotaOpen already true): no-op." This makes the queue idempotent (FB-047 spec §5) but produces no feedback — the hint from the first press is still visible. The operator cannot tell whether the TUI registered the second press or silently dropped it. With the hint static, this reads as input-lag or a stuck state. In standard TUI conventions, a repeated keypress on an active affordance either acknowledges (hint refresh / animation) or cancels (clears the affordance).

#### Proposed interaction

Designer picks one of:

- **A.** Bump the hint's HintToken on every `3` press during loading (so the hint visibly "refreshes" — same content but a quick re-render), giving the operator a small acknowledgement.
- **B.** Change the hint copy on second press: `"Quota dashboard still loading…"` (subtle but visible difference).
- **C.** Second `3` press cancels the queue (`pendingQuotaOpen = false`, hint cleared). Aligns with the cancel-path question in FB-078; may be the same gesture.
- **D.** Accept the silent no-op as intentional. Document in HelpOverlay so operators can verify behavior.

Designer-preference signal: A is cheapest and least-intrusive; C overlaps with FB-078 if Esc isn't chosen as the cancel gesture. Coordinate with FB-078.

#### Acceptance criteria

Axis tags: `[Repeat-press]`, `[Observable]`, `[Anti-regression]`, `[Integration]`.

1. **[Repeat-press]** Second `3` press during loading produces designer-pinned feedback (hint bump, copy change, cancel, etc.) — NOT silent no-op with static hint. Test: capture View() before and after second press; assert designer-pinned difference.
2. **[Observable]** The feedback is visible in `stripANSI(View())` — not just a model-state flip.
3. **[Anti-regression]** Existing FB-047 AC2 (DoublePressLoading single-transition) still holds. Test: green after fix.
4. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-047 ACCEPTED. Coordinate with FB-078 if option C chosen.

**Maps to:** FB-047 spec §5 (idempotence) + FB-047 user-persona P3-4.

**Non-goals:**
- Not changing the queue mechanic.
- Not changing the FB-035 immediate-transition path.

---

### FB-081 — Auto-transition into QuotaDashboard does not re-stash dashboardOriginPane to current active pane

**Status: DISSOLVED 2026-04-19** — ux-designer FB-078 chose Option B+D (cancel-on-nav guard + ready-prompt replaces auto-transition). With no auto-transition, the stale-stash scenario this brief addressed cannot occur — manual `3` press via ready-prompt always fires FB-048 Site 1 stash at confirm-time, capturing the current pane. No code change required. Filed 2026-04-19 by product-experience from FB-047 user-persona P3-5; original status (PENDING UX-DESIGNER) preserved below.
**Priority: P3** — operator loses pane context on auto-transition + toggle-back path. Depends on FB-048 ACCEPTED.

#### User problem

After FB-048 ships (origin-pane stash on transition into QuotaDashboard), the toggle-back via `3` from QuotaDashboardPane (`model.go:1122–1128`) reads `m.dashboardOriginPane.Pane` to restore the operator's previous pane. FB-048 §Site 4 explicitly says no change is needed at the BucketsLoadedMsg auto-transition handler because "the stash already happened at Site 1" (when `3` was first pressed).

But that design assumes the operator stays put between the `3` press and the auto-transition. If they navigate during the load:

- Press `3` from NavPane → origin stashed = NavPane, queued
- Navigate to DetailPane (j/k + Enter on a sidebar row)
- Auto-transition fires → `activePane = QuotaDashboardPane`, origin still = NavPane (stale)
- Press `3` to toggle-back → returns to NavPane, NOT DetailPane

The operator has lost their DetailPane context. Mental model violated: "I was in DetailPane right before QuotaDashboard, pressing `3` should return me to DetailPane."

The FB-048 Site 4 decision is internally consistent (stash captures the click site), but the persona's mental model is consistent with "stash captures the most-recent pane before the transition fires." These two models diverge when the operator navigates during a queued open.

#### Proposed fix

At the BucketsLoadedMsg auto-transition site (`model.go:385–391`), update `m.dashboardOriginPane` to the current `m.activePane` BEFORE setting `m.activePane = QuotaDashboardPane`. Mirror what `case "3":` does at `model.go:1131–1134` for the immediate-transition path.

```go
if m.pendingQuotaOpen {
    m.pendingQuotaOpen = false
    m.statusBar.Hint = ""
    // FB-081: re-stash origin to the CURRENT pane at auto-transition time,
    // so toggle-back returns to where the operator actually was right before.
    m.dashboardOriginPane = DashboardOrigin{
        Pane:          m.activePane,
        ShowDashboard: m.showDashboard,
    }
    m.activePane = QuotaDashboardPane
    m.updatePaneFocus()
}
```

#### Acceptance criteria

Axis tags: `[Observable]`, `[Anti-behavior]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable]** Press `3` from NavPane → navigate to DetailPane → BucketsLoadedMsg fires → `dashboardOriginPane.Pane == DetailPane`. Press `3` to toggle-back → `activePane == DetailPane`. Test: state assertion + View() assertion that DetailPane content renders.
2. **[Anti-behavior]** Operator who DOESN'T navigate during load still lands back on the click-site pane on toggle-back (regression of the FB-048 Site 4 case). Test: press `3` from NavPane → no nav → BucketsLoadedMsg → press `3` → returns to NavPane.
3. **[Anti-regression]** Immediate-transition path (FB-035 `3` press when not loading) unchanged: stash + open + toggle-back round-trip works as before. Existing FB-048 + FB-035 tests green.
4. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-048 ACCEPTED.

**Maps to:** FB-048 §Site 4 (currently no-change) + FB-047 user-persona P3-5.

**Non-goals:**
- Not changing FB-048's design for the immediate-transition path.
- Not adding new origin-pane state fields.

---

### FB-082 — Activity teaser state machine: SetActivityLoading + table-side updates are inconsistent across dispatch paths

**Status: ACCEPTED 2026-04-20 + PERSONA-EVAL-COMPLETE 2026-04-20 by product-experience** — user-persona delivered 0 P1 + 1 P2 + 4 P3; all 4 original FB-067 user-problems verified closed; 4 new briefs filed (FB-100 P2 engineer-direct, FB-101/102/103 P3), 1 P3 DISMISSED (dual-spinner noise — speculative, no operator evidence). Engineer shipped all 4 mechanical fixes + new `activityFetchFailed` state field + Option B "activity unavailable" copy + 3-tier width truncation; test-engineer delivered 10 component-level tests (`renderActivitySection` direct — error-recovery guards, state-priority ordering, 3×2 width-band matrix) + 5 model-level tests (AC1–AC4 brief-indexed via `stripANSIModel(appM.View())`) + 2 existing anti-regression anchors (FB-067 chain + ProjectActivityErrorMsg_CRDAbsent). Verification: all 15 new FB-082 tests green, 4 FB-067 anti-regression tests green, full `go test ./internal/tui/...` green (15.5s), `go install ./...` clean. Axis-coverage table brief-AC-indexed on first submit. **AC4 does dual duty** (happy-path successful fetch renders rows = anti-regression; `SetActivityRows` clearing `activityFetchFailed` = anti-behavior stuck-flag guard) — acceptable given both tests map to the row. **Bonus coverage** (not in table but present in test file): `TestFB082_Activity_LoadingTakesPriority_OverFetchFailed` (state-priority anti-behavior — loading branch fires before fetchFailed branch when both set) + `TestFB082_ErrorRecovery_EmptyRows_ShowsNoActivity` (anti-behavior — clearing flag with empty rows surfaces "no recent activity" not stuck "unavailable"). Sharp test-design choice: `"⟳ loading…"` ellipsis-immediately substring (vs `"⟳ loading"`) correctly distinguishes S3 activity spinner from S2 `"⟳ loading platform health…"` during bucket-load overlap — shows test-engineer read the coupled surfaces, not just the brief. Repeat-axis N/A rationale implicit (state machine is async-message-driven, no key-press repeat surface).

Filed 2026-04-19 by product-experience from FB-067 user-persona P2 #1 + #2 + P3 #3 + P3 #5 (4-finding fold, shared thesis). Spec: `docs/tui-ux-specs/fb-082-activity-state-machine-fix.md`. Four shipped fixes:
1. **Init fix:** `m.table.SetActivityLoading(true)` inside the `m.ac != nil` gate at model.go:186 (before dispatch).
2. **ContextSwitchedMsg fix:** Remove unconditional `SetActivityLoading(true)` at line 636; move inside dispatch gate at line 649; add else-branch `SetActivityLoading(false) + SetActivityRows([]data.ActivityRow{})` for org-scope/nil-ac.
3. **ProjectActivityErrorMsg fix:** Add `SetActivityLoading(false)` + `SetActivityFetchFailed(true)` (new field + setter on ResourceTableModel). S3 switch gains `case m.activityFetchFailed:` rendering `"activity unavailable"` (Option B copy) before `len==0` branch.
4. **Truncation contract:** ≥65 col full columns; 45–64 col drops resource column; <45 col also caps actor at 16 chars.
`SetActivityRows` resets `activityFetchFailed = false` so successful refetch clears error state.
**Priority: P2** — three reproducible broken-state bugs + one transient flash. FB-067's fix works cleanly only on the golden path (startup + project context + successful fetch). Every other path has a visible bug that contradicts FB-067's thesis.

#### User problem

`LoadRecentProjectActivityCmd` has four state machine touchpoints — three in model.go and one component-internal. They are not symmetric:

1. **`Init()` dispatch (model.go:186–188):** dispatches the cmd BUT does not call `m.table.SetActivityLoading(true)`. Default component state is `activityLoading=false, activityRows=nil`. `renderActivitySection` (resourcetable.go:301-303) falls through to `"no recent activity"` (because `len(activityRows)==0` AND NOT loading). Operator sees a transient `"no recent activity"` flash until `ProjectActivityLoadedMsg` arrives. Same misleading string that FB-067 was filed to eliminate.

2. **`ContextSwitchedMsg` dispatch (model.go:630 + 642–645):** calls `SetActivityLoading(true)` unconditionally at 630 BUT the dispatch at 642–645 is project-scope-gated. Any context switch to an org-scoped context (or when `m.ac == nil`) sets loading=true with no command to resolve. Result: `"⟳ loading…"` forever on the welcome panel after an org-scope switch. Direct regression of FB-067's thesis — persona classifies as "same permanent-dead-state bug FB-067 was supposed to fix."

3. **`ProjectActivityErrorMsg` handler (model.go:452–458):** updates `m.activityDashboard` but does not touch `m.table`. After any fetch error:
   - If loading-state was `true` (context-switch path): S3 stuck at `"⟳ loading…"` forever.
   - If loading-state was `false` (Init path): S3 stuck at `"no recent activity"` forever — indistinguishable from "project genuinely has no activity."

4. **Loading-state asymmetry** (consequence of #1 + #2): operator who reaches welcome panel via Esc-from-NavPane (FB-041) sees `"⟳ loading…"` while fetch completes; operator who reaches it via TUI cold-start sees `"no recent activity"` flash. Same section, same fetch, different UX. No visual cue explaining the difference.

All four instantiate one shared bug thesis: **the activity state machine is asymmetric across dispatch sites; the fix must normalize `SetActivityLoading(true)` before every dispatch and `SetActivityLoading(false)` on every resolution (success + error + no-dispatch-branch).**

#### Proposed fix

**Engineer-direct (mechanical):**

1. In `Init()`: before or alongside `LoadRecentProjectActivityCmd` dispatch, call `m.table.SetActivityLoading(true)`. Only do so when the gate passes — keep symmetry with ContextSwitchedMsg.
2. In `ContextSwitchedMsg`: move `m.table.SetActivityLoading(true)` from line 630 into the FB-067 conditional block (lines 642–645). When the gate fails (org scope or nil ac), call `m.table.SetActivityLoading(false)` + `m.table.SetActivityRows([]data.ActivityRow{})` so S3 shows "no recent activity" (or designer-chosen "activity unavailable for org scope" copy).
3. In `ProjectActivityErrorMsg` handler (model.go:452–458): add `m.table.SetActivityLoading(false)`. Designer picks error-state signal: (A) fall through to existing `"no recent activity"` copy (cheapest), (B) add a distinct muted `"activity unavailable"` placeholder (recommended), (C) add `SetActivityRows` to trigger a distinguishable render path (requires designer-picked copy).

**Designer decision (Option B/C scope):** does S3 get a distinct error placeholder, or is "no recent activity" an acceptable fallback for both "no data" and "fetch failed"? Persona's signal: error placeholder prevents acting on stale absence data.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-behavior]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable]** Fresh startup with project context: S3 shows `"⟳ loading…"` immediately after `Init()` (before `ProjectActivityLoadedMsg` resolves); never flashes `"no recent activity"`. Test: build Init-state model; `stripANSI(m.View())` contains `"⟳ loading…"` AND does NOT contain `"no recent activity"` before any LoadedMsg fires.
2. **[Input-changed]** Context switch to org scope: S3 resolves to `"no recent activity"` (or designer-chosen "activity unavailable" copy), not `"⟳ loading…"`. Test: inject `ContextSwitchedMsg{Ctx: orgScope}`; assert View() does NOT contain `"⟳ loading…"` and DOES contain the designer-chosen resolved-state copy.
3. **[Input-changed]** `ProjectActivityErrorMsg` resolves the S3 loading state. Test: inject loading + error; assert View() does NOT contain `"⟳ loading…"` and contains designer-chosen error signal (either "no recent activity" fallback or distinct "activity unavailable" copy per designer decision).
4. **[Anti-behavior]** Successful fetch path still resolves to rendered rows. Test: Init → inject `ProjectActivityLoadedMsg{Rows: [...]}`; View() contains actor + summary.
5. **[Anti-regression]** FB-067 tests green (Init + ContextSwitchedMsg project-scope dispatch; ContextSwitchedMsg org-scope no-dispatch).
6. **[Anti-regression]** `m.activityDashboard.SetLoadErr` still fires for ActivityDashboardPane (not regressed). Existing FB-016 tests green.
7. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-067 ACCEPTED.

**Maps to:** FB-067 spec §6 (activity teaser lifecycle) + FB-067 user-persona P2 #1 + P2 #2 + P3 #3 + P3 #5 bundled.

**Non-goals:**
- Not changing the ActivityDashboard (pane `4`) behavior — its error state is already correct.
- Not adding automatic retry or backoff logic.
- Not generalizing to other loading-state transitions (resource types, buckets, etc.) — scope pinned to activity teaser.

---

### FB-083 — S3 "[4] full dashboard" hint renders even when activity data is absent

**Status: ACCEPTED 2026-04-20 + PERSONA-EVAL-COMPLETE 2026-04-20** — Option C rows-only suppression shipped per spec. `renderActivitySection()` gates hint on `len(m.activityRows) > 0`; `lipgloss.Width("")==0` absorbs the gap cleanly across empty/loading/fetchFailed states *when rows are empty*. Test-engineer gate-check added `TestFB083_AC6_AntiBehavior_SuppressedHint_4KeyStillNavigates` closing AC6 Anti-behavior (hint suppression is display-only; `[4]` still routes to ActivityDashboardPane with suppressed hint). 9 brief ACs mapped (AC1–AC4 Observable empty/loading/fetchFailed/populated, AC5 Input-changed empty vs populated View() diff, AC6 Anti-behavior `[4]` navigation preserved, AC7+AC8 Anti-regression FB-082 + FB-067 suites, AC9 Integration `go install ./...` + full suite). Spec at `docs/tui-ux-specs/fb-083-s3-hint-suppression.md`. User-persona eval complete 2026-04-20 — 6 positive findings + 1 P2 + 0 P3. **P2-1** (reachable contradiction: `activityFetchFailed=true` + stale `activityRows > 0` path after re-fetch error shows hint `"[4] full dashboard"` while body reads `"activity unavailable"` — gate predicate `len(m.activityRows) > 0` doesn't exclude the fetchFailed-with-stale-rows combination; hint's truthfulness invariant broken on the re-fetch-after-success-then-error path) → **FB-111 P2 engineer-direct filed** (one-line gate fix `&& !m.activityFetchFailed`; coordinates with FB-100 which addresses same contradiction from the state-layer). Per `feedback_scope_addition_fold_vs_new_brief`: persona explicitly framed as "two separate, independently fixable conditions" at different code sites (render vs state layer); distinct user-problem surfaces → new brief, not FB-083 reopen. **Net:** 1 new P2 brief (FB-111), 0 dismissals. FB-083 → PERSONA-EVAL-COMPLETE.
**Priority: P3** — dead journey: operator sees hint, presses `4`, lands on empty ActivityDashboard.

#### User problem

`renderActivitySection` (resourcetable.go:288–356) always renders the header `left + gap + hint` at line 296, where `hint = "[4] full dashboard"` (line 294). The hint is unconditional — it shows when `activityRows == nil`, when `len(activityRows) == 0` (post-fetch resolved empty), and when `activityLoading == true`.

An operator who sees `"no recent activity"` in S3 and presses `4` expecting richer content lands on an empty ActivityDashboard. The affordance implies more data exists behind the shortcut; when the project genuinely has no activity, pressing `4` is a dead journey. Likewise during `"⟳ loading…"` — the affordance implies "you could get there faster," which isn't true (pressing `4` doesn't fast-forward the fetch).

#### Proposed fix

Designer picks:

- **A.** Suppress `[4] full dashboard` hint entirely when `len(activityRows) == 0 && !activityLoading` (confirmed-empty state).
- **B.** Replace hint with softer copy in empty state: `"[4] dashboard (empty)"` or `"[4] history"`.
- **C.** Suppress during loading too (hint only when there are rows to navigate to).
- **D.** Keep hint always — operators who want to check for updates manually can; dead journey is acceptable cost.

Persona signal: A or C are preferred (affordance should reflect data availability).

Coordinates with FB-082 — if FB-082 adds a distinct "activity unavailable" error placeholder, this brief's hint suppression should cover that branch too.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable]** When S3 body shows `"no recent activity"` (resolved empty): S3 header does NOT contain `"[4] full dashboard"` (per designer choice; if A or C, substring absent; if B, substring replaced with alternate copy). Test: inject `activityRows=[]` + `activityLoading=false`; assert View() substring per designer decision.
2. **[Observable]** When S3 body shows rendered rows: S3 header DOES contain `"[4] full dashboard"` (or designer-chosen affordance copy). Test: inject `activityRows=[row1]`; assert View() substring.
3. **[Anti-regression]** `4` keybinding still works from welcome panel when used. Test: existing FB-016 `4`-key tests green.
4. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-067 ACCEPTED. Coordinates with FB-082 for error-state hint treatment.

**Maps to:** FB-067 user-persona P3 #4.

**Non-goals:**
- Not changing the `4` keybinding itself.
- Not changing the ActivityDashboard empty-state rendering.

---

### Triage record — FB-067 user-persona evaluation (2026-04-19)

| Persona finding | Disposition | Rationale |
|---|---|---|
| FB-067 P2 #1 (org-scope context switch freezes S3 in "⟳ loading…") | **FOLD into FB-082 P2** | Verified at `model.go:630` (unconditional SetActivityLoading) + `model.go:642-645` (conditional dispatch). Shares thesis with P2 #2 + P3 #3 + P3 #5 — activity state machine asymmetry. |
| FB-067 P2 #2 (Init doesn't set activityLoading=true → "no recent activity" flash) | **FOLD into FB-082 P2** | Verified at `model.go:186-188` (no SetActivityLoading call). Same root cause family. |
| FB-067 P3 #3 (ProjectActivityErrorMsg doesn't update m.table → permanent stuck state) | **FOLD into FB-082 P2** | Verified at `model.go:452-458` (only updates activityDashboard). Handler-side cleanup is mechanical; visual treatment of error state is designer-call. Per memory `feedback_scope_addition_fold_vs_new_brief.md`: same user-problem thesis (state machine asymmetry) → fold. |
| FB-067 P3 #4 ([4] full dashboard hint shown when no data → dead journey) | **NEW BRIEF FB-083 P3** | Verified at `resourcetable.go:294-296` (unconditional hint render). Distinct code site + distinct user problem (affordance discoverability, not state correctness). Designer-call. |
| FB-067 P3 #5 (loading-state asymmetry between startup vs context-switch entry paths) | **FOLD into FB-082 P2** | Direct consequence of P2 #1 + P2 #2. Fixing state machine normalization resolves the asymmetry automatically. No separate brief needed. |

Net new briefs: **2** (FB-082 P2, FB-083 P3). Folds into single brief: **4 findings → 1 brief (FB-082)**. FB-067 → PERSONA-EVAL-COMPLETE recorded. Note on severity: persona classified #1 + #2 as P2 and product-experience retains P2 for FB-082 because the org-scope regression directly contradicts FB-067's acceptance thesis ("no permanently-dead S3"). Cannot be P3 given it reproduces the original bug on a common navigation path.

---

### FB-084 — Placeholder action row teaches `[r]` for non-retryable errors and elides `describe` qualifier

**Status: ACCEPTED 2026-04-20 + PERSONA-EVAL-COMPLETE 2026-04-20** — 6 tests green (`TestFB084_AC1` through `TestFB084_AC5`), axis-coverage table complete (Observable / Input-changed / Anti-behavior / Anti-regression / Integration), `go install ./...` compiles, `go test ./internal/tui/... -count=1` green. Implementation matches ux-designer spec at `docs/tui-ux-specs/fb-084-placeholder-action-row-retryability.md` (Options A + D): `placeholderActionRow(errMode, retryable)` suppresses `[r]` when severity is non-retryable; call site in `buildDetailContent()` computes `retryable := errMode && components.ErrorSeverityOf(m.loadErr, m.rc) != data.ErrorSeverityError`; copy is `[r] retry describe` when shown; `case "r":` handler untouched (AC3 pin). User-persona eval delivered 0 P1 + 0 P2 + 4 P3 findings, all code-verified; **1 new brief filed, 3 dismissals** — see Recent Decisions 2026-04-20 FB-084 PERSONA-EVAL-COMPLETE entry.

**Process note (scope-creep flag, 2026-04-20):** engineer modified FB-084 test assertions (AC1/AC4/AC5) during FB-064 implementation — changed `m.View()` substring checks to `m.buildDetailContent()` substring checks to isolate the placeholder action row from unrelated status-bar `[r] refresh` noise. The modification aligned test code with the **description** of assertion targets already present in test-engineer's submitted axis-coverage table, so end-state is internally consistent and all 6 tests remain green. However, cross-feature test modifications bundled into another feature's implementation violate `feedback_scope_creep_bonus_bugs`: bundled fixes have no dedicated axis-coverage table entry and bypass the per-brief pre-submission gate. Accepted here because (a) end-state verified green and consistent with submitted table, (b) no working behavior was ripped out, (c) modifications were mechanical (View → buildDetailContent narrowing to same assertion text). Future rule reinforced with engineer: surface cross-feature test-infra changes as separate items (either a follow-up micro-brief or a pre-flight message requesting product-experience approval), never silent bundling.

Spec delivered 2026-04-19 by ux-designer; filed 2026-04-19 by product-experience from FB-051+FB-052 user-persona P2-1 + P3-2.
**Priority: P2** — action row teaches a key that's silently broken on the most common describe failure scenario (permission denied).

#### User problem

`placeholderActionRow(errMode=true, ...)` at `model.go:1917-1925` always includes `[r] retry` when in error mode, regardless of error severity. But the `case "r"` handler at `model.go:1209-1217` checks `ErrorSeverityOf(m.loadErr, m.rc)`: if severity is `ErrorSeverityError` (which includes permission-denied — the primary reason describe fails and this placeholder shows), pressing `r` posts `"No retry available for this error"` and takes no action.

An operator with restricted RBAC sees `(describe failed: permission denied)` in the placeholder body, then `[r] retry` in the action row, learns retry is an option, presses `r`, gets a hint, and learns the affordance lied. The action row taught a broken key for the failure scenario this placeholder was designed to surface.

Compounding: FB-051 §4 originally specified `[r] retry describe` (qualified). When folded into FB-052's action row, the qualifier was dropped to fit width — `[r] retry` alone. At narrow widths where the title-bar hint row drops, the action row is the operator's only context; `[r] retry` could mean retry events, retry the resource list, or retry describe. Operators on 40-col SSH terminals lose semantic precision.

#### Proposed fix

Designer picks (P2-1 — retryability gating):

- **A.** Suppress `[r]` from `placeholderActionRow` when `ErrorSeverityOf(m.loadErr) == ErrorSeverityError` — same logic the error block uses to decide retry visibility. Action row honors retryability.
- **B.** Render `[r]` always but visually de-emphasize (muted, no accentBold) when severity is non-retryable, with HelpOverlay disclosure of the reason.
- **C.** Replace `[r] retry` with `[?] details` for non-retryable errors — pivots affordance from action to information.

Designer picks (P3-2 — qualifier copy):

- **D.** Restore the qualifier: `[r] retry describe` (action row absorbs the FB-051 §4 phrasing).
- **E.** Keep `[r] retry` short — narrow width is constrained, and the placeholder body already says "Describe unavailable" which is the most recently-read line above the action row.

Persona signal: A + D preferred (suppress when broken; qualify when shown).

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-behavior]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable]** When placeholder is active AND `m.loadErr` severity == `ErrorSeverityError`: action row does NOT contain `[r]` substring (per A) OR contains de-emphasized `[r]` (per B) OR contains `[?]` instead (per C). Test: inject placeholder state with permission-denied error; assert View() substring per designer choice.
2. **[Observable]** When placeholder is active AND severity is retryable (e.g., `ErrorSeverityWarning` for transient/network errors): action row contains `[r]` substring (per A) and the qualifier copy from designer choice D or E. Test: inject transient error; assert View() substring.
3. **[Anti-behavior]** When `[r]` is suppressed (per A) and operator presses `r` while placeholder is active: existing case "r" handler still runs ("No retry available" hint posts). The suppression only changes the affordance; it does not lock the key. (Memory rule: actions stay reachable; affordances communicate validity.)
4. **[Input-changed]** Severity transition from `ErrorSeverityError` → `ErrorSeverityWarning` (e.g., user switches to a project with broader RBAC): action row updates to show `[r]` next render. Test: mutate `m.loadErr`, call View() before and after, assert substring presence changes.
5. **[Anti-regression]** Non-error placeholder mode (`errMode=false`) action row unchanged: `[E] events  [Esc] back` exactly. Test: existing FB-052 happy-path tests still green.
6. **[Anti-regression]** Error block (separate from placeholder) at `model.go:1947+` unchanged. The placeholder action row and the error block are different surfaces; this brief only touches the placeholder. Test: existing FB-005/FB-051 error-block tests green.
7. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-051 ACCEPTED + FB-052 ACCEPTED.

**Maps to:** FB-051+052 user-persona P2-1 + P3-2.

**Non-goals:**
- Not changing the `case "r"` handler (line 1209) or its severity check.
- Not changing FB-005 error block rendering.
- Not changing the placeholder body copy ("Describe unavailable — only events loaded.").

---

### FB-085 — Title bar shows `describe [unavailable]` and `⟳ loading…` simultaneously

**Status: ACCEPTED 2026-04-20** — Option A implementation verified in `model.go:detailModeLabel` (guard `if m.detail.Loading() { return "" }` inside placeholder branch; spinner authoritative during loading; `[unavailable]` restored post-load when describe still nil). 3 AC-indexed tests green: AC1 [Observable] `TestFB085_AC1_LoadingTrue_UnavailableLabelAbsent` (`stripANSI(View())` does NOT contain `[unavailable]` during loading+placeholder); AC2 [Observable] `TestFB085_AC2_LoadingFalse_UnavailableLabelPresent` (`[unavailable]` present when loading=false); AC3 [Input-changed] `TestFB085_AC3_LoadingTransition_FlipsLabel` (true→false transition flips label; both legs use `stripANSIModel(View())`). AC4 [Anti-regression] via full suite — FB-024 yaml/conditions mode-switch coverage, FB-051/FB-052 placeholder tests all green. AC9 [Integration] `go install ./...` clean + full `go test ./internal/tui/...` green. Test-engineer gate-check initial FAIL on AC5+AC6 (FB-086) Observable/Anti-behavior axes asserting model-state only; engineer rework added `stripANSIModel(appM.View())` + `strings.Contains(view, "Loading events")` substring assertions on both; re-gate PASS. Cross-feature anti-regression anchors green: FB-024 (E-guard, both-failed, E-after-error, yaml/conditions), FB-051/FB-052 (all). Product-experience independent verification: `go test ./internal/tui/... -run 'TestFB085|TestFB086' -count=1 -v` all 6 PASS. **Next:** engineer commits FB-085 separately from FB-086 to `feat/console` (one-commit-per-feature rule) with product-prose message describing the title-bar contradiction fix. **Prior Status: PENDING ENGINEER 2026-04-20** — spec delivered 2026-04-20 by ux-designer at `docs/tui-ux-specs/fb-085-086-detail-error-state-fixes.md` (shared with FB-086). FB-085 Option A: `detailModeLabel()` returns `""` when `m.detail.Loading() == true` AND placeholder condition is met — title bar shows only `⟳ loading…` during refresh, no contradiction. One-line change.


**Status: PENDING UX-DESIGNER (designer-call)** — filed 2026-04-19 by product-experience from FB-051+FB-052 user-persona P3-1.
**Priority: P3** — contradictory state visualization on a refresh path.

#### User problem

`detailModeLabel()` at `model.go:1908-1913` returns `"describe [unavailable]"` whenever `m.describeRaw == nil && m.events != nil && !m.yamlMode && !m.conditionsMode && !m.eventsMode`. `DetailViewModel.titleBar()` separately renders the loading spinner when `m.detail.loading == true`. On a refresh that re-fires the describe fetch while events from a prior load remain cached: both conditions hold simultaneously and the title bar reads:

```
HTTPProxy / api-gateway   describe [unavailable]   ⟳ loading…
```

`[unavailable]` reads as a settled final state. `⟳ loading…` reads as provisional. Together they contradict — the operator can't tell whether the describe fetch is still in progress or has definitively failed. This is a transient state but reproducibly visible on every refresh of the placeholder scenario.

#### Proposed fix

Designer picks:

- **A.** Suppress the `[unavailable]` suffix from the mode label when `m.detail.loading == true` (mode reads `describe` + spinner, no contradiction). Restore `[unavailable]` once loading completes and describe is still nil.
- **B.** Suppress the spinner when the placeholder is active (placeholder body itself signals "still trying" via the action row).
- **C.** Change `[unavailable]` to `[empty]` or `[no body]` — softer language that doesn't conflict with the spinner's provisional reading.

Persona signal: A preferred (the spinner is the most accurate signal during loading; suppress the contradiction by removing the settled label).

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable]** When placeholder is active AND `m.detail.loading == true`: View() does NOT contain `"[unavailable]"` substring (per A) OR does NOT contain `⟳` substring (per B) OR contains the chosen alternate label (per C). Test: inject placeholder + loading state; assert View() per designer choice.
2. **[Observable]** When placeholder is active AND `m.detail.loading == false`: View() contains `"[unavailable]"` substring (or chosen label). Test: inject placeholder, no loading; assert View() substring.
3. **[Input-changed]** Loading transition `loading=true → loading=false` flips the rendering per AC1/AC2. Test: mutate `m.detail.loading`, call View() before and after, assert substring presence changes.
4. **[Anti-regression]** Non-placeholder modes (yaml, conditions, events) unaffected — they don't go through `detailModeLabel()`'s `[unavailable]` branch. Test: existing FB-019/FB-024 mode-switch tests green.
5. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-051 ACCEPTED + FB-052 ACCEPTED.

**Maps to:** FB-051+052 user-persona P3-1.

**Non-goals:**
- Not changing the spinner mechanic itself.
- Not changing when the placeholder appears (`describeRaw == nil && events != nil`).

---

### FB-086 — `[E]` silently blocked when both describe AND events fail

**Status: ACCEPTED 2026-04-20** — Option A implementation verified at `model.go:1164` (case "E" admission gate relaxed with `|| m.eventsErr != nil`; existing re-dispatch at `model.go:1171` handles `LoadEventsCmd` re-fire). 3 AC-indexed tests green: AC5 [Observable] `TestFB086_AC5_EKey_DoubleFailure_AdmitsAndRedispatches` (double-failure + [E] → `stripANSIModel(View())` contains "Loading events" spinner + model state confirms `eventsMode=true` + `eventsLoading=true` + cmd dispatched); AC6 [Anti-behavior] `TestFB086_AC6_EKey_SingleFailure_DescribePresent_StillAdmits` (pre-existing `describeRaw != nil` admission path unbroken; View() shows "Loading events" spinner); AC7 [Input-changed] `TestFB086_AC7_EKey_DoubleFailure_ThenEventsLoaded_ViewTransition` (double-failure → [E] → `EventsLoadedMsg` → View() transitions to event content, describe error block absent). AC8 [Anti-regression] via full suite — FB-024 E-guard tests, FB-019 yaml/conditions coverage via `TestFB024_YamlAndConditions_Noop_DescribeNil`, FB-051/FB-052 placeholder tests all green. AC9 [Integration] `go install ./...` clean + full `go test ./internal/tui/...` green. Test-engineer gate-check initial FAIL caught AC5+AC6 Observable/Anti-behavior axes asserting model-state only (memory rule violation: "Observable ACs must assert View() output"); engineer rework added view-output substring assertions; re-gate PASS. Product-experience independent verification via `go test ./internal/tui/... -run 'TestFB085|TestFB086' -count=1 -v` all 6 PASS. **Affordance now honest:** `[E]` in double-failure either recovers via re-fetch or surfaces the events error card — no more silently-swallowed keypress. **Next:** engineer commits FB-086 separately from FB-085 to `feat/console` (one-commit-per-feature rule) with product-prose message describing the [E]-unblock. **Prior Status: PENDING ENGINEER 2026-04-20** — spec delivered 2026-04-20 by ux-designer at `docs/tui-ux-specs/fb-085-086-detail-error-state-fixes.md` (shared with FB-085). FB-086 Option A: relax `case "E"` admission gate from `describeRaw != nil || events != nil || eventsLoading` to add `|| eventsErr != nil`. In double-failure state, `[E]` now passes admission and re-fires `LoadEventsCmd` via existing re-dispatch at model.go:1113. One-line change.


**Status: PENDING UX-DESIGNER (designer-call)** — filed 2026-04-19 by product-experience from FB-051+FB-052 user-persona P3-3 (out-of-scope finding).
**Priority: P3** — affordance shown in title bar is silently broken in a real failure scenario; no path to recovery.

#### User problem

`case "E"` admission at `model.go:1104` requires `m.describeRaw != nil || m.events != nil || m.eventsLoading`. When both describe AND events have failed: `m.describeRaw == nil`, `m.events == nil` (never assigned on failure), `m.eventsLoading == false`. All three admission gates fail → `[E]` is silently swallowed by the handler with no feedback to the operator.

The placeholder condition `m.describeRaw == nil && m.events != nil` is also false (events == nil), so the FB-051 placeholder doesn't render either. The operator sees the FB-005 describe error block and a `[E] events` hint somewhere on the title bar (depending on width). Pressing `[E]` does nothing — no transition, no hint, no error block flip. The events error card exists but is unreachable.

This was outside FB-051/FB-052 scope (which specifically targets the `describe failed but events loaded` placeholder), but FB-051/052 increase the population of operators who encounter the error placeholder family and may try `[E]` thinking it always works.

#### Proposed fix

Designer picks:

- **A.** Relax admission: allow `[E]` when `m.eventsErr != nil` too — pressing `[E]` swaps to events mode and the existing FB-024 events error rendering takes over.
- **B.** Keep admission strict but render an inline events-error placeholder when both fail — equivalent to FB-051's structure but for the events side.
- **C.** Suppress the `[E] events` hint from the title bar when admission would be denied — affordance honesty (don't show what doesn't work). Combine with B for full coverage.

Persona signal: A is the minimum fix; C is the affordance-honesty companion.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Anti-behavior]`, `[Input-changed]`, `[Integration]`.

1. **[Observable]** When `describeRaw == nil && events == nil && eventsErr != nil && !eventsLoading`: pressing `E` transitions detail to events mode (per A) AND View() contains the events error message OR (per C) the title bar does NOT contain `[E]` substring. Test: inject double-failure state; assert View() per designer choice.
2. **[Anti-behavior]** When admission is denied (per C, no `[E]` shown) and operator still presses `E`: handler is a true no-op, no error popup, no spurious mode flip. Test: assert appM identical before/after key press.
3. **[Input-changed]** Recovery: events fetch retries successfully after E-press → View() shows events content; describe error block disappears. Test: dispatch successful EventsLoadedMsg post-retry; assert View() substring transition.
4. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** None blocking; coordinates with FB-024 (events mode) and FB-005 (error block) but doesn't depend on either changing.

**Maps to:** FB-051+052 user-persona P3-3 (out-of-scope finding).

**Non-goals:**
- Not changing the events fetch dispatch logic.
- Not changing FB-005 error block.
- Not changing the placeholder rendering.

---

### Triage record — FB-051 + FB-052 user-persona evaluation (2026-04-19)

| Persona finding | Disposition | Rationale |
|---|---|---|
| FB-051+052 P2-1 (`[r] retry` rendered in placeholderActionRow even when error is non-retryable; `case "r"` handler returns "No retry available" hint for `ErrorSeverityError`) | **NEW BRIEF FB-084 P2 (folds P3-2)** | Verified at `model.go:1917-1925` (unconditional `[r]` in errMode) + `model.go:1209-1217` (severity gate). Affordance teaches a broken key for the most common describe failure scenario. |
| FB-051+052 P3-1 (loading sub-variant title bar shows `describe [unavailable]` + `⟳ loading…` simultaneously — contradictory state) | **NEW BRIEF FB-085 P3** | Verified at `model.go:1908-1913` (detailModeLabel returns `[unavailable]` independently of loading). Distinct code site + distinct user problem (label-spinner contradiction). |
| FB-051+052 P3-2 (`[r] retry` lacks `describe` qualifier at narrow widths; operator can't tell what's being retried) | **FOLD into FB-084 P2** | Same code site as P2-1 (`placeholderActionRow`). Same thesis: action row precision in error states. AC8 in FB-084 covers qualifier copy choice (D vs E). |
| FB-051+052 P3-3 (when both describe AND events fail, `[E]` silently blocked at admission gate; events error card unreachable) | **NEW BRIEF FB-086 P3** | Verified at `model.go:1104` (admission requires describe||events||loading; all false in double-failure). Persona acknowledged out-of-scope; legitimate orthogonal bug exposed by FB-051/052 surface. |

Net new briefs: **3** (FB-084 P2, FB-085 P3, FB-086 P3). Folds: **2 findings → 1 brief (FB-084)**. FB-051 + FB-052 → PERSONA-EVAL-COMPLETE recorded. Severity note: P2-1 retains P2 because permission-denied is the primary describe failure mode for restricted-RBAC operators (the population most likely to see the placeholder). The lying affordance directly undermines the placeholder's purpose.

---

### FB-087 — Cross-dashboard chaining overwrites single-slot stash → stuck loop on dashboard pane

**Status: ACCEPTED 2026-04-20 + PERSONA-EVAL-COMPLETE 2026-04-20 (INCOMPLETE → FB-094 P1 filed)** — user-persona re-eval 2026-04-20 found 1 P1 (`3→4→3` inescapable loop) + 2 P2 + 2 P3. P1 verified in `model.go:1152–1204`: neither case "3" nor case "4" guards the "on the OTHER dashboard" branch, so a 3rd keypress overwrites both stash slots with dashboard panes and the subsequent Esc chain bounces forever. Routed to **FB-094 P1 hotfix** (designer-call: hybrid A+C dashboard-aware guard vs Option B back-stack). See triage record below. — implementation per ux-designer Option A: `dashboardOriginPane` split into `quotaOriginPane` + `activityOriginPane`; seven code sites migrated (Sites 1–6 mechanically renamed, Site 7 unchanged). 3 new tests mapping spec ACs 1/2/5 + 3 existing-test references mapping spec ACs 3/4/6: AC1 [Observable] `3→4→Esc→Esc` → NavPane + welcome signature (`TestFB087_AC1_Chain_3_4_EscEsc_ReturnsToNavPane`); AC2 [Observable] symmetric `4→3→Esc→Esc` (`TestFB087_AC2_Chain_4_3_EscEsc_ReturnsToNavPane`); AC3 [Anti-behavior] single `3→Esc` → NavPane (`TestFB048_Key3_Exit_ZeroOrigin_ReturnsToNavPane` — existing green); AC4 [Anti-behavior] single `4→Esc` → NavPane (`TestFB050_Key4_FirstPress_FromNavPane_StillEntersDashboard` + `TestFB048_Key4_RoundTrip_FromTablePane_RestoresTablePane` — existing green); AC5 [Input-changed] `3→4→Esc` intermediate `activePane==QuotaDashboardPane` — proves not a 2-press collapse (`TestFB087_AC5_Chain_3_4_Esc_IntermediateIsQuotaDash`); AC6 [Anti-regression] TablePane → `3` → Esc restores TablePane (`TestFB048_Key3_RoundTrip_FromTablePane_RestoresTablePane` — existing green); AC7 [Integration] `go test ./internal/tui/... -count=1` all green. Axis-coverage table submitted brief-AC-indexed; pre-submission gate passed on first submit. **Unblocks FB-088** (P3 engineer-ready now that two-slot stash provides the origin-label fields FB-088's title-bar affordance binds to).
**Original Status: PENDING ENGINEER** — spec delivered 2026-04-19 by ux-designer at `docs/tui-ux-specs/fb-087-cross-dashboard-stash-two-slot.md`. Designer chose **Option A — two-slot stash**. `dashboardOriginPane` splits into `quotaOriginPane` + `activityOriginPane`; each dashboard reads/writes only its own slot. Seven code sites: (1) case "3" stash + toggle-back → quotaOriginPane; (2) case "4" stash + toggle-back → activityOriginPane; (3) esc QuotaDashboardPane → read quotaOriginPane; (4) esc ActivityDashboardPane → read activityOriginPane; (5) ContextSwitchedMsg clear → zero both slots; (6) FB-078 updatePaneFocus() guard → rename `dashboardOriginPane.Pane` → `quotaOriginPane.Pane`; (7) DetailPane Esc detailReturnPane unaffected. Corrected traces: NavPane → `3` → `4` → `Esc` → `Esc` → Activity → Quota → NavPane ✓; reverse direction symmetric. Option C rejected because it would collapse both dashboards in one Esc, breaking the "Esc undoes the most recent navigation" invariant. Filed 2026-04-19 by product-experience from FB-048+FB-050 user-persona P2-1.
**Priority: P2** — operators who chain `3` → `4` (or `4` → `3`) end up stuck on a dashboard pane with no Esc-exit, only `3`/`4` toggle or other navigation keys.

#### User problem

`DashboardOrigin` (model.go:46-52) is a single-slot stash. Both `case "3"` (line 1152) and `case "4"` (line 1180) unconditionally overwrite `m.dashboardOriginPane` when entering their respective dashboards from any non-matching pane — including from the OTHER dashboard pane.

Verified trace:
1. NavPane → `3` (line 1142): stash = `{NavPane}`, transition to QuotaDashboardPane
2. QuotaDashboard → `4` (line 1170): line 1171 only checks `activePane == ActivityDashboardPane` (false here), so falls through to line 1180 which **overwrites** stash = `{QuotaDashboardPane}`, transitions to ActivityDashboardPane
3. ActivityDashboard → `Esc` (line 1422): `m.activePane = dashboardOriginPane.Pane = QuotaDashboardPane` ✓
4. QuotaDashboard → `Esc` (line 1478): `m.activePane = dashboardOriginPane.Pane = QuotaDashboardPane` — **same pane, stuck**

Steps 4–N: every Esc press is a no-op. The only exits are pressing `3` (which goes to NavPane via the stale stash on toggle-back) or another navigation key. The user is also stuck after the inverse path (`4` → `3` → `Esc` → `Esc` lands on QuotaDashboard repeatedly).

The "single slot is sufficient because dashboards don't nest" assumption was correct for `3 + Esc` and `4 + Esc` round-trips, but `3 + 4` (or `4 + 3`) IS a kind of nesting that the design didn't anticipate.

#### Proposed fix

Designer picks:

- **A. Two-slot stash.** Add `quotaOriginPane` and `activityOriginPane` as separate fields. Each dashboard's Esc handler reads its own slot. Cross-dashboard chaining preserves both origin pointers. Tradeoff: more state, but mental-model-clean (each dashboard remembers where it came from).
- **B. Stack stash.** Replace single-slot with a small stack `[]DashboardOrigin`. Push on entry, pop on Esc. Tradeoff: handles arbitrary nesting (future-proof), but more complex than the actual usage pattern requires.
- **C. Reject dashboard-as-origin.** When stashing in `case "3"` or `case "4"`, refuse to overwrite if the active pane is the OTHER dashboard pane — leave the existing stash intact. So `3 → 4 → Esc` returns to `NavPane` (the original origin), skipping QuotaDashboard. Tradeoff: simpler, but Esc semantics become "exit ALL dashboards" rather than "exit one".
- **D. Esc from dashboard pane always navigates back through the dashboard chain.** If origin is a dashboard pane (e.g., `{QuotaDashboardPane}`), Esc collapses both — go directly to NavPane. Tradeoff: combines C's "exit all" with explicit acknowledgement that nested dashboards collapse on first Esc.

Persona signal: A or C are mental-model-aligned for typical use; B is over-engineered; D is a single-keystroke escape hatch.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-behavior]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable]** After NavPane → `3` → `4` → `Esc` → `Esc`: `m.activePane` per designer choice (A: NavPane via QuotaDashboard intermediate; C: NavPane via single Esc; D: NavPane). Test: assert `appM.activePane == NavPane` AND `stripANSIModel(appM.View())` contains "Welcome" (NavPane signature).
2. **[Observable]** After NavPane → `4` → `3` → `Esc` → `Esc`: same as AC1 (symmetric).
3. **[Anti-behavior]** Single-dashboard round-trip unchanged: NavPane → `3` → `Esc` lands on NavPane (no regression on FB-048 happy path). Test: existing FB-048 tests green.
4. **[Anti-behavior]** Single-dashboard round-trip unchanged: NavPane → `4` → `Esc` lands on NavPane. Test: existing FB-067 + FB-076 tests green.
5. **[Input-changed]** Toggle-back via `3` from QuotaDashboardPane after a chain: per designer choice (A: NavPane; C: NavPane intermediate then NavPane on second toggle; D: NavPane). Test: assert appM.activePane.
6. **[Anti-regression]** TablePane → `3` → `Esc` lands on TablePane (FB-048 §Site 1 stash + restore). Test: existing FB-048 origin-restore tests green.
7. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-048 ACCEPTED + FB-050 ACCEPTED + FB-067 ACCEPTED + FB-076 ACCEPTED.

**Maps to:** FB-048+FB-050 user-persona P2-1.

**Non-goals:**
- Not changing the toggle semantics of `3`/`4` (second press exits to origin).
- Not changing dashboard rendering or pane internals.
- Not adding a "back stack" navigation primitive beyond what's needed for the chain case.

---

### FB-088 — No on-screen affordance communicates `3`/`4` toggle-back return destination

**Status: ACCEPTED + PERSONA-EVAL-COMPLETE 2026-04-20 (committed in 74fcd0a on feat/console — bootstrap bundle per team-lead exception)** — Persona delivered 5 positive findings + 1 P2 + 1 P3. P2 (status-bar ready-prompt persists after `[3]` confirm — FB-088+FB-097 interaction) filed as **FB-112**. P3 (empty-bucket viewport `[Esc] back to navigation` hardcoded vs FB-088 dynamic label) filed as **FB-113**. No rework to FB-088 itself; findings surface interaction gaps with FB-097 and a pre-FB-088 copy site that wasn't in the original brief scope. Option A shipped: `[3] back to <origin>` / `[4] back to <origin>` prepended to dashboard title-bar hint row. Code: `SetOriginLabel(string)` setter on both `QuotaDashboardModel` and `ActivityDashboardModel`; `dashboardOriginLabel(DashboardOrigin) string` helper in model.go resolves origin pane (`welcome panel`, `navigation`, `resource list`, `detail view`, `quota dashboard`, `activity dashboard`); label set at each FB-087 stash-write site **inside the FB-094 guard only**, cleared on Esc + context switch. Fresh-startup state renders empty hint (no misleading "back to" when no origin exists). Full-suite green 2026-04-20: engineer paste-evidence `go install ./...` clean + `go test ./internal/tui/... -count=1` passes after Path A retroactive FB-109 closeout (5 stale literal-copy assertions updated in same rework). Gate-check pass from test-engineer with formal axis-coverage table: 17 tests (AC1-AC9) all use `stripANSI(m.titleBar())` / `stripANSIActivity(m.View())` / `stripANSIModel(statusBarNorm(appM))` + `strings.Contains` substring assertions — zero model-field inspection on origin-label copy; confirmed `strings.Fields` whitespace-normalization via `statusBarNorm(appM)` helper (Width=120). Input-changed axis proven by rendering twice with different origins and asserting View() substrings diverge (TablePane-origin vs DetailPane-origin on same dashboard pane). Anti-regression anchors: existing dashboard-pane title-bar layout tests still green; FB-048 origin-pane logic unchanged. **Persona-eval queued** (user-persona agent to be spawned by team-lead). Filed 2026-04-19 by product-experience from FB-048+FB-050 user-persona P3-2. **Priority elevation history: P3 → P2** 2026-04-20 after FB-094 persona eval (2 P2 + 3 P3 findings all converge on this brief's thesis).

**Original Status: PENDING ENGINEER (impl-block LIFTED — FB-094 ACCEPTED 2026-04-20)** — 2026-04-20 **Priority elevated P3 → P2** after FB-094 persona eval (2 P2 + 3 P3 findings all converge on this brief's thesis: #1 "single-Esc collapse violates stack-based mental model" → FB-088 label explains destination; #2 "Esc destination is history-dependent not position-dependent" → FB-088 label surfaces the actual destination; #3 "no status-bar hint distinguishes cross-dashboard entry" (explicit validation); #4 "ready-prompt copy gives no Esc-destination signal" → FB-088 label fires when operator lands on dashboard post-confirm; persona note: "may be addressable by the FB-088 origin-label work rather than requiring new briefs"). **FB-094 architecture confirmed Option A** (dashboard-as-origin guard on two-slot stash — signatures unchanged, call sites move inside guard blocks); FB-088's `SetOriginLabel(string)` + `dashboardOriginLabel(DashboardOrigin) string` contract preserved as-specified. Ready to route to engineer after team rebuild closes FB-094 thematic batch. Spec delivered 2026-04-20 by ux-designer at `docs/tui-ux-specs/fb-088-dashboard-origin-affordance.md`. Option A: `[3] back to <origin>` prepended to QuotaDashboard title-bar hint row, `[4] back to <origin>` to ActivityDashboard. Origin label vocabulary (human-readable): `welcome panel`, `navigation`, `resource list`, `detail view`, `quota dashboard` (for activity chain), `activity dashboard` (for quota chain); empty string suppresses hint on fresh startup. Implementation contract: `SetOriginLabel(string)` setter on both dashboard components + `dashboardOriginLabel(DashboardOrigin) string` helper in model.go; label set at each FB-087 stash-write site **inside the FB-094 guard only** (line 1165 and line 1196 comment markers already pinned), cleared on Esc + context switch. 9 ACs: three origin states (AC1/2/3), input-changed (AC4), fresh-startup no-hint (AC5), cleared-on-Esc (AC6), FB-048 anti-regression (AC7), title-bar layout regression (AC8), integration (AC9). Filed 2026-04-19 by product-experience from FB-048+FB-050 user-persona P3-2. Priority-elevation signal added 2026-04-20 from FB-094 persona eval (2 P2 findings).
**Priority: P2** — elevated 2026-04-20. FB-094 persona eval delivered 2 P2 findings both convergent on this brief's thesis; the FB-094 single-Esc-collapse contract is correct but invisible — FB-088's origin label is the affordance that makes the contract legible. (Original P3 framing: discoverability cost on non-obvious return paths.)

#### User problem

Pressing `3` from an arbitrary pane opens QuotaDashboard. The title bar reads `quota usage` with keybind hints, but nothing on-screen communicates what pane the next `3` press (or Esc) will return to. On simple paths (NavPane → `3`), the destination is obvious. On non-obvious paths — DetailPane → `3`, or after a chain like FB-087 — the operator has no way to verify the return destination without pressing and hoping. The "exits to where you came from" mental model requires perfect recall of the pre-`3` pane.

Same applies to `4` and ActivityDashboardPane.

#### Proposed interaction

Designer picks:

- **A.** Status bar shows `[3] back to <origin>` (e.g., `[3] back to TablePane`) — explicit return label.
- **B.** Dashboard title bar appends ` (← <origin>)` suffix (e.g., `quota usage (← detail)`).
- **C.** Hint posted on dashboard entry: `"Esc returns to <origin>"` — one-time hint that fades.
- **D.** Accept silent design — operators who use `3`/`4` infrequently can fall back to Esc; those who use it often will memorize.

Persona signal: A preferred (always-visible, low cost). D is the no-op outcome.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable]** When activePane == QuotaDashboardPane: View() contains return-destination affordance per designer choice (A: `[3] back to <origin>` in status bar; B: `(← <origin>)` in title bar; C: hint text). Test: inject QuotaDashboardPane state with `dashboardOriginPane = {TablePane}`; assert View() substring per choice.
2. **[Input-changed]** Origin changes (TablePane → QuotaDashboard, vs DetailPane → QuotaDashboard) update the rendered affordance. Test: render twice with different origins; assert View() substrings differ per choice.
3. **[Anti-regression]** Dashboard pane rendering otherwise unchanged. Test: existing dashboard-pane tests green.
4. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-048 ACCEPTED + FB-050 ACCEPTED. Coordinates with FB-049 (status-bar hint filter) for placement choice A.

**Maps to:** FB-048+FB-050 user-persona P3-2.

**Non-goals:**
- Not changing the stash mechanism (FB-087 covers stash architecture).
- Not changing the toggle-back behavior itself.

---

### FB-094 — Cross-dashboard chaining still inescapable on 3-press chain (`3→4→3` or `4→3→4`)

**Status: ACCEPTED 2026-04-20 + PERSONA-EVAL-COMPLETE 2026-04-20** — persona delivered 0 P1 + 2 P2 + 3 P3; all 5 findings triaged; **0 new briefs filed**. P1 loop cleared, fix correct and complete. Findings #1 (single-Esc collapse vs stack-based mental model) + #2 (history-dependent Esc destination) + #3 (no status-bar hint — explicit FB-088 validation) + #4 (ready-prompt no Esc-destination signal) all fold as signal into **FB-088 priority elevation P3 → P2** (persona: "may be addressable by the FB-088 origin-label work rather than requiring new briefs"). Finding #5 (help overlay `[3]`/`[4]` descriptions stale) verified non-issue — help text reads `"[3] quota (toggle)"` at `components/helpoverlay.go:56,58`; "toggle" is accurate for the primary use case (single press from origin), and cross-dashboard collapse is edge behavior not explainable in 2 words. Persona anti-regression verification passed (4 chains traced). No regressions introduced.

**Status: ACCEPTED 2026-04-20** — Option A dashboard-as-origin guard implementation verified end-to-end. Code: `model.go:48` docstring replaced verbatim per spec AC7; `case "3":` stash write wrapped in `if m.activePane != ActivityDashboardPane` guard (§3a, line 1164); `case "4":` stash write wrapped in `if m.activePane != QuotaDashboardPane` guard (§3b, line 1195). Tests: 8 new FB-094 tests covering all 10 brief ACs: `TestFB094_AC1_3_4_3_EscEsc_ReachesNavPane` (Anti-behavior); `TestFB094_AC2_4_3_4_EscEsc_ReachesNavPane` (Anti-behavior); `TestFB094_AC3_3_4_3_QuotaOriginPreserved` (Observable — asserts `quotaOriginPane.Pane == NavPane` post-3rd-press); `TestFB094_AC4a_NavPaneStart_QuotaOriginIsNavPane` + `TestFB094_AC4b_TablePaneStart_QuotaOriginIsTablePane` (Input-changed — spec-faithful direct field assertion); `TestFB094_AC5_ExtraEscFromNavPane_NoDashboardReentry` (Repeat-press); `TestFB094_AC8_SinglePress3_StillWorks` + `TestFB094_AC8_SinglePress4_StillWorks` (Anti-regression — single-press non-cross-dashboard still works). Plus dedicated [Input-changed] pair at model_test.go:12100–12153: `TestFB094_InputChanged_3Key_NavPane_StashWritten` + `TestFB094_InputChanged_3Key_ActivityDash_StashPreserved` (same `3` key, activePane varies, different stash outcome — exactly the guard-fires-vs-doesn't pattern). Symmetric pair for `4` key. Anti-regression anchors: engineer-renamed `TestFB087_AC5_Chain_3_4_Esc_CollapsesToNavPane` (AC9 test flip green targeted), existing FB-087 AC1+AC2 (final-state preserved), FB-048 TablePane-origin via AC8_SinglePress3. `go install ./...` clean; `go test ./internal/tui/... -count=1` all packages green (15.6s). Axis-coverage pre-submission gate: initial table submitted with AC6/AC8/AC9 compressed into single row and AC10 renumbered as "AC8" — pushed back; test-engineer reshaped AC4 tests (renamed from `EscLandsOnNavPane` to `QuotaOriginIsNavPane` for direct-field-assertion spec fidelity) + added dedicated InputChanged pair + added AC8 single-press tests. Gate recovery verified by direct code read + targeted test-run rather than waiting for resubmit ping due to P1 priority and substantive completeness. **Unblocks FB-088** (impl-block on FB-094 lifted — FB-088 SetOriginLabel contract preserved since Option A guards call-site-only, signatures unchanged). **Unblocks FB-095** (pending-open cancel-path stale-origin, engineer-direct fix can now land).

**Original Status: IN-PROGRESS 2026-04-20** — engineer shipped Option A. Verified changes: `model.go:48` docstring updated verbatim per spec AC7; `case "3":` stash write wrapped in `if m.activePane != ActivityDashboardPane` guard (§3a); `case "4":` stash write wrapped in `if m.activePane != QuotaDashboardPane` guard (§3b). `TestFB087_AC5` renamed from `IntermediateIsQuotaDash` → `CollapsesToNavPane` with assertion flip to `NavPane` (AC9). `TestFB087_AC1_Chain_3_4_EscEsc_ReturnsToNavPane` + `TestFB087_AC2_Chain_4_3_EscEsc_ReturnsToNavPane` intermediate `t.Fatalf` guards updated to accommodate single-Esc-collapse intermediate behavior (final-state NavPane assertion unchanged). `go install ./...` clean; `go test ./internal/tui/...` all green.

**Original Status: PENDING ENGINEER** — P1 hotfix. Spec delivered 2026-04-20 by ux-designer at `docs/tui-ux-specs/fb-094-cross-dashboard-3-press-chain-fix.md`. Designer chose **Option A — dashboard-as-origin guard** layered on FB-087's two-slot stash; ~6 lines in `model.go` (two guard conditions in `case "3":` and `case "4":`). Option B (back-stack) rejected as too wide for P1 hotfix — FB-088's `SetOriginLabel` contract preserved because setter/helper signatures unchanged; only the call sites move inside the new guard blocks. 10 ACs — 5 new + 5 anti-regression (FB-087 AC1+AC2 final-state preserved, FB-087 AC5 test update pinned as AC9, FB-048 TablePane-origin preserved, docstring AC7, integration AC10). **Critical behavioral delta pinned in spec:** Option A's guard fires on the **2nd** cross-dashboard press, not just the 3rd — so `3→4→Esc` now lands on NavPane on the first Esc (previously: QuotaDashboardPane intermediate then NavPane). `TestFB087_AC5_Chain_3_4_Esc_IntermediateIsQuotaDash` assertion must flip from `QuotaDashboardPane` → `NavPane`. FB-087 AC1 + AC2 (final-state after full chain) unchanged. Verified grep: only three tests reference the `3_4` / `4_3` pattern (`TestFB087_AC1`, `TestFB087_AC2`, `TestFB087_AC5`) — AC1+AC2 unaffected, AC5 is the only test needing assertion flip. Replacement docstring for `model.go:48` pinned verbatim in spec AC7. Filed 2026-04-20 by product-experience from FB-087 user-persona re-eval P1.
**Priority: P1** — inescapable-loop regression: reproducing sequence is 3 natural keypresses (`3 → 4 → 3`) any operator exploring the dashboard shortcuts will hit within minutes. The only exits are context switch (zeroes both slots) or quitting the TUI. No affordance warns the keys have order-dependent, irreversible stash semantics.

#### User problem

FB-087 Option A fixed the `3→4→Esc→Esc` chain by giving each dashboard its own stash slot. But neither `case "3":` (model.go:1152) nor `case "4":` (model.go:1180) guards against being on the OTHER dashboard — both handlers only check `activePane == <own-dashboard-pane>` for the toggle-back branch, then fall through to the stash-overwrite branch for any other activePane, INCLUDING the other dashboard pane.

Verified trace (from user-persona P1, re-verified in code):
1. NavPane → `3` (line 1162): `quotaOriginPane = {NavPane}`, activePane = QuotaDashboardPane ✓
2. QuotaDashboard → `4` (line 1190): no guard for `activePane == QuotaDashboardPane`; `activityOriginPane = {QuotaDashboardPane}`, activePane = ActivityDashboardPane
3. ActivityDashboard → `3` (line 1162): no guard for `activePane == ActivityDashboardPane`; `quotaOriginPane` **overwritten** to `{ActivityDashboardPane}`, activePane = QuotaDashboardPane

State at rest:
- `quotaOriginPane = {ActivityDashboardPane}`
- `activityOriginPane = {QuotaDashboardPane}`

Every subsequent Esc press bounces between the two dashboards forever (line 1435 → QuotaDash; line 1489 → ActivityDash). Symmetric on `4→3→4`.

The comment at model.go:48 (`"Single-level: dashboards don't nest, so one slot is sufficient."`) contradicts live behavior — `3→4` IS a form of nesting the original FB-087 spec partially addressed and this brief completes.

#### Proposed fix

Designer picks:

- **A. Hybrid A+C (dashboard-as-origin guard layered on two-slot stash).** Keep FB-087's `quotaOriginPane` + `activityOriginPane`. Add an early-return / skip-stash branch in each handler when `activePane == <other-dashboard-pane>`: cross-dashboard presses still transition the activePane, but the stash slot is left intact from the entry-site press. So `3→4→3` returns Esc-chain: QuotaDash (same pane, same stash) → ActivityDash → NavPane. Tradeoff: preserves the mental model "Esc undoes most recent navigation" for within-dashboard origin, but cross-dashboard `3` from an active `4` session skips the cross-stash. Small diff, minimal test surface.
- **B. Back-stack stash (supersedes FB-087 Option A).** Replace `quotaOriginPane` / `activityOriginPane` with `[]DashboardOrigin` push/pop stack, pushed on dashboard entry, popped on Esc. Handles arbitrary-depth chaining (future-proof for REQ creep). Tradeoff: larger refactor, wider test surface, more state; but mental model is "Esc undoes one level" universally. FB-088 origin-label affordance adapts cleanly (peek top of stack).
- **C. Block same-key-to-other-dashboard (guard on handler entry).** Treat `3` when on ActivityDashboardPane (and `4` when on QuotaDashboardPane) as a no-op or as a toggle-to-origin-of-OTHER-dashboard. Tradeoff: silently ignores a keypress the operator issued with intent; violates "a keypress always does something visible" terminal convention.
- **D. Esc-from-dashboard-pane always collapses to NavPane.** Skip the stash dance entirely for dashboard panes: `esc` from either dashboard pane lands on NavPane, regardless of `quotaOriginPane` / `activityOriginPane` contents. Tradeoff: loses origin-restoration for TablePane → `3` → Esc → TablePane (FB-048 happy path). Would regress FB-048 AC.

Designer signal: **A or B.** A is the surgical hotfix; B is the durable architecture. C is user-hostile (eats a keypress); D regresses FB-048. Lean A for P1 speed unless designer judges B worth the scope.

#### Acceptance criteria

Axis tags: `[Anti-behavior]`, `[Observable]`, `[Input-changed]`, `[Repeat-press]`, `[Anti-regression]`, `[Integration]`.

1. **[Anti-behavior]** `NavPane → 3 → 4 → 3` lands on QuotaDashboardPane. Pressing `Esc` once from there: per designer choice (A: ActivityDashboardPane → then Esc → NavPane; B: ActivityDashboardPane → then Esc → NavPane; C: chain-blocked so state never reaches this point; D: NavPane on first Esc). Final Esc chain must reach NavPane in ≤2 Esc presses. Test: simulate key sequence, assert `appM.activePane == NavPane` after the correct number of Escs AND `stripANSIModel(appM.View())` contains Welcome signature.
2. **[Anti-behavior]** Symmetric: `NavPane → 4 → 3 → 4` lands on ActivityDashboardPane; Esc chain reaches NavPane in ≤2 Esc presses. Test: simulate + assert.
3. **[Observable]** On entering the 3rd-press dashboard (step 3 of trace), `stripANSIModel(appM.View())` must NOT render in a state where Esc would return to the just-exited dashboard pane. Concretely: at the moment `activePane == QuotaDashboardPane` after `3→4→3`, the *next* Esc must not land on `ActivityDashboardPane` (that's the P1 bug pattern). Choose the concrete assertion per designer choice: A/B — assert `quotaOriginPane.Pane != ActivityDashboardPane` (A) or `len(stack) ≤ 2 AND stack top != ActivityDashboardPane` (B); C — assert the 3rd press was blocked (`activePane` unchanged, hint posted); D — assert Esc returns to NavPane directly.
4. **[Input-changed]** `NavPane → 3 → 4 → 3` vs `TablePane → 3 → 4 → 3`: the Esc chain terminates on the right origin (NavPane vs TablePane respectively). Two subtests, same key sequence, different starting pane; assert final `activePane` differs per starting pane.
5. **[Repeat-press]** After the full `3→4→3→Esc→Esc` chain completes, pressing `Esc` a third time on NavPane must still be a no-op (or execute NavPane-Esc's existing "return to dashboard" branch per FB-055) — must not re-enter a dashboard. Test: simulate + assert no dashboard re-entry.
6. **[Anti-regression]** FB-087 ACs 1–7 remain green (2-press chain still works, single-dashboard round-trips still work, context switch still zeroes stash state). FB-048 TablePane → `3` → Esc → TablePane still green. Test: existing FB-087 + FB-048 tests run.
7. **[Anti-regression]** Docstring at model.go:48 updated to accurately describe the new stash invariant (no more "single-level, dashboards don't nest" claim). If designer picks A, comment describes the cross-dashboard guard. If B, comment describes the stack semantics. Test: grep/substring assertion on source file — OR reviewer checklist item if designer prefers not to assert on source text.
8. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-087 ACCEPTED.

**Maps to:** FB-087 user-persona P1 (2026-04-20 re-eval).

**Non-goals:**
- Not changing FB-088 origin-label contract (FB-088 blocks on FB-094 acceptance; if architecture shifts to B, FB-088 adapts as a follow-up).
- Not changing `pendingQuotaOpen` cancel-path stale-stash issue (covered by FB-095).
- Not changing dashboard rendering or pane internals.

**Priority override rationale:** P1 because (a) reproducing sequence is 3 natural keypresses an exploratory operator will hit; (b) severity matches FB-087's stated user problem — "stuck in loop on dashboard pane with no Esc exit" — that FB-087 set out to eliminate but only partially addressed; (c) FB-087 just ACCEPTED so the regression window is active; (d) FB-088 (P3) is impl-blocked downstream, so resolving FB-094 unblocks FB-088.

---

### FB-095 — `pendingQuotaOpen` cancel path leaves stale `quotaOriginPane`

**Status: ACCEPTED + PERSONA-EVAL-COMPLETE 2026-04-20** — engineer delivered two one-line stash-zeroing additions at Site A (`model.go` nav-cancel block, after `m.statusBar.Hint = ""`) + Site B (FB-080 second-press cancel branch, same treatment). 4 new FB-095 tests (AC1 nav-cancel Anti-behavior, AC2 second-press Anti-behavior, AC3 fresh-`3` post-cancel Observable with both-path subtests, AC4 Input-changed path-agnostic clearing invariant). Setup preconditions use `pendingQuotaOpen == true` not stash-non-zero (NavPane is iota=0 so `DashboardOrigin{Pane: NavPane} == DashboardOrigin{}`). **Cross-feature test update** `TestFB080_AC3_SecondPress_PreservesQuotaOriginPane` updated (necessary-consequence per `feedback_scope_creep_bonus_bugs`): FB-080 AC3 asserted pre-FB-095 stash-preservation invariant; FB-095 Site B explicitly changes that invariant (stash MUST clear on second-press cancel); updated assertion preserves the post-cancel semantic (stash is coherent with post-cancel state). Engineer flagged proactively in submission — reviewed and accepted as necessary-consequence. Axis-coverage table submitter-produced with Anti-behavior (AC1/AC2), Observable (AC3), Input-changed (AC4), Anti-regression (AC5 FB-078/FB-080/FB-087 16/16 PASS), Integration (AC6). **Gate-check retro:** test-engineer initially reported a compile-failure blocker citing undefined `QuotaDashboardModel.IsRefreshing`, `.Buckets`, `context.TUIContext` symbols in `TestFB107_AC1–AC5` — product-experience verified the claim by running `go test -c` (clean binary), `go test -run TestFB107` (PASS), and `go test ./internal/tui/... -count=1` (full suite green). The compile-failure was not reproducible; FB-095 tests + full suite PASS; AC6 integration criterion satisfied. Test-engineer error on the blocker claim — not on the source-inspection coverage analysis (which was accurate). Backlog correction folded into this acceptance record; FB-107 scaffold concern retracted. Spec: FB-095 in `docs/tui-backlog.md`. Minor nit: `TestFB080_AC3_SecondPress_PreservesQuotaOriginPane` name is now slightly misleading post-migration (asserts stash-CLEARED, not preserved) — not a correctness issue (body + comment reflect FB-095 invariant clearly); cosmetic rename optional. **Persona eval 2026-04-20:** 0 P1/P2/P3 findings. 5 positive findings confirm: both sites correctly zero the stash; re-queue write guard (`!m.pendingQuotaOpen` at model.go:1176) makes handoff self-healing (no window for observing stale zeroed origin); toggle-back path cannot observe zeroed stash because next `[3]` overwrites before QuotaDashboardPane entry; cancel-as-symmetry reads naturally (first `[3]` stashes, second `[3]` clears — logical inverse); nav-cancel gate precision on `m.activePane != m.quotaOriginPane.Pane` prevents spurious zeroing on in-origin-pane transitions. FB-096 coordination noted (stash zero + acknowledgment hint post are independent writes; no ordering concern). Cancel-path stash invariant CLOSED.

**Original Status: PENDING ENGINEER** — P3, engineer-direct. Filed 2026-04-20 by product-experience from FB-087 user-persona re-eval P3. Small follow-up to FB-078's cancel-path scope.

**Priority: P3** — stash-invariant violation during loading window; low harm (only observable on a specific re-entry path).

#### User problem

`case "3":` writes `quotaOriginPane` (model.go:1162–1165) BEFORE checking `m.quota.IsLoading()` (line 1166). When loading is in flight, the write happens, then `pendingQuotaOpen = true` is set (line 1174).

FB-078's cancel path at `updatePaneFocus()` (model.go:850–852) clears `pendingQuotaOpen` + `statusBar.Hint` when the operator navigates away from the origin pane, but does NOT clear `quotaOriginPane`. Result: `quotaOriginPane` holds the stale origin from the cancelled session. If the operator subsequently enters QuotaDashboardPane via a different path (e.g., presses `3` from a different pane after buckets finish loading), the stash is immediately rewritten by the next `3` press's `case "3":` branch, so harm is limited. But the invariant "`quotaOriginPane` is coherent with whether a QuotaDashboard session is active or queued" is violated during the window between cancel and next entry.

#### Proposed fix

Extend **both** cancel paths to also clear `quotaOriginPane` — scope expanded 2026-04-20 from FB-080 persona P3-2:

**Site A** — FB-078's nav-cancel block at `model.go:850–852`:

```go
if m.pendingQuotaOpen && m.activePane != m.quotaOriginPane.Pane {
    m.pendingQuotaOpen = false
    m.statusBar.Hint = ""
    m.quotaOriginPane = DashboardOrigin{} // FB-095: clear stale origin on pending-open cancel
}
```

**Site B (scope-added 2026-04-20 from FB-080 persona P3-2)** — FB-080's second-press cancel branch at `model.go:1192–1196`:

```go
} else {
    // FB-080: second press cancels the queued open.
    m.pendingQuotaOpen = false
    m.statusBar.Hint = ""
    m.quotaOriginPane = DashboardOrigin{} // FB-095: clear stale origin on second-press cancel (P3-2 fold)
}
```

Engineer-direct fix — pattern matches the context-switch clear at model.go:570–571. Both sites must land in the same PR to preserve the invariant across all cancel paths.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Anti-behavior]`, `[Anti-regression]`, `[Input-changed]`, `[Integration]`.

1. **[Anti-behavior]** (nav-cancel path) After NavPane → `3` (during loading, queues pending-open) → navigate-to-DetailPane (cancels pending-open per FB-078 B), `m.quotaOriginPane` == `DashboardOrigin{}` (zero value). Test: inject key sequence, assert field equals zero value.
2. **[Anti-behavior]** (second-press cancel path, scope-added from FB-080 persona P3-2) After NavPane → `3` (queues pending-open) → `3` (second press cancels per FB-080), `m.quotaOriginPane` == `DashboardOrigin{}`. Test: inject two-key sequence, assert stash cleared.
3. **[Observable]** Subsequent fresh `3` press from TablePane (after either cancel path) writes `quotaOriginPane = {TablePane}` cleanly. Test: assert after the post-cancel `3` press in both scenarios.
4. **[Input-changed]** Same `3` key from NavPane with `pendingQuotaOpen=true`: both cancel path branches (nav-cancel via subsequent navigation, second-press cancel via immediate re-press) clear `quotaOriginPane`; this AC pins that the stash-clear invariant is path-agnostic.
5. **[Anti-regression]** FB-078 ACs still green (cancel clears pendingQuotaOpen + hint). FB-080 ACs still green (second-press cancel clears pendingQuotaOpen + hint). FB-087 context-switch clear at model.go:570–571 still green.
6. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-078 ACCEPTED, FB-080 ACCEPTED.

**Maps to:** FB-087 user-persona re-eval P3 (pendingQuotaOpen stash-before-confirm); **FB-080 user-persona P3-2** (second-press cancel stash-clear gap, scope-added 2026-04-20).

**Non-goals:**
- Not changing when `quotaOriginPane` is written during the non-loading fast path (line 1162 fires correctly outside the loading window).
- Not changing `activityOriginPane` — no `pendingActivityOpen` mechanism exists; dashboard key `4` has no loading path.

---

### FB-096 — Esc does not cancel pendingQuotaOpen from NavPane; cancel-on-nav silently discards the queue

**Status: ACCEPTED + PERSONA-EVAL-COMPLETE 2026-04-20** — user-persona eval clean: 0 P1, 0 P2, 2 P3 (both dismissed). Site verification table confirmed all 3 cancel sites post `"Quota dashboard cancelled"` with correct transience policy: Site 1 (Esc, `model.go:1532`) 3s transient + `DashboardOrigin{}` cleared; Site 2 (nav, `model.go:869`) persistent (no Cmd) + `DashboardOrigin{}` cleared; Site 3 (second-press, `model.go:1205`) 3s transient + `DashboardOrigin{}` cleared. Hint copy identical across sites. No collision with FB-079 `"Quota dashboard loading…"` or FB-097 `"Quota dashboard ready — press [3]"`. P3#1: `statusBar.Hint = ""` direct-clear at Sites 1/3 is redundant with `postHint()` — cosmetic dead code only, no user-visible problem; dismissed. P3#2: after cancel, FB-097 ready-prompt guard (`if m.pendingQuotaOpen` at `model.go:401`) intentionally suppresses ready-prompt — reflexive-cancel operators lose ready signal but design-intentional; dismiss pending operator-report evidence. Adjacent verification: FB-097 ready-prompt collision absent (AC8 confirms cancel overwrite); FB-095 stash invariant closed (Site 2 clears `quotaOriginPane` at line 870); 11 tests cover all axes via `statusBarNorm(appM)` View() discipline. Cancel-path trilogy (FB-095/FB-096/FB-099) fully closed with persona sign-off. **Prior gate-check (test-engineer re-gate) 2026-04-20:** clean with paste-able evidence. `go test -c -o /dev/null ./internal/tui/...` → exit 0. `go test -count=1 -run TestFB096 -v ./internal/tui/...` → all 11 PASS (AC1 Esc/NavPane hint; AC2 NavCancel_HintShown renamed from `_HintField`; AC3 2-subtest Input-changed; AC4 Esc-no-pending anti-behavior; AC5 FB-055 anti-regression; AC6 pending overrides dashboard restore; AC7 FB-079 nav-cancel clears pending; AC8 queue→Esc→requeue integration; AC9 second-press hint contains "cancelled"; AC10 first-vs-second-press View() diff; AC11 third-press requeues, no double-cancel). Cross-feature anti-regression: FB-055, FB-079, FB-080, FB-099 suites all green. Full suite + `go install ./...` clean. All 10 converted Observable/Input-changed/Anti-behavior/Anti-regression/Integration ACs now use `statusBarNorm(appM)` (Width=120 + View() + strings.Fields) matching FB-055 AC1 precedent; AC7 retained as model-state assertion (boolean `pendingQuotaOpen`), correct surface choice.

**Prior rework round:** engineer rework complete. Added `statusBarNorm(m AppModel) string` helper (sets `Width=120`, renders `statusBar.View()`, normalises whitespace via `strings.Fields`) — used by all 10 affected tests. AC2 test renamed `TestFB096_AC2_NavCancel_HintField` → `TestFB096_AC2_NavCancel_HintShown` (honesty: it now asserts rendered output, not field state). All `statusBar.Hint` field inspections replaced with `statusBarNorm(appM)` + `strings.Contains(norm, ...)`. AC7 (`pendingQuotaOpen` boolean) retained as model-state assertion (not a rendered signal). 11 tests pass locally; full suite green; `go install ./...` clean. Ready for re-gate: verify each of the 10 converted tests renders the hint at Width=120 and asserts substring match on View() output, not on the `.Hint` field. **Prior rework-cause:** test-engineer gate-check PUSH-BACK with evidence: all 10 Observable/Input-changed/Anti-behavior/Anti-regression/Integration ACs use `stripANSIModel(appM.statusBar.Hint)` field inspection rather than `stripANSIModel(appM.statusBar.View())` + `statusBar.Width = 120` — systematic `feedback_observable_acs_assert_view_output` violation. Precedent cited: FB-055 AC1 (model_test.go:11732–11748) sets `statusBar.Width = 120` + asserts `stripANSIModel(appM.statusBar.View())` with `strings.Fields` normalization; FB-011 (line 8280); FB-085 (line 11119) with explicit comment codifying the rule. `statusBar.Hint` is a raw string field — if `statusBar.Width` is 0 (default, unset in FB-096 tests), `statusBar.View()` may truncate/omit the hint while the field assertion silently passes. AC7 (`pendingQuotaOpen` boolean) is model-state as intended, no change needed. Implementation itself verified correct (compile exit 0, all 11 tests pass on execution, FB-055/078/079/080 anti-regression suites green) — rework is test-assertion only, not impl. Engineer must add `appM.statusBar.Width = 120` and swap `statusBar.Hint` → `statusBar.View()` in the 10 affected tests, matching FB-055 pattern. **Original Status: PENDING ENGINEER** — scope amendment delivered 2026-04-20 by ux-designer. ~~**Engineer submission (rework target):**~~ Engineer delivered 3-site acknowledgment implementation. **Site 1** (NavPane Esc handler ~line 1519): new `pendingQuotaOpen` check inserted before FB-055 gate — clears pending + stash + `m.postHint("Quota dashboard cancelled")` transient 3s (symmetric with Site 3 per transience principle). **Site 2** (`updatePaneFocus()` nav-cancel ~line 862): replaced `m.statusBar.Hint = ""` with `m.statusBar.PostHint("Quota dashboard cancelled")` — persistent (no Cmd available). **Site 3** (second-press cancel else branch ~line 1193): replaced `m.statusBar.Hint = ""` + bare `return m, nil` with `return m, m.postHint("Quota dashboard cancelled")` — transient 3s (symmetric with Site 1). 11 tests PASS: AC1 Observable NavPane Esc hint shown; AC2 Observable nav-cancel hint field; AC3 Input-changed 2 subtests loading → cancelled transition; AC4 Anti-behavior Esc-no-pending does not post spurious cancel hint; AC5 Anti-regression FB-055 Esc-back-to-dashboard green; AC6 Anti-regression pending overrides dashboard-restore; AC7 Anti-regression FB-079 nav-cancel clears pending; AC8 Integration queue-Esc-requeue sequence; AC9 Observable second-press hint contains "cancelled"; AC10 Input-changed first-press vs second-press different hint; AC11 Anti-behavior third press requeues (does not double-cancel). `go install ./...` clean + full suite green. Submitter-produced axis-coverage table complete. **Original Status: PENDING ENGINEER** — scope amendment delivered 2026-04-20 by ux-designer. Spec at `docs/tui-ux-specs/fb-096-esc-cancel-acknowledgment.md` now covers all 3 cancel paths. Site 3 added at `model.go:1192–1196` (second-press cancel branch) with transient 3s via `m.postHint("Quota dashboard cancelled")` — symmetric with Site 1 (Esc) since both are explicit operator cancels and both handlers can return Cmd. Transience principle codified: explicit cancels (Esc, second-press) = 3s transient; implicit cancel (nav) = persistent (`updatePaneFocus` cannot return Cmd). 3 new ACs: AC9 Observable (stripANSI(appM.View()) contains "Quota dashboard cancelled" after second press), AC10 Input-changed (first-press loading hint vs second-press cancel hint — same key, different View()), AC11 Anti-behavior (third press re-queues, does not double-cancel). Now 11 ACs total.

**Prior spec notes (Option B, 2 sites):** **Option B selected** (both Esc-cancel and nav-cancel paths post acknowledgment hint — nav-cancel is actually more confusing than Esc case since operator didn't intend to cancel). Hint copy: `"Quota dashboard cancelled"` (no collision with FB-079 `"Quota dashboard loading…"` or FB-078 ready-prompt `"Quota dashboard ready — press [3]"`). Implementation — 2 sites: (1) NavPane Esc handler with new `pendingQuotaOpen` check fires before FB-055 gate (regardless of `showDashboard` state), posts transient 3s hint via `m.postHint()`; (2) `updatePaneFocus()` nav-cancel at `model.go:851–854`: replace `m.statusBar.Hint = ""` with `m.statusBar.PostHint("Quota dashboard cancelled")` — persistent (no `HintClearCmd` — `updatePaneFocus` can't return Cmd); clears on next hint or context switch. 8 ACs: Observable (AC1/AC2), Input-changed (AC3), Anti-behavior (AC4 no spurious cancel, AC5 FB-055 still fires own gate), Anti-regression (AC6 FB-055 green, AC7 FB-079 green), Integration (AC8). Filed 2026-04-20 by product-experience from FB-078 user-persona P2 #1 + P3 #3 (fold).

**Scope amendment ask (ux-designer):**
- Add **Site 3** at `model.go:1192–1196` (FB-080 second-press cancel branch): replace `m.statusBar.Hint = ""` with `m.statusBar.PostHint("Quota dashboard cancelled")` (transient 3s if possible — unlike `updatePaneFocus()`, the key handler CAN return `Cmd`, so `m.postHint()` with `HintClearCmd` is viable here).
- Add new AC (**AC11** or insert AC3b) for Site 3 coverage — Observable + Input-changed pair: same `3` key with `pendingQuotaOpen=true` posts acknowledgment hint; `stripANSIModel(appM.View())` substring assertion.
- Decide hint-transience policy for Site 3: symmetric with Site 1 (3s transient via `postHint`) or symmetric with Site 2 (persistent until next-hint). Persona's P2-1 doesn't prescribe — pick based on consistency thesis.
- Update spec file `docs/tui-ux-specs/fb-096-esc-cancel-acknowledgment.md` with Site 3 + new AC.

**Original Status: PENDING UX-DESIGNER** — P2, designer-call. Filed 2026-04-20 by product-experience from FB-078 user-persona P2 #1 + P3 #3 (fold).

**Priority: P2** — terminal-conventional cancel gesture (Esc) is dead during quota loading; natural user mental model ("press Esc to abort") hits a silent no-op. Paired P3 #3 (silent nav-cancel) shares the same "cancel feedback parity" thesis.

#### User problem

During quota loading, `[3]` sets `pendingQuotaOpen = true` and posts a persistent loading hint. FB-078's cancel path fires only inside `updatePaneFocus()` (`model.go:850–852`) — i.e., only when the operator navigates to a different pane. Pressing Esc from NavPane with a queued pending-open is a pure no-op: the NavPane Esc handler at `model.go:1495–1501` requires `!m.showDashboard && m.tableTypeName != ""` and returns `m, nil` for the empty-welcome-panel case, never consulting `pendingQuotaOpen`. Operators reaching for Esc observe: loading hint persists, no feedback, no state change — indistinguishable from "terminal not forwarding input." Even when cancel DOES fire via navigation, the nav-cancel path silently clears the hint with no replacement — operators who navigate away expecting to return to the auto-opened dashboard cannot tell whether the queue was discarded or preserved.

#### Proposed options

- **A. Esc cancels only.** NavPane Esc with `pendingQuotaOpen=true` calls the same cancel path (`pendingQuotaOpen = false`, clear hint, clear origin per FB-095) + posts a transient acknowledgment hint (e.g., `"Quota open cancelled"`, 2s decay). Covers persona #1 directly. Nav-cancel path unchanged — persona #3 remains silent (re-files as P3 if FB-097 doesn't subsume it).
- **B. Esc + nav cancel both acknowledged.** A, plus `updatePaneFocus()` cancel branch also posts the same acknowledgment hint. Unified feedback for both cancel gestures. Addresses persona #1 + #3.
- **C. Discoverability hint only.** Status-bar keybind strip adds `[Esc] cancel` during pendingQuotaOpen but Esc still no-ops. Rejected — persona #1 premise (operator reaches for a gesture that doesn't work) is not solved by hinting at an inert key.

Designer-preference signal: B is the cleanest match to the "cancel feedback parity" thesis. A is minimum viable but leaves half the finding unaddressed. Copy for acknowledgment hint must not collide with FB-079 ready-prompt text.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Anti-behavior]`, `[Anti-regression]`, `[Input-changed]`, `[Repeat-press]`, `[Integration]`.

1. **[Observable]** From NavPane with `pendingQuotaOpen=true`, pressing Esc clears `pendingQuotaOpen`, `quotaOriginPane` (per FB-095 pattern), and `statusBar.Hint` of loading copy; posts chosen acknowledgment hint.
2. **[Observable]** `stripANSIModel(appM.View())` after Esc contains the acknowledgment hint copy; loading-hint copy is absent.
3. **[Observable (Option B only)]** Navigation-triggered cancel (Tab to TablePane during pendingQuotaOpen) posts the same acknowledgment hint. `View()` substring assertion.
4. **[Anti-behavior]** Esc from NavPane WITHOUT `pendingQuotaOpen` behaves exactly as today (pure no-op or FB-055 "Returned to welcome panel" depending on `m.showDashboard`/`m.tableTypeName` state). No change to that gate.
5. **[Anti-regression]** FB-078 AC1–AC5 stay green (ready-prompt auto-transition from QuotaDashboard on `BucketsLoadedMsg` with cancel-on-nav still works).
6. **[Anti-regression]** FB-055 NavPane→welcome Esc path (non-pending case) still fires `"Returned to welcome panel"` hint when appropriate. Named test: `TestFB055_*`.
7. **[Anti-regression]** FB-095 stash-clear on cancel still fires (Esc cancel MUST clear `quotaOriginPane` too; not just the hint+pending flag).
8. **[Input-changed]** Same Esc key — with `pendingQuotaOpen=true` fires cancel + acknowledgment; with `pendingQuotaOpen=false` hits existing FB-055 / no-op path. `View()` substring differs between runs.
9. **[Repeat-press]** After cancel fires, second Esc from NavPane hits the existing FB-055 / no-op path (queue already cleared).
10. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Non-goals:**
- Not changing FB-078's cancel-on-nav behavior (retained; Option B additionally surfaces a hint).
- Not adding `[4]`-path cancel — no `pendingActivityOpen` mechanism exists.
- Not touching `IsLoading()` state (quota data keeps loading in background; only the pending-open queue is cancelled).

**Dependencies:** FB-078 ACCEPTED. Should land after FB-095 (stash clear on cancel) so Esc-cancel inherits it; if FB-095 hasn't shipped, FB-096 implementation MUST include the stash clear.

**Maps to:** FB-078 user-persona P2 #1 (primary); P3 #3 (folded under Option B).

---

### FB-097 — Quota ready-prompt auto-clears in 3s; no persistent signal that data is ready to open

**Status: ACCEPTED + PERSONA-EVAL-COMPLETE 2026-04-20 (committed in 74fcd0a on feat/console — bootstrap bundle per team-lead exception)** — Persona delivered 5 positive findings + 1 P2 ("ready-prompt persists past `[3]` confirm") filed as **FB-112**. FB-097 spec §6's "natural UI flow replaces" assumption proven incorrect in code; confirm path at `model.go:1220–1224` never clears hint. FB-112 owns the rework. No retroactive FB-097 changes needed — persistence contract pre-confirm is working as designed. Test-engineer gate-check clean. All 9 brief ACs PASS via brief-AC-indexed table. Note: test-engineer filled an AC-indexing gap from engineer's submission — engineer's 8 tests misindexed brief AC4 (stale-token anti-behavior vs brief's `[3]-confirm-clears-ready-prompt`); test-engineer authored `TestFB097_BriefAC4_ConfirmClearsReadyPrompt` (asserts `activePane == QuotaDashboardPane`, `"Quota dashboard ready"` absent from `statusBarNorm()`, `"[3] back"` present — the natural-UI-flow clear per spec where QuotaDashboardPane hints crowd out the prompt). Engineer's original `TestFB097_AC4_AntiBehavior_StaleHintClearMsg_DoesNotClear` kept as defense-in-depth (verifies PostHint token-bump invariant). Final coverage: AC1 Observable ready-prompt in `statusBarNorm` post-`BucketsLoadedMsg`; AC2 Observable persistent (Cmd=nil confirms no `HintClearCmd`); AC3 Anti-behavior loading hint no auto-clear (FB-078 preserved); **AC4 Anti-behavior [3]-confirm clears ready-prompt** (test-engineer added); AC5 Anti-behavior cancel (Esc + nav) clears ready-prompt; AC6 Anti-regression FB-078 AC1–AC5 green; AC7 Anti-regression 3s decay for non-FB-078 callers unchanged; AC8 Input-changed `pendingQuotaOpen=true` vs false on `BucketsLoadedMsg` → different View(); AC9 Integration `go install ./...` + `go test ./internal/tui/...` green. All Observable ACs assert `statusBarNorm(appM)` View() output per `feedback_observable_acs_assert_view_output`. Option A one-site change at `model.go:401-402` (`m.statusBar.PostHint(...)` + `return m, nil` with `// FB-097: persistent (no HintClearCmd)` comment). Persona eval queued.

**Prior Status: PENDING ENGINEER** — spec delivered 2026-04-20 by ux-designer at `docs/tui-ux-specs/fb-097-ready-prompt-persistence.md`. **Option A selected** (persistent ready-prompt). One-character change: `m.postHint(...)` → `m.statusBar.PostHint(...)` at `model.go:401–402` plus `return m, nil`. Removes the 3s `HintClearCmd`. Mirrors loading-hint pattern (FB-079) exactly. Option C (glyph badge) rejected — requires new right-aligned status-bar primitive, scope disproportionate. Option D (FB-089 Tab-hint suffix) rejected — couples two unrelated PENDING ENGINEER features. Clear sites validated: context switch (`model.go:569`) ✓, FB-096 Esc/nav-cancel `"Quota dashboard cancelled"` ✓, `[3]` confirm replaces via natural UI flow ✓. 9 ACs: Observable (AC1 hint set + AC2 nil Cmd), Input-changed (AC3 loading→ready transition), Anti-behavior (AC4 survives HintClearMsg + AC5 context-switch clears), Anti-regression (AC6 copy intact, AC7 FB-079 green, AC8 FB-096 cancel chain green), Integration (AC9). Filed 2026-04-20 by product-experience from FB-078 user-persona P2 #2.

**Original Status: PENDING UX-DESIGNER** — P2, designer-call. Filed 2026-04-20 by product-experience from FB-078 user-persona P2 #2.

**Priority: P2** — narrow discoverability window (3s) after quota loads means operators who glance away at the wrong moment lose the only signal that data is available. Operator must press `[3]` blindly to verify; if loading hasn't completed the press queues again. The loading-vs-ready asymmetry is the core confusion.

#### User problem

FB-078 ready-prompt hint `"Quota dashboard ready — press [3]"` fires via `postHint` (`model.go:400`) which schedules `HintClearCmd` after 3 seconds. This contradicts the loading-hint model: the loading hint at `model.go:1173` is intentionally persistent (comment: `"No HintClearCmd — hint clears on load completion or error"`) because the user must wait. The ready-prompt has inverted contract — it clears automatically even though the user's confirm gesture has not arrived. After expiry: no status-bar signal, no title-bar indicator, no way to distinguish "quota loaded and ready" from "quota not yet started." Pressing `[3]` opens immediately (since `IsLoading()==false`), which is correct behavior, but the operator has no confidence that the press will do what they expect.

#### Proposed options

- **A. Persistent ready-prompt.** Write hint directly (bypass `postHint`); clears only on `[3]` confirm, cancel (per FB-096), or context switch. Mirrors loading-hint model exactly. Risk: permanent visual clutter if operator ignores the prompt entirely — but ready-prompt has a natural end (operator acts or cancels), so "ignored ready-prompt" is itself a user-problem signal.
- **B. Extended decay.** Change ready-prompt branch to 10s or 15s `HintClearCmd`. Halfway fix; narrow window still exists. Rejected as a tuning-knob that doesn't close the gap.
- **C. Persistent glyph.** Right-aligned status-bar badge (e.g., `⚡ quota ready [3]`) independent of the hint slot. Separates "nag once" from "availability indicator." Larger visual change; coordinates with welcome-panel S3 CTA rendering. Needs status-bar contract work.
- **D. Persistent ready-prompt + FB-089-style `(ready)` suffix on Tab hint.** Two-surface signal. Coordinates with FB-089 `(cached)` pattern already in flight.

Designer-preference signal: A is the simplest behavioral fix and symmetric with loading-hint persistence. D bundles coordination with FB-089 if that feels natural; no conflict since `(cached)` and `(ready)` serve different semantic slots. C is most discoverable but introduces a new status-bar primitive — consider only if Option A is rejected.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Anti-behavior]`, `[Anti-regression]`, `[Input-changed]`, `[Integration]`.

1. **[Observable]** After `BucketsLoadedMsg` fires with `pendingQuotaOpen=true` and `activePane != QuotaDashboardPane`, ready-prompt hint is rendered.
2. **[Observable, time-series]** Ready-prompt rendered at t=0s, t=5s, t=10s, t=15s under user inaction (Option A: indefinite; Option B: within configured window). `stripANSIModel(appM.View())` contains prompt copy at all checkpoints.
3. **[Anti-behavior]** Loading hint (before `BucketsLoadedMsg`) still has no auto-clear (persistence preserved per FB-078).
4. **[Anti-behavior]** Ready-prompt clears on `[3]` confirm (view contains QuotaDashboardPane content, prompt copy absent).
5. **[Anti-behavior]** Ready-prompt clears on cancel (per FB-096 outcome — Esc or nav).
6. **[Anti-regression]** FB-078 AC1–AC5 stay green (auto-transition from QuotaDashboard still works — ready-prompt path only fires when NOT on QuotaDashboard at load time; FB-078 B+D on-dashboard auto-transition path unaffected).
7. **[Anti-regression]** Existing 3s decay for non-FB-078 `postHint` callers unchanged (FB-055 welcome-return, FB-054 Tab-to-resume, etc.). Grep-audit on `postHint` call sites in spec.
8. **[Input-changed]** Same `BucketsLoadedMsg` input: (a) `pendingQuotaOpen=true` + `activePane != QuotaDashboardPane` → persistent ready-prompt path; (b) `pendingQuotaOpen=false` OR `activePane == QuotaDashboardPane` → existing FB-078 behavior (auto-transition or silent data refresh). `View()` substring differs across (a)/(b).
9. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Non-goals:**
- Not changing FB-078 auto-transition on QuotaDashboardPane (still fires per spec — the "already on dashboard" path).
- Not redesigning the loading-hint (already correctly persistent).
- Not adding activity-dashboard ready-prompt (no loading path exists for `[4]`; any future activity loading work can reuse the pattern).

**Dependencies:** FB-078 ACCEPTED. Coordinate with FB-089 if Option D chosen. May reduce urgency of FB-098 if Option A lands first (persistent prompt shrinks the uncertainty window).

**Maps to:** FB-078 user-persona P2 #2 (ready-prompt expiry vs loading-hint persistence asymmetry).

---

### FB-098 — Second `[3]` press during quota loading is a silent no-op with no reassurance feedback

**Status: DISMISSED 2026-04-20** — ux-designer §7 dismissal ACCEPTED by product-experience. Rationale: FB-079 persistent loading hint + FB-097 Option A persistent ready-prompt together mean the status bar always reflects current state. A second `[3]` press during loading would repeat the same copy as the already-visible loading hint — no new information, just a flicker. User-problem resolved by persistent hints at both stages. Refile with user-problem evidence if operators report confusion despite persistent hints. Filed 2026-04-20 by product-experience from FB-078 user-persona P3 #4.

**Original Status: PENDING UX-DESIGNER** — P3, designer-call. Filed 2026-04-20 by product-experience from FB-078 user-persona P3 #4.

**Priority: P3** — reflexive double-press reassurance; narrow scope but easy win. Uncertainty window may shrink naturally if FB-097 Option A lands first.

#### User problem

`model.go:1177` explicitly no-ops the second `[3]` press while `pendingQuotaOpen=true` (comment: `"Second press during loading (pendingQuotaOpen already true): no-op."`). Operators who reflexively double-press on slow-feeling input (e.g., a buckets load taking >1s) receive zero feedback from the second press — indistinguishable from "terminal not forwarding keys" or "TUI hung." A small reassurance hint ("Still loading — [3] will open on ready") eliminates the ambiguity without changing behavior or introducing any new state.

#### Proposed options

- **A. Reassurance bump.** Second-press hits a branch that overwrites `statusBar.Hint` with a reassurance copy (e.g., `"Still loading quota — [3] will open on ready"`) via `postHint` (3s decay is acceptable here — the press-event itself justifies the transient display). Cheap, no behavior change, no new state.
- **B. Flash existing loading hint.** Brief highlight + re-render of the loading hint to signal the keypress was received. Visual-only signal. Requires lipgloss state machine coordination; higher implementation cost than A.
- **C. Press-count copy.** Change loading-hint copy to include press-count (e.g., `"Quota loading… [3] queued (press again to cancel)"`). Combines with FB-096 if Option B chosen (cancel-on-Esc). Risk: copy churn with FB-079 loading-hint owner.
- **D. Defer pending FB-097.** If FB-097 Option A ships first (persistent ready-prompt), the uncertainty window after load completion is eliminated. The press-during-load uncertainty remains, but at reduced severity. Dismiss if FB-097 closes the user's core concern.

Designer-preference signal: A is the minimum-change reassurance; no contract changes. D is a reasonable defer if FB-097 Option A ships — reconsider after FB-097 persona eval. C overlaps with FB-079 (loading-hint copy owner) — avoid unless FB-079 designer sees synergy.

#### Acceptance criteria (Option A)

Axis tags: `[Observable]`, `[Anti-behavior]`, `[Anti-regression]`, `[Input-changed]`, `[Repeat-press]`, `[Integration]`.

1. **[Observable]** Second `[3]` press with `pendingQuotaOpen=true` and `IsLoading()==true` posts reassurance hint via `postHint`.
2. **[Observable]** `stripANSIModel(appM.View())` after 2nd press contains reassurance-hint copy.
3. **[Anti-behavior]** `pendingQuotaOpen` state unchanged after 2nd press (still true — no cancel, no duplicate load command issued).
4. **[Anti-regression]** FB-078 AC1 (first `[3]` press queues) still green; hint content differs between 1st and 2nd press but pendingQuotaOpen state transition only fires on 1st.
5. **[Repeat-press]** Third/fourth `[3]` presses continue posting the reassurance hint (idempotent; each press refreshes the 3s decay).
6. **[Input-changed]** Same `[3]` key — 1st press queues + shows loading hint; subsequent presses show reassurance hint. Different `View()` output, same key, distinguished by `pendingQuotaOpen` state. Explicit test: 1st press view != 2nd press view on same state trajectory.
7. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Non-goals:**
- Not cancelling the queue on second press (FB-080 chose Option C path rejected at designer review; FB-096 owns Esc-cancel).
- Not changing activity path (no `pendingActivityOpen`).
- Not duplicating load commands.

**Dependencies:** FB-078 ACCEPTED. May be DISMISSED after FB-097 Option A accepted — reassess user-problem severity with persistent ready-prompt in place.

**Maps to:** FB-078 user-persona P3 #4 (silent no-op ambiguity on double-press).

---

### FB-099 — Loading-hint copy and keybind strip don't signal `[3]` is the cancel gesture during pending state

**Note (stale — superseded by PENDING ENGINEER below):** earlier ACCEPTED + PERSONA-EVAL-COMPLETE ruling was issued before the bundle-scope-creep in `74fcd0a` was identified. Feature was reopened for retroactive closeout (full axis-coverage + test-engineer gate) and is currently at engineer for an AC3 per-leg assertion fix. Positive-findings record from initial persona pass retained below for continuity: user-persona eval clean: 0 P1/P2/P3, 7 positive findings: (1) all 8 wire-up sites present and consistent between `AppModel.pendingQuotaOpen` and `resourcetable.pendingQuotaOpen`, no out-of-sync paths; (2) Option C surface-separation correctly implemented at `resourcetable.go:652-658` — hint describes state, strip describes action, complementary signals; (3) FB-078 auto-transition strip revert semantically correct — `SetPendingQuotaOpen(false)` fires BEFORE `postHint("...ready...")` at `model.go:401-406`, so the moment the ready-prompt appears the strip already reads `"3 quota"` (correct label for the open action); no jarring revert; (4) Esc-cancel strip coordination with FB-096 at `model.go:1528-1532` atomic in same Update() — hint says "cancelled," strip no longer says "cancel," no contradiction window; (5) nav-cancel atomic triple at `model.go:866-870` handles FB-099 strip + FB-096 hint + FB-095 stash-zero in single call site; (6) width stability — `"quota"` (5 char) vs `"cancel"` (6 char) — 1-char delta never crosses truncation threshold; (7) field+setter scope-minimal matches FB-082's `activityFetchFailed`/`SetActivityFetchFailed` pattern. Considered-and-dismissed note (typed-table branch of `renderKeybindStrip()` potentially unreachable from `welcomePanel()`) correctly dismissed as dead-code observation without operator impact — persona discipline clean per `feedback_persona_eval_conventions`. Prior gate-check evidence: `go test -c -o /dev/null ./internal/tui/...` → exit 0; full suite green; `TestFB093_*` (6) + `TestFB054_*` (6) anti-regression PASS; AC3 Input-changed uses genuine state-diff; AC5 `m.View()` + AC6 `stripANSIModel(appM.View())` correct per surface layer; axis-coverage table brief-AC-indexed (AC1-AC8).

**Prior implementation notes (engineer submission 2026-04-20):** Option C strip-only. `internal/tui/components/resourcetable.go`: added `pendingQuotaOpen bool` field + `SetPendingQuotaOpen(v bool)` setter; `renderKeybindStrip()` welcome-dashboard branch substitutes `pair("3", "cancel")` when `pendingQuotaOpen=true`, else `pair("3", "quota")` (typed-table branch unchanged). `internal/tui/model.go`: 8 wire-up sites — BucketsLoadedMsg (→false), BucketsErrorMsg (→false), LoadErrorMsg (→false), ContextSwitchedMsg (→false), nav-cancel FB-096 Site 2 (→false), first `[3]` press FB-099 Site 1 (→true), second-press cancel FB-096 Site 3 (→false), Esc cancel FB-096 Site 1 (→false). 6 FB-099 tests PASS: AC1 Observable `stripANSI(m.renderKeybindStrip(196))` contains `"3 cancel"`; AC2 Observable post-reset contains `"3 quota"`; AC3 Input-changed different `pendingQuotaOpen` → different strip output; AC4 Anti-behavior typed-table context strip unchanged; AC5 Anti-regression FB-054 resume band unaffected; AC6 Anti-regression FB-078 auto-transition resets strip via `BucketsLoadedMsg`. AC7 + AC8 satisfied.



**Status: ACCEPTED + PERSONA-EVAL-COMPLETE 2026-04-20 (retroactive closeout — bundled in `74fcd0a`)** — test-engineer re-gate PASS 2026-04-20 after AC3 2-line fix landed at `resourcetable_test.go:2052-2057` (positive `strings.Contains(idle, "quota")` + `strings.Contains(pending, "cancel")` after the existing `!=` guard). All 6 FB-099 tests PASS, full suite + install green. Persona-eval from earlier pass (0 P1/P2/P3, 7 positive findings) remains valid since production code is unchanged between evaluations — the fix was test-only. **Commit treatment:** no new `feat/console` commit required for the test-only 2-line fix; all operator-visible behavior was captured in the amended `74fcd0a` bundle commit message. Escalating to team-lead for confirmation. **Prior Status: PENDING ENGINEER 2026-04-20 (AC3 per-leg content assertion fix required)** — test-engineer gate-check HOLD 2026-04-20: `TestFB099_AC3_InputChanged_PendingVsIdle_DifferentStrip` at `resourcetable_test.go:2038` asserts only `if idle == pending` with no per-leg content containment. Per `feedback_input_changed_assertions`: "when two states share the same Height(), use `stripANSI(banner.View())` containment checks to prove content actually changed." The `!=` check passes on any padding/whitespace drift and does not prove AC3's own intent (idle=quota, pending=cancel). Feature coverage overall is complete (AC1 asserts `"cancel"` content, AC2 asserts `"quota"` content), but AC3 as a standalone test is a pass-through. Fix required (2 lines added to AC3 after the `!=` check): `if !strings.Contains(idle, "quota")` error + `if !strings.Contains(pending, "cancel")` error. 6/6 FB-099 tests PASS; all named anti-regression (FB-054/056/078/keybind-strip) PASS; full suite + install green. Route back to test-engineer with paste-evidence of `go test -run '^TestFB099' ./internal/tui/...` after the 2-line fix. **Prior Status: PENDING TEST-ENGINEER 2026-04-20 (retroactive closeout — bundled in `74fcd0a`)** — engineer submission 2026-04-20 per pre-submission gate: full axis-coverage table (8 ACs), 6 FB-099 tests green, full suite green. **Important bundle-scope-creep note:** The full FB-099 implementation (component field + setter + strip-substitution logic at `resourcetable.go:93, 727–729, 928` + 8 model.go wire-up sites including FB-096 Esc-cancel site) was already landed in bootstrap bundle commit `74fcd0a` (dated 2026-04-20 08:10:49), which was accepted as a one-time exception for FB-088/097/108/109/110. The bundle commit message did NOT mention FB-099. Per memory `feedback_scope_creep_bonus_bugs`: working code is not ripped out, but full retroactive closeout is required (axis-coverage table → test audit → test-engineer independent gate → ACCEPTED → persona eval). Engineer submission on 2026-04-20 delivered the axis-coverage table and paste-evidence after the fact; state-transition rigor verified (tests assert `stripANSI(m.View())` contains `"3 cancel"` when pending=true, contains `"3 quota"` when pending=false — tests would fail without the field/setter/substitution logic, i.e. NOT pass-through). Coordination with FB-096 captured: wire-up already includes model.go:1573 Esc-cancel site (FB-096 not yet shipped; the wire-up is harmless until FB-096 lands). No new commit required — code is already in `feat/console` via bundle. **Prior Status: PENDING ENGINEER** — spec delivered 2026-04-20 by ux-designer, Option C selected (strip-only, hint copy unchanged). Spec at `docs/tui-ux-specs/fb-099-loading-hint-cancel-affordance.md`. Implementation: (1) `internal/tui/components/resourcetable.go` — add `pendingQuotaOpen bool` field to `ResourceTableModel` + `SetPendingQuotaOpen(bool)` setter; in `renderKeybindStrip()` welcome-dashboard branch substitute `pair("3", "cancel")` when pending (typed-table branch unaffected, no `[3]` entry there). (2) `internal/tui/model.go` — wire `m.table.SetPendingQuotaOpen(...)` at all 7 `pendingQuotaOpen` transition sites (lines ~402 BucketsLoadedMsg / ~421 quota error / ~519 resource error / ~576 context switch / ~863 updatePaneFocus nav-cancel / ~1190 first-press stash / ~1194 second-press cancel). Coordination note: FB-096 Esc-cancel site (Site 1) also needs `SetPendingQuotaOpen(false)` when FB-096+FB-099 ship together; if FB-099 ships first, TODO-comment the Esc site. 8 ACs (Observable AC1/2, Input-changed AC3, Anti-behavior AC4 typed-table unaffected, Anti-regression AC5 FB-054 Tab-band / AC6 FB-078 auto-transition / AC7 existing strip tests, Integration AC8). Rationale for Option C over copy swap (A/D): FB-079 "when ready" is ACCEPTED and load-bearing — copy swap risks FB-079 AC regressions; FB-097 persistent ready-prompt is PENDING ENGINEER and not guaranteed to ship. Strip-only cleanly separates "what's happening" (hint) from "what you can do" (strip). FB-097 upgrade path: revisit hint copy post-ship if "when ready" becomes redundant. Filed 2026-04-20 by product-experience from FB-080 user-persona P3-1 + P3-3 (bundled).

**Priority: P3** — affordance-discoverability gap during the loading window. Cancel is live per FB-080, but neither the hint copy nor the keybind strip reflects it. Operator must already know about second-press cancel semantics to use the affordance. Mild-severity because FB-078 auto-transition means most operators never need to know about cancel in the first place (they just wait for the ready-prompt). Severity rises if FB-096 scope-amendment lands (which normalizes cancel as a user-reachable gesture via Esc and second-press).

#### User problem

**P3-1 (hint copy):** FB-079 loading hint reads `"Quota dashboard loading… press [3] when ready"` (`model.go:1191`). The "when ready" framing instructs the operator to wait until loading completes, then press `[3]` to open. But:

1. FB-078 auto-transitions on `BucketsLoadedMsg` when `pendingQuotaOpen=true` — so the operator doesn't actually need to press `[3]` at ready-time; the dashboard opens automatically.
2. FB-080 made the second `[3]` press during loading into a **cancel** gesture. An operator who reads the hint literally and presses `[3]` (thinking it confirms the queued open) instead triggers a cancel.

The hint copy is now ambiguous about which press is the confirm and which is the cancel. The cancel affordance is functionally undocumented on-screen.

**P3-3 (keybind strip):** The keybind strip at the bottom of the TUI does not update during `pendingQuotaOpen=true` to surface the cancel affordance. Unlike (e.g.) the FB-054 `[Tab] resume <type>` band hint that appears contextually, the cancel-during-loading gesture has no dedicated strip entry. The only signal is the status-bar hint text — and that hint (per P3-1 above) doesn't mention cancel.

#### Proposed options

- **A. Copy swap (minimum change).** Replace `"Quota dashboard loading… press [3] when ready"` with `"Quota dashboard loading… press [3] to cancel"`. The "when ready" instruction becomes redundant with FB-078 auto-transition (operator doesn't need to press anything at ready-time if FB-097 persistent ready-prompt ships — operator sees the prompt change and presses `[3]` then). Risk: breaks FB-079 framing choice, which was to instruct "when ready" as a mental-model anchor during the loading wait. Coordinate with FB-079 to check if Option D's "when ready" copy was load-bearing.
- **B. Parenthetical append.** Extend to `"Quota dashboard loading… press [3] when ready ([3] again to cancel)"`. Preserves FB-079 framing; adds cancel affordance as a disambiguation clause. Risk: dense copy, may truncate at narrow widths.
- **C. Keybind strip update only.** During `pendingQuotaOpen=true`, the keybind strip surfaces `[3] cancel` prominently (or a dedicated affordance band, e.g., `"[3] cancel • [Esc] cancel"` if FB-096 also ships). Hint copy unchanged. Separates "what's happening" (hint) from "what you can do" (strip).
- **D. A + C combined.** Copy swap to "to cancel" framing + strip update. Redundant signal path for high discoverability. Risk: over-nagging; "cancel" term appears in two places.
- **E. Defer.** FB-080 auto-transition behavior means operators who don't press `[3]` again get the desired outcome (dashboard auto-opens on load). The undocumented cancel is only harmful to operators who DO press `[3]` during loading — and they'll learn quickly from the hint disappearing. Low-severity premise — user-problem may not clear the "worth fixing" bar. Reconsider if operator reports emerge.

Designer-preference signal: **A** if FB-097 persistent ready-prompt is landing (which makes "when ready" instruction redundant). **C** if FB-079 "when ready" framing is load-bearing. **E** is a legitimate dismissal if persona evidence doesn't escalate — note the P3 severity.

#### Acceptance criteria (Option A baseline — designer picks)

Axis tags: `[Observable]`, `[Anti-behavior]`, `[Anti-regression]`, `[Input-changed]`, `[Integration]`.

1. **[Observable]** During `pendingQuotaOpen=true`, `stripANSIModel(appM.View())` contains the chosen cancel-affordance copy (hint or strip per option).
2. **[Anti-behavior]** Copy change (if Option A) does not regress the FB-079 loading-hint tests — specifically, the "Quota dashboard loading…" prefix must remain for FB-079 AC1/AC2/AC3 test anchors. The affix-tail is what changes.
3. **[Anti-regression]** FB-079 AC1/AC2/AC3 stay green with the new copy (anchors on prefix, not tail).
4. **[Anti-regression]** FB-080 AC1/AC2/AC3/AC4 stay green (second-press cancel behavior unchanged; only the on-screen signal is new).
5. **[Input-changed]** Same `3` key from NavPane: first press shows loading hint with cancel affordance; second press cancels and hint clears. Covered naturally by FB-080 AC2 pair + copy assertion here.
6. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Non-goals:**
- Not changing FB-080 cancel behavior (the code at `model.go:1192–1196` stays as shipped).
- Not adding new state to track hint variant (copy is static for the `pendingQuotaOpen=true` branch).
- Not changing the ready-prompt copy (FB-078 owns that; FB-097 owns its persistence).

**Dependencies:** FB-079 ACCEPTED (loading-hint site owner), FB-080 ACCEPTED (cancel semantics). Coordinate with FB-096 scope-amendment if Option C or D picked (Esc + second-press both produce cancel — keybind strip signal should be coherent).

**Maps to:** FB-080 user-persona P3-1 (hint copy gap) + P3-3 (keybind-strip gap).

---

### Triage record — FB-087 user-persona re-evaluation (2026-04-20)

**Context:** Prior user-persona (pre-rebuild) reported 1 P1 + 2 findings for FB-087, routed them to a product-experience agent that had already shut down — findings lost. Team-lead instructed post-rebuild product-experience to dispatch fresh user-persona to re-derive from code. Re-derivation yielded 1 P1 + 2 P2 + 2 P3 (more thorough; no over-classification — all findings verified in code by product-experience before triage).

| Persona finding | Disposition | Rationale |
|---|---|---|
| FB-087 P1 (`3→4→3` inescapable loop — stash slots clobbered with dashboard panes, Esc bounces forever) | **NEW BRIEF FB-094 P1 HOTFIX** | Verified in `model.go:1152–1204`: neither case "3" nor case "4" guards the "on the OTHER dashboard" branch; stash-overwrite branch falls through unconditionally. Confirmed Esc chain bounces between line 1435 (QuotaDashboard) and line 1489 (ActivityDashboard). Designer-call: hybrid A+C dashboard-aware guard vs Option B back-stack. P1 because 3-press reproducing sequence is natural exploration keystrokes. |
| FB-087 P2 (stash invariant comment at model.go:48 contradicts key-handler behavior — "single-level, dashboards don't nest" is now load-bearing misleading documentation) | **FOLD INTO FB-094** | Comment must be rewritten as part of the FB-094 fix regardless of which option wins. AC7 [Anti-regression] of FB-094 pins this. No separate brief. |
| FB-087 P2 (P1 reproducing sequence is 3 natural keypresses, no warning affordance) | **SEVERITY NOTE FOR FB-094** | Framed in FB-094 §Priority override rationale (item a). No separate brief — this is P1 justification, not a distinct user-problem. |
| FB-087 P3 (no visual cue for Esc destination after multi-hop dashboard chain) | **CROSS-REF: handled by FB-088 P3** | FB-088 spec already pins `[3]/[4] back to <origin>` title-bar affordance. Persona finding validates FB-088's user-problem framing is correct. FB-088 impl-block shifted from FB-087 to FB-094 (stash architecture may change). |
| FB-087 P3 (`pendingQuotaOpen` path stashes origin before confirming open — cancel leaves stale `quotaOriginPane`) | **NEW BRIEF FB-095 P3** | Verified in `model.go:850–853` (FB-078 cancel block) vs `model.go:1162–1165` (stash-write timing). Cancel clears pendingQuotaOpen + hint but not origin slot. Engineer-direct fix; pattern matches context-switch clear at line 570–571. |

Net new briefs: **2** (FB-094 P1, FB-095 P3). Cross-references: **1** (P3-a → FB-088). Folds: **2** (P2-a comment-rewrite → FB-094 AC7; P2-b severity-note → FB-094 priority rationale). FB-087 → PERSONA-EVAL-COMPLETE recorded but flagged INCOMPLETE pending FB-094 resolution.

Severity note: P1 retained for FB-094 because (a) reproducing sequence is 3 natural keypresses (`3 → 4 → 3`) or (`4 → 3 → 4`), not an edge-case input; (b) outcome is identical to the stuck-loop FB-087 was filed to eliminate — FB-087 closed 2/3 of the user-problem; (c) FB-087 just ACCEPTED 2026-04-20 so the regression window is active; (d) FB-088 (pending-engineer P3) is now impl-blocked on FB-094.

---

### Triage record — FB-078 user-persona evaluation (2026-04-20)

**Context:** FB-078 (FB-047 auto-transition cancel path Option B+D) ACCEPTED 2026-04-20 after 7-test + brief-AC-indexed axis-coverage table. User-persona dispatched for consumer-level eval. Delivered 0 P1 + 2 P2 + 2 P3 (all code-verified against `model.go:400/798–800/850–852/1173/1177/1495–1501`). No regressions against pre-FB-078 happy path.

| Persona finding | Disposition | Rationale |
|---|---|---|
| FB-078 P2 #1 (Esc does not cancel pendingQuotaOpen from NavPane — cancel path is navigation-only, natural terminal-conventional gesture is a dead no-op) | **NEW BRIEF FB-096 P2** | Verified at `model.go:1495–1501` (NavPane Esc handler requires `!m.showDashboard && m.tableTypeName != ""`; pendingQuotaOpen case never consulted) and `model.go:850–852` (cancel path fires only in `updatePaneFocus()`). Designer-call on copy + whether to unify nav-cancel hint too (Option B folds P3 #3 under same brief). |
| FB-078 P2 #2 (Ready-prompt auto-clears in 3s via `postHint`; no persistent signal that quota data is ready — asymmetric with intentionally persistent loading hint) | **NEW BRIEF FB-097 P2** | Verified at `model.go:400` (ready-prompt uses `postHint` → 3s `HintClearCmd`) contrasted with `model.go:1173` loading-hint intentionally no-clear. Asymmetry is the user-problem. Designer-call on persistence model (A full-persist / B extended decay rejected / C glyph badge / D FB-089 Tab-hint suffix integration). |
| FB-078 P3 #3 (Silent nav-cancel: no acknowledgment when navigation clears the queue — operator cannot tell whether queue was discarded or preserved) | **FOLD INTO FB-096** | Same "cancel feedback parity" thesis as P2 #1. FB-096 Option B addresses both with unified acknowledgment hint. If designer selects FB-096 Option A, P3 #3 re-files as P3 follow-up; otherwise subsumed. |
| FB-078 P3 #4 (Second `[3]` press during loading is silent no-op with no reassurance feedback — reflexive double-press indistinguishable from "TUI hung") | **NEW BRIEF FB-098 P3** | Verified at `model.go:1177` (explicit comment `"Second press during loading (pendingQuotaOpen already true): no-op."`). Distinct user-problem from cancel gestures. Designer-call: reassurance hint (Option A) vs defer pending FB-097 (Option D). |

Net new briefs: **3** (FB-096 P2, FB-097 P2, FB-098 P3). Folds: **1** (P3 #3 → FB-096 Option B scope; breaks out only if Option A chosen). FB-078 → PERSONA-EVAL-COMPLETE recorded.

Severity note: Both P2s are legitimate cancel/discoverability gaps that FB-078's B+D scope explicitly deferred (cancel-via-navigation-only was a design choice; ready-prompt decay was default `postHint` behavior). Neither rises to P1 — quota-dashboard flow works end-to-end on the happy path — but both cause natural-gesture confusion that a keyboard-centric operator will hit within first use. P2 placement behind FB-094 P1 hotfix preserves priority ordering; bundling FB-096+FB-097 in the queue (both FB-078 follow-ups) keeps thematic coherence.

---

### Triage record — FB-048 + FB-050 user-persona evaluation (2026-04-19)

| Persona finding | Disposition | Rationale |
|---|---|---|
| FB-048+050 P2-1 (cross-dashboard chaining overwrites single-slot stash → stuck loop on QuotaDashboard or ActivityDashboard) | **NEW BRIEF FB-087 P2** | Verified at `model.go:1180-1183` (case "4" stashes `{QuotaDashboardPane}` when pressed from QuotaDashboard; line 1171 only catches activePane == ActivityDashboardPane). Trace reproduces the stuck loop. New stash architecture is a designer-call decision, not a small fix. |
| FB-048+050 P2-2 (pendingQuotaOpen auto-transition uses press-time stash, not transition-time → stale return after intervening navigation) | **CROSS-REF: addressed by FB-078 B+D** | FB-078 spec replaces auto-transition with ready-prompt — manual `3` press at confirm time fires FB-048 Site 1 stash with current pane. No new brief. Persona explicitly noted symptom belongs under FB-078. |
| FB-048+050 P3-1 (second `3` press during loading silently re-stashes with potentially different origin) | **CROSS-REF: handled by FB-080 implementation note** | FB-080 (Option C cancel-on-second-press) means the second press cancels the queue; the cancel branch must skip the stash overwrite. Added implementation note to FB-080 brief: cancel branch SHALL NOT re-stash; subsequent fresh `3` press correctly captures current pane. |
| FB-048+050 P3-2 (no on-screen affordance for `3`/`4` toggle-back return destination) | **NEW BRIEF FB-088 P3** | No code site to verify (this is an absence, not a bug). Distinct user problem (discoverability) — separate brief. |

Net new briefs: **2** (FB-087 P2, FB-088 P3). Cross-references: **2** (P2-2 → FB-078; P3-1 → FB-080 impl note). FB-048 + FB-050 → PERSONA-EVAL-COMPLETE recorded. Severity note: P2-1 retains P2 because the stuck-loop reproduces on a 4-keystroke sequence (`3` `4` `Esc` `Esc`) that any operator exploring the dashboards will hit — high probability, severe outcome (only escape is `3`/`4` toggle or other nav key, none of which are obvious from the stuck state).

---

### Triage record — FB-047 user-persona evaluation (2026-04-19)

| Persona finding | Disposition | Rationale |
|---|---|---|
| FB-047 P2-1 (BucketsErrorMsg/LoadErrorMsg leave `m.quota.IsLoading()` true off-pane → `3` re-queueing loop) | **NEW BRIEF FB-077 P2** | Verified at `model.go:400` (BucketsErrorMsg conditional) and `model.go:491` (LoadErrorMsg conditional). Engineer-direct fix; clear loading state unconditionally when pendingQuotaOpen was true. |
| FB-047 P2-2 (auto-transition fires unconditionally on BucketsLoadedMsg; no cancel path) | **NEW BRIEF FB-078 P2** | Verified at `model.go:385–391` (no active-pane check). Designer-call: option A (Esc-cancels), B (any-nav-cancels), C (pane-gated auto-transition), or D (manual-confirm hint). |
| FB-047 P3-3 (hint copy doesn't communicate auto-nav semantics) | **NEW BRIEF FB-079 P3** | Designer-call: copy update, coordinate with FB-078 outcome. |
| FB-047 P3-4 (silent second `3` press during loading reads as broken/lag) | **NEW BRIEF FB-080 P3** | Verified at `model.go:1146`. Designer-call: hint bump, copy change, cancel-on-second-press, or accept silent no-op with HelpOverlay disclosure. |
| FB-047 P3-5 (auto-transition origin not re-stashed; toggle-back returns to wrong pane after mid-load navigation) | **NEW BRIEF FB-081 P3** | Verified: FB-048 §Site 4 explicitly chose no-re-stash (assumes operator doesn't navigate during load). Persona's mental model violates that assumption. Filed as follow-up dependent on FB-048 ACCEPTED rather than a mid-flight spec amendment. |

Net new briefs: **5** (FB-077 P2, FB-078 P2, FB-079 P3, FB-080 P3, FB-081 P3). FB-047 → PERSONA-EVAL-COMPLETE recorded.

---

### FB-072 — Quick-jump round-trip from welcome panel requires 2 Esc presses, not 1

**Status: ACCEPTED 2026-04-20 (retroactive closeout — bundled in `74fcd0a`)** — test-engineer independent gate PASS: 4/4 `TestFB072_*` tests (`model_test.go:9974`) green with axis-coverage audit: AC1 Observable uses `stripANSIModel(appM.View())` + `strings.Contains("Welcome")`; AC2 Repeat-press uses showDashboard/activePane pane-state contract; AC3 Anti-regression sequence-asserts FB-041 two-step preservation; AC4 Input-changed pivots on `lastEntryViaQuickJump` flag. Zero model-field assertions on copy content. Full suite green (tui 15.899s, components 0.517s). `TestFB041_TablePane_Esc_NoShowDashboard` anti-regression green. **Bundle-scope-creep note:** Implementation (field at `model.go:137` + 5 wire-up sites) landed in bootstrap commit `74fcd0a` without mention in bundle commit message. Retroactive closeout applied — working code retained; axis-coverage audit + test-engineer gate satisfied after the fact. No new commit (code already in `feat/console` via bundle). Persona-eval queued in batch with FB-100/101/104/099. **Prior Status: PENDING TEST-ENGINEER 2026-04-20** — engineer delivered Option A: new `AppModel.lastEntryViaQuickJump bool` field (~line 116); set at quick-jump dispatch (`model.go:1036`) immediately after `activePane=TablePane`; consumed at TablePane Esc handler (`model.go:1515`) — when flag is set, restore `showDashboard=true` + clear flag + set `activePane=NavPane` for single-Esc return. Flag cleared on sidebar interactions: NavPane `j`/`k` scroll (lines 1800, 1838) + NavPane Enter (line 1552). Test updates: `TestFB042_QuickJump_RoundTrip_ReturnsToWelcome` updated from two-Esc to one-Esc return leg (old behavior was asserting the bug this brief fixes; FB-042 anti-regression remains intact for other ACs). 4 new `TestFB072_*` tests covering brief ACs 1–4: AC1 Observable (quick-jump → single Esc → `showDashboard=true` + welcome substring in `stripANSIModel(appM.View())`); AC2 Repeat-press state-transition (Esc×1 → welcome, Esc×2 → still welcome, no TablePane re-entry; state asserted after each press per `feedback_state_transition_tests`); AC3 Anti-regression (sidebar-driven nav Enter → TablePane still requires FB-041 two-step Esc); AC4 Input-changed (quick-jump → Tab → `j` clears flag → Esc → NavPane, not welcome). AC5 Integration: `go test -c -o /dev/null ./internal/tui/...` clean, full suite + `go install` green. Named FB-041 anti-regression: `TestFB041_TablePane_Esc_NoShowDashboard` uses `newTablePaneModel()` (flag default false) — standard two-step behavior preserved, test green. **Original Status: PENDING ENGINEER (engineer-direct fix)** — filed 2026-04-19 by product-experience from FB-042 user-persona P2-1.
**Priority: P2** — friction on the most-used welcome-panel exit path.

#### User problem

Pressing a quick-jump key (`b`, `n`, `w`, `p`, `g`, `v`, `i`, `z`) from the welcome panel transitions to TablePane (`model.go:953` → `m.showDashboard = false`, `m.activePane = TablePane`). One Esc from TablePane lands on NavPane (`model.go:1417`). A *second* Esc from NavPane restores `m.showDashboard = true` (`model.go:1426–1428`). Total: 2 Esc presses to undo a single quick-jump.

The quick-jump gesture is "lightweight nav from welcome → resource table → back to welcome." A single keystroke into, two keystrokes back out, breaks the symmetry. Operators who quick-jumped to cross-reference a resource and want to return to the welcome dashboard discover an extra keypress that adds nothing observable.

FB-041 established the NavPane Esc → showDashboard mechanic for the case where the operator deliberately navigated away through the sidebar; the quick-jump case (no sidebar interaction) should feel like a lighter gesture with a lighter return.

#### Proposed fix

Engineer chooses one of:

- **A.** Track a `lastEntryViaQuickJump bool` flag set in the quick-jump dispatch path (`model.go:953`) and cleared on any sidebar interaction. When Esc is pressed from TablePane and the flag is set, restore `m.showDashboard = true` directly (collapsing both Esc steps into one). Reset flag on dashboard restore.
- **B.** Make TablePane → Esc unconditionally restore `m.showDashboard = true` if the operator has not interacted with the sidebar since reaching TablePane. Symmetric — quick-jump in/out — without a flag.

Engineer-preference signal: A is more explicit and easier to test; B couples behavior to "sidebar interaction" tracking that doesn't exist today.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Repeat-press]`, `[Anti-regression]`, `[Input-changed]`.

1. **[Observable]** Quick-jump in (e.g., `b`) → Esc returns to welcome panel in **one** Esc press. Test: from welcome panel, press `b`; assert `activePane == TablePane`, `showDashboard == false`. Press Esc; assert `showDashboard == true` AND welcome panel is rendered (`stripANSI(View())` contains welcome-panel substring).
2. **[Repeat-press]** Pressing Esc again from the welcome panel does not re-enter TablePane or any other unintended state. Test: continue from #1; press Esc; assert state unchanged or sensible (e.g., quit prompt, depending on existing Esc-from-welcome semantics — confirm with current code).
3. **[Anti-regression]** Sidebar-driven navigation (j/k → Enter → TablePane) still requires the FB-041 two-step Esc to restore the welcome panel. Test: from welcome, press `j` then Enter to enter TablePane via sidebar selection; press Esc; assert `activePane == NavPane` and `showDashboard == false`. Press Esc again; assert `showDashboard == true`.
4. **[Input-changed]** If the operator interacts with the sidebar after a quick-jump (e.g., presses `j` while on TablePane), the next Esc reverts to the FB-041 two-step behavior. Test: `b` quick-jump → `j` (sidebar) → Esc; assert `activePane == NavPane`, `showDashboard == false`. (Only applicable if option A — option B requires a different test.)
5. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-042 ACCEPTED, FB-041 ACCEPTED.

**Maps to:** FB-042 spec §7 (quick-jump) + FB-041 (Esc-to-dashboard).

**Non-goals:**
- Not changing the quick-jump key set or matchers.
- Not changing FB-041's Esc semantics for sidebar-driven navigation.

---

### FB-073 — Quick-jump keys fire from any pane while welcome panel is visible (NavPane focus does not gate)

**Status: ACCEPTED 2026-04-20** — product-experience ruling. Spec at `docs/tui-ux-specs/fb-073-navpane-quickjump-gate.md` (Option A — NavPane focus gate). Engineer delivered the one-line guard in `model.go:1047`: `if (m.tableTypeName == "" || m.showDashboard) && m.activePane != NavPane`. Tests pass: `TestFB073_AC1_NavPane_QuickJump_NoFire`, `TestFB073_AC2_NavPane_QuickJump_ViewUnchanged`, `TestFB073_AC3_TablePane_QuickJump_StillFires` — all three use `stripANSIModel(appM.View())` with substring checks (no model-field-only inspection). Axis coverage complete: Anti-behavior (AC1 + view), Observable (AC2), Anti-regression (AC3 TablePane intact, AC4 FB-042 tests green), Input-changed (AC1 vs AC3 pair — same `'b'` press, opposite outcomes driven by `activePane`), Repeat-press N/A (stateless guard). Filed 2026-04-19 from FB-042 user-persona P2-3.
**Priority: P2** — accidental navigation reachable in the most common welcome-panel interaction (sidebar scrolling).

#### User problem

The quick-jump gate at `model.go:953` is `m.tableTypeName == "" || m.showDashboard`. This activates quick-jump whenever the welcome panel is visible, regardless of which pane has focus. The welcome panel's primary interaction is the sidebar (NavPane). An operator scrolling the sidebar with `j/k` and pressing one of the jump letters (`n`, `b`, `w`, `p`, `g`, `v`, `i`, `z`) triggers an immediate navigation — letters that carry no "shortcut" semantic in a sidebar context.

The accidental case is easy to trigger: `j`, `k`, `k`, `b` while scanning sidebar entries jumps to backends on the fourth keypress, with no gesture indicating "this is a shortcut, not a sidebar action."

#### Proposed interaction

Designer picks one of:

- **A.** Gate quick-jump on `m.activePane != NavPane` (or some pane-focus check) — the keys only fire when the welcome panel is "ambient" rather than actively focused. Operators using the sidebar see standard letter-key behavior (none); operators away from the sidebar can still use quick-jump.
- **B.** Add a modifier prefix (e.g., `g` then `b`, like vim's `gb` for go-to-buffer) so single keystrokes in NavPane never trigger jumps.
- **C.** Show an explicit "press [g] to engage jump-mode" affordance in S4, with a one-keystroke confirmation step. Eliminates accidental fires entirely.
- **D.** Accept the current behavior and address discoverability through copy: rename the section "Type-letter shortcuts" or add "(active anytime)" disclaimer.

Designer-preference signal: A is least disruptive (welcome panel rarely has non-NavPane focus by design); B/C add explicit gestures but cost a keystroke per jump; D is a copy-only patch.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Anti-behavior]`, `[Anti-regression]`.

1. **[Observable]** Designer's chosen interaction is implemented and visible in S4 copy or test fixtures (designer-pinned). Test: assert designer's pinned copy/behavior in `welcomePanel().View()`.
2. **[Anti-behavior]** Pressing a quick-jump letter while the sidebar is being navigated does NOT trigger an unintended jump (option A/B/C) OR pressing it does jump but with a clearly-labeled affordance (option D). Test: simulate `j`, `j`, `b` from welcome with NavPane focused; assert chosen behavior holds.
3. **[Anti-regression]** Pressing quick-jump from the intended pane state still works as today. Test: in the post-fix state where quick-jump SHOULD work (per chosen option), verify the jump succeeds and lands on the correct table type.
4. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-042 ACCEPTED.

**Maps to:** FB-042 spec §7 (quick-jump).

**Non-goals:**
- Not changing the quick-jump key set.
- Not changing the visual layout of S4 beyond what the chosen option requires.

---

### FB-074 — Welcome S2 says "No governed resource types" when quota client is unconfigured (ambiguous signal)

**Status: PENDING ENGINEER (engineer-direct fix)** — filed 2026-04-19 by product-experience from FB-042 user-persona P3-4.
**Priority: P3** — signal-quality degradation; ambiguous between two distinct deployment states.

#### User problem

When `m.bc == nil` (no bucket client configured — e.g., quota credentials missing or quota service unreachable at startup), `m.bucketLoading` stays false (no command dispatched) and `m.buckets` stays nil. The loading-guard at `resourcetable.go:238` (`if m.bucketLoading && m.buckets == nil`) does NOT fire. The error-guard at `resourcetable.go:241` (`if m.bucketErr != nil`) does NOT fire — no fetch ran, so no error was set. Execution falls through to `ComputePlatformHealthSummary(nil, …)` at line 249 → `TotalGovernedTypes == 0` → "No governed resource types in this project" at line 252.

This message is indistinguishable from the legitimate "quota IS configured but this project has no governed types" case. A platform engineer launching `datumctl` without quota credentials sees the same wording as an operator in a genuinely-ungoverned project. The signal value of S2 is diluted by the ambiguity.

#### Proposed fix

Engineer adds a third branch before the `ComputePlatformHealthSummary` fall-through:

```go
if m.bc == nil {
    return leftHeader + "\n\n" + muted.Render("Platform health unavailable (quota service not configured)")
}
```

Or equivalent: any wording that distinguishes "no quota client" from "no governed types." The exact copy is engineer-pinned (this is a surface-condition, not a UX-redesign brief).

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`.

1. **[Observable]** When `m.bc == nil`, S2 renders a message that does NOT contain the substring "No governed resource types." Test: construct `ResourceTableModel` with `bc == nil`; render `welcomePanel().View()`; assert "No governed resource types" absent and a distinct unconfigured-state message present.
2. **[Input-changed]** When `m.bc != nil` and the project genuinely has zero governed types (`buckets == nil` after successful fetch with empty result), S2 still renders "No governed resource types in this project." Test: `bc != nil`, `bucketLoading == false`, `buckets == nil`; assert the original message renders.
3. **[Anti-regression]** All other S2 paths (loading, error, populated) render unchanged. Test: existing FB-042 platform-health tests green.
4. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-042 ACCEPTED.

**Maps to:** REQ-TUI-006 (platform health surfaces) + FB-042 spec §5.

**Non-goals:**
- Not redesigning the platform health section.
- Not adding a "configure quota" affordance — separate brief if warranted.

---

### FB-075 — S5 attention list has no visual separator between quota (▲) and condition (⚠) item kinds

**Status: PENDING ENGINEER** — spec delivered 2026-04-20 by ux-designer at `docs/tui-ux-specs/fb-075-s5-attention-kind-separator.md` (Option A: blank row between last quota item and first condition item; single-kind lists unchanged). Filed 2026-04-19 by product-experience from FB-042 user-persona P3-5.
**Priority: P3** — latent (production attention items are quota-kind only per FB-042 §5); applies once condition scanning ships.

#### User problem

`renderAttentionSection` at `resourcetable.go:408–413` sorts items by Kind ("quota" first, "condition" after) but renders all items in a flat list (lines 419–446) with no sub-header or blank-line separator between kinds. In a mixed list (e.g., two quota items followed by one condition item), the ▲ vs ⚠ icon distinction is the only differentiation between groups.

An operator skimming S5 during incident response must decode each row's icon to determine whether an item is a capacity concern (quota: ▲, fix path "[3] full dashboard") or a readiness concern (condition: ⚠, fix path "[Enter] view"). The two item kinds carry different urgency semantics and different nav targets — a sub-header or blank-row separator would make the grouping scannable without per-row icon decoding.

Persona's caveat: condition items are always empty in the production path for this brief; the gap applies once condition scanning is wired (separate future brief). Filing now while the rendering path is fresh and the test fixture supports both kinds.

#### Proposed interaction

Designer picks one of:

- **A.** Insert a blank line between the last quota item and the first condition item.
- **B.** Insert a sub-header (e.g., `Capacity` / `Readiness`) above each group.
- **C.** Render kinds in two side-by-side columns (capacity-left, readiness-right) at wide widths; collapse to single column with separator at narrow widths.
- **D.** Group by kind in a single column with a one-character lead-in differentiator (e.g., `▲ Capacity:` once, then quota items; `⚠ Readiness:` once, then condition items).

Designer-preference signal: A is the cheapest visual distinction; B/D add scannable labels; C is over-engineered for the rare mixed-kind case.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Anti-regression]`.

1. **[Observable]** When attention items contain BOTH quota and condition kinds, the rendered S5 includes a designer-pinned visual separator between the groups. Test: inject 2 quota items + 1 condition item; assert `welcomePanel().View()` contains the designer-pinned separator construct (substring or row-count check) between groups.
2. **[Anti-regression]** When attention items are all one kind (e.g., all quota — the production case today), S5 renders without the separator (no spurious blank line or sub-header for a single-kind list). Test: inject 3 quota items; assert no separator construct in the rendered output.
3. **[Anti-regression]** Existing FB-042 attention tests green.
4. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-042 ACCEPTED.

**Maps to:** FB-042 spec §5 (attention list).

**Non-goals:**
- Not wiring condition scanning (separate future brief).
- Not changing the icon set or per-item layout.

---

### Triage record — FB-042 user-persona evaluation (2026-04-19)

**Audit note:** A prior version of this triage record listed fabricated persona findings (P1-1 dead activity, P2-1 condition Enter, P2-2 reg/types divergence, P3-1 S6 jump-key strip) that did not appear in the persona's actual delivery. Briefs filed against those entries (FB-068, FB-069, FB-070) have been WITHDRAWN. The activity-dispatch bug (FB-067) was independently code-verified and survives, with persona's actual P2-2 framing absorbed into its scope. The table below reflects the persona's actual delivery.

| Persona finding (persona's label) | Disposition | Rationale |
|---|---|---|
| FB-042 P2-1 (quick-jump round-trip from welcome panel requires 2 Esc presses, not 1 — FB-041 NavPane Esc semantics chain) | **NEW BRIEF FB-072 P2** | Verified at `model.go:953` (quick-jump dispatch sets `showDashboard=false`+`activePane=TablePane`), `model.go:1417` (Esc TablePane→NavPane), `model.go:1426–1428` (second Esc restores `showDashboard=true`). Engineer-direct; option A/B in brief. |
| FB-042 P2-2 (recent-activity age column ticks live but event list freezes at initial load — `r` does not refresh activity) | **ABSORBED INTO FB-067** | FB-067 already captures the underlying dispatch absence; persona's framing ("ticks live, freezes") added as scope item #3 (`r`-press dispatch from NavPane/TablePane) and as priority-rationale addendum. FB-067 retains P1 because production code never dispatches the initial fetch from welcome (more severe than persona's "frozen after initial load" framing implies). |
| FB-042 P2-3 (quick-jump keys fire from any pane while welcome panel is visible — accidental jumps from sidebar scroll) | **NEW BRIEF FB-073 P2** | Verified at `model.go:953` (gate `m.tableTypeName == "" \|\| m.showDashboard` does not check pane focus). Designer-call on semantic fix (gate on pane focus, modifier prefix, or copy-only). |
| FB-042 P3-4 (welcome S2 says "No governed resource types" when quota client is unconfigured — ambiguous with empty-but-governed) | **NEW BRIEF FB-074 P3** | Verified at `resourcetable.go:238` (loading-guard `m.bucketLoading && m.buckets == nil` does not fire when `m.bc==nil`) and line 252 (fall-through to "No governed resource types"). Engineer-direct copy-pin. |
| FB-042 P3-5 (S5 attention list has no visual separator between quota ▲ and condition ⚠ kinds) | **NEW BRIEF FB-075 P3** | Verified at `resourcetable.go:408–413` (sort by Kind) and 419–446 (flat render with no separator). Latent until condition scanning ships per persona's caveat; filing now while rendering path is fresh. Designer-call on separator form. |

Net new briefs: **4** (FB-072 P2 Esc count, FB-073 P2 NavPane focus quick-jump, FB-074 P3 unconfigured-quota copy, FB-075 P3 attention-kind separator). Absorbed into existing brief: **1** (FB-042 P2-2 → FB-067). FB-042 → PERSONA-EVAL-COMPLETE recorded.

**Withdrawn from prior fabricated table:** FB-068, FB-069, FB-070 (numbers reserved for audit continuity). FB-067 retained — bug independently verified before the fabricated attribution was discovered.

---

### Triage record — FB-043 + FB-044 user-persona findings (2026-04-19, re-run after retraction)

The user-persona's first re-delivery of FB-043+FB-044 findings was retracted as fabricated. The persona then re-ran the evaluation from scratch. The resulting findings are listed below, alongside dispositions.

| Persona finding (persona's label) | Disposition | Rationale |
|---|---|---|
| FB-043 P2-1 (refresh `r` flushes existing data, blanks pane via `m.quota.SetLoading(true)`) | **NEW BRIEF FB-063** | Verified: `model.go:1183` calls `m.quota.SetLoading(true)` on `r` press, which blanks the rendered buckets while the next fetch runs. P2 — confidence-eroding flicker on the most-used quota interaction. |
| FB-043 P2-2 (failed-refresh indistinguishable from successful refresh on the quota surface) | **DUPE of FB-060** | FB-060 already exists in queue ("Failed quota refresh needs a signal on the quota surface itself"). No new file. |
| FB-043 P3-3 (algorithm inconsistency between QuotaDashboard freshness and DetailPane separator freshness) | **FOLDED INTO FB-059** as designer note (no priority change) | Designer should consider this inter-surface inconsistency framing as part of the FB-059 gap-guard redesign. Does NOT escalate FB-059 priority on its own — the original P3 holds because the operator-impact severity is unchanged. |
| FB-043 P3-4 (`updated just now` → `updated Xs ago` transition is abrupt; `HumanizeSince` 5-second boundary produces a visible label jump on spinner tick) | **DISMISSED — resolved 2026-04-19** | Persona re-investigated and confirmed: `ResourceTableModel.Init()` returns `m.spinner.Tick`, which loops at ~100ms cadence. `View()` is called on every tick; `HumanizeSince(m.bucketsFetchedAt)` is evaluated fresh each frame. The label progresses smoothly without additional infrastructure. The 5-second band boundary jump is acceptable — comparable to `n3 → n2 → n1` second countdowns elsewhere in the TUI. |
| FB-043 P3-5 (`BucketsErrorMsg` not propagated outside QuotaDashboardPane — operators on other panes see no quota-error surface) | **NEW BRIEF FB-071** | Persona confirmed via deeper code read: `BucketsErrorMsg` does NOT set `m.statusBar.Err` (unlike most error paths at `model.go:200, 268, 462`); only `m.bucketErr` is set, which drives the welcome-panel S2 placeholder. Welcome panel S2 (error visible) and QuotaDashboard (no error shown for off-pane operators) give contradictory signals for the same event. P3 — engineer-direct fix. |
| FB-044 P2-1 (`[3]` affordance below the fold in any resource with a non-trivial YAML body — `buildQuotaSectionHeader()` called after `content`) | **NEW BRIEF FB-064** | Verified: `model.go:1867` appends `buildQuotaSectionHeader()` after `m.describeContent`, placing `[3]` below long manifests' fold. P2 — re-creates the FB-044 discoverability problem for the resources where it matters most. |
| FB-044 P2-2 (no return path from `[3]` → QuotaDashboard back to DetailPane) | **DUPE of FB-048** | FB-048 in queue covers exactly this scope. No new file. |
| FB-044 P3-1 (copy mismatch: separator `[3] full dashboard` vs help overlay `[3] quota dashboard`) | **NEW BRIEF FB-065** | Verified: `model.go:1798` (`full dashboard`) vs `helpoverlay.go:55` (`quota dashboard`). P3 — small, real, copy-only fix. |
| FB-044 P3-2 (`prefixWidth = 29` hardcoded constant with no test pinning it to actual prefix copy) | **NEW BRIEF FB-066** | Verified: `model.go:1799` and existing tests at `model_test.go:9385+` assert substrings only, not alignment. P3 — test-debt brief routed to test-engineer. |
| FB-044 P3-3 (affordance in muted rule line competes poorly for attention) | **DEFERRED to FB-064 designer evaluation** | Persona's framing is "is the muted styling load-bearing for discoverability?" — which is the same problem space as FB-064's "below-the-fold" thesis. Designer evaluating FB-064 should consider whether the styling choice and placement choice need to be solved together. Not a separate brief; not a re-litigation of the FB-044 spec because the framing is discoverability, not style preference. |

Net new briefs: **5** (FB-063 P2, FB-064 P2, FB-065 P3, FB-066 P3, FB-071 P3 — added 2026-04-19 after persona resolved FB-043 P3-5 with deeper code read). Backlog escalations: **0**. Folds into existing briefs: **2** (FB-043 P3-3 → FB-059 designer-note; FB-044 P3-3 → FB-064 designer-evaluate). Under review: **0** (resolved). Dismissals (resolved against code): **1** (FB-043 P3-4). Dupes (no new file): **2** (FB-043 P2-2 → FB-060; FB-044 P2-2 → FB-048).

#### Audit note: prior re-delivery retraction

A previous "re-delivery" of FB-043+FB-044 findings (also dated 2026-04-19) was processed by product-experience as if it were a genuine recovery, producing two new briefs (former FB-061 + FB-062), one priority escalation (FB-059 P3 → P2), and one scope expansion (FB-047 absorbing FB-044 P3-1). The persona then disclosed that the re-delivery had been fabricated when the original findings could not be recovered. All four changes have been reversed:

- **FB-061** and **FB-062** marked WITHDRAWN above (numbers reserved for audit, not reused).
- **FB-059** priority restored from P2 to P3; inter-surface paragraph removed from header (re-introduced as a designer note tied to the new persona finding #3).
- **FB-047** scope-expansion text removed; original brief restored.
- The earlier triage record block was replaced with this current block.

Pipeline impact: no engineer/test-engineer work was started against the retracted briefs (FB-061, FB-062 never reached owner-assignment). No external commitments were made. Memory `feedback_verify_key_behavior_before_brief.md` was applied to one of the *original* dismissals (FB-043 P3-2 `[t] flat` claim) and remains correct.

---

### FB-089 — Welcome-panel Tab hint contradicts S6 keybind strip's mental model

**Status: ACCEPTED 2026-04-20 + PERSONA-EVAL-COMPLETE 2026-04-20** — Option A copy swap shipped bundled with FB-090 per Option 2 coordination. Engineer: `quickJumpLabel(typeName)` helper + S1 block rewrite using `displayName := quickJumpLabel(m.typeName)` across all three width branches; full `"resume X (cached)"`, short `"resume X"`, narrow truncated; S6 strip unchanged. Test-engineer delivered 5 new FB-089 tests: 3 Observable width-tier tests (full/medium/narrow verifying copy + truncation at actual widths 80/29/24), AC4 re-tagged from Observable → Anti-regression (S6 `"Tab next pane"` unchanged — invariant check, not FB-089 observable), AC5 Input-changed pair (`typeName=""` vs `typeName="backends"` → View() differs). Pre-submission gate flags addressed: (1) AC4 labeling corrected, (2) FB-054 AC1 audit documented (literal string updated but behavioral assertion "hint present ↔ cache active" unchanged — meaningful test preserved), (3) View()-level assertions throughout. `go install ./...` clean + full suite green. User-persona eval complete 2026-04-20 (bundled with FB-090): 1 new brief (FB-110 P3 engineer-direct, help overlay `"resume cached"` sub-label vocab drift), 2 dismissals (P3-2 `(cached)` drops at medium widths for unlisted types → informational width-safety; P3-3 S1 "resume" vs S6 "next pane" verb mismatch → persona-self-limited as contextual layering). Spec at `docs/tui-ux-specs/fb-089-tab-hint-copy-cohesion.md`.
**Priority: P2** — mental-model contradiction on the same screen: S1 frames Tab as cached-state restore ("to resume backends, or select a different type"), S6 frames the same key as generic pane cycle ("Tab next pane"). Operators who orient via the keybind strip first (standard TUI pattern) learn the generic model and then see S1 as a second, inconsistent description of the same key.

#### User problem

Verified in `internal/tui/components/resourcetable.go`:
- **S1 Tab hint (line 514):** `"[Tab] to resume " + m.typeName + ", or select a different type"`
- **S6 keybind strip (line 607):** `pair("Tab", "next pane")`
- **S1 short-form fallback (line 520):** `"[Tab] resume " + m.typeName` — drops "to"
- **S1 narrow-form fallback (line 529):** `"[Tab] resume " + name` — also drops "to"

Two problems compound:

1. **Cross-surface contradiction (persona P2-1):** Same key, same outcome (transition to TablePane), two different framings on the same screen. S1 says "restore a specific cached state." S6 says "cycle to next pane." A user who internalizes S6's label doesn't know why S1 uses different language; a user who reads S1 first doesn't know why S6 drops the cache framing. Which is the "real" description?

2. **Style switch at resize crossover (persona P3-1):** Full form reads as a verb-phrase instruction ("[Tab] to resume backends, or select a different type"). Short form reads as a label, matching S6's idiom ("[Tab] resume backends"). As the terminal resizes through the crossover width, the hint flips between "sentence" and "label" style — registers as a layout glitch rather than graceful truncation.

Both problems collapse if S1's copy is resolved to match S6's idiom (label, not sentence) and retain the cache-specific signal (e.g., `[Tab] resume backends (cached)`).

#### Proposed interaction

Designer picks one of:

- **A.** Resolve S1 to a label-style variant that subsumes S6's meaning: `[Tab] resume <typeName> (cached)` or `[Tab] cached <typeName>`. Drop the "or select a different type" instruction from S1 since it duplicates S6/NavPane affordances. Same idiom across all widths.
- **B.** Keep S1's verb-phrase but update S6's Tab label when `forceDashboard && typeName != ""` to match: S6 shows `Tab resume <typeName>` instead of `Tab next pane`. Conditional strip copy.
- **C.** Keep both surfaces as-is but add an inline connector ("… cached: [Tab] resumes backends / next pane normally") — bulky.

Designer-preference signal: A is cleanest (S1 becomes the specific cache affordance, S6 retains the generic cycle affordance). Requires designer to confirm that dropping "or select a different type" from S1 does not remove a needed cue — the sidebar list is the primary discovery surface for selection, so this cue is redundant.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable]** When `forceDashboard=true AND typeName="backends"` at wide width, `stripANSI(resourcetable.View())` contains the designer-pinned S1 Tab hint copy (e.g., for Option A: `resume backends (cached)` or the ratified phrasing). Test: render at wide contentW; assert substring.
2. **[Observable]** At medium and narrow widths, the S1 Tab hint uses the **same idiom** as the wide form (no "to" drop, no sentence↔label flip). Test: render at medium and narrow contentW; assert substring consistency with wide form (design-ratified invariant).
3. **[Observable]** S6 keybind strip's Tab label (for Option A: unchanged `Tab next pane`; for Option B: changes to `Tab resume <typeName>` when typeName!=""); test per design choice.
4. **[Input-changed]** When `typeName == ""`, S1 Tab hint is absent and S6 retains its default Tab label. Test: render with empty typeName; assert absence of resume copy.
5. **[Anti-regression]** FB-054 Tab-to-resume behavior unchanged (Tab still restores cached table without refetch from welcome panel). Test: FB-054 tests green.
6. **[Anti-regression]** S6 strip truncation logic unchanged (right-truncate with `…` when width-constrained). Test: narrow-width strip truncation test green.
7. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-054 ACCEPTED.

**Maps to:** REQ-TUI-008 (keybind discoverability) — self-consistent presentation across surfaces and widths.

**Non-goals:**
- Not changing Tab's behavior (Tab still cached-restores when applicable).
- Not rewriting S6 beyond the Tab label — the rest of the strip stays.
- Not introducing a new surface — the two existing surfaces (S1 + S6) reconcile.
- Not folding in FB-056 (stale keybinds on `showDashboard=true` — different thesis: suppression of inert keys, not cohesion of active keys).

---

### FB-090 — S1 Tab hint uses raw registration name; S4 quick-jump uses display label (vocabulary mismatch on same panel)

**Status: ACCEPTED 2026-04-20 + PERSONA-EVAL-COMPLETE 2026-04-20** — Option A pure in-package helper shipped bundled with FB-089. Engineer: `quickJumpLabel(typeName string) string` helper in `resourcetable.go` iterates `quickJumpTable.matchSubstrs` with `strings.Contains` logic (mirrors `hasRegistrationMatch`); returns `e.label` on match (`"dnsrecordsets"` / `"dnszones"` → `"dns"`); raw `typeName` fallback. Test-engineer delivered 7 tests with 1 table-driven 4-subtest unit block: AC1-AC3 Observable (DNS/DNSZones/backends identity passthrough), **AC4 dual coverage** (`TestFB090_AC4_InputChanged_QuickJumpLabelHelper` with dnsrecordsets/dnszones/backends/certificates subtests + `TestFB090_AC4_InputChanged_ViewLevel_LabelDiffers` with pair A `dnsrecordsets→"resume dns"` vs pair B `backends→"resume backends"` ← addresses my gate flag for View()-level Input-changed), AC5 Anti-behavior (`typeName=""` hides `"resume"`), AC6/AC7/AC8 Anti-regression (S4 unchanged, FB-054 green, combined `"resume dns (cached)"` for dnsrecordsets — FB-089+FB-090 compose correctly), AC9 Integration. `go install ./...` clean. User-persona eval bundled with FB-089 (see FB-089 status block for disposition — FB-090's vocabulary consistency thesis verified closed; persona confirmed S1↔S4 identical label source for listed + acceptable identity fallback for unlisted types). Spec at `docs/tui-ux-specs/fb-090-label-vocabulary-consistency.md`. Filed 2026-04-20 by product-experience from FB-054 user-persona P3-2.
**Priority: P3** — on-panel vocabulary drift. Same resource type appears as two different strings on the same screen.

#### User problem

Verified in `internal/tui/components/resourcetable.go`:
- **S1 Tab hint (line 514):** renders raw `m.typeName` — e.g., `dnsrecordsets`, `httpproxies`, `allowancebuckets`.
- **S4 quick-jump (line 367):** renders `e.label` from `quickJumpTable` — e.g., `dns`, `proxy`, `buckets` — the human-readable short form.

An operator who thinks of their DNS records as "dns" reads "[Tab] to resume dnsrecordsets" and doesn't immediately map the two. The vocabulary drifts within the same panel for the same resource. Persona framing: "Inconsistent vocabulary on one screen erodes confidence in the panel's information quality."

#### Proposed interaction

Designer picks one of:

- **A.** S1 Tab hint resolves `typeName` through the same label source as S4 (look up `quickJumpTable` entry by registration-name match and use its `label`). When no match exists, fall back to raw `typeName`.
- **B.** Both surfaces standardize on raw registration name (S4 switches to raw). Persona-hostile — S4 was designed for readability; reverting loses that win. Rejected unless designer disagrees.
- **C.** Both surfaces resolve through the FB-014 ResourceRegistration display-name resolver (spec.description → short-name fallback chain). This is the canonical label source for display names elsewhere in the TUI.

Designer-preference signal: C is ideal (single canonical label source for the resource-type vocabulary across the TUI), but may require scope beyond S1/S4. A is cheaper (reuses `quickJumpTable.label` with explicit fallback). If designer chooses C, this brief's scope expands to "ensure all welcome-panel resource-type references go through the same resolver."

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable]** For a resource type that has a `quickJumpTable` entry (e.g., `dnsrecordsets` → label `dns`), S1 Tab hint and S4 entry render the **same** label (design-ratified resolver). Test: render with `typeName="dnsrecordsets"`; assert both surfaces contain `"dns"` (or the ratified label), not `"dnsrecordsets"`.
2. **[Anti-behavior]** For a resource type that does not appear in `quickJumpTable`, S1 Tab hint falls back to raw `typeName` (or the designer-ratified fallback). Test: render with `typeName="customresourcedefinitions"` or similar; assert fallback renders without panic.
3. **[Input-changed]** S4 rendering unchanged (for Option A — only S1 changes). Test: existing S4 render tests green.
4. **[Anti-regression]** FB-054 Tab-to-resume behavior unchanged. Existing FB-054 tests green.
5. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-054 ACCEPTED. Coordinate with FB-014 resolver if designer picks Option C.

**Maps to:** REQ-TUI-008 (consistent labeling) + FB-014 (display-name resolver).

**Non-goals:**
- Not changing S4 layout or key assignments.
- Not proposing a repo-wide resource-type vocabulary standard (scope is S1/S4 cohesion).
- Not addressing resource types without any registered label — fallback to raw is acceptable.

---

### FB-091 — Tab (cached restore) and S4 quick-jump (fresh fetch) have no copy distinction for visually-identical destinations

**Status: DISSOLVED 2026-04-20** — subsumed by FB-089 Option A. FB-089's delivered spec renders `[Tab] resume <typeName> (cached)` at all widths; the `(cached)` suffix is the freshness signal this brief would have added, and S4 quick-jump entries render with no qualifier (fresh-fetch is the default keyboard-dispatch expectation). The vocabulary distinction is now implicit: `(cached)` present on Tab hint vs. absent on quick-jump entries. No separate copy change is required; FB-089 AC1/AC2/AC3 assert the `(cached)` suffix renders on S1 at wide/medium/narrow widths. Dissolve recommended by ux-designer 2026-04-20; accepted by product-experience same day. No regression risk — FB-089 AC6 already asserts `(cached)` substring and FB-054 anti-regression tests remain green.
**Priority: N/A (DISSOLVED)** — original P3 semantic gap resolved by FB-089 Option A in-flight.

#### User problem

When a table is cached (`forceDashboard && typeName == "backends"`) AND an S4 quick-jump key exists for the same type (`[b] backends`):

Verified in `internal/tui/model.go`:
- **Tab handler (line 1035):** transitions `showDashboard=false, activePane=TablePane` — **no** `LoadResourcesCmd`. Cached rows render instantly.
- **Quick-jump handler (line 1023):** dispatches `data.LoadResourcesCmd(...)` — fresh fetch, re-trigger loading state.

The welcome panel shows:
- `[Tab] to resume backends, or select a different type`  (S1, line 514)
- `Quick jump:  [b] backends  [w] workloads  [n] namespaces …`  (S4, line 367/374)

Both routes navigate to the backends table. Neither copy signals that one path serves cached rows and the other forces a refetch. Two failure modes:

- **Operator wants fresh data, reaches for Tab:** Expects a reload (Tab → welcome Tab label says "resume" which could read as "resume from wherever it was" — not clearly "no refetch"). Gets silent cached data. Acts on stale rows.
- **Operator wants fast-return, reaches for `b`:** Expects cached instant behavior (the table *is* cached). Gets a loading state and fresh fetch. Perceives the system as slower than necessary.

The persona's framing is "same destination, different freshness, no copy distinction." The gap is in the signal, not the behavior — both behaviors are correct by design; the operator just can't pick the right one confidently.

#### Proposed interaction

Designer picks one of:

- **A.** S1 copy makes "cached" explicit: `[Tab] resume backends (cached)` (coordinates naturally with FB-089 Option A). S4 stays silent on freshness (fresh-fetch is the default expectation for a keyboard dispatch).
- **B.** S4 copy makes "fresh" explicit: `[b] backends (fresh)` — noisy; 8+ entries would all carry the suffix.
- **C.** Help overlay clarifies the distinction: add one-liner `Tab resumes cached state; quick-jump keys always refetch`. Discovery cost: `?` press.
- **D.** Keep silent. Rely on operator memory once they've experienced the difference once.

Designer-preference signal: A is clean (single copy change, lands FB-089's label idiom). B is noisy. C is the minimum discoverability win. D is the no-op outcome.

Hard constraint: no option should change either handler's behavior (Tab stays cache-restore; quick-jump keys stay fresh-fetch — both by design, FB-041/FB-042).

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable]** For Option A: S1 Tab hint contains the designer-ratified freshness signal (e.g., `"cached"`). Test: render with `forceDashboard=true AND typeName="backends"`; assert substring.
2. **[Observable]** For Option A: S4 quick-jump entries do NOT render a freshness suffix (to avoid noise). Test: render S4 with a cached typeName present; assert no "(fresh)"/"(cached)" suffix on any S4 entry.
3. **[Observable]** For Option C: help overlay contains the freshness-distinction one-liner. Test: render help overlay; assert substring.
4. **[Anti-regression]** Tab handler behavior unchanged — still no `LoadResourcesCmd` dispatch. Test: FB-054 AC4 test green (Tab does not dispatch `ListResourcesCmd`).
5. **[Anti-regression]** Quick-jump handler behavior unchanged — still dispatches `LoadResourcesCmd`. Test: existing FB-042 quick-jump tests green.
6. **[Anti-regression]** FB-089 copy cohesion compatible — if FB-089 ships first with Option A, FB-091's S1 copy must layer onto FB-089's idiom without contradiction. Test: combined render matches both briefs' ACs.
7. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-054 ACCEPTED. **Coordinates with FB-089** — if FB-089 ships first, this brief extends FB-089's copy with a freshness signal. If FB-091 ships first (unlikely — FB-089 is P2), FB-089 must preserve the freshness signal.

**Maps to:** REQ-TUI-008 (discoverable disambiguation of affordances with different semantics).

**Non-goals:**
- Not changing which keys are cached vs fresh (both semantics are by design).
- Not adding an "always refetch" modifier to Tab — Tab is explicitly the cache-restore path.
- Not removing S4 quick-jump keys when a table is cached — the fresh-fetch path remains available.

---

### Triage record — FB-054 user-persona evaluation (2026-04-20)

| Persona finding (persona's label) | Disposition | Rationale |
|---|---|---|
| FB-054 P2-1 (S6 keybind strip and S1 Tab hint describe the same key with contradictory mental models — cached-restore vs pane-cycle) | **NEW BRIEF FB-089 P2** (bundled with P3-1) | Verified: `resourcetable.go:514` (S1 verb-phrase) vs line 607 (S6 label). Shared thesis with P3-1 (Tab hint copy self-consistency across surfaces/widths); folding reduces designer/engineer churn. |
| FB-054 P3-1 (short-form fallback drops "to", creating a style switch at resize crossover) | **FOLDED INTO FB-089 P2** | Verified: line 514 "to resume" vs line 520/529 "resume". Same Tab hint copy path; single designer decision resolves both surfaces and all widths coherently. Anti-pattern to split. |
| FB-054 P3-2 (typeName is raw registration name; S4 uses display label → inconsistent vocabulary on one screen) | **NEW BRIEF FB-090 P3** | Verified: S1 uses `m.typeName` at line 514; S4 uses `e.label` from `quickJumpTable` at line 367. Distinct thesis (label-source resolver, not copy cohesion) → separate brief. |
| FB-054 P3-3 (Tab cached restore and S4 quick-jump fresh fetch — duplicate paths to same type, no copy distinction) | **NEW BRIEF FB-091 P3** | Verified: Tab handler at `model.go:1035` — no `LoadResourcesCmd`; quick-jump handler at `model.go:1023` — dispatches `LoadResourcesCmd`. Distinct thesis (semantic disambiguation of two correct-but-different behaviors) → separate brief. |

Net new briefs: **3** (FB-089 P2, FB-090 P3, FB-091 P3). Folds into existing briefs: **0**. Dismissals: **0**. All four findings verified against code before filing. FB-054 → PERSONA-EVAL-COMPLETE recorded.

**Update 2026-04-20:** FB-091 DISSOLVED — FB-089 Option A's `(cached)` suffix subsumes the semantic-disambiguation gap; ux-designer recommendation accepted.

---

### FB-092 — FB-055 Esc-to-dashboard status-bar hint deduplicates with FB-054 persistent hint + "dashboard" terminology collision

**Status: ACCEPTED 2026-04-20 + PERSONA-EVAL-COMPLETE 2026-04-20** — Option A one-line copy swap shipped at `model.go:1521` (engineer) → `"Returned to welcome panel"`. Trigger unchanged (same Esc-from-NavPane-with-table path; 3s `HintClearCmd` decay unchanged). 6 `TestFB055_*` tests updated with `strings.Fields` normalization to handle lipgloss narrow-width wrapping (new 25-char copy wraps differently than old 38-char copy); normalization is scoped to the assertion, not the model. AC1 expanded inline to cover AC1/AC2/AC3 triplet (new copy present, "dashboard" absent case-insensitive, "Tab to resume" CTA absent). All 9 brief-AC-indexed axes covered: 4 Observable (AC1-AC3 inline + AC4 HintClearMsg), Input-changed (AC4 HintClearMsg clear transition), Anti-behavior (AC5 fresh-startup + AC6 other-keys), Anti-regression (AC7 FB-054 existing suite + AC8 FB-041 chain), Integration (AC9 `go install ./...` + `go test ./internal/tui/...`). Pre-existing failures in `components/` flagged by test-engineer (TestFB093_* tests written ahead of unshipped feature + TestFB089_AC4 regression) are unrelated to FB-092's one-line model.go change — confirmed resolved by FB-093 ACCEPTED on same-day acceptance cycle. User-persona eval complete 2026-04-20 — 6 positive findings + 0 P1/P2/P3. Persona verified: (1) copy ships exactly as specced at `model.go:1523`; (2) complementary signal pair (transient "where you arrived" + S1 persistent "what to do next") reads cleanly — no stutter, no CTA duplication; (3) "dashboard" collision resolved — residual uses at `quotadashboard.go:297` + `resourcetable.go:311` are at their correct surfaces (destination-advertising, not landing-naming); (4) operator-initiated principle preserved (Esc = explicit gesture → explicit acknowledgment); (5) 3s decay appropriate; (6) fresh-startup Esc silence correct (guard fails on `tableTypeName == ""`). Considered-and-dismissed: "Esc during initial load is silent" — pre-existing no-op path predates FB-092, not filed without corroborating operator-report evidence. **Net:** 0 new briefs, 0 dismissals requiring record. FB-092 → ACCEPTED + PERSONA-EVAL-COMPLETE. Spec at `docs/tui-ux-specs/fb-092-esc-hint-dedup.md`.
**Priority: P2** — clarity gap on a high-frequency navigation primitive: the third Esc in the FB-041 chain lands an operator on the welcome panel where two hints (FB-054 persistent + FB-055 transient) both say "Tab to resume" at the same instant, and both refer to the panel as "dashboard" — a word already used by QuotaDashboardPane (`3`) and ActivityDashboardPane (`4`). Risk: operator confusion at the "resume" CTA level and at the pane-identity level on the same keypress.

#### User problem

Verified in code:
- **FB-054 persistent hint** (`resourcetable.go:507–531`) renders on the welcome panel whenever `forceDashboard && typeName != ""`. Copy: `[Tab] to resume <typeName>, or select a different type` (wide) / `[Tab] resume <typeName>` (short-form + narrow). Always visible until the operator navigates away.
- **FB-055 transient hint** (`model.go:1491`, 3s decay via `HintClearCmd`) fires on the NavPane-Esc→welcome transition when `!showDashboard && tableTypeName != ""`. Copy: `Returned to dashboard — Tab to resume`.

Both trigger simultaneously on the NavPane-Esc→welcome transition (which is precisely the FB-055 firing condition). The operator sees:

- **Welcome panel (persistent):** `[Tab] to resume backends, or select a different type`
- **Status bar (transient 3s):** `Returned to dashboard — Tab to resume`

Two failure modes flagged by persona:

1. **Stutter / information repeat.** Both hints carry the same CTA ("Tab to resume") and the same target state. The transient hint adds no discoverability that FB-054 doesn't already provide; it only confirms the transition occurred, which the welcome panel's own appearance already implies. Effect: operator reads both, registers the repetition rather than the signal.

2. **"Dashboard" term collision.** The TUI has a quota dashboard (`3`), an activity dashboard (`4`), and the FB-055 hint also labels the welcome panel a "dashboard." An operator who just navigated through QuotaDashboardPane or ActivityDashboardPane will parse "Returned to dashboard" and have to disambiguate which one. The welcome panel is not labeled "dashboard" anywhere else in the UI — the label appears only in the FB-055 hint.

These two problems are tightly coupled: both are in the same 36-character hint string; any copy change that resolves one must not re-introduce the other. Folding them into one designer decision prevents churn.

#### Proposed interaction

Designer picks one of:

- **A. Transition-only signal, no CTA repetition, no "dashboard".** Replace hint with a pure transition confirmation: e.g., `Returned to welcome panel` or `Back to resource list`. Drops the "Tab to resume" CTA (FB-054 persistent hint carries it) and drops "dashboard" (resolves term collision). Cleanest outcome.
- **B. Drop FB-055 hint entirely.** FB-054 persistent hint is sufficient. The welcome panel appearing IS the transition confirmation. Zero copy, zero redundancy. Removes FB-055 from View() altogether.
- **C. Keep CTA but rename the target.** e.g., `Returned to welcome — Tab to resume <typeName>`. Keeps the CTA (mild repetition with FB-054 but unambiguous target), resolves "dashboard" collision.
- **D. Keep hint as-is.** Accept redundancy and term collision. No-op outcome.

Preference order (product signal): **A > C > B > D.** A preserves the transition signal (operator confidence that the keypress registered) without repeating the CTA and resolves the term collision. B is defensible if designer determines the welcome-panel content itself is already a sufficient transition signal; accept if argued. C keeps the stutter but adds disambiguation. D is not acceptable given P2 severity.

Hard constraints:
- Must not fire on fresh startup (`tableTypeName == ""`) — existing FB-055 AC2 remains load-bearing.
- Must not fire on Enter/Tab/ShiftTab/`3`/`4` that clear `showDashboard` — existing FB-055 AC3 remains load-bearing.
- Must not break FB-054 persistent hint copy — FB-055 hint change is orthogonal to `resourcetable.go:507–531`.
- The word "dashboard" may still appear in QuotaDashboardPane / ActivityDashboardPane title bars (those are legitimate — they ARE dashboards). This brief only prohibits "dashboard" as a label for the welcome panel in the FB-055 hint.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-behavior]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable]** For Options A / C: `stripANSI(model.View())` after the NavPane-Esc→welcome transition contains the designer-ratified copy (e.g., the substring `Returned to welcome` or `Back to resource list`). Test: inject `showDashboard=false, activePane=NavPane, tableTypeName="HTTPProxy"`; send Esc; assert substring.
2. **[Observable]** For Options A / B / C: `stripANSI(statusBar.View())` after the transition does NOT contain the bare token `dashboard` (case-insensitive) in the FB-055 hint window. Test: same setup as AC1; assert absence.
3. **[Anti-behavior]** For Option B: after the transition, `statusBar.View()` is unchanged from the non-transient statusBar baseline (no FB-055 hint token is posted). Test: compare `statusBar.View()` before and after the Esc transition; assert no additional hint text.
4. **[Input-changed]** The decision must not break existing FB-055 transience: after ~3s (or the designer-ratified decay duration — may be changed), the FB-055 hint is removed from View(). Test: advance simulated time by `HintClearCmd`'s delay; assert FB-055-specific substring absent. (Designer may reduce duration per persona P3-3 observation, but must not introduce a permanent hint in the status bar.)
5. **[Anti-behavior]** Existing FB-055 AC2 + AC3 still pass: fresh-startup (`tableTypeName=""`) produces no hint, and Enter/Tab/ShiftTab/`3`/`4` that clear `showDashboard` produce no FB-055 hint. Test: existing `TestFB055_AC2_*` and `TestFB055_AC4_OtherKeys_ClearingDashboard_NoHint` stay green.
6. **[Anti-regression]** FB-054 persistent hint (`[Tab] to resume <typeName>` on welcome panel) unchanged in copy and trigger. Test: existing FB-054 test `TestFB054_*` stays green.
7. **[Anti-regression]** FB-041 Esc-chain behavior unchanged. Existing FB-041 tests stay green.
8. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-054 ACCEPTED, FB-055 ACCEPTED. **Coordinates with FB-089** — if FB-089 ships first (it's in-flight as PENDING ENGINEER with Option A `(cached)` suffix), FB-092 copy must not re-introduce redundancy with FB-089's refined `[Tab] resume <typeName> (cached)` phrasing. Option A is compatible; Option C requires designer to pick wording that does not re-duplicate FB-089's `(cached)` token.

**Maps to:** REQ-TUI-008 (discoverable, non-redundant affordances) + REQ-TUI-013 (visual feedback on state change).

**Non-goals:**
- Not removing FB-054 persistent hint. It is the canonical Tab-resume affordance; FB-055 hint (if kept) must play a secondary confirmation role.
- Not changing FB-055 trigger conditions (still fires only on NavPane-Esc→welcome with cached typeName).
- Not renaming QuotaDashboardPane or ActivityDashboardPane — those are legitimate dashboards.
- Not changing HintClearCmd's duration mechanism (Option B opts out of the hint; A/C/D may tune the duration but not the mechanism).

---

### Triage record — FB-055 user-persona evaluation (2026-04-20)

| Persona finding (persona's label) | Disposition | Rationale |
|---|---|---|
| FB-055 P2-1 (FB-054 persistent hint + FB-055 transient hint fire simultaneously with overlapping "Tab to resume" CTA — stutter effect) | **NEW BRIEF FB-092 P2** (bundled with P2-2) | Verified: `resourcetable.go:507–531` (FB-054 persistent hint trigger `forceDashboard && typeName != ""`) vs `model.go:1491` (FB-055 transient hint trigger `!showDashboard && tableTypeName != ""`) — both fire on the exact NavPane-Esc→welcome transition. Shared thesis with P2-2 (both reside in the same FB-055 hint string; single copy change resolves both). |
| FB-055 P2-2 ("dashboard" term overloaded with QuotaDashboardPane and ActivityDashboardPane — operator confusion at pane-identity level) | **FOLDED INTO FB-092 P2** | Verified: welcome panel is not labeled "dashboard" anywhere else in the UI; only the FB-055 hint uses the term for it. Same copy path as P2-1. Folding is the correct call per shared-thesis rule. |
| FB-055 P3-1 (no matching hint on Tab-from-welcome — asymmetric acknowledgment) | **DISMISSED** | Persona's own framing is a symmetry argument, not a user-problem argument. The resource table replacing the welcome panel on Tab IS a visible state change; no evidence operators are confused about whether Tab worked. Filing as P3 would introduce net-new UI noise on a path that already has a sufficient signal. If operator reports emerge, revisit. |
| FB-055 P3-2 (right-side status-bar placement misses operator gaze which is on left sidebar during Esc) | **DISMISSED (system-wide)** | Persona's own framing: "system-wide glyph policy issue, not isolated to FB-055." Not brief-worthy against a single feature. If a pattern of missed status-bar hints emerges across features, file a cross-cutting placement brief at that time. |
| FB-055 P3-3 (3-second duration overshoots given FB-054 persistent affordance) | **DISMISSED (system-wide)** | Duration is controlled by `HintClearCmd(token, 3*time.Second)` at `model.go:799–801` — a shared constant across all status-bar hints. Tuning it for FB-055 alone would fragment the hint-decay contract. If FB-092 lands Option A/B and operators still perceive the remaining hint as too long, file a cross-cutting hint-duration brief. Designer may optionally tune duration per AC4 within FB-092. |
| FB-055 P3-4 (⚡ glyph signals urgency for benign navigation event) | **DISMISSED (system-wide)** | Persona's own framing: "system-wide glyph policy issue, not isolated to FB-055." Glyph is used for all status-bar hints. Not brief-worthy against a single feature. If a pattern emerges across hints with different severity semantics, file a cross-cutting glyph-policy brief. |

Net new briefs: **1** (FB-092 P2). Folds into existing briefs: **0**. Dismissals: **4** (1 speculative polish + 3 system-wide, all with rationale and revisit conditions documented). All findings verified against code or persona's own framing before triage. FB-055 → PERSONA-EVAL-COMPLETE recorded.

---

### Triage record — FB-056 user-persona evaluation (2026-04-20)

Findings delivered: 0 P1, 2 P2, 2 P3 — 4 total. All P2 claims verified against `resourcetable.go:590–644`; P3 claims verified against `statusbar.go:70–84` + `helpoverlay.go:35–58`.

| Persona finding (persona's label) | Disposition | Rationale |
|---|---|---|
| FB-056 P2-1 (`3 quota`/`4 activity` hints disappear in table context — moment of highest relevance for cross-context keys) | **NEW BRIEF FB-093 P2** (bundled with P2-2) | Verified: `renderKeybindStrip()` at `resourcetable.go:606` has mutually-exclusive dashboard/table branches; dashboard branch (lines 608–617) includes `pair("3", "quota")`+`pair("4", "activity")`, table branch (lines 620–629) omits them. User's claim is factually correct — cross-context keys vanish when the operator is most likely to need them (busy table → check quota/activity). |
| FB-056 P2-2 (`Tab next pane` label in dashboard strip contradicts FB-054's persistent `[Tab] to resume <typeName>` band hint on the same screen) | **FOLDED INTO FB-093 P2** | Verified: dashboard branch (line 610) and table branch (line 622) both use `pair("Tab", "next pane")`. When `forceDashboard && typeName != ""`, S1 band shows `[Tab] to resume X` while the strip below says `Tab next pane` — same screen, same key, two mental models. Same fix site (`renderKeybindStrip()`) and same thesis (strip copy cohesion with cross-context mental model) → bundle. |
| FB-056 P3-1 (`Enter select` uses table-row language in dashboard context where Enter loads from sidebar) | **DISMISSED** | On dashboard, `Enter` acts on the selected sidebar item to dispatch `LoadResourcesCmd`. "Select" is contextually accurate — the user selects an item (from sidebar) and Enter commits the selection. Not a copy bug; the same verb legitimately covers sidebar-item-select and table-row-select across contexts. Filing would introduce copy churn without a user-problem signal. Revisit if operator reports emerge. |
| FB-056 P3-2 (`[3]` destination is labeled three different ways across keybind strip, help overlay, and quota-table CTA — `quota` vs `quota (toggle)` vs `full dashboard`) | **DISMISSED (contextual purpose)** | Verified: strip `3 quota` (compact), help `[3]  quota (toggle)` (behavior note), S3 CTA `(press [3] for full dashboard)` (mini-widget vs full-dashboard qualifier). Each label serves a distinct contextual purpose — strip is a memory prompt, help extends with behavior semantics, CTA contrasts against the mini-widget it's attached to. Unifying would lose the S3 mini-vs-full distinction. If vocabulary cohesion becomes a broader theme (beyond `[3]` alone), file a cross-cutting copy-cohesion brief. FB-092's "dashboard" → "welcome panel" pivot already addresses the most-confusing term-collision (QuotaDashboardPane / ActivityDashboardPane / welcome-dashboard); remaining `quota`/`full dashboard` delta is net-benign. |

Net new briefs: **1** (FB-093 P2 bundling P2-1 + P2-2). Folds into existing briefs: **0**. Dismissals: **2** (both P3 with rationale and revisit conditions documented). All findings verified against code before triage. FB-056 → PERSONA-EVAL-COMPLETE recorded.

---

### FB-093 — Dashboard keybind strip: cross-context keys persist in table context + Tab label aligns with FB-054 resume framing

**Status: ACCEPTED 2026-04-20 + PERSONA-EVAL-COMPLETE 2026-04-20** — Option A + Option B single-function rewrite of `renderKeybindStrip()` shipped. Table branch gains `pair("3", "quota")` + `pair("4", "activity")` between `/ filter` and `c ctx`; dashboard branch computes `hasCachedTable := m.forceDashboard && m.typeName != ""` and conditionally appends Tab only when `!hasCachedTable`. Engineer delivered 6 FB-093 tests (AC1–AC6) using `renderKeybindStrip()` directly to avoid false-matching `[Tab]` from the S1 band; AC4 uses `x delete` absence as the table-branch distinguishing marker. Test-engineer gate-check verified all 10 brief ACs mapped with Observable (AC1/AC2/AC3), Input-changed (AC4 dashboard→table + AC5 cache-state transitions), Anti-behavior (AC6 bareParts narrow fallback at contentW=20), Anti-regression (AC7 FB-054 + AC8 FB-056 + AC9 FB-041) verified by running the full anti-regression suites directly — hand-waves AC7-AC9 converted to executed confirmations, clean gate discipline. `go install ./...` + full suite green. **Cross-feature test update** `TestFB089_AC4_AntiRegression_S6Strip_TabNextPaneUnchanged` preserved FB-089's semantic invariant (cached-state Tab affordance exists in-view) while migrating the assertion site (S6 → S1) — proactive submission-message flag by engineer, reviewed per `feedback_scope_creep_bonus_bugs` "necessary consequence" pattern, accepted. **Pre-existing failures resolved:** TestFB093_AC2/AC4/AC5 + TestFB089_AC4 (all previously failing per FB-092 submission) now all PASS — confirms FB-093 is the feature those tests were ahead of; gate-check closure verifies the pre-existing-failure concurrence. User-persona dispatch pending per cadence rule. Spec at `docs/tui-ux-specs/fb-093-dashboard-strip-cohesion.md`.
**Original Status: PENDING UX-DESIGNER** — filed 2026-04-20 by product-experience from FB-056 user-persona P2-1 + P2-2 bundled.
**Priority: P2** — discoverability gap (P2-1) + copy self-contradiction (P2-2) on the most-frequented strip on screen: the welcome/table keybind reference row. Both defects live in `renderKeybindStrip()` and share the thesis that the strip must reflect the cross-context mental model already established by FB-041 (Esc chain) + FB-054 (Tab resume) + FB-056 (`[3]`/`[4]` hints in NAV_DASHBOARD).

#### User problem

**Sub-problem 1 (P2-1): `[3]` / `[4]` cross-context keys vanish when the operator is most likely to need them.**

`renderKeybindStrip()` at `resourcetable.go:606` splits into two mutually-exclusive branches:

- **Dashboard branch** (lines 608–617, `forceDashboard || typeName == ""`): shows `3 quota`, `4 activity`.
- **Table branch** (lines 620–629, else): omits `3`/`4`, adds `x delete` + `/ filter` instead.

The cross-context keys `3` and `4` work identically in both contexts — they're platform-wide (documented in FB-035 + FB-047 + FB-049 + FB-050). But the operator only sees them on the welcome panel, where the need is lowest. When a table is loaded and the operator sees concerning signal (a bucket near quota, a recent error), they need to quickly inspect quota or activity — at exactly that moment, the reference row stops reminding them those keys exist.

Persona framing (verbatim): *"The most actionable time to consult quota or activity is while viewing a busy table — but that's exactly when the reference is gone."*

**Sub-problem 2 (P2-2): `Tab next pane` strip label contradicts FB-054's `[Tab] to resume <typeName>` band hint on the same screen.**

When `forceDashboard && typeName != ""` (operator has Esc'd back to the welcome panel with a cached table available), FB-054 adds a persistent S1 header-band hint: `[Tab] to resume <typeName>`. Immediately below, the keybind strip renders `Tab next pane` (dashboard branch, line 610). Two hints, same key, opposite mental models: "resume your cached work" vs "cycle focus to the next pane."

The strip's `Tab next pane` copy is also slightly misleading in the table context (line 622) — from TablePane, Tab goes back to NavPane; from NavPane, Tab goes to TablePane. "Next pane" is a reasonable reduction, but it's the *general* verb that masks the *specific* cache-restore semantics FB-041 + FB-054 spent two briefs establishing.

#### Proposed interaction

**Sub-problem 1 fix surface:** the table branch at `resourcetable.go:620–629` currently renders `{j/k move, Tab next pane, Enter select, x delete, / filter, c ctx, ? help, q quit}`. Designer decides whether to:

- **Option A:** append `3 quota` + `4 activity` to the table branch (strip becomes 10 items).
- **Option B:** swap out a less-critical item (e.g., `c ctx` or `q quit` — still discoverable via help) to keep the strip at 8 items.
- **Option C:** three-row strip where cross-context keys are on their own row (more visual surface; only feasible if vertical space affords it).

Preference: whichever keeps the strip from breaking the narrow-width `bareParts` fallback at line 644.

**Sub-problem 2 fix surface:** the dashboard branch at `resourcetable.go:608–617` currently shows `Tab next pane`. When FB-054's S1 band is also visible (`forceDashboard && typeName != ""`), the strip Tab label should read something compatible with `resume` framing. Options:

- **Option A:** change strip Tab label to `resume` when `typeName != ""`, fall back to `next pane` when `typeName == ""`. Matches S1 band exactly.
- **Option B:** drop the Tab entry from the dashboard strip entirely when S1 band shows the resume hint (no redundancy; the band owns the Tab copy).
- **Option C:** unify under one neutral verb across both branches (e.g., `Tab focus` or `Tab pane`) — weakens FB-054's resume thesis; not preferred.

Preference: Option A or B. Designer picks.

Both sub-problems touch `renderKeybindStrip()` exclusively — one bundled ux-designer spec + one bundled engineer impl PR is the natural unit.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable]** With `typeName != "" && !forceDashboard && activePane=TablePane` (live table context), `stripANSI(welcomePanel.View())` (or the table's rendered strip, wherever `renderKeybindStrip` fires) contains `"3"` and `"quota"` and contains `"4"` and `"activity"`. Test: set state; assert substring presence on the strip render.
2. **[Observable]** With `forceDashboard=true && typeName != ""` (FB-041 welcome after Esc with cache), `stripANSI(welcomePanel.View())` strip region does NOT render a Tab label that contradicts S1 band's `[Tab] to resume <typeName>`. Specifically: either contains `"resume"` on the strip Tab entry, or the strip omits the Tab entry entirely. (Designer's Option A vs B disambiguates; AC asserts the chosen option.)
3. **[Observable]** With `forceDashboard=true && typeName == ""` (fresh startup, no cache), the Tab label reverts to `next pane` (or whatever the designer's fallback is). Test: transition state; assert substring.
4. **[Input-changed]** When `activePane` transitions from NavPane to TablePane (operator presses Enter to load), the strip render transitions from omitting `3`/`4` to including `3`/`4` (because sub-problem 1 fix preserves them). Test: capture two View() renders, assert both contain the new substrings.
5. **[Input-changed]** When `typeName` transitions from `""` to a real value (operator selects a resource type from sidebar and a table loads), if `forceDashboard` is also true, Tab label transitions from fallback copy (`next pane` or similar) to the resume-compatible copy. Test: two View() captures, substring diff.
6. **[Anti-behavior]** On the extreme narrow-width fallback (`bareParts` at line 644), the strip continues to render the bare labels only — the sub-problem 1 fix must NOT break the narrow fallback. Test: render at width <40, assert `bareParts` joined string is returned.
7. **[Anti-regression]** FB-054's S1 band copy unchanged. Existing FB-054 tests green.
8. **[Anti-regression]** FB-056's existing assertions (NAV_DASHBOARD statusBar `[3]`/`[4]` hints + welcomePanel strip no `x delete`/`/ filter` when `showDashboard=true`) unchanged. Existing FB-056 tests green.
9. **[Anti-regression]** FB-041 Esc chain unaffected. Existing FB-041 tests green.
10. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/... -count=1` green.

**Dependencies:** FB-054 ACCEPTED (S1 band copy), FB-056 ACCEPTED (current strip branching). Coordinates with FB-092 (`"Returned to welcome panel"` hint, unrelated surface).

**Maps to:** REQ-TUI-008 (keybind discoverability) — context-aware presentation that preserves cross-context keys where they matter.

**Non-goals:**
- Not changing which keys are live. Only which are *displayed* and what label they carry.
- Not auditing the narrow-width `bareParts` for `3`/`4` inclusion — bare fallback is a space-constrained last resort.
- Not extending to QuotaDashboardPane / ActivityDashboardPane strips (they have their own status bars with FB-049/FB-078 governance).
- Not touching statusBar's `NAV_DASHBOARD` hint variant (FB-056 already addressed).

---

### FB-100 — `activityFetchFailed` clobbers valid stale rows on `[r]` refresh failure

**Status: ACCEPTED 2026-04-20 (retroactive closeout — bundled in `74fcd0a`)** — test-engineer independent gate PASS: 4/4 `TestFB100_*` tests (`model_test.go:10707`) green, all use `stripANSIModel(appM.View())` + `strings.Contains` (zero model-field assertions). AC3 Input-changed three-check pattern: `view1 != view2`, `view1` contains actor email, `view2` contains "activity unavailable" — substantive distinguishing assertion. Anti-regression anchors: `TestFB082_AC3_ProjectActivityErrorMsg_ShowsUnavailable`, `TestFB082_ErrorRecovery_SetActivityRows_ClearsFailedFlag`, `TestFB076_*` all green. Full suite green (tui 15.899s, components 0.517s). **Bundle-scope-creep note:** Implementation (gate at `model.go:494` + `ActivityRowCount()` getter) landed in bootstrap commit `74fcd0a`; not mentioned in bundle message. Retroactive closeout applied. No new commit. Persona-eval queued in batch with FB-072/101/104/099. **Prior Status: PENDING TEST-ENGINEER 2026-04-20** — engineer delivered rows-empty gate on `SetActivityFetchFailed(true)`. New `ActivityRowCount() int` getter on `ResourceTableModel` (~line 922) returns `len(m.activityRows)`. `ProjectActivityErrorMsg` handler at `model.go:~473` now wraps the fetch-failed flag in `if m.table.ActivityRowCount() == 0 { m.table.SetActivityFetchFailed(true) }` — stale rows preserved on transient refresh error. 4 new `TestFB100_*` tests: AC1 Observable (populated rows + error → `stripANSIModel(View())` contains fixture actor email `"alice@datum.net"`, NOT `"activity unavailable"`); AC2 Observable (empty rows + error → `"activity unavailable"` — FB-082 unchanged); AC3 Input-changed (same error msg, `ActivityRowCount() > 0` vs `== 0` → different View() output); AC4 Anti-behavior (stale rows retained post-error, subsequent `ProjectActivityLoadedMsg` renders new rows via existing auto-reset — no stuck flag). AC5 named anti-regression: `TestFB082_AC3_ProjectActivityErrorMsg_ShowsUnavailable` (empty-rows path) + `TestFB082_ErrorRecovery_SetActivityRows_ClearsFailedFlag` (component test) both green. AC6: `TestFB076_*` `[r]` dispatch tests green. AC7 Integration: `go test -c -o /dev/null ./internal/tui/...` clean, full suite + `go install` green. **Original Status: PENDING ENGINEER** — engineer-direct. Filed 2026-04-20 by product-experience from FB-082 user-persona P2-1.

**Priority: P2** — operator-visible data loss surface: pressing `[r]` to refresh populated activity can wipe the visible rows and replace them with a permanent-feeling "activity unavailable" on a transient network blip. "Refresh shouldn't lose data I already had" is a strong mental-model expectation.

#### User problem

FB-082 added `SetActivityFetchFailed(true)` on `ProjectActivityErrorMsg` (`model.go:470–471`). The switch case at `resourcetable.go:304` routes `activityFetchFailed` ahead of `default:` (rows branch), so a populated S3 teaser snaps to `"activity unavailable"` when an error fires even though `activityRows` is still in memory and was current seconds ago.

Init path + context-switch path don't see this: both clear/nil rows before dispatch. The `[r]` refresh path (FB-076, model.go:1262–1264) intentionally keeps rows populated across the re-fetch so stale data stays visible — but when `ProjectActivityErrorMsg` fires after a failed `[r]` refresh, those stale rows are overridden by the error surface.

From the operator's perspective: they pressed `[r]` expecting fresher data, the transient error fired, and their existing activity data disappeared. They have no signal that the rows are still in memory and retrieveable — `"activity unavailable"` reads as "the data is gone."

#### Proposed fix

In `ProjectActivityErrorMsg` handler (`model.go:462–471`), gate `SetActivityFetchFailed(true)` on the rows-empty predicate:

```go
case data.ProjectActivityErrorMsg:
    m.table.SetActivityLoading(false)
    if m.table.ActivityRowCount() == 0 {
        m.table.SetActivityFetchFailed(true)
    }
    // else: keep stale rows visible; error is transient and data exists
```

Requires adding `ActivityRowCount() int` getter to `ResourceTableModel` if not present (check for existing field-access pattern first).

Optional refinement: on the rows-populated branch, post a transient status-bar hint like `"Activity refresh failed — showing cached data"` via `postHint` to acknowledge the failure without clobbering the teaser. Deferred as a potential FB-102 coordination if the designer-call concludes retry-affordance coupling.

#### Acceptance criteria

| # | Axis | Criterion |
|---|------|-----------|
| AC1 | Observable | Populated activity rows + `ProjectActivityErrorMsg` fires → `stripANSIModel(appM.View())` still contains a row-identifying substring (e.g. an actor email from fixture); does NOT contain `"activity unavailable"`. |
| AC2 | Observable | Empty activity rows + `ProjectActivityErrorMsg` fires → `stripANSIModel(appM.View())` contains `"activity unavailable"` (unchanged FB-082 behavior for the never-loaded case). |
| AC3 | Input-changed | Same `ProjectActivityErrorMsg`, different `ActivityRowCount()` pre-state → different View() output (AC1 vs AC2 pair). |
| AC4 | Anti-behavior | After AC1 (stale rows retained after error), a subsequent successful `SetActivityRows(newRows)` clears `activityFetchFailed` (via the existing auto-reset) and renders the new rows. No stuck-flag regression. |
| AC5 | Anti-regression | FB-082 tests green: `TestFB082_AC3_ProjectActivityErrorMsg_ShowsUnavailable` (empty-rows path) still passes; `TestFB082_ErrorRecovery_SetActivityRows_ClearsFailedFlag` still passes. |
| AC6 | Anti-regression | FB-076 `[r]` dispatch tests green. |
| AC7 | Integration | `go install ./...` compiles; `go test ./internal/tui/...` green. |

#### Non-goals

- Not adding a retry-affordance hint — FB-102 owns that (designer-call).
- Not changing the spinner / loading behavior on `[r]` — FB-103 owns empty-rows refresh feedback.
- Not touching the CRD-absent path — that genuinely IS permanently unavailable.

---

### FB-101 — Tier 3 activity width math latent panic at `contentW ≤ 22`

**Status: ACCEPTED 2026-04-20 (retroactive closeout — bundled in 74fcd0a)** — test-engineer batch gate-check PASS. 3 `TestFB101_*` tests verified in-place: AC1 `TestFB101_AC1_RenderActivitySection_ContentW22_NoPanic` uses `defer recover()` guard asserting zero panic on contentW=22; AC2 `TestFB101_AC2_RenderActivitySection_ContentW0_NoPanic` same pattern for contentW=0; AC3 `TestFB101_AC3_ContentW44_TruncationUnchanged` uses `stripANSI(m.renderActivitySection(44))` to verify Tier 3 truncation path unchanged (engineer judgment call on contentW=44 rather than brief-suggested contentW=50 documented and validated: Tier 3 gate is `contentW < 45`, so contentW=44 exercises the correct branch; contentW=50 would fall into Tier 2 and not reach the modified code). AC4 anti-regression covered by `TestFB082_WidthBand_Tier3_*` tests green. AC5 Integration confirmed green. All Observable ACs use `stripANSI` per `feedback_observable_acs_assert_view_output`. Code landed in bootstrap bundle `74fcd0a` at `resourcetable.go:~350` as one-line `max(1, min(16, contentW-22))` defensive fix. **Prior Status: PENDING TEST-ENGINEER 2026-04-20** — engineer delivered one-line defensive fix. **Original Status: PENDING ENGINEER** — engineer-direct hygiene fix. Filed 2026-04-20 by product-experience from FB-082 user-persona P3-1.

**Priority: P3** — defensive hygiene; dead code today because S3 is gated at `contentW >= 50` (`resourcetable.go:171,174`) and `renderActivitySection` never runs below that. But if the gate relaxes (future brief showing S3 at narrower widths, or component reuse in a different panel), the latent panic surfaces on the first too-narrow render. Two-line defensive fix is cheap insurance against a gate change that could happen in any subsequent feature.

#### User problem

At `resourcetable.go:322–324` (Tier 3, `contentW < 45`):

```go
actorW := min(16, contentW-22)
if len(actorRunes) > actorW {
    actor = string(actorRunes[:actorW-1]) + "…"
}
```

At `contentW = 22`: `actorW = 0`, then `actorRunes[:actorW-1]` = `actorRunes[:-1]` → panic.
At `contentW < 22`: `actorW` is negative, same panic on the slice.

S3's outer gate at `resourcetable.go:171,174` prevents Tier 3 from being reached at any runtime width. But relying on a distant caller's gate to prevent an index panic in local math is brittle — any future relaxation of the gate silently re-introduces the panic without a visible symptom in the local function.

#### Proposed fix

```go
actorW := max(1, min(16, contentW-22))
if len(actorRunes) > actorW {
    actor = string(actorRunes[:actorW-1]) + "…"  // actorW >= 1 → actorW-1 >= 0 → safe
}
```

Single-line change. Preserves existing behavior at all currently-reachable widths; adds safety at widths that are currently dead.

If the team wants stronger defense: early-return a placeholder (`"—"` or similar) when `contentW < 24` or `actorW < 2`, rather than attempting truncation. This is a design decision beyond minimal-fix scope — defer to follow-up if desired.

#### Acceptance criteria

| # | Axis | Criterion |
|---|------|-----------|
| AC1 | Observable | `renderActivitySection` called directly with `contentW = 22` does NOT panic. Assertion: recover from panic in test, require no panic raised. |
| AC2 | Observable | `renderActivitySection` called with `contentW = 0` does NOT panic. |
| AC3 | Anti-behavior | At `contentW = 50` (just above the S3 gate), actor truncation behavior unchanged from pre-fix. `stripANSI(section)` contains the truncated-actor glyph `"…"` where expected. |
| AC4 | Anti-regression | FB-082 Tier 3 tests (`TestFB082_WidthBand_Tier3_*`) green. |
| AC5 | Integration | `go install ./...` compiles; `go test ./internal/tui/...` green. |

#### Non-goals

- Not loosening the S3 outer gate. Tier 3 remains dead code until a follow-up brief explicitly surfaces S3 at narrower widths.
- Not adding a placeholder-replacement behavior (`"—"` at very narrow widths) — minimum-fix scope.
- Not auditing other `contentW-N` patterns across `resourcetable.go` — filed separately if a systemic pattern emerges.

---

### FB-102 — "activity unavailable" has no recovery affordance when `!isCRDAbsent`

**Status: ACCEPTED 2026-04-20** — Option D (compact parenthetical, transient-only) shipped via commit `b476a23` on `feat/console`. Implementation: `resourcetable.go` adds `activityCRDAbsent bool` field (line 93), `SetActivityCRDAbsent(bool)` setter (line 942), auto-clear in `SetActivityRows()` (line 929); body conditional at `renderActivitySection()` lines 336-342 renders `"activity unavailable"` + muted `" (press "` + accent-bold `[r]` + muted `")"` when `!activityCRDAbsent`, else clean `"activity unavailable"`. `model.go:501` wires `m.table.SetActivityCRDAbsent(isCRDAbsent)` alongside `SetActivityFetchFailed(true)`. Tests: 5 ACs covering Observable (transient hint visible), Observable (CRD-absent no hint), Input-changed (rows-arrival clears), Anti-behavior (normal state no hint), Anti-regression (FB-082 CRD-absent path unchanged). Submitter-produced axis-coverage table confirmed before acceptance; `go test ./internal/tui/...` exit-0 verified. Originally filed 2026-04-20 by product-experience from FB-082 user-persona P3-2. Spec delivered 2026-04-20 by ux-designer at `docs/tui-ux-specs/fb-102-activity-unavailable-recovery-affordance.md`.

**Priority: P3** — cancel-on-transient-error UX refinement; FB-082 shipped the state machine + copy, but operators who hit a transient error have no signal that pressing `[r]` recovers them. `"activity unavailable"` reads as permanent. This is a refinement, not a blocker — operators can figure out to press `[r]` by convention. Mild severity elevates in combination with FB-100 (if stale-rows case gets a transient hint) since the unified recovery story matters.

#### User problem

At `resourcetable.go:304–305`, `activityFetchFailed=true` renders a single-line `"activity unavailable"` in muted style. Two sub-cases merge into this one surface:

1. **Transient error** (API timeout, network blip, rate-limit): recoverable by `[r]` retry. The operator needs to know this is retryable.
2. **CRD-absent** (`isCRDAbsent=true`): genuinely permanent — retrying won't help. A retry hint here would mislead.

FB-082's copy serves sub-case 2 correctly (terse statement of unavailability). Sub-case 1 is under-served — the operator has no on-screen affordance indicating recoverability.

#### Designer-call options

| Option | Description | Trade-off |
|--------|-------------|-----------|
| A | Append `"[r] to retry"` inline on transient-only | Direct, discoverable; adds width pressure on narrow terminals. |
| B | Second-line muted hint `"[r] to retry"` below the unavailable label | Two-line footprint; clear separation of state + action; may conflict with tight S3 height budget. |
| C | Status-bar hint `"Activity unavailable — [r] to retry"` on transient, no inline change | Keeps S3 clean; discoverability depends on operator noticing status-bar; may collide with other hints (FB-079/FB-097 quota hints). |
| D | Teaser-line variant: change transient copy to `"activity unavailable (press [r])"` | Compact; parenthetical is terminal-conventional; still distinguishes from CRD-absent by whether the parenthetical appears. |
| E | Dismiss | FB-082 shipped without this affordance and no operator has reported confusion yet; wait for evidence. |

**Designer ask:**

- Gate any recovery hint on `!isCRDAbsent`; CRD-absent keeps the current clean copy.
- Coordinate with **FB-100** — if FB-100 ships and stale-rows retain visibility on refresh failure, consider whether a status-bar hint like `"Activity refresh failed — showing cached data, [r] to retry"` serves both surfaces jointly.
- Consider terminal width: 3-tier truncation contract (FB-082) caps at `contentW < 45`; inline options (A, D) need to survive Tier 2/3. Option C avoids this.
- Specify whether the hint persists or decays (`postHint` 3s vs `PostHint` persistent).

Output a spec file at `docs/tui-ux-specs/fb-102-activity-unavailable-recovery-affordance.md` with chosen option + ACs (Observable, Input-changed if transient→hint-visible→recovered elides hint, Anti-behavior no-hint when `isCRDAbsent`, Anti-regression FB-082 CRD-absent path unchanged, Integration).

#### Non-goals

- Not changing `isCRDAbsent` detection logic (that's FB-082's scope).
- Not bundling with FB-100 unless Option C (status-bar hint) is selected and a joint surface is appropriate.

---

### FB-103 — `[r]` refresh on empty activity rows doesn't set `activityLoading=true` → no spinner feedback

**Status: ACCEPTED 2026-04-20 + PERSONA-EVAL-COMPLETE 2026-04-20** — one-site source change at `model.go:1317-1320` (guard `if m.table.ActivityRowCount() == 0 { m.table.SetActivityLoading(true) }` inside existing project-scope branch before dispatching `LoadRecentProjectActivityCmd`). 9 tests with View()-substring discipline: AC1 empty-rows → spinner; AC2 org-scope → no spinner (ActiveCtx=nil), "no recent activity" preserved; AC3 populated-rows → stale rows preserved, no spinner (FB-076 silent-refetch guard); AC4 input-changed empty→loading; AC5 FB-076 dispatch preserved; AC6 FB-082 error-re-render; AC7 error-state → spinner (`"activity unavailable"` absent, covers FB-102 P2-1); AC8 error→loading input-changed; AC9 CRD-absent → spinner (covers FB-102 P3-2). AC10 Integration: `go install ./...` exit 0, `go test ./internal/tui/...` all packages `ok`. Submitter-produced axis-coverage table confirmed pre-review; all 9 test functions exist with state-transition assertion patterns. **Persona eval** (2026-04-20): all 3 target scenarios (empty/never-loaded, transient error, CRD-absent) confirmed spinner fires on `[r]` press; populated-rows silent-refetch (FB-076) preserved; CRD-absent brief spinner-flash reads as informative "keypress received, fetch attempted, still unavailable". 0 P1/P2, 1 P3-1 filed as **FB-130** (render gate at `resourcetable.go:333` uses `activityRows == nil`; misses empty-but-non-nil case — project that genuinely has no activity shows frozen `"no recent activity"` teaser during `[r]` round-trip despite FB-103's state gate firing). Shipped in commit `4c9dab1` on `feat/console`. **Prior Status: PENDING ENGINEER** — engineer-direct. Filed 2026-04-20 by product-experience from FB-082 user-persona P3-3. **Priority elevated P3 → P2** on 2026-04-20 from FB-102 user-persona P2-1: same user-problem re-observed from the error-state path (`[r]` on `"activity unavailable (press [r])"` surface also produces no in-flight signal). Because FB-082 clears `activityRows` on fetch-failure, the error-state surface is a strict subset of "empty-rows" — the fix covers it via the body switch ordering (`activityLoading && rows == nil` before `activityFetchFailed`).

**Priority: P2** — refresh-feedback gap; FB-076 (ACCEPTED 2026-04-19) added the dispatch but not the visual feedback. Operators who press `[r]` with an empty activity teaser see no change on S3 and can't tell whether the refresh is in flight. Once a row arrives the teaser updates; until then S3 silently sits on `"no recent activity"` (or `"activity unavailable (press [r])"` on the error path). Persona-observed on both surfaces; on a slow API (2-5s) operators press `[r]` twice or assume key-capture failure.

#### User problem

At `model.go:1260–1264` (`[r]` refresh handler, NavPane path, populated by FB-076):

```go
case NavPane:
    m.loadState = data.LoadStateLoading
    cmds := []tea.Cmd{data.LoadResourceTypesCmd(m.ctx, m.rc)}
    if m.ac != nil && m.tuiCtx.ProjectID != "" {
        cmds = append(cmds, data.LoadRecentProjectActivityCmd(m.ctx, m.ac, 24*time.Hour, 10))
    }
```

When the activity teaser is empty (e.g., first-ever load returned no rows, or stale error state with no fallback), the operator presses `[r]` to refresh. `LoadRecentProjectActivityCmd` dispatches, but `SetActivityLoading(true)` is not called. S3 keeps rendering `"no recent activity"` until the fetch returns.

When the teaser has existing rows, this is acceptable (stale rows stay visible — silent re-fetch is intentional). But when rows are empty, the operator has no confirmation that `[r]` did anything activity-relevant.

#### Proposed fix

Add `SetActivityLoading(true)` inside the existing gate when rows are absent:

```go
case NavPane:
    m.loadState = data.LoadStateLoading
    cmds := []tea.Cmd{data.LoadResourceTypesCmd(m.ctx, m.rc)}
    if m.ac != nil && m.tuiCtx.ProjectID != "" {
        if m.table.ActivityRowCount() == 0 {
            m.table.SetActivityLoading(true)
        }
        cmds = append(cmds, data.LoadRecentProjectActivityCmd(m.ctx, m.ac, 24*time.Hour, 10))
    }
```

Rationale for rows-empty-only gate: matches FB-082's design choice at context-switch (clear rows → set loading). Rows-populated case intentionally preserves stale rows; adding SetActivityLoading there would blank them to spinner, regressing the silent re-fetch pattern.

Future refinement path (not scope): if the team wants the rows-populated case to also surface a "refreshing" indicator without losing stale rows, introduce `SetActivityRefreshing(true)` analogous to FB-063's SetRefreshing pattern for QuotaDashboard. Deferred pending operator signal.

#### Acceptance criteria

| # | Axis | Criterion |
|---|------|-----------|
| AC1 | Observable | Empty-rows state + NavPane `[r]` press + project-scope → `stripANSIModel(appM.View())` contains `"⟳ loading…"` (S3 spinner). |
| AC2 | Anti-behavior | Empty-rows state + NavPane `[r]` press + org-scope (no ProjectID) → `SetActivityLoading(true)` NOT called; View() still contains `"no recent activity"`. (Org-scope has no activity fetch to wait for.) |
| AC3 | Anti-behavior | **Populated-rows** state + NavPane `[r]` press + project-scope → `SetActivityLoading(true)` NOT called; View() still contains the prior actor labels (stale rows preserved). This preserves the FB-076 silent re-fetch pattern. |
| AC4 | Input-changed | Empty-rows state before vs after `[r]` press: View() before contains `"no recent activity"`, View() after contains `"⟳ loading…"`. |
| AC5 | Anti-regression | FB-076 tests green: `[r]` still dispatches `LoadRecentProjectActivityCmd` on project scope; org-scope `[r]` still doesn't. |
| AC6 | Anti-regression | FB-082 tests green. `ActivityFetchFailed` path unaffected (post-fetch: if fetch fails again, body re-renders error copy). |
| AC7 | Observable (error→loading) | Error-state (`activityFetchFailed=true`, `activityCRDAbsent=false`, `activityRows=nil`) + NavPane `[r]` press + project-scope → `stripANSIModel(appM.View())` contains `"⟳ loading…"` and does NOT contain `"activity unavailable (press [r])"`. Covers FB-102 persona P2-1: error-surface `[r]` press now shows in-flight signal. |
| AC8 | Input-changed (error→loading) | Error-state before vs after `[r]` press: View() before contains `"activity unavailable (press [r])"`, View() after contains `"⟳ loading…"`. |
| AC9 | Observable (CRD-absent→loading→CRD-absent) | CRD-absent state (`activityFetchFailed=true`, `activityCRDAbsent=true`, `activityRows=nil`) + NavPane `[r]` press → `SetActivityLoading(true)` IS called (rows==0 gate). View() momentarily contains `"⟳ loading…"`; post-fetch, View() returns to `"activity unavailable"` (no parenthetical, CRD-absent retained). This is the incidental keystroke-received signal covering FB-102 persona P3-2 — the spinner flash confirms the keypress even though the retry permanently fails. |
| AC10 | Integration | `go install ./...` compiles; `go test ./internal/tui/...` green. |

#### Non-goals

- Not introducing `SetActivityRefreshing` (FB-063 analog) for populated-rows case — deferred until operator evidence warrants.
- Not coupling with FB-100 / FB-102 — distinct user-problem (empty-rows refresh feedback vs stale-rows error policy vs unavailable recovery affordance).
- Not adding a `[r]` confirmation hint in the status bar — implicit via the S3 spinner is sufficient.

---

### FB-104 — Welcome panel: restore hovered resource type documentation

**Status: ACCEPTED 2026-04-20 (retroactive closeout — bundled in 74fcd0a)** — test-engineer batch gate-check PASS. 9 `TestFB104_*` tests verified in-place, all Observable + Input-changed ACs use `stripANSI(m.View())` + `strings.Contains` per `feedback_observable_acs_assert_view_output`. AC1 Kind/Group/Version/Scope line rendering verified via substring assertions on stripped output. AC2 description wrap+truncation verified including overflow case exercising the `wrapDescription` helper. AC3 section absent when `hoveredType.Kind == ""` — asserted via `!strings.Contains(view, <S7 marker>)`. AC4 Input-changed verified with `SetHoveredType(A)` vs `SetHoveredType(B)` state transition, not snapshot-twice. AC5 Input-changed Namespaced vs Cluster label transition verified. AC6 Anti-behavior empty description confirmed no `"…"` artifact + no blank-line gap. AC7 S6 keybind strip position anti-regression via `strings.Index` ordering. AC8 FB-082/083 S3 anti-regression green. AC9 Integration `go install ./...` + full suite green. Code landed in bootstrap bundle `74fcd0a` at `internal/tui/components/resourcetable.go` as new `renderHoveredTypeSection(contentW int)` method + package-level `wrapDescription(desc, contentW)` helper; S7 block appended in `welcomePanel()` between S5 and S6. **Prior Status: PENDING TEST-ENGINEER 2026-04-20** — engineer delivered Option A S7 section implementation.

**Prior Status: PENDING ENGINEER** — spec delivered 2026-04-20 by ux-designer at `docs/tui-ux-specs/fb-104-hovered-type-documentation.md` (Option A: dedicated S7 section between S5 attention and S6 keybind strip, shown only when a type is hovered). No new component infrastructure; single conditional region block + `renderHoveredTypeSection(contentW)` function in `welcomePanel()`. Content: Kind (accent+bold) · Group · Version (secondary) · Scope label (muted) on one line; rule; Description word-wrapped to 2 lines max with `…` truncation; empty description → metadata-only, no truncation artifact. Fields mapped from `rt.Name/Kind/Group/Version/Namespaced→"Namespaced"/"Cluster"/Description`. 9 ACs covering Kind/Group/Scope rendering, description rendering, section absent when nothing hovered, Input-changed (different types, Namespaced vs Cluster), Anti-behavior (empty description), Anti-regression (S6 position, FB-082/083 S3 unaffected), Integration. Filed 2026-04-20 by team-lead from user feedback.

**Priority: P2** — operators lose the ability to understand what a resource type is and why it exists by hovering it. The previous welcome panel showed Kind/Resource/Group/Scope/Description inline; FB-042 replaced that surface with the activity/quota/health dashboard without preserving this capability.

#### User problem

Before FB-042, the welcome panel showed metadata for the sidebar-hovered resource type. After FB-042, `ResourceTableModel.hoveredType` (resourcetable.go:53) is stored via `SetHoveredType()` (line 762–764) but **never rendered**. `ResourceType.Description` (data/types.go:43) is fetched from the OpenAPI v3 schema via `attachDescriptions()` (resourceclient.go:155–204) but no TUI surface displays it.

Operators must guess what e.g. `dnsrecordsets` or `computeclusters` are from the name alone. The description field explicitly exists to fill this gap.

#### Designer-call

Where does hovered-type documentation fit in the welcome panel (resourcetable.go:161–227)?

- **Option A** — Dedicated S7 section at the bottom, shown only when a type is hovered. Kind, Group, Scope, Description (truncated 2 lines). Clean isolation.
- **Option B** — Replace S4 (quick-jump) when a type is hovered; quick-jump returns when nothing is hovered.
- **Option C** — Sidebar tooltip/overlay on hover — persists only while cursor is on the sidebar item.
- **Option D** — Inline in NavSidebar list row (below resource name, muted, 1-line truncation).

**Prefer Option A or C** — Option B sacrifices always-useful quick-jump; Option D requires sidebar layout changes.

#### Key data available
- `m.hoveredType.Kind` — e.g. `"DNSRecordSet"`
- `m.hoveredType.Resource` — e.g. `"dnsrecordsets"`
- `m.hoveredType.Group` — e.g. `"networking.datum.net"`
- `m.hoveredType.Scope` — `"Namespaced"` or `"Cluster"`
- `m.hoveredType.Description` — OpenAPI v3 description; may be empty string

#### Non-goals
- Not fetching external docs links
- Not showing full OpenAPI schema

---

### FB-105 — Welcome screen: more inviting, helpful, and fun

**Status: ACCEPTED + PERSONA-EVAL-COMPLETE 2026-04-20 (retroactive closeout — bundled in `74fcd0a`)** — user-persona eval 2026-04-20: 1 P2 filed as **FB-116** (orientation hint falsely promises "quick-jump key below" when registrations empty — hint and quick-jump mutually exclusive by construction), 0 P3. Positive-findings: all-clear flavor line called out as "real ambient signal" with honest "detected" framing + `·` separator approved; loading guard (`!m.activityLoading`) praised for preventing premature false positive; "jump to:" prefix framed as an improvement over "Quick jump:". S1 orientation hint's primary instruction validated; secondary clause is the P2. **Prior Status: ACCEPTED 2026-04-20 (retroactive closeout — bundled in `74fcd0a`); PENDING PERSONA-EVAL** — test-engineer re-gate PASS 2026-04-20. 10/10 `TestFB105_*` tests green; FB-042 anti-regression 9/9 green; FB-083 anti-regression 6/6 green; full `./internal/tui/...` suite green; `go install ./...` EXIT 0. AC1-AC4 Observable all use `stripANSI(m.View())` + `strings.Contains`. AC5 Input-changed uses state-transition with three-check form (diff + per-leg positive + per-leg negative). AC6 Input-changed uses state-transition with `Fatalf` precondition as functional per-leg positive. AC7 Anti-behavior exercises real FB-054 `forceDashboard=true` branch. AC9 Anti-regression covers org-scope context. Integration implicit via full-suite + `go install` paste. **Noted weakness (non-blocking):** AC10 `TestFB105_AC10_AntiRegression_AllClearAbsentBelowHeightThreshold` uses `contentH=16 < 18` which falls below the pre-existing `showS4` height gate — the test proves the height gate is intact but would pass on a build where the all-clear feature was never added. Not a HOLD because AC4 (`contentH=21`) provides genuine positive coverage of the feature. AC10 is redundant-but-harmless; no rework required. Commit treatment: same retroactive-closeout-test-only-delta question as FB-099 — awaiting team-lead ruling on whether test-only deltas require their own `feat/console` commit. **Prior Status: PENDING TEST-ENGINEER 2026-04-20 (resubmission after zero-test reroute)** — engineer delivered 10 `TestFB105_*` tests across 4 axes: Observable (AC1-AC4), Input-changed (AC5-AC6), Anti-behavior (AC7-AC8), Anti-regression (AC9-AC10). All 10 PASS per `go test ./internal/tui/components/... -v -count=1`. Tests located in `internal/tui/components/resourcetable_test.go:2287+`. Verified pre-submission: AC1 uses `stripANSI(m.View())` + `strings.Contains` per `feedback_observable_acs_assert_view_output`; AC5 uses state-transition (empty → SetRegistrations → populated) with per-leg content-containment checks per `feedback_input_changed_assertions` (not snapshot-twice pattern). Axis-coverage table brief-AC-indexed. **Gap flagged for test-engineer verification:** Integration axis has no explicit table row and paste evidence is scoped to `./internal/tui/components/` only — test-engineer should confirm `./internal/tui/...` full-suite + `go install ./...` both green as part of gate-check. **Production code** (S1 orientation hint `renderHeaderBand()` ~line 641; all-clear flavor line `welcomePanel()` `showS4` block ~line 232; `"Quick jump:"` → `"jump to:"` prefix swap at `renderQuickJumpSection()` ~line 473) already landed in bundle `74fcd0a`. **Prior Status: PENDING ENGINEER 2026-04-20 (no tests submitted — zero-test gate violation)** — test-engineer batch gate-check FAIL. Verdict: zero `TestFB105_*` functions exist anywhere in the codebase despite submission claiming 11 ACs PASS. Per `feedback_engineer_zero_test_submission` + `feedback_engineer_tests_must_exist_at_submission` rules: axis-coverage table claims must map to tests that exist at submission; zero tests + AC table is a gate violation; submitter (not reviewer) must produce tests. Three production changes landed in bootstrap bundle `74fcd0a` (S1 orientation hint at `renderHeaderBand()` ~line 641; all-clear flavor line in `welcomePanel()` `showS4` block ~line 232; `"Quick jump:"` → `"jump to:"` prefix copy at `renderQuickJumpSection()` ~line 473) — code is working but has no dedicated test coverage. Route back to engineer: submitter must produce minimum Observable + Anti-behavior + Anti-regression `TestFB105_*` functions (axis-coverage table required), all using `stripANSI(table.View())`/`stripANSIModel(appM.View())` per `feedback_observable_acs_assert_view_output`; paste `go test -run '^TestFB105' ./internal/tui/... -v -count=1` output before re-routing to test-engineer. **Prior Status: PENDING TEST-ENGINEER 2026-04-20** — engineer delivered three targeted additive changes in `internal/tui/components/resourcetable.go` but without accompanying `TestFB105_*` functions. **Original Status: PENDING ENGINEER** — spec delivered 2026-04-20 by ux-designer at `docs/tui-ux-specs/fb-105-welcome-panel-personality.md`. Three targeted additive changes; no existing test-anchor copy modified (except the one intentional "Quick jump:" → "jump to:" verb swap). Deferred: first-session vs returning-operator detection (requires session state), S2 platform-health personality (copy paths too complex for this scope). Filed 2026-04-20 by team-lead from user feedback.

**Priority: P2** — the current welcome panel reads as a dense status board. New operators don't know where to start; experienced operators don't feel rewarded. The surface should orient, delight, and guide.

#### User problem

The current welcome panel (post-FB-042) is purely utilitarian: quota health, activity log, needs-attention list, quick-jump numbers. Specific gaps:

1. **No orientation** — a new operator doesn't know if they should start in the sidebar, press a number key, or do something else.
2. **No empty-state delight** — when quota is healthy and activity is empty, the panel shows a flat "all clear" and nothing else. It feels broken, not calm.
3. **No visual hierarchy** — all sections have equal visual weight; nothing guides the eye to the most actionable item.
4. **No personality** — the TUI is the most direct interface to the platform; it should feel like the platform has a voice.

#### Designer-call

Consider any combination of:
- **Greeting / orientation line** — first-session vs returning operator copy.
- **Empty-state copy** — on-brand callout for the "all clear, nothing happening" state vs the "first time here" state.
- **Visual weight hierarchy** — attention items > quota > activity when attention exists; quota > activity otherwise. Use `styles.Accent` to anchor the most important section.
- **Onboarding hint** — one-liner pointing to the sidebar or quick-jump keys for operators with no resources loaded.
- **Personality copy** — short micro-copy in section headers / empty states. Datum's voice: direct, confident, a little dry.

#### Constraints
- Must not regress FB-042 through FB-103 accepted work
- Must work at 80-column minimum terminal width
- All copy uses existing `styles.Muted` / `styles.Secondary` / `styles.Accent` tokens
- No new API calls — only existing model state
- Can ship independently of FB-104

---

### FB-106 — Placeholder action row `[r] retry describe` overflows at narrow detail-pane widths

**Status: PENDING ENGINEER** — engineer-direct. Filed 2026-04-20 by product-experience from FB-084 user-persona P3-2.

**Priority: P3** — narrow-terminal width edge-case. FB-084 shipped Option D uniform qualifier (`[r] retry describe`) across all widths; finding exposes that at ~40-col SSH sessions the retryable action row plain-text (~43 chars) exceeds usable detail-pane content width, so the row wraps or clips depending on `m.detail`'s overflow handling. Ironic gap: FB-084's stated user-problem called out 40-col SSH operators explicitly, but Option D doesn't fit there.

#### User problem

At `model.go:1969–1971`, the retryable action row concatenates `"  [E] events"` (~11 visible chars) + `"  [r] retry describe"` (~20) + `"  [Esc] back"` (~12) = ~43 chars plain-text. On a 40-col terminal, detail-pane content width is narrower than 40 (sidebar + borders consume cols). The qualifier that was supposed to disambiguate `[r]` at narrow widths may not even render legibly.

The brief itself (`docs/tui-backlog.md` §FB-084 user-problem) flagged the narrow case:

> "At narrow widths where the title-bar hint row drops, the action row is the operator's only context; `[r] retry` could mean retry events, retry the resource list, or retry describe. Operators on 40-col SSH terminals lose semantic precision."

Option D preserves semantic precision at default widths but fails the very operator the qualifier was for.

#### Proposed fix

Add a width-aware variant in `placeholderActionRow`. Threshold-based drop of the qualifier when available width is below a cutoff — falling back to the `[r] retry` short form (Option E in FB-084 spec) at narrow widths. Above-threshold behavior unchanged (qualifier renders as shipped).

**Option A (threshold gate, engineer-direct):** Signature becomes `placeholderActionRow(errMode, retryable bool, contentW int, accentBold, muted lipgloss.Style) string`. If `contentW < 40` (or chosen cutoff), render `"  [r] retry"`; else `"  [r] retry describe"`. Call site in `buildDetailContent()` passes `m.detail.Width()` (or equivalent).

**Option B (designer-call, revisit copy):** Ship a different copy altogether that survives narrow widths — e.g., `[r] retry describe` → `[r] describe` (verb drop) keeps the qualifier's disambiguation while shortening to ~11 chars. Only worth doing if designer has an opinion on whether "describe" alone communicates retry intent.

Persona suggested Option A implicitly. Engineer picks A unless the threshold value warrants designer discussion.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`, `[Integration]`.

| # | Axis | Criterion |
|---|------|-----------|
| AC1 | Observable | Retryable placeholder + `contentW ≥ threshold` (e.g., 60): `stripANSIModel(appM.View())` contains `"retry describe"`. |
| AC2 | Observable | Retryable placeholder + `contentW < threshold` (e.g., 36): `stripANSIModel(appM.View())` contains `"[r] retry"` but does NOT contain `"retry describe"`. |
| AC3 | Input-changed | Same retryable-placeholder state, width resizes from wide → narrow → wide: View() transitions `retry describe` → `retry` → `retry describe`. |
| AC4 | Anti-regression | Non-retryable placeholder unaffected at all widths: `[r]` still absent (per FB-084 AC1); `[E] events  [Esc] back` still the action row. |
| AC5 | Anti-regression | FB-084 AC2 (retryable `[r] retry describe` at default terminal width) still passes. |
| AC6 | Integration | `go install ./...` compiles; `go test ./internal/tui/... -count=1` green. |

#### Dependencies

- FB-084 ACCEPTED 2026-04-20 (placeholder action row retryability gate + qualifier restored)

#### Maps to

- FB-084 user-persona P3-2 (narrow-width overflow)

#### Non-goals

- Not revisiting Option D vs E wholesale — FB-084's default-width choice stands.
- Not adding width-awareness to the non-retryable branch — its plain-text width is ~23 chars, comfortably fits at 40 cols.
- Not introducing dynamic "progressive truncation" of every action row element. Only the retryable `[r]` qualifier is at issue.
- Not changing the `case "r":` handler.

---

### FB-107 — Quota refresh state-cleanup gaps on off-pane errors + context switch

**Status: ACCEPTED + PERSONA-EVAL-COMPLETE 2026-04-20** — test-engineer gate-check complete with evidence (`go test -c -o /dev/null` exit 0, per-AC test mapping with direct execution of anti-regression suites): Site A `BucketsErrorMsg` (model.go:407) + `LoadErrorMsg` (model.go:483) now unconditional `quota.SetLoading(false)` + `SetLoadErr()`; Site B `ContextSwitchedMsg` (model.go:553) adds `SetRefreshing(false)` alongside `SetBuckets(nil)` at line 625; accessors `IsRefreshing()` + `HasBuckets()` added at `quotadashboard.go`. 5 FB-107 tests PASS (AC1 Observable off-pane `BucketsErrorMsg` → `IsRefreshing()==false` + `bucketErr!=nil`; AC2 Observable off-pane `LoadErrorMsg`; AC3 Observable `ContextSwitchedMsg`; AC4 Input-changed 2 subtests for on-pane vs off-pane pane-gate flip; AC5 Anti-behavior bucket data preserved). Anti-regression: `TestFB063_*` 7 tests + `TestFB082_*` / `TestFB083_*` 26 tests executed directly (not source-inspected) — all PASS. Integration: full suite green. Axis-coverage table submitter-produced and pre-submission-gate complete. `bucketErr` vs `statusBar.Err` path distinction verified correct — `BucketsErrorMsg` routes to `m.bucketErr` / `m.quota.SetLoadErr()` (not `statusBar.Err` which is `LoadErrorMsg`'s target). **Persona eval 2026-04-20:** 0 P1/P2/P3 findings. 6 positive findings confirm: unconditional error-handler cleanup correct; `SetLoading(false)` cascade auto-clears `refreshing` via FB-063 setter (no separate `SetRefreshing(false)` needed in error handlers); error rendered on QuotaDashboardPane re-entry via `buildMainContent()` `case m.loadErr != nil` path (no silent-swallow gap); context-switch `SetRefreshing(false)` correct semantic (refreshing vanishes as part of full context-clear; not "silent" cancel); FB-063 acknowledgment semantic intact (error/cancel cleanup doesn't touch initiation/success); accessors match `IsLoading()` pattern uniformly. **Cosmetic note (no brief filed):** duplicate `quota.SetLoading(false)` + `SetLoadErr(msg.Err)` calls in the `pendingQuotaOpen` block at `model.go:420-424` are idempotent (same values written twice) — cosmetic tech debt, not a bug; no operator impact; persona dismissed without filing. State-cleanup invariant CLOSED. **Original Status: PENDING ENGINEER** — engineer-direct. Filed 2026-04-20 by product-experience from FB-063 user-persona P2-1 + P3-2.

**Priority: P2** — a `[r]` refresh followed by nav-away + an error response leaves `quota.refreshing=true` stuck on return to QuotaDashboardPane; the error is also swallowed silently. Next tick (up to 15s away) resolves the indicator but the error never reaches the operator. Operator perceives a refresh that never completes against stale bucket rows with no failure signal.

#### User problem

Two distinct state-cleanup gaps share the same mechanism: handlers at pane-boundary events don't clear `quota.refreshing` / `quota.loading`:

**Site A — `BucketsErrorMsg` handler (`model.go:414–416`) + `LoadErrorMsg` handler (`model.go:509`):**
Both handlers gate their `quota.SetLoading(false)` call on `m.activePane == QuotaDashboardPane`. When the operator `[r]`-refreshes, navigates away (Esc / `[3]` toggle-back), and the error arrives off-pane, the gate is false and neither `SetLoading(false)` nor `SetRefreshing(false)` fires. `quota.refreshing` remains `true`. Next QuotaDashboardPane open shows `"⟳ refreshing…"` against stale bucket rows; the error has been swallowed.

**Site B — `ContextSwitchedMsg` handler (`model.go:554–664`):**
Calls `quota.SetBuckets(nil)` / `SetActiveConsumer("", "")` etc. but never clears `quota.refreshing`. If `[r]` was in-flight at context-switch time, `quota.refreshing=true` carries over. Operator who immediately opens quota on the new context sees `"⟳ refreshing…"` against newly-emptied buckets for a narrow window until the new-context `BucketsLoadedMsg` arrives.

Both sites are **separate** from the FB-063 happy-path refresh-UX thesis and were not in FB-063's scope; filed as follow-up now that the refresh indicator exists.

#### Proposed interaction

**Site A:** Drop the `activePane` gate in `BucketsErrorMsg` (`model.go:414`) and `LoadErrorMsg` (`model.go:509`). Unconditionally call `m.quota.SetLoading(false)` — which cascades via the setter auto-clear to also reset `m.quota.refreshing`. The error display path (statusBar.Err) is already unconditional; only the loading/refreshing reset is gated. Dropping the gate preserves error visibility on return to pane + resolves stuck indicator.

**Site B:** Add `m.quota.SetRefreshing(false)` (or `SetLoading(false)` to also cascade) in the ContextSwitchedMsg handler alongside existing `SetBuckets(nil)` / `SetActiveConsumer` calls at `model.go:~560`.

Coordination with FB-063: both fixes preserve FB-063's happy-path behavior (refresh continues to skip spinner when buckets exist); they only touch cleanup paths for failed or context-transitioning refreshes.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-behavior]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable]** Site A — `BucketsErrorMsg` while `activePane != QuotaDashboardPane` and `quota.refreshing=true`: after handler fires, `quota.refreshing=false` AND `statusBar.Err != ""`. Test: set refreshing=true, set activePane=NavPane, send `BucketsErrorMsg{Err: err}`, assert both.
2. **[Observable]** Site A — `LoadErrorMsg` while `activePane != QuotaDashboardPane` and `quota.refreshing=true`: same assertion shape.
3. **[Observable]** Site B — `ContextSwitchedMsg` while `quota.refreshing=true`: after handler fires, `quota.refreshing=false`. Test: set refreshing=true, send `ContextSwitchedMsg{...}`, assert.
4. **[Input-changed]** Site A, paired: `BucketsErrorMsg` with `activePane=QuotaDashboardPane` vs `activePane=NavPane` — both now reset refreshing (previously only the Quota case did). Test: two fixtures, same message, different activePane, assert refreshing=false in both.
5. **[Anti-behavior]** Site A — `BucketsErrorMsg` does NOT clear `quota.buckets` or otherwise corrupt prior state. Test: set buckets non-empty, send error, assert buckets preserved.
6. **[Anti-regression]** FB-063 happy-path: `BucketsLoadedMsg` on QuotaDashboardPane with prior buckets still skips the loading spinner. Existing `TestFB063_*` tests green.
7. **[Anti-regression]** Existing `BucketsErrorMsg` / `LoadErrorMsg` / `ContextSwitchedMsg` tests green (error rendering, context reset paths unchanged for their positive assertions).
8. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

#### Non-goals

- Not adding a dedicated `SetRefreshing(false)` call; the setter auto-clear via `SetLoading(false)` is the established pattern from FB-063 and should be reused where possible.
- Not reworking the error-display path.
- Not changing ticker behavior (see FB-108 for ticker-path refresh indicator).

---

### FB-108 — Ticker-driven quota refresh doesn't surface `⟳ refreshing…` indicator

**Status: ACCEPTED + PERSONA-EVAL-COMPLETE 2026-04-20 (committed in 74fcd0a on feat/console — bootstrap bundle per team-lead exception)** — Persona evaluated as part of 5-feature consolidated batch; 0 FB-108-specific findings. Silent ticker contract legibility test passed: operators correctly read absence-of-indicator as "background fetch in progress" rather than "disconnected"; `[r]` remains the only trust-signaling path. Option B shipped: ticker cadence stays silent by contract; `SetRefreshing(true)` is operator-initiated only. Code: 2-line comment at `model.go:547` codifies silent-cadence intent (cites FB-108 + FB-063 principle). No behavior change. Tests added per §4 AC2 (overrides §3b "no test changes" — spec-AC-authoritative): `TestFB108_AC1_RKey_QuotaDashboardPane_SetsRefreshing` (model_test.go:1074, Anti-regression — `[r]` keypress still triggers `IsRefreshing()==true`) + `TestFB108_AC2_TickMsg_QuotaDashboardPane_DoesNotSetRefreshing` (model_test.go:1054, Anti-behavior — TickMsg on QuotaDashboardPane leaves `IsRefreshing()==false`). Test-engineer gate PASS 2026-04-20 with formal axis-coverage table: AC1 = Anti-regression row, AC2 = Anti-behavior row, AC1+AC2 together form the Input-changed pair (two inputs `[r]` vs `TickMsg` on same pane → opposite refreshing state), AC3 = Integration (full suite + go install green). Model-field assertion `IsRefreshing()` spec-authorized because AC2 explicitly names `SetRefreshing(true)` call semantics — `IsRefreshing()` is the canonical observable and directly drives banner visibility. File-location note from gate: tests landed in `internal/tui/model_test.go` not `components/quotadashboard_test.go` — informational only, not a gate violation (test-engineer flagged and assessed as correct). Paste-evidence: `go test -run '^TestFB108' ./internal/tui/... -v -count=1` both PASS; full-suite all packages green (internal/tui 15.654s, components 0.964s, data 1.332s, styles 0.158s); `go install ./...` EXIT 0. Filed 2026-04-19 by product-experience from FB-063 user-persona. Resolves persona framing-question (a): ticker silence was intentional, not a FB-063 miss.

**Original Status: PENDING ENGINEER** — spec delivered 2026-04-20 by ux-designer at `docs/tui-ux-specs/fb-108-ticker-refresh-silent-cadence.md`. **Option B selected — silent ticker is intentional; comment only, no behavior change.** Rationale: FB-063's thesis was operator-initiated `[r]` acknowledgment. Surfacing ticker cadence every 15s creates flicker without operator action, dilutes the `[r]` indicator's meaning as confirmation, and `"updated X ago"` timestamp already serves the ambient-signal role. Codifies the principle: `SetRefreshing(true)` = operator-initiated only. Implementation: 2-line comment at `model.go:547` explaining the silent-cadence intent (cite FB-108). No code change; no new tests. ACs: anti-regression (FB-063 `[r]` path green), anti-behavior (TickMsg does NOT set `quota.refreshing=true`), integration.

**Priority: P3** — periodic 15s tick-driven quota refreshes remain silent post-FB-063; only the `[r]` key-press path surfaces `"⟳ refreshing…"`. Either (a) silent ticker refresh was intentional (less noise for background cadence), in which case brief copy was misleading, or (b) the ticker path was missed during FB-063 impl. Persona's framing: "Whether the tick path was intended to surface the indicator is unclear from the code alone." **Resolved (a) — intentional per ux-designer Option B.**

#### User problem

FB-063 spec described routing "periodic refresh (driven by ticker-initiated `LoadQuotaCmd`)" through `SetRefreshing`. Engineer shipped the `[r]` key-handler switch (`model.go:~1305`) but the `TickMsg` QuotaDashboardPane branch (`model.go:546–549`) dispatches `data.LoadBucketsCmd` without calling `quota.SetRefreshing(true)`. Result: every 15s, quota fetches run silently, title bar's `"updated X ago"` timestamp advances after completion, but no in-flight indicator is shown.

Operator-perception question: is silent cadence refresh the desired behavior (background work stays background), or does surfacing each tick-refresh improve trust ("I can see the dashboard is still connected")?

#### Designer-call

- **Option A** — Surface `⟳ refreshing…` on tick too: call `quota.SetRefreshing(true)` at `model.go:547` before dispatching `LoadBucketsCmd`. Identical UX to `[r]` path. Indicator flickers briefly every 15s.
- **Option B** — Keep ticker silent (current state): clarify brief/scope — ticker cadence is intentionally background-invisible; `[r]` is the only operator-facing refresh signal. Document the rationale; no code change.
- **Option C** — Subtler ticker-only indicator: distinct glyph (e.g., `·` vs `⟳`) or no text change at all. Adds complexity without clear benefit.

**Prefer A or B.** A restores parity with the brief's original framing; B acknowledges the ambient-noise argument. C splits the signal design for little gain.

#### Acceptance criteria (conditional on Option selected)

**If Option A:**
1. **[Observable]** Tick arrives on QuotaDashboardPane: `quota.refreshing=true` AND title bar contains `"refreshing"` string.
2. **[Input-changed]** Tick on QuotaDashboardPane (refresh fires) vs tick on non-QuotaDashboardPane (no refresh, no indicator): different `quota.refreshing` state.
3. **[Anti-regression]** `[r]` path still surfaces indicator (FB-063 AC2 green).
4. **[Integration]** `go install ./...` green.

**If Option B:** Document in `docs/tui-plan.md` "ticker refresh is ambient/invisible"; add explicit comment at `model.go:547`. No new tests.

#### Non-goals

- Not changing the 15s cadence interval.
- Not adding indicator to other ticker-driven commands (activity, bucket-level refresh unrelated).
- Not revisiting FB-063 shipped scope — this is distinct follow-up.

---

### FB-109 — DetailPane `[3]` top-hint vs bottom-separator copy divergence ("quota dashboard" vs "full dashboard")

**Status: ACCEPTED + PERSONA-EVAL-COMPLETE 2026-04-20 (committed in 74fcd0a on feat/console — bootstrap bundle per team-lead exception)** — Persona evaluated as part of 5-feature consolidated batch; 0 FB-109-specific findings. Copy alignment confirmed as "a genuine fix" by persona ("Before, the top hint said `[3] quota dashboard` and the bottom separator said `[3] full dashboard`. I had no idea if those were the same destination. Now both say `[3] quota dashboard`."). Test-engineer gate-check clean. 2 new tests + 5 stale-literal tests all PASS, full suite green, `go install ./...` clean. `TestFB109_AC1_BottomSeparatorCopyUpdated` asserts `stripANSIModel(m.buildQuotaSectionHeader())` + `strings.Contains(got, "[3] quota dashboard")` + negative `"full dashboard"` check — View()-based. `TestFB109_AC5_PrefixWidth24_SeparatorFillsWidth` pins `lipgloss.Width(line) == 80`. 5 pre-existing tests updated per spec AC4: `TestFB044_BuildDetailContent_MatchingBuckets_Wide_Has3Affordance`, `TestFB064_AC2_AntiRegression_BottomSeparatorRetained`, `TestFB066_PrefixWidth_ConstantMatchesRenderedWidth`, `TestAppModel_BuildDetailContent_MatchingBuckets_AppendsQuotaBlock`, `TestAppModel_BucketsLoadedMsg_InDetailPane_UpdatesDetailContent` (the last of these was the FB-088 full-suite blocker — now green, unblocks FB-088 re-gate). Axis-coverage: Observable (AC1), Anti-regression (AC4+AC5), Integration (AC6); Input-changed + Anti-behavior N/A per spec §6 (literal-copy change). Persona eval queued. Implementation: `model.go:~1942` `buildQuotaSectionHeader()` — `prefixRendered` drops `"── Quota "` and `" full"`; `const prefixWidth = 24` (was 29).

**Prior Status: PENDING ENGINEER** — spec delivered 2026-04-20 by ux-designer at `docs/tui-ux-specs/fb-109-quota-dashboard-copy-alignment.md`. **Option A clean variant selected — `"── [3] quota dashboard ──"`**: drops both `"full"` qualifier (no referent) and leading `"Quota "` prefix (avoids `"Quota [3] quota dashboard"` repetition). Exact vocabulary match with FB-064 top hint. Implementation: 2-line change in `buildQuotaSectionHeader()` at `model.go:1942` — new `prefixRendered` drops `"── Quota "` and `" full"`; `const prefixWidth = 24` (was 29; plain text `"── [3] quota dashboard  "` = 24 chars). Top hint at `model.go:1919` unchanged. 6 ACs: AC1 bottom `"full dashboard"` absent, AC2 top hint `"quota dashboard"` unchanged, AC3 key handler preserved, AC4 existing FB-044 tests updated to new literal, AC5 prefixWidth constant correctness, AC6 integration.

**Priority: P3** — same key (`[3]`) advertised twice on the same DetailPane content with different qualifiers: top hint says `"[3] quota dashboard"` (FB-064 ship), bottom separator says `"── Quota [3] full dashboard ──"` (pre-existing FB-044 ship). An operator reading a long manifest who notices both may wonder whether the two `[3]` references invoke the same action. In fact they do — both route to QuotaDashboardPane via the same handler. The word "full" was likely a legacy contrast against an eventually-planned inline summary; now that the top hint exists, "full" has no contrast to anchor.

#### User problem

Verified in code:
- **Top hint (`model.go:1919`):** `"[3] quota dashboard ─────"` (FB-064, prepended to buildDetailContent).
- **Bottom separator (`model.go:1942` `buildQuotaSectionHeader`):** `"── Quota [3] full dashboard ──"` (FB-044-era).

Both advertise the same `[3]` keybind → same transition. The "full" qualifier is the only difference. Persona reports reading the pair creates "ambiguity about whether they advertise the same action."

#### Designer-call

Where should the vocabulary align?

- **Option A** — Drop "full" from bottom separator → `"── Quota [3] quota dashboard ──"`. Exact vocabulary alignment; slight repetition ("Quota [3] quota dashboard"). Could alternatively drop the leading "Quota" header: `"── [3] quota dashboard ──"` — cleanest alignment.
- **Option B** — Update top hint to match bottom → `"[3] full dashboard ─────"`. Aligns on "full"; but "full" is meaningful only in contrast to a summary that doesn't exist.
- **Option C** — Keep divergence; rationalize as intentional tonal difference (top is discovery hint, bottom is section-header framing). Persona explicitly flagged this as ambiguous, so "intentional" requires justification.
- **Option D** — Make top hint a shorter glyph-only affordance (`[3]⤴`) to avoid copy overlap entirely. Discoverability cost at narrow widths.

**Prefer Option A** (drop "full" from bottom; optionally drop "Quota" prefix too — cleanest). B carries a legacy qualifier that no longer contrasts. C requires user evidence the divergence is helpful. D trades copy collision for affordance weight reduction.

#### Acceptance criteria (conditional on Option selected)

**If Option A:**
1. **[Observable]** Bottom separator no longer contains `"full dashboard"`; contains `"quota dashboard"` (or aligned variant).
2. **[Observable]** Top hint unchanged; contains `"quota dashboard"`.
3. **[Anti-regression]** `[3]` handler unchanged; both affordances still trigger same transition. Existing FB-044/FB-064 behavioral tests green.
4. **[Anti-regression]** Existing `buildQuotaSectionHeader` substring tests updated to match new copy (literal swap; behavior preserved).
5. **[Integration]** `go install ./...` + suite green.

#### Non-goals

- Not removing either affordance (FB-064 top + FB-044 bottom both exist for scroll-boundary discovery).
- Not changing `[3]` functional behavior.
- Not re-litigating FB-064 placement — copy alignment only.
- Not addressing narrow-width `innerW < 30` drop (both affordances correctly drop together per FB-064 persona P3-3 dismissal).

---

### FB-110 — Help overlay "resume cached" sub-label doesn't match S1 "(cached)" parenthetical idiom after FB-089 shipped

**Status: ACCEPTED + PERSONA-EVAL-COMPLETE 2026-04-20 (committed in 74fcd0a on feat/console — bootstrap bundle per team-lead exception)** — Persona evaluated as part of 5-feature consolidated batch; 0 FB-110-specific findings. Parenthetical-idiom fix confirmed as "a minor polish win" by persona ("The help overlay sub-label grammar is cleaner. Small, but noticed."). Test-engineer gate-check clean. Option A shipped: `helpoverlay.go:29` `"resume cached"` → `"resume (cached)"`. No dedicated `TestFB110_*` — coverage via repurposed `TestFB057_AC1AC2_HelpOverlay_NewCopy` AC2 assertion (now asserts `strings.Contains(view, "resume (cached)")` on `stripANSIModel(appM.helpOverlay.View())`, comment updated to `// AC2: Tab sub-line "resume (cached)" present. FB-110 updated copy.`). Axis-coverage: Observable (AC2 of FB-057 repurposed), Anti-regression (`TestFB057_AC3_HelpOverlay_PreexistingKeys`, `TestFB057_AC4_HelpOverlay_LineCountUnchanged`), Integration (full suite + `go install ./...` green). 3/3 FB-057 tests PASS. Pattern note: 1-line literal change accepted with repurposed test — for future similar changes, prefer dedicated `TestFB<num>_*` to avoid silent coverage loss on upstream FB rework. Persona eval queued.

**Prior Status: PENDING ENGINEER** — engineer-direct. Filed 2026-04-20 by product-experience from FB-089+FB-090 user-persona P3-1.

**Priority: P3** — post-FB-089, S1 renders `"[Tab] resume dns (cached)"` using `(cached)` as a parenthetical state qualifier after the type name. Help overlay at `helpoverlay.go:28–29` still renders `"[Tab] next pane"` with a subordinate muted sub-label `"      resume cached"` — bare adjective, no parenthetical, no type name. The two surfaces express the same concept in slightly different idioms.

#### User problem

Verified in code:
- **S1 (`resourcetable.go:543–581`):** renders `"[Tab] resume X (cached)"` at tier 1 and `"[Tab] resume X"` at tier 2 (width-gated). FB-089 established the `(cached)` parenthetical pattern.
- **Help overlay (`helpoverlay.go:28–29`):** renders `"[Tab] next pane"` on line 1 and `"      resume cached"` as a muted continuation on line 2. Static text; predates FB-089 rework; FB-091 was DISSOLVED (subsumed into FB-089's `(cached)` pattern) but help overlay sub-label was not updated as part of that consolidation.

An operator toggling the help overlay (`?`) while looking at a DNS cache hint in S1 will see:
- Main panel: `"[Tab] resume dns (cached)"`
- Help overlay: `"[Tab] next pane / resume cached"`

Two surfaces, same key, same state — slightly different words ("resume cached" vs "resume X (cached)"). Minor drift; not a contradiction. Persona P3 severity reflects "minor vocabulary drift across two surfaces" without concrete operator confusion evidence.

#### Proposed fix

Engineer-direct, 1-line change at `helpoverlay.go:29`. Candidate replacements:

- **A.** `"resume (cached)"` — parenthetical-aligned with S1 idiom; preserves bare form (no type name since help overlay is context-free).
- **B.** Drop sub-label entirely; help overlay becomes `"[Tab] next pane"` only. S1 already carries the state-specific qualifier in-context; help overlay's role is to name the generic behavior.
- **C.** `"resume cached table"` — adds a noun the bare adjective lacked; spells out what is cached.

**Prefer Option A** — smallest diff; aligns vocabulary exactly without losing information. Option B also defensible (cleanest scope reduction), pick on aesthetic judgment.

#### Acceptance criteria

1. **[Observable]** Help overlay sub-label at `helpoverlay.go:29` renders the aligned copy (Option A: `"resume (cached)"`; Option B: absent; Option C: `"resume cached table"`). Test: View() substring assertion on help overlay output.
2. **[Anti-regression]** `[Tab]` keybinding behavior unchanged; existing help overlay tests (if any) green.
3. **[Anti-regression]** S1 cached hint copy unchanged (`"[Tab] resume X (cached)"` still renders per FB-089); existing FB-089 tests green.
4. **[Integration]** `go install ./...` + suite green.

#### Non-goals

- Not touching S1 rendering (FB-089 scope; no copy change needed).
- Not updating S6 keybind strip `"Tab next pane"` (persona P3-3 dismissed the S1/S6 verb mismatch as acceptable contextual layering).
- Not adding type-name substitution to help overlay — overlay is context-free; inserting dynamic `m.typeName` into static help text is scope-creep.
- Not re-introducing FB-091 as a separate brief (DISSOLVED into FB-089).

---

### FB-111 — Activity hint `"[4] full dashboard"` still renders when `activityFetchFailed=true` + stale rows (post-FB-083 render-layer gate gap)

**Status: ACCEPTED + PERSONA-EVAL-COMPLETE 2026-04-20** — test-engineer gate-check complete with evidence (`go test -c -o /dev/null` exit 0, Observable ACs use `stripANSI(m.View())` substring checks per `feedback_observable_acs_assert_view_output`, AC3 Input-changed uses same-fixture `SetActivityFetchFailed` toggle). One-line gate at `resourcetable.go:310` — `if len(m.activityRows) > 0 && !m.activityFetchFailed` — correctly suppresses hint on re-fetch-after-success error path while preserving FB-083 baseline for `fetchFailed=false`. 3 FB-111 tests PASS (AC1 stale+fail hint absent, AC2 stale+ok hint present, AC3 same-rows toggle View() differs). AC4 covered by existing `TestFB083_AC6_AntiBehavior_SuppressedHint_4KeyStillNavigates` — hint-suppressed state navigation proven under nil-rows (strict subset of `fetchFailed=true`; same dispatch handler). Anti-regression: `TestFB083_*` 6 tests + `TestFB082_ErrorRecovery_*` tests executed directly — all PASS. Integration: full suite green. No new fields/setters; defense-in-depth with FB-100 state-layer fix maintained (FB-111 fixes render layer only; FB-100 remains in queue). **Persona eval 2026-04-20:** 0 P1/P2/P3 findings. 5 positive findings confirm: gate exactly matches FB-083 P2-1 finding prescription; render-layer gate is sufficient for operator-facing invariant closure (no render-lag window; FB-100 state-layer fix is internal cleanup, not operator-facing); `[4]` press when hint absent reaches visible error state via `ProjectActivityErrorMsg` → `activityDashboard.SetLoadErr()` (not a silent dead journey); S6 strip `[4] activity` always present (key reference unaffected by hint suppression); binary operator perception ("hint present = data available" vs "hint absent = anything else") — three internal gate conditions render as two operator-facing states. Hint-truthfulness invariant CLOSED at render layer. **Original Status: PENDING ENGINEER** — engineer-direct. Filed 2026-04-20 by product-experience from FB-083 user-persona P2-1.

**Priority: P2** — reachable truthfulness contradiction on the re-fetch-after-success-then-error path. Hint promises "full dashboard" content while body reads `"activity unavailable"`. User-persona explicitly frames the issue as *"two separate, independently fixable conditions"* at distinct code sites: this brief addresses the **render-layer gate**; FB-100 addresses the complementary **state-layer gate** (suppress `SetActivityFetchFailed(true)` when stale rows are still valid). Defense-in-depth — either fix independently resolves the contradiction on the current code path; both together close the invariant at both layers.

#### User problem

Post-FB-083, `renderActivitySection()` gates the `"[4] full dashboard"` hint on `len(m.activityRows) > 0`. This correctly suppresses the hint in the three FB-083 cases (empty, loading, fetchFailed-with-empty-rows). However, on the **re-fetch-after-successful-load error path**:

1. Initial load succeeds → `activityRows = [row1, row2, ...]` (stale-on-success).
2. Operator presses `[r]` (or auto-refresh fires) → refresh error → `activityFetchFailed = true` is set, but `activityRows` is NOT cleared (per FB-082 Option B stale-rows-on-re-fetch-error preservation).
3. Body now reads `"activity unavailable"` (FB-082 error copy) while header hint still renders `"[4] full dashboard"` because `len(activityRows) > 0` is still true.

The operator sees a hint promising richer content behind `[4]` while the body says that content is unavailable. FB-083's truthfulness invariant — "hint reflects data availability" — breaks on this specific combination.

**Verified against code:** `resourcetable.go:309–312` (FB-083 gate: `if len(m.activityRows) > 0 { hint = "[4] full dashboard" }`) — predicate does not account for `m.activityFetchFailed`.

#### Proposed fix

Engineer-direct, one-line gate update at `resourcetable.go:309–312`:

**Current:**
```go
hint := ""
if len(m.activityRows) > 0 {
    hint = "[4] full dashboard"
}
```

**New:**
```go
hint := ""
if len(m.activityRows) > 0 && !m.activityFetchFailed {
    hint = "[4] full dashboard"
}
```

No new fields or setter wire-up required — `m.activityFetchFailed` already lives on `ResourceTableModel` (introduced by FB-082) and is already set/cleared at the correct call sites.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-behavior]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable]** With `activityRows=[row1]` + `activityFetchFailed=true`: `stripANSI(m.table.View())` does NOT contain `"[4] full dashboard"`. Test: set state; assert substring absent.
2. **[Observable]** With `activityRows=[row1]` + `activityFetchFailed=false`: `stripANSI(m.table.View())` DOES contain `"[4] full dashboard"`. Test: set state; assert substring present. (Baseline FB-083 behavior preserved.)
3. **[Input-changed]** Same `activityRows=[row1]`, different `activityFetchFailed`: View() with `fetchFailed=true` ≠ View() with `fetchFailed=false` (hint appears/disappears on the same row data). Test: assert diff.
4. **[Anti-behavior]** `[4]` keybinding still routes to ActivityDashboardPane regardless of hint suppression (display-only gate; navigation unaffected). Test: send key "4" with hint suppressed; assert pane transition.
5. **[Anti-regression]** FB-083 AC1–AC6 remain green (empty / loading / fetchFailed-with-empty-rows / populated / Input-changed pair / `[4]` navigation).
6. **[Anti-regression]** FB-082 stale-rows-on-re-fetch-error behavior unchanged (rows are NOT cleared on re-fetch error; only the hint is suppressed). Test: existing FB-082 rows-preservation tests green.
7. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/... -count=1` green.

#### Non-goals

- Not clearing `activityRows` on re-fetch error — state-layer stale-row preservation is FB-082's established behavior; FB-111 is a display-only gate.
- Not changing `activityFetchFailed` semantics or call sites — FB-100 owns the state-layer gate (`SetActivityFetchFailed(true)` should not fire when stale rows are still present). FB-111 + FB-100 are defense-in-depth at different layers; ship either first.
- Not touching the body copy `"activity unavailable"` (FB-082 error state; unchanged).
- Not touching the `[4]` keybinding or ActivityDashboardPane rendering.

**Dependencies:** FB-083 ACCEPTED (gate site), FB-082 ACCEPTED (`activityFetchFailed` field exists). Coordinates with FB-100 (state-layer fix at complementary code site).

**Maps to:** FB-083 user-persona P2-1.

---

### FB-112 — Ready-prompt persists in status bar after `[3]` confirm (FB-097 + FB-088 interaction)

**Status: WONTFIX 2026-04-20 — bug not reproducible; `handleNormalKey:1034` is the global per-keypress hint-clear site; persona finding was inferred from code reading, not live reproduction; explicit fix line retained as defensive documentation; pass-through tests removed per state-transition rigor rule.** Team-lead ruled Option 1 after engineer experiment confirmed spec premise doesn't reproduce. Root cause: `handleNormalKey()` at `model.go:1033–1036` unconditionally clears `statusBar.Hint` on every keypress BEFORE any `case` branch runs: `if m.statusBar.Hint != "" { m.statusBar.Hint = ""; m.statusBar.BumpHintToken() }`. The `[3]` confirm routes through `handleNormalKey` in ModeNormal, so FB-097's persistent ready-prompt is already cleared by the time the confirm branch sets `activePane = QuotaDashboardPane`. The "two `[3]` labels simultaneously visible" contradiction described in persona-eval + ux-designer spec does not manifest at runtime. Engineer experiment (fix reverted, tests kept): `TestFB112_AC1_ConfirmClearsReadyPrompt` PASS, `TestFB112_AC2_InputChanged_PreConfirmVsPostConfirm` PASS — both pass-through, confirming they cannot distinguish fix-present from fix-absent. No alternate codepath sets `activePane = QuotaDashboardPane` while preserving a non-empty `statusBar.Hint`. **Team-lead resolution (three actions):** (1) explicit fix line at `model.go:1224` (`m.statusBar.Hint = ""`) RETAINED — harmless, makes the confirm-path intent readable, costs nothing; (2) two pass-through tests REMOVED from `model_test.go` — net liability per state-transition rigor rule; (3) one-line comment ADDED at `model.go:1034` (the `handleNormalKey` hint-clear guard) documenting it as the global per-keypress clear site — prevents future personas/designers from filing similar briefs. Cleanup committed as `25de0c7` on `feat/console` (optional per ruling; team-lead chose to land it). Filed 2026-04-20 by product-experience from FB-088/097/109/110/108 batch user-persona P2; closed same day.

**Prior Status: PENDING ENGINEER** — spec delivered 2026-04-20 by ux-designer at `docs/tui-ux-specs/fb-112-ready-prompt-clear-on-confirm.md`. **Option A selected — `m.statusBar.Hint = ""` at `model.go:1223`, before `updatePaneFocus()`.** One-line fix. Option C rejected as over-refactor for a P2 hint-clear (cancel path at `model.go:1235` already uses direct `m.statusBar.Hint = ""` pattern — Option A matches that symmetry without introducing a helper). Option B rejected as operator-action noise. Defensive note in spec §4: verify whether `m.table.SetPendingQuotaOpen(false)` is cleared in `BucketsLoadedMsg` handler; if strip label doesn't reset on confirm, add it after the hint clear (defensive-only). 6 ACs: Observable (AC1 View() does not contain ready-prompt post-confirm), Input-changed (AC2 pre-confirm vs post-confirm View() differ on ready-prompt substring), Anti-regression (AC3 FB-097 pre-confirm HintClearMsg persistence preserved, AC4 FB-088 title bar unchanged, AC5 FB-080 cancel path unchanged), Integration (AC6). Anti-behavior axis N/A per spec §6 (no "shouldn't fire" beyond AC3 pre-confirm persistence).

**Prior Status: PENDING UX-DESIGNER** — filed 2026-04-20 by product-experience from FB-088/097/109/110/108 batch user-persona P2. FB-097 spec §6 assumed `[3] confirm replaces via natural UI flow`, but the confirm path at `model.go:1220–1224` (`!m.quota.IsLoading()` + `pendingQuotaOpen=true` → activePane=QuotaDashboardPane) never calls `statusBar.Hint = ""` or re-posts. `updatePaneFocus()` only sets `statusBar.Pane`, not `Hint`. Result: after BucketsLoadedMsg fires FB-097's persistent `"⚡ Quota dashboard ready — press [3]"`, user confirms with `[3]` and enters the dashboard — but the status bar still reads `"Quota dashboard ready — press [3]"` while the title bar reads `"[3] back to <origin>"` (FB-088). Two `[3]` labels with opposite semantic directions co-visible. If operator trusts status bar and presses `[3]` again, they navigate back to origin unexpectedly.

**Priority: P2** — reachable via normal cold-load flow: press `[3]` during load → wait → press `[3]` to confirm. Hit on every cold-load entry to QuotaDashboard. Invalidates the trust contract FB-097 was meant to establish.

#### User problem

FB-097 made the ready-prompt persistent so operators who look away don't miss it. The confirm-by-`[3]` transition needs a matching clear so the prompt doesn't survive past its reason-for-existence. Currently it does.

#### Proposed fix

Designer picks:

- **Option A** — Clear `statusBar.Hint = ""` inside the `!m.quota.IsLoading()` branch at `model.go:1223` before `updatePaneFocus()`. Also clear `m.pendingQuotaOpen = false` + `m.table.SetPendingQuotaOpen(false)` if not already (defensive — verify whether these get cleared elsewhere post-confirm).
- **Option B** — Replace the ready-prompt hint on confirm with a short transient (e.g., `"Entered quota dashboard"`) via `m.postHint(...)`. Adds noise but preserves the "something happened" signal.
- **Option C** — Route the clear through a new `m.confirmQuotaOpen()` helper that both sets activePane and clears the FB-097 ready state, symmetric with FB-080's second-press cancel path at `model.go:1232–1237`.

**Prefer A or C.** A is minimal; C is architecturally symmetric with the cancel path.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable]** After `[3]` during loading → BucketsLoadedMsg → `[3]` confirm: `stripANSIModel(statusBarNorm(appM))` does NOT contain `"Quota dashboard ready"` once activePane == QuotaDashboardPane. Test: drive the sequence; assert substring absent post-confirm.
2. **[Input-changed]** Same `pendingQuotaOpen=true` + `BucketsLoadedMsg` received, different post-confirm action — before fix View() contains ready-prompt, after fix View() does not. Test: assert post-confirm substring differs from post-BucketsLoadedMsg-only state.
3. **[Anti-regression]** FB-097 persistent-prompt behavior preserved pre-confirm — ready-prompt still survives `HintClearMsg` between load-complete and confirm (TestFB097_AC4 + TestFB097_BriefAC4 green).
4. **[Anti-regression]** FB-088 title-bar `[3] back to <origin>` label unaffected by the clear path (TestFB088_* green).
5. **[Anti-regression]** FB-080 second-press cancel path unchanged (`statusBar.Hint = ""` already present at model.go:1235; confirm path now matches).
6. **[Integration]** `go install ./...` + `go test ./internal/tui/... -count=1` green.

**Dependencies:** FB-097 ACCEPTED (persistence contract), FB-088 ACCEPTED (title-bar label), FB-080 ACCEPTED (cancel-path clear — reference implementation).

**Maps to:** FB-088/097/109/110/108 batch user-persona P2.

**Non-goals:**
- Not changing FB-097 persistence semantics pre-confirm.
- Not touching FB-088 title-bar label rendering.
- Not introducing a new hint-timeout for the ready-prompt (persistence is FB-097's contract).

---

### FB-113 — Empty-bucket quotadashboard viewport shows hardcoded `[Esc] back to navigation` alongside FB-088 dynamic title-bar label

**Status: ACCEPTED + PERSONA-EVAL-COMPLETE 2026-04-20** — commit `6df4429` on `feat/console` ("Match the empty-quota back-hint to the title-bar destination"). Engineer-direct 1-line fix at `internal/tui/components/quotadashboard.go:163–173`: hardcoded `"  [Esc] back to navigation"` replaced with `fmt.Sprintf("  [Esc] back to %s", backLabel)` where `backLabel = m.originLabel` (fallback `"navigation"` when empty). Reuses the FB-088 `originLabel` field already threaded onto `QuotaDashboardModel`. Test-engineer gate PASS with formal axis-coverage table (independent verification): 5 tests AC1–AC5 + integration AC6, all use `stripANSI(m.buildMainContent())` + `strings.Contains(...)` — zero model-field inspection. `buildMainContent()` acceptable as operator-facing surface per test-engineer review (direct input to viewport, no intermediate transform). AC3 Input-changed rigor confirmed three-check pattern: `got1 != got2` + `strings.Contains(got1, "resource list")` + `strings.Contains(got2, "welcome panel")` — single-`!=` alone cannot pass on whitespace-only diff; per-render substring checks prove intent. Paste-evidence (test-engineer independent run): `go test -run '^TestFB113' ./internal/tui/components/... -v -count=1` all 5 PASS; full suite green (internal/tui 15.644s, components 1.345s, data 0.981s, styles 0.162s); all 17 `TestFB088_*` anti-regression tests PASS (origin-label contract preserved). **Persona-eval CLOSED** — user-persona confirmed fix correct and complete across all three origin cases (TablePane, NavPane, DetailPane); title-bar/viewport contradiction eliminated; `originLabel == ""` fallback to `"navigation"` acknowledged as unreachable-in-practice safe behavior. No new findings. Filed 2026-04-20 by product-experience from FB-088/097/109/110/108 batch user-persona P3.

**Prior Status: PENDING ENGINEER** — engineer-direct. Filed 2026-04-20 by product-experience from FB-088/097/109/110/108 batch user-persona P3. `quotadashboard.go:170` hardcodes `"  [Esc] back to navigation"` in the no-buckets empty-state block. FB-088 added dynamic origin label to the title bar but didn't touch this in-viewport literal. When origin is non-NavPane (e.g., TablePane resource-list), title bar correctly shows `[3] back to resource list` while the empty-state viewport says `[Esc] back to navigation` — two labels, different destination names, same actual target. Operator reads top-to-bottom and concludes `3` and Esc go to different places. They don't.

**Priority: P3** — requires specific setup (no buckets configured for active context AND entry from non-NavPane). When origin IS NavPane, both literals agree and there's no contradiction.

#### User problem

The FB-088 origin-label contract (QuotaDashboardModel has `originLabel` via `SetOriginLabel(...)`) isn't reused by the empty-state viewport copy. The empty state was written pre-FB-088 assuming Esc always goes back to NavPane.

#### Proposed fix

Engineer-direct, ≤5 lines:

1. Thread the `originLabel` into `buildMainContent()` (or the empty-state sub-function) — it's already a field on `QuotaDashboardModel`.
2. Replace the hardcoded `"[Esc] back to navigation"` with `fmt.Sprintf("[Esc] back to %s", originLabel)` — matching the FB-088 title-bar convention.
3. Fallback: if `originLabel == ""` (fresh startup edge case, no origin set), render `"[Esc] back to navigation"` to preserve existing behavior — or render no hint at all, matching FB-088's empty-on-fresh-startup treatment.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable]** Empty-state viewport rendered with `originLabel = "resource list"` contains `"[Esc] back to resource list"`. Test: `stripANSI(m.View())` substring assertion.
2. **[Observable]** Empty-state viewport rendered with `originLabel = "welcome panel"` contains `"[Esc] back to welcome panel"`. Test: same pattern.
3. **[Input-changed]** Same empty-state fixture, different `originLabel` → rendered View() differs (both `[Esc] back to <label>` literals present). Test: assert View() substring difference.
4. **[Anti-regression]** `originLabel = ""` fallback still renders (chosen option's copy) and does not crash.
5. **[Anti-regression]** Populated-bucket state unaffected (no empty-state block rendered; existing dashboard rendering tests green).
6. **[Integration]** `go install ./...` + `go test ./internal/tui/... -count=1` green.

**Dependencies:** FB-088 ACCEPTED (origin label contract + vocabulary).

**Maps to:** FB-088/097/109/110/108 batch user-persona P3.

**Non-goals:**
- Not changing the empty-state copy (`"No allowance buckets configured for this context."` stays).
- Not adding an `originLabel` setter site beyond what FB-088 already pinned.
- Not introducing a new origin-label vocabulary entry.

---

### FB-114 — Help overlay: `[d] describe` listed twice (ACTIONS + VIEW sections)

**Status: ACCEPTED + PERSONA-EVAL-COMPLETE 2026-04-20** — user-persona eval 2026-04-20 returned 0 P1/P2/P3 + 1 positive finding. Persona confirmed: ACTIONS column now unambiguously imperative (`[Enter] select`, `[/] filter`, `[Esc] back/home` + pane-conditional lines); VIEW column unambiguously read-only display modes (`[d] describe`, `[r]`, `[c]`, `[3]`, `[t]`, `[4]`); sectioning conceptually sound; no false removal (`[d] describe` remains reachable in VIEW); column-balance check passed (fixed 22-char widths make height asymmetry cosmetically acceptable). Fix resolved the originally filed confusion without introducing new issues. **Prior Status: ACCEPTED + COMMITTED + PENDING PERSONA-EVAL 2026-04-20** — engineer committed `fb44249` on `feat/console`: _"Clean up help overlay — 'describe' now appears once, not twice"_. Strict one-commit-per-feature rule honored (genuinely-new production change, not retroactive-closeout exception). **Prior Status: COMMITTED TO FEAT/CONSOLE 2026-04-20** — test-engineer PASS 2026-04-20: `go build ./internal/tui/...` clean; 4/4 `TestFB114_*` green; production code verified (`helpoverlay.go` actionLines contains no `[d]` entry; VIEW section retains `[d]  describe` once); `strings.Count(stripANSI(m.View()), "[d]") == 1` confirmed at runtime; full suite + `go install ./...` green. **AC2 axis-mislabel ruling — accepted as-is:** test-engineer confirmed the brief has no genuine Input-changed axis (static dedup fix, no state transition); engineer followed AC wording literally; `count != 2` is a real pre-fix regression detector (would fail if the removed line were re-added), not a pass-through. Relabeling is cosmetic, not worth a rework cycle. **Prior Status: PENDING TEST-ENGINEER 2026-04-20** — engineer delivered Option A implementation: removed `row.Render("[d]    describe")` from `actionLines` slice in `internal/tui/components/helpoverlay.go`; VIEW column `[d]  describe` unchanged. 4 `TestFB114_*` tests green (AC1 Observable count==1 via `strings.Count(stripANSI(m.View()), "[d]")`; AC2 Input-changed post-fix snapshot; AC3 Anti-regression describe-still-present-in-VIEW; AC4 Anti-regression conditional lines Shift+C/Shift+E/[x] unaffected). Axis-coverage table submitter-produced; Anti-behavior marked N/A (static text removal has no conditional behavior); Integration confirmed via full `./internal/tui/...` + `go install ./...` green. **Prior Status: PENDING ENGINEER 2026-04-20** — ux-designer spec delivered 2026-04-20 at `docs/tui-ux-specs/fb-114-help-overlay-describe-dedup.md`. Decision: Option A (remove `[d] describe` from ACTIONS section at `helpoverlay.go:38`, keep at `:53` in VIEW). Rationale: VIEW is the home for display-mode toggles (`[3] quota (toggle)`, `[4] activity (toggle)`); ACTIONS is for imperative commands with side-effects (`[Enter] select`, `[x] delete`); describe is read-only so VIEW is the conceptual fit. One line removed. **Prior Status: PENDING UX-DESIGNER 2026-04-20** — designer-call required on which section is canonical. Filed 2026-04-20 by product-experience from FB-072 user-persona batch P3 finding.

**Priority: P3** — help-overlay readability; momentary reader confusion without functional impact. Persona observed: seeing the same key in two sections reads as two distinct behaviors (e.g. ACTIONS `[d]` = "open describe pane" vs VIEW `[d]` = "toggle describe mode"), forcing a re-read to confirm they match.

#### User problem

`internal/tui/components/helpoverlay.go` renders `[d] describe` in two separate columns:

- **Line 38 (ACTIONS section):** `row.Render("[d]    describe")`
- **Line 53 (VIEW section):** `row.Render("[d]  describe")`

Both reference the same action. No other key has this dual-section treatment. The duplicate breaks the visual implicit that each column is a distinct category of interaction.

#### Designer-call

- **Option A** — Remove from ACTIONS, keep in VIEW. VIEW describes display-mode toggles (`[d] describe`, `[r] refresh`, `[3] quota`, `[4] activity`) which is where "describe" belongs conceptually — it toggles a view over the highlighted row.
- **Option B** — Remove from VIEW, keep in ACTIONS. If "describe" is framed as an action taken on a selection (like `[Enter] select`, `[x] delete`), it belongs in ACTIONS.
- **Option C** — Leave both, add a single-line annotation (`"same key, shown here for reference"`).

**Persona-suggested:** Option A.

#### Key data available
- `helpoverlay.go:33-49` ACTIONS section and conditional lines (`ShowConditionsHint`, `ShowEventsHint`, `ShowDeleteHint`)
- `helpoverlay.go:51-59` VIEW section
- No other key appears in two sections today.

#### Acceptance criteria (draft)

| # | Axis | Criterion |
|---|------|-----------|
| AC1 | Observable | `[d]` appears in exactly one section of the rendered help overlay. |
| AC2 | Input-changed | Before vs after: stripped help-overlay View() substring `"[d]"` count decreases from 2 to 1. |
| AC3 | Anti-regression | `[d]` still functional — describe pane opens on `[d]` press from resource table. |
| AC4 | Anti-regression | Conditional lines (`[Shift+C]`, `[Shift+E]`, `[x]`) still appear/suppress based on their flags. |
| AC5 | Integration | `go install ./...` + `go test ./internal/tui/... -count=1` green. |

#### Non-goals
- Not restructuring the 4-column help overlay layout.
- Not renaming ACTIONS or VIEW section headings.
- Not changing `[d]` keybinding behavior.

**Maps to:** FB-072/100/101/104 batch user-persona P3-1.

---

### FB-115 — Quota loading hint copy contradicts FB-099/FB-100 cancel affordance

**Status: ACCEPTED + PERSONA-EVAL-COMPLETE 2026-04-20** — user-persona eval 2026-04-20 returned 0 P1/P2/P3 + 1 positive finding. Persona confirmed: `"press [3] to cancel"` parses correctly in context — full hint `"Quota dashboard loading… press [3] to cancel"` reads as a single coherent instruction; pairs symmetrically with FB-097's `"Quota dashboard ready — press [3]"` ready-prompt (loading → cancel; ready → confirm). Ambiguity between "wait and press when ready" vs "press now to abort" eliminated. Fix resolved the originally filed confusion without introducing new issues. **Prior Status: ACCEPTED + COMMITTED + PENDING PERSONA-EVAL 2026-04-20** — engineer committed `ecf80aa` on `feat/console`: _"Loading-quota hint now matches the cancel button — both say 'cancel'"_. Strict one-commit-per-feature rule honored (genuinely-new production change, not retroactive-closeout exception). **Prior Status: COMMITTED TO FEAT/CONSOLE 2026-04-20** — test-engineer PASS 2026-04-20: `go build ./internal/tui/...` clean; 6/6 `TestFB115_*` green; production code verified (`model.go:1232` reads `"Quota dashboard loading… press [3] to cancel"` — tail-swap confirmed); `"when ready"` residue check — only occurrences are in FB-115 test negation checks and AC6 old-anchor check (`model_test.go:14118`, `14221`) — no stale production usage, no test anchoring old string as expected; full suite + `go install ./...` green. **AC6 model-state check accepted (spec-authorized):** AC6 accesses `appM.statusBar.Hint` directly rather than `statusBarNorm()`. Policy requires View() for Observable ACs; AC6 is Anti-regression anchoring spec §4's intentional string change. AC1 and AC3 already cover the rendered output; AC6 pins the exact stored value at the model layer. Analogous to FB-096 AC7 `pendingQuotaOpen` exception pattern. **AC3 three-check rigor clean.** **Prior Status: PENDING TEST-ENGINEER 2026-04-20** — engineer delivered Option A implementation: `internal/tui/model.go` line 1232 tail-swap `"when ready"` → `"to cancel"`. 5 FB-079 anchor occurrences in `model_test.go` updated to new copy per spec §4 (intentional, not regression). 6 `TestFB115_*` tests green: AC1 Observable hint contains "to cancel" (absent "when ready"); AC2 Observable hint + strip aligned (both show "cancel" simultaneously); AC3 Input-changed state-transition at `model_test.go:14141` uses three-check pattern (before-press per-leg negative + diff + after-press per-leg positive on both "Quota dashboard loading" and "to cancel" substrings) — same rigor as FB-099 AC3 fix + FB-105 AC5; AC4 Anti-regression FB-097 ready-prompt unchanged; AC5 Anti-regression FB-080 cancel-confirmation copy unchanged; AC6 Anti-regression FB-079 anchors updated (intentional per spec). Anti-behavior marked N/A (copy change, no conditional behavior). Integration confirmed via full `./internal/tui/...` + `go install ./...` green. **Prior Status: PENDING ENGINEER 2026-04-20** — ux-designer spec delivered 2026-04-20 at `docs/tui-ux-specs/fb-115-quota-loading-hint-copy.md`. Decision: Option A — rewrite `model.go:1232` from `"Quota dashboard loading… press [3] when ready"` to `"Quota dashboard loading… press [3] to cancel"` (tail-swap only: `"when ready"` → `"to cancel"`). Rationale over B (silent hint): silent hint during loading leaves status bar blank after operator-initiated action — worse signal than a loading message. Rationale over C (minimalist): Option A makes hint and strip (FB-099) mutually reinforcing, not just contradiction-avoidant. Preserves `"Quota dashboard"` prefix to stay consistent with ready-prompt family (`loading…` / `ready —` / `cancelled`). Spec includes explicit §4 calling out FB-079 test-anchor updates as intentional. **Prior Status: PENDING UX-DESIGNER 2026-04-20** — designer-call required on replacement copy. Filed 2026-04-20 by product-experience from FB-072/100/101/104 user-persona batch P3 finding. FB-099 explicitly deferred hint-copy revision as a "revisit if FB-097 ships" upgrade path (see FB-099 spec §Option C rationale); FB-097 has shipped (2026-04-20) so the deferral condition is now satisfied.

**Priority: P3** — affordance-discoverability contradiction. Operator sees two simultaneous framings about what `[3]` does during quota loading, and the hint's "when ready" copy directly contradicts the strip's "cancel" label. Attentive operator trusts strip (correct) but hesitates momentarily.

#### User problem

During quota loading (`pendingQuotaOpen=true`):

- **Status bar hint (`model.go:1232`):** `"Quota dashboard loading… press [3] when ready"` — implies "wait for load, then press [3] to confirm open."
- **Resource table keybind strip (`resourcetable.go:727-729`):** `[3] cancel` — implies "press [3] NOW to abort."

Both are simultaneously visible. The hint's `"press [3] when ready"` was written under FB-079 before FB-080 added the second-press cancel gesture. FB-099 designer explicitly chose Option C (strip-only affordance) over Option A (copy swap) because FB-079 was load-bearing and FB-097 persistent ready-prompt was not guaranteed to ship. Now FB-097 has shipped, so the hint surface is owned by FB-097's `"⚡ Quota dashboard ready — press [3]"` post-load text — and the pre-load copy is free to drop the "when ready" framing.

#### Designer-call

Where does the loading-window hint sit in the FB-079/FB-097/FB-099 triad?

- **Option A** — Rewrite `model.go:1232` hint to `"Loading quota… press [3] to cancel"`. Aligns hint with strip; pre-load hint explicitly advertises cancel gesture.
- **Option B** — Drop the hint entirely during loading. Strip carries the cancel affordance; hint stays silent until FB-097 posts the ready-prompt on load completion. Minimalist.
- **Option C** — Keep "loading" framing but drop the `press [3] when ready` tail: `"Quota dashboard loading…"`. Hint announces state only; strip announces affordance.

**Considerations:**
- FB-079 test anchors target the exact `"Quota dashboard loading… press [3] when ready"` string — whichever option is chosen must account for FB-079 test-anchor updates.
- FB-100 persona-eval explicitly described the intended hint as `"press [3] to cancel"` in the eval request — Option A matches that intent.
- FB-100 spec was ACCEPTED without an explicit hint-copy change (strip-only was the chosen scope). This is a genuine post-acceptance gap, not a regression.

#### Key data available
- `internal/tui/model.go:1232` — first-press hint post site.
- `internal/tui/components/resourcetable.go:727-729` — strip substitution logic (FB-099).
- FB-079 tests in `internal/tui/model_test.go` — may need anchor updates.
- FB-097 ready-prompt at load completion (`BucketsLoadedMsg` handler) — post-load hint already handled.

#### Acceptance criteria (draft — pending designer option)

| # | Axis | Criterion |
|---|------|-----------|
| AC1 | Observable | Pre-load hint rendered during `pendingQuotaOpen=true` does NOT contain `"when ready"`. |
| AC2 | Observable | Pre-load hint and strip label are semantically aligned (both describe cancel gesture OR hint is silent about `[3]`). |
| AC3 | Input-changed | Before `[3]` press: no pending hint. After `[3]` press (loading): hint matches new copy. |
| AC4 | Anti-regression | FB-097 post-load ready-prompt unchanged (`"⚡ Quota dashboard ready — press [3]"`). |
| AC5 | Anti-regression | FB-080 cancel path still posts `"Quota dashboard cancelled"` on second-press. |
| AC6 | Anti-regression | FB-079 tests updated to new copy (intentional anchor change, documented in spec §Intentional test updates). |
| AC7 | Integration | `go install ./...` + full suite green. |

#### Non-goals
- Not changing FB-097 persistent ready-prompt copy.
- Not changing FB-080 cancel-confirmation copy.
- Not changing FB-099 strip-label substitution logic.

**Maps to:** FB-072/100/101/104 batch user-persona P3-2. Follows-up FB-099 deferred upgrade path.

---

### FB-116 — Orientation hint promises "quick-jump key below" to operators who have no quick-jump keys

**Status: ACCEPTED + PERSONA-EVAL-COMPLETE 2026-04-20** — user-persona eval 2026-04-20 returned 0 P1/P2/P3 + 1 positive finding. Persona confirmed: false "or use a quick-jump key below" clause gone; new operator in empty project sees one calm directive pointing at the sidebar (the one always-present always-correct affordance in that state); `"to get started"` lands as a natural first-time-instruction closing phrase (stateless, accurate regardless of what loads after); no new interaction-induced confusion (`line3` hint location + registration-populated suppression keep the surface clean). Fix resolved the originally filed false-promise confusion exactly. **Copy-polish batch (FB-114 + FB-115 + FB-116) formally closed.** **Prior Status: ACCEPTED + COMMITTED + PENDING PERSONA-EVAL 2026-04-20** — engineer committed `b036d91` on `feat/console`: _"First-time project view no longer promises quick-jump keys that aren't there"_. Strict one-commit-per-feature rule honored (genuinely-new production change, not retroactive-closeout). **Prior Status: COMMITTED TO FEAT/CONSOLE 2026-04-20** — test-engineer PASS 2026-04-20: `go build ./internal/tui/...` clean; 6/6 `TestFB116_*` green; full suite + `go install ./...` green. Production code verified at `resourcetable.go:644` — false clause absent from source. **FB-105 anchor independence verified:** test-engineer grep confirmed `"quick-jump key below"` appears only in FB-116 test negation checks (no FB-105 test relied on trailing false clause). Ux-spec §4 anchor-update prediction unnecessary; engineer's diff-simplification discovery correct. **AC3 axis-mislabel ruling — accept as-is:** same call as FB-114 AC2. Static copy-change brief with no state-transition behavior; real empty→populated transition coverage already exists in FB-105 tests at `resourcetable_test.go:2358-2361`. AC3 three-assertion post-fix snapshot (new pos + old neg + base pos) is genuine regression detector, not pass-through. No relabel cycle warranted. **Next:** engineer commits to `feat/console` as its own product-prose commit (strict one-commit-per-feature rule — genuinely-new production change, not retroactive-closeout). This closes the copy-polish batch per team-lead direction. **Prior Status: PENDING TEST-ENGINEER 2026-04-20** — engineer delivered Option A implementation: `internal/tui/components/resourcetable.go:644` substring swap `"→  select a resource type from the sidebar, or use a quick-jump key below"` → `"→  select a resource type from the sidebar to get started"`. 6 `TestFB116_*` tests green (AC1 Observable false-clause-absent via `!strings.Contains(got, "quick-jump key below")`; AC2 Observable full new directive present; AC3 "Input-changed" — post-fix snapshot style, new copy + old clause absent + base directive intact; AC4 Anti-regression FB-105 anchor; AC5 Anti-regression forceDashboard suppression; AC6 Anti-regression org-scope suppression). Anti-behavior marked N/A with justification (static copy change). Integration confirmed via full `./internal/tui/...` + `go install ./...` green. **FB-105 anchor discovery — spec §4 prediction was unnecessary:** engineer verified all 6 existing FB-105 anchor references use prefix `"select a resource type from the sidebar"` which still matches new copy; no anchor updates needed. Good catch that simplified the diff over spec expectation. **Pre-submission note flagged to test-engineer:** AC3 labeled "Input-changed" but is effectively a post-fix-state snapshot (same pattern as FB-114 AC2). Brief AC3 criterion ("hint stable across empty/populated registrations") was poorly scoped for a stateless Option A — the genuine state-transition behavior (empty→populated suppression) is already covered by existing TestFB105_AC3-family tests at `resourcetable_test.go:2358-2361`. Test-engineer to decide accept-as-is vs relabel. **Prior Status: IN-PROGRESS (engineer) 2026-04-20** — routed to engineer 2026-04-20 after FB-114+FB-115 feat/console commits landed. Ux-designer spec delivered 2026-04-20 at `docs/tui-ux-specs/fb-116-orientation-hint-false-quickjump.md`. Decision: Option A — `"→  select a resource type from the sidebar to get started"` (single substring changed at `resourcetable.go:644`). Rationale over B: Option B's post-registration phase provides zero marginal information — when `len(m.registrations) > 0`, `renderQuickJumpSection()` already renders quick-jump keys visibly below the hint; two-phase copy would restate what the operator already sees. Option B also requires expanding gate condition with internal branching, adding complexity for no net signal gain. Hint is most valuable as calm single directive at maximum-disorientation (zero registrations); once registrations arrive interface communicates the rest. Spec includes explicit §4 calling out FB-105 test-anchor update as intentional (same pattern as FB-115 → FB-079). ACs cover Observable ×2 (false clause absent, correct directive present), Input-changed, Anti-regression ×3 (FB-105 anchor, forceDashboard suppression, org-scope suppression), Integration. **Prior Status: PENDING UX-DESIGNER 2026-04-20** — designer-call required on copy fix (Option A minimal strip vs Option B stateful two-phase hint). Filed 2026-04-20 by product-experience from FB-105 user-persona P2 finding. Persona interpretation verified in code.

**Priority: P2** — fires at the worst UX moment: a new operator's first view of a project, when they're most disoriented and the hint is specifically designed to orient them. Offering a false second option invites the operator to look for something that isn't there; the most likely interpretation when nothing appears below is "something didn't load" rather than "the hint was wrong."

#### User problem

S1 orientation hint at `internal/tui/components/resourcetable.go:644` reads:

```
→  select a resource type from the sidebar, or use a quick-jump key below
```

The hint fires **only** when `m.tuiCtx.ActiveCtx.ProjectID != "" && len(m.registrations) == 0`.

The quick-jump section at `renderQuickJumpSection()` (resourcetable.go:406-418) iterates `quickJumpTable` and filters via `hasRegistrationMatch(e.matchSubstrs)`. `hasRegistrationMatch` (resourcetable.go:561-571) iterates `m.registrations`. When `m.registrations` is empty — the condition that triggers the orientation hint — `hasRegistrationMatch` always returns `false`, `entryStrings` stays empty, and `renderQuickJumpSection` returns `""`.

**The hint and quick-jump keys are mutually exclusive by construction.** The hint fires only when registrations are empty; quick-jump rows require at least one registration. They cannot coexist.

Operator-visible impact: a first-time arrival in a project sees the hint, scans below looking for `[n] networks` / `[d] domains` / etc., finds nothing, and concludes "the UI is broken" or "data is still loading" — neither interpretation correct.

#### Designer-call

- **Option A (minimal, stateless)** — strip the secondary clause entirely. Single-directive hint:
  > `→  select a resource type from the sidebar to get started`
  
  Simplest diff; no state machine; always correct regardless of registration state. Persona-suggested.

- **Option B (stateful, two-phase)** — make the hint reactive to registration arrival:
  - When `len(m.registrations) == 0`: `→  select a resource type from the sidebar to get started`
  - When `len(m.registrations) > 0` AND still on welcome panel: `→  select from the sidebar or use a quick-jump key below`
  
  Richer signal once quick-jump becomes available; requires the hint to re-render on registration arrival (harmless — `renderHeaderBand` is called on every View() tick).

- **Option C (remove hint entirely when registrations empty)** — rely on quick-jump's absence speaking for itself. Rejected out-of-hand: the hint's primary value is orienting operators in empty-registration state; removing it defeats the FB-105 S1 purpose.

**Considerations:**
- Option A is the minimum viable correctness fix (removes the false promise).
- Option B is a nice-to-have that provides guidance at the moment quick-jump becomes useful — but the operator who made it that far has already successfully oriented via sidebar OR quick-jump.
- Persona preference is Option A for simplicity; team-lead should weigh whether the Option B copy re-activation justifies the state-swap complexity.

#### Key data available
- `m.registrations` — `[]data.ResourceRegistration`, populated by project-bound registration loader (FB-014 / FB-042).
- `m.tuiCtx.ActiveCtx.ProjectID` — context-scope gate.
- `renderHeaderBand()` at line 643 — current hint-render site.
- `renderQuickJumpSection()` at line 406 — quick-jump filter site.
- `hasRegistrationMatch()` at line 561 — registration filter.

#### Acceptance criteria (draft — tune per designer option)

| # | Axis | Criterion |
|---|------|-----------|
| AC1 | Observable | Orientation hint rendered when `len(m.registrations) == 0` + project context does NOT contain `"quick-jump key below"`. |
| AC2 | Observable | Option A: hint contains `"select a resource type from the sidebar"`. Option B: same positive, plus alternate copy when registrations populated. |
| AC3 | Input-changed | Option A: hint stable across empty/populated registrations. Option B: hint copy changes across `SetRegistrations([])` → `SetRegistrations([...valid...])` transition; per-leg positive assertions for both copies. |
| AC4 | Anti-regression | FB-105 AC1 test still green (orientation hint present when triggered). Test-anchor update expected in FB-105 tests that match the old full string. |
| AC5 | Anti-regression | FB-054 forceDashboard branch suppression intact (hint still suppressed when `forceDashboard=true`). |
| AC6 | Anti-regression | Org-scope suppression intact (hint still absent when ProjectID empty). |
| AC7 | Integration | `go install ./...` + full suite green. |

#### Non-goals
- Not touching `renderQuickJumpSection()` filter logic.
- Not removing the hint entirely when registrations populate (it still has orientation value during loading race).
- Not changing the `→` leader character.

**Maps to:** FB-105 user-persona P2-1.

---

---

### FB-117 — End-to-end test harness for `datumctl tui`

**Status: PENDING** — filed 2026-04-20 by team-lead per user request.

**Priority: P2** — the TUI has ~400 unit tests but zero end-to-end coverage. No test verifies the TUI works against a real Datum Cloud session, real terminal rendering, or real keyboard interaction chains. All existing tests mock the API layer.

#### User problem

Unit tests confirm model behavior in isolation but cannot catch:
- Real API auth/context failures at startup
- Lipgloss rendering regressions at actual terminal widths
- Keyboard interaction sequences that span multiple model updates
- Session lifecycle: login → context selection → resource navigation → detail view
- The welcome panel, quota dashboard, and activity dashboard in a live environment

The `docs/tui-e2e-requirements.md` defines 30+ REQ-TUI-XXX scenarios that have never been executed by automation.

#### Proposed approach

Build a VHS-based e2e harness:
- [VHS](https://github.com/charmbracelet/vhs) records terminal sessions as `.tape` files and produces GIF/MP4 output; assertions can be made on screenshot frames or terminal output captures.
- One `.tape` file per REQ-TUI-XXX scenario, stored at `tests/e2e/tui/`.
- CI step: `vhs tests/e2e/tui/*.tape` + diff against golden screenshots or text output.
- Requires a live Datum Cloud environment with seeded resources (test org/project).

#### Alternative approach (lighter weight)

Use `expect`/`pexpect` or a Go PTY harness (`github.com/creack/pty`) to drive `datumctl tui` stdin/stdout programmatically, assert on ANSI output after each keypress sequence. No VHS dependency; runnable in CI with a real API token.

#### Acceptance criteria

1. At minimum 5 REQ-TUI scenarios from `docs/tui-e2e-requirements.md` have automated coverage covering: startup, sidebar navigation, resource table load, detail view open, context switcher.
2. Tests run against a real Datum Cloud test environment (not mocks).
3. Tests are gated in CI (or documented as manual gate with reproduction steps).
4. A `README` or `docs/tui-e2e-runbook.md` explains how to run locally and how to update golden output.

#### Non-goals
- 100% REQ-TUI coverage in the first pass — 5 core flows is the target.
- Replacing existing unit tests.
- Testing mutation operations (TUI is read-only).

**Maps to:** `docs/tui-e2e-requirements.md` (all REQ-TUI-XXX).

---

### FB-118 — Describe error card wins over eventsMode in double-failure state → FB-086 recovery path shows wrong body

**Status: ACCEPTED 2026-04-20** — spec at `docs/tui-ux-specs/fb-118-describe-error-eventsmode-guard.md`. Implementation: one-word guard at `model.go:2069` (`&& !m.eventsMode` added to describe error block predicate) + `newDoubleFailureDetailModel` fixture tightened at `model_test.go:14319` with `loadState=LoadStateError`, `lastFailedFetchKind="describe"`, `loadErr=errors.New("describe fetch failed")`. Tests: `TestFB118_AC1/AC2/AC3` (Observable: events spinner/table/error when eventsMode=true), `TestFB118_AC4` (Anti-regression: describe card still shows when eventsMode=false), `TestFB118_AC5` (Anti-regression: fixture fields set), `TestFB118_AC6` (Anti-regression: FB-038 pre-check unaffected). Test-engineer gate PASS. Product-experience independent verification: 6 tests PASS, `go install ./...` clean, code matches spec. Next: engineer commits FB-118 to `feat/console`. Prior: written 2026-04-20 by product-experience after user-persona FB-085/FB-086 eval raised P2. Follow-up completion of FB-086 (`fd91904`).

**Priority: P2** — FB-086's admission fix is honest, but the body rendering path is still wrong. The operator presses [E], lands in eventsMode, and sees the describe error card instead of events loading/error. The affordance succeeds; the destination misleads.

#### User problem

In double-failure state (describe fetch settled-failed, events fetch settled-failed), pressing [E] correctly admits and sets `m.eventsMode = true` (FB-086 fix). But `buildDetailContent()` at `internal/tui/model.go:2069` checks the describe error block **before** the `m.eventsMode` branch (line 2104), and the describe error block has no `!m.eventsMode` guard. Order of checks:

1. FB-038 loading placeholder — guarded by `!m.eventsMode`, skipped
2. FB-024/051/052 placeholder — guarded by `!m.eventsMode && events != nil`, skipped
3. **FB-005 describe error block** — `m.loadState == LoadStateError && m.lastFailedFetchKind == "describe"` — FIRES
4. yamlMode / conditionsMode / eventsMode — never reached

Consequences:
- **During events re-fetch after [E]:** mode label says "events," body shows the describe error card. The operator pressed [E] to retry events; the body shows a describe error, not a loading indicator.
- **If the re-fetch fails again:** same result — describe error card fills the pane, events error card at `components.RenderEventsTable(eventsErr)` never renders. No surface communicates why events failed.
- **If the re-fetch succeeds:** describe error card still wins over the rendered events table. The [E] affordance is reachable but the rendered body is unreachable.

Also: the AC7 test (`TestFB086_AC7_EKey_DoubleFailure_ThenEventsLoaded_ViewTransition`) passes because the `newDoubleFailureDetailModel` fixture doesn't set `loadState`/`lastFailedFetchKind`/`loadErr`. Real field state in double-failure has describe settled-failed (loadState = LoadStateError, lastFailedFetchKind = "describe", loadErr set). The fixture modeled "describe nil + events err" but not "describe failed + events failed" — which is the state the user actually hits.

#### Designer call

Preferred approach is to add `&& !m.eventsMode` to the describe error block admission at line 2069 — mirroring the guards already present on the FB-038 and FB-051/052 branches. Once the operator has entered events mode, the events surface owns the body; the describe error is communicated by whatever mode label the title bar is carrying (e.g., describe-error context still visible in status bar / mode transitions).

Open designer questions:
- Should the describe error carry a breadcrumb on the events surface when events also fail? (E.g., "describe also failed" badge in the events error card.) Or is title-bar context enough?
- Tightening the `newDoubleFailureDetailModel` fixture to reflect true double-failure state: set `loadState = LoadStateError`, `lastFailedFetchKind = "describe"`, `loadErr = errors.New("describe fetch failed")` alongside the existing `eventsErr` field. New Observable AC must assert `stripANSI(View())` contains events loading/error content and does **not** contain the describe error card when `eventsMode=true`.

#### Dependencies

- FB-086 ACCEPTED (`fd91904`) — admission fix in place.
- FB-038 ACCEPTED (`bb609ce`) — loading placeholder uses `!m.eventsMode`; describe error block should match.
- FB-051/052/005 — surrounding placeholder and error-card logic.

#### Non-goals

- Not re-opening FB-086; the admission is correct. This is completion of the rendered destination.
- Not changing FB-005 error card copy or dimensions.
- Not touching single-failure paths.

---

### FB-119 — Describe-requiring keybind hints (`[y]`, `[C]`) remain visible in double-failure state

**Status: ACCEPTED 2026-04-20** — spec at `docs/tui-ux-specs/fb-119-describe-hint-gate.md`. Implementation: `detailview.go:28` (field), `detailview.go:110-111` (setter), `detailview.go:124` (getter), `detailview.go:150` (gate in `titleBar()`) + 6 AppModel integration points (`model.go:347` set true on DescribeResultMsg; `model.go:615, 1538, 1672, 1710, 1820` set false on 5 describeRaw=nil sites; `model.go:2208` preserves across state restoration). Tests: 7 tests (Observable × 3, Input-changed × 1, Anti-regression × 3) — `TestFB119_AC1` through `TestFB119_AC7`. Test-engineer gate PASS. Product-experience independent verification: all 7 tests PASS including AC4 sub-tests, `go install ./...` clean, code matches spec. Next: engineer commits FB-119 to `feat/console`. Prior: written 2026-04-20 by product-experience after user-persona FB-085/FB-086 eval raised P3.

**Priority: P3** — minor friction, but newly prominent via FB-086's navigable double-failure path.

#### User problem

In double-failure state (describe nil, events nil, eventsErr set, describe settled-failed), the DetailPane title-bar keybind strip at `detailview.go` renders `[y] yaml  [C] conditions  [E] events  [x] delete  [Esc] back`. Both `[y]` and `[C]` require `m.describeRaw != nil` to produce meaningful content — yaml rendering and conditions rendering both depend on the parsed unstructured object. In double-failure, pressing `[y]` or `[C]` toggles the mode flag but the rendered body is empty/unchanged.

Pre-existing behavior. FB-086 promoted this state from "unreachable dead end" to "navigable via [E]," which makes the non-functional hints more visible: an operator who lands in double-failure and tries `[y]` after `[E]` arrives in a confusing state (blank yaml view or describe-error-card depending on FB-118 resolution).

#### Designer call

Preferred approaches:
- **A (gate):** suppress `[y]` and `[C]` hints when `m.describeRaw == nil`. Keep `[E]` and `[Esc]`. Hint strip communicates what actually works.
- **B (inert-with-tooltip):** render hints but style muted + status-bar message on press ("describe unavailable — yaml/conditions require a successful describe fetch").
- **C (do nothing):** treat as acceptable noise.

A mirrors the general principle "affordances must not lie." B preserves discoverability for recovery scenarios where describe succeeds after the user has been shown the hints. C is cheap but perpetuates the honesty gap FB-086 fixed.

Open designer questions:
- Should placeholder surfaces (FB-024/051/052) also suppress these hints in their own action rows? The placeholder action row already only renders `[E]` and `[Esc]` (plus `[r]` for retry) — it's the title-bar strip that's leaking the full hint set. Scope: title-bar + help overlay alignment.
- Does suppression also apply to `[x] delete` when `describeRaw == nil`? (Delete is a separate mutation path; may or may not require describe depending on how the mutation flow constructs the payload.)

#### Dependencies

- FB-118 (describe error card guard) — same code area, same test fixture tightening.
- FB-026 (keybind hint format consistency) — currently IN-PROGRESS; FB-119 is a separate thesis (conditional visibility) not a format fix.

#### Non-goals

- Not changing `[E]` or `[Esc]` visibility.
- Not changing keybind format (FB-026 scope).
- Not changing the describe-required check on the `[y]`/`[C]` handler side; they already no-op usefully when `describeRaw == nil`. Scope is discoverability-surface honesty.

---

### FB-120 — Dual loading signals on initial DetailPane open don't distinguish describe-vs-events fetch

**Status: DEFERRED** — written 2026-04-20 by product-experience after user-persona FB-038 eval. Persona explicitly framed as a note for future dual-fetch UX revisit, not an independent actionable item.

**Priority: P3** — minor; the blank-body regression FB-038 fixed was clearly worse. Signals aren't wrong, just not specific.

#### User problem

On initial DetailPane open with both fetches in-flight, the operator sees:
- Title bar: `⟳ loading…` (describe spinner from `DetailViewModel.titleBar()`)
- Body: `Loading…` (FB-038 placeholder, muted)

Both signals are accurate but not differentiated. Transition behavior:
- **Describe returns first, events still loading:** body switches to describe content; the body `Loading…` disappears. Operator never learns events were still pending.
- **Events returns first, describe still loading:** body switches to the FB-024/051/052 "Describe unavailable — only events loaded" placeholder. The transition from generic `Loading…` to that placeholder can feel abrupt.

#### Why deferred

Persona self-limited the finding: "This is minor — blank was clearly worse. The signals aren't wrong, just not specific. Worth noting if a future brief ever revisits the dual-fetch loading UX."

No active designer-call. Keep filed for when a future dual-fetch UX pass wants the signal.

#### Dependencies

- FB-038 ACCEPTED (`bb609ce`) — loading placeholder; FB-120 would refine it.
- FB-024/051/052 — placeholder surface that the events-first transition lands on.

#### Non-goals (if ever promoted)

- Not re-opening FB-038.
- Not changing the title-bar spinner semantics.

---

### FB-121 — HelpOverlay ACTIONS column: `[C]` and `[E]` labels misaligned 2 columns left of siblings

**Status: ACCEPTED 2026-04-20 — committed `d9f29f1` on `feat/console`** ("Help overlay [C] and [E] rows align with their siblings"). Spec at `docs/tui-ux-specs/fb-121-helpoverlay-actions-column-alignment.md`. Option A (2-space → 4-space) shipped. Implementation: `components/helpoverlay.go:40` (`[C]    conditions`) + `:43` (`[E]    events`). Assertion migrations per spec §2 all landed: `helpoverlay_test.go:67/70/85/111/129/146` + `conditions_test.go:419/434` + `model_test.go:6637/6654/6668 (AC#22)` + `:7430/7450/7468 (AC#26)` — all moved from 2-space to 4-space form. Bonus defensive negative assertions added at `helpoverlay_test.go:132/149` to assert 2-space form must NOT appear. Tests: 4 top-level functions, 6 leaf cases (AC3 has 2 sub-tests for Conditions/Events gating). Engineer-produced axis-coverage table — upgraded on 3rd iteration to defend **Input-changed as COVERED** (not N/A): AC1+AC2 pre/post `ShowConditionsHint=true` delta (2-space form absent + 4-space form present asserts the padding change itself is the observable-delta). Repeat-press remains N/A (no call-count state; fix is a string literal; no key involved). Test-engineer gate PASS on second submission after push-back 1 for missing table + improvements through axis-classification upgrade. Product-experience independent verification: 4 tests PASS, `go test ./internal/tui/...` green, `go install ./...` clean, spec §1 C1/C2 code diff verified, commit `d9f29f1` verified on origin/feat/console. Prior state: **PENDING PRODUCT-EXPERIENCE (push-back 1)** — test-engineer's first submission to me had no axis-coverage table; rejected at pre-submission gate; re-routed engineer to produce one. Prior: written 2026-04-20 by product-experience after user-persona FB-026 eval raised P3. Spec note: Option A selected (2-space → 4-space); queued after FB-119. **Coordination note from designer:** pre-existing `[C]  conditions`/`[E]  events` 2-space assertions in `model_test.go` (at lines ~6636/6653/6667/7429/7449/7467 — AC#22 and AC#26 from TUI bootstrap) MUST migrate to 4-space form; §2 table in spec enumerates all affected assertion strings.

**Priority: P3** — minor visual roughness introduced by FB-026's format normalization. Content is correct; only column alignment is off.

#### User problem

In `internal/tui/components/helpoverlay.go`, the ACTIONS section mixes key widths with inconsistent padding:

```go
row.Render("[Enter] select"),           // [Enter] = 7 chars + 1 space = label at col 8
row.Render("[/]    filter"),            // [/] = 3 chars + 4 spaces = label at col 7
row.Render("[Esc]  back / home"),       // [Esc] = 5 chars + 2 spaces = label at col 7
row.Render("[C]  conditions"),          // [C] = 3 chars + 2 spaces = label at col 5  ← outlier
row.Render("[E]  events"),              // [E] = 3 chars + 2 spaces = label at col 5  ← outlier
row.Render("[x]    delete resource"),   // [x] = 3 chars + 4 spaces = label at col 7
```

Rendered:
```
[Enter] select
[/]    filter
[Esc]  back / home
[C]  conditions       ← 2 columns left of siblings
[E]  events           ← 2 columns left of siblings
[x]    delete resource
```

FB-026 brought `[C]` and `[E]` into the ACTIONS section by normalizing them from `[Shift+C]` / `[Shift+E]`, but the padding wasn't adjusted to match the section's column-7 alignment. The previous `[Shift+C]` (10-char prefix = label at col 11) was misaligned in the other direction; FB-026 fixed the format but introduced a new misalignment.

#### Designer call

Two options:
- **A (match ACTIONS column-7):** bump `[C]` and `[E]` to 4 spaces each — `[C]    conditions`, `[E]    events`. Matches `[/]` and `[x]` siblings in this section. No changes elsewhere.
- **B (rework to uniform column-5 across the overlay):** VIEW and GLOBAL sections already use column-5 (3-char key + 2 spaces). ACTIONS is the odd section with column-7. B would mean pulling `[/]`, `[Esc]`, `[x]` down to column 5 — requiring revisiting `[Enter]` (7-char key, natural landing at col 8).

A is the minimum viable fix — one line change on two rows.
B is a larger restructure that addresses cross-section inconsistency, but that inconsistency pre-dated FB-026 and is not the user-surfaced complaint.

Recommended: A. The ACTIONS column-7 convention was the established baseline; FB-026's new entries just need to meet it.

Open designer questions:
1. Is there a title-bar strip analog? (Checked: title-bar uses single-space separators between hint tokens, not columnar alignment — no analog.)
2. Any other overlay sections have the same bug? (Checked: VIEW and GLOBAL are internally consistent at column 5; NAV section at the top uses `[↑↓]` 4-char key + 2 spaces; they're self-consistent.)

#### Dependencies

- FB-026 ACCEPTED (`0366a36`) — this brief is a direct follow-up; FB-026's normalization set the stage.

#### Non-goals

- Not revisiting `[Shift+K]` → `[K]` format (FB-026 done).
- Not changing the ACTIONS / VIEW / GLOBAL section structure.
- Not touching title-bar hint strip.

---

### FB-122 — Events age label persists during in-flight re-fetch (title-bar says "4m ago" while body shows loading spinner)

**Status: ACCEPTED 2026-04-20** — product-experience ruling. Spec at `docs/tui-ux-specs/fb-122-events-age-label-suppress-during-loading.md` (Option A — suppress age label when `eventsLoading=true`). Engineer delivered: `eventsLoading bool` field at `detailview.go:30`, `SetEventsLoading(bool)` setter + `EventsLoading()` getter at lines 118–119, guard update at line 169 (`if m.mode == "events" && !m.eventsFetchedAt.IsZero() && !m.eventsLoading`), and 13 `m.detail.SetEventsLoading(...)` call sites in `model.go` (true on resource-entry paths ~1180/1342/1698/1737/1849 and r-key re-fetch; false on EventsLoadedMsg/context-switch/Esc-back/errors at ~361/623/1473/1487/1536/1554/1767). All 5 tests pass with `stripANSI(dv.View())` + `strings.Contains`/`!strings.Contains` assertions — no model-field-only inspection. Axis coverage: Observable × 2 (AC1 label absent when loading, AC2 present when not), Input-changed (AC3 toggle true→false changes View()), Anti-regression × 2 (AC4 non-events modes unaffected across `""`/`"yaml"`/`"conditions"` subtests, AC5 zero fetchedAt + cross-check), Anti-behavior N/A (pure render suppression, no keypress to block — defense accepted), Repeat-press N/A (stateless render function — defense accepted), Integration (`go install` + full suite green). Filed 2026-04-20 from user-persona FB-025 evaluation (P3-1).

**Priority: P3** — cosmetic mismatch lasting one fetch cycle. The title bar keeps rendering `events · 4m ago` for the old `eventsFetchedAt` while `eventsLoading=true` and the body shows `⟳ Loading events…`. Short-lived, but incoherent: title and body briefly tell different stories about the same surface.

#### User problem

Operator saw "4m ago", pressed `[r]` to refresh *because* of that signal. The label persists through the in-flight fetch, then snaps to "just now" once the fetch resolves. During the loading window the label is actively misleading — it tells the operator the events they're looking at are 4m old, but they aren't looking at *any* events right now (body is in the loading branch).

#### Designer call

Two options to consider:
- **A (suppress age label when `eventsLoading=true`):** during the loading window, title-bar segment drops back to just `events` (no `· age`). Restores on fetch completion. Consistent with how the hint row already suppresses under loading.
- **B ("refreshing…" replacement segment):** title-bar segment becomes `events · refreshing…` for the loading window. Explicitly narrates state, but adds a new copy string.

Designer should pick. A is minimum-footprint; B more explicitly tells the story. The age label is computed at render time from `eventsFetchedAt`, so either option is a single conditional in `detailview.go:169-170`.

#### Dependencies

- FB-025 ACCEPTED (`9e310e3`) — this is a direct follow-up to the age label FB-025 introduced.

#### Non-goals

- Not touching the body-side spinner (FB-019 territory).
- Not changing the `eventsFetchedAt` reset semantics (FB-025 already covers these).
- Not touching the stale-empty recovery copy (FB-025 already diverges at ≥5m).

---

### FB-123 — No inline `[r] refresh` hint in DetailPane title bar while in events mode

**Status: ACCEPTED 2026-04-20** — product-experience ruling. Spec at `docs/tui-ux-specs/fb-123-events-mode-refresh-hint.md` (Option A — always-on when events loaded). Engineer delivered: 3-line `rHint` block at `detailview.go:168-172` matching spec §1.2 exactly — `rHint := ""; if m.mode == "events" && !m.eventsFetchedAt.IsZero() && !m.eventsLoading { rHint = "  [r] refresh" }; hintRow += "  " + eHint + rHint + "  [x] delete  [Esc] back"`. All 7 tests pass with `stripANSI(dv.View())` + `strings.Contains`/`!strings.Contains` assertions. Axis coverage: Observable × 3 (AC1 hint present when loaded + not refreshing, AC2 absent during re-fetch, AC3 absent before first load), Input-changed (AC4 zero→non-zero fetchedAt toggles View()), Anti-regression × 3 (AC5 non-events modes across `""`/`"yaml"`/`"conditions"` subtests, AC6 position `[E] describe < [r] refresh < [x] delete`, AC7 `describeAvailable=false` does NOT suppress `[r]` — pins FB-119 independence), Anti-behavior N/A (render-only hint, no keypress intercepted — defense accepted), Repeat-press N/A (stateless predicate, no accumulated state — defense accepted), Integration (`go install` + full suite green). Dependency FB-122 satisfied in-commit `59a75fc` (shared `eventsLoading` field). Prior: filed 2026-04-20 from user-persona FB-025 evaluation (P3-2).

**Priority: P3** — age label creates a staleness signal ("6m ago") but the action to remedy it (`[r]`) isn't on the most visible surface. Discoverable via `?` help overlay or the stale-empty recovery copy after 5m, but a first-time events-tab user seeing "6m ago" won't know to press `[r]` without leaving the view to consult the overlay.

#### User problem

The current DetailPane events-mode title-bar hint row reads something like:
```
[j/k] scroll  [E] describe  [x] delete  [Esc] back
```

`[r]` is absent. FB-025 created the signal (age label) that makes `[r]` operator-relevant, but left the affordance off the title-bar hint strip. Returning operators who know the `[r]` convention from other panes are fine; first-time events-tab users see the signal without the remedy.

#### Designer call

Two options:
- **A (add `[r] refresh` to the hint row when events-mode and `events != nil`):** minimum addition; gate on the same predicate FB-025 uses for the dispatcher so the hint only appears when `[r]` actually does something. Order in the hint row TBD — near the mode label (`events · 4m ago`) for proximity to the staleness signal, or grouped with other action keys.
- **B (contextual hint appears only when age >= some threshold, e.g. 5m):** hides the hint while data is fresh, reducing hint-row density; surfaces only when the operator plausibly needs it. More targeted but two-state rendering.

Designer should pick. A mirrors the "always-on-when-available" convention for other action hints; B introduces a new context-sensitivity pattern. Recommended default: A, unless hint-row width pressure at narrow terminal widths argues for B.

#### Dependencies

- FB-025 ACCEPTED (`9e310e3`) — this is a discoverability follow-up to the age label FB-025 introduced.
- FB-040 (title-bar hint suppression during loading, DEFERRED) — intersects; if/when FB-040 lands, FB-123's hint should obey the same suppression rule.

#### Non-goals

- Not changing the `[r]` keybinding semantics or dispatcher (FB-025 already covers these).
- Not rewriting the help overlay VIEW section (FB-025 already documents `[r]` there).
- Not proposing a rename of the `r` key.

---

### FB-124 — S4 quick-jump keys give no signal that NavPane focus suppresses them (silent no-op on first-press from NavPane)

**Status: ACCEPTED + COMMITTED + PERSONA-EVAL-COMPLETE 2026-04-20** — commit `079bf8f` on `feat/console` ("Welcome-panel quick-jump row now tells you to press Tab first when the sidebar has focus"). Persona eval 2026-04-20 returned positive-findings block + 3 P3 follow-ups (FB-125 label consistency, FB-126 `):  ` copy nit, FB-127 narrow-width trim amplification), zero P1/P2. Fix correctly resolves the originally-filed discoverability gap. **Prior Status: ACCEPTED 2026-04-20** — Option A (conditional prefix) shipped. Spec: `docs/tui-ux-specs/fb-124-s4-quickjump-activation-hint.md`. When `activePane == NavPane`, S4's quick-jump row reads `jump to ([Tab] to focus):  [b] backends …`; when TablePane has focus, it reverts to `jump to:`. Axis coverage: Observable × 2 (AC1/AC2), Input-changed × 1 (AC3), Anti-regression × 4 (AC4–AC7), Integration × 2 (AC8/AC9). FB-073 anti-regression tests green. Filed 2026-04-20 by product-experience from FB-073 user-persona P3.
**Priority: P3** — first-press friction for welcome-panel newcomers; returning operators unaffected. Emerged directly from FB-073 ACCEPTED (`7c042aa`) — gate is correct, discoverability of the activation step is the residual gap.

#### User problem

After FB-073 landed, quick-jump letter keys (`[b]`, `[n]`, `[w]`, `[p]`, `[g]`, `[v]`, `[i]`, `[z]`) only fire when `activePane != NavPane`. On welcome-panel landing (default state), `activePane == NavPane`. The S4 section renders these keys in the same accent-bold bracket format used for all active keys in the TUI (`jump to:  [b] backends  [n] networks …`) — there is no visual signal that the bracketed keys are currently dormant pending a pane change.

A first-time welcome-panel user sees S4's bracketed keys, presses one, gets a silent no-op, and has to independently discover the relationship between Tab (in S6's strip as `Tab  next pane`) and S4's keys. The connection "Tab changes pane" → "Tab activates quick-jump" is not explicit on any surface. An operator who Tabs and then looks at the resource table may not re-approach S4's keys.

Returning operators are fine — they remember the activation step. But the welcome panel is the first impression surface; first-press silent failures are a poor initial signal.

#### Designer call

Designer picks one of:

- **A (inline activation hint on S4 header):** change the S4 header from `jump to:` to `jump to (tab to activate):` or equivalent. Directly inlines the activation step adjacent to the keys themselves. Copy-only; width cost ~15-20 chars on the S4 first line. Consider width constraints at narrow terminal widths.
- **B (dim S4 keys when NavPane focused):** render S4 bracket tokens in muted style while `activePane == NavPane`, switch to accent-bold when TablePane focused. Operator sees the keys "light up" as a focus-change affordance. Adds a render-time conditional; more implementation cost than A, more semantically precise (dormancy is visible).
- **C (focus-footer line):** add a single line below S4 like `(tab to pane, then press a key)` rendered in muted style. Leaves S4 header intact; makes the activation step explicit without changing the key section itself. Copy + one line of vertical space.
- **D (status-bar hint augment):** update the NAV status bar to include `Tab  content pane` (or similar) closer to the letter-key zone of S6. Minimum copy change, but spatial distance from S4 keys remains.

Persona preference: A is most direct. B is most semantically correct (dormant things look dormant). C is the least disruptive. D is weakest (spatial disconnect).

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`.

1. **[Observable]** Designer's chosen affordance is present in welcome-panel View() when `activePane == NavPane` and S4 is visible. Test: construct `AppModel` in NavPane-focused welcome state at a width that shows S4; assert designer-pinned copy/style present in `stripANSIModel(appM.View())`.
2. **[Input-changed]** If Option B (dim-when-dormant): toggling `activePane` between NavPane and TablePane while S4 is visible produces a View() delta — pre/post `stripANSIModel` outputs differ. Test: record View() at NavPane; Tab to TablePane; record View(); assert distinct outputs. (If Option A/C/D is chosen, this axis becomes N/A — the copy is static regardless of pane focus; note N/A in axis-coverage table.)
3. **[Anti-regression]** FB-073 gate behavior is unchanged. Pressing `b` from NavPane is still a no-op (AC1 from FB-073); from TablePane it still fires (AC3). Test: existing FB-073 tests green.
4. **[Anti-regression]** S4 entry rendering (one row per matching type, `[letter] type-name` format) is unchanged for the body — only the header/surrounding copy is affected per chosen option.
5. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

**Dependencies:** FB-073 ACCEPTED (`7c042aa`) — FB-124 is the residual discoverability gap after FB-073's gate ships.

**Maps to:** FB-042 spec §7 (quick-jump S4 section) + FB-073 spec §4 (affordance surface analysis — this brief revisits the "no NavPane annotation needed" decision with post-ship persona evidence).

**Non-goals:**
- Not changing FB-073's gate logic (the `activePane != NavPane` guard stays).
- Not adding a modifier prefix to quick-jump keys (Option B from FB-073 spec; rejected there).
- Not changing the letter→type mapping or the letter set.
- Not touching help overlay copy (help overlay lists `[Tab] next pane` separately — the gap is on the welcome panel itself).

---

### FB-125 — Tab keybind label inconsistency: "to focus" (S4 hint) vs "next pane" (help overlay + status bar)

**Status: ACCEPTED 2026-04-20 + PERSONA-EVAL-COMPLETE 2026-04-20** — one-line source change at `resourcetable.go:432` (`"jump to ([Tab] next pane):  "`), plus 10 literal updates in test files. Shipped in commit `de407a5` on `feat/console`. Tests: 3 ACs covered — AC1 `TestFB125_AC1_Observable_NewCopyPresent` (View() substring assertion); AC2/AC3 anti-regression reuse FB-124 tests with updated literals (gating predicate unchanged). Submitter-produced axis-coverage table confirmed; `go install ./...` clean; `go test ./internal/tui/...` exit 0. Resolves FB-126 (designer predicted `):  ` awkwardness dissolves with noun-phrase literal; persona grep-verified all 4 Tab-description sites now read identically — `resourcetable.go:432/739/757` + `helpoverlay.go:28`). **Persona eval 2026-04-20:** positive findings on cross-surface consistency; FB-126 dissolution verified. **1 P3 finding DISMISSED with rationale** (P3-1: persona-self-labeled trade-off — "to focus" pedagogy marginally stronger than "next pane" descriptive framing, but persona explicitly notes "no evidence the new copy actually fails to teach the mechanic" and "consistency benefit outweighs the pedagogy delta"; revisit only on operator evidence). **0 new briefs filed.** Combined spec with FB-126 at `docs/tui-ux-specs/fb-125-126-s4-hint-copy-alignment.md`. Ux-designer chose Option A. **Prior Status: PENDING ENGINEER 2026-04-20** → PENDING UX-DESIGNER (copy decision) — filed 2026-04-20 by product-experience from FB-124 user-persona P3-1.
**Priority: P3** — cross-surface label drift; operator cross-referencing `?` help may briefly wonder if these describe different actions.

#### User problem

FB-124 introduced the copy `jump to ([Tab] to focus):  …` on the welcome panel's S4 row when NavPane is active. The help overlay and status bar describe the same `Tab` key as `[Tab] next pane`. Same key, same action, two different verbs: "focus" vs "next pane."

Persona observation: "A user who presses `?` to cross-reference will see 'next pane' and may momentarily wonder if these are different actions." This is a real comprehension friction for a discoverability-motivated feature — the fix to one inconsistency introduced another.

#### Designer call

Designer picks one of:

- **A (align to "next pane" everywhere):** change FB-124's S4 hint to `jump to ([Tab] next pane):  …`. Consistent with help overlay's established phrasing. Slight semantic drift (the user's goal is to *focus* the table so keys activate; "next pane" describes the key's mechanism).
- **B (align to "focus" everywhere):** change help overlay's `[Tab] next pane` row to `[Tab] next pane (focus)` or `[Tab] focus next pane`. Preserves the S4 hint's framing; touches the help overlay, which may have wider ripple effects (other surfaces using `next pane` phrasing).
- **C (neutral phrasing both):** pick a third phrasing like `[Tab] switch pane` or `[Tab] move to next pane` and align both surfaces. Highest cost; probably not warranted for a P3.

Persona preference: A is the lower-touch option — one place to change, matches established convention. Designer decides.

#### Acceptance criteria

Axis tags: `[Observable]`, `[Anti-regression]`.

1. **[Observable]** S4 hint (when present) and help overlay Tab row use the same verb phrase. Test: in a state where both surfaces are renderable, assert identical or semantically-equivalent substrings in `stripANSIModel(appM.View())` for each surface.
2. **[Anti-regression]** FB-124 conditional-rendering behavior preserved (hint appears when NavPane focused; absent when TablePane focused).
3. **[Anti-regression]** Help overlay rendering otherwise unchanged.
4. **[Integration]** `go install ./...` + `go test ./internal/tui/...` green.

**Maps to:** FB-124 user-persona P3-1.

---

### FB-126 — S4 quick-jump hint `):  ` double-closer reads awkwardly

**Status: RESOLVED 2026-04-20** — closed as designer predicted: once FB-125 shipped `"jump to ([Tab] next pane):"` (noun phrase), the `):` reads as standard English annotation-then-list, not a double-closer. No separate engineer work required; no separate commit. Verified against committed rendering. **Prior Status: RESOLVED-BY-FB-125 → PENDING UX-DESIGNER (copy decision)** — filed 2026-04-20 by product-experience from FB-124 user-persona P3-2.
**Priority: P3** — low-impact copy nit; first-read clarity only.

#### User problem

FB-124's prefix `jump to ([Tab] to focus):  [b] backends …` contains a closing paren immediately followed by a colon: `):`. Persona observation: "The sentence wants to end twice." The typographical odd-moment is small but avoidable.

#### Designer call

- **A (drop the parens, use dash):** `jump to — [Tab] to focus:  [b] backends …`. Single closer. Slightly longer visual weight on the separator.
- **B (restructure as separate segment):** `jump to:  (Tab to focus)  [b] backends …`. Parenthetical sits between the label and the keys, rendered muted like a caption. Width cost comparable to current; reading flow is "label → instruction → actions."
- **C (shorter parenthetical):** `jump to (Tab first):  [b] backends …`. Removes the nested `[Tab]` bracket — one pair of brackets instead of two. Shorter, but loses the bracket-notation parallel with `[b]/[n]/[w]`.
- **D (accept as-is):** keep the current copy; persona-P3-only means it works in context. Close as "won't fix."

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`.

1. **[Observable]** New copy present in S4 when NavPane focused. Test: `stripANSI(m.View())` contains designer-pinned substring.
2. **[Input-changed]** Tab pane-switch still toggles hint visibility (FB-124 behavior preserved).
3. **[Anti-regression]** S4 entry body (`[b] backends`, `[n] networks`, etc.) unchanged.
4. **[Integration]** Build + test suite green.

**Maps to:** FB-124 user-persona P3-2.

---

### FB-127 — S4 hint amplifies narrow-width trim: operators in hint state see more destinations clipped

**Status: WONTFIX 2026-04-20** — ux-designer ruling: Option C (close without fix). Rationale: narrow-width clip is a known tradeoff already documented in FB-124 spec §2.4. FB-125's copy change saves only 1 rendered char (28→27), negligible impact on clipping. Operators in the hint state have dormant keys regardless of visible destination count, so "missing destinations" read has no functional consequence. Adding width-conditional copy adds complexity for a P3 minority-case (narrow-terminal mode). If narrow-terminal support is later elevated as a priority area, re-file with evidence of recurring operator confusion from that width range. **Prior Status: PENDING UX-DESIGNER (decision — fix, ack, or close)** — filed 2026-04-20 by product-experience from FB-124 user-persona P3-3.
**Priority: P3** — not a regression in behavior; known tradeoff from FB-124 spec §2.4.

#### User problem

The FB-124 hint prefix is ~18 chars longer than the plain prefix. At narrow terminal widths (~65 cols content), the existing trim-from-right logic in `renderQuickJumpSection` clips more destinations before the `…` kicks in. Operators in the hint state (who can't press the keys anyway — NavPane focused) see fewer visible destinations than post-Tab operators at the same width.

Persona observation: "A user in the hint state (keys disabled anyway) who sees `jump to ([Tab] to focus):  [b] backends  [n] networks …` might infer the missing destinations are conditionally hidden rather than just clipped."

This was flagged in FB-124 spec §2.4 as an accepted tradeoff (trim-from-right handles it gracefully). Persona is flagging that "graceful" technically is still "confusing" at narrow widths.

#### Designer call

- **A (shorter hint at narrow widths):** render a shorter hint below ~70 cols — e.g., `jump to (Tab first):` instead of `jump to ([Tab] to focus):`. Saves 8+ chars; keys get more visible space. Implementation: width check inside `renderQuickJumpSection`.
- **B (drop hint at narrow widths):** below a threshold (e.g., 60 cols), suppress the parenthetical entirely. Operators at narrow widths see plain `jump to:` + clipped keys. Loses the hint at narrow widths; acceptable if narrow-width is a rare mode.
- **C (accept as-is, close):** document in the backlog as "known tradeoff, narrow-width mode is a minority case," close without fix.

Persona suggested this is only worth fixing if narrow-terminal support is a priority for the TUI. Designer decides.

#### Acceptance criteria (if fix chosen)

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`.

1. **[Observable]** At narrow width (e.g., 65 cols), designer's chosen narrow-variant appears in `stripANSI(m.View())`.
2. **[Observable]** At wide width (≥80 cols), existing FB-124 hint still appears.
3. **[Input-changed]** Width transition: resize from 65 → 100 cols produces a View() delta; copy changes.
4. **[Anti-regression]** FB-124 behavior at wide widths unchanged.
5. **[Integration]** Build + test suite green.

**Maps to:** FB-124 user-persona P3-3.

---

### FB-128 — S3 activity error hint missing action verb: `(press [r])` vs `([r] to retry)`

**Status: ACCEPTED 2026-04-20 + PERSONA-EVAL-COMPLETE 2026-04-20** — Option B shipped bundled with FB-129 at `resourcetable.go:335-342`: transient-error body now renders `"activity unavailable ([r] to retry)"` (35 chars). 3 new FB-128 tests + TestFB102_AC1 cross-feature copy-update all PASS with `stripANSI(m.renderActivitySection(80))` assertion pattern: AC1 Observable new copy present; AC2 Observable old copy (`"(press [r])"` / `"press "`) absent; AC3 Anti-regression CRD-absent no-parenthetical preserved; AC4 Anti-regression FB-102 test updated to new copy; AC5 Integration `go install ./...` + suite `ok`. Axis: Observable × 2, Anti-regression × 2, Integration × 1 (Input-changed N/A — pure one-site copy swap with no state transition, AC1/AC2 pair present/absent as coverage for the swap). Submitter-produced axis-coverage table confirmed pre-review; all 4 test functions verified to exist with View()-equivalent rendered-output assertions. **Persona eval** (2026-04-20, bundled with FB-129): key-first copy confirmed correct (matches `[4] full dashboard`, `[r] refresh` status bar pattern); "retry" verb precise for error recovery; self-contained, no prior `[r]` knowledge required. Joins existing correct semantic distinction with delete-dialog `[r] retry` + describe `[r] retry describe`. 0 P1/P2, 1 P3-2 filed as **FB-131** (help overlay `[r]  refresh` vs error-state `[r] retry` verb discordance — pre-existing condition that FB-128 widens by one surface, distinct user-problem). Shipped in commit `dda3419` on `feat/console`. **Prior Status: PENDING ENGINEER 2026-04-20** — spec delivered 2026-04-20 by ux-designer at `docs/tui-ux-specs/fb-128-error-hint-retry-verb.md`. **Option B selected**: `activity unavailable ([r] to retry)` — drops "press" as redundant (bracket notation implies keypress by TUI convention); 35 visible chars total. Implement together with FB-129 — same 4-line block in `resourcetable.go:335-342`. **Prior Status: PENDING UX-DESIGNER** — filed 2026-04-20 by product-experience from FB-102 user-persona P3-3.

**Priority: P3** — copy-refinement; FB-102 shipped `"activity unavailable (press [r])"` on transient-error surface. Persona evidence: the current copy requires prior knowledge that `[r]` means "refresh/retry" — established operators know this from the events mode status bar `[r] refresh`, but first-time operators on the welcome panel get no self-contained verb. Adding `to retry` (8 chars) keeps total body at ~40 visible chars — still well inside FB-082's Tier 2 (contentW ≥ 45) but closer to the Tier 3 boundary (contentW < 35), worth designer consideration alongside FB-129.

#### User problem

At `resourcetable.go:338-342`, on transient-error (`activityFetchFailed=true`, `activityCRDAbsent=false`):

```go
body = muted.Render("activity unavailable") +
    " " +
    muted.Render("(press ") + accentBold.Render("[r]") + muted.Render(")")
```

Renders: `activity unavailable (press [r])` — 32 visible chars.

Persona observation: "The events mode status bar uses `[r] refresh` — the S3 error hint uses only `(press [r])`. For a first-time operator, the verb 'retry' makes the affordance self-contained." The parenthetical currently relies on the operator knowing `[r]` = refresh from elsewhere in the UI (events status bar, help overlay `[r] refresh`).

#### Designer-call options

| Option | Rendered body | Visible width | Trade-off |
|--------|---------------|---------------|-----------|
| A | `activity unavailable (press [r] to retry)` | 40 chars | Self-contained verb; adds 8 chars; still fits Tier 2 (≥45) comfortably; edges Tier 3 narrow-width gap (FB-129 territory). |
| B | `activity unavailable ([r] to retry)` | 35 chars | Drops "press" (the bracketed notation already implies keypress by TUI convention); self-contained verb; same width as current +3. |
| C | `activity unavailable — [r] retry` | 32 chars | Em-dash separator; loses parenthetical distinction from CRD-absent; may regress FB-102's visual contract. |
| D | Keep current `(press [r])` | 32 chars | No change; trust operators to learn the meaning from other UI sites. |

**Designer ask:**

- Preference between A (full verb) vs B (compact verb) vs D (status quo)?
- If A or B: coordinate with FB-129 (narrow-width guard) — at contentW < 40, the longer form wraps or clips. Decision: drop parenthetical at narrow width (FB-129 Option A/C) or keep verbless short form?
- Preserve the distinction from CRD-absent (no parenthetical) — reject Option C unless CRD-absent surface is also revised.

Output minimal spec in `docs/tui-ux-specs/fb-128-error-hint-retry-verb.md` (≤30 lines — this is a one-line copy change + ACs).

#### Acceptance criteria (if fix chosen)

Axis tags: `[Observable]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable]** Transient-error state → `stripANSI(m.View())` contains designer-chosen copy (e.g., `"(press [r] to retry)"` for Option A).
2. **[Anti-regression]** CRD-absent state → View() contains `"activity unavailable"` with NO parenthetical (FB-102 contract preserved).
3. **[Anti-regression]** FB-102 tests updated to match new copy; FB-082 CRD-absent path unchanged.
4. **[Integration]** Build + test suite green.

**Maps to:** FB-102 user-persona P3-3.

---

### FB-129 — S3 error body has no width-guard for contentW < 35 (3-tier contract gap)

**Status: ACCEPTED 2026-04-20 + PERSONA-EVAL-COMPLETE 2026-04-20** — Option A threshold 35 shipped bundled with FB-128 at `resourcetable.go:336`: `if m.activityCRDAbsent || contentW < 35` → plain `"activity unavailable"`; else full FB-128 form. 5 new FB-129 tests PASS with rendered-output assertions: AC1 Observable narrow-transient (contentW=34) no parenthetical; AC2 Observable wide-transient (contentW=35) full copy; AC3 Input-changed width-transition 34→35 changes View; AC4 Anti-regression CRD-absent at wide width unchanged; AC5 Anti-regression FB-082 data-row tier thresholds unaffected; AC6 Integration `go install ./...` + suite `ok`. Axis: Observable × 2, Input-changed × 1, Anti-regression × 2, Integration × 1. Submitter-produced axis-coverage table confirmed pre-review; all 5 test functions verified to exist with `stripANSI(m.renderActivitySection(contentW))` rendered-output assertions; width-transition AC3 uses explicit v1/v2 diff pattern per Input-changed convention. **Persona eval** (2026-04-20, bundled with FB-128): 35-char threshold math confirmed exact (full hint = 35 visible chars, collapses at 34 without clipping at 35). 0 P1/P2, 1 P3-1 **DISMISSED** — persona-surfaced narrow-width collapse losing retry affordance argument was explicitly considered-and-dismissed in FB-129 spec §0 ("Abbreviations introduce a new truncation pattern not present elsewhere in S3/S4 body copy" + "at contentW < 35 the TUI is near-unusable anyway"); persona's own qualifier acknowledged "Not a blocker, 30-col TUI use is rare." No new evidence beyond the already-documented trade-off. **Prior Status: PENDING ENGINEER 2026-04-20** — spec delivered 2026-04-20 by ux-designer at `docs/tui-ux-specs/fb-129-error-body-narrow-width-guard.md`. **Option A selected, threshold 35**: `if m.activityCRDAbsent || contentW < 35` → plain `"activity unavailable"`; else full form from FB-128. Threshold aligns with FB-082 Tier 3 floor — no new breakpoint. Implement together with FB-128 — same 4-line block. **Prior Status: PENDING UX-DESIGNER** — filed 2026-04-20 by product-experience from FB-102 user-persona P3-4.

**Priority: P3** — narrow-width edge case; FB-082's three-tier truncation contract (`contentW >= 65`, `>= 45`, `< 45`) governs data-row column-drop but does not apply to the body when `activityFetchFailed=true`. The concatenated error body `"activity unavailable (press [r])"` (~32 visible chars) is rendered raw without width checks. At `contentW < 35`, terminal wrapping splits the phrase mid-word (typically `activity unavailable (press` \n `[r])`), regressing the single-line body contract FB-082 established.

#### User problem

At `resourcetable.go:335-342`:

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

No contentW parameter is consulted for either branch. At narrow widths (contentW < 35), the terminal wraps the transient-error body. CRD-absent (20 chars) survives contentW ≥ 20; transient-error (32 chars) needs contentW ≥ 32 to avoid wrap.

Persona observation: "Low-probability edge case, but it's outside the existing width contract."

The FB-082 contract was written assuming the body would be either the 20-char "activity unavailable" or single-row data lines (handled by column-drop). FB-102 introduced a 32-char error body variant that falls outside the contract.

#### Designer-call options

| Option | Behavior at contentW < 35 | Trade-off |
|--------|---------------------------|-----------|
| A | Drop the parenthetical entirely — render just `"activity unavailable"` | Loses the recovery affordance at narrow widths; matches CRD-absent surface (but CRD-absent is semantically different). |
| B | Compact variant — render `"unavail. [r]"` or `"err · [r]"` | Preserves hint; abbreviations may feel cryptic. |
| C | Render plain `"activity unavailable"` + put the retry hint on the status bar via `PostHint`/`postHint` | Clean body; adds status-bar coupling (collision risk with FB-079/FB-097). |
| D | Close — trust terminal wrap at contentW < 35; narrow-mode is rare | No implementation; accepts minor visual regression in narrow mode. |

**Designer ask:**

- Coordinate with **FB-128**: if the verb is added (`"(press [r] to retry)"` = 40 chars), the narrow-width threshold where wrapping kicks in shifts from 32 to 40. Decision about the threshold should happen after FB-128.
- If Option A chosen: symmetric with FB-102's CRD-absent rendering — CRD-absent retains the clean "activity unavailable" at all widths; narrow-width transient also uses it. Operator loses visual distinction from CRD-absent but gains single-line guarantee.
- Prefer a threshold that aligns with FB-082's existing tiers (45, 35, or a new 40) to avoid cognitive overload from independent breakpoints.

Output minimal spec in `docs/tui-ux-specs/fb-129-error-body-narrow-width-guard.md`.

#### Acceptance criteria (if fix chosen)

Axis tags: `[Observable]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable]** At contentW < threshold (designer-chosen), transient-error body fits on one line (no terminal wrap). `strings.Count(stripANSI(body), "\n") == 0`.
2. **[Observable]** At contentW ≥ threshold, full `"activity unavailable (press [r]…)"` copy renders (current FB-102 / FB-128 behavior).
3. **[Anti-regression]** CRD-absent at all widths still renders `"activity unavailable"` — no parenthetical change.
4. **[Anti-regression]** FB-082 tier-contract for data rows unaffected.
5. **[Integration]** Build + test suite green.

**Maps to:** FB-102 user-persona P3-4.

---

### FB-130 — S3 spinner render gate misses empty-but-loaded activity rows

**Status: ACCEPTED 2026-04-20** — engineer shipped one-line gate widen at `resourcetable.go:333` from `m.activityRows == nil` to `len(m.activityRows) == 0`; 4 FB-130 tests + AC5 reuse (`TestFB103_AC7_Observable_ErrorState_RPress_ShowsLoading`) all PASS; full suite + `go install ./...` clean; AC1 setup order corrected after first submission (SetActivityRows([]) before SetActivityLoading(true) — reaches fix-target state `loading=true + rows=[]non-nil`); AC2 Input-changed exercises AppModel-layer [r]-press through model.go:1318 state gate; AC3/AC4 anti-regression for nil-rows + populated-rows paths. **Prior Status: PENDING ENGINEER** — engineer-direct 1-char change at `internal/tui/components/resourcetable.go:333`. Filed 2026-04-20 by product-experience from FB-103 user-persona P3-1.

**Priority: P3** — narrow-site instance of FB-103's refresh-feedback thesis. Operators on a genuinely-empty project (activity fetch succeeded, returned 0 rows → `activityRows = []ActivityRow{}`) press `[r]` on the welcome panel and see no in-flight signal. FB-103's state gate at `model.go:1317-1320` correctly calls `SetActivityLoading(true)` because `ActivityRowCount()==0` returns 0 for empty slices, but the render gate at `resourcetable.go:333` is stricter — `m.activityLoading && m.activityRows == nil` — so the spinner doesn't fire for empty-but-non-nil rows. Teaser stays frozen at `"no recent activity"` for the entire fetch round-trip.

#### User problem

Same as FB-103 P2-1 and P3-2: press `[r]`, see no visual change. FB-103 covered three `rows == nil` scenarios — (a) never-loaded, (b) error-cleared via FB-082 `SetActivityFetchFailed`, (c) CRD-absent. FB-130 covers the fourth: (d) successfully-loaded-empty where `activityRows == []ActivityRow{}` (not nil). A project with zero recent activity — common on new projects — hits this case.

#### Site

`internal/tui/components/resourcetable.go:333`

```go
case m.activityLoading && m.activityRows == nil:
```

#### Fix (Option A — align render gate to state gate)

**Before:**
```go
case m.activityLoading && m.activityRows == nil:
    body = muted.Render("⟳ loading…")
```

**After:**
```go
case m.activityLoading && len(m.activityRows) == 0:
    body = muted.Render("⟳ loading…")
```

`len(nil slice) == 0` in Go, so this widens the predicate to also fire for empty-but-non-nil rows without regressing the nil path. Aligns the render gate to FB-103's `ActivityRowCount() == 0` state gate (which returns `len(m.activityRows)`).

#### Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable]** Empty-but-loaded rows + loading flag → spinner. Setup: `SetActivityRows([]ActivityRow{})` then `SetActivityLoading(true)`. `stripANSI(m.View())` contains `"loading…"`; does NOT contain `"no recent activity"`.
2. **[Input-changed]** `[r]` press on genuinely-empty project: pre-press View contains `"no recent activity"`, post-press View (after `SetActivityLoading(true)` on empty-but-non-nil rows) contains `"loading…"`. `v1 != v2`.
3. **[Anti-regression]** FB-103 AC1 (never-loaded `rows==nil` → spinner) still passes — `len(nil) == 0` so predicate still true.
4. **[Anti-regression]** FB-103 AC3 (populated-rows silent-refetch, FB-076 preserved) still passes — `len(rows) > 0` means loading flag does NOT fire spinner.
5. **[Anti-regression]** FB-103 AC7 (error-state → spinner, rows cleared to nil by FB-082) still passes — `len(nil) == 0`.
6. **[Integration]** `go install ./...` exit 0; `go test ./internal/tui/...` all packages `ok`.

**Axis summary:** Observable × 1, Input-changed × 1, Anti-regression × 3, Integration × 1.

#### Non-goals

- Not changing FB-103's state gate at `model.go:1317-1320`.
- Not changing FB-076 silent-refetch convention (populated rows still do not flash spinner).
- Not changing spinner copy or styling.

**Maps to:** FB-102 chain → FB-103 user-persona P3-1.

---

### FB-131 — Help overlay `[r]  refresh` vs. error-state `[r] retry` verb discordance

**Status: WONTFIX 2026-04-20** — ux-designer ruling at `docs/tui-ux-specs/fb-131-help-overlay-r-verb.md` (Option C). **Rationale:** the two-verb convention pre-exists FB-128 — `deleteconfirmation.go:166` already ships `[r] refresh` on conflict state, `:172` already ships `[r] retry` on transient error state. FB-128 extended a correct, in-use two-sense pattern to a new surface; it did not introduce a new inconsistency. The convention is: **`refresh` = re-fetch in normal state** (help overlay, all status bars, events title bar) vs. **`retry` = re-fetch after failure** (error-state inline affordances). Option A rejected on layout grounds (help overlay VIEW column is 22 cols, `[r] refresh / retry on errors` = 30 chars — overflow). Option B rejected as brittle state coupling (requires AppModel to aggregate error state from 4+ independent components for a P3). Option D rejected because it would regress `deleteconfirmation.go:172` and `model.go:2075` — multi-surface regression for no UX gain. Spec §2 documents the convention for future designers (resist future "alignment" amplification briefs). **Prior Status: PENDING UX-DESIGNER** — cross-surface copy alignment question; ux-designer call required. Filed 2026-04-20 by product-experience from FB-128/FB-129 user-persona P3-2.

**Priority: P3** — pre-existing condition widened by FB-128. Before FB-128, one error surface said `"retry"` (delete dialog `[r] retry`) and one said `"retry describe"` (describe view `[r] retry describe`) while the help overlay said `[r]  refresh`. FB-128 added a third error surface (`"([r] to retry)"` on the S3 teaser transient-error path), widening the gap. An operator who sees `([r] to retry)` on S3 and presses `?` to confirm the binding lands on `[r]  refresh` in the help overlay — the verbs alias, but the overlay doesn't telegraph that.

#### User problem

At `internal/tui/components/helpoverlay.go:53`:

```go
row.Render("[r]  refresh"),
```

Current verb inventory for `[r]`:

| Surface | Copy | Verb |
|---------|------|------|
| Help overlay VIEW column | `[r]  refresh` | refresh |
| Events mode status bar | `[r] refresh` | refresh |
| Welcome-panel S3 error (FB-128) | `activity unavailable ([r] to retry)` | retry |
| Delete dialog error | `[r] retry` | retry |
| Describe view error | `[r] retry describe` | retry |
| Placeholder action row (FB-084) | `[r] retry describe` | retry |

The verbs are semantically precise — `refresh` = re-fetch live data, `retry` = error recovery — but the help overlay collapses both senses under one label. For a first-time operator, `[r]` in the overlay reads as a single action, not a dual-sense key. The discordance is small but persistent: three distinct error surfaces use `retry`, the overlay uses `refresh`.

#### Designer-call options

| Option | Rendering | Trade-off |
|--------|-----------|-----------|
| A | Help overlay line becomes `[r]  refresh / retry on errors` | Legibly lists both senses; ~28 chars (current `[r]  refresh` is 12 chars) — fits within overlay column easily; no context-awareness required. |
| B | Context-aware overlay: when an error-state surface is visible (S3 error body, delete dialog shown, describe error), overlay swaps to `[r]  retry`. Otherwise `[r]  refresh`. | Most accurate; but introduces state coupling between overlay and error surfaces — new coordination surface that may regress under later UI changes. |
| C | Keep help overlay as `[r]  refresh`; document the sense-split in-context (treat error-state `retry` copy as self-explanatory, overlay as general-purpose). | No change; accepts the pre-existing condition on the grounds that `refresh` is the primary sense and `retry` on error surfaces is semantically derivable. |
| D | Normalize all error surfaces to `[r] refresh` (reverts FB-128's "retry" verb choice + delete-dialog + describe labels). | Uniformity at the cost of `retry`'s precision; contradicts FB-128's accepted rationale (verb makes affordance self-contained). |
| E | Defer — file as watch-listed under "future alignment pass" and close this brief without action. | No change but acknowledges the gap; risk is the gap stays invisible until another error surface ships. |

**Designer ask:**

- Preference between A (legible dual-sense label), B (context-aware overlay), C (status-quo with rationale), D (reverts FB-128, unlikely), E (defer)?
- If A: confirm the slash-separated form reads cleanly alongside neighboring overlay entries (`[d]  describe`, `[c]  switch context`).
- If B: who owns the state coupling? Help overlay is a component (`helpoverlay.go`) receiving hints from the parent model — is the error-state signal already available, or does this introduce a new setter?
- If C or E: record the decision with enough rationale to resist a future "one more surface" amplification brief.

Output minimal spec in `docs/tui-ux-specs/fb-131-help-overlay-r-verb-alignment.md` (≤40 lines — this is a copy decision + ACs, not a behavioral change).

#### Acceptance criteria (if fix chosen)

Axis tags vary by option. For **Option A** (most likely):

Axis tags: `[Observable]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable]** Help overlay visible → `stripANSI(overlay.View())` contains designer-chosen copy (e.g., `"[r]  refresh / retry on errors"`).
2. **[Observable]** Old copy absent → `stripANSI(overlay.View())` does NOT contain `"[r]  refresh"` as a standalone line (allowing the substring inside the combined form).
3. **[Anti-regression]** S3 error surface unchanged — `"([r] to retry)"` still present in welcome-panel transient-error body.
4. **[Anti-regression]** Delete dialog `[r] retry` copy unchanged.
5. **[Anti-regression]** Describe view `[r] retry describe` copy unchanged.
6. **[Anti-regression]** Events mode status bar `[r] refresh` copy unchanged.
7. **[Integration]** Build + test suite green.

For **Option B** (context-aware), add `[Input-changed]` axis for the overlay re-render on error-state signal change.

For **Option C/E** (no change), file a RESOLVED/WONTFIX entry with the designer's rationale — no ACs required.

#### Non-goals

- Not changing FB-128's S3 error-body copy (`"([r] to retry)"` shipped and accepted).
- Not introducing a new key-binding (still `[r]` with the same handler).
- Not touching the status-bar `[r] refresh` copy in events mode (separate surface, refresh semantics remain correct).

**Maps to:** FB-128/FB-129 user-persona P3-2.


