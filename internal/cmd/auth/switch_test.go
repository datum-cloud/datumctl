package auth

import (
	"bytes"
	"strings"
	"testing"

	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/datumconfig"
	customerrors "go.datum.net/datumctl/internal/errors"
	"go.datum.net/datumctl/internal/keyring"
	"go.datum.net/datumctl/internal/picker"
)

const (
	sharedEmail = "swells@datum.net"
	prodHost    = "api.datum.net"
	stagingHost = "api.staging.env.datum.net"
	otherEmail  = "solo@example.com"
)

var (
	prodSession    = datumconfig.SessionName(sharedEmail, prodHost)
	stagingSession = datumconfig.SessionName(sharedEmail, stagingHost)
	soloSession    = datumconfig.SessionName(otherEmail, prodHost)
	prodCtx        = datumconfig.QualifiedContextName(prodSession, "org-prod")
	stagingCtx     = datumconfig.QualifiedContextName(stagingSession, "org-staging")
	soloCtx        = datumconfig.QualifiedContextName(soloSession, "org-solo")
)

// setupSwitchEnv points HOME at a temp dir, mocks the keyring, forces the
// no-terminal path unless tty is true, and writes a config with one email
// signed in on prod and staging plus a second, single-session email.
func setupSwitchEnv(t *testing.T, tty bool) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	keyring.MockInit()

	orig := picker.IsTerminal
	picker.IsTerminal = func() bool { return tty }
	t.Cleanup(func() { picker.IsTerminal = orig })

	cfg := datumconfig.NewV1Beta1()
	cfg.Sessions = []datumconfig.Session{
		{Name: prodSession, UserKey: "key-prod", UserEmail: sharedEmail, UserName: "Scot Prod",
			Endpoint: datumconfig.Endpoint{Server: "https://" + prodHost}, LastContext: prodCtx},
		{Name: stagingSession, UserKey: "key-staging", UserEmail: sharedEmail, UserName: "Scot Staging",
			Endpoint: datumconfig.Endpoint{Server: "https://" + stagingHost}, LastContext: stagingCtx},
		{Name: soloSession, UserKey: "key-solo", UserEmail: otherEmail, UserName: "Solo",
			Endpoint: datumconfig.Endpoint{Server: "https://" + prodHost}, LastContext: soloCtx},
	}
	cfg.Contexts = []datumconfig.DiscoveredContext{
		{Name: prodCtx, Session: prodSession, OrganizationID: "org-prod"},
		{Name: stagingCtx, Session: stagingSession, OrganizationID: "org-staging"},
		{Name: soloCtx, Session: soloSession, OrganizationID: "org-solo"},
	}
	cfg.ActiveSession = soloSession
	cfg.CurrentContext = soloCtx
	if err := datumconfig.SaveV1Beta1(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
}

func runSwitchCmd(t *testing.T, args ...string) error {
	t.Helper()
	cmd := Command()
	cmd.SetArgs(append([]string{"switch"}, args...))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	return cmd.Execute()
}

func loadCfg(t *testing.T) *datumconfig.ConfigV1Beta1 {
	t.Helper()
	cfg, err := datumconfig.LoadAuto()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	return cfg
}

