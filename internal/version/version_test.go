package version

import (
	"runtime/debug"
	"testing"

	apimachineryversion "k8s.io/apimachinery/pkg/version"
)

func TestFallbackFromBuildInfo(t *testing.T) {
	cases := []struct {
		name     string
		settings []debug.BuildSetting
		want     string
		wantOK   bool
	}{
		{
			name: "clean checkout",
			settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "931f022eb016a57fcc8ee63e328e9ba12ded6aba"},
				{Key: "vcs.modified", Value: "false"},
			},
			want:   "v0.0.0+931f022e",
			wantOK: true,
		},
		{
			name: "dirty checkout",
			settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "931f022eb016a57fcc8ee63e328e9ba12ded6aba"},
				{Key: "vcs.modified", Value: "true"},
			},
			want:   "v0.0.0+931f022e-dev",
			wantOK: true,
		},
		{
			name: "short revision left untruncated",
			settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "abc123"},
			},
			want:   "v0.0.0+abc123",
			wantOK: true,
		},
		{
			name:     "no vcs info",
			settings: nil,
			wantOK:   false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := fallbackFromBuildInfo(&debug.BuildInfo{Settings: tc.settings})
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if got != tc.want {
				t.Fatalf("fallback = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	sha := []debug.BuildSetting{{Key: "vcs.revision", Value: "818025e9fe34abcdef0123456789abcdef012345"}}
	cases := []struct {
		name    string
		ldflags string
		info    *debug.BuildInfo
		want    string
	}{
		{
			name:    "ldflags win",
			ldflags: "v0.23.4-0.20261103201002-818025e9fe34",
			info:    &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}},
			want:    "v0.23.4-0.20261103201002-818025e9fe34",
		},
		{
			name:    "pseudo-version from toolchain",
			ldflags: placeholder,
			info:    &debug.BuildInfo{Main: debug.Module{Version: "v0.23.4-0.20261103201002-818025e9fe34"}, Settings: sha},
			want:    "v0.23.4-0.20261103201002-818025e9fe34",
		},
		{
			name:    "dirty pseudo-version marked dev",
			ldflags: placeholder,
			info:    &debug.BuildInfo{Main: debug.Module{Version: "v0.23.4-0.20261103201002-818025e9fe34+dirty"}, Settings: sha},
			want:    "v0.23.4-0.20261103201002-818025e9fe34+dev",
		},
		{
			name:    "tagged release build",
			ldflags: placeholder,
			info:    &debug.BuildInfo{Main: debug.Module{Version: "v0.23.3"}, Settings: sha},
			want:    "v0.23.3",
		},
		{
			name:    "devel falls back to sha",
			ldflags: placeholder,
			info:    &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}, Settings: sha},
			want:    "v0.0.0+818025e9",
		},
		{
			name:    "no build info",
			ldflags: placeholder,
			info:    nil,
			want:    placeholder,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolve(tc.ldflags, tc.info); got != tc.want {
				t.Fatalf("resolve = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCompleteInfo(t *testing.T) {
	defaults := apimachineryversion.Info{GitCommit: commitPlaceholder, BuildDate: buildDatePlaceholder, GoVersion: "go1.27.3"}
	vcs := &debug.BuildInfo{Settings: []debug.BuildSetting{
		{Key: "vcs.revision", Value: "818025e9fe34abcdef0123456789abcdef012345"},
		{Key: "vcs.time", Value: "2026-11-03T20:10:02Z"},
		{Key: "vcs.modified", Value: "true"},
	}}

	t.Run("fills defaults from vcs info", func(t *testing.T) {
		got := completeInfo(defaults, "v0.23.4-0.20261103201002-818025e9fe34+dev", vcs)
		want := apimachineryversion.Info{
			Major: "0", Minor: "23",
			GitVersion:   "v0.23.4-0.20261103201002-818025e9fe34+dev",
			GitCommit:    "818025e9fe34abcdef0123456789abcdef012345",
			GitTreeState: "dirty",
			BuildDate:    "2026-11-03T20:10:02Z",
			GoVersion:    "go1.27.3",
		}
		if got != want {
			t.Fatalf("got %+v\nwant %+v", got, want)
		}
	})

	t.Run("keeps ldflags values", func(t *testing.T) {
		in := apimachineryversion.Info{Major: "9", Minor: "8", GitCommit: "abc", GitTreeState: "clean", BuildDate: "2026-01-01T00:00:00Z"}
		got := completeInfo(in, "v0.23.3", vcs)
		in.GitVersion = "v0.23.3"
		if got != in {
			t.Fatalf("got %+v\nwant %+v", got, in)
		}
	})

	t.Run("clean tree without build info", func(t *testing.T) {
		got := completeInfo(defaults, "v0.23.3", nil)
		if got.Major != "0" || got.Minor != "23" || got.GitCommit != commitPlaceholder {
			t.Fatalf("got %+v", got)
		}
	})
}
