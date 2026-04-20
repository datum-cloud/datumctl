# FB-023 — Sidebar resource grouping by service/product

**Status:** PENDING ENGINEER
**Priority:** P2
**Brief source:** `docs/tui-backlog.md` — search `### FB-023`
**Dependencies:** None (consumes existing `ResourceType.Group` field)
**Maps to:** REQ-TUI-043

---

## 1. Option selection: Option 1 — section headers in the flat list

**Why Option 1:**
The sidebar today has ~12–15 types. Option 1 ships the primary scannability benefit
(visual group clustering) with minimal structural change — reuse existing `bubbles/list`,
add a thin `headerItem` type, and a cursor-skip interceptor. The refactor risk is low
and the 23-AC test skeleton is already written for this option.

**Why not Option 2 (collapsible tree):**
Building a tree widget from scratch is a significant engineering effort. At current
type counts, collapsibility is not yet warranted — an operator can see all groups in
a single scroll. This is the right follow-up if type count exceeds ~25 or operators
request it. Deferred per brief non-goals.

**Why not Option 3 (tab strip):**
Digit keys `1`/`2`/`3`/`4` are already bound to pane switches (FB-016 convention).
Remapping or adding new tab-navigation keys would require its own AC set and
HelpOverlay update. For a scannability improvement, this complexity is not justified.
Deferred.

**No AC extensions needed** — the 23-AC brief skeleton covers Option 1 fully.

---

## 2. Header styling

### 2.1 Color and weight

Headers use `styles.Muted` + Bold. Rationale: muted keeps headers from competing with
the focused/selected type rows (which use `styles.Primary` Bold and `styles.Secondary`
for cursor highlights). Bold differentiates headers from unselected type rows (Muted,
not Bold).

No additional accent color. The bold-muted combination is already the "section label"
register used elsewhere in the TUI (quota group headers in `quotadashboard.go`).

### 2.2 Indent

Headers: `" "` prefix — 1 space. Type rows: `"  "` prefix — 2 spaces (unchanged from
current rendering). The 1-char offset places headers visually "above" their member
rows without requiring a separate indentation of the type rows themselves.

```
 NETWORKING
  gateways (3)
  httproutes
▸ ipaddresspools
 IAM
  policies
  roles
```

The selected-type glyph (`▸`) occupies position 1 (the first char of the 2-char prefix),
which is the same char column as the space in the header prefix. This creates a natural
visual ledger.

### 2.3 Casing

All-caps. Consistent with the brief's ASCII art and with standard TUI section-header
conventions (k9s, Kubernetes dashboard sections).

### 2.4 Truncation

If a header label is wider than `available - 1` chars (the 1-char indent consumed),
truncate to `available - 2` chars and append `…`. Uses the same truncation pattern as
type-name rows. In practice, all curated labels are ≤13 chars and fit comfortably at
typical sidebar widths (≥20 chars).

---

## 3. Curated group → display-name mapping

Confirmed known Datum API groups and their sidebar labels:

| API group | Display name | Notes |
|---|---|---|
| `networking.datumapis.com` | `NETWORKING` | Gateways, HTTPRoutes, IP pools, domains |
| `iam.datumapis.com` | `IAM` | Policies, roles, service accounts |
| `compute.datumapis.com` | `COMPUTE` | Deployments, workloads |
| `resourcemanager.datumapis.com` | `RESOURCE MGMT` | Projects, organizations |
| `""` (empty / core) | `CORE` | Kubernetes core types (Namespaces, Nodes, etc.) |

**Fallback for unknown groups:** `strings.ToUpper(group)` — raw group name in all-caps.
Example: `foo.bar.baz` → `FOO.BAR.BAZ`. This is the AC#3 behavior from the brief.
Note: the brief's design-rules section says "lowercase, rendered as-is" but the AC
skeleton says "uppercased raw group." The AC definition wins; all-caps is consistent
with the curated label register.

**Map location:** new file `internal/tui/data/groups.go` — package-level
`var groupDisplayNames = map[string]string{...}` plus a `GroupDisplayName(group string) string`
function. This keeps the mapping with the data layer and away from the renderer.

