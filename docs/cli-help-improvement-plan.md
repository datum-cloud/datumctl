# datumctl CLI Help Text Improvement Plan

## Guiding Principle: datumctl is the primary tool

**datumctl is the complete, first-class CLI for Datum Cloud.** Users should be able to log in, manage all their resources, and accomplish everything they need using datumctl commands directly — no knowledge of Kubernetes or kubectl required.

kubectl integration (`auth update-kubeconfig`, `auth get-token`, `auth whoami`, `auth can-i`) is an **advanced, opt-in feature** for users who already use kubectl and want to point it at a Datum Cloud control plane. It should never appear in the primary getting-started flow and should be clearly scoped as "for kubectl users" wherever it is mentioned.

This principle must be applied consistently throughout all help text: kubectl must not appear in primary examples, default workflows, or any description read by a new user.

---

## Overview of the Problems

datumctl presents two distinct sets of commands to users, and each set has a different category of help-text problem.

**Category 1 — Custom auth and tooling commands** (`auth`, `mcp`, `docs`, and their subcommands). These were written specifically for datumctl. Most have an adequate one-line `Short` description, but none have a full `Long` description, and none have an `Example` block. Users who run `datumctl auth login --help` learn almost nothing about what the command actually does, what flags are meaningful, or what to run next.

**Category 2 — Wrapped kubectl commands** (`get`, `delete`, `create`, `apply`, `edit`, `describe`, `diff`, `explain`, `api-resources`, `api-versions`, `version`). These are imported directly from `k8s.io/kubectl/pkg/cmd` and their help text is the verbatim kubectl text. Every example refers to `kubectl`, to Kubernetes-native resource types such as pods, deployments, replicationcontrollers, and services, and to Kubernetes-specific concepts such as grace-period pod termination, node scheduling, and RBAC cluster roles. A Datum Cloud user who runs `datumctl get --help` and sees "List all pods in ps output format" is confused and misled: datumctl has no pods.

---

## Known Issues and Gaps (plan review findings)

The following issues were identified during review and must be addressed before or during implementation.

### Blocking

**1. `activity` command is entirely missing from this plan.**
`root.go` adds a top-level `activity` command (with `audit`, `events`, `feed`, `history` subcommands) from the external package `go.miloapis.com/activity`. Its Long description currently contains `kubectl activity` examples throughout — a direct violation of the guiding principle. Since it is externally constructed, the fix is the same pattern as other kubectl-wrapped commands: override `.Long` and `.Example` on the returned command object in `root.go` after construction. This must be added to the plan as P0/P1.

**2. `logout` current Long description is more severely broken than described.**
The current Long says users should specify `email@hostname` format, but the code actually stores and looks up users by plain email address only. This is stale documentation from a previous implementation — it is actively wrong, not just overly technical.

**3. Implementation approach description is inaccurate about `WrapResourceCommand`.**
`WrapResourceCommand` is only applied to `get`, `delete`, `edit`, and `describe`. `apply`, `create`, and `diff` set their `GroupID` manually inline — they do not go through `WrapResourceCommand`. Developers following this plan should not assume it is a universal wrapper.

### Significant

**4. `logout` `Use` string should also be updated.**
Current: `logout [user]`. Should be updated to `logout <email>` to match `switch <user-email>` convention and reflect that the argument is specifically an email address.

**5. `auth update-kubeconfig` proposed Long omits the `--hostname` flag.**
The command has a `--hostname` flag to override the API server hostname (useful for self-hosted environments). This is not mentioned in the proposed Long text and should be added.

**6. `auth login` proposed Long/Example omits `--api-hostname` flag.**
The login command has an `--api-hostname` flag for specifying the API hostname separately from the auth hostname. This is relevant for the "self-hosted environment" example already in the plan but is not documented.

**7. `docs openapi` proposed text omits `--no-browser` flag.**
The command has `--no-browser` to suppress automatic browser opening. The proposed Long says "Your browser opens automatically" without mentioning the opt-out. This matters for headless/CI environments.

**8. `docs openapi` proposed Example omits `--platform-wide` flag.**
The command supports `--platform-wide` which is mutually exclusive with `--organization` and `--project`. The proposed Example does not demonstrate this flag.

**9. `generate-cli-docs` current Example is broken (runtime error), not just a style issue.**
The `-o` short flag does not exist for this command — running the current Example would produce a runtime error. The plan should characterize this as a broken example, not just a wrong flag name.

**10. `KUBE_EDITOR` and `KUBECTL_EXTERNAL_DIFF` conflict with the guiding principle.**
The proposed text for `edit` (featuring `KUBE_EDITOR`) and `diff` (featuring `KUBECTL_EXTERNAL_DIFF`) use kubectl-branded environment variable names as primary examples. These are the actual variable names the underlying code reads, so they are factually accurate. Two options to resolve the tension:
- For `edit`: mention `EDITOR` first, then `KUBE_EDITOR` as a secondary note ("also supports `KUBE_EDITOR` for compatibility with kubectl workflows").
- For `diff`: consider adding a small code change to also check `DATUMCTL_EXTERNAL_DIFF` as a primary alias, or add a parenthetical noting this is an inherited variable name.

**11. `datumctl get` tip about `organizations` shorthand applies to other commands too.**
`WrapResourceCommand` applies the `organizations → organizationmemberships` alias to `get`, `delete`, `edit`, and `describe`. The tip in the proposed `get` Long text implies it is `get`-specific. Either broaden the wording or add a similar note to `delete`, `edit`, and `describe`.

### Moderate

