package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/datumconfig"
	customerrors "go.datum.net/datumctl/internal/errors"
	"go.datum.net/datumctl/internal/keyring"
	"go.datum.net/datumctl/internal/updatecheck"
)

const (
	ovSharedEmail = "swells@datum.net"
	ovProdHost    = "api.datum.net"
	ovStagingHost = "api.staging.env.datum.net"
	ovSoloEmail   = "solo@example.com"
)

var (
	ovProd       = datumconfig.SessionName(ovSharedEmail, ovProdHost)
	ovStaging    = datumconfig.SessionName(ovSharedEmail, ovStagingHost)
	ovSolo       = datumconfig.SessionName(ovSoloEmail, ovProdHost)
	ovProdCtx    = datumconfig.QualifiedContextName(ovProd, "org-prod")
	ovStagingCtx = datumconfig.QualifiedContextName(ovStaging, "org-staging/proj-staging")
)

// setupOverrideEnv points HOME at a temp dir, mocks the keyring, and writes a
// config with one email signed in on prod and staging plus a second email
// signed in once. Prod is active. Staging remembers a project context; the solo
// session has no context. It returns the config path.
func setupOverrideEnv(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv(updatecheck.EnvDisable, "1")
	t.Setenv(datumconfig.SessionEnvVar, "")
	t.Setenv("DATUM_PROJECT", "")
	t.Setenv("DATUM_ORGANIZATION", "")
	keyring.MockInit()
	t.Cleanup(datumconfig.ClearSessionOverride)

	cfg := datumconfig.NewV1Beta1()
	cfg.Sessions = []datumconfig.Session{
		{Name: ovProd, UserKey: "key-prod", UserEmail: ovSharedEmail, UserName: "Scot Prod",
			Endpoint: datumconfig.Endpoint{Server: "https://" + ovProdHost}, LastContext: ovProdCtx},
		{Name: ovStaging, UserKey: "key-staging", UserEmail: ovSharedEmail, UserName: "Scot Staging",
			Endpoint: datumconfig.Endpoint{Server: "https://" + ovStagingHost}, LastContext: ovStagingCtx},
		{Name: ovSolo, UserKey: "key-solo", UserEmail: ovSoloEmail, UserName: "Solo",
			Endpoint: datumconfig.Endpoint{Server: "https://" + ovProdHost}},
	}
	cfg.Contexts = []datumconfig.DiscoveredContext{
		{Name: ovProdCtx, Session: ovProd, OrganizationID: "org-prod"},
		{Name: ovStagingCtx, Session: ovStaging, OrganizationID: "org-staging", ProjectID: "proj-staging"},
	}
	cfg.ActiveSession = ovProd
	cfg.CurrentContext = ovProdCtx
	if err := datumconfig.SaveV1Beta1(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	path, err := datumconfig.DefaultPath()
	if err != nil {
		t.Fatalf("config path: %v", err)
	}
	return path
}

// runRoot executes a fresh root command and returns what the user sees: the
// command's output plus the formatted error, if any.
func runRoot(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := RootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.Execute()
	if err != nil {
		customerrors.Format(&out, err, customerrors.FormatHuman, 0)
	}
	return out.String(), err
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return b
}

func TestSessionOverrideWhoami(t *testing.T) {
	tests := []struct {
		name    string
		env     string
		args    []string
		want    []string
		notWant []string
		wantErr bool
	}{
		{
			name: "no override acts as the active session",
			args: []string{"whoami"},
			want: []string{"User:         Scot Prod (swells@datum.net)", "Context:      org-prod"},
			notWant: []string{"Session:"},
		},
		{
			name: "flag by session name uses that session and its last context",
			args: []string{"whoami", "--session", ovStaging},
			want: []string{
				"User:         Scot Staging (swells@datum.net)",
				"Session:      " + ovStaging + " (from --session; the active session is unchanged)",
				"Endpoint:     " + ovStagingHost,
				"Context:      org-staging/proj-staging",
				"Project:      proj-staging",
			},
		},
		{
			name: "flag before the command works too",
			args: []string{"--session", ovStaging, "whoami"},
			want: []string{"User:         Scot Staging (swells@datum.net)"},
		},
		{
			name: "flag by unique email; session without a context has none",
			args: []string{"whoami", "--session", ovSoloEmail},
			want: []string{
				"User:         Solo (solo@example.com)",
				"Session:      " + ovSolo + " (from --session",
				"Context:      (none)",
				"Pass --project or --organization",
			},
			notWant: []string{"org-prod"},
		},
		{
			name: "env var selects the session",
			env:  ovStaging,
			args: []string{"whoami"},
			want: []string{
				"User:         Scot Staging (swells@datum.net)",
				"(from DATUM_SESSION; the active session is unchanged)",
			},
		},
		{
			name: "env var by unique email",
			env:  ovSoloEmail,
			args: []string{"whoami"},
			want: []string{"User:         Solo (solo@example.com)"},
		},
		{
			name:    "flag beats env var",
			env:     ovStaging,
			args:    []string{"whoami", "--session", ovSoloEmail},
			want:    []string{"User:         Solo (solo@example.com)", "(from --session;"},
			notWant: []string{"Scot Staging"},
		},
		{
			name:    "shared email fails and names each session",
			args:    []string{"whoami", "--session", ovSharedEmail},
			wantErr: true,
			want: []string{
				"swells@datum.net is signed in on more than one endpoint, so --session swells@datum.net is ambiguous.",
				"--session " + ovProd,
				"--session " + ovStaging,
			},
		},
		{
			name:    "shared email in env var fails the same way",
			env:     ovSharedEmail,
			args:    []string{"whoami"},
			wantErr: true,
			want:    []string{"so DATUM_SESSION swells@datum.net is ambiguous", "--session " + ovStaging},
		},
		{
			name:    "unknown value fails and lists valid sessions",
			args:    []string{"whoami", "--session", "nobody@example.com"},
			wantErr: true,
			want: []string{
				"No session matches --session nobody@example.com.",
				"Valid sessions:",
				ovProd, ovStaging, ovSolo,
			},
		},
		{
			name:    "unknown env var fails even when the flag is absent",
			env:     "nobody@example.com",
			args:    []string{"whoami"},
			wantErr: true,
			want:    []string{"No session matches DATUM_SESSION nobody@example.com."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := setupOverrideEnv(t)
			before := readFile(t, path)
			t.Setenv(datumconfig.SessionEnvVar, tt.env)

			out, err := runRoot(t, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v\noutput:\n%s", err, tt.wantErr, out)
			}
			if tt.wantErr {
				if _, ok := customerrors.IsUserError(err); !ok {
					t.Errorf("error is not a UserError: %v", err)
				}
			}
			for _, w := range tt.want {
				if !strings.Contains(out, w) {
					t.Errorf("output missing %q\noutput:\n%s", w, out)
				}
			}
			for _, nw := range tt.notWant {
				if strings.Contains(out, nw) {
					t.Errorf("output unexpectedly contains %q\noutput:\n%s", nw, out)
				}
			}
			if after := readFile(t, path); !bytes.Equal(before, after) {
				t.Errorf("config file changed:\nbefore:\n%s\nafter:\n%s", before, after)
			}
		})
	}
}

// Each process picks its own session; an override must not leak into the next
// command, and a command with no override acts as the active session again.
func TestSessionOverrideSequence(t *testing.T) {
	path := setupOverrideEnv(t)
	before := readFile(t, path)

	steps := []struct {
		env      string
		args     []string
		wantUser string
	}{
		{args: []string{"whoami", "--session", ovStaging}, wantUser: "Scot Staging"},
		{args: []string{"whoami", "--session", ovSoloEmail}, wantUser: "Solo"},
		{env: ovStaging, args: []string{"whoami"}, wantUser: "Scot Staging"},
		{args: []string{"whoami"}, wantUser: "Scot Prod"},
		{args: []string{"whoami", "--session", "nobody@example.com"}},
		{args: []string{"whoami"}, wantUser: "Scot Prod"},
	}
	for i, s := range steps {
		t.Setenv(datumconfig.SessionEnvVar, s.env)
		out, err := runRoot(t, s.args...)
		if s.wantUser == "" {
			if err == nil {
				t.Fatalf("step %d: expected an error\n%s", i, out)
			}
			continue
		}
		if err != nil {
			t.Fatalf("step %d: %v\n%s", i, err, out)
		}
		if !strings.Contains(out, "User:         "+s.wantUser+" (") {
			t.Errorf("step %d (%v): want user %q\noutput:\n%s", i, s.args, s.wantUser, out)
		}
		if s.wantUser == "Scot Prod" && strings.Contains(out, "Session:") {
			t.Errorf("step %d: override leaked into a command without one\n%s", i, out)
		}
	}
	if after := readFile(t, path); !bytes.Equal(before, after) {
		t.Errorf("config file changed across the sequence")
	}
}

func TestSessionOverrideRejectedByActiveSessionCommands(t *testing.T) {
	tests := []struct {
		name string
		args []string
		cmd  string
	}{
		{"login", []string{"login", "--session", ovStaging}, "datumctl login"},
		{"logout", []string{"logout", ovSoloEmail, "--session", ovStaging}, "datumctl logout"},
		{"auth switch", []string{"auth", "switch", ovSoloEmail, "--session", ovStaging}, "datumctl auth switch"},
		{"ctx use", []string{"ctx", "use", "org-prod", "--session", ovStaging}, "datumctl ctx use"},
		{"flag before the command", []string{"--session", ovStaging, "auth", "switch", ovSoloEmail}, "datumctl auth switch"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := setupOverrideEnv(t)
			before := readFile(t, path)

			out, err := runRoot(t, tt.args...)
			if err == nil {
				t.Fatalf("expected an error\noutput:\n%s", out)
			}
			if _, ok := customerrors.IsUserError(err); !ok {
				t.Errorf("error is not a UserError: %v", err)
			}
			want := "'" + tt.cmd + "' changes the active session, so it does not accept --session."
			if !strings.Contains(out, want) {
				t.Errorf("output missing %q\noutput:\n%s", want, out)
			}
			if after := readFile(t, path); !bytes.Equal(before, after) {
				t.Errorf("config file changed after a rejected command")
			}
		})
	}
}