---

## 4. CORE placement rule — confirmed

`CORE` always renders last, regardless of alphabetical order. The sort for all other
groups is alphabetical by display name: `COMPUTE`, `IAM`, `NETWORKING`, `RESOURCE MGMT`,
then `CORE` appended at the end.

Unknown-group headers sort alphabetically among the curated headers by their uppercased
raw-group name, before `CORE`.

---

## 5. Sort order within groups

Types within each group sort alphabetically by `ResourceType.Name` (ascending,
case-insensitive). This is applied at `SetItems` time — the caller passes unsorted
`[]data.ResourceType`; `SetItems` sorts per-group before building the item list.

---

## 6. `bubbles/list` item model

Two item types implement `list.Item`:

```
resourceTypeItem{rt data.ResourceType}   // existing — unchanged contract
headerItem{label string}                 // new — non-selectable group header
```

`compactDelegate.Render` gains a branch for `headerItem`:
- render as: `" " + boldMuted(label)` — 1-char indent + bold muted text
- no cursor glyph, no count annotation, no truncation beyond §2.4 rule
- Height() and Spacing() unchanged (both 1 and 0) — headers occupy one line, no blank line after

`SetItems(types []data.ResourceType)` is the only public setter that changes. It:
1. Groups types by `Group` field
2. Sorts groups by display name (curated or fallback), `CORE` always last
3. Sorts types within each group by `Name`
4. Builds `[]list.Item` with interleaved `headerItem` values
5. Passes the list to `m.list.SetItems`
6. Restores cursor to the previously selected type (cursor preservation — §7)

---

## 7. Cursor skip logic

After `m.list.Update(msg)` processes a navigation key (`j`/`k`), `NavSidebarModel.Update`
checks if `m.list.SelectedItem()` is a `headerItem`. If so, advance in the same
direction as the key pressed: one more step forward for `j`/`down`, one more step back
for `k`/`up`.

This produces a single final position per keypress — no visual flicker. The list's
internal scroll follows the final position. The engineer may implement this either by:
(a) intercepting navigation keys before the list and driving cursor manually, or
(b) post-processing after the list update.

Either approach is acceptable; the behavior contract is the same: no header is ever
the resting cursor position after a user-driven `j`/`k` press.

The `SelectedType()` contract is unchanged: `headerItem` at cursor → returns
`(data.ResourceType{}, false)`. All callers of `SelectedType()` already handle the
`false` case. The skip logic is the primary guarantee; the `false` return is the
defensive backstop.

---

## 8. Cursor preservation on re-render

When `SetItems` is called with a new `[]data.ResourceType` (e.g., on
`ResourceTypesLoadedMsg`), the sidebar must:
1. Record `SelectedType()` before rebuilding the list (name string only — Group may have
   changed)
