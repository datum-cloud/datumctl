package api

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/client"
	"go.datum.net/datumctl/internal/datumconfig"
	"go.datum.net/datumctl/internal/keyring"
)

// TestProxyLifecycle drives the real 'datumctl api proxy' command end to end,
// hermetically: HOME points at a temp dir (so the config and any credential
// fallback file live there), the keyring is mocked, and the session endpoint
// is an httptest upstream. It pins the readiness contract — with --port 0 the
// first stdout line parses as a URL and a request to it succeeds — and that
// SIGINT terminates the command within the shutdown grace period.
func TestProxyLifecycle(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	keyring.MockInit()

	const userKey = "maya@datum.net@auth.datum.net"

	var authHeader string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		io.WriteString(w, "ok")
	}))
	defer upstream.Close()

	// Config fixture: one active session whose endpoint is the upstream.
	cfg := datumconfig.NewV1Beta1()
	cfg.Sessions = []datumconfig.Session{{
		Name:      "maya@datum.net@api.datum.net",
		UserKey:   userKey,
		UserEmail: "maya@datum.net",
		Endpoint: datumconfig.Endpoint{
			Server:       upstream.URL,
			AuthHostname: "auth.datum.net",
		},
	}}
	cfg.ActiveSession = "maya@datum.net@api.datum.net"
	if err := datumconfig.SaveV1Beta1(cfg); err != nil {
		t.Fatalf("save config fixture: %v", err)
	}

	// Keyring fixture: a token valid well past the test, so no refresh runs.
	blob, err := json.Marshal(authutil.StoredCredentials{
		Hostname:    "auth.datum.net",
		APIHostname: "api.datum.net",
		UserEmail:   "maya@datum.net",
		Token: &oauth2.Token{
			AccessToken: "lifecycle-token",
			Expiry:      time.Now().Add(time.Hour),
		},
	})
	if err != nil {
		t.Fatalf("marshal creds: %v", err)
	}
	if err := keyring.Set(authutil.ServiceName, userKey, string(blob)); err != nil {
		t.Fatalf("seed keyring: %v", err)
	}

	factory := &client.DatumCloudFactory{ConfigFlags: &client.CustomConfigFlags{}}
	cmd := proxyCommand(factory)
	stdoutR, stdoutW := io.Pipe()
	cmd.SetOut(stdoutW)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--port", "0", "--quiet"})

	done := make(chan error, 1)
	go func() { done <- cmd.Execute() }()

	// Readiness contract: the first stdout line is the bare proxy URL,
	// printed once the listener is serving (and the signal handler is armed).
	lines := make(chan string, 1)
	go func() {
		line, _ := bufio.NewReader(stdoutR).ReadString('\n')
		lines <- line
	}()
	var readyLine string
	select {
	case readyLine = <-lines:
	case err := <-done:
		t.Fatalf("proxy exited before printing its URL: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for the readiness line on stdout")
	}
	proxyURL, err := url.Parse(strings.TrimSpace(readyLine))
	if err != nil {
		t.Fatalf("first stdout line %q does not parse as a URL: %v", readyLine, err)
	}
	if proxyURL.Scheme != "http" || !strings.HasPrefix(proxyURL.Host, "127.0.0.1:") {
		t.Fatalf("proxy URL = %q, want http://127.0.0.1:<port>", proxyURL)
	}

	// A request to the printed URL succeeds and carries the injected token.
	resp, err := http.Get(proxyURL.String() + "/apis/resourcemanager.miloapis.com/v1alpha1/organizations")
	if err != nil {
		t.Fatalf("request through the proxy: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if string(body) != "ok" {
		t.Fatalf("body = %q, want the upstream body", body)
	}
	if authHeader != "Bearer lifecycle-token" {
		t.Fatalf("upstream Authorization = %q, want the injected session token", authHeader)
	}

	// SIGINT terminates within the grace period with a clean exit.
	if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
		t.Fatalf("send SIGINT: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("proxy exited with error after SIGINT: %v", err)
		}
	case <-time.After(shutdownGrace + 5*time.Second):
		t.Fatal("proxy did not terminate within the shutdown grace period after SIGINT")
	}
	stdoutW.Close()
}
