package plugindispatch

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/client"
	"go.datum.net/datumctl/internal/keyring"
)

// buildMinimalFactory creates a DatumCloudFactory suitable for unit tests.
// It uses NewDatumFactory but in an isolated home/config environment to avoid
// picking up the developer's real credentials.
func buildMinimalFactory(t *testing.T) *client.DatumCloudFactory {
	t.Helper()

	// Point HOME to a temp dir so LoadAuto returns an empty config rather
	// than reading real credentials.
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome) // Windows compat

	f, err := client.NewDatumFactory(context.Background())
	if err != nil {
		t.Fatalf("NewDatumFactory: %v", err)
	}
	return f
}

// writeFakeBinary writes an empty executable file at dir/name and returns its path.
func writeFakeBinary(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write fake binary %s: %v", path, err)
	}
	return path
}

// TestFindPlugin_managedWins verifies that a binary in the managed dir is
// returned (as managed=true) even when a same-named binary exists on PATH.
func TestFindPlugin_managedWins(t *testing.T) {
	// Not parallel — uses t.Setenv.
	managedDir := t.TempDir()
	pathDir := t.TempDir()

	writeFakeBinary(t, managedDir, "datumctl-dns")
	writeFakeBinary(t, pathDir, "datumctl-dns")

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", pathDir+string(os.PathListSeparator)+origPath)

	gotPath, managed, err := FindPlugin("dns", managedDir)
	if err != nil {
		t.Fatalf("FindPlugin: %v", err)
	}
	if !managed {
		t.Errorf("FindPlugin: managed=false, want true — managed dir binary should win")
	}
	wantPath := filepath.Join(managedDir, "datumctl-dns")
	wantAbs, _ := filepath.Abs(wantPath)
	gotAbs, _ := filepath.Abs(gotPath)
	if gotAbs != wantAbs {
		t.Errorf("FindPlugin path: got %q, want %q", gotAbs, wantAbs)
	}
}

// TestFindPlugin_fallbackToPath verifies that when the binary is not in the
// managed dir, FindPlugin falls back to PATH and returns managed=false.
func TestFindPlugin_fallbackToPath(t *testing.T) {
	// Not parallel — uses t.Setenv.
	managedDir := t.TempDir()
	pathDir := t.TempDir()

	writeFakeBinary(t, pathDir, "datumctl-dns")

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", pathDir+string(os.PathListSeparator)+origPath)

	gotPath, managed, err := FindPlugin("dns", managedDir)
	if err != nil {
		t.Fatalf("FindPlugin: %v", err)
	}
	if managed {
		t.Errorf("FindPlugin: managed=true, want false — binary is on PATH only")
	}
	if !strings.Contains(gotPath, "datumctl-dns") {
		t.Errorf("FindPlugin path %q does not contain 'datumctl-dns'", gotPath)
	}
}

// TestFindPlugin_noneFound verifies that FindPlugin returns an error when the
// binary is in neither the managed dir nor PATH.
func TestFindPlugin_noneFound(t *testing.T) {
	// Not parallel — uses t.Setenv.
	managedDir := t.TempDir()

	// Use an empty temp dir as the entire PATH so the binary cannot be found.
	t.Setenv("PATH", t.TempDir())

	_, _, err := FindPlugin("no-such-plugin", managedDir)
	if err == nil {
		t.Fatal("FindPlugin: want error for missing plugin, got nil")
	}
}

// TestIsBuiltIn_builtinReturnsTrue verifies that real cobra subcommand names
// return true.
func TestIsBuiltIn_builtinReturnsTrue(t *testing.T) {
	t.Parallel()

	root := buildMinimalCobraTree()

	builtins := []string{"get", "login", "plugin", "help", "completion", "__complete"}
	for _, name := range builtins {
		if !IsBuiltIn(root, name) {
			t.Errorf("IsBuiltIn(%q): got false, want true", name)
		}
	}
}

// TestIsBuiltIn_unknownReturnsFalse verifies that unknown command names return false.
func TestIsBuiltIn_unknownReturnsFalse(t *testing.T) {
	t.Parallel()

	root := buildMinimalCobraTree()

	unknown := []string{"dns", "myplugin", "foobar", "xyz"}
	for _, name := range unknown {
		if IsBuiltIn(root, name) {
			t.Errorf("IsBuiltIn(%q): got true, want false", name)
		}
	}
}

// buildMinimalCobraTree returns a minimal cobra root command that mirrors the
// real datumctl command tree just enough to test IsBuiltIn.
func buildMinimalCobraTree() *cobra.Command {
	root := &cobra.Command{Use: "datumctl"}
	root.AddCommand(
		&cobra.Command{Use: "get"},
		&cobra.Command{Use: "login"},
		&cobra.Command{Use: "plugin"},
	)
	return root
}

// TestBuildEnv_absoluteCredentialsHelper verifies that DATUM_CREDENTIALS_HELPER
// is an absolute path (not a bare "datumctl" basename).
func TestBuildEnv_absoluteCredentialsHelper(t *testing.T) {
	// Not parallel — uses t.Setenv via buildMinimalFactory.
	factory := buildMinimalFactory(t)

	env, err := BuildEnv(factory)
	if err != nil {
		t.Fatalf("BuildEnv: %v", err)
	}

	helper := envValue(env, "DATUM_CREDENTIALS_HELPER")
	if helper == "" {
		t.Fatal("DATUM_CREDENTIALS_HELPER not found in env")
	}
	if !filepath.IsAbs(helper) {
		t.Errorf("DATUM_CREDENTIALS_HELPER=%q is not an absolute path", helper)
	}
}

