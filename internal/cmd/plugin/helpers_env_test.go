package plugin

import (
	"runtime"
	"strings"
	"testing"
)

// envHas reports whether the KEY=VALUE slice contains an entry for name.
func envHas(env []string, name string) bool {
	prefix := name + "="
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			return true
		}
	}
	return false
}

// TestMinimalManifestEnv_includesEssentials verifies that the scrubbed
// environment still forwards the variables a plugin binary plausibly needs to
// start up and emit its manifest: PATH, a home directory, and the platform
// temp-dir hint.
func TestMinimalManifestEnv_includesEssentials(t *testing.T) {
	t.Setenv("PATH", "/usr/bin")
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", `C:\Users\test`)
		t.Setenv("TEMP", `C:\Temp`)
	} else {
		t.Setenv("HOME", "/home/test")
		t.Setenv("TMPDIR", "/tmp")
	}

	env := minimalManifestEnv()

	if !envHas(env, "PATH") {
		t.Errorf("minimalManifestEnv() missing PATH; got %v", env)
	}

	if runtime.GOOS == "windows" {
		if !envHas(env, "USERPROFILE") {
			t.Errorf("minimalManifestEnv() missing USERPROFILE on windows; got %v", env)
		}
		if !envHas(env, "TEMP") {
			t.Errorf("minimalManifestEnv() missing TEMP on windows; got %v", env)
		}
	} else {
		if !envHas(env, "HOME") {
			t.Errorf("minimalManifestEnv() missing HOME; got %v", env)
		}
		if !envHas(env, "TMPDIR") {
			t.Errorf("minimalManifestEnv() missing TMPDIR; got %v", env)
		}
	}
}

// TestMinimalManifestEnv_excludesSensitive verifies that credential and token
// variables present in the host environment are not forwarded to the plugin
// binary.
func TestMinimalManifestEnv_excludesSensitive(t *testing.T) {
	t.Setenv("DATUM_CREDENTIALS_HELPER", "secret-helper")
	t.Setenv("DATUM_TOKEN", "super-secret-token")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "fake-aws-secret")

	env := minimalManifestEnv()

	for _, name := range []string{"DATUM_CREDENTIALS_HELPER", "DATUM_TOKEN", "AWS_SECRET_ACCESS_KEY"} {
		if envHas(env, name) {
			t.Errorf("minimalManifestEnv() leaked sensitive var %s; got %v", name, env)
		}
	}
}

// TestMinimalManifestEnv_isAllowList verifies the environment is built as an
// allow-list: an arbitrary unrelated variable set in the host environment does
// not appear in the result.
func TestMinimalManifestEnv_isAllowList(t *testing.T) {
	t.Setenv("FOO", "bar")
	t.Setenv("SOME_UNRELATED_VAR", "value")

	env := minimalManifestEnv()

	if envHas(env, "FOO") {
		t.Errorf("minimalManifestEnv() forwarded unrelated var FOO; got %v", env)
	}
	if envHas(env, "SOME_UNRELATED_VAR") {
		t.Errorf("minimalManifestEnv() forwarded unrelated var SOME_UNRELATED_VAR; got %v", env)
	}
}