**12. `api-resources` proposed Long incorrectly calls all resources "Custom Resource Definitions."**
The command lists all API resource types served by the platform, which may not all be CRDs technically. Remove the parenthetical "(Custom Resource Definitions)" or replace with "resource types" to be accurate.

**13. Priority for `auth whoami` and `auth can-i` (P4) is based on frequency, not severity.**
These two commands have the most pervasive kubectl text of any commands in the plan — their entire Long descriptions and all examples are verbatim kubectl content about Kubernetes-specific concepts. The P4 priority reflects expected low usage, not low severity of the documentation problem. This should be noted.

### Minor

**14. `generate-cli-docs` Long also contains the wrong flag name `(-o)`.**
The current Long says "The directory where you place the documentation (-o) should exists" — `-o` is wrong here too (flag is `--output-dir`), not just in the Example. The proposed replacement Long fixes this implicitly but the plan's diagnosis only mentions it in the Example.

**15. `root.go` is missing from "Files to Edit" for the root command update.**
The root command's `Short` and `Long` updates should be listed under Files to Edit. Currently only kubectl-wrapped overrides are listed for `root.go`.

**16. `mcp` proposed Long tool list formatting will render poorly in a terminal.**
~~The multi-line tool pairs with trailing commas (e.g., `create_resource,` followed by `get_resource,` on the next line) will look awkward in `--help` output.~~ **Resolved.** The five CRUD tools are now grouped on consecutive lines with the shared description "Generic CRUD for any Datum Cloud resource type" appearing next to `create_resource, get_resource,` on the first line, so all five tools are visually grouped and no tool appears without context.

**17. `version` proposed Long uses kubectl-centric language.**
"when a control plane context is configured" is opaque for datumctl-primary users. Replace with "when you are logged in and have an organization or project configured."

---

## Flag Cleanup (Implemented)

kubectl-inherited flags that are irrelevant or confusing to Datum Cloud users are now hidden using `MarkHidden`. Hidden flags still function if passed explicitly — they simply don't appear in `--help` output.

Two helper functions were added to `internal/cmd/root.go`:
- `hideFlags(cmd, flags...)` — hides per-command flags
- `hidePersistentFlags(cmd, flags...)` — hides persistent/global flags

### Global flags hidden on `rootCmd`

| Flags | Reason |
|---|---|
| `as`, `as-group`, `as-uid`, `as-user-extra` | kubectl impersonation — not applicable to Datum auth model |
| `certificate-authority`, `insecure-skip-tls-verify`, `tls-server-name` | TLS config — handled internally |
| `server`, `token`, `user` | kubectl auth — users should use `datumctl auth login` |
| `log-flush-frequency`, `v`, `vmodule`, `disable-compression` | Internal/logging — not user-facing |
| `warnings-as-errors` | kubectl internal |

### Per-command flags hidden

| Command | Flags hidden |
|---|---|
| `get`, `delete`, `edit`, `describe` | `allow-missing-template-keys`, `chunk-size`, `kustomize`, `output-watch-events`, `raw`, `server-print`, `show-managed-fields`, `subresource`, `template` |
| `create` | `allow-missing-template-keys`, `kustomize`, `template`, `save-config` |
| `apply` | `allow-missing-template-keys`, `kustomize`, `template`, `server-dry-run`, `prune-allowlist` |
| `diff` | `allow-missing-template-keys`, `kustomize`, `template` |
| `explain` | `api-version` |

---

## Implementation Approach

### For custom commands (auth, mcp, docs subcommands)

Edit the respective source files under `internal/cmd/auth/`, `internal/cmd/mcp/mcp.go`, and `internal/cmd/docs/`. Assign `Long` and `Example` fields directly on the `cobra.Command` struct literal. Follow the existing pattern used by `switchCmd` and `logoutCmd`, which already have `Long` fields.

### For kubectl-wrapped commands

After each constructor call in `internal/cmd/root.go`, add field assignments:

```go
cmd := get.NewCmdGet("datumctl", factory, ioStreams)
cmd.Short = "List Datum Cloud resources"
cmd.Long = datumGetLong
cmd.Example = datumGetExample
```

These assignments happen at process startup before any command is parsed, so they take full effect for all help rendering and auto-generated docs. The `WrapResourceCommand` helper in `internal/cmd/utils.go` handles `GroupID`; the individual `Short`/`Long`/`Example` overrides should live in `root.go`.

---

## Priority Order

| Priority | Commands | Reason |
|---|---|---|
| P0 | `auth login`, `auth logout`, `auth list`, `auth switch` | Every new user hits these within the first five minutes |
| P0 | `get`, `apply`, `create`, `delete` | Core resource management commands; all currently show kubectl/Kubernetes examples |
| P0 | `activity` | Top-level command with pervasive `kubectl activity` examples; from external package — must be overridden in `root.go` |
| P1 | `describe`, `edit`, `diff` | Everyday resource management; kubectl examples are misleading |
| P1 | `mcp` | Already has reasonable `Long`; needs an `Example` block |
| P2 | `explain`, `api-resources`, `api-versions`, `version` | Discoverability helpers; kubectl examples still say `kubectl` |
| P2 | `docs openapi`, `docs generate-cli-docs` | Internal/power-user tools |
| P3 | `auth get-token`, `auth update-kubeconfig` | kubectl integration only — secondary use case, clearly scoped as such |
| P4 | `auth whoami`, `auth can-i` | kubectl integration only — lowest expected usage (note: these have the *most* pervasive kubectl text of any commands; priority is based on frequency, not severity) |

---

## Command-by-Command Analysis and Proposed Text

---

### `datumctl` (root command)

**Current Short:** `A CLI for interacting with the Datum platform`
**Current Long:** none | **Current Example:** none

