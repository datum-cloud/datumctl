---
title: "Authentication"
sidebar:
  order: 2
---

`datumctl` uses OAuth 2.0 and OpenID Connect (OIDC) with the PKCE extension for
secure authentication against Datum Cloud. This avoids the need to handle static
API keys directly.

Authentication involves the following commands:

*   `datumctl login`
*   `datumctl logout`
*   `datumctl whoami`
*   `datumctl auth list`
*   `datumctl auth get-token`
*   `datumctl auth update-kubeconfig`
*   `datumctl auth switch`

Credentials and tokens are stored securely in your operating system's default
keyring.

## Logging in

To authenticate with Datum Cloud, use the `login` command:

```
datumctl login [--hostname <auth-hostname>] [--no-browser] [-v]
```

*   `--hostname <auth-hostname>`: (Optional) Specify the hostname of the Datum
    Cloud authentication server. Defaults to `auth.datum.net`.
*   `--no-browser`: (Optional) Do not attempt to open a browser; print the login
    URL and use the device authorization flow (enter a user code) so you can
    complete login without a local callback.
*   `-v, --verbose`: (Optional) Print the full ID token claims after successful
    login.

Running this command will:

1.  Attempt to open your default web browser to the Datum Cloud authentication
    page (or use device authorization if `--no-browser` is used).
2.  If the browser cannot be opened automatically, it will print a URL for you
    to visit manually.
3.  Authenticate via the web page (this might involve entering your
    username/password or using single sign-on).
4.  After successful authentication, you will be redirected back to `datumctl`
    (via a local webserver started temporarily), which completes the process.

Your credentials (including refresh tokens) are stored securely in the system
keyring, associated with your user identifier (typically your email address).

## Account onboarding

Before you can use datumctl against an organization, that organization must
complete onboarding in the [Datum Cloud portal](https://cloud.datum.net). This
includes finishing billing setup (contact information, billing account, and
payment method).

Onboarding is checked **per organization**, based on your active context or
`--organization` / `--project` flags. User-scoped discovery commands such as
`datumctl get organizations` always run against your user control plane, even
when your active context points at an incomplete organization. That command
also shows each organization's setup status and portal links for any that
still need setup.

When the targeted organization is incomplete:

*   `datumctl login` stores your credentials but blocks selecting that
    organization's context and prints a link to the portal.
*   Org- or project-scoped commands are blocked with a message pointing you to
    the portal.
*   `datumctl whoami` shows onboarding status for your current context's
    organization.

Machine accounts are subject to the same requirement for the organization they
target.

For staging environments, use the staging portal at
`https://cloud.staging.env.datum.net`.

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
automatically use your active `datumctl` login session for authentication.

## Listing logged-in users

To see which users you have authenticated locally, use the `list` command:

```
datumctl auth list
# Alias: datumctl auth ls
```

This will output a table showing the Name, Email, and Status (Active or blank)
for each set of stored credentials. The user marked `Active` is the one whose
credentials will be used by default for other `datumctl` commands and
`kubectl` (if configured via `update-kubeconfig`).

## Switching active user

If you have logged in with multiple user accounts (visible via
`datumctl auth list`), you can switch which account is active using the
`switch` command:

```
datumctl auth switch <user-email>
```

Replace `<user-email>` with the email address of the user you want to make
active. This user must already be logged in.

After switching, subsequent commands that require authentication (like
`datumctl organizations list` or `kubectl` operations configured via
`update-kubeconfig`) will use the credentials of the newly activated user.

## Logging out

To remove stored credentials, use the `logout` command.

**Log out a specific user:**

```
datumctl logout <user-email>
```

Replace `<user-email>` with the email address shown in the
`datumctl auth list` command.

**Log out all users:**

```
datumctl logout --all
```

This removes all Datum Cloud credentials stored by `datumctl` in your keyring.

## Getting tokens (advanced)

The `get-token` command retrieves the current access token for the *active*
authenticated user. This is primarily used internally by other tools (like
`kubectl`) but can be used directly if needed.

```
datumctl auth get-token [-o <format>]
```

*   `-o, --output <format>`: (Optional) Specify the output format. Defaults to
    `token`.
    *   `token`: Prints the raw access token to standard output.
    *   `client.authentication.k8s.io/v1`: Prints a Kubernetes `ExecCredential`
        JSON object containing the ID token, suitable for `kubectl`
        authentication.

If the stored access token is expired, `get-token` will attempt to use the
refresh token to obtain a new one automatically.
