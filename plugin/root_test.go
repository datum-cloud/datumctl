package plugin

import (
	"testing"
)

// TestNewRootCmd_defaultsFromEnv verifies that when DATUM_ORG and DATUM_PROJECT
// are set, the --org and --project persistent flags on the root command have
// defaults matching those values.
func TestNewRootCmd_defaultsFromEnv(t *testing.T) {
	// Not parallel — uses t.Setenv.
	t.Setenv("DATUM_ORG", "my-org")
	t.Setenv("DATUM_PROJECT", "my-project")
	// Clear other DATUM_* vars to keep test focused.
	t.Setenv("DATUM_PLUGIN_API_VERSION", "")
	t.Setenv("DATUM_CREDENTIALS_HELPER", "")
	t.Setenv("DATUM_SESSION", "")

	cmd := NewRootCmd("test-plugin", "A test plugin")

	orgFlag := cmd.PersistentFlags().Lookup("org")
	if orgFlag == nil {
		t.Fatal("--org flag not found on root command")
	}
	if orgFlag.DefValue != "my-org" {
		t.Errorf("--org default = %q, want %q", orgFlag.DefValue, "my-org")
	}

	projectFlag := cmd.PersistentFlags().Lookup("project")
	if projectFlag == nil {
		t.Fatal("--project flag not found on root command")
	}
	if projectFlag.DefValue != "my-project" {
		t.Errorf("--project default = %q, want %q", projectFlag.DefValue, "my-project")
	}
}

// TestNewRootCmd_outputFlagPresent verifies that the --output / -o flag is
// wired up with a default of "table".
func TestNewRootCmd_outputFlagPresent(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd("test-plugin", "A test plugin")

	outputFlag := cmd.PersistentFlags().Lookup("output")
	if outputFlag == nil {
		t.Fatal("--output flag not found on root command")
	}
	if outputFlag.DefValue != "table" {
		t.Errorf("--output default = %q, want %q", outputFlag.DefValue, "table")
	}
}