// A stale DATUM_SESSION — left over from a previous logout, or inherited from
// a plugin — must not break commands that never consult the active session.
// It is resolved lazily, only when something actually needs a session.
func TestSessionOverrideStaleEnvDoesNotBreakSessionlessCommands(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"version --client", []string{"version", "--client"}},
		{"plugin list", []string{"plugin", "list"}},
		{"completion bash", []string{"completion", "bash"}},
		{"--help", []string{"--help"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupOverrideEnv(t)
			t.Setenv(datumconfig.SessionEnvVar, "nobody@example.com")

			out, err := runRoot(t, tt.args...)
			if err != nil {
				t.Fatalf("%v should succeed with a stale DATUM_SESSION, got: %v\n%s", tt.args, err, out)
			}
			if strings.Contains(out, "No session matches") {
				t.Errorf("%v should not resolve DATUM_SESSION at all:\n%s", tt.args, out)
			}
		})
	}
}

// Bare `datumctl` (the landing page) does read the active session to render
// itself, but a stale DATUM_SESSION should not turn a no-subcommand invocation
// into a hard error: it renders the page for the real active session and says
// why DATUM_SESSION was ignored.
func TestSessionOverrideStaleEnvOnLandingPage(t *testing.T) {
	setupOverrideEnv(t)
	t.Setenv(datumconfig.SessionEnvVar, "nobody@example.com")

	out, err := runRoot(t)
	if err != nil {
		t.Fatalf("bare datumctl should succeed with a stale DATUM_SESSION, got: %v\n%s", err, out)
	}
	if !strings.Contains(out, "DATUM_SESSION matches no signed-in session") {
		t.Errorf("output missing the stale-override note:\n%s", out)
	}
	if !strings.Contains(out, "swells@datum.net") {
		t.Errorf("output should still show the real active session:\n%s", out)
	}
}

