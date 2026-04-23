# datumctl — AI Coding Assistant Context

## What this project is

`datumctl` is the official CLI for **Datum Cloud**, a connectivity infrastructure platform. It is **NOT a Kubernetes CLI** — users manage Datum Cloud resources (organizations, projects, DNS zones, networking, etc.) with no Kubernetes knowledge required. `k8s.io/kubectl` is an internal implementation detail; the only Kubernetes-facing feature is the opt-in `auth update-kubeconfig` command for existing kubectl users.

**Module:** `go.datum.net/datumctl` | **Go:** 1.25+ | **Task runner:** [go-task](https://taskfile.dev)

---

## Critical constraints

- **Never** use `kubectl` in examples or help text — always `datumctl`
- **Never** reference Kubernetes-native resource types (pods, deployments, services, nodes) in user-facing text
- kubectl-only commands (`auth update-kubeconfig`, `auth whoami`, `auth can-i`) must say "kubectl users only" upfront in their help text
- Primary login is `datumctl login` (top-level) — not `auth login`; for CI/headless use `--no-browser`; for staging use `--hostname auth.staging.env.datum.net`
- Use `UserError` from `internal/errors` for user-facing errors (no stack traces shown)
- Prefer `--dry-run=server` and `datumctl diff -f` before destructive write operations; `delete` has no confirmation prompt
- The `activity` command comes from `go.miloapis.com/activity` — override its `Long` and `Example` in `root.go` after construction
- `WrapResourceCommand` applies to `get`, `delete`, `edit`, `describe` only; `apply`, `create`, `diff` set `GroupID` inline
- Use `templates.LongDesc()` for `Long` fields and `templates.Examples()` for `Example` fields on Cobra commands

---

## Adding and modifying commands

**Custom commands** (login, logout, auth subcommands, ctx, docs, mcp): add files under `internal/cmd/<name>/`, register the returned `*cobra.Command` in `root.go`.

**kubectl-wrapped commands** (get, delete, apply, create, edit, describe, diff, explain, api-resources, api-versions, version): override help text *after* the constructor call in `root.go`:

```go
cmd := get.NewCmdGet("datumctl", factory, ioStreams)
cmd.Short = "List or retrieve Datum Cloud resources"
cmd.Long = datumGetLong
cmd.Example = datumGetExample
```

---

## Non-obvious packages

```
internal/errors/      UserError — clean user-facing messages, no stack traces
internal/authutil/    OAuth2 PKCE, device flow, machine-account auth
internal/client/      DatumCloudFactory — org/project scope resolution + API clients
internal/discovery/   Org/project API discovery and context cache
internal/picker/      Interactive TUI context/account selector
```
