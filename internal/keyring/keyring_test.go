package keyring

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

const (
	testService = "datumctl-test"
	testUser    = "alice@example.com"
	testSecret  = "shhh"
)

// withTempHome points os.UserHomeDir() at a temporary directory and resets
// fallback state so each test starts from a clean slate.
func withTempHome(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))
	resetFallback()
	t.Cleanup(resetFallback)
	return tmp
}

func TestKeyringWorks_NoFallback(t *testing.T) {
	tmp := withTempHome(t)
	MockInit()

	if err := Set(testService, testUser, testSecret); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := Get(testService, testUser)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != testSecret {
		t.Fatalf("Get = %q, want %q", got, testSecret)
	}

	// File fallback should not have been triggered, so no credentials file.
	if _, err := os.Stat(filepath.Join(tmp, ".datumctl", "credentials.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no credentials file, got stat err = %v", err)
	}
}

func TestKeyringFails_FallsBackToFile(t *testing.T) {
	tmp := withTempHome(t)
	MockInitWithError(errors.New("dbus unavailable"))

	if err := Set(testService, testUser, testSecret); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Subsequent reads should succeed via the file backend.
	got, err := Get(testService, testUser)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != testSecret {
		t.Fatalf("Get = %q, want %q", got, testSecret)
	}

	// Credentials file should exist with restricted permissions.
	path := filepath.Join(tmp, ".datumctl", "credentials.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat credentials file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("credentials file perm = %o, want 0600", perm)
	}
}

func TestDelete_ViaFileFallback(t *testing.T) {
	withTempHome(t)
	MockInitWithError(errors.New("dbus unavailable"))

	if err := Set(testService, testUser, testSecret); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := Delete(testService, testUser); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := Get(testService, testUser); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get after Delete err = %v, want ErrNotFound", err)
	}
}

func TestGet_NotFound_DoesNotFallBack(t *testing.T) {
	tmp := withTempHome(t)
	MockInit()

	if _, err := Get(testService, testUser); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get err = %v, want ErrNotFound", err)
	}
	// ErrNotFound from a working keyring must not create a file.
	if _, err := os.Stat(filepath.Join(tmp, ".datumctl", "credentials.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("fallback file unexpectedly created on ErrNotFound: stat err = %v", err)
	}
}

func TestExistingFile_TriggersFallbackOnStartup(t *testing.T) {
	tmp := withTempHome(t)

	// Simulate a credentials file from a previous run that fell back.
	dir := filepath.Join(tmp, ".datumctl")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(dir, "credentials.json")
	contents := []byte(`{"` + testService + `":{"` + testUser + `":"` + testSecret + `"}}`)
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// Even though the mock keyring works, the existing file should be used.
	MockInit()

	got, err := Get(testService, testUser)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != testSecret {
		t.Fatalf("Get = %q, want %q", got, testSecret)
	}
}
