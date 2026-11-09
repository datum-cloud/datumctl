// Package version resolves datumctl's own version string.
//
// Release, nix, and goreleaser builds inject a version into
// k8s.io/component-base/version via ldflags. Any other build (go build,
// go run, go install …@main) gets no ldflags, but the Go toolchain stamps
// a module pseudo-version derived from the nearest tag into the binary
// (e.g. "v0.23.4-0.20261103201002-818025e9fe34"). That pseudo-version is
// what Get returns in that case: it is based on the last release, carries
// the commit, and sorts after that release and before the next one, so it
// can never be mistaken for a release.
//
// component-base can't carry that string itself: its SetDynamicVersion
// rejects any version whose major/minor/patch differ from the compiled-in
// placeholder's (0.0.0). Code that needs datumctl's version must call Get
// instead of reading component-base directly.
package version

import (
	"runtime/debug"
	"strings"

	"golang.org/x/mod/semver"
	apimachineryversion "k8s.io/apimachinery/pkg/version"
	componentversion "k8s.io/component-base/version"
)

// placeholder is k8s.io/component-base/version's compiled-in default,
// present whenever no ldflags have set a real GitVersion.
const placeholder = "v0.0.0-master+$Format:%H$"

// component-base's compiled-in defaults for the other Info fields.
const (
	commitPlaceholder    = "$Format:%H$"
	buildDatePlaceholder = "1970-01-01T00:00:00Z"
)

// resolved is computed at package init, before ApplyFallback can replace
// component-base's placeholder, so Get can tell whether ldflags set it.
var resolved = resolve(componentversion.Get().GitVersion, readBuildInfo())

func readBuildInfo() *debug.BuildInfo {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return nil
	}
	return info
}

// Get returns datumctl's version: the ldflags-injected version if any,
// otherwise the Go module pseudo-version stamped by the toolchain,
// otherwise "v0.0.0+<sha>[-dev]".
func Get() string {
	return resolved
}

// Info returns component-base's version info with GitVersion set to Get
// and any field ldflags left at its compiled-in default filled from the
// VCS info the Go toolchain stamps into the binary. BuildDate gets the
// commit time, as reproducible builds (SOURCE_DATE_EPOCH) do.
func Info() apimachineryversion.Info {
	return completeInfo(componentversion.Get(), resolved, readBuildInfo())
}

func completeInfo(info apimachineryversion.Info, gitVersion string, bi *debug.BuildInfo) apimachineryversion.Info {
	info.GitVersion = gitVersion
	if semver.IsValid(gitVersion) {
		mm := strings.SplitN(strings.TrimPrefix(semver.MajorMinor(gitVersion), "v"), ".", 2)
		if info.Major == "" {
			info.Major = mm[0]
		}
		if info.Minor == "" {
			info.Minor = mm[1]
		}
	}
	if bi == nil {
		return info
	}
	for _, setting := range bi.Settings {
		switch setting.Key {
		case "vcs.revision":
			if info.GitCommit == commitPlaceholder || info.GitCommit == "" {
				info.GitCommit = setting.Value
			}
		case "vcs.time":
			if info.BuildDate == buildDatePlaceholder || info.BuildDate == "" {
				info.BuildDate = setting.Value
			}
		case "vcs.modified":
			if info.GitTreeState == "" {
				info.GitTreeState = "clean"
				if setting.Value == "true" {
					info.GitTreeState = "dirty"
				}
			}
		}
	}
	return info
}

// ApplyFallback replaces component-base's unparseable placeholder with a
// valid "v0.0.0+<sha>[-dev]" so kubectl internals that read it (e.g. the
// User-Agent) see a well-formed version. It can't install the value Get
// returns; see the package doc.
func ApplyFallback() {
	if componentversion.Get().GitVersion != placeholder {
		return
	}
	info := readBuildInfo()
	if info == nil {
		return
	}
	if fallback, ok := fallbackFromBuildInfo(info); ok {
		_ = componentversion.SetDynamicVersion(fallback)
	}
}

func resolve(ldflagsVersion string, info *debug.BuildInfo) string {
	if ldflagsVersion != placeholder {
		return ldflagsVersion
	}
	if info == nil {
		return ldflagsVersion
	}
	if v, ok := moduleVersion(info); ok {
		return v
	}
	if v, ok := fallbackFromBuildInfo(info); ok {
		return v
	}
	return ldflagsVersion
}

// moduleVersion returns the main module version the Go toolchain stamped
// into the binary, with Go's "+dirty" marker rewritten to "+dev". It
// reports false for "(devel)", which Go uses when there's no VCS info.
func moduleVersion(info *debug.BuildInfo) (string, bool) {
	v := info.Main.Version
	if v == "" || v == "(devel)" {
		return "", false
	}
	if base, ok := strings.CutSuffix(v, "+dirty"); ok {
		v = base + "+dev"
	}
	return v, true
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
