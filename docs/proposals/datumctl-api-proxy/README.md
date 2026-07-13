---
status: provisional
stage: alpha
latest-milestone: "v0.x"
---

# A Local Authenticated API Proxy: `datumctl api proxy`

- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [CLI surface](#cli-surface)
  - [Resolving the session and upstream](#resolving-the-session-and-upstream)
  - [The proxy engine and streaming design](#the-proxy-engine-and-streaming-design)
  - [Path semantics](#path-semantics)
  - [Token lifecycle](#token-lifecycle)
  - [Security model](#security-model)
  - [Request logging](#request-logging)
  - [Failure and lifecycle UX](#failure-and-lifecycle-ux)
  - [Prior art](#prior-art)
  - [Testing strategy](#testing-strategy)
  - [V1 milestone cut](#v1-milestone-cut)
- [Open Questions](#open-questions)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed](#infrastructure-needed)

## Summary

`datumctl api proxy` starts a local HTTP proxy bound to the loopback
interface that forwards every request to the Datum Cloud API endpoint of the
user's datumctl session, injecting a valid `Authorization: Bearer` token on
each request and transparently refreshing it when it expires. Any local tool
— a dev server, a test harness, `curl` — can then talk to the platform with
zero token plumbing:

```console
$ datumctl api proxy --port 8001
$ curl http://127.0.0.1:8001/apis/resourcemanager.miloapis.com/v1alpha1/organizations
```

By default the proxy is a pure passthrough of the platform API surface: the
same paths that work against `https://api.datum.net` work against the local
port, including the scoped org/project control-plane prefixes and —
critically — long-lived streaming responses (Kubernetes-style watches,
server-sent events, chunked transfer), which pass through unbuffered. With
an explicit `--project` or `--organization` flag, the proxy instead
re-bases itself at that control plane, so it looks like a single dedicated
API server on localhost and URLs lose the long control-plane prefix:

```console
$ datumctl api proxy --port 8001 --project my-project
$ curl http://127.0.0.1:8001/apis/networking.datumapis.com/v1alpha/dnszones
```

All of the machinery this needs already exists in datumctl: keyring-backed
credential storage, an auto-refreshing, self-persisting
`oauth2.TokenSource` (`internal/authutil/credentials.go`), and per-session
endpoint resolution (`internal/client/factory.go`,
`internal/datumconfig`). This enhancement wraps that machinery in a small,
conservative HTTP server, in the spirit of `kubectl proxy` and
`cloud-sql-proxy`.

## Motivation

Today, a local tool that wants to call the Datum Cloud API has three
options, all of which push token plumbing onto the tool:

- Run `datumctl auth get-token` and thread the raw bearer token through
  environment variables or config — and re-thread it when it expires,
  because a static token dies mid-session.
- Be a datumctl *plugin*, and use the `DATUM_CREDENTIALS_HELPER` exec
  protocol (`internal/plugindispatch/dispatch.go`) — which only works for
  processes datumctl itself spawns.
- Re-implement OAuth refresh against the keyring — nobody should do this.

Real teams are hitting this today:

1. **Cloud-portal local development.** The portal dev server needs a
   platform bearer token per request (session-injected today). With the
   proxy, a developer sets `API_URL=http://127.0.0.1:8001` and datumctl
   owns authentication and refresh for the whole dev session. No token in
   `.env` files, no expiry-induced mystery 401s an hour into work.
2. **The portal's plugin-registry watch client.** The portal watches CRDs
   (e.g. `portalplugins.portal.miloapis.com`) on a control plane. Watches
   are long-lived chunked HTTP responses; a proxy that buffers them is
   useless. **Unbuffered streaming passthrough is a hard requirement**, not
   an optimization.
3. **E2E test harnesses.** A test suite can start a proxy on a random free
   port, read the URL from stdout, point every test at it, and never
   distribute credentials into test processes at all. Auth setup for an
   entire suite becomes one spawned process.
4. **Scripting and exploration.** `curl`, `httpie`, or a notebook against
   `localhost` beats copy-pasting tokens that expire mid-investigation.

### Goals

- One command that gives any local HTTP client an authenticated view of the
  active session's Datum Cloud API endpoint.
- Transparent token injection and refresh, reusing the existing
  `authutil` token sources — including service-account sessions — with
  refreshed tokens persisted back to the keyring exactly as other commands
  do.
- Unbuffered passthrough of streaming responses (watch, SSE, chunked
  transfer) suitable for long-lived watch clients.
- A conservative security posture: loopback-only, Host-header validation,
  local `Authorization` headers never forwarded upstream, tokens never
  logged.
- Predictable lifecycle: the session is pinned when the proxy starts, the
  bound URL is machine-readable, failures produce actionable messages.

### Non-Goals

- **Exposing the proxy beyond the local machine.** No flag to bind
  non-loopback addresses ships in v1 (see [Security model](#security-model)).
  Users who want remote access can build their own tunnel and own that
  decision.
- **Being an API gateway.** No caching, no rewriting of response bodies, no
  rate limiting, no request transformation beyond auth injection and
  standard proxy header hygiene.
- **Serving raw tokens over HTTP.** `datumctl auth get-token` exists for
  that, gated by process execution rather than an open local port
  (aws-vault's metadata-server mode shows why serving credentials over
  loopback HTTP is a weaker boundary).
- **Implicit scoping from the current context.** Easy URLs come from the
  explicit `--project`/`--organization` scoped mode, never from silently
  reading the active context; a mixed-mode `/-/` convenience prefix inside
  the unscoped proxy is also deferred (see
  [Path semantics](#path-semantics)).
- **A general-purpose one-shot request command** (`gh api` style). That is a
  natural sibling under the same `api` command group, but a separate
  enhancement.

## Proposal

Add a new `api` command group with a single v1 subcommand, `proxy`,
implemented as a custom command under `internal/cmd/api/` and registered in
`internal/cmd/root.go` like the other custom commands (login, logout, ctx,
docs). The proxy engine itself lives in a new `internal/apiproxy` package so
it can be tested without Cobra, the keyring, or a real network.

On startup the command:

1. Resolves a session — the active session by default, or one pinned with
   `--session` — and from it the user key, the upstream endpoint, and any
   endpoint TLS settings, using the same resolution the request factory in
   `internal/client/factory.go` performs today.
2. Resolves the proxy root: the endpoint root by default, or a single
   control plane when `--project`/`--organization`/`--platform-wide` is
   given (see [Path semantics](#path-semantics)).
3. Builds the auto-refreshing token source via
   `authutil.GetTokenSourceForUser`.
4. Binds a loopback listener (random free port by default, `--port` to fix
   it), prints a human banner to stderr and the bare proxy URL as a single
   line to stdout, and serves until interrupted.

Every proxied request gets a fresh-enough bearer token injected; every
response streams back unbuffered.

### User Stories

#### Story 1: A portal developer stops thinking about tokens

Maya works on the cloud portal. Today her dev server needs a platform token
injected per session, and when it expires she restarts things and mutters.
Now her `package.json` dev script assumes a proxy:

```console
$ datumctl api proxy --port 8001
  Session:    maya@datum.net (api.datum.net)
  Upstream:   https://api.datum.net
  Listening:  http://127.0.0.1:8001
```

She sets `API_URL=http://127.0.0.1:8001` once in `.env.local`. The portal
sends plain HTTP requests; datumctl owns auth for the whole workday,
refreshing the token in the background. When her session eventually needs a
real re-login, the proxy tells her exactly that on stderr and in the error
body, and `datumctl login` fixes it without restarting the dev server.

#### Story 2: The portal watches a CRD through the proxy

The portal's plugin registry watches `portalplugins.portal.miloapis.com` on
a project control plane. Through the proxy this is just:

```
GET /apis/resourcemanager.miloapis.com/v1alpha1/projects/my-project/control-plane/apis/portal.miloapis.com/v1alpha1/portalplugins?watch=true
```

Each watch event flushes through to the client the moment the upstream
sends it. The watch runs for as long as the upstream allows; when the
server ends it (watch timeout, token expiry), the portal's standard
watch-reconnect logic re-establishes it and the new request carries a
freshly refreshed token. Nobody wrote any auth code.

If the registry client prefers a dedicated base URL over path assembly, a
second proxy started with `--project my-project` serves that control plane
at its root, and the watch path shrinks to
`/apis/portal.miloapis.com/v1alpha1/portalplugins?watch=true`.

#### Story 3: An E2E suite gets auth for free

An E2E harness starts `datumctl api proxy` (no `--port`: random free port),
reads the URL from the first line of stdout as its readiness signal, and
passes it to every test worker. No test process ever holds a credential.
Teardown is killing one process.

```go
cmd := exec.Command("datumctl", "api", "proxy", "--quiet", "--project", testProject)
stdout, _ := cmd.StdoutPipe()
_ = cmd.Start()
apiURL, _ := bufio.NewReader(stdout).ReadString('\n') // first line = ready + address
```

#### Story 4: Pinning a non-active session

Sam is logged into both production and staging and keeps a proxy running
against staging while their *active* session stays on production:

```console
$ datumctl auth list
$ datumctl api proxy --session sam@datum.net@api.staging.env.datum.net --port 8002
```

Running `datumctl auth switch` in another terminal does not silently
repoint Sam's staging proxy — the session is pinned at startup.

### Notes/Constraints/Caveats

- **The proxy pins its session at startup.** `datumctl auth switch` mutates
  `ActiveSession` in the on-disk config (`internal/cmd/auth/switch.go`); a
  proxy that followed the active session would change identity *and
  endpoint* under a running dev server mid-request-stream. The proxy
  resolves the session once, states it in the banner, and never re-reads
  the config. Users who want the new session restart the proxy.
- **Pure passthrough means the local client speaks real platform paths.**
  This is a feature: anything recorded against the real API (docs examples,
  HAR files, client SDK output) replays against the proxy unchanged.
- **Scoping is explicit and pinned, never inherited from the current
  context.** For resource commands, `resolveScope` in
  `internal/client/factory.go` falls back to the active context when no
  flag is given; the proxy deliberately does **not**. A proxy whose URLs
  silently mean something different depending on which context was active
  at launch (or worse, changes with `datumctl ctx use`) is a footgun for
  the tools pointed at it — and the flagship portal use case requires the
  unscoped endpoint so dev paths match production paths exactly. No flags →
  endpoint root; `--project`/`--organization`/`--platform-wide` → that
  scope, pinned at startup and shown in the banner, exactly like the
  session.
- **A new built-in `api` command shadows any user plugin named `api`.**
  Plugin dispatch only fires for non-built-in names
  (`internal/cmd/root.go`, `plugindispatch.IsBuiltIn`). No known plugin
  uses the name; worth a release-note line.
- **Response bodies are not rewritten.** Kubernetes-style APIs return
  relative `selfLink`-free bodies, so passthrough is safe. If some platform
  endpoint ever embeds absolute API URLs in response bodies, clients will
  see upstream URLs — acceptable and documented, not silently rewritten.

### Risks and Mitigations

#### Risk: Any local process can act as the user while the proxy runs

The proxy deliberately does not authenticate local clients — same-user
loopback is the trust boundary, matching `kubectl proxy`, cloud-sql-proxy,
and gcloud emulators. But loopback is reachable by *every* local process,
including other users on a shared machine.

*Mitigations:* loopback-only binding with no override flag; the proxy only
exists while the user runs it, and its startup banner names the identity it
serves; every request is logged by default, so misuse is visible; the token
itself is never exposed — a local client can make API calls but cannot
exfiltrate the credential to use elsewhere, which is a strictly better
boundary than token-in-env approaches it replaces. A Unix-socket listener
(file-permission-enforced, same-user-only) is the natural hardening step
and is listed under [Open Questions](#open-questions).

#### Risk: Browsers as confused deputies (DNS rebinding, CSRF)

A malicious web page can make a victim's browser send requests to
`127.0.0.1`, and DNS rebinding can defeat the same-origin policy if the
proxy answers arbitrary `Host` headers.

*Mitigations:* the proxy rejects any request whose `Host` is not
`localhost`, `127.0.0.1`, or `[::1]` (with the bound port) — this defeats
DNS rebinding. It emits **no** CORS headers, so same-origin policy blocks
scripted reads from web pages. The cloud-portal use case is unaffected: the
portal's *dev server* (a local non-browser process) calls the proxy;
browsers talk to the dev server. Browser-direct use would need an explicit
opt-in CORS flag, deferred.

#### Risk: Buffering silently breaks watch clients

An innocent-looking default (a buffered reverse proxy, a response timeout)
would make watches appear to "hang" and fail only under real use.

*Mitigations:* unbuffered flush is a stated hard requirement with a
dedicated integration test that fails if events do not arrive
incrementally (see [Testing strategy](#testing-strategy)); no
response-duration timeout exists anywhere in the path.

#### Risk: Refresh storms against the token endpoint

A polling dev server multiplies requests; if the refresh token is dead,
each request could trigger a fresh refresh attempt against the auth server.

*Mitigations:* refresh is already single-flight — `persistingTokenSource`
serializes on a mutex (`internal/authutil/credentials.go`). The proxy adds
a short failure cooldown: after a refresh failure, requests fail fast with
the same actionable error for a few seconds instead of re-attempting
refresh per request.

## Design Details

### CLI surface

```console
$ datumctl api proxy --help
Start a local proxy that authenticates requests to the Datum Cloud API.

The proxy listens on 127.0.0.1 and forwards every request to the API
endpoint of your datumctl session, adding your credentials automatically
and refreshing them as needed. Point any local tool at the printed URL —
no tokens to copy, no expiry to manage.

By default the proxy serves the full API endpoint, so requests use the
same paths as the real API. Pass --project or --organization to serve a
single control plane instead, with shorter paths.

The session and scope are pinned when the proxy starts. Switching your
active account or context does not affect a running proxy.

Usage:
  datumctl api proxy [flags]

Flags:
      --port int            Local port to listen on (default: a random free port)
      --session string      Pin a specific session by name (defaults to the
                            active session; see 'datumctl auth list')
      --project string      Serve this project's control plane as the proxy root
      --organization string Serve this organization's control plane as the proxy root
  -q, --quiet               Suppress per-request log lines
```

`--project`, `--organization`, and `--platform-wide` are the root command's
existing persistent flags (`DatumCloudFactory.AddFlags`,
`internal/client/factory.go`), already mutually exclusive; the proxy reads
them at launch rather than defining its own. Unlike resource commands, the
proxy does **not** fall back to the active context when no flag is given —
see [Path semantics](#path-semantics).

Registered in `internal/cmd/root.go` as a custom command group:

```go
apiCmd := apicmd.Command()   // internal/cmd/api
apiCmd.GroupID = "other"
rootCmd.AddCommand(apiCmd)
```

`Long` and `Example` use `templates.LongDesc()` / `templates.Examples()`
per repo convention, reference only `datumctl` and Datum Cloud resources,
and never mention kubectl:

```console
  # Start a proxy on a fixed port for a dev server
  datumctl api proxy --port 8001

  # Start on a random free port; the URL is printed on the first stdout line
  datumctl api proxy

  # List organizations through the proxy
  curl http://127.0.0.1:8001/apis/resourcemanager.miloapis.com/v1alpha1/organizations

  # Watch DNS zones on a project control plane through the proxy
  curl "http://127.0.0.1:8001/apis/resourcemanager.miloapis.com/v1alpha1/projects/my-project/control-plane/apis/networking.datumapis.com/v1alpha/dnszones?watch=true"

  # Serve one project's control plane directly, for shorter URLs
  datumctl api proxy --port 8001 --project my-project
  curl "http://127.0.0.1:8001/apis/networking.datumapis.com/v1alpha/dnszones?watch=true"

  # Pin a non-active session
  datumctl api proxy --session sam@datum.net@api.staging.env.datum.net
```

Flag decisions:

- **`--port` defaults to 0 (random free port).** A fixed default (kubectl
  proxy's 8001) fails on the second concurrent proxy and trains tools to
  assume a well-known port that another local program may squat. Random
  never fails at startup; stable configurations opt in with `--port 8001`.
  The bound URL is always printed, so nothing is guessing.
- **`--session` names a session** (format `email@api-hostname`, from
  `datumconfig.SessionName`), mirroring the existing `--session` flag on
  `datumctl auth get-token` (`internal/cmd/auth/get_token.go`). No separate
  `--user` flag: emails can match multiple endpoints
  (`SessionByEmail` returns a slice), and the session name is the
  unambiguous handle `auth list` already shows.
- **No `--listen`/`--address` flag in v1.** Loopback-only is a design
  guarantee, not a default (see [Security model](#security-model)).

### Resolving the session and upstream

The command reuses the exact resolution chain the request factory performs
in `internal/client/factory.go`, minus the org/project scoping (the proxy
targets the endpoint root):

1. `datumconfig.LoadAuto()` +
   `authutil.EnsureUserKeysMigrated` — load the v1beta1 config, as
   `CustomConfigFlags.loadDatumContext` does (factory.go).
2. Pick the session: `cfg.SessionByName(--session)` if pinned, else
   `cfg.ActiveSessionEntry()`; fall back to the keyring active user
   (`authutil.GetActiveCredentials`) for pre-session setups, matching
   `resolveUserKey`.
3. Upstream base URL: `session.Endpoint.Server` →
   `datumconfig.CleanBaseServer(datumconfig.EnsureScheme(...))`, falling
   back to `authutil.GetAPIHostnameForUser(userKey)` — the same order as
   `resolveBaseServer` (factory.go).
4. Upstream TLS: honor the session's `Endpoint.TLSServerName`,
   `Endpoint.InsecureSkipTLSVerify`, and base64
   `Endpoint.CertificateAuthorityData`, exactly as `ToRESTConfig` applies
   them (factory.go) — required for staging endpoints with private CAs.
5. Scoped mode: if `--project`/`--organization`/`--platform-wide` is set,
   compute the proxy root with `miloapi.ProjectControlPlaneURL` /
   `OrgControlPlaneURL` (or the base itself for `--platform-wide`) —
   mirroring the `switch` in `ToRESTConfig` (factory.go), *without* its
   env-var and active-context fallbacks.
6. Token source: `authutil.GetTokenSourceForUser(ctx, userKey)`
   (`internal/authutil/credentials.go`), which transparently handles both
   interactive OAuth refresh (`persistingTokenSource`, persists refreshed
   tokens to the keyring) and service-account JWT re-mint
   (`serviceAccountTokenSource`). The proxy inherits machine-account
   support for free.

Steps 2–4 duplicate logic currently private to `CustomConfigFlags`. Rather
than copy it, extract a small shared helper (e.g.
`client.ResolveSessionEndpoint(cfg, sessionName) (userKey, baseURL,
tlsConfig, error)`) used by both `ToRESTConfig` and the proxy, so the
kubectl-wrapped path and the proxy path cannot drift.

Proposed layout:

```
internal/apiproxy/           # engine, no Cobra/keyring imports
  proxy.go                   # Server{Upstream, TokenSource, TLS, Logger}; ReverseProxy setup
  handler.go                 # host validation, header hygiene, error mapping
  logging.go                 # request log lines, redaction rules
internal/cmd/api/
  api.go                     # parent 'api' group command
  proxy.go                   # flags, session resolution, banner, signal handling
```

### The proxy engine and streaming design

The engine is a `net/http/httputil.ReverseProxy` — the same primitive
kubectl proxy is built on — configured for unbuffered streaming:

```go
proxy := &httputil.ReverseProxy{
    Rewrite: func(r *httputil.ProxyRequest) {
        r.Out.URL.Scheme = upstream.Scheme
        r.Out.URL.Host = upstream.Host
        r.Out.Host = upstream.Host
        // Never forward a locally supplied credential upstream; the
        // oauth2.Transport below sets the real one.
        r.Out.Header.Del("Authorization")
    },
    Transport: &oauth2.Transport{
        Source: tokenSource,     // from authutil.GetTokenSourceForUser
        Base:   upstreamTransport,
    },
    FlushInterval: -1,           // flush every write immediately: watch/SSE
    ErrorHandler:  writeProxyError,
}
```

Key properties:

- **`FlushInterval: -1`** flushes each upstream write to the local client
  immediately — this is the entire unbuffered-streaming design, and it is
  the setting kubectl proxy uses for watch support. Chunked
  transfer-encoding and `text/event-stream` need no additional handling.
- **Token injection via `oauth2.Transport`** is byte-for-byte the pattern
  `CustomConfigFlags.ToRESTConfig` already uses
  (`config.WrapTransport = ... &oauth2.Transport{Source: tknSrc, Base: rt}`
  in factory.go). If the token source fails, the request is never sent and
  the error surfaces in `ErrorHandler`.
- **Timeouts are asymmetric by design.** Connection setup is bounded
  (dial ~10s, TLS handshake ~10s, response headers ~30s); response
  *duration* is unbounded — watches are infinite on purpose. The local
  `http.Server` sets only `ReadHeaderTimeout` (slowloris hygiene; loopback
  makes it low-stakes) and a keep-alive `IdleTimeout`; no `ReadTimeout` or
  `WriteTimeout`, which would kill streams.
- **Upgrades (WebSockets)** pass through via `ReverseProxy`'s built-in
  `Connection: Upgrade` handling. v1 treats this as best-effort
  passthrough, not a tested guarantee: the bearer token is injected at
  handshake time only, and post-expiry connection lifetime is upstream
  policy. The motivating watch client uses chunked HTTP, not WebSockets.
- **HTTP/2 upstream** is fine (multiplexed watches share a connection);
  the local side is HTTP/1.1, which every target client (curl, Node fetch,
  Go clients) handles, and which chunked watch streaming works over.
- **Compression is passed through**, not re-encoded: the client's
  `Accept-Encoding` travels upstream and the response body is relayed
  verbatim.
- **Header hygiene:** hop-by-hop headers are stripped by `ReverseProxy`
  per RFC 7230; inbound `Authorization` is deleted (above) so a local
  client can never smuggle an alternate credential upstream or trick the
  proxy into forwarding a stale one; no `X-Forwarded-*` headers are added
  (`Rewrite` without `SetXForwarded` — the upstream gains nothing from
  knowing about 127.0.0.1). All other headers, including inbound
  `X-Request-ID`, pass through untouched.

### Path semantics

The proxy has exactly one knob: **what its root maps to**. Requests are
otherwise forwarded verbatim — path and query untouched.

**Default: the endpoint root (pure passthrough).** Because scoped control
planes are just path prefixes under the endpoint —
`internal/miloapi/urls.go` builds them as
`/apis/resourcemanager.miloapis.com/v1alpha1/projects/{id}/control-plane`
(and the org/user equivalents) — every control plane is reachable through
one unscoped proxy:

| Local request path | Meaning |
| --- | --- |
| `/apis/…/organizations` | platform root (endpoint root) |
| `/apis/resourcemanager.miloapis.com/v1alpha1/organizations/{org}/control-plane/…` | org control plane |
| `/apis/resourcemanager.miloapis.com/v1alpha1/projects/{proj}/control-plane/…` | project control plane |
| `/apis/iam.miloapis.com/v1alpha1/users/{uid}/control-plane/…` | user control plane |

**Scoped mode: `--project` / `--organization` / `--platform-wide`
re-base the proxy root at that control plane**, using the same
`miloapi.*ControlPlaneURL` helpers `ToRESTConfig` uses (factory.go). A
scoped proxy presents a complete, single API-server surface at `/` —
discovery under `/apis`, resources at their natural short paths — so a
generic Kubernetes-style client library can be pointed at
`http://127.0.0.1:PORT` as its base URL with no path assembly at all:

```console
$ datumctl api proxy --project my-project --port 8001
$ curl "http://127.0.0.1:8001/apis/networking.datumapis.com/v1alpha/dnszones?watch=true"
```

**Why scoping is explicit rather than inherited from the current
context.** It is tempting to default to the active context (as resource
commands do via `resolveScope`), making URLs short out of the box. Rejected
for v1, deliberately:

- *Dev/prod parity is the flagship requirement.* The cloud portal talks to
  `https://api.datum.net` in production using full paths; `API_URL`
  swapping only works if the proxy accepts those same paths. A
  context-scoped default would force portal code to use different paths in
  development than in production.
- *The default example would break.* `…/organizations` is served at the
  platform root, not under a project control plane; a project-scoped
  default turns the most obvious first request into a 404.
- *Ambient state is the enemy of a proxy.* Tools cache the proxy URL. If
  its meaning depended on whatever context was active at launch — or
  followed `datumctl ctx use` live — the same URL would silently address
  different control planes on different days. Scope, like the session, is
  pinned at startup, explicit in the command line, and shown in the banner.

Convenience roots serving *both* at once (a reserved `/-/` prefix inside
the unscoped proxy, e.g. `/-/project/my-proj/…`) are **deferred**: scoped
mode already delivers the short-URL ergonomics without a reserved
namespace or rewrite rules. The `/-/` prefix is noted as reserved so a
future addition is non-breaking.

### Token lifecycle

- **Refresh-ahead-of-expiry.** `oauth2`'s reuse semantics refresh a token
  shortly before its recorded expiry on the next request; refreshed tokens
  are persisted to the keyring by `persistingTokenSource`
  (`internal/authutil/credentials.go`), so a long proxy run keeps the
  user's stored session fresh for every other datumctl command too.
  Service-account sessions re-mint their JWT and re-exchange on expiry via
  `serviceAccountTokenSource` — no proxy-specific code.
- **Mid-stream expiry.** Requests are authenticated at request start; the
  proxy never interrupts a response because the token that opened it has
  since expired. Whether a long-lived watch outlives its token is upstream
  policy. When the upstream ends the stream — token expiry, watch timeout,
  anything — the client's normal watch-reconnect (resourceVersion resume)
  opens a new request, which gets a fresh token. This matches how
  Kubernetes-ecosystem clients already behave and requires nothing from
  the proxy.
- **Refresh failure.** `persistingTokenSource` already maps `invalid_grant`
  / `invalid_request` to a `UserError` with the hint to re-run
  `datumctl login`. The proxy's `ErrorHandler` turns any token-source
  failure into a synthesized **`502 Bad Gateway`** — deliberately *not*
  `401`: a passthrough `401` must mean "the platform rejected this
  request," so client-side re-auth logic never misfires on a proxy-local
  problem. The body is a Kubernetes-style `Status` object (the dialect the
  target clients parse), plus a marker header:

  ```
  HTTP/1.1 502 Bad Gateway
  Content-Type: application/json
  X-Datum-Proxy-Error: true

  {"kind":"Status","apiVersion":"v1","status":"Failure","code":502,
   "reason":"ProxyAuthenticationFailed",
   "message":"datumctl session expired or revoked — run 'datumctl login' to re-authenticate"}
  ```

  The same message is logged once to stderr (not per request). An upstream
  `401`/`403` passes through byte-for-byte with no marker header.
- **Backoff.** After a refresh failure, a cooldown (~5s) short-circuits
  further refresh attempts; requests during the cooldown fail immediately
  with the same 502. Combined with the mutex in `persistingTokenSource`,
  the auth server sees at most one refresh attempt per cooldown window no
  matter how hot the local client polls.
- **Credentials deleted mid-run** (`datumctl logout`): the next refresh
  fails and the proxy degrades to the 502-with-hint behavior. It stays up
  — a dev server pointed at it keeps getting actionable errors instead of
  connection refused — and recovers without restart if the user logs back
  in to the same session (the token source re-reads the keyring on
  refresh).

### Security model

Conservative by default, with the rationale stated so future changes are
deliberate:

- **Loopback only, no override.** The listener binds `127.0.0.1` (v1 does
  not bind `::1`; see Open Questions). There is no flag to bind other
  addresses — not even a scary one — because every legitimate "remote
  proxy" story we could name is better served by the user running their own
  tunnel (SSH, tailscale) whose security model they already own. Loopback
  binding also has in-repo precedent: the PKCE login flow's callback server
  binds `localhost:0` (`internal/authutil/login.go`).
- **Same-user loopback is the trust boundary — with eyes open.** Like
  `kubectl proxy` and cloud-sql-proxy, the proxy does not authenticate
  local clients: requiring a local secret would just recreate the token
  plumbing this command exists to remove. The residual risk (any local
  process, or another user on a shared host, can use the proxy while it
  runs) is bounded by proxy lifetime, named in the docs, mitigated by
  default request logging, and — unlike token-in-env — never exposes the
  credential itself. A Unix-socket mode with `0600` permissions is the
  documented hardening path (Open Questions).
- **Host-header validation** (DNS-rebinding defense): requests whose
  `Host` is not `localhost`, `127.0.0.1`, or `[::1]` — with or without the
  bound port — are rejected with `403` before proxying. This is the same
  defense kubectl proxy's default `--accept-hosts` regex provides, made
  non-configurable.
- **No CORS headers, ever, in v1.** Without `Access-Control-Allow-Origin`,
  browsers refuse scripted cross-origin reads of proxy responses, closing
  the malicious-web-page vector. The cloud-portal case is server-to-server
  and unaffected. If browser-direct access is ever wanted, it arrives as an
  explicit `--cors-allow-origin` flag in a later revision — off by
  default, exact-origin only, no `*`.
- **Auth header discipline.** Inbound `Authorization` is stripped
  unconditionally before the real token is injected — local clients cannot
  smuggle credentials upstream through the proxy, and stale tokens baked
  into a client config are ignored rather than half-working. No other
  identity-bearing headers are synthesized by the proxy.
- **Tokens never appear in logs.** The request log records method, path,
  status, duration, and byte counts. Query strings are logged (watch
  debugging needs `?watch=true&resourceVersion=…`) with a defensive
  redaction pass for token-shaped parameter names (`access_token`, `token`,
  `authorization`), even though the platform API never puts credentials in
  queries. Headers are never logged at default verbosity; `-v 6+` HTTP
  dumps go through client-go's `transport.DebugWrappers` (already wired in
  `root.go`'s `PersistentPreRunE`), which masks bearer tokens.

### Request logging

One line per request to stderr, on by default, silenced by `--quiet`:

```
10:42:03 GET  /apis/resourcemanager.miloapis.com/v1alpha1/organizations 200 143ms 8.1kB
10:42:05 GET  /apis/…/projects/my-project/control-plane/…/portalplugins?watch=true 200 …streaming
```

Streaming responses log when headers are sent (marked `…streaming`) and
again on stream end with total duration and bytes, so an abruptly closed
watch is visible. The proxy generates an internal per-request ID used only
to correlate its own log lines at higher verbosity; inbound `X-Request-ID`
passes through to the upstream untouched.

### Failure and lifecycle UX

- **Startup banner** (stderr) names the identity, upstream, and address —
  the things a user must be able to verify at a glance:

  ```
    Session:    maya@datum.net (api.datum.net)
    Upstream:   https://api.datum.net
    Scope:      full endpoint (use --project/--organization to serve one control plane)
    Listening:  http://127.0.0.1:52347

    Press Ctrl+C to stop. Requests are logged below (silence with --quiet).
  ```

  In scoped mode the `Scope` line names the control plane (e.g.
  `project my-project`) and `Upstream` shows the full control-plane URL.

- **Machine-readable readiness:** the bare URL (`http://127.0.0.1:52347`)
  is printed as the **first and only line on stdout**, after the listener
  is bound and serving. Harnesses read one line and go; no port files, no
  sleep-and-retry.
- **Startup failures are `UserError`s** (`internal/errors`) with hints:
  no session → "run `datumctl login`"; unknown `--session` → list names via
  `datumctl auth list`; port in use → suggest another `--port` or omitting
  the flag.
- **Graceful shutdown:** on SIGINT/SIGTERM the listener closes, in-flight
  non-streaming requests get a ~2s grace, and long-lived streams are then
  cut — watch clients treat that as a normal stream end and reconnect
  (to a dead port, failing cleanly). A second signal exits immediately.
- **`datumctl auth switch` while running:** no effect, by design — the
  session was pinned at startup (see
  [Notes/Constraints/Caveats](#notesconstraintscaveats)). The banner is the
  contract: the proxy serves the identity it printed until it exits.

### Prior art

| Tool | Take | Leave |
| --- | --- | --- |
| `kubectl proxy` | `ReverseProxy` + `FlushInterval: -1` for watch streaming; loopback default; Host-header acceptance as rebinding defense | Fixed default port 8001 (collides); configurable `--accept-hosts`/`--address` foot-guns (documented rebinding incidents when loosened); path-filter machinery (`--reject-paths`) we don't need — Datum's surface has no exec/attach-style local-effect endpoints; `--www` static file serving (scope creep) |
| cloud-sql-proxy | Explicit machine-readable readiness signal; pin the target at startup; background credential refresh as a first-class lifecycle concern | TCP-level opacity — being HTTP-level lets us inject headers, validate Host, and log requests |
| `gh api` | Proof that "CLI owns auth, tool speaks plain HTTP" is the right developer contract; the `api` command-group naming | It's per-invocation — useless as an `API_URL` for a dev server; a future `datumctl api request` sibling can cover the one-shot case |
| aws-vault (`--server` / ECS-metadata mode) | The cautionary tale: it serves *raw credentials* over loopback HTTP, so any local requester can exfiltrate them; aws-vault added a random-token handshake to patch this | Our proxy never serves the token — local processes can act *through* it while it runs, but can't take the credential with them. Keep `auth get-token` (process-exec gated) as the only raw-token path |

### Testing strategy

The engine takes an injected `oauth2.TokenSource` and upstream URL, so unit
and integration tests need no keyring, no Cobra, and no network beyond
`httptest`.

- **Passthrough unit tests** (`internal/apiproxy`, httptest upstream):
  bearer token injected; inbound `Authorization` never reaches upstream;
  method/body/query/`X-Request-ID` pass through; hop-by-hop headers
  stripped; no CORS headers on responses; `Host: evil.example` → 403 and
  the upstream sees nothing; upstream 401 passes through without
  `X-Datum-Proxy-Error`.
- **Watch/streaming integration test — the load-bearing one.** The
  httptest upstream emulates a Kubernetes-style watch: writes one JSON
  watch event, flushes (`http.Flusher`), waits on a channel, repeats. The
  test client reads through the proxy and asserts each event is received
  **before** the test unblocks the next write — proving no proxy-side
  buffering with zero timing flakiness (the upstream cannot even produce
  event N+1 until the client has observed event N). Variants: chunked
  JSON watch, `text/event-stream`, and a slow trickle with small writes.
- **Token lifecycle tests.** A fake `TokenSource` with a controllable
  clock: (a) token refreshed at most once across N concurrent requests
  (single-flight); (b) a stream opened before expiry survives expiry
  mid-stream; the next new request carries the refreshed token; (c)
  refresh failure → 502 with `Status` body, `X-Datum-Proxy-Error: true`,
  and at most one refresh attempt per cooldown window under concurrent
  load.
- **Lifecycle tests** (`internal/cmd/api`): with `--port 0`, the first
  stdout line parses as a URL and a request to it succeeds; SIGINT
  terminates within the grace period with a clean exit; `--session
  bogus` and no-active-session produce `UserError`s with the documented
  hints.
- **Resolution tests**: the shared session-endpoint resolver honors
  `Endpoint.TLSServerName` / `CertificateAuthorityData` /
  `InsecureSkipTLSVerify` identically to `ToRESTConfig` (table test over a
  fixture `ConfigV1Beta1`), pinning the no-drift guarantee.
- **Scoped-mode tests**: with `--project`, local `/apis/...` paths arrive
  upstream under the project control-plane prefix (and the watch streaming
  test re-runs against a scoped proxy); with no scope flag and an active
  project context in the fixture config, the proxy root is still the
  endpoint root — pinning the no-context-inheritance rule.
- **Manual verification** against staging
  (`--hostname auth.staging.env.datum.net` login): a real
  `datumctl get dnszones --watch` equivalent via `curl` through the proxy,
  left running past a token expiry.

### V1 milestone cut

In scope for the first release:

- `datumctl api proxy` with `--port`, `--session`, `--quiet`, and
  launch-time scoping via `--project`/`--organization`/`--platform-wide`
- Pure passthrough by default; scoped mode re-basing at one control plane;
  unbuffered streaming (chunked + SSE); best-effort Upgrade passthrough
- Token injection, refresh, keyring persistence, service-account support,
  502-with-hint on refresh failure, refresh cooldown
- Loopback-only bind, Host validation, no CORS, Authorization stripping,
  redacting request log
- Banner + stdout URL readiness contract, graceful shutdown, pinned session
- The test suite above; docs page under `docs/`

Explicitly deferred (in likely priority order):

1. `--unix-socket` listener (strongest same-user boundary; also nice for
   hermetic E2E)
2. Convenience scoped roots under the reserved `/-/` prefix
3. `--cors-allow-origin` for browser-direct use
4. `datumctl api request` one-shot sibling (`gh api` analog)
5. Tested WebSocket guarantee, if a platform feature comes to need it

## Open Questions

1. **Bind `::1` as well as `127.0.0.1`?** Clients that resolve `localhost`
   to `::1` first would otherwise fail. Leaning yes (two listeners, one
   port), but it doubles the Host/binding matrix; decide during
   implementation.
2. **Retry-once on upstream 401?** If the upstream rejects a token the
   proxy believed valid (clock skew, early revocation), should the proxy
   force-refresh and retry idempotent requests once before passing the 401
   through? kubectl does not; v1 passes through. Revisit with field data.
3. **Unix-socket mode timing** — fast-follow or v1? It is the honest answer
   to the shared-machine caveat, and the E2E harness would prefer it. Cut
   from v1 only for scope discipline.
4. **Session liveness UX.** Should the banner (or a `-v` log line) surface
   token expiry/refresh events so a developer can see the refresh machinery
   working? Cheap and probably worth it; needs a non-noisy format.
5. **`datumctl proxy` alias.** Is the extra `api` level worth it? This doc
   says yes — it groups a future `api request` and keeps the root
   uncluttered — but if usage shows the command is overwhelmingly
   interactive, a hidden top-level alias is cheap.
6. **Should a bare `datumctl api proxy` ever inherit the active context?**
   This doc says no (see [Path semantics](#path-semantics)): parity with
   production paths and freedom from ambient state outweigh saving one
   flag. Flagged for cloud-portal review, since they are the consumers who
   would feel a wrong default first; if interactive users consistently
   expect context inheritance, an explicit `--current-context` opt-in is
   the compromise that preserves both properties.

## Production Readiness Review Questionnaire

### Feature enablement and rollback

- **How can this feature be enabled / disabled?** It is a new opt-in
  command; not running it disables it. No config flags, no state left
  behind after exit (the only persistent side effect — refreshed tokens in
  the keyring — is identical to running any other datumctl command).
- **Can the feature be rolled back?** Yes; removing the command removes
  the feature. Local tools pointed at a dead proxy fail with connection
  refused, which is the same failure mode as the proxy simply not running.

### Monitoring and supportability

- Per-request log lines (with streaming start/end markers) are the primary
  diagnostic; `-v 6+` reuses the existing client-go debug transport wiring
  in `root.go` with token masking.
- Proxy-synthesized errors are distinguishable from upstream errors by the
  `X-Datum-Proxy-Error` header and `Status.reason`, so support can tell
  "your session is dead" from "the platform said no" from a single client
  screenshot.

### Dependencies

- Standard library `net/http/httputil` and the already-vendored
  `golang.org/x/oauth2`. No new third-party dependencies.
- Runtime dependencies are the existing keyring and datum config; both are
  already required for every authenticated command.

### Security

- See [Security model](#security-model). Summary of the reviewed posture:
  loopback-only without override, Host-header validation, no CORS, no
  local-client authentication (accepted, documented trust boundary),
  inbound Authorization stripped, tokens never logged, raw tokens never
  served. The threat model change vs. status quo is that a credential
  formerly exposed *as a value* (token in env/config for the dev server) is
  replaced by an *ambient capability* scoped to proxy lifetime — an
  intentional improvement.

## Implementation History

- 2026-07-11: Provisional proposal drafted.
- 2026-07-11: V1 implemented (`internal/apiproxy` engine, `internal/cmd/api`
  command, shared `internal/client/session_endpoint.go` resolver, docs page
  at `docs/api-proxy.md`) with the full test suite from
  [Testing strategy](#testing-strategy); not yet merged.

## Drawbacks

- **A standing ambient capability on loopback.** While the proxy runs, any
  local process can act as the user against the pinned endpoint. This is
  the price of removing token plumbing, shared with every tool in the
  prior-art table; mitigations and the Unix-socket path are covered above.
- **Another long-running surface to support.** Streaming, timeout, and
  shutdown behavior must be defended against regressions (Go version bumps
  can change `ReverseProxy`/HTTP2 details); the watch integration test is
  the tripwire.
- **`api` becomes a reserved name**, shadowing any user plugin called
  `api` via the dispatch path in `internal/cmd/root.go`.
- **Passthrough honesty has edges**: response bodies embedding absolute
  upstream URLs (none known today) would leak the real endpoint to clients
  configured only with the proxy URL.

## Alternatives

- **Status quo: `datumctl auth get-token` + env var.** Works for
  one-shots; fails the dev-server case (token expires mid-session, token
  value exposed to the whole process tree) and the watch case (no refresh
  across reconnects without re-running the CLI).
- **The plugin credentials-helper protocol** (`DATUM_CREDENTIALS_HELPER`
  exec, `internal/plugindispatch/dispatch.go`): right answer for datumctl
  plugins, unavailable to processes datumctl didn't spawn (the portal dev
  server, curl), and requires client-side integration code the proxy makes
  unnecessary.
- **A one-shot `datumctl api request <path>`** (`gh api` model): solves
  scripting, not the `API_URL=`-for-a-dev-server or watch cases; remains a
  good future sibling.
- **Teach the portal dev server to exec datumctl itself:** couples one
  consumer to CLI internals and helps no other tool; the proxy solves the
  class.
- **`auth update-kubeconfig` + an external kubectl proxy:** kubectl-users
  only, drags Kubernetes tooling into a workflow this product deliberately
  keeps Kubernetes-free, and still lacks session pinning against datumctl's
  own config.
- **Serve the token itself on loopback** (aws-vault-metadata style):
  strictly weaker boundary — a local requester keeps the credential after
  the proxy exits. Rejected on principle; `auth get-token` remains the
  only raw-token path, gated by process execution.

## Infrastructure Needed

None. The feature is entirely client-side; no new services, endpoints, or
repositories.
