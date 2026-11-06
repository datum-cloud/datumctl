package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	"go.datum.net/datumctl/internal/client"
)

// newFakeGet stands in for kubectl get: the flags the wrapper reads, and a Run
// that fails the way kubectl does when the server refuses the list.
func newFakeGet(t *testing.T, runErr error) *cobra.Command {
	t.Helper()
	factory, err := client.NewDatumFactory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	cmd := &cobra.Command{
		Use: "get",
		Run: func(cmd *cobra.Command, args []string) { cmdutil.CheckErr(runErr) },
	}
	cmd.Flags().BoolP("all-namespaces", "A", false, "")
	cmd.Flags().StringP("output", "o", "", "")
	factory.AddFlags(cmd.Flags())
	return WrapGetCommand(cmd, factory, genericclioptions.IOStreams{Out: &bytes.Buffer{}, ErrOut: &bytes.Buffer{}})
}

func captureFatal(t *testing.T) *string {
	t.Helper()
	var got string
	cmdutil.BehaviorOnFatal(func(msg string, code int) { got = msg })
	t.Cleanup(cmdutil.DefaultBehaviorOnFatal)
	t.Cleanup(func() { getFatalHint = "" })
	return &got
}

func TestGetForbiddenAllNamespacesHint(t *testing.T) {
	forbidden := apierrors.NewForbidden(
		schema.GroupResource{Group: "dns.networking.miloapis.com", Resource: "dnszones"}, "",
		errFake("cross-namespace queries not allowed for organization-scoped requests"),
	)
	notFound := apierrors.NewNotFound(schema.GroupResource{Resource: "dnszones"}, "zone-a")

	tests := []struct {
		name     string
		args     []string
		runErr   error
		wantHint []string
	}{
		{
			name:     "organization scope from the context",
			args:     []string{"dnszones", "-A"},
			runErr:   forbidden,
			wantHint: []string{"datumctl get dnszones --project <project> -A", "datumctl get projects --organization org-prod"},
		},
		{
			name:     "organization flag",
			args:     []string{"dnszones", "--all-namespaces", "--organization", "acme"},
			runErr:   forbidden,
			wantHint: []string{"datumctl get dnszones --project <project> -A", "datumctl get projects --organization acme"},
		},
		{
			name:   "project scope gets no hint",
			args:   []string{"dnszones", "-A", "--project", "web"},
			runErr: forbidden,
		},
		{
			name:   "organization scope without -A gets no hint",
			args:   []string{"dnszones", "--organization", "acme"},
			runErr: forbidden,
		},
		{
			name:   "an error other than Forbidden gets no hint",
			args:   []string{"dnszones", "-A", "--organization", "acme"},
			runErr: notFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupOverrideEnv(t)
			fatal := captureFatal(t)

			cmd := newFakeGet(t, tt.runErr)
			cmd.SetArgs(tt.args)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("execute: %v", err)
			}
			if *fatal == "" {
				t.Fatal("the wrapped get command never reported its error")
			}

			hint := FatalHint(*fatal)
			if len(tt.wantHint) == 0 {
				if hint != "" {
					t.Errorf("expected no hint, got %q", hint)
				}
				return
			}
			for _, want := range tt.wantHint {
				if !strings.Contains(hint, want) {
					t.Errorf("hint %q does not contain %q", hint, want)
				}
			}
		})
	}
}

type errFake string

func (e errFake) Error() string { return string(e) }

func TestGetExampleListsAllNamespacesInAProject(t *testing.T) {
	setupOverrideEnv(t)
	get, _, err := RootCmd().Find([]string{"get"})
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(get.Example, "\n") {
		if strings.Contains(line, "--all-namespaces") || strings.Contains(line, " -A") {
			if strings.Contains(line, "--organization") {
				t.Errorf("example lists across all namespaces at organization scope, which the server refuses: %q", line)
			}
		}
	}
	if !strings.Contains(get.Example, "--project <project-id> --all-namespaces") {
		t.Error("example should show listing across all namespaces in a project")
	}
}
