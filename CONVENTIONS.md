# datumctl — Conventions (Aider)

## What this project is

`datumctl` is the official CLI for **Datum Cloud**, a connectivity infrastructure platform. It is **NOT a Kubernetes CLI** — `k8s.io/kubectl` is an internal implementation detail. Users manage Datum Cloud resources with no Kubernetes knowledge required.

**Module:** `go.datum.net/datumctl` | **Go:** 1.25+

---

## Conventions

- **Never** use `kubectl` in examples or help text — always `datumctl`
- **Never** reference Kubernetes-native resource types (pods, deployments, services, nodes) in user-facing text
- kubectl-only commands (`auth update-kubeconfig`, `auth whoami`, `auth can-i`) must say "kubectl users only" upfront
- Primary login is `datumctl login` — not `auth login`
- Use `UserError` from `internal/errors` for user-facing errors (no stack traces shown to users)
- Prefer `--dry-run=server` and `datumctl diff -f` before destructive write operations
- `activity` command is from `go.miloapis.com/activity` — override its help text in `root.go` after construction
- `WrapResourceCommand` applies to `get`, `delete`, `edit`, `describe` only; `apply`, `create`, `diff` set `GroupID` inline
- Use `templates.LongDesc()` for `Long` fields and `templates.Examples()` for `Example` fields on Cobra commands

---

## Adding commands

**Custom commands** (login, auth, ctx, docs, mcp): add under `internal/cmd/<name>/`, register in `root.go`. **kubectl-wrapped commands** (get, delete, apply, etc.): override `Short`/`Long`/`Example` after the constructor in `root.go`.
