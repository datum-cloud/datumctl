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
