package plugindispatch

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"go.datum.net/datumctl/internal/client"
	customerrors "go.datum.net/datumctl/internal/errors"
)

// buildCobraTreeWithGlobalFlags is buildMinimalCobraTree with datumctl's global
// flags registered the way root.go does, so the forward paths can tell them
// apart from a plugin's own flags.
func buildCobraTreeWithGlobalFlags(factory *client.DatumCloudFactory) *cobra.Command {
	root := buildMinimalCobraTree()
	root.PersistentFlags().String("error-format", customerrors.FormatHuman, "")
	root.PersistentFlags().String("session", "", "")
	factory.AddFlags(root.PersistentFlags())
	factory.AddFlagMutualExclusions(root)
	return root
}

// stubExec replaces execPlatform for the test and reports what it was handed.
func stubExec(t *testing.T) (called *bool, args *[]string, env *[]string) {
	t.Helper()
	called, args, env = new(bool), new([]string), new([]string)
	orig := execPlatform
	execPlatform = func(_ string, a []string, e []string) error {
		*called, *args, *env = true, a, e
		return nil
	}
	t.Cleanup(func() { execPlatform = orig })
	return called, args, env
}

func setArgs(t *testing.T, args ...string) {
	t.Helper()
	orig := os.Args
	os.Args = append([]string{"datumctl"}, args...)
	t.Cleanup(func() { os.Args = orig })
}

