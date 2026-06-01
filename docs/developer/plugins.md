---
title: "Plugin System Design"
sidebar:
  order: 6
---

# datumctl Plugin System

This document covers the architectural design for the datumctl plugin system.
It is intended as a reference for contributors building the plugin infrastructure
and for teams authoring first-party plugins.

---

## Goals

- Let domain teams (compute, networking, billing, audit) ship CLI extensions
  independently without modifying core `datumctl`.
- Establish a stable contract that survives datumctl version upgrades.
- Keep the security blast radius of a third-party plugin small.
- Give plugin authors a clear, low-friction path to ship.

---

## How plugins work

A plugin is any executable named `datumctl-<name>` that is either:

1. **Managed** — installed via `datumctl plugin install` into
   `~/.datumctl/plugins/` and recorded in `plugins.json`.
2. **Unmanaged** — placed on the user's `PATH` by any other means.

When a user runs `datumctl <name>`, datumctl resolves the command using this
precedence order:

1. **Built-in commands** — always win. A plugin named `datumctl-get` or
   `datumctl-login` will never shadow a built-in. `datumctl plugin install`
   rejects any plugin whose name collides with a built-in at install time.
2. **Managed plugins** — binaries in `~/.datumctl/plugins/`.
3. **Unmanaged plugins** — binaries named `datumctl-<name>` found on `PATH`.

If a matching plugin is found (steps 2–3), datumctl execs it, passing through
all remaining arguments.

Unmanaged plugins trigger a one-time warning:

```
warning: 'datumctl-compute' is not a managed plugin and has not been verified.
```

---

## Directory layout

```
~/.datumctl/
  config                       # CLI configuration
  credentials.json             # stored credentials
  plugins/                     # managed plugin binaries
  plugins/plugins.json         # install record (see below)
  plugins/plugin-index.json    # cached plugin index
```

datumctl searches `plugins/` before `PATH` so managed and unmanaged plugins
are always distinguishable.

---

## Installation

### From the curated index

