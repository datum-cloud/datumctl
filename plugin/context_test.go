package plugin

import (
	"testing"
)

// TestContext_readsAllEnvVars verifies that Context() reads all DATUM_* variables
// and reflects them in the returned PluginContext.
func TestContext_readsAllEnvVars(t *testing.T) {
	// Not parallel — uses t.Setenv.
	t.Setenv("DATUM_ORG", "test-org")
	t.Setenv("DATUM_PROJECT", "test-project")
	t.Setenv("DATUM_API_HOST", "api.test.datum.net")
	t.Setenv("DATUM_PLUGIN_API_VERSION", "3")
	t.Setenv("DATUM_CREDENTIALS_HELPER", "/usr/local/bin/datumctl")
	t.Setenv("DATUM_SESSION", "prod")

	ctx := Context()

	if ctx.Org != "test-org" {
		t.Errorf("Org = %q, want %q", ctx.Org, "test-org")
	}
	if ctx.Project != "test-project" {
		t.Errorf("Project = %q, want %q", ctx.Project, "test-project")
	}
	if ctx.APIHost != "api.test.datum.net" {
		t.Errorf("APIHost = %q, want %q", ctx.APIHost, "api.test.datum.net")
	}
	if ctx.PluginAPIVersion != 3 {
		t.Errorf("PluginAPIVersion = %d, want 3", ctx.PluginAPIVersion)
	}
	if ctx.CredentialsHelper != "/usr/local/bin/datumctl" {
		t.Errorf("CredentialsHelper = %q, want %q", ctx.CredentialsHelper, "/usr/local/bin/datumctl")
	}
	if ctx.Session != "prod" {
		t.Errorf("Session = %q, want %q", ctx.Session, "prod")
	}
}

// TestContext_missingVarsReturnEmpty verifies that Context() returns zero values
// when no DATUM_* variables are set, and does not panic.
func TestContext_missingVarsReturnEmpty(t *testing.T) {
	// Not parallel — clears env vars.
	t.Setenv("DATUM_ORG", "")
	t.Setenv("DATUM_PROJECT", "")
	t.Setenv("DATUM_API_HOST", "")
	t.Setenv("DATUM_PLUGIN_API_VERSION", "")
	t.Setenv("DATUM_CREDENTIALS_HELPER", "")
	t.Setenv("DATUM_SESSION", "")

	ctx := Context()

	if ctx.Org != "" {
		t.Errorf("Org = %q, want empty", ctx.Org)
	}
	if ctx.Project != "" {
		t.Errorf("Project = %q, want empty", ctx.Project)
	}
	if ctx.APIHost != "" {
		t.Errorf("APIHost = %q, want empty", ctx.APIHost)
	}
	if ctx.PluginAPIVersion != 0 {
		t.Errorf("PluginAPIVersion = %d, want 0", ctx.PluginAPIVersion)
	}
	if ctx.CredentialsHelper != "" {
		t.Errorf("CredentialsHelper = %q, want empty", ctx.CredentialsHelper)
	}
	if ctx.Session != "" {
		t.Errorf("Session = %q, want empty", ctx.Session)
	}
}

// TestContext_apiVersionParseError verifies that a non-numeric
// DATUM_PLUGIN_API_VERSION produces PluginAPIVersion == 0 (not a panic).
func TestContext_apiVersionParseError(t *testing.T) {
	// Not parallel — uses t.Setenv.
	t.Setenv("DATUM_PLUGIN_API_VERSION", "not-a-number")

	ctx := Context()

	if ctx.PluginAPIVersion != 0 {
		t.Errorf("PluginAPIVersion = %d, want 0 for non-numeric input", ctx.PluginAPIVersion)
	}
}

// TestPluginContext_ControlPlaneURL verifies scope selection (project wins over
// org), API host scheme normalization, and the error cases for an incomplete
// context.
func TestPluginContext_ControlPlaneURL(t *testing.T) {
	t.Parallel()

	const (
		projectPrefix = "/apis/resourcemanager.miloapis.com/v1alpha1/projects/"
		orgPrefix     = "/apis/resourcemanager.miloapis.com/v1alpha1/organizations/"
	)

	tests := []struct {
		name    string
		ctx     PluginContext
		want    string
		wantErr bool
	}{
		{
			name: "project wins over org",
			ctx:  PluginContext{APIHost: "api.datum.net", Org: "my-org", Project: "my-project"},
			want: "https://api.datum.net" + projectPrefix + "my-project/control-plane",
		},
		{
			name: "org only",
			ctx:  PluginContext{APIHost: "api.datum.net", Org: "my-org"},
			want: "https://api.datum.net" + orgPrefix + "my-org/control-plane",
		},
		{
			name: "https scheme already present",
			ctx:  PluginContext{APIHost: "https://api.datum.net", Project: "my-project"},
			want: "https://api.datum.net" + projectPrefix + "my-project/control-plane",
		},
		{
			name: "http scheme preserved",
			ctx:  PluginContext{APIHost: "http://localhost:8080", Project: "my-project"},
			want: "http://localhost:8080" + projectPrefix + "my-project/control-plane",
		},
		{
			name: "trailing slash trimmed",
			ctx:  PluginContext{APIHost: "https://api.datum.net/", Org: "my-org"},
			want: "https://api.datum.net" + orgPrefix + "my-org/control-plane",
		},
		{
			name:    "missing api host",
			ctx:     PluginContext{Org: "my-org", Project: "my-project"},
			wantErr: true,
		},
		{
			name:    "missing org and project",
			ctx:     PluginContext{APIHost: "api.datum.net"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.ctx.ControlPlaneURL()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ControlPlaneURL() = %q, want error", got)
				}
				if got != "" {
					t.Errorf("ControlPlaneURL() = %q on error, want empty", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ControlPlaneURL() error = %v, want nil", err)
			}
			if got != tt.want {
				t.Errorf("ControlPlaneURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestControlPlaneURL_packageLevel verifies that the package-level
// ControlPlaneURL() reads the injected environment.
func TestControlPlaneURL_packageLevel(t *testing.T) {
	// Not parallel — uses t.Setenv.
	t.Setenv("DATUM_API_HOST", "api.test.datum.net")
	t.Setenv("DATUM_ORG", "test-org")
	t.Setenv("DATUM_PROJECT", "test-project")

	got, err := ControlPlaneURL()
	if err != nil {
		t.Fatalf("ControlPlaneURL() error = %v, want nil", err)
	}
	want := "https://api.test.datum.net/apis/resourcemanager.miloapis.com/v1alpha1/projects/test-project/control-plane"
	if got != want {
		t.Errorf("ControlPlaneURL() = %q, want %q", got, want)
	}
}