func TestSplitGlobalFlags(t *testing.T) {
	globals := pflag.NewFlagSet("test", pflag.ContinueOnError)
	globals.String("project", "", "")
	globals.String("organization", "", "")
	globals.Bool("platform-wide", false, "")
	globals.StringP("namespace", "n", "", "")
	globals.String("error-format", "human", "")

	tests := []struct {
		name       string
		in         []string
		wantValues []globalFlagValue
		wantRest   []string
		wantErr    bool
	}{
		{name: "no flags", in: []string{"compute", "logs"},
			wantRest: []string{"compute", "logs"}},
		// Issue #316: the plugin's -c must not be seen by datumctl.
		{name: "flag before plugin name", in: []string{"--project", "P", "compute", "logs", "W", "-c", "zitadel"},
			wantValues: []globalFlagValue{{"project", "P"}},
			wantRest:   []string{"compute", "logs", "W", "-c", "zitadel"}},
		{name: "flag=value form", in: []string{"--project=P", "compute", "logs", "W", "-c", "zitadel"},
			wantValues: []globalFlagValue{{"project", "P"}},
			wantRest:   []string{"compute", "logs", "W", "-c", "zitadel"}},
		// A bool flag takes no value: the next word is the plugin name.
		{name: "bool flag", in: []string{"--platform-wide", "compute", "-c", "x"},
			wantValues: []globalFlagValue{{"platform-wide", "true"}},
			wantRest:   []string{"compute", "-c", "x"}},
		{name: "shorthand", in: []string{"-n", "default", "compute"},
			wantValues: []globalFlagValue{{"namespace", "default"}},
			wantRest:   []string{"compute"}},
		{name: "several flags, in order", in: []string{"--error-format", "json", "--organization=O", "compute"},
			wantValues: []globalFlagValue{{"error-format", "json"}, {"organization", "O"}},
			wantRest:   []string{"compute"}},
		// Everything after the plugin name is the plugin's, even flags that
		// share datumctl's names and shorthands.
		{name: "flags after plugin name untouched", in: []string{"compute", "--project", "X", "-n", "y"},
			wantRest: []string{"compute", "--project", "X", "-n", "y"}},
		// A bare "--" ends datumctl's flags, as it does for cobra; a later one
		// is passed to the plugin as is.
		{name: "-- passthrough", in: []string{"--project", "P", "--", "compute", "logs", "--", "--project", "X"},
			wantValues: []globalFlagValue{{"project", "P"}},
			wantRest:   []string{"compute", "logs", "--", "--project", "X"}},
		{name: "unknown flag", in: []string{"--bogus", "compute"}, wantErr: true},
		{name: "missing value", in: []string{"--project"}, wantErr: true},
		{name: "help", in: []string{"--help", "compute"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, rest, err := splitGlobalFlags(globals, tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !slices.Equal(values, tt.wantValues) {
				t.Errorf("values = %v, want %v", values, tt.wantValues)
			}
			if !slices.Equal(rest, tt.wantRest) {
				t.Errorf("rest = %q, want %q", rest, tt.wantRest)
			}
		})
	}
}

// Issue #316: "datumctl --project P <plugin> ..." must run the plugin exactly
// as "datumctl <plugin> ... --project P" does. The leading global flags are
// datumctl's: they set the scope the plugin receives through DATUM_PROJECT and
// DATUM_ORG and never reach the plugin as arguments, while everything from the
// plugin name on is forwarded verbatim.
func TestForwardPlugin_globalFlagsBeforePluginName(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantArgs    []string
		wantProject string
		wantOrg     string
	}{
		{name: "--project value", args: []string{"--project", "other-proj", "assistant", "logs", "W", "-c", "zitadel"},
			wantArgs: []string{"logs", "W", "-c", "zitadel"}, wantProject: "other-proj"},
		{name: "--project=value", args: []string{"--project=other-proj", "assistant", "logs", "W", "-c", "zitadel"},
			wantArgs: []string{"logs", "W", "-c", "zitadel"}, wantProject: "other-proj"},
		{name: "--organization", args: []string{"--organization", "other-org", "assistant", "card"},
			wantArgs: []string{"card"}, wantOrg: "other-org"},
		// The active context is org-scoped; --platform-wide clears it.
		{name: "--platform-wide", args: []string{"--platform-wide", "assistant", "card", "-c", "x"},
			wantArgs: []string{"card", "-c", "x"}},
		{name: "mixed with --session and a shorthand", args: []string{"-n", "default", "--session", ovStaging, "--project", "other-proj", "assistant", "card"},
			wantArgs: []string{"card"}, wantProject: "other-proj"},
		{name: "-- passthrough", args: []string{"--project", "other-proj", "--", "assistant", "logs", "--", "--project", "X"},
			wantArgs: []string{"logs", "--", "--project", "X"}, wantProject: "other-proj"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Not parallel — mutates os.Args, execPlatform, and environment.
			factory := setupPluginOverrideEnv(t)
			managedDir := t.TempDir()
			t.Setenv("PATH", t.TempDir())
			bin := writeFakeBinary(t, managedDir, "assistant")
			recordManaged(t, managedDir, "assistant", sha256HexFile(t, bin))
			called, gotArgs, gotEnv := stubExec(t)
			setArgs(t, tt.args...)

			if err := ForwardPlugin(managedDir, buildCobraTreeWithGlobalFlags(factory), factory); err != nil {
				t.Fatalf("ForwardPlugin: %v", err)
			}
			if !*called {
				t.Fatal("plugin was not exec'd; the line fell through to cobra, which rejects the plugin's flags")
			}
			if !slices.Equal(*gotArgs, tt.wantArgs) {
				t.Errorf("forwarded args = %q, want %q", *gotArgs, tt.wantArgs)
			}
			if got := envValue(*gotEnv, "DATUM_PROJECT"); got != tt.wantProject {
				t.Errorf("DATUM_PROJECT = %q, want %q", got, tt.wantProject)
			}
			if got := envValue(*gotEnv, "DATUM_ORG"); got != tt.wantOrg {
				t.Errorf("DATUM_ORG = %q, want %q", got, tt.wantOrg)
			}
		})
	}
}

// Without --project and without --platform-wide the plugin still gets the
// active context's scope, so the leading-flag parse must not clear it.
func TestForwardPlugin_noGlobalFlags_keepsContextScope(t *testing.T) {
	factory := setupPluginOverrideEnv(t)
	managedDir := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	bin := writeFakeBinary(t, managedDir, "assistant")
	recordManaged(t, managedDir, "assistant", sha256HexFile(t, bin))
	called, _, gotEnv := stubExec(t)
	setArgs(t, "assistant", "card")

	if err := ForwardPlugin(managedDir, buildCobraTreeWithGlobalFlags(factory), factory); err != nil {
		t.Fatalf("ForwardPlugin: %v", err)
	}
	if !*called {
		t.Fatal("plugin was not exec'd")
	}
	if got := envValue(*gotEnv, "DATUM_ORG"); got != "org-prod" {
		t.Errorf("DATUM_ORG = %q, want the active context's %q", got, "org-prod")
	}
}

