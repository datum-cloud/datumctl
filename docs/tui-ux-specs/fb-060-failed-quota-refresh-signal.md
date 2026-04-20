# FB-060 — Failed quota refresh signal on the quota surface

**Status:** PENDING ENGINEER
**Priority:** P3
**Brief source:** `docs/tui-backlog.md` — search `### FB-060`
**Dependencies:** FB-043 ACCEPTED, FB-063 ACCEPTED, FB-059 ACCEPTED

---

## 1. Problem statement

After pressing `r` on QuotaDashboard when the fetch fails, the title bar reverts to the
pre-failure `"  updated Xs ago"` (because `fetchedAt` is unchanged). Two outcomes —
"refresh succeeded, data is fresh" and "refresh failed, data is stale" — render identically
on the quota surface. The error appears only in the status bar, which is out of the operator's
focused attention when looking at the quota grid.

---

## 2. Option selection

### Chosen: Option (a) — inline failure annotation in the title bar

**Why (a):**
The title bar is where the operator looks for freshness confirmation. Adding a failure
annotation there puts the signal exactly where the expectation lives. The copy is a
substring the AC test can assert directly, satisfying AC#1.

**Why not (b) — color shift on freshness string:**
FB-059 owns the freshness age string palette (muted → warning → error based on
staleness threshold). Shifting the same string's color on a refresh failure would
conflict with FB-059's threshold-driven coloring, as both conditions can coexist
(old data + failed refresh = both rules apply simultaneously). Introducing a second
color-shift axis onto one string creates unresolvable ambiguity.

**Why not (c) — toast / status-bar correlation:**
The brief notes the error is already in the status bar. A second status-bar signal
is redundant; a toast overlay is heavyweight for a P3 signal. Neither reaches the
operator looking at the quota grid.

---

## 3. Designer-ratified copy

The failure indicator replaces the freshness annotation in the title bar left segment.
It does NOT modify the right-side hint.

| Width mode | Failure label | Style |
|---|---|---|
| `w >= 80` (wide) | `"  ✗ refresh failed"` | `styles.Warning` |
| `w < 80` (narrow) | `" ✗"` | `styles.Warning` |

`styles.Warning` is the amber/yellow semantic token — signals a recoverable condition
without the alarm of `styles.Error`. This parallels the platform-health pattern where
transient failures use a muted warning rather than a hard error color.

The wide label `"  ✗ refresh failed"` (18 chars) mirrors `"  ⟳ refreshing…"` (16 chars)
in the existing wide `refreshingLabel`. The narrow label `" ✗"` (2 chars) mirrors `" ↻"`
(2 chars) in the narrow `refreshingLabel`. The parallel structure is intentional.

---

## 4. Visual spec

### 4.1 Title bar left segment — state machine

Three states occupy the same position (immediately after `baseLeft`):

| State | Left segment | Condition |
|---|---|---|
| Refreshing | `baseLeft + "  ⟳ refreshing…"` (wide) / `" ↻"` (narrow) | `m.refreshing == true` |
| **Refresh failed** | **`baseLeft + "  ✗ refresh failed"` (wide) / `" ✗"` (narrow)** | **`m.refreshFailed == true && !m.refreshing`** |
| Freshness age | `baseLeft + "  updated Xs ago"` (wide) / `" · Xs ago"` (narrow) | `!m.refreshFailed && !m.fetchedAt.IsZero()` |
| No annotation | `baseLeft` only | `fetchedAt.IsZero() && !m.refreshFailed && !m.refreshing` |

Priority in if-chain: `refreshing` → `refreshFailed` → `fetchedAt` (normal freshness).

### 4.2 Gap guard applies to all states

The same `>= 2` gap check used for the refreshing and freshness labels applies to the
failure label. If there is not enough space, the label silently drops — identical to
the existing gap-guard behavior for freshness and refreshing. No new threshold.

### 4.3 Placement

Wide example (w = 100):
```
quota usage  ✗ refresh failed         [↑/↓] move  [t] table  [s] group  [r] refresh
```

Narrow example (w = 58):
```
quota usage ✗            [↑/↓] [t] [s] [r]
```

The `✗` glyph is styled with `styles.Warning` only — no bold, no additional decoration.

---

## 5. Interaction with FB-063 `refreshing…` indicator

FB-063 added `m.refreshing` which preempts the freshness label while a re-fetch is in
flight. FB-060 adds `m.refreshFailed` which shows only when `refreshing == false`.

During a re-attempt after a prior failure:
1. Operator presses `r` → `refreshing = true` → indicator shows `"  ⟳ refreshing…"`,
   the `✗ refresh failed` label disappears.
2. If re-attempt succeeds → `refreshFailed = false`, freshness updated.
3. If re-attempt fails again → `refreshing = false`, `refreshFailed = true` → `✗ refresh failed`
   reappears.

The two indicators never render simultaneously. The if-chain order enforces this.

---

