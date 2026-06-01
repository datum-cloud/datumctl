package plugin

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// buildTokenHelper compiles a small credential-helper binary for tests.
// The binary writes a fixed token to stdout when called with "auth get-token",
// and optionally checks that --session <name> is/isn't present in os.Args.
func buildTokenHelper(t *testing.T, src string) string {
	t.Helper()

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write token helper source: %v", err)
	}

	binName := "credhelper"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binPath := filepath.Join(dir, binName)
	cmd := exec.Command("go", "build", "-o", binPath, srcPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build token helper: %v\n%s", err, out)
	}
	return binPath
}

// TestToken_callsHelperWithSession verifies that when DATUM_SESSION is set,
// Token() invokes the helper with --session <name>.
func TestToken_callsHelperWithSession(t *testing.T) {
	// Not parallel — uses t.Setenv.
	const wantSession = "staging"

	// Helper that exits 1 if --session <name> is not found in os.Args.
	helperSrc := `package main

import (
	"fmt"
	"os"
)

func main() {
	args := os.Args[1:]
	for i, a := range args {
		if a == "--session" && i+1 < len(args) && args[i+1] == "` + wantSession + `" {
			fmt.Println("mytoken")
			os.Exit(0)
		}
	}
	fmt.Fprintln(os.Stderr, "--session ` + wantSession + ` not found in args")
	os.Exit(1)
}
`
	helperPath := buildTokenHelper(t, helperSrc)

	t.Setenv("DATUM_CREDENTIALS_HELPER", helperPath)
	t.Setenv("DATUM_SESSION", wantSession)

	token, err := Token()
	if err != nil {
		t.Fatalf("Token with session: %v", err)
	}
	if strings.TrimSpace(token) != "mytoken" {
		t.Errorf("Token = %q, want %q", token, "mytoken")
	}
}

// TestToken_callsHelperWithoutSession verifies that when DATUM_SESSION is empty,
// Token() does not pass --session to the helper.
func TestToken_callsHelperWithoutSession(t *testing.T) {
	// Not parallel — uses t.Setenv.

	// Helper that exits 1 if --session appears in os.Args at all.
	helperSrc := `package main

import (
	"fmt"
	"os"
)

func main() {
	for _, a := range os.Args[1:] {
		if a == "--session" {
			fmt.Fprintln(os.Stderr, "--session should not appear when DATUM_SESSION is empty")
			os.Exit(1)
		}
	}
	fmt.Println("mytoken")
}
`
	helperPath := buildTokenHelper(t, helperSrc)

	t.Setenv("DATUM_CREDENTIALS_HELPER", helperPath)
	t.Setenv("DATUM_SESSION", "")

	token, err := Token()
	if err != nil {
		t.Fatalf("Token without session: %v", err)
	}
	if strings.TrimSpace(token) != "mytoken" {
		t.Errorf("Token = %q, want %q", token, "mytoken")
	}
}

// TestToken_helperNotSet verifies that Token() returns a descriptive error
// when DATUM_CREDENTIALS_HELPER is not set.
func TestToken_helperNotSet(t *testing.T) {
	// Not parallel — uses t.Setenv.
	t.Setenv("DATUM_CREDENTIALS_HELPER", "")

	_, err := Token()
	if err == nil {
		t.Fatal("Token with no helper: want error, got nil")
	}
	if !strings.Contains(err.Error(), "DATUM_CREDENTIALS_HELPER") {
		t.Errorf("error %q does not mention DATUM_CREDENTIALS_HELPER", err.Error())
	}
}

// TestToken_helperExitsNonZero verifies that Token() returns a wrapped error
// when the credentials helper exits non-zero.
func TestToken_helperExitsNonZero(t *testing.T) {
	// Not parallel — uses t.Setenv.
	helperSrc := `package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "auth error: not logged in")
	os.Exit(1)
}
`
	helperPath := buildTokenHelper(t, helperSrc)

	t.Setenv("DATUM_CREDENTIALS_HELPER", helperPath)
	t.Setenv("DATUM_SESSION", "")

	_, err := Token()
	if err == nil {
		t.Fatal("Token with failing helper: want error, got nil")
	}
}
