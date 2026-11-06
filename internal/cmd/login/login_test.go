package login

import (
	"testing"

	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/datumconfig"
	"go.datum.net/datumctl/internal/discovery"
)

func testDirectory() *discovery.Directory {
	return &discovery.Directory{
		Session: "alice@example.com@api.datum.net",
		Orgs: []discovery.DiscoveredOrg{
			{Name: "acme", DisplayName: "Acme"},
			{Name: "globex", DisplayName: "Globex"},
		},
		Projects: []discovery.DiscoveredProject{
			{Name: "acme-web", DisplayName: "Web", OrgName: "acme"},
			{Name: "acme-api", DisplayName: "API", OrgName: "acme"},
			{Name: "globex-web", DisplayName: "Web", OrgName: "globex"},
		},
	}
}

func TestChooseContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		dir         *discovery.Directory
		requested   string
		interactive bool
		wantRef     string
		wantErr     bool
	}{
		{name: "project by id", dir: testDirectory(), requested: "acme-api", wantRef: "acme/acme-api"},
		{name: "org/project", dir: testDirectory(), requested: "globex/globex-web", wantRef: "globex/globex-web"},
		{name: "org only", dir: testDirectory(), requested: "acme", wantRef: "acme"},
		{name: "requested works without a terminal", dir: testDirectory(), requested: "acme/acme-web", interactive: false, wantRef: "acme/acme-web"},
		{name: "unknown requested context errors", dir: testDirectory(), requested: "nope", wantErr: true},
		{name: "ambiguous display name errors", dir: testDirectory(), requested: "Web", wantErr: true},
		{name: "no terminal and nothing requested selects nothing", dir: testDirectory(), interactive: false, wantRef: ""},
		{
			name: "a lone context is chosen without a terminal",
			dir: &discovery.Directory{
				Session: "bob@example.com@api.datum.net",
				Orgs:    []discovery.DiscoveredOrg{{Name: "solo"}},
			},
			interactive: false,
			wantRef:     "solo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := chooseContext(tt.dir, "", tt.requested, tt.interactive)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got context %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			gotRef := ""
			if got != nil {
				gotRef = got.Ref()
			}
			if gotRef != tt.wantRef {
				t.Fatalf("chooseContext() = %q, want %q", gotRef, tt.wantRef)
			}
		})
	}
}

// loginCmdWithScopeFlags mirrors how login is mounted: --project and
// --organization are persistent flags on the root command.
func loginCmdWithScopeFlags(t *testing.T, args ...string) *cobra.Command {
	t.Helper()
	root := &cobra.Command{Use: "datumctl"}
	root.PersistentFlags().String("project", "", "")
	root.PersistentFlags().String("organization", "", "")
	login := &cobra.Command{Use: "login", RunE: func(*cobra.Command, []string) error { return nil }}
	root.AddCommand(login)
	root.SetArgs(append([]string{"login"}, args...))
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	return login
}

func TestRequestedContext(t *testing.T) {
	t.Parallel()

	env := func(vars map[string]string) func(string) string {
		return func(k string) string { return vars[k] }
	}

	tests := []struct {
		name string
		args []string
		env  map[string]string
		want string
	}{
		{name: "nothing", want: ""},
		{name: "project flag", args: []string{"--project", "acme-web"}, want: "acme-web"},
		{name: "project flag with org prefix", args: []string{"--project", "acme/acme-web"}, want: "acme/acme-web"},
		{name: "organization flag", args: []string{"--organization", "acme"}, want: "acme"},
		{name: "project env", env: map[string]string{"DATUM_PROJECT": "acme-web"}, want: "acme-web"},
		{name: "organization env", env: map[string]string{"DATUM_ORGANIZATION": "acme"}, want: "acme"},
		{name: "both env", env: map[string]string{"DATUM_ORGANIZATION": "acme", "DATUM_PROJECT": "acme-web"}, want: "acme/acme-web"},
		{name: "flag beats env", args: []string{"--project", "acme-api"}, env: map[string]string{"DATUM_PROJECT": "acme-web"}, want: "acme-api"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := loginCmdWithScopeFlags(t, tt.args...)
			if got := requestedContext(cmd, env(tt.env)); got != tt.want {
				t.Fatalf("requestedContext() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClearOtherSessionContext(t *testing.T) {
	t.Parallel()

	newCfg := func() *datumconfig.ConfigV1Beta1 {
		cfg := &datumconfig.ConfigV1Beta1{}
		ctx := datumconfig.DiscoveredContext{Session: "old@example.com@api.datum.net", OrganizationID: "acme"}
		ctx.Name = ctx.QualifiedName()
		cfg.UpsertContext(ctx)
		cfg.CurrentContext = ctx.Name
		return cfg
	}

	t.Run("another session's context is cleared", func(t *testing.T) {
		t.Parallel()
		cfg := newCfg()
		clearOtherSessionContext(cfg, "new@example.com@api.datum.net")
		if cfg.CurrentContext != "" {
			t.Fatalf("CurrentContext = %q, want it cleared", cfg.CurrentContext)
		}
	})

	t.Run("the same session's context is kept", func(t *testing.T) {
		t.Parallel()
		cfg := newCfg()
		want := cfg.CurrentContext
		clearOtherSessionContext(cfg, "old@example.com@api.datum.net")
		if cfg.CurrentContext != want {
			t.Fatalf("CurrentContext = %q, want %q", cfg.CurrentContext, want)
		}
	})
}