**Problems:** Says "the Datum platform" but the product name is "Datum Cloud". No long description orienting a new user.

**Proposed Short:**
```
The official CLI for Datum Cloud
```

**Proposed Long:**
```
datumctl is the official command-line interface for Datum Cloud, the connectivity
infrastructure platform for developers and forward-thinking companies.

Use datumctl to authenticate and manage all your Datum Cloud resources —
projects, organizations, networking, compute, and more — directly from the
terminal. No knowledge of Kubernetes or kubectl required.

Get started:
  datumctl auth login
  datumctl get organizations
  datumctl get projects
```

---

### `datumctl auth`

**Current Short:** `Authenticate with Datum Cloud`
**Current Long:** none | **Current Example:** none

**Problems:** Functional short description, but no orientation about what subcommands exist and when to use them.

**Proposed Short:**
```
Manage Datum Cloud authentication credentials
```

**Proposed Long:**
```
The auth group provides commands to log in to Datum Cloud, manage multiple
user sessions, and retrieve tokens for scripting.

Typical workflow:
  1. Log in:          datumctl auth login
  2. Verify sessions: datumctl auth list
  3. Switch accounts: datumctl auth switch <email>
  4. Log out:         datumctl auth logout <email>

Advanced — kubectl integration:
  If you use kubectl and want to point it at a Datum Cloud control plane,
  see 'datumctl auth update-kubeconfig --help'.
```

---

### `datumctl auth login`

**Current Short:** `Authenticate with Datum Cloud via OAuth2 PKCE flow`
**Current Long:** none | **Current Example:** none

**Problems:** Short description exposes an implementation detail (OAuth2 PKCE) that most users don't need. No long description explains the browser flow. No examples.

**Proposed Short:**
```
Log in to Datum Cloud
```

**Proposed Long:**
```
Authenticate with Datum Cloud using a secure browser-based login flow.

Running this command will:
  1. Open your default web browser to the Datum Cloud authentication page.
     If the browser cannot open automatically, a URL is printed for manual use.
  2. Complete authentication in the browser (username/password or SSO).
  3. Return to datumctl, which stores your credentials (including a refresh
     token) securely in the system keyring.

After login, credentials are associated with your email address and
automatically refreshed when they expire. Use 'datumctl auth list' to see
all stored sessions.

By default, logs into auth.datum.net (the production Datum Cloud environment).
Use --hostname to target a different environment (e.g., staging).
```

**Proposed Example:**
```
# Log in to Datum Cloud (opens browser)
datumctl auth login

# Log in to a staging environment
datumctl auth login --hostname auth.staging.env.datum.net

# Log in to a self-hosted environment with an explicit client ID
datumctl auth login --hostname auth.example.com --client-id 123456789
```

---

### `datumctl auth logout`

**Current Short:** `Remove local authentication credentials for a specified user or all users`
**Current Long:** present (basic) | **Current Example:** none

**Problems:** Short description is wordier than necessary. Long description uses the internal `email@hostname` key format that users don't normally see. No examples.

**Proposed Short:**
```
Remove stored credentials for a user or all users
```

**Proposed Long:**
```
Remove locally stored Datum Cloud credentials from the system keyring.

Provide the email address of the user to log out (as shown by
'datumctl auth list'). Use --all to remove credentials for every
logged-in user at once.

If you log out the currently active user, the active session is cleared.
You will need to run 'datumctl auth login' again before running commands
that require authentication.
```

**Proposed Example:**
```
# Log out a specific user
datumctl auth logout user@example.com

# Log out all authenticated users
datumctl auth logout --all
```

---

### `datumctl auth list`

**Current Short:** `List locally authenticated users`
**Current Long:** none | **Current Example:** none

**Problems:** Good short description, but no explanation of what the table columns mean or how to act on the output.

**Proposed Short:** *(keep as-is)*

**Proposed Long:**
```
Display a table of all Datum Cloud users whose credentials are stored
locally in the system keyring, along with their status.

Columns:
  Name    The display name from the user's Datum Cloud account.
  Email   The email address used to log in. Pass this to 'datumctl auth switch'
          or 'datumctl auth logout' to act on a specific account.
  Status  "Active" marks the account whose credentials are used by default
          for all subsequent datumctl commands.
```

**Proposed Example:**
```
# Show all logged-in users
datumctl auth list

# Alias
datumctl auth ls
```

---

### `datumctl auth switch`

**Current Short:** `Set the active authenticated user session`
**Current Long:** present (decent) | **Current Example:** none

**Problems:** No examples. Long description starts with "Switches" — should use imperative or noun phrase by convention.

**Proposed Short:**
```
Switch the active Datum Cloud user session
```

**Proposed Long:**
```
Change which locally stored user account is treated as active.

The active user's credentials are used for all datumctl commands that
require authentication.

The email address must match an account shown by 'datumctl auth list'.
To add a new account, run 'datumctl auth login' first.
```

**Proposed Example:**
```
# See which accounts are available
datumctl auth list

# Switch to a different account
datumctl auth switch user@example.com
```

---

### `datumctl auth get-token`

**Current Short:** `Retrieve access token for active user (raw or K8s format)`
**Current Long:** present (decent) | **Current Example:** none

**Problems:** Short description has awkward "(raw or K8s format)" parenthetical. This is a kubectl integration command — most datumctl users will never need it. The help text should say so clearly upfront rather than implying it is a general-purpose command.

**Proposed Short:**
```
Print the active user's access token (advanced / kubectl integration)
```

