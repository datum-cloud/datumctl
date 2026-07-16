---
title: "API Proxy"
sidebar:
  order: 4
---

`datumctl api proxy` starts a local HTTP proxy that forwards every request to
the Datum Cloud API endpoint of your datumctl session, adding your credentials
automatically and refreshing them as needed. Point any local tool — a dev
server, a test harness, `curl` — at the printed URL and it can talk to the
platform with no tokens to copy and no expiry to manage:

```
$ datumctl api proxy --port 8001
$ curl http://127.0.0.1:8001/apis/resourcemanager.miloapis.com/v1alpha1/organizations
```

## Starting the proxy

```
# Start a proxy on a fixed port for a dev server
datumctl api proxy --port 8001

# Start on a random free port; the URL is printed on the first stdout line
datumctl api proxy

# Pin a non-active session (see 'datumctl auth list' for names)
datumctl api proxy --session sam@datum.net@api.staging.env.datum.net

# Suppress per-request log lines
datumctl api proxy --quiet
```

By default the proxy picks a random free port, so starting a second proxy
never fails. The bound URL is always printed:

- A human-readable banner on **stderr** names the session, upstream, scope,
  and listen address, so you can verify at a glance whose credentials the
  proxy serves.
- The bare proxy URL (for example `http://127.0.0.1:52347`) is printed as the
  **first and only line on stdout**, after the listener is serving. Scripts
  and test harnesses can read that one line as their readiness signal:

```go
cmd := exec.Command("datumctl", "api", "proxy", "--quiet", "--project", testProject)
stdout, _ := cmd.StdoutPipe()
_ = cmd.Start()
apiURL, _ := bufio.NewReader(stdout).ReadString('\n') // first line = ready + address
```

Press `Ctrl+C` to stop the proxy. In-flight requests get a short grace period
before their connections are closed.

## Paths and scoping

By default the proxy is a pure passthrough of the platform API surface: the
same paths that work against the real API endpoint work against the local
port, including the scoped organization/project control-plane prefixes:

```
# Watch DNS zones on a project control plane through the proxy
curl "http://127.0.0.1:8001/apis/resourcemanager.miloapis.com/v1alpha1/projects/my-project/control-plane/apis/networking.datumapis.com/v1alpha/dnszones?watch=true"
```

With an explicit `--project` or `--organization` flag, the proxy instead
serves that single control plane at its root, so URLs lose the long
control-plane prefix:

```
$ datumctl api proxy --port 8001 --project my-project
$ curl "http://127.0.0.1:8001/apis/networking.datumapis.com/v1alpha/dnszones?watch=true"
```

The session and the scope are **pinned when the proxy starts** and shown in
the banner. Switching your active account (`datumctl auth switch`) or context
(`datumctl ctx use`) does not affect a running proxy, and the proxy never
inherits a scope from your current context — scoping is always an explicit
flag. Restart the proxy to pick up a new session or scope.

## Streaming

Streaming responses — watch requests, server-sent events, chunked transfer —
pass through unbuffered: each event is flushed to your client the moment the
upstream sends it, and response duration is never limited by the proxy. This
makes the proxy suitable for long-lived watch clients.

## Credentials and token refresh

Every proxied request carries a fresh access token for the pinned session;
the proxy refreshes the token before expiry and persists refreshed tokens
exactly as other datumctl commands do. Service-account sessions are supported
the same way.

If the session cannot be refreshed (expired, revoked, or logged out with
`datumctl logout`), the proxy stays up and answers requests with a
synthesized `502 Bad Gateway` carrying a JSON `Status` body, the
`X-Datum-Proxy-Error: true` marker header, and a message telling you to run
`datumctl login`. The proxy recovers automatically — without a restart — once
you log back in to the same session. A `401` or `403` from the platform
itself passes through unchanged, without the marker header, so your client
can always tell a proxy-local authentication problem from a platform answer.

## Security model

- **Loopback only.** The proxy listens on `127.0.0.1` and there is no flag to
  bind other addresses. Anything that should be reachable remotely deserves a
  tunnel whose security model you own.
- **Local clients are trusted.** Like other local developer proxies, the
  proxy does not authenticate local clients: while it runs, any process on
  your machine can make API calls as the pinned session. The banner names the
  identity it serves, requests are logged by default, and the credential
  itself is never exposed — a local client can act through the proxy but
  cannot take your token with it.
- **Host-header validation.** Requests whose `Host` is not `localhost`,
  `127.0.0.1`, or `[::1]` are rejected with `403`, which defeats DNS-rebinding
  attacks from web pages.
- **No CORS headers.** Browsers refuse scripted cross-origin reads of proxy
  responses; the proxy is meant for server-side and command-line clients.
- **Authorization discipline.** Any `Authorization` header your local client
  sends is stripped and replaced with the session's real token, and tokens
  never appear in the request log.

## Request logging

One line per request is written to stderr (silence with `--quiet`):

```
10:42:03 GET  /apis/resourcemanager.miloapis.com/v1alpha1/organizations 200 143ms 8.1kB
10:42:05 GET  /apis/…/projects/my-project/control-plane/…/portalplugins?watch=true 200 …streaming
```

Streaming responses log once when headers arrive (marked `…streaming`) and
again when the stream ends, with total duration and bytes — so an abruptly
closed watch is visible.
