## GEMINI.md - A Developer's Guide to datumctl

This document provides a developer-centric overview of `datumctl`, the official command-line interface for Datum Cloud. It is intended to help new developers get up to speed with the project's architecture, conventions, and development workflow.

### Project Overview

`datumctl` is a Go application built using the [Cobra](https://cobra.dev/) library. It provides a command-line interface for interacting with the Datum Cloud platform, including features for authentication, resource management, and Kubernetes integration.

A key feature of `datumctl` is the **Model Context Protocol (MCP) server**. This allows AI agents, such as Gemini, to interact with the Datum Cloud API in a structured and secure way.

### Getting Started

#### Prerequisites

*   Go 1.18 or later
*   [Nix](https://nixos.org/download.html) (optional, for environment management)

#### Development Environment

The recommended way to set up a development environment is to use the provided `flake.nix` file. This will ensure that you have all the necessary dependencies and tools installed.

To activate the development environment, run the following command from the root of the project:

```sh
nix develop
```

#### Building and Running

To build the `datumctl` binary, run the following command:

```sh
go build
```

To run the `datumctl` command, you can use the following command:

```sh
./datumctl --help
```

### Architecture

The `datumctl` codebase is organized into several key components:

*   **`internal/cmd`**: This directory contains the definitions for all of the CLI commands, which are built using the Cobra library. Each command is in its own file, and the `root.go` file ties them all together.
*   **`internal/authutil`**: This package provides shared utilities and types for handling authentication. It includes functions for interacting with the keyring, creating OIDC token sources, and deriving API hostnames.
*   **`internal/keyring`**: This package provides a simple wrapper around the `go-keyring` library, which is used for securely storing user credentials.
*   **`internal/output`**: This package contains helpers for formatting CLI output in various formats, such as tables, JSON, and YAML.
*   **`internal/mcp`**: This package contains the implementation of the Model Context Protocol (MCP) server.

### Authentication Flow

`datumctl` uses OAuth 2.0 with PKCE for user authentication. The authentication flow is as follows:

1.  The user runs the `datumctl auth login` command.
2.  `datumctl` opens a web browser to the Datum Cloud authentication server.
3.  The user authenticates with the Datum Cloud.
4.  The authentication server redirects the user back to `datumctl` with an authorization code.
5.  `datumctl` exchanges the authorization code for an access token, refresh token, and ID token.
6.  The tokens are stored securely in the user's OS keyring.

When a user runs a command that requires authentication, `datumctl` retrieves the tokens from the keyring and uses them to make authenticated API requests. If the access token is expired, `datumctl` will automatically use the refresh token to get a new one.

### Adding a New Command

To add a new command to `datumctl`, you will need to do the following:

1.  Create a new file in the appropriate subdirectory of `internal/cmd`.
2.  In the new file, define a new Cobra command.
3.  Add the new command to the root command in `internal/cmd/root.go`.

For more detailed information, please refer to the [Cobra documentation](httpss://cobra.dev/#getting-started).

### Git Conventions

This project does not have explicit commit message conventions. However, looking at the git history, we can see the following patterns:

*   **Merge commits**: These are used to merge pull requests. The commit message usually includes the pull request number and a brief description of the changes.
*   **Feature commits**: These are used to add new features. The commit message usually starts with `feat:` and includes a brief description of the feature.
*   **Fix commits**: These are used to fix bugs. The commit message usually starts with `fix:` and includes a brief description of the fix.
*   **Chore commits**: These are used for maintenance tasks, such as updating dependencies. The commit message usually starts with `chore:` and includes a brief description of the task.

When contributing to the project, please try to follow these conventions.