func TestSwitch(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantSession string
		wantContext string
		// Error expectations; set when the switch must fail.
		wantMsg   []string
		wantHint  []string
		wantLines int // number of "datumctl auth switch" commands in the hint
	}{
		{
			name:        "single-session email switches directly",
			args:        []string{otherEmail},
			wantSession: soloSession,
			wantContext: soloCtx,
		},
		{
			name:        "shared email with endpoint picks staging",
			args:        []string{sharedEmail, "--endpoint", stagingHost},
			wantSession: stagingSession,
			wantContext: stagingCtx,
		},
		{
			name:        "shared email with endpoint picks prod",
			args:        []string{sharedEmail, "--endpoint", prodHost},
			wantSession: prodSession,
			wantContext: prodCtx,
		},
		{
			name:        "endpoint with scheme and trailing slash",
			args:        []string{sharedEmail, "--endpoint", "https://" + stagingHost + "/"},
			wantSession: stagingSession,
			wantContext: stagingCtx,
		},
		{
			name:        "exact session name",
			args:        []string{stagingSession},
			wantSession: stagingSession,
			wantContext: stagingCtx,
		},
		{
			name:     "unknown endpoint lists the endpoints for that email",
			args:     []string{sharedEmail, "--endpoint", "api.nowhere.example"},
			wantMsg:  []string{"No session for " + sharedEmail + " on endpoint api.nowhere.example."},
			wantHint: []string{"Signed-in endpoints: " + prodHost + ", " + stagingHost},
		},
		{
			name:     "unknown endpoint without email lists endpoints across every account",
			args:     []string{"--endpoint", "api.nowhere.example"},
			wantMsg:  []string{"No session on endpoint api.nowhere.example."},
			wantHint: []string{"Endpoints any account is signed in on: " + prodHost + ", " + stagingHost},
		},
		{
			name:    "ambiguous email without terminal lists one command per session",
			args:    []string{sharedEmail},
			wantMsg: []string{sharedEmail + " is signed in on more than one endpoint", "requires a terminal"},
			wantHint: []string{
				"datumctl auth switch " + sharedEmail + " --endpoint " + prodHost,
				"datumctl auth switch " + sharedEmail + " --endpoint " + stagingHost,
			},
			wantLines: 2,
		},
		{
			name:    "no argument without terminal lists every session",
			args:    nil,
			wantMsg: []string{"Multiple sessions found. Choosing one interactively requires a terminal."},
			wantHint: []string{
				"datumctl auth switch " + sharedEmail + " --endpoint " + prodHost,
				"datumctl auth switch " + sharedEmail + " --endpoint " + stagingHost,
				"datumctl auth switch " + otherEmail + "\n",
			},
			wantLines: 3,
		},
		{
			name:     "unknown email",
			args:     []string{"nobody@example.com"},
			wantMsg:  []string{"No sessions found for nobody@example.com."},
			wantHint: []string{"datumctl auth list"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setupSwitchEnv(t, false)
			err := runSwitchCmd(t, tc.args...)

			if tc.wantMsg == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				cfg := loadCfg(t)
				if cfg.ActiveSession != tc.wantSession {
					t.Errorf("ActiveSession = %q, want %q", cfg.ActiveSession, tc.wantSession)
				}
				if cfg.CurrentContext != tc.wantContext {
					t.Errorf("CurrentContext = %q, want %q", cfg.CurrentContext, tc.wantContext)
				}
				if got := cfg.ActiveSessionEntry(); got == nil || got.Name != tc.wantSession {
					t.Errorf("ActiveSessionEntry = %v, want %q", got, tc.wantSession)
				}
				return
			}

			if err == nil {
				t.Fatal("expected error, got nil")
			}
			userErr, ok := customerrors.IsUserError(err)
			if !ok {
				t.Fatalf("expected *UserError, got %T: %v", err, err)
			}
			for _, want := range tc.wantMsg {
				if !strings.Contains(userErr.Message, want) {
					t.Errorf("Message = %q, want it to contain %q", userErr.Message, want)
				}
			}
			// Trailing newline lets a case pin an exact line end.
			hint := userErr.Hint + "\n"
			for _, want := range tc.wantHint {
				if !strings.Contains(hint, want) {
					t.Errorf("Hint = %q, want it to contain %q", userErr.Hint, want)
				}
			}
			if tc.wantLines > 0 {
				if got := strings.Count(userErr.Hint, "datumctl auth switch "); got != tc.wantLines {
					t.Errorf("Hint has %d commands, want %d:\n%s", got, tc.wantLines, userErr.Hint)
				}
			}
			if strings.Contains(userErr.Message, "for this email") {
				t.Errorf("Message still says %q: %q", "for this email", userErr.Message)
			}

			// A failed switch must not change the active session.
			cfg := loadCfg(t)
			if cfg.ActiveSession != soloSession || cfg.CurrentContext != soloCtx {
				t.Errorf("config changed on failure: session=%q context=%q", cfg.ActiveSession, cfg.CurrentContext)
			}
		})
	}
}

// Switching back and forth must flip the active session and current context
// every time, not only on the first switch.
func TestSwitchBackAndForthByEndpoint(t *testing.T) {
	setupSwitchEnv(t, false)

	steps := []struct {
		endpoint    string
		wantSession string
		wantContext string
	}{
		{prodHost, prodSession, prodCtx},
		{stagingHost, stagingSession, stagingCtx},
		{prodHost, prodSession, prodCtx},
	}
	for i, step := range steps {
		if err := runSwitchCmd(t, sharedEmail, "--endpoint", step.endpoint); err != nil {
			t.Fatalf("step %d: switch to %s: %v", i, step.endpoint, err)
		}
		cfg := loadCfg(t)
		if cfg.ActiveSession != step.wantSession || cfg.CurrentContext != step.wantContext {
			t.Errorf("step %d (%s): session=%q context=%q, want %q %q",
				i, step.endpoint, cfg.ActiveSession, cfg.CurrentContext, step.wantSession, step.wantContext)
		}
	}
}

// The endpoint 'datumctl auth list' prints must be accepted by --endpoint.
func TestSwitchAcceptsListedEndpoint(t *testing.T) {
	setupSwitchEnv(t, false)
	cfg := loadCfg(t)
	for _, s := range cfg.SessionByEmail(sharedEmail) {
		listed := datumconfig.StripScheme(s.Endpoint.Server) // what auth list prints
		if err := runSwitchCmd(t, sharedEmail, "--endpoint", listed); err != nil {
			t.Fatalf("--endpoint %q from auth list rejected: %v", listed, err)
		}
		if got := loadCfg(t).ActiveSession; got != s.Name {
			t.Errorf("--endpoint %q: ActiveSession = %q, want %q", listed, got, s.Name)
		}
	}
}

// Switching selects one session without signing out the others.
func TestSwitchLeavesOtherSessionsSignedIn(t *testing.T) {
	setupSwitchEnv(t, false)
	creds := map[string]string{
		"key-prod":    `{"token":"prod"}`,
		"key-staging": `{"token":"staging"}`,
		"key-solo":    `{"token":"solo"}`,
	}
	for k, v := range creds {
		if err := keyring.Set(authutil.ServiceName, k, v); err != nil {
			t.Fatalf("seed keyring: %v", err)
		}
	}
	before := loadCfg(t)

	if err := runSwitchCmd(t, sharedEmail, "--endpoint", stagingHost); err != nil {
		t.Fatalf("switch: %v", err)
	}

	after := loadCfg(t)
	if len(after.Sessions) != len(before.Sessions) {
		t.Fatalf("sessions = %d, want %d", len(after.Sessions), len(before.Sessions))
	}
	for i := range before.Sessions {
		if after.Sessions[i] != before.Sessions[i] {
			t.Errorf("session %q changed:\n got %+v\nwant %+v", before.Sessions[i].Name, after.Sessions[i], before.Sessions[i])
		}
	}
	for k, want := range creds {
		got, err := keyring.Get(authutil.ServiceName, k)
		if err != nil || got != want {
			t.Errorf("keyring %q = %q, %v; want %q", k, got, err, want)
		}
	}
}