2. Rebuild the item list with headers
3. Scan the new list for the recorded name; if found, set cursor to its position
4. If not found (type was removed from the new context), leave cursor at position 0
   (the list's own default after `SetItems`) — which is the first selectable type

`NavSidebarModel` needs a helper that finds the item index for a given type name,
skipping `headerItem` positions when counting.

---

## 9. Context switch behavior

`ContextSwitchedMsg` clears the sidebar by calling `SetItems(nil)` or
`SetItems([]data.ResourceType{})`. No cursor preservation across context switches — the
new context has a different type list; cursor resets to position 0. This is the current
behavior, unchanged.

---

## 10. Render site

**Primary change:** `internal/tui/components/navsidebar.go`
- Add `headerItem` type
- Add `headerItem` branch in `compactDelegate.Render`
- Update `SetItems` to group + sort + interleave headers
- Update `Update` with cursor-skip post-processing
- Add internal cursor-preservation helper

**New file:** `internal/tui/data/groups.go`
- `groupDisplayNames` map (curated table from §3)
- `GroupDisplayName(group string) string` function

**No changes:** `model.go` (no new message types needed), key handlers, HelpOverlay,
`ResourceType` struct, `data/types.go`.

---

## 11. Acceptance criteria

The 23 ACs from the brief stand without extension (Option 1 selected). Reproduced with
implementation guidance:

1. **[Happy/Observable]** 3-group fixture: header substrings appear in expected order.
2. **[Happy/Observable]** Curated mapping: `networking.datumapis.com`→`NETWORKING`,
   `iam.datumapis.com`→`IAM`, `compute.datumapis.com`→`COMPUTE`, `""`→`CORE`.
3. **[Happy]** Unknown group `foo.bar.baz`→`FOO.BAR.BAZ` (uppercased raw group).
4. **[Happy]** Core types (empty `Group`) render under `CORE` header, last position.
5. **[Happy]** Within-group types sorted alphabetically by `Name`.
6. **[Repeat-press]** `j` from a type advances to next type, skipping same-group header
   boundary items. (Headers don't appear between same-group items, but header-adjacent
   navigation must not halt.)
7. **[Repeat-press]** `j` at last type of group N → first type of group N+1, skipping
   the N+1 header.
8. **[Repeat-press]** `k` at first type of group N → last type of group N-1, skipping
   the N header.
9. **[Repeat-press]** `j` at last type overall: no-op (no wrap). `k` at first: no-op.
10. **[Input-changed]** `ResourceTypesLoadedMsg` re-render: previously-selected type
    preserved if still in list; snaps to position 0 if removed.
11. **[Input-changed]** `ContextSwitchedMsg`: cursor resets to first type, no
    cross-context preservation.
12. **[Input-changed]** `j` traversal covers every type exactly once (count of presses
    = total type count across all groups).
13. **[Anti-behavior]** `Enter` on a header position (only reachable via explicit cursor
    set in tests): no-op.
14. **[Anti-behavior]** Digit keys `1`–`9` in NavPane: no group-jump; FB-016
    pane-switch behavior unchanged.
15. **[Edge]** Single-group fixture: one header, types below.
16. **[Edge]** All-core fixture (all `Group == ""`): one `CORE` header.
17. **[Edge]** Group with one type: header + one row; cursor behavior unchanged.
18. **[Edge]** Small-height render: list scrolls; headers scroll with types (not sticky).
19. **[Observable]** HelpOverlay NAVIGATION section unchanged (`[j/k] nav` only).
20. **[Observable]** Count annotation `<type> (N)` preserved within group (e.g.,
    `httproutes (3)` still appears under `NETWORKING`).
21. **[Refactor-parity]** REQ-TUI-005/006 tests green: border/active-color, cursor
    highlight, count annotation, name truncation.
22. **[Integration/Anti-regression]** FB-001–FB-021 test suite green.
    `SelectedType()` contract unchanged.
23. **[Integration]** `go install ./...` compiles cleanly.

---

## 12. Axis-coverage table (engineer fills at submission)

| AC | Axis | Test function |
|---|---|---|
| 1 | Happy / Observable | |
| 2 | Happy / Observable | |
| 3 | Happy | |
| 4 | Happy | |
| 5 | Happy | |
| 6 | Repeat-press | |
| 7 | Repeat-press | |
| 8 | Repeat-press | |
| 9 | Repeat-press / Anti-behavior | |
| 10 | Input-changed | |
| 11 | Input-changed | |
| 12 | Input-changed | |
| 13 | Anti-behavior | |
| 14 | Anti-behavior | |
| 15 | Edge | |
| 16 | Edge | |
| 17 | Edge | |
| 18 | Edge | |
| 19 | Observable | |
| 20 | Observable | |
| 21 | Refactor-parity | |
| 22 | Integration / Anti-regression | |
| 23 | Integration | |

---

## 13. Non-goals

- Not user-reorderable groups.
- Not collapsible groups (Option 2 deferred — follow-up brief if type count grows past ~25).
- Not a tab strip (Option 3 deferred — digit key conflicts with FB-016 pane-switcher).
- Not pinning / favorites.
- Not group-level search/filter (FB-021 `:` bar is the global search surface).
- Not auto-collapsing empty groups (header omitted when group has zero types in context).
- Not changing `ResourceType.Group` field semantics or discovery logic.
- REQ-TUI-037 mutation test-safety NOT in scope — pure display refactor.
