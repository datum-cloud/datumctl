package plugin

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Token calls the datumctl credentials helper and returns a fresh access token.
// It resolves the helper path from DATUM_CREDENTIALS_HELPER.
// Plugins should call Token() immediately before each API call — tokens are short-lived.
func Token() (string, error) {
	ctx := Context()
	if ctx.CredentialsHelper == "" {
		return "", fmt.Errorf("DATUM_CREDENTIALS_HELPER is not set; is this plugin running via datumctl?")
	}

	args := []string{"auth", "get-token"}
	if ctx.Session != "" {
		args = append(args, "--session", ctx.Session)
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command(ctx.CredentialsHelper, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("credentials helper failed: %w\nstderr: %s", err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}