// TestBuildEnv_sessionPropagated verifies that DATUM_SESSION reflects the
// active session name from the datumctl config.
func TestBuildEnv_sessionPropagated(t *testing.T) {
	// Not parallel — writes a config file to a temp HOME.
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	cfgDir := filepath.Join(tmpHome, ".datumctl")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir .datumctl: %v", err)
	}
	cfgContent := "kind: DatumctlConfig\n" +
		"active-session: my-session\n" +
		"sessions:\n" +
		"  - name: my-session\n" +
		"    user-key: user@example.com\n" +
		"    user-email: user@example.com\n" +
		"    endpoint:\n" +
		"      server: https://api.example.com\n" +
		"      auth-hostname: auth.example.com\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "config"), []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	f, err := client.NewDatumFactory(context.Background())
	if err != nil {
		t.Fatalf("NewDatumFactory: %v", err)
	}

	env, err := BuildEnv(f)
	if err != nil {
		t.Fatalf("BuildEnv: %v", err)
	}

	session := envValue(env, "DATUM_SESSION")
	if session != "my-session" {
		t.Errorf("DATUM_SESSION=%q, want %q", session, "my-session")
	}
	apiHost := envValue(env, "DATUM_API_HOST")
	if apiHost != "api.example.com" {
		t.Errorf("DATUM_API_HOST=%q, want %q", apiHost, "api.example.com")
	}
}

// TestBuildEnv_FollowsSwitchedAccountNotLegacyKeyring reproduces
// datumctl/datumctl#292: `auth switch` updates the config's ActiveSession, but
// never the legacy keyring `active_user` marker (still set from whichever
// account logged in last). BuildEnv must pair DATUM_SESSION and
// DATUM_API_HOST from the switched-to session, never a stale value pulled
// from that legacy marker.
func TestBuildEnv_FollowsSwitchedAccountNotLegacyKeyring(t *testing.T) {
	// Not parallel — writes a config file and legacy keyring entry to a temp HOME.
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)
	keyring.MockInit()

	// Simulate the account that logged in last, before the switch — this is
	// what the legacy keyring marker names, and what a call site reading it
	// directly would still use.
	if err := keyring.Set(authutil.ServiceName, authutil.ActiveUserKey, "old@example.com@auth.datum.net"); err != nil {
		t.Fatalf("seed legacy active_user: %v", err)
	}

	cfgDir := filepath.Join(tmpHome, ".datumctl")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir .datumctl: %v", err)
	}
	// `auth switch` moved ActiveSession to the staging account and cleared
	// CurrentContext (no LastContext for the newly-selected session), exactly
	// as internal/cmd/auth/switch.go does.
	cfgContent := "kind: DatumctlConfig\n" +
		"active-session: new@example.com@api.staging.env.datum.net\n" +
		"sessions:\n" +
		"  - name: old@example.com@api.datum.net\n" +
		"    user-key: old@example.com@auth.datum.net\n" +
		"    user-email: old@example.com\n" +
		"    endpoint:\n" +
		"      server: https://api.datum.net\n" +
		"      auth-hostname: auth.datum.net\n" +
		"  - name: new@example.com@api.staging.env.datum.net\n" +
		"    user-key: new@example.com@auth.staging.env.datum.net\n" +
		"    user-email: new@example.com\n" +
		"    endpoint:\n" +
		"      server: https://api.staging.env.datum.net\n" +
		"      auth-hostname: auth.staging.env.datum.net\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "config"), []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	f, err := client.NewDatumFactory(context.Background())
	if err != nil {
		t.Fatalf("NewDatumFactory: %v", err)
	}

	env, err := BuildEnv(f)
	if err != nil {
		t.Fatalf("BuildEnv: %v", err)
	}

	if got := envValue(env, "DATUM_SESSION"); got != "new@example.com@api.staging.env.datum.net" {
		t.Errorf("DATUM_SESSION=%q, want the switched-to session", got)
	}
	if got := envValue(env, "DATUM_API_HOST"); got != "api.staging.env.datum.net" {
		t.Errorf("DATUM_API_HOST=%q, want %q (the switched-to account's host, not the old account's)", got, "api.staging.env.datum.net")
	}
}

// TestBuildEnv_sessionEmptyWhenNone verifies that DATUM_SESSION is present in
// the env slice but set to "" when no active session is configured.
func TestBuildEnv_sessionEmptyWhenNone(t *testing.T) {
	// Not parallel — uses t.Setenv via buildMinimalFactory.
	factory := buildMinimalFactory(t)

	env, err := BuildEnv(factory)
	if err != nil {
		t.Fatalf("BuildEnv: %v", err)
	}

	found := false
	for _, kv := range env {
		if strings.HasPrefix(kv, "DATUM_SESSION=") {
			found = true
			val := strings.TrimPrefix(kv, "DATUM_SESSION=")
			if val != "" {
				t.Errorf("DATUM_SESSION=%q, want empty string when no session configured", val)
			}
		}
	}
	if !found {
		t.Error("DATUM_SESSION key not found in BuildEnv output")
	}
}

// envValue extracts the value of key from a []string "KEY=VALUE" slice.
// Returns "" if not found.
func envValue(env []string, key string) string {
	prefix := key + "="
	for _, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			return strings.TrimPrefix(kv, prefix)
		}
	}
	return ""
}
