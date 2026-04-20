# FB-144 — FB-143 transient sub-line `"Refresh to retry."` should use explicit `[r]` keybind

**Status:** PENDING ENGINEER
**Priority:** P3
**Brief source:** `docs/tui-backlog.md` — search `### FB-144`
**Dependencies:** FB-143 ACCEPTED

---

## 1. Problem statement

FB-143 ACCEPTED copy `"Refresh to retry."` uses "Refresh" as an implicit verb mapping to the `[r]` keybind. An operator already in muscle-memory knows the mapping; a first-encounter operator must glance at the status bar to find the key. This is the only sub-line in the S2 Platform health region whose remedy is an in-TUI keybind — the other two sub-lines from FB-140 point to people outside the TUI and correctly do not name a key.

Brief is scope-limited to one string literal at `internal/tui/components/resourcetable.go:281`.

---

## 2. Pinned design: Option A — `"Press [r] to retry."`

**Replace `"Refresh to retry."` with `"Press [r] to retry."`**

### Rationale

- Option A (`"Press [r] to retry."`) adds the explicit keybind the persona requested. "retry" is semantically precise for an error-state re-attempt — it correctly signals "you're re-attempting something that failed," not just "you're pulling fresh data."
- Option B (`"Press [r] to refresh."`) was considered. Matching the status-bar label vocabulary (`[r] refresh`) is a real benefit, but "refresh" in the status bar is a generic action name that covers all refresh scenarios; it does not preclude "retry" in an error-specific context. The two uses coexist without conflict.
- Option C (keep current) explicitly declined by persona evaluation.
- Option D (retrofit other branches) pre-rejected: the other two sub-lines from FB-140 point to people, not keys; adding `[r]` would be semantically wrong for those branches.

### Copy

| Location | Before | After |
|---|---|---|
| `resourcetable.go:281` | `"Refresh to retry."` | `"Press [r] to retry."` |

No gate logic change. No structural change. One string literal replacement.

---

## 3. Render site — exact change

File: `internal/tui/components/resourcetable.go`

**Before (L279–283):**
```go
line := muted.Render("Platform health temporarily unavailable")
if contentW >= 40 {
    line += "\n" + muted.Render("Refresh to retry.")
}
return leftHeader + "\n\n" + line
```

**After:**
```go
line := muted.Render("Platform health temporarily unavailable")
if contentW >= 40 {
    line += "\n" + muted.Render("Press [r] to retry.")
}
return leftHeader + "\n\n" + line
```

`"Press [r] to retry."` is 19 chars — shorter than `"Refresh to retry."` (17 chars) by 2 chars but well within the `contentW >= 40` gate. No gate change needed.

---

## 4. File locations

- **Render site:** `internal/tui/components/resourcetable.go` L281 — one string literal
- **HelpOverlay:** `internal/tui/components/helpoverlay.go` — **no changes required**
- **Key handler:** `internal/tui/model.go` — **no changes required**
- **Other FB-140 sub-line branches** (L275, L290) — **no changes required**

---

## 5. Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable]** When `m.bucketErr != nil && !m.bucketUnauthorized` and `contentW >= 40`, `stripANSI(welcomePanel().View())` contains `"Press [r] to retry."`. Test: render with non-unauthorized error and sufficient width; assert new substring present.

2. **[Observable — old copy absent]** Same render state does NOT contain `"Refresh to retry."`. Test: render; assert old substring absent.

3. **[Input-changed]** Three error branches (unauthorized, unconfigured, transient) produce three visibly distinct View() outputs at `contentW >= 40`. Test: render each state; assert pairwise inequality.

4. **[Anti-regression]** FB-143 AC2 (narrow suppression at `contentW < 40`) green: sub-line absent at narrow widths.

5. **[Anti-regression]** FB-140 AC1–AC4 green unmodified.

6. **[Anti-regression]** FB-139 AC1–AC3 green unmodified.

7. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

---

## 6. Non-goals

- Not touching the other two FB-140 sub-line branches (semantically different).
- Not changing the primary copy `"Platform health temporarily unavailable"`.
- Not changing the `contentW >= 40` gate.
- Not adding `[r]` label anywhere else in the Platform health region.
