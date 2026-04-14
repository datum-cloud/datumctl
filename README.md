# datumctl: The Datum Cloud CLI

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

`datumctl` is the official command-line interface for interacting with [Datum Cloud](https://www.datum.net), the connectivity infrastructure platform designed to unlock networking superpowers for developers and forward-thinking companies.

Use `datumctl` to manage your Datum Cloud resources, authenticate securely, and integrate with tools like `kubectl`.

## Features

*   **Secure Authentication:** Uses modern OAuth 2.0 and OIDC PKCE flow for secure, browser-based login. No static API keys required.
*   **Multi-User Support:** Manage credentials for multiple Datum Cloud user accounts.
*   **Resource Management:** Interact with Datum Cloud resources (e.g., list organizations).
*   **Kubernetes Integration:** Seamlessly configure `kubectl` to use your Datum Cloud credentials for accessing Kubernetes clusters.
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

## Documentation

For comprehensive user and developer guides, including detailed command references and authentication flow explanations, please see the [**Documentation**](./docs/README.md).

## Contributing

Contributions are welcome! Please refer to the contribution guidelines (link to be added) for more information.

## License

`datumctl` is licensed under the Apache License, Version 2.0. See the [LICENSE](./LICENSE) file for details.