**Proposed Long:**
```
Print the current access token for the active Datum Cloud user.

Most datumctl users do not need this command — datumctl handles
authentication automatically for all its own commands.

This command exists for two advanced use cases:

  1. kubectl credential plugin: invoked automatically by kubectl after you
     run 'datumctl auth update-kubeconfig'. You do not need to call it
     directly in that case.

  2. Scripting or direct API calls: use --output=token to get a raw bearer
     token to pass to curl or other HTTP clients.

If the stored token is expired, datumctl automatically uses the stored
refresh token to obtain a new one before printing.

Output formats (--output / -o):
  token                         Print the raw access token (default).
  client.authentication.k8s.io/v1  Print a Kubernetes ExecCredential JSON
                                object for kubectl credential plugin use.
```

**Proposed Example:**
```
# Get a raw token for use in a script or direct API call
datumctl auth get-token

# Get a Kubernetes ExecCredential JSON object (used by kubectl automatically)
datumctl auth get-token --output=client.authentication.k8s.io/v1
```

---

### `datumctl auth update-kubeconfig`

**Current Short:** `Update the kubeconfig file`
**Current Long:** none | **Current Example:** none

**Problems:** Short description is extremely sparse. More importantly, this is a kubectl-specific integration command that most datumctl users will never need. The help text must clearly say "this is for kubectl users" upfront, so new users aren't confused by it appearing in the auth group.

**Proposed Short:**
```
Configure kubectl to access a Datum Cloud control plane (kubectl users only)
```

**Proposed Long:**
```
For kubectl users only. datumctl users do not need this command —
manage your resources directly with 'datumctl get', 'datumctl apply', etc.

This command adds or updates a cluster, user, and context entry in your
kubeconfig file so that kubectl can authenticate to a Datum Cloud control
plane using your active datumctl session.

After running this command, kubectl will automatically call
'datumctl auth get-token' to obtain a fresh credential on each request.

You must specify exactly one of --organization or --project:

  --organization <id>   Configure kubectl access to an organization's
                        control plane.
  --project <id>        Configure kubectl access to a specific project's
                        control plane.

The kubeconfig is updated at $HOME/.kube/config by default, or the path
set by the KUBECONFIG environment variable. Use --kubeconfig to override.
```

**Proposed Example:**
```
# Configure kubectl for an organization's control plane
datumctl auth update-kubeconfig --organization my-org-id

# Configure kubectl for a specific project's control plane
datumctl auth update-kubeconfig --project my-project-id

# Write to a custom kubeconfig file
datumctl auth update-kubeconfig --organization my-org-id --kubeconfig ~/.kube/datum-config
```

---

### `datumctl auth whoami`

**Current Short (from kubectl):** `Experimental: Check self subject attributes`
**Problems:** Long description mentions "Kubernetes cluster", "token webhook", "auth proxy". Example says `kubectl auth whoami`. This is a kubectl integration command — it requires a control plane context configured via `update-kubeconfig`.

**Proposed Short:**
```
Show your identity on a Datum Cloud control plane (kubectl users only)
```

**Proposed Long:**
```
For kubectl users only. Requires a control plane context configured via
'datumctl auth update-kubeconfig'.

Queries the Datum Cloud API server to display the user identity and group
memberships it has resolved from your current credentials. Useful for
confirming which account kubectl is using and troubleshooting access-denied
errors against the control plane.

To see which datumctl account is currently active (without needing a
control plane context), use 'datumctl auth list' instead.
```

**Proposed Example:**
```
# Show your identity on the configured control plane
datumctl auth whoami

# Show your identity in JSON format
datumctl auth whoami -o json
```

---

### `datumctl auth can-i`

**Current Short (from kubectl):** `Check whether an action is allowed`
**Problems:** Long description and all examples use `kubectl` and Kubernetes resources (pods, deployments). This is a kubectl integration command.

**Proposed Short:**
```
Check permissions on a Datum Cloud control plane (kubectl users only)
```

**Proposed Long:**
```
For kubectl users only. Requires a control plane context configured via
'datumctl auth update-kubeconfig'.

Verify whether the active user has permission to perform a specific action
against a Datum Cloud resource type on the configured control plane.

VERB is an API verb: get, list, watch, create, update, patch, delete, or '*'.
TYPE is a Datum Cloud resource type (e.g., projects, dnszones, domains).

Use 'datumctl api-resources' to see all available resource types.
```

**Proposed Example:**
```
# Check if you can list projects on the control plane
datumctl auth can-i list projects

# Check if you can create DNS zones
datumctl auth can-i create dnszones

# List all your permitted actions
datumctl auth can-i --list

# List permitted actions in a specific namespace
datumctl auth can-i --list --namespace default
```

---

### `datumctl get`

**Current Short (from kubectl):** `Display one or many resources`
**Problems:** Every example references `kubectl` and Kubernetes-native resources (pods, replicationcontrollers, deployments). Completely irrelevant to a Datum Cloud user.

**Proposed Short:**
```
List or retrieve Datum Cloud resources
```

**Proposed Long:**
```
Display one or more Datum Cloud resources in a formatted table, or in JSON
or YAML for scripting and inspection.

Use the --organization or --project flags to target a specific context.
Use 'datumctl api-resources' to see all available resource types.

Tip: 'datumctl get organizations' is a shorthand that resolves to
organization memberships so you can see which organizations you belong to.
```

**Proposed Example:**
```
# List all projects
datumctl get projects

# List your organization memberships
datumctl get organizations

# List DNS zones in a specific namespace
datumctl get dnszones --namespace default

# Get a specific project by name
datumctl get project my-project-id

# List all resources of a type across all namespaces
datumctl get dnszones --all-namespaces

# Get a project and output as YAML
datumctl get project my-project-id -o yaml

# Get a project and output as JSON
datumctl get project my-project-id -o json

# Watch for changes to projects
datumctl get projects --watch
```

