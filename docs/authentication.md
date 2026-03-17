---
title: "Authentication"
sidebar:
  order: 2
---

`datumctl` uses OAuth 2.0 and OpenID Connect (OIDC) with the PKCE extension for
secure authentication against Datum Cloud. This avoids the need to handle static
API keys directly.

Authentication involves the following commands:

*   `datumctl auth login`
*   `datumctl auth logout`
*   `datumctl auth get-token`
*   `datumctl auth update-kubeconfig`

Credentials and tokens are stored securely in your operating system's default
keyring.

## Logging in

To authenticate with Datum Cloud, use the `login` command:

```
datumctl auth login [--hostname <auth-hostname>] [-v]
```

*   `--hostname <auth-hostname>`: (Optional) Specify the hostname of the Datum
    Cloud authentication server. Defaults to `auth.datum.net`.
*   `-v, --verbose`: (Optional) Print the full ID token claims after successful
    login.

Running this command will:

1.  Attempt to open your default web browser to the Datum Cloud authentication
    page.
2.  If the browser cannot be opened automatically, it will print a URL for you
    to visit manually.
3.  Authenticate via the web page (this might involve entering your
    username/password or using single sign-on).
4.  After successful authentication, you will be redirected back to `datumctl`
    (via a local webserver started temporarily), which completes the process.

Your credentials (including refresh tokens) are stored securely in the system
keyring, associated with your user identifier (typically your email address).

On every successful login, `datumctl` also ensures a matching cluster/context
entry exists in `~/.datumctl/config` for the API host you authenticated against.
If a current context already exists, it remains unchanged.

`datumctl` stores a list of users in `~/.datumctl/config` and links each context
to a user key (in the `subject@auth-hostname` format). The actual tokens are
stored in your OS keyring under the `datumctl-auth` service.

## Updating kubeconfig

Once logged in, you typically need to configure `kubectl` to authenticate to
Datum Cloud Kubernetes clusters using your `datumctl` login session. Use the
`update-kubeconfig` command:

```
datumctl auth update-kubeconfig [--kubeconfig <path>] [--project <name>] [--organization <name>]
```

*   `--kubeconfig <path>`: (Optional) Path to the kubeconfig file to update.
    Defaults to the standard location (`$HOME/.kube/config` or the path
    specified by the `KUBECONFIG` environment variable).
*   `--project <name>`: Specify the Datum Cloud project name to configure access
    for. You can find project IDs after creating projects in Datum Cloud.
*   `--organization <name>`: Specify the Datum Cloud organization name to
    configure access for. You can find your organization ID using
    `datumctl organizations list`.

> [!IMPORTANT]
> You must specify either `--project` or `--organization`.

This command adds or updates the necessary cluster, user, and context entries
in your kubeconfig file. The user entry will be configured to use
`datumctl auth get-token --output=client.authentication.k8s.io/v1` as an `exec`
credential plugin. This means `kubectl` commands targeting this cluster will
automatically use the credentials associated with your current `datumctl`
context for authentication.

## Logging out

To remove stored credentials, use the `logout` command.

**Log out a specific user:**

```
datumctl auth logout <user-key>
```

Replace `<user-key>` with the key shown in the `users` list in
`~/.datumctl/config`. Use `--all` to remove all credentials.

**Log out all users:**

```
datumctl auth logout --all
```

This removes all Datum Cloud credentials stored by `datumctl` in your keyring.

## Getting tokens (advanced)

The `get-token` command retrieves the current access token for the credentials
associated with the current context. This is primarily used internally by other tools (like
`kubectl`) but can be used directly if needed.

```
datumctl auth get-token [-o <format>] [--cluster <datumctl-cluster>]
```

*   `-o, --output <format>`: (Optional) Specify the output format. Defaults to
    `token`.
    *   `token`: Prints the raw access token to standard output.
    *   `client.authentication.k8s.io/v1`: Prints a Kubernetes `ExecCredential`
        JSON object containing the ID token, suitable for `kubectl`
        authentication.
*   `--cluster <datumctl-cluster>`: (Optional) Use credentials bound to the
    specified datumctl cluster instead of the current context.

If the stored access token is expired, `get-token` will attempt to use the
refresh token to obtain a new one automatically.
