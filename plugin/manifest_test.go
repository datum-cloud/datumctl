package plugin

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestServeManifest_noFlag verifies that ServeManifest does nothing and returns
// normally when --plugin-manifest is not present in os.Args.
//
// We test this via a subprocess that sets SERVE_MANIFEST_SUBPROCESS=no_flag so
// the helper does NOT inject --plugin-manifest into os.Args. If ServeManifest
// returns normally the subprocess exits 0.
func TestServeManifest_noFlag(t *testing.T) {
	t.Parallel()

	cmd := exec.Command(
		os.Args[0],
		"-test.run=TestServeManifestSubprocess",
		"-test.v",
	)
	cmd.Env = append(os.Environ(), "SERVE_MANIFEST_SUBPROCESS=no_flag")

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("subprocess exited non-zero: %v\n%s", err, out)
	}
}

// TestServeManifest_withFlag verifies that ServeManifest prints valid JSON and
// exits 0 when --plugin-manifest is in os.Args.
//
// We test this via a subprocess that sets SERVE_MANIFEST_SUBPROCESS=with_flag
// so the helper injects --plugin-manifest into os.Args before calling
// ServeManifest. ServeManifest should call os.Exit(0), ending the subprocess.
func TestServeManifest_withFlag(t *testing.T) {
	t.Parallel()

	cmd := exec.Command(
		os.Args[0],
		"-test.run=TestServeManifestSubprocess",
		"-test.v",
	)
	cmd.Env = append(os.Environ(), "SERVE_MANIFEST_SUBPROCESS=with_flag")

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("subprocess exited non-zero: %v\n%s", err, out)
	}

	// The subprocess output should contain valid JSON with the manifest fields.
	raw := string(out)
	jsonStart := strings.Index(raw, "{")
	if jsonStart < 0 {
		t.Fatalf("no JSON object found in subprocess output:\n%s", raw)
	}
	// Find the matching closing brace for the JSON object.
	jsonPart := raw[jsonStart:]
	depth := 0
	jsonEnd := -1
	for i, ch := range jsonPart {
		switch ch {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				jsonEnd = i + 1
			}
		}
		if jsonEnd > 0 {
			break
		}
	}
	if jsonEnd < 0 {
		t.Fatalf("cannot find JSON end in subprocess output:\n%s", raw)
	}

	var m Manifest
	if err := json.Unmarshal([]byte(jsonPart[:jsonEnd]), &m); err != nil {
		t.Fatalf("parse subprocess manifest JSON: %v\noutput:\n%s", err, raw)
	}
	if m.Name != "subprocess-test-plugin" {
		t.Errorf("manifest.Name = %q, want %q", m.Name, "subprocess-test-plugin")
	}
	if m.APIVersion != 1 {
		t.Errorf("manifest.APIVersion = %d, want 1", m.APIVersion)
	}
}

// TestServeManifestSubprocess is the subprocess helper used by the
// TestServeManifest_* tests. It only executes when SERVE_MANIFEST_SUBPROCESS
// is set to "no_flag" or "with_flag".
func TestServeManifestSubprocess(t *testing.T) {
	mode := os.Getenv("SERVE_MANIFEST_SUBPROCESS")
	if mode == "" {
		t.Skip("not running as subprocess")
	}

	m := Manifest{
		Name:        "subprocess-test-plugin",
		Version:     "v0.0.1",
		Description: "test only",
		APIVersion:  1,
	}

	switch mode {
	case "no_flag":
		// os.Args does NOT contain --plugin-manifest; ServeManifest must return.
		ServeManifest(m)
		// If we reach here, ServeManifest returned normally — test passes.

	case "with_flag":
		// Inject --plugin-manifest into os.Args so ServeManifest triggers.
		orig := os.Args
		os.Args = append([]string{orig[0]}, "--plugin-manifest")
		// ServeManifest will call os.Exit(0), ending the subprocess.
		ServeManifest(m)
		// Unreachable if ServeManifest works correctly.
		t.Error("ServeManifest should have called os.Exit(0) but returned")
	}
}