---

### `datumctl delete`

**Current Short (from kubectl):** `Delete resources by file names, stdin, resources and names, or by resources and label selector`
**Problems:** Long description is almost entirely about Kubernetes pod graceful deletion semantics. Examples reference pods, services, nodes.

**Proposed Short:**
```
Delete Datum Cloud resources
```

**Proposed Long:**
```
Delete one or more Datum Cloud resources by name, label selector, or
by providing a resource manifest file.

Resources can be specified as TYPE NAME pairs, or from a YAML/JSON file
with -f. JSON and YAML formats are accepted.

Note: this command does not perform a version check before deletion. If
someone has updated a resource between when you fetched it and when you
delete it, the deletion still proceeds. Use --dry-run=client to preview
what would be deleted before committing.
```

**Proposed Example:**
```
# Delete a project by name
datumctl delete project my-project-id

# Delete a DNS zone by name
datumctl delete dnszone my-zone --namespace default

# Delete resources defined in a manifest file
datumctl delete -f ./my-resource.yaml

# Delete all resources of a type in a namespace
datumctl delete dnszones --all --namespace default

# Delete resources with a specific label
datumctl delete dnszones -l environment=dev

# Preview what would be deleted without actually deleting
datumctl delete project my-project-id --dry-run=client
```

---

### `datumctl create`

**Current Short (from kubectl):** `Create a resource from a file or from stdin`
**Problems:** Examples reference `kubectl` and `pod.json`. Note: datumctl's `create` command strips all kubectl subcommands (namespace, secret, deployment, etc.) — only the raw `-f` invocation is available.

**Proposed Short:**
```
Create a Datum Cloud resource from a file or stdin
```

**Proposed Long:**
```
Create a new Datum Cloud resource by providing a manifest in YAML or JSON
format, either from a file or piped through stdin.

datumctl create accepts Datum Cloud resource manifests — not Kubernetes
built-in resources. Use 'datumctl apply' for idempotent creation or updates.

Resource manifests must specify the correct apiVersion and kind for the
Datum Cloud resource type. Use 'datumctl explain <type>' to see the schema
for a resource type and 'datumctl api-resources' to list available types.
```

**Proposed Example:**
```
# Create a project from a manifest file
datumctl create -f ./project.yaml

# Create a resource from stdin
cat dnszone.yaml | datumctl create -f -

# Validate the resource without creating it
datumctl create -f ./project.yaml --dry-run=server
```

---

### `datumctl apply`

**Current Short (from kubectl):** `Apply a configuration to a resource by file name or stdin`
**Problems:** Examples reference `kubectl` and pods/nginx. The Alpha disclaimer about `--prune` is jarring to Datum users.

**Proposed Short:**
```
Apply a Datum Cloud resource manifest (create or update)
```

**Proposed Long:**
```
Create or update Datum Cloud resources by applying a manifest file or
reading from stdin. If the resource does not exist it is created; if it
already exists it is updated to match the desired state in the manifest.

This is the recommended way to manage Datum Cloud resources declaratively.
Store your manifests in source control and apply them to keep your
infrastructure in sync.

JSON and YAML formats are accepted. Multiple resources can be placed in a
single file using YAML document separators (---).

Use --dry-run=server to validate your manifests against the API server
without persisting any changes.
```

**Proposed Example:**
```
# Apply a project manifest
datumctl apply -f ./project.yaml

# Apply all manifests in a directory
datumctl apply -f ./infra/

# Apply from stdin
cat dnszone.yaml | datumctl apply -f -

# Preview changes without applying them
datumctl apply -f ./project.yaml --dry-run=server

# Diff then apply
datumctl diff -f ./project.yaml && datumctl apply -f ./project.yaml
```

---

### `datumctl edit`

**Current Short (from kubectl):** `Edit a resource on the server`
**Problems:** Short description says "on the server" which is confusing. Examples reference `kubectl` and Kubernetes resources (services, jobs).

**Proposed Short:**
```
Open a Datum Cloud resource in your editor and apply the changes
```

**Proposed Long:**
```
Fetch a Datum Cloud resource, open it in your local text editor, and
apply any changes you save back to the platform.

The editor is determined by the KUBE_EDITOR or EDITOR environment variable,
falling back to 'vi' on Linux/macOS or 'notepad' on Windows.

The resource is displayed in YAML by default. Use -o json to edit in JSON
format instead.

Changes are applied when you save and close the file. If a conflict is
detected (the resource was modified server-side while your editor was open),
datumctl saves your changes to a temporary file so you can reconcile them.
```

**Proposed Example:**
```
# Edit a project
datumctl edit project my-project-id

# Edit a DNS zone, opening in a specific editor
KUBE_EDITOR="code --wait" datumctl edit dnszone my-zone --namespace default

# Edit a resource in JSON format
datumctl edit project my-project-id -o json
```

---

### `datumctl describe`

**Current Short (from kubectl):** `Show details of a specific resource or group of resources`
**Problems:** Examples reference Kubernetes nodes, pods, replication controllers.

**Proposed Short:**
```
Show detailed information about a Datum Cloud resource
```

**Proposed Long:**
```
Print a detailed, human-readable description of one or more Datum Cloud
resources, including status conditions and related events where available.

You can select resources by name, by label selector (-l), or from a
manifest file (-f). If you provide a name prefix, datumctl will show
details for all resources whose names start with that prefix.

Use 'datumctl get' for a concise list, and 'datumctl describe' when you
need full status information, such as when troubleshooting a resource that
is not reaching a ready state.
```