## 6. Interaction with FB-059 freshness threshold styling

FB-059 applies `styles.Warning` or `styles.Error` to the age string when `fetchedAt` is
older than a threshold. The failure indicator replaces the freshness age string when
`refreshFailed == true` — they never co-render on the same pass. FB-059's threshold logic
is undisturbed; it only executes when the freshness branch is reached (after both
`refreshing` and `refreshFailed` guards pass).

---

## 7. Model changes required

### 7.1 New field

```
refreshFailed bool  // FB-060: true after a failed r, cleared when SetBuckets is called
```

### 7.2 New setter

```
SetRefreshFailed(failed bool)
```

Called by `model.go` on the error path after a refresh attempt: set to `true`.

### 7.3 Clearing behavior — AC#2

`SetBuckets()` clears `refreshFailed` as a side effect. Rationale: receiving new bucket
data is the canonical "success" signal. This means the clearing happens automatically
on any successful load — initial or refresh — without a separate caller.

`SetLoading(false)` does NOT clear `refreshFailed` (it is called on both success and
error paths; clearing here would erase the indicator before it renders).

### 7.4 AC#3 — `bucketsFetchedAt` semantics unchanged

The error path in `model.go` must NOT call `SetBucketFetchedAt`. Only the success path
(BucketsLoadedMsg with no error) calls `SetBucketFetchedAt`. This is the existing
behavior; the spec pins it as non-negotiable.

---

## 8. Render site

File: `internal/tui/components/quotadashboard.go` — `titleBar()` function.

The engineer adds `refreshFailedLabel` alongside the existing `refreshingLabel` in the
wide/narrow branch, then inserts the `refreshFailed` check between the `refreshing` and
`fetchedAt` checks:

```
// Pseudo-shape only — no Go code required

if w < 80:
    refreshingLabel    = " ↻"
    refreshFailedLabel = " ✗"    // styles.Warning
    freshPrefix        = " · "
else:
    refreshingLabel    = "  ⟳ refreshing…"
    refreshFailedLabel = "  ✗ refresh failed"  // styles.Warning
    freshPrefix        = "  updated "

if m.refreshing:
    // existing FB-063 branch
elif m.refreshFailed:
    // new FB-060 branch
    failLabel = warning.Render(refreshFailedLabel)
    candidate = baseLeft + failLabel
    if gap-guard passes: left = candidate
elif !m.fetchedAt.IsZero():
    // existing freshness branch (FB-043 + FB-059)
```

---

## 9. File locations

- **Primary change:** `internal/tui/components/quotadashboard.go` — `titleBar()` + new field + `SetRefreshFailed` + `SetBuckets` side-effect clear
- **Caller change:** `internal/tui/model.go` — error path for refresh must call `m.quota.SetRefreshFailed(true)`; success path already calls `SetBuckets` which clears it
- **No changes:** HelpOverlay, key handlers, status bar, FB-059 threshold logic

---

## 10. Acceptance criteria

Axis tags: `[Observable]`, `[Input-changed]`, `[Anti-regression]`, `[Integration]`.

1. **[Observable]** After a failed refresh with existing bucket data visible,
   `stripANSI(QuotaDashboard.View())` contains `"refresh failed"` (wide mode, w >= 80).
   Test: load buckets; call `SetRefreshFailed(true)` with `refreshing=false`; inject
   `width=100`; assert `"refresh failed"` present in `stripANSI(m.View())`.

2. **[Input-changed — fail → succeed cycle]** After fail state, calling `SetBuckets(buckets)`
   clears the indicator. Test:
   - State A: `SetRefreshFailed(true)`, `refreshing=false`, `width=100` →
     assert `"refresh failed"` present in View()
   - State B: call `SetBuckets(buckets)` on the same model →
     assert `"refresh failed"` absent in `stripANSI(m.View())`
   - States A and B must produce View() outputs that differ at the title bar substring.

3. **[Anti-regression — fetchedAt unchanged on error]** After `SetRefreshFailed(true)`,
   `m.fetchedAt` is identical to its pre-failure value. Test: record `fetchedAt` before
   failure; assert it is unchanged after `SetRefreshFailed(true)`.

4. **[Anti-regression — refreshing preempts failure]** When `refreshing=true` and
   `refreshFailed=true` simultaneously, View() contains `"refreshing"` and does NOT contain
   `"refresh failed"`. Test: set both fields; assert.

5. **[Anti-regression]** FB-043/FB-059/FB-063 existing tests green.

6. **[Integration]** `go install ./...` compiles; `go test ./internal/tui/...` green.

---

## 11. Non-goals

- Not adding a `[r] retry` affordance in the failure indicator (separate brief if warranted).
- Not changing error-dispatching logic in model.go beyond the `SetRefreshFailed` call.
- Not showing the failure indicator on initial-load errors (those use the full-pane `loadErr` path).
- Not persisting failure state across navigation (navigating away and back resets the component).
