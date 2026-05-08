# Changelog

## v0.15.0 — Your terminal just got a lot more interesting

Until now, getting a picture of your Datum Cloud project meant stringing together `get` commands and mentally assembling the results. That changes today. This release ships `datumctl console` — a full interactive terminal UI that turns your project into something you can actually *navigate*. Plus: a breaking rename you'll be glad happened before GA, structured error output for the automation crowd, and a fix for a sneaky bug that was making project-scoped resources vanish into thin air.

---

### New: `datumctl console` ([#166](https://github.com/datum-cloud/datumctl/pull/166))

Just run `datumctl console`. No flags, no arguments — it opens straight into a live view of your project.

**Resource browser** — The left sidebar lists every resource type in your project, grouped by API category. Navigate with arrow keys or jump directly with letter shortcuts. The main panel renders a live, filterable, sortable table of resources for whatever type is selected. Pick any row and a detail pane opens alongside it: description, status conditions, Kubernetes events, raw YAML, and full change history, all accessible as tabs without leaving the screen.

**Quota dashboard** (press `3`) — Every governed resource type, your current usage, and how much headroom you have left — all in one place. The numbers refresh automatically and carry a freshness timestamp so you always know if you're looking at live data or something a few seconds stale. If the quota service goes sideways, the title bar tells you and `[r]` fires an immediate retry.

**Activity feed** (press `4`) — A scrollable timeline of every recent change across the project. Select any entry to expand a structured diff showing exactly what changed and when. Great for answering "wait, who changed that?" without opening another tab.

**Welcome panel** — First time in, you get a quick-start guide and a cheat sheet of key bindings. After that, the panel switches to a summary view: platform health, recent activity, and quota status at a glance, so you can spot anything worth attention before you start drilling.

The UI adapts gracefully to narrow terminals, collapsing hints and labels rather than breaking layout. A persistent status bar shows context-sensitive hints at all times, and `?` opens a full keybinding reference whenever you need it.

---

### Machine accounts are now service accounts ([#164](https://github.com/datum-cloud/datumctl/pull/164))

"Machine account" was always a bit of an odd name. In v0.15.0, it's gone — replaced by **service account** everywhere it appeared: flags, credential types, and the key file directory, which moves from `~/.config/datumctl/machine-accounts/` to `~/.config/datumctl/service-accounts/`. This is a pre-GA breaking rename to align with what the rest of the industry calls this concept.

If you have scripts or tooling that reference `machine-account` paths or credential types, update them before upgrading.

---

### Structured error output for agents and automation ([#161](https://github.com/datum-cloud/datumctl/pull/161))

Pass `--error-format=json` (or `yaml`) and errors come back as machine-readable structured output instead of human prose. The default is unchanged — plain text for humans, structured output when you ask for it. Useful when you're wrapping `datumctl` in a script, CI pipeline, or an AI coding agent that needs to do something with the error beyond just printing it.

---

### Shell completions actually work now ([#150](https://github.com/datum-cloud/datumctl/pull/150))

A bug was silently blocking resource name and ID completion — the kind of thing you'd only notice once you'd been tab-completing into the void for a while. Fixed across `bash`, `zsh`, `fish`, and PowerShell. One caveat: completing project-scoped resource names still requires passing `--organization` or `--project` explicitly. That constraint goes away once those flags become optional.

---

### Bug fixes

**The vanishing resources bug** ([#165](https://github.com/datum-cloud/datumctl/pull/165)) — Running `datumctl get serviceaccounts --project <id>` returned `the server doesn't have a resource type "serviceaccounts"` even when the service account was sitting right there. The culprit: the discovery client was ignoring the `--project` flag entirely and always hitting the user control plane, where project-scoped resource types are intentionally filtered out. The error message was misleading, the silent fallback was the real problem, and it's fixed.