**Proposed Example:**
```
# Describe a specific project
datumctl describe project my-project-id

# Describe all DNS zones in a namespace
datumctl describe dnszones --namespace default

# Describe resources matching a label selector
datumctl describe dnszones -l environment=production

# Describe a resource from a manifest file
datumctl describe -f ./project.yaml
```

---

### `datumctl diff`

**Current Short (from kubectl):** `Diff the live version against a would-be applied version`
**Problems:** Examples reference `kubectl` and non-Datum resources.

**Proposed Short:**
```
Preview changes a manifest would make to live resources
```

**Proposed Long:**
```
Show the difference between what is currently deployed on the Datum Cloud
platform and what would be applied from a given manifest file.

The output is always YAML. The diff uses the 'diff' tool in your PATH
with -u (unified) and -N (treat absent files as empty) flags by default.

Set the KUBECTL_EXTERNAL_DIFF environment variable to use a different diff
tool, for example: KUBECTL_EXTERNAL_DIFF="colordiff -N -u"

Exit codes:
  0   No differences were found.
  1   Differences were found.
  >1  An error occurred.
```

**Proposed Example:**
```
# Preview changes from a manifest file before applying
datumctl diff -f ./project.yaml

# Diff from stdin
cat dnszone.yaml | datumctl diff -f -

# Use a color diff tool
KUBECTL_EXTERNAL_DIFF="colordiff -N -u" datumctl diff -f ./project.yaml
```

---

### `datumctl explain`

**Current Short (from kubectl):** `Get documentation for a resource`
**Problems:** Examples reference `kubectl` and Kubernetes native resource types (pods, deployments).

**Proposed Short:**
```
Show the schema and field documentation for a Datum Cloud resource type
```

**Proposed Long:**
```
Display the schema definition and field-level documentation for any
Datum Cloud resource type supported by the current control plane.

Fields are referenced using dot notation: TYPE.fieldName.subFieldName.
Information is retrieved from the API server in OpenAPI format, so it
always reflects the exact version of the platform you are connected to.

Use 'datumctl api-resources' to see all available resource types.
```

**Proposed Example:**
```
# Show the schema for the Project resource type
datumctl explain projects

# Show all fields recursively
datumctl explain projects --recursive

# Show documentation for a specific field
datumctl explain projects.spec

# Show documentation using the OpenAPI v2 format
datumctl explain projects --output=plaintext-openapiv2
```

---

### `datumctl api-resources`

**Current Short (from kubectl):** `Print the supported API resources on the server`
**Problems:** Examples reference `kubectl` and the RBAC API group.

**Proposed Short:**
```
List all resource types available in the current Datum Cloud context
```

**Proposed Long:**
```
Print a table of all resource types (Custom Resource Definitions) available
from the Datum Cloud API server for the currently configured control plane.

This is the starting point for discovering what you can manage with
'datumctl get', 'datumctl apply', and 'datumctl explain'. The output
includes short names, API group, whether the resource is namespaced,
and the kind name.

The list is fetched fresh from the server on each invocation. To use a
cached copy, pass --cached.
```

**Proposed Example:**
```
# List all available resource types
datumctl api-resources

# List resource types with additional detail (verbs, short names)
datumctl api-resources -o wide

# List resource types sorted by name
datumctl api-resources --sort-by=name

# List only namespaced resource types
datumctl api-resources --namespaced=true

# List resource types for a specific API group
datumctl api-resources --api-group=networking.datumapis.com
```

---

### `datumctl api-versions`

**Current Short (from kubectl):** `Print the supported API versions on the server, in the form of "group/version"`
**Problems:** Example says `kubectl api-versions`.

**Proposed Short:**
```
List all API group/version pairs supported by the current Datum Cloud context
```

**Proposed Long:**
```
Print all API group and version combinations available from the Datum Cloud
API server for the currently configured control plane, one per line in the
form group/version (e.g., networking.datumapis.com/v1alpha).

Use 'datumctl api-resources' to also see the individual resource types
within each group.
```

**Proposed Example:**
```
# List all API versions
datumctl api-versions
```

---

### `datumctl version`

**Current Short (from kubectl):** `Print the client and server version information`
**Problems:** Example says `kubectl version`.

**Proposed Short:**
```
Print the datumctl client and API server version
```

**Proposed Long:**
```
Print version information for the datumctl client binary and, when a
control plane context is configured, for the Datum Cloud API server.

Use --client to print only the local client version without contacting
the server.
```

**Proposed Example:**
```
# Print client and server versions
datumctl version

# Print client version only (no server connection required)
datumctl version --client

# Print version in JSON format
datumctl version -o json
```

---

### `datumctl mcp`

**Current Short:** `Start the Datum MCP server`
**Current Long:** present (reasonable) | **Current Example:** none

**Problems:** Short and Long are reasonably good. Missing an `Example` block. Long description could note the safety default (dry-run).

**Proposed Short:**
```
Start a Model Context Protocol (MCP) server for Datum Cloud
```

**Proposed Long:**
```
Start a local MCP server that exposes Datum Cloud resource management
capabilities to AI agents and MCP-compatible clients (e.g., Claude).

Available tools:
  list_crds, get_crd       Discover and inspect resource type schemas
  validate_yaml            Validate manifests via server-side dry run
  create_resource,         Generic CRUD for any Datum Cloud resource type
  get_resource,
  update_resource,
  delete_resource,
  list_resources
  change_context           Switch between organization and project contexts
                           at runtime

Safety: all write operations default to dry-run mode. Pass dryRun: false
in the tool arguments to apply changes for real.

MCP clients connect over STDIO. Use --port to also expose a local HTTP
debug API on 127.0.0.1:<port> for testing tool calls with curl.

Exactly one of --organization or --project is required.
```

