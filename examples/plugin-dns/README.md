# datumctl-dns — Reference Plugin Example

This directory contains a reference implementation of a datumctl plugin using
the Go SDK (`go.datum.net/datumctl/plugin`). It demonstrates the complete
end-to-end integration: manifest protocol, context injection, and token
acquisition.

## Quick Start for Plugin Authors

### 1. Add the SDK dependency

```sh
go get go.datum.net/datumctl/plugin
```

The SDK lives inside the main `datumctl` module. You get context reading,
credential helper invocation, and pre-wired Cobra flags in a single dependency
with no `internal/` exposure.

### 2. Declare your manifest and call ServeManifest

Call `plugin.ServeManifest(m)` at the very top of `main()`, before Cobra runs:

```go
var m = plugin.Manifest{
    Name:          "dns",
    Version:       "v0.1.0",
    Description:   "Manage Datum Cloud DNS zones",
    APIVersion:    1,
    MinAPIVersion: 1,
}

func main() {
    plugin.ServeManifest(m) // handles --plugin-manifest and exits
    // ...
}
```

`ServeManifest` scans `os.Args` for `--plugin-manifest`. When found, it prints
the manifest as JSON to stdout and exits 0. This must run before `cobra.Execute()`
so it works even when other flags would fail parsing.

### 3. Use NewRootCmd for pre-wired flags

```go
root := plugin.NewRootCmd("dns", "Manage Datum Cloud DNS resources")
```

This gives you `--org`, `--project`, and `--output` flags pre-populated from
the `DATUM_ORG`, `DATUM_PROJECT`, and default values injected by datumctl.

### 4. Read context and fetch tokens

```go
ctx := plugin.Context()      // reads all DATUM_* env vars
token, err := plugin.Token() // calls $DATUM_CREDENTIALS_HELPER auth get-token
```

`Token()` automatically passes `--session $DATUM_SESSION` when a session is
active, and omits it when `DATUM_SESSION` is empty. Call `Token()` immediately
before each API request — tokens are short-lived.

### 5. Build and test locally

```sh
# Build the plugin binary
go build -o datumctl-dns .

# Place it on your PATH or in your managed plugins dir
cp datumctl-dns ~/.datumctl/plugins/

# Run it via datumctl (context and credentials are injected automatically)
datumctl dns zones list

# Test the manifest protocol
./datumctl-dns --plugin-manifest
```

## Distributing Your Plugin

There are two distribution paths: the **curated index** (recommended for
first-party plugins) and **direct GitHub install** (for third-party plugins).

### Curated index (datumctl plugin install dns)

The curated index lives at
[datum-cloud/datumctl-plugins](https://github.com/datum-cloud/datumctl-plugins).
When your plugin is listed there, users install it with just a name:

```sh
datumctl plugin install dns
```

To submit a plugin:

1. Add the `datumctl-plugin` topic to your GitHub repository.
2. Open a PR adding `plugins/<your-plugin-name>.yaml` to the index repo,
   following the [schema](https://github.com/datum-cloud/datumctl-plugins/blob/main/schema/plugin-v1alpha1.json):

```yaml
apiVersion: datumctl.datum.net/v1alpha1
kind: Plugin
metadata:
  name: dns
spec:
  shortDescription: Manage Datum Cloud DNS zones
  homepage: https://github.com/datum-cloud/datumctl-dns
  version: v1.0.0
  platforms:
    - selector:
        matchLabels:
          os: linux
          arch: amd64
      uri: https://github.com/datum-cloud/datumctl-dns/releases/download/v1.0.0/datumctl-dns_Linux_x86_64.tar.gz
      sha256: <lowercase hex sha256 of the archive>
    - selector:
        matchLabels:
          os: darwin
          arch: arm64
      uri: https://github.com/datum-cloud/datumctl-dns/releases/download/v1.0.0/datumctl-dns_Darwin_arm64.tar.gz
      sha256: <lowercase hex sha256 of the archive>
```

The binary inside each archive must be named `datumctl-<name>` (or
`datumctl-<name>.exe` on Windows).

### Direct GitHub install (datumctl plugin install owner/repo)

Third-party plugins can be installed directly from any GitHub Release without
being listed in the curated index:

```sh
datumctl plugin install datum-cloud/datumctl-dns          # latest release
datumctl plugin install datum-cloud/datumctl-dns@v1.0.0   # specific version
```

This path requires a `checksums.txt` file alongside the release archives, in
goreleaser's default two-column format:

```
abc123...  datumctl-dns_Linux_x86_64.tar.gz
def456...  datumctl-dns_Darwin_arm64.tar.gz
```

Use goreleaser with the following asset naming convention:

| GOOS    | GOARCH | Filename                              |
|---------|--------|---------------------------------------|
| linux   | amd64  | `datumctl-dns_Linux_x86_64.tar.gz`   |
| linux   | arm64  | `datumctl-dns_Linux_arm64.tar.gz`    |
| darwin  | amd64  | `datumctl-dns_Darwin_x86_64.tar.gz`  |
| darwin  | arm64  | `datumctl-dns_Darwin_arm64.tar.gz`   |
| windows | amd64  | `datumctl-dns_Windows_x86_64.zip`    |

## Environment Variables Injected by datumctl

datumctl sets these variables before exec-replacing with the plugin binary:

| Variable                   | Description                                        |
|----------------------------|----------------------------------------------------|
| `DATUM_ORG`                | Current organization slug                          |
| `DATUM_PROJECT`            | Current project slug (empty if not set)            |
| `DATUM_API_HOST`           | API base URL (e.g. `api.datum.net`)                |
| `DATUM_PLUGIN_API_VERSION` | Integer API version declared by this host          |
| `DATUM_CREDENTIALS_HELPER` | Absolute path to the datumctl binary               |
| `DATUM_SESSION`            | Active session name (may be empty)                 |

The Go SDK's `plugin.Context()` reads all of these automatically.

## Credentials Helper Protocol

Plugins fetch tokens by running:

```sh
$DATUM_CREDENTIALS_HELPER auth get-token --session $DATUM_SESSION
```

When `DATUM_SESSION` is empty, omit the `--session` flag. The Go SDK's
`plugin.Token()` handles this automatically.

## Shell Completion

datumctl forwards `__complete` invocations to the plugin binary, so your
Cobra command's built-in completion works transparently when users run:

```sh
datumctl dns <TAB>
```

No special configuration is needed — this is automatic.
