package plugindispatch

import (
	"context"
	"slices"
	"strings"
	"testing"

	"go.datum.net/datumctl/internal/client"
	"go.datum.net/datumctl/internal/datumconfig"
	customerrors "go.datum.net/datumctl/internal/errors"
	"go.datum.net/datumctl/internal/keyring"
)

const (
	ovProd    = "swells@datum.net@api.datum.net"
	ovStaging = "swells@datum.net@api.staging.env.datum.net"
	ovSolo    = "solo@example.com@api.datum.net"
)

// setupPluginOverrideEnv writes a config where prod is active and staging
// remembers a project context.
func setupPluginOverrideEnv(t *testing.T) *client.DatumCloudFactory {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv(datumconfig.SessionEnvVar, "")
	t.Setenv("DATUM_PROJECT", "")
	t.Setenv("DATUM_ORGANIZATION", "")
	keyring.MockInit()
	t.Cleanup(datumconfig.ClearSessionOverride)

	prodCtx := datumconfig.QualifiedContextName(ovProd, "org-prod")
	stagingCtx := datumconfig.QualifiedContextName(ovStaging, "org-staging/proj-staging")
	cfg := datumconfig.NewV1Beta1()
	cfg.Sessions = []datumconfig.Session{
		{Name: ovProd, UserKey: "key-prod", UserEmail: "swells@datum.net",
			Endpoint: datumconfig.Endpoint{Server: "https://api.datum.net"}, LastContext: prodCtx},
		{Name: ovStaging, UserKey: "key-staging", UserEmail: "swells@datum.net",
			Endpoint: datumconfig.Endpoint{Server: "https://api.staging.env.datum.net"}, LastContext: stagingCtx},
		{Name: ovSolo, UserKey: "key-solo", UserEmail: "solo@example.com",
			Endpoint: datumconfig.Endpoint{Server: "https://api.datum.net"}},
	}
	cfg.Contexts = []datumconfig.DiscoveredContext{
		{Name: prodCtx, Session: ovProd, OrganizationID: "org-prod"},
		{Name: stagingCtx, Session: ovStaging, OrganizationID: "org-staging", ProjectID: "proj-staging"},
	}
	cfg.ActiveSession = ovProd
	cfg.CurrentContext = prodCtx
	if err := datumconfig.SaveV1Beta1(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	f, err := client.NewDatumFactory(context.Background())
	if err != nil {
		t.Fatalf("NewDatumFactory: %v", err)
	}
	return f
}

// A plugin run with a session override receives that session's name, API
// host, and scope, so a datumctl it calls back into acts as the same session.
func TestBuildEnv_SessionOverride(t *testing.T) {
	tests := []struct {
		name        string
		flag        string
		hasFlag     bool
		env         string
		project     string
		wantSession string
		wantHost    string
		wantOrg     string
		wantProject string
	}{
		{name: "no override", wantSession: ovProd, wantHost: "api.datum.net", wantOrg: "org-prod"},
		{name: "flag by name", flag: ovStaging, hasFlag: true,
			wantSession: ovStaging, wantHost: "api.staging.env.datum.net", wantProject: "proj-staging"},
		{name: "flag by unique email, no context", flag: "solo@example.com", hasFlag: true,
			wantSession: ovSolo, wantHost: "api.datum.net"},
		{name: "env var", env: ovStaging,
			wantSession: ovStaging, wantHost: "api.staging.env.datum.net", wantProject: "proj-staging"},
		{name: "flag beats env var", flag: ovSolo, hasFlag: true, env: ovStaging,
			wantSession: ovSolo, wantHost: "api.datum.net"},
		{name: "--project beats the override's context", flag: ovStaging, hasFlag: true, project: "other-proj",
			wantSession: ovStaging, wantHost: "api.staging.env.datum.net", wantProject: "other-proj"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := setupPluginOverrideEnv(t)
			t.Setenv(datumconfig.SessionEnvVar, tt.env)
			*f.ConfigFlags.Project = tt.project

			if err := applyPluginSessionOverride(tt.flag, tt.hasFlag); err != nil {
				t.Fatalf("apply override: %v", err)
			}
			env, err := BuildEnv(f)
			if err != nil {
				t.Fatalf("BuildEnv: %v", err)
			}
			for key, want := range map[string]string{
				"DATUM_SESSION":  tt.wantSession,
				"DATUM_API_HOST": tt.wantHost,
				"DATUM_ORG":      tt.wantOrg,
				"DATUM_PROJECT":  tt.wantProject,
			} {
				if got := envValue(env, key); got != want {
					t.Errorf("%s = %q, want %q", key, got, want)
				}
			}
		})
	}
}

func TestApplyPluginSessionOverride_Errors(t *testing.T) {
	tests := []struct {
		name    string
		flag    string
		env     string
		hasFlag bool
		want    []string
	}{
		{name: "shared email", flag: "swells@datum.net", hasFlag: true,
			want: []string{"ambiguous", "--session " + ovProd, "--session " + ovStaging}},
		{name: "unknown env value", env: "nobody@example.com",
			want: []string{"No session matches DATUM_SESSION nobody@example.com.", ovSolo}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupPluginOverrideEnv(t)
			t.Setenv(datumconfig.SessionEnvVar, tt.env)
			err := applyPluginSessionOverride(tt.flag, tt.hasFlag)
			if err == nil {
				t.Fatal("expected an error")
			}
			if _, ok := customerrors.IsUserError(err); !ok {
				t.Errorf("error is not a UserError: %v", err)
			}
			// Error() carries the message and the hint the user sees.
			for _, w := range tt.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error missing %q:\n%s", w, err)
				}
			}
			if datumconfig.HasSessionOverride() {
				t.Error("a failed override must not leave one installed")
			}
		})
	}
}

func TestSplitLeadingSessionFlag(t *testing.T) {
	tests := []struct {
		in        []string
		wantValue string
		wantRest  []string
		wantFound bool
	}{
		{[]string{"ipam", "list"}, "", []string{"ipam", "list"}, false},
		{[]string{"--session", "a@b@c", "ipam", "list", "-o", "wide"}, "a@b@c", []string{"ipam", "list", "-o", "wide"}, true},
		{[]string{"--session=a@b", "ipam"}, "a@b", []string{"ipam"}, true},
		// A --session after the plugin name belongs to the plugin.
		{[]string{"ipam", "--session", "x"}, "", []string{"ipam", "--session", "x"}, false},
		{[]string{"--project", "p", "ipam"}, "", []string{"--project", "p", "ipam"}, false},
	}
	for _, tt := range tests {
		v, rest, found := splitLeadingSessionFlag(tt.in)
		if v != tt.wantValue || found != tt.wantFound || !slices.Equal(rest, tt.wantRest) {
			t.Errorf("splitLeadingSessionFlag(%q) = %q, %q, %v; want %q, %q, %v",
				tt.in, v, rest, found, tt.wantValue, tt.wantRest, tt.wantFound)
		}
	}
}
