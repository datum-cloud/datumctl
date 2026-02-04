# datumctl: The Datum Cloud CLI

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

`datumctl` is the official command-line interface for interacting with [Datum Cloud](https://www.datum.net), the connectivity infrastructure platform designed to unlock networking superpowers for developers and forward-thinking companies.

Use `datumctl` to manage your Datum Cloud resources, authenticate securely, and integrate with tools like `kubectl`.

## Features

*   **Secure Authentication:** Uses modern OAuth 2.0 and OIDC PKCE flow for secure, browser-based login. No static API keys required.
*   **Multi-User Support:** Manage credentials for multiple Datum Cloud user accounts.
*   **Resource Management:** Interact with Datum Cloud resources (e.g., list organizations).
*   **Kubernetes Integration:** Seamlessly configure `kubectl` to use your Datum Cloud credentials for accessing Kubernetes clusters.
*   **MCP Server (optional):** Start an MCP server (`datumctl mcp`) for Datum Cloud so AI agents (e.g., Claude) can discover resources, inspect schemas, validate manifests, and perform CRUD operations via server-side dry-run.
*   **Cross-Platform:** Pre-built binaries available for Linux, macOS, and Windows.

## Getting Started

### Installation

See the [Installation Guide](https://www.datum.net/docs/quickstart/datumctl/) for detailed instructions, including Homebrew for macOS, nix for Linux and macOS, and pre-built binaries for all platforms.

### Basic Usage

1.  **Log in to Datum Cloud:**
    ```bash
    datumctl auth login
    ```
    (This will open your web browser to complete authentication.)

2.  **List your organizations:**
    ```bash
    datumctl get organizations
    ```

3.  **Configure `kubectl` access (optional):**
    Use the organization ID (or a specific project ID) from the previous step
    to configure `kubectl`.
    ```bash
    # Example using an organization ID
    datumctl auth update-kubeconfig --organization <org-id>

    # Example using a project ID
    datumctl auth update-kubeconfig --project <project-id>
    ```
    Now you can use `kubectl` to interact with your Datum Cloud control plane.

### MCP Setup

MCP can target either an **organization** or **project** control plane. For maximum flexibility, we recommend starting with an organization context.

**A) If you already have a project:**
```bash
# Ensure your kube context points at an organization control plane
datumctl auth update-kubeconfig --organization <org-id>

# List projects; copy the NAME column (that is the Project ID)
kubectl get projects
# Or JSON-friendly:
kubectl get projects -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}'
```

**B) If you need to create a project:**
```bash
# Make sure your kube context targets an organization control plane
datumctl auth update-kubeconfig --organization <org-id>

cat > intro-project.yaml <<'YAML'
apiVersion: resourcemanager.miloapis.com/v1alpha1
kind: Project
metadata:
  generateName: intro-project-
spec: {}
YAML

kubectl create -f intro-project.yaml

# Wait until Ready
PRJ_ID="$(kubectl get projects -o jsonpath='{.items[-1:].metadata.name}')"
kubectl wait --for=condition=Ready --timeout=15m project/$PRJ_ID
echo "Project ready: $PRJ_ID"
```

#### MCP subcommand (optional)

Start the Model Context Protocol (MCP) server targeting a specific Datum Cloud context:
```bash
# Exactly one of --organization or --project is required.
datumctl mcp --organization <org-id> --namespace <ns> [--port 8080]
# or
datumctl mcp --project <project-id> --namespace <ns> [--port 8080]
```

##### Available Tools

- **Discovery:** `list_crds`, `get_crd` - Discover and inspect Custom Resource Definitions
- **Validation:** `validate_yaml` - Validate manifests via server-side dry-run
- **Context:** `change_context` - Switch between organization and project contexts
- **CRUD Operations:** `create_resource`, `get_resource`, `update_resource`, `delete_resource`, `list_resources`
- **Safety:** All write operations default to dry-run mode; use `dryRun: false` to apply changes

##### Startup & safety

- **Preflight:** On startup, `datumctl mcp` verifies connectivity and auth by calling Kubernetes discovery (e.g., `GET /version`). If this check fails, the server exits.
- **Dry-run by default:** All write operations use server-side dry-run (`dryRun=true`) by default for safety.

> [!NOTE]
> The MCP server builds its own Kubernetes connection for the selected Datum context; it does **not** depend on your local kubeconfig or `--kube-context`. Provide either `--organization` or `--project`.

##### Scope: organization vs. project

> [!IMPORTANT]
> **Organization scope** provides access to all projects within the organization and allows switching between them using `change_context`.  
> **Project scope** provides direct access to project-specific resources but limits visibility to that single project.

**Recommended (organization scope)**
```bash
datumctl mcp --organization <org-id> --namespace <ns> [--port 8080]
```

##### Claude config (macOS)
```json
{
  "mcpServers": {
    "datum_mcp": {
      "command": "/absolute/path/to/datumctl",
      "args": ["mcp", "--organization", "your-org-id", "--namespace", "default"]
    }
  }
}
```

**Project scope (alternative)**
```bash
datumctl mcp --project <project-id> --namespace <ns> [--port 8080]
```

**HTTP debug (if `--port` is set):**
```bash
# List CRDs
curl -s localhost:8080/datum/list_crds | jq

# List resources
curl -s localhost:8080/datum/list_resources -H 'Content-Type: application/json' -d '{"kind":"Project"}' | jq

# Validate a YAML file (wrap safely into JSON)
printf '{"yaml":%s}\n' "$(jq -Rs . </path/to/file.yaml)" | curl -s -X POST localhost:8080/datum/validate_yaml -H 'Content-Type: application/json' -d @- | jq
```

For more detailed tool setup instructions, refer to the official
[Set Up Tools](https://docs.datum.net/docs/tasks/tools/) guide on docs.datum.net.

## Documentation

For comprehensive user and developer guides, including detailed command references and authentication flow explanations, please see the [**Documentation**](./docs/README.md).

## Contributing

Contributions are welcome! Please refer to the contribution guidelines (link to be added) for more information.

## License

`datumctl` is licensed under the Apache License, Version 2.0. See the [LICENSE](./LICENSE) file for details.