// A wrong --session value is a mistake in this invocation, not a stale
// ambient export, so it still fails immediately — even for a command that
// never otherwise consults the session.
func TestSessionOverrideBadFlagFailsSessionlessCommand(t *testing.T) {
	setupOverrideEnv(t)

	out, err := runRoot(t, "version", "--client", "--session", "nobody@example.com")
	if err == nil {
		t.Fatalf("expected an error, got output:\n%s", out)
	}
	if _, ok := customerrors.IsUserError(err); !ok {
		t.Errorf("error is not a UserError: %v", err)
	}
	if !strings.Contains(out, "No session matches --session nobody@example.com.") {
		t.Errorf("output missing the clear error:\n%s", out)
	}
}

// Commands that change the active session ignore DATUM_SESSION, even when it
// is unresolvable, and act on the stored active session as before.
func TestSessionOverrideEnvIgnoredByActiveSessionCommands(t *testing.T) {
	for _, env := range []string{ovStaging, "nobody@example.com"} {
		t.Run(env, func(t *testing.T) {
			setupOverrideEnv(t)
			t.Setenv(datumconfig.SessionEnvVar, env)

			if out, err := runRoot(t, "auth", "switch", ovSoloEmail); err != nil {
				t.Fatalf("auth switch: %v\n%s", err, out)
			}
			cfg, err := datumconfig.LoadAuto()
			if err != nil {
				t.Fatal(err)
			}
			if cfg.ActiveSession != ovSolo {
				t.Errorf("active session = %q, want %q", cfg.ActiveSession, ovSolo)
			}

			// ctx use resolves in the stored active session (solo now has no
			// contexts), not in the DATUM_SESSION session.
			out, err := runRoot(t, "ctx", "use", "org-staging/proj-staging")
			if err == nil {
				t.Fatalf("ctx use should resolve in the active session, not DATUM_SESSION\n%s", out)
			}
			if strings.Contains(out, "No session matches") {
				t.Errorf("ctx use read DATUM_SESSION:\n%s", out)
			}
		})
	}
}

