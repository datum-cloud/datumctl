package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	"go.datum.net/datumctl/internal/client"
	"go.datum.net/datumctl/internal/datumconfig"
)

func TestDeleteScopeMessage(t *testing.T) {
	tests := []struct {
		name   string
		scope  client.Scope
		dryRun bool
		want   string
	}{
		{
			name:  "project from the active context names its organization",
			scope: client.Scope{Project: "proj-staging", ProjectOrganization: "org-staging", Source: client.ScopeSourceContext},
			want:  `Deleting in project "proj-staging" (organization "org-staging", from the active context)`,
		},
		{
			name:  "organization from the active context",
			scope: client.Scope{Organization: "org-prod", Source: client.ScopeSourceContext},
			want:  `Deleting in organization "org-prod" (from the active context)`,
		},
		{
			name:  "project flag",
			scope: client.Scope{Project: "my-project", Source: client.ScopeSourceFlag},
			want:  `Deleting in project "my-project"`,
		},
		{
			name:  "organization flag",
			scope: client.Scope{Organization: "my-org", Source: client.ScopeSourceFlag},
			want:  `Deleting in organization "my-org"`,
		},
		{
			name:  "project from the environment",
			scope: client.Scope{Project: "env-project", Source: client.ScopeSourceEnv},
			want:  `Deleting in project "env-project" (from DATUM_PROJECT)`,
		},
		{
			name:  "organization from the environment",
			scope: client.Scope{Organization: "env-org", Source: client.ScopeSourceEnv},
			want:  `Deleting in organization "env-org" (from DATUM_ORGANIZATION)`,
		},
		{
			name:  "platform-wide",
			scope: client.Scope{PlatformWide: true, Source: client.ScopeSourceFlag},
			want:  "Deleting platform-wide",
		},
		{
			name:   "dry run",
			scope:  client.Scope{Project: "proj-staging", ProjectOrganization: "org-staging", Source: client.ScopeSourceContext},
			dryRun: true,
			want:   `Dry run in project "proj-staging" (organization "org-staging", from the active context)`,
		},
		{
			name:  "no scope says nothing",
			scope: client.Scope{},
			want:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deleteScopeMessage(tt.scope, tt.dryRun); got != tt.want {
				t.Errorf("deleteScopeMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

// newFakeDelete builds a stand-in for the kubectl delete command: the same
// flags, but a Run that only records that it was reached. Wrapping it the way
// root.go wraps the real one exercises the notice without touching the API.
func newFakeDelete(t *testing.T, ran *bool) (*cobra.Command, *client.DatumCloudFactory) {
	t.Helper()
	factory, err := client.NewDatumFactory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	cmd := &cobra.Command{
		Use: "delete",
		Run: func(cmd *cobra.Command, args []string) { *ran = true },
	}
	cmd.Flags().Bool("all-namespaces", false, "")
	cmdutil.AddDryRunFlag(cmd)
	factory.AddFlags(cmd.Flags())
	return WrapDeleteCommand(cmd, factory), factory
}

func TestWrapDeleteCommandPrintsScope(t *testing.T) {
	tests := []struct {
		name       string
		session    string
		envProject string
		args       []string
		want       string
	}{
		{
			name: "organization context",
			args: []string{"dnszone", "zone-a"},
			want: `Deleting in organization "org-prod" (from the active context)` + "\n",
		},
		{
			name:    "project context names its organization",
			session: ovStaging,
			args:    []string{"dnszone", "zone-a"},
			want:    `Deleting in project "proj-staging" (organization "org-staging", from the active context)` + "\n",
		},
		{
			name: "explicit project flag",
			args: []string{"dnszone", "zone-a", "--project", "other"},
			want: `Deleting in project "other"` + "\n",
		},
		{
			name: "explicit organization flag",
			args: []string{"dnszone", "zone-a", "--organization", "other-org"},
			want: `Deleting in organization "other-org"` + "\n",
		},
		{
			name:       "environment variable",
			envProject: "env-project",
			args:       []string{"dnszone", "zone-a"},
			want:       `Deleting in project "env-project" (from DATUM_PROJECT)` + "\n",
		},
		{
			name: "dry run",
			args: []string{"dnszone", "zone-a", "--dry-run=client"},
			want: `Dry run in organization "org-prod" (from the active context)` + "\n",
		},
		{
			name:    "organizations alias still rewrites the resource",
			session: ovStaging,
			args:    []string{"organizations", "org-staging"},
			want:    `Deleting in project "proj-staging" (organization "org-staging", from the active context)` + "\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupOverrideEnv(t)
			t.Setenv("DATUM_PROJECT", tt.envProject)
			if tt.session != "" {
				datumconfig.SetSessionOverride(tt.session, datumconfig.SessionOverrideFromFlag)
			}

			ran := false
			cmd, _ := newFakeDelete(t, &ran)
			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			cmd.SetArgs(tt.args)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("execute: %v", err)
			}
			if !ran {
				t.Fatal("the wrapped delete command never ran")
			}
			if got := stderr.String(); got != tt.want {
				t.Errorf("stderr = %q, want %q", got, tt.want)
			}
			if stdout.Len() != 0 {
				t.Errorf("stdout should stay clean, got %q", stdout.String())
			}
		})
	}
}

// A scope error (here, a project and an organization both from the
// environment) is the command's to report, not the notice's.
func TestWrapDeleteCommandSkipsNoticeOnScopeError(t *testing.T) {
	setupOverrideEnv(t)
	t.Setenv("DATUM_PROJECT", "p")
	t.Setenv("DATUM_ORGANIZATION", "o")

	ran := false
	cmd, _ := newFakeDelete(t, &ran)
	var stderr bytes.Buffer
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"dnszone", "zone-a"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !ran {
		t.Fatal("the wrapped delete command never ran")
	}
	if stderr.Len() != 0 {
		t.Errorf("expected no notice, got %q", stderr.String())
	}
}

func TestDeleteHelpMentionsContextAndConfirmation(t *testing.T) {
	setupOverrideEnv(t)
	out, err := runRoot(t, "delete", "--help")
	if err != nil {
		t.Fatalf("delete --help: %v\n%s", err, out)
	}
	for _, want := range []string{
		"the active context's project",
		"-i/--interactive",
		"datumctl delete dnszone my-zone --project <project-id> -i",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("help missing %q\n%s", want, out)
		}
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "  --") || strings.HasPrefix(line, "  -") {
			// Flag descriptions come from the wrapped command and are out of
			// scope here; the prose and examples are ours.
			break
		}
		if strings.Contains(strings.ToLower(line), "kubectl") || strings.Contains(strings.ToLower(line), "pod") {
			t.Errorf("help prose mentions kubectl or pods: %q", line)
		}
	}
}