The curated plugin index lives at
[datum-cloud/datumctl-plugins](https://github.com/datum-cloud/datumctl-plugins).
Plugins listed there can be installed by name:

```sh
datumctl plugin install compute
datumctl plugin search
datumctl plugin list
datumctl plugin upgrade compute
datumctl plugin remove compute
```

The index is cached locally at `~/.datumctl/plugins/plugin-index.json`
and refreshed automatically when stale (default TTL: 1 hour). Override the
index URL with `DATUMCTL_PLUGIN_INDEX_URL` for testing.

### From a GitHub Release

Any plugin can be installed directly from a GitHub Release without being listed
in the curated index:

```sh
datumctl plugin install datum-cloud/datumctl-compute          # latest release
datumctl plugin install datum-cloud/datumctl-compute@v1.2.0   # specific version
```

This path requires a `checksums.txt` file alongside the release archives in
goreleaser's default two-column format.

### Restoring all plugins

Running `datumctl plugin install` with no arguments restores all plugins
recorded in `plugins.json` — useful for reproducing a plugin set on a new
machine.

---

## plugins.json schema

`plugins.json` records every managed install. It is the source of truth for
`plugin list`, `plugin upgrade`, and `plugin remove`.

```json
{
  "plugins": {
    "compute": {
      "source": "compute",
      "version": "v0.8.0",
      "sha256": "abc123...",
      "installed_at": "2026-05-26T00:00:00Z",
      "manifest": {
        "name": "compute",
        "version": "v0.8.0",
        "description": "Deploy and manage containerized workloads on Datum Cloud",
        "api_version": 1
      }
    }
  }
}
```

`source` is either the short index name (e.g. `compute`) or a
`github.com/owner/repo` path for direct GitHub installs.

---

## Plugin manifest

Every plugin binary must respond to `--plugin-manifest` with a JSON document
on stdout:

```json
{
  "name": "compute",
  "version": "v0.8.0",
  "description": "Deploy and manage containerized workloads on Datum Cloud",
  "min_datumctl_version": "v0.10.0",
  "api_version": 1,
  "min_api_version": 1
}
```

datumctl reads this manifest at install time to validate compatibility. If a
plugin does not respond to `--plugin-manifest`, datumctl treats it as
unversioned and skips compatibility checks.

---

## Context passthrough

datumctl sets the following environment variables before execing a plugin:

| Variable                   | Value                                              |
|----------------------------|----------------------------------------------------|
| `DATUM_ORG`                | Current organization slug                          |
| `DATUM_PROJECT`            | Current project slug (may be empty)                |
| `DATUM_API_HOST`           | API base URL (e.g. `api.datum.net`)                |
| `DATUM_PLUGIN_API_VERSION` | Integer API version (currently `1`)                |
| `DATUM_CREDENTIALS_HELPER` | Absolute path to the datumctl binary               |
| `DATUM_SESSION`            | Active session name (may be empty)                 |

**Tokens are not passed as environment variables.** Plugins fetch a token on
demand via the credentials helper:

```sh
$DATUM_CREDENTIALS_HELPER auth get-token --session $DATUM_SESSION
```

Omit `--session` when `DATUM_SESSION` is empty. The Go SDK's `plugin.Token()`
handles this automatically.

### Why not `DATUM_TOKEN`?

Passing a raw token in an environment variable freezes the auth mechanism —
every plugin that reads `DATUM_TOKEN` directly must be updated if tokens become
shorter-lived, audience-scoped, or replaced by a different credential type.
The credentials helper insulates plugins from these changes entirely.

---

## `DATUM_PLUGIN_API_VERSION`

This integer increments only when the plugin contract (env var names, manifest
schema, credentials helper interface) changes in a breaking way. Plugin authors
check this value if they need to handle multiple datumctl generations. It is
independent of datumctl's own semver version.

Current version: **1**

---

## Go SDK

First-party and community plugins written in Go should use:

```
go.datum.net/datumctl/plugin
```

The SDK provides:

- `plugin.Context()` — reads all `DATUM_*` env vars into a typed struct.
- `plugin.Token()` — calls the credentials helper and returns a token string.
- `plugin.NewRootCmd(name, short)` — returns a pre-configured `*cobra.Command`
  with `--org`, `--project`, and `--output` flags wired to the injected context.
- `plugin.ServeManifest(m)` — handles `--plugin-manifest` and exits before
  Cobra runs.

See `examples/plugin-dns/` for a working reference implementation.

Plugins written in other languages can implement the same contract manually —
the protocol is just environment variables and a subprocess call.

---

## Security model

| Plugin type | Token access | Verification |
|-------------|--------------|--------------|
| Managed (index) | On demand via helper | SHA256 verified against index manifest |
| Managed (GitHub) | On demand via helper | SHA256 verified against `checksums.txt` |
| Unmanaged | On demand via helper | None — user warning shown |

Because tokens are fetched on demand rather than injected at startup, a plugin
process that exfiltrates its environment variables does not automatically
capture a usable credential. A determined attacker can still call the helper,
but this raises the bar meaningfully over raw env var injection.

Future: audience-scoped tokens (e.g., `datumctl auth get-token
--audience=dns.datum.net`) will let datumctl issue tokens that are only valid
for a specific plugin's API surface.

---

## Compatibility and versioning

Plugin authors declare a minimum datumctl version in their manifest. datumctl
validates this at install time and warns (but does not block) at invocation if
the running version is below the declared minimum.

datumctl guarantees that `DATUM_PLUGIN_API_VERSION=1` env vars and the
credentials helper interface are stable for the lifetime of API version 1.
Breaking changes increment the version and are announced in release notes with
a migration guide.

---

## V1 scope

| Component | Status |
|-----------|--------|
| PATH shim (`datumctl <name>`) | V1 |
| Managed install dir + `plugins.json` | V1 |
| `datumctl plugin install/list/upgrade/remove` | V1 |
| Curated plugin index (`datum-cloud/datumctl-plugins`) | V1 |
| Direct GitHub Release install (`owner/repo[@version]`) | V1 |
| ENV context passthrough | V1 |
| Credentials helper (`datumctl auth get-token`) | V1 |
| Plugin manifest (`--plugin-manifest`) | V1 |
| Go SDK (`go.datum.net/datumctl/plugin`) | V1 |
| Reference first-party plugin (`compute`) | V1 |
| TUI panel extension points | V2 |
| MCP tool registration | V2 |
| `datumctl plugin new` scaffolding | V2 |
| Audience-scoped tokens | V2 |
