package plugin

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTagFromLatestReleaseURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    string
		wantErr string
	}{
		{name: "stable tag", url: "https://github.com/o/r/releases/tag/v1.2.3", want: "v1.2.3"},
		{name: "pre-release tag", url: "https://github.com/o/r/releases/tag/v1.0.0-preview.28", want: "v1.0.0-preview.28"},
		{name: "build metadata", url: "https://github.com/o/r/releases/tag/v1.0.0%2Bbuild.1", want: "v1.0.0+build.1"},
		{name: "no stable release redirects to list", url: "https://github.com/o/r/releases", want: ""},
		{name: "no stable release with trailing slash", url: "https://github.com/o/r/releases/", want: ""},
		{name: "tag without v prefix", url: "https://github.com/o/r/releases/tag/1.2.3", wantErr: `"1.2.3"`},
		{name: "non-semver tag", url: "https://github.com/o/r/releases/tag/vNext", wantErr: `"vNext"`},
		{name: "unexpected page", url: "https://github.com/o/r/releases/latest", wantErr: "could not parse release tag"},
		{name: "traversal in tag", url: "https://github.com/o/r/releases/tag/..%2F..%2Fx", wantErr: "not a semantic version"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tagFromLatestReleaseURL(tt.url)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("tagFromLatestReleaseURL(%q) error = %v, want containing %q", tt.url, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("tagFromLatestReleaseURL(%q) unexpected error: %v", tt.url, err)
			}
			if got != tt.want {
				t.Fatalf("tagFromLatestReleaseURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestNewestReleaseTag(t *testing.T) {
	tests := []struct {
		name     string
		releases []githubRelease
		want     string
		wantErr  string
	}{
		{
			name: "only pre-releases picks highest",
			releases: []githubRelease{
				{TagName: "v1.0.0-preview.9", Prerelease: true},
				{TagName: "v1.0.0-preview.28", Prerelease: true},
				{TagName: "v1.0.0-preview.27", Prerelease: true},
			},
			want: "v1.0.0-preview.28",
		},
		{
			name: "drafts are skipped",
			releases: []githubRelease{
				{TagName: "v2.0.0-rc.1", Draft: true},
				{TagName: "v1.0.0-rc.1", Prerelease: true},
			},
			want: "v1.0.0-rc.1",
		},
		{
			name: "invalid tags are skipped",
			releases: []githubRelease{
				{TagName: "nightly", Prerelease: true},
				{TagName: "v0.3.0-beta", Prerelease: true},
			},
			want: "v0.3.0-beta",
		},
		{
			name:     "no releases",
			releases: nil,
			wantErr:  "has no published releases",
		},
		{
			name: "no semver tags",
			releases: []githubRelease{
				{TagName: "nightly", Prerelease: true},
				{TagName: "latest"},
			},
			wantErr: `"nightly"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newestReleaseTag("o", "r", tt.releases)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("newestReleaseTag error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("newestReleaseTag unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("newestReleaseTag = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFetchNewestReleaseTag_onlyPrereleases(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/o/r/releases" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"tag_name":"v1.0.0-preview.28","draft":false,"prerelease":true},
			{"tag_name":"v1.0.0-preview.27","draft":false,"prerelease":true}
		]`))
	}))
	t.Cleanup(srv.Close)

	origBase := githubAPIBaseURL
	githubAPIBaseURL = srv.URL
	t.Cleanup(func() { githubAPIBaseURL = origBase })

	origTransport := http.DefaultTransport
	http.DefaultTransport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // test only
	}
	t.Cleanup(func() { http.DefaultTransport = origTransport })

	got, err := fetchNewestReleaseTag(context.Background(), "o", "r")
	if err != nil {
		t.Fatalf("fetchNewestReleaseTag: %v", err)
	}
	if got != "v1.0.0-preview.28" {
		t.Fatalf("fetchNewestReleaseTag = %q, want v1.0.0-preview.28", got)
	}

	if _, err := fetchNewestReleaseTag(context.Background(), "o", "missing"); err == nil ||
		!strings.Contains(err.Error(), "o/missing@<tag>") {
		t.Fatalf("fetchNewestReleaseTag on 404 error = %v, want hint naming o/missing@<tag>", err)
	}
}
