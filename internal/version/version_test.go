package version

import (
	"runtime/debug"
	"testing"
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
