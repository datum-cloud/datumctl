// Package version derives a fallback for k8s.io/component-base/version's
// GitVersion when datumctl is built without the ldflags that release, nix,
// and `task build` all inject (e.g. a bare `go run .` or `go test`). Go's
// toolchain stamps VCS info into the binary automatically from the module's
// git checkout, so this uses that instead of leaving component-base's
// compiled-in placeholder, which fails to parse in places that expect a real
// semantic version (plugin compatibility checks, the update checker).
//
// The fallback can only ever be "v0.0.0+<sha>[-dev]", never a real tag:
// component-base's SetDynamicVersion rejects any version whose major/minor/
// patch don't match the compiled-in placeholder's (0.0.0), and Go's VCS
// stamping has no notion of the nearest tag anyway. Getting the real last
// tag requires setting GitVersion via ldflags at compile time instead, which
// is what `task build` (and nix/goreleaser) do.
package version

import (
	"runtime/debug"

	componentversion "k8s.io/component-base/version"
)

// placeholder is k8s.io/component-base/version's compiled-in default,
// present whenever no ldflags have set a real GitVersion.
const placeholder = "v0.0.0-master+$Format:%H$"

// ApplyFallback installs a fallback GitVersion derived from the binary's
// embedded VCS info, unless ldflags have already set a real one or no VCS
// info was stamped into the binary (e.g. building outside a git checkout).
func ApplyFallback() {
	if componentversion.Get().GitVersion != placeholder {
		return
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	if fallback, ok := fallbackFromBuildInfo(info); ok {
		_ = componentversion.SetDynamicVersion(fallback)
	}
}

func fallbackFromBuildInfo(info *debug.BuildInfo) (string, bool) {
	var revision string
	var dirty bool
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			dirty = setting.Value == "true"
		}
	}
	if revision == "" {
		return "", false
	}
	if len(revision) > 8 {
		revision = revision[:8]
	}

	fallback := "v0.0.0+" + revision
	if dirty {
		fallback += "-dev"
	}
	return fallback, true
}
