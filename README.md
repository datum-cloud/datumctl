# datumctl: The Datum Cloud CLI

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

`datumctl` is the official command-line interface for interacting with [Datum Cloud](https://www.datum.net), the connectivity infrastructure platform designed to unlock networking superpowers for developers and forward-thinking companies.

Use `datumctl` to manage your Datum Cloud resources, authenticate securely, and integrate with tools like `kubectl`.

## Features

*   **Secure Authentication:** Modern OAuth 2.0 / OIDC PKCE with device-code fallback for headless environments. No static API keys.
*   **Context Discovery:** After login, `datumctl` fetches the organizations and projects you can access and lets you pick a default context — no more passing `--organization` or `--project` on every command.
*   **Multi-User Support:** Manage credentials for multiple Datum Cloud accounts and switch between them with `datumctl auth switch`.
*   **Resource Management:** Interact with Datum Cloud resources with a kubectl-style interface (`get`, `apply`, `describe`, `delete`, ...).
*   **Kubernetes Integration:** Configure `kubectl` to use your Datum Cloud credentials for accessing control planes.
*   **AI Agents / MCP:** `datumctl` can be used directly by agents for CLI-driven workflows. The standalone [`datum-mcp`](https://github.com/datum-cloud/datum-mcp) project provides a Model Context Protocol server for tool-based integrations.
*   **Cross-Platform:** Pre-built binaries available for Linux, macOS, and Windows.

## Getting Started

### Installation

See the [Installation Guide](https://www.datum.net/docs/quickstart/datumctl/) for detailed instructions, including Homebrew for macOS, nix for Linux and macOS, and pre-built binaries for all platforms.

### Basic Usage

1.  **Log in and pick a context:**
    ```bash
    datumctl login
    ```
    Opens your browser for authentication, then fetches your organizations and projects and prompts you to pick a default context. If you only have a single project, the picker is skipped.

2.  **Work with resources:**
    ```bash
    datumctl get dnszones        # uses the active context automatically
    datumctl get organizations   # list your org memberships
    datumctl api-resources       # discover available resource types
    ```

3.  **Switch contexts or accounts:**
    ```bash
    datumctl ctx                 # list contexts (tree view by org)
    datumctl ctx use my-org/my-project
    datumctl auth list           # list accounts
    datumctl auth switch alice@example.com
    ```

4.  **Configure `kubectl` access (optional):**
    ```bash
    # Point kubectl at your organization's control plane
    datumctl auth update-kubeconfig --organization <org-id>

    # Or at a specific project's control plane
    datumctl auth update-kubeconfig --project <project-id>
    ```
    kubectl then uses `datumctl auth get-token` automatically to refresh credentials.

### CI and scripting

For non-interactive use, environment variables override the active context per-invocation:

```bash
DATUM_PROJECT=my-project datumctl get dnszones
DATUM_ORGANIZATION=my-org datumctl get projects
```

`--project` and `--organization` flags work too. For machine-to-machine auth, see `datumctl auth login --credentials` for the machine-account flow.

## Agent Skills

This repository includes a bundled [`datumctl` skill](./skills/datumctl/SKILL.md) for agents that need lightweight guidance for direct CLI usage.

Use the Datum repositories for different integration layers:

*   [`datumctl`](https://github.com/datum-cloud/datumctl): the CLI itself, plus a small routing skill for direct `datumctl` usage.
*   [`datum-cloud/skills`](https://github.com/datum-cloud/skills): the canonical repository for Datum skills that should be installed into agent runtimes such as Claude, Codex, Cursor, and similar tools.
*   [`datum-mcp`](https://github.com/datum-cloud/datum-mcp): the MCP server for tool-based integrations.

If you are installing skills into an agent environment, prefer `datum-cloud/skills` as the public install target. Keep `datumctl` focused on CLI behavior and lightweight agent guidance rather than agent-specific installation logic.

## Documentation

For comprehensive user and developer guides, including detailed command references and authentication flow explanations, please see the [**Documentation**](./docs/README.md).

## Contributing

Contributions are welcome! Please refer to the contribution guidelines (link to be added) for more information.

## License

`datumctl` is licensed under the Apache License, Version 2.0. See the [LICENSE](./LICENSE) file for details.