// A line that is not a plugin invocation is left to cobra untouched: the
// leading global flags are parsed to find the command name, but not applied,
// so cobra starts from a clean flag set.
func TestForwardPlugin_globalFlagsBeforeNonPlugin_fallThroughUntouched(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "built-in command", args: []string{"--project", "P", "get", "dnszones"}},
		{name: "unknown command", args: []string{"--project", "P", "nosuchplugin", "logs", "-c", "x"}},
		{name: "unknown leading flag", args: []string{"--bogus", "assistant", "card"}},
		{name: "help", args: []string{"--help", "assistant"}},
		{name: "only flags", args: []string{"--project", "P"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := setupPluginOverrideEnv(t)
			managedDir := t.TempDir()
			t.Setenv("PATH", t.TempDir())
			bin := writeFakeBinary(t, managedDir, "assistant")
			recordManaged(t, managedDir, "assistant", sha256HexFile(t, bin))
			called, _, _ := stubExec(t)
			setArgs(t, tt.args...)
			root := buildCobraTreeWithGlobalFlags(factory)

			if err := ForwardPlugin(managedDir, root, factory); err != nil {
				t.Fatalf("ForwardPlugin: want nil (fall through), got %v", err)
			}
			if *called {
				t.Fatal("a non-plugin line was exec'd")
			}
			if *factory.ConfigFlags.Project != "" {
				t.Errorf("factory project = %q, want untouched", *factory.ConfigFlags.Project)
			}
			if root.PersistentFlags().Changed("project") {
				t.Error("--project marked changed before cobra parsed it")
			}
		})
	}
}

// An untrusted PATH plugin behind a global flag is still never exec'd: the
// trust gate runs before the flags are applied.
func TestForwardPlugin_globalFlagsBeforeUntrustedPlugin_notExeced(t *testing.T) {
	pathDir := t.TempDir()
	managedDir := t.TempDir()
	writeFakeBinary(t, pathDir, "datumctl-search")
	t.Setenv("PATH", pathDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("DATUMCTL_TRUSTED_PLUGINS", "")
	factory := buildMinimalFactory(t)
	called, _, _ := stubExec(t)
	setArgs(t, "--project", "P", "search", "kinds")

	if err := ForwardPlugin(managedDir, buildCobraTreeWithGlobalFlags(factory), factory); err != nil {
		t.Fatalf("ForwardPlugin: want nil (fall through), got %v", err)
	}
	if *called {
		t.Fatal("untrusted PATH binary was exec'd")
	}
	if *factory.ConfigFlags.Project != "" {
		t.Errorf("factory project = %q, want untouched", *factory.ConfigFlags.Project)
	}
}

// Mutually exclusive scope flags are reported in datumctl's own words rather
// than running the plugin with one of them or falling through to cobra's
// "unknown flag" for the plugin's flags.
func TestForwardPlugin_conflictingScopeFlags_returnsUserError(t *testing.T) {
	factory := setupPluginOverrideEnv(t)
	managedDir := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	bin := writeFakeBinary(t, managedDir, "assistant")
	recordManaged(t, managedDir, "assistant", sha256HexFile(t, bin))
	called, _, _ := stubExec(t)
	setArgs(t, "--project", "P", "--organization", "O", "assistant", "card", "-c", "x")

	err := ForwardPlugin(managedDir, buildCobraTreeWithGlobalFlags(factory), factory)
	if err == nil {
		t.Fatal("expected an error for --project with --organization")
	}
	if _, ok := customerrors.IsUserError(err); !ok {
		t.Errorf("error is not a UserError: %v", err)
	}
	for _, w := range []string{"project", "organization"} {
		if !strings.Contains(err.Error(), w) {
			t.Errorf("error does not name %q:\n%s", w, err)
		}
	}
	if *called {
		t.Fatal("plugin was exec'd despite conflicting scope flags")
	}
}

// "datumctl --project P <plugin> --help" must reach the plugin's help, so the
// plugin name has to be found behind the global flags. The untrusted-plugin
// error proves the name was resolved; before the fix the line fell through to
// cobra and printed datumctl's root help.
func TestForwardHelp_globalFlagsBeforePluginName(t *testing.T) {
	pathDir := t.TempDir()
	managedDir := t.TempDir()
	writeFakeBinary(t, pathDir, "datumctl-myext")
	t.Setenv("PATH", pathDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("DATUMCTL_TRUSTED_PLUGINS", "")
	factory := buildMinimalFactory(t)
	setArgs(t, "--project", "P", "myext", "sub", "--help")

	err := ForwardHelp(managedDir, buildCobraTreeWithGlobalFlags(factory))
	if err == nil || !strings.Contains(err.Error(), "has not been trusted") {
		t.Fatalf("ForwardHelp: want the untrusted-plugin error (plugin name resolved behind --project), got %v", err)
	}
}