// captureStdout runs fn and returns what it wrote to os.Stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()
	fn()
	w.Close()
	b, _ := io.ReadAll(r)
	return string(b)
}

func seedToken(t *testing.T, userKey, token string) {
	t.Helper()
	creds := authutil.StoredCredentials{
		Hostname: "auth.datum.net",
		Token:    &oauth2.Token{AccessToken: token, Expiry: time.Now().Add(time.Hour)},
	}
	blob, err := json.Marshal(creds)
	if err != nil {
		t.Fatal(err)
	}
	if err := keyring.Set(authutil.ServiceName, userKey, string(blob)); err != nil {
		t.Fatal(err)
	}
}

// Kubeconfig exec entries call "auth get-token --session <name>"; that must
// keep returning that session's token, whatever DATUM_SESSION says.
func TestGetTokenSessionFlagUnchanged(t *testing.T) {
	tests := []struct {
		name string
		env  string
		args []string
		want string
	}{
		{"session flag by name", "", []string{"auth", "get-token", "--session", ovStaging}, "tok-staging"},
		{"session flag beats env var", ovSoloEmail, []string{"auth", "get-token", "--session", ovStaging}, "tok-staging"},
		{"no flag uses the active session", "", []string{"auth", "get-token"}, "tok-prod"},
		{"no flag honors DATUM_SESSION", ovSoloEmail, []string{"auth", "get-token"}, "tok-solo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := setupOverrideEnv(t)
			seedToken(t, "key-prod", "tok-prod")
			seedToken(t, "key-staging", "tok-staging")
			seedToken(t, "key-solo", "tok-solo")
			before := readFile(t, path)
			t.Setenv(datumconfig.SessionEnvVar, tt.env)

			var err error
			var out string
			got := captureStdout(t, func() { out, err = runRoot(t, tt.args...) })
			if err != nil {
				t.Fatalf("get-token: %v\n%s", err, out)
			}
			if got != tt.want {
				t.Errorf("token = %q, want %q", got, tt.want)
			}
			if after := readFile(t, path); !bytes.Equal(before, after) {
				t.Errorf("config file changed")
			}
		})
	}

	// The local flag keeps its old, exact-name-only error.
	setupOverrideEnv(t)
	_, err := runRoot(t, "auth", "get-token", "--session", ovSoloEmail)
	if err == nil || !strings.Contains(err.Error(), `no session named "`+ovSoloEmail+`"`) {
		t.Errorf("get-token --session <email> error = %v, want the unchanged exact-name error", err)
	}
}

// The overriding session's context never leaks into the config file, even
// when a command saves the config for other reasons.
func TestSessionOverrideLeavesStoredSessionAlone(t *testing.T) {
	path := setupOverrideEnv(t)
	if _, err := runRoot(t, "ctx", "list", "--session", ovStaging); err != nil {
		t.Fatalf("ctx list: %v", err)
	}
	cfg, err := datumconfig.LoadAutoFromPath(filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ActiveSession != ovProd || cfg.CurrentContext != ovProdCtx {
		t.Errorf("stored active session/context = %q/%q, want %q/%q",
			cfg.ActiveSession, cfg.CurrentContext, ovProd, ovProdCtx)
	}
}