**Proposed Example:**
```
# Start MCP server targeting an organization
datumctl mcp --organization my-org-id

# Start MCP server targeting a specific project with a debug HTTP port
datumctl mcp --project my-project-id --port 8080

# Start MCP server with a default namespace for resource operations
datumctl mcp --organization my-org-id --namespace default

# Claude Desktop config (macOS) — add to mcpServers in claude_desktop_config.json:
# {
#   "datum_mcp": {
#     "command": "/usr/local/bin/datumctl",
#     "args": ["mcp", "--organization", "my-org-id", "--namespace", "default"]
#   }
# }
```

---

### `datumctl docs`

**Current Short:** `Documentation and API exploration commands`
**Current Long:** `Commands for exploring and browsing API documentation.`

**Problems:** Short and Long are nearly identical and add little value. No examples.

**Proposed Short:**
```
Explore API documentation and generate CLI reference docs
```

**Proposed Long:**
```
The docs group provides tools for discovering and exploring the Datum Cloud
API, as well as generating offline documentation for datumctl itself.

Subcommands:
  openapi               Launch a local Swagger UI to browse OpenAPI specs
                        for any API group available in the current context.
  generate-cli-docs     Generate markdown documentation files for all
                        datumctl commands (used to build the published
                        CLI reference at datum.net/docs).
```

---

### `datumctl docs openapi`

**Current Short:** `Browse OpenAPI specs for platform APIs` *(keep)*
**Problems:** Examples are embedded in the `Long` field instead of the `Example` field. Cobra renders them differently — `Example` gets its own labeled section in help output and is picked up by documentation generators.

**Proposed Long (remove embedded examples):**
```
Start a local Swagger UI server that lets you browse and interact with
the OpenAPI specifications for any Datum Cloud API group.

A dropdown in the Swagger UI allows switching between API groups without
restarting the server. Your browser opens automatically.

By default, browses APIs available at the platform root. Use --organization
or --project to explore the APIs of a specific control plane, which may
include additional resource types beyond the platform-wide ones.
```

**Proposed Example:**
```
# Browse platform-wide APIs (opens browser automatically)
datumctl docs openapi

# Browse APIs available in an organization's control plane
datumctl docs openapi --organization my-org-id

# Browse APIs available in a project's control plane
datumctl docs openapi --project my-project-id

# Use a fixed port
datumctl docs openapi --port 8080
```

---

### `datumctl activity`

**Current Short (from external package):** unknown — from `go.miloapis.com/activity`
**Current Long (from external package):** contains `kubectl activity audit`, `kubectl activity events`, `kubectl activity feed`, `kubectl activity history` examples throughout
**Current Example:** same as above — all examples say `kubectl`

**Problems:** This is an externally sourced command, but the same post-construction override pattern applies. All examples in the current Long say `kubectl activity ...`. This violates the guiding principle and must be fixed in `root.go` after the `activity.NewActivityCommand(...)` call. The subcommands `audit`, `events`, `feed`, and `history` also need their `Long` and `Example` fields overridden.

The `history` subcommand example currently shows `kubectl activity history deployments my-app -n default --diff` — referencing Kubernetes `deployments`, which do not exist in Datum Cloud.

**Proposed Short:**
```
View activity logs for Datum Cloud resources
```

**Proposed Long (activity parent):**
```
The activity group provides commands to query audit logs, events, and
change history for Datum Cloud resources.

Subcommands:
  audit     View audit log entries for actions taken on your resources
  events    View events emitted by Datum Cloud resources
  feed      View a combined activity feed across resource types
  history   View the change history for a specific resource
```

**Proposed Example (activity parent):**
```
# View recent audit events
datumctl activity audit --start-time "now-7d"

# View warning events
datumctl activity events --type Warning --start-time "now-7d"

# View a combined activity feed filtered by human-initiated changes
datumctl activity feed --change-source human
```

> Note: Subcommand-level Short/Long/Example for `audit`, `events`, `feed`, and `history` need to be written once the subcommand structure is confirmed from the external package source.

---

### `datumctl docs generate-cli-docs`

**Current Short:** `Generate markdown documentation`
**Current Long:** has two typos ("correspondine", "should exists")
**Current Example:** uses wrong flag name and missing `docs` prefix

**Problems:** Typos in Long. Example shows `datumctl generate-cli-docs -o` — should be `datumctl docs generate-cli-docs --output-dir`. The `Long` string must be wrapped with `templates.LongDesc()` (not `templates.Examples()`) so that Cobra applies correct prose formatting rather than example indentation.

**Proposed Short:**
```
Generate markdown reference documentation for all datumctl commands
```

**Proposed Long:**
```
Generate a markdown file for every datumctl command and write them to
the specified output directory.

Each command produces one markdown file named after its full command path
(e.g., datumctl_get.md). Files include front matter compatible with the
Datum Cloud documentation site.

The output directory must already exist before running this command.
This command is primarily used by the Datum Cloud documentation pipeline
to publish the CLI reference at datum.net/docs/datumctl.
```

**Proposed Example:**
```
# Generate documentation into a temporary directory
datumctl docs generate-cli-docs --output-dir /tmp/datumctl-docs

# Generate documentation into the docs output directory
datumctl docs generate-cli-docs --output-dir ./site/content/cli
```

---

## Summary Table

| Command | Current Short | Main Problem | Proposed Short |
|---|---|---|---|
| `datumctl` | A CLI for interacting with the Datum platform | Wrong product name, no Long | The official CLI for Datum Cloud |
| `auth` | Authenticate with Datum Cloud | No Long or Examples | Manage Datum Cloud authentication credentials |
| `auth login` | Authenticate with Datum Cloud via OAuth2 PKCE flow | Exposes impl detail; no Long/Examples | Log in to Datum Cloud |
| `auth logout` | Remove local authentication credentials for a specified user or all users | No Examples; confusing internal key format in Long | Remove stored credentials for a user or all users |
| `auth list` | List locally authenticated users | No Long or Examples | *(keep)* |
| `auth switch` | Set the active authenticated user session | No Examples | Switch the active Datum Cloud user session |
| `auth get-token` | Retrieve access token for active user (raw or K8s format) | Not scoped as kubectl-only; confusing for regular users | Print the active user's access token (advanced / kubectl integration) |
| `auth update-kubeconfig` | Update the kubeconfig file | Not scoped as kubectl-only; almost no description | Configure kubectl to access a Datum Cloud control plane (kubectl users only) |
| `auth whoami` | Experimental: Check self subject attributes | Not scoped as kubectl-only; kubectl text throughout | Show your identity on a Datum Cloud control plane (kubectl users only) |
| `auth can-i` | Check whether an action is allowed | Not scoped as kubectl-only; kubectl examples throughout | Check permissions on a Datum Cloud control plane (kubectl users only) |
| `get` | Display one or many resources | All kubectl examples reference pods/deployments | List or retrieve Datum Cloud resources |
| `delete` | Delete resources by file names, stdin... | Long text is about pod graceful deletion | Delete Datum Cloud resources |
| `create` | Create a resource from a file or from stdin | kubectl Examples; stripped subcommands not noted | Create a Datum Cloud resource from a file or stdin |
| `apply` | Apply a configuration to a resource | kubectl Examples; Alpha disclaimer | Apply a Datum Cloud resource manifest (create or update) |
| `edit` | Edit a resource on the server | "on the server" is confusing; kubectl Examples | Open a Datum Cloud resource in your editor and apply the changes |
| `describe` | Show details of a specific resource... | kubectl Examples with nodes/pods | Show detailed information about a Datum Cloud resource |
| `diff` | Diff the live version against a would-be applied version | kubectl Examples | Preview changes a manifest would make to live resources |
| `explain` | Get documentation for a resource | kubectl Examples with pods/deployments | Show the schema and field documentation for a Datum Cloud resource type |
| `api-resources` | Print the supported API resources on the server | kubectl Examples; RBAC example | List all resource types available in the current Datum Cloud context |
| `api-versions` | Print the supported API versions... | Example says `kubectl api-versions` | List all API group/version pairs supported by the current Datum Cloud context |
| `version` | Print the client and server version information | Example says `kubectl version` | Print the datumctl client and API server version |
| `mcp` | Start the Datum MCP server | Good; missing Examples and safety note | Start a Model Context Protocol (MCP) server for Datum Cloud |
| `docs` | Documentation and API exploration commands | Long identical to Short | Explore API documentation and generate CLI reference docs |
| `docs openapi` | Browse OpenAPI specs for platform APIs | Examples buried in Long not Example field | *(keep Short)* — move examples to Example field |
| `docs generate-cli-docs` | Generate markdown documentation | Typos in Long; wrong flag in Example | Generate markdown reference documentation for all datumctl commands |

---

## Files to Edit

### Root command and kubectl-wrapped overrides — all in one file

**`internal/cmd/root.go`** — changes needed:
- Update `rootCmd.Short` and add `rootCmd.Long` on the root command struct literal
- Add `Short`, `Long`, and `Example` field assignments after each constructor call for:
  - `get.NewCmdGet(...)`
  - `delcmd.NewCmdDelete(...)`
  - `create.NewCmdCreate(...)` *(note: does NOT go through WrapResourceCommand)*
  - `apply.NewCmdApply(...)` *(note: does NOT go through WrapResourceCommand)*
  - `edit.NewCmdEdit(...)`
  - `describe.NewCmdDescribe(...)`
  - `diff.NewCmdDiff(...)` *(note: does NOT go through WrapResourceCommand)*
  - `explain.NewCmdExplain(...)`
  - `apiresources.NewCmdAPIVersions(...)`
  - `apiresources.NewCmdAPIResources(...)`
  - `version.NewCmdVersion(...)`
  - `kubeauth.NewCmdWhoAmI(...)` and `kubeauth.NewCmdCanI(...)` (added to authCommand)
  - `activity.NewActivityCommand(...)` — external package, same override pattern applies; also override subcommands (`audit`, `events`, `feed`, `history`)

### Custom auth command files

- `internal/cmd/auth/auth.go` — add Long to parent command
- `internal/cmd/auth/login.go` — update Short, add Long and Example
- `internal/cmd/auth/logout.go` — update Short and Long, add Example
- `internal/cmd/auth/list.go` — add Long and Example
- `internal/cmd/auth/switch.go` — update Short and Long, add Example
- `internal/cmd/auth/get_token.go` — update Short and Long, add Example
- `internal/cmd/auth/update-kubeconfig.go` — update Short, add Long and Example

### MCP and docs

- `internal/cmd/mcp/mcp.go` — update Short, expand Long with safety note, add Example
- `internal/cmd/docs/docs.go` — update Short and Long
- `internal/cmd/docs/openapi.go` — move embedded examples from Long to Example field
- `internal/cmd/docs/generate-cli-documentation.go` — fix typos in Long, fix Example flag name and add `docs` prefix
