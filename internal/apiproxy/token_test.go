package apiproxy

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	customerrors "go.datum.net/datumctl/internal/errors"
	"golang.org/x/oauth2"
)

// fakeTokenSource simulates authutil's refreshing source: it hands out its
// current token until told it is expired, then "refreshes" by minting the
// configured next token (or failing). The gap between the expired check and
// the refresh is deliberately not atomic, so only the engine's serialization
// makes refresh single-flight — which is exactly what the tests prove.
type fakeTokenSource struct {
	mu         sync.Mutex
	token      string
	next       string
	expired    bool
	refreshErr error

	refreshCount atomic.Int32
	onRefresh    func()
}

func (f *fakeTokenSource) Token() (*oauth2.Token, error) {
	f.mu.Lock()
	expired := f.expired
	current := f.token
	refreshErr := f.refreshErr
	f.mu.Unlock()

	if !expired {
		return &oauth2.Token{AccessToken: current}, nil
	}

	f.refreshCount.Add(1)
	if f.onRefresh != nil {
		f.onRefresh()
	}
	if refreshErr != nil {
		return nil, refreshErr
	}
	f.mu.Lock()
	f.token = f.next
	f.expired = false
	minted := f.token
	f.mu.Unlock()
	return &oauth2.Token{AccessToken: minted}, nil
}

// expire marks the current token stale; the next Token call refreshes to next.
func (f *fakeTokenSource) expire(next string) {
	f.mu.Lock()
	f.expired = true
	f.next = next
	f.mu.Unlock()
}

func TestConcurrentRequestsSingleFlightRefresh(t *testing.T) {
	const parallel = 8
	started := make(chan struct{}, parallel)
	release := make(chan struct{})
	source := &fakeTokenSource{
		token:   "stale",
		next:    "fresh",
		expired: true,
		onRefresh: func() {
			started <- struct{}{}
			<-release
		},
	}
	upstream := newRecordingUpstream(t, nil)
	proxy := newTestProxy(t, upstream.server.URL, func(c *Config) { c.TokenSource = source })

	var wg sync.WaitGroup
	failures := make(chan error, parallel)
	for range parallel {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(proxy.URL + "/apis/foo")
			if err != nil {
				failures <- err
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				failures <- fmt.Errorf("status = %d, want 200", resp.StatusCode)
			}
		}()
	}

	// One refresh is now in flight and blocked; every other request is
	// queued behind it. Releasing it must satisfy them all.
	<-started
	close(release)
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}

	if got := source.refreshCount.Load(); got != 1 {
		t.Fatalf("refresh attempts = %d, want exactly 1 across %d concurrent requests", got, parallel)
	}
	for _, req := range upstream.all() {
		if auth := req.header.Get("Authorization"); auth != "Bearer fresh" {
			t.Errorf("upstream Authorization = %q, want the refreshed token", auth)
		}
	}
}

func TestStreamSurvivesTokenExpiryMidStream(t *testing.T) {
	source := &fakeTokenSource{token: "token-A"}

	var authMu sync.Mutex
	var authHeaders []string
	proceed := make(chan struct{})
	events := []string{
		`{"type":"ADDED","object":{"metadata":{"name":"zone-a"}}}`,
		`{"type":"MODIFIED","object":{"metadata":{"name":"zone-a"}}}`,
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authMu.Lock()
		authHeaders = append(authHeaders, r.Header.Get("Authorization"))
		authMu.Unlock()
		if r.URL.Query().Get("watch") != "true" {
			io.WriteString(w, "ok")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		flusher := w.(http.Flusher)
		io.WriteString(w, events[0]+"\n")
		flusher.Flush()
		<-proceed
		io.WriteString(w, events[1]+"\n")
		flusher.Flush()
	}))
	defer upstream.Close()
	proxy := newTestProxy(t, upstream.URL, func(c *Config) { c.TokenSource = source })

	resp, err := http.Get(proxy.URL + "/apis/foo?watch=true")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	reader := newLineReader(resp.Body)
	if got := reader.next(t); got != events[0]+"\n" {
		t.Fatalf("event 0 = %q", got)
	}

	// The token expires while the stream is open. The stream must not be
	// interrupted: authentication happens at request start only.
	source.expire("token-B")
	proceed <- struct{}{}
	if got := reader.next(t); got != events[1]+"\n" {
		t.Fatalf("event 1 after expiry = %q", got)
	}
	reader.expectEOF(t)

	// The next NEW request refreshes and carries the new token.
	resp2, err := http.Get(proxy.URL + "/apis/foo")
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp2.Body)
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("post-expiry request status = %d, want 200", resp2.StatusCode)
	}

	authMu.Lock()
	defer authMu.Unlock()
	if want := []string{"Bearer token-A", "Bearer token-B"}; len(authHeaders) != 2 ||
		authHeaders[0] != want[0] || authHeaders[1] != want[1] {
		t.Fatalf("upstream Authorization sequence = %q, want %q", authHeaders, want)
	}
	if got := source.refreshCount.Load(); got != 1 {
		t.Fatalf("refresh attempts = %d, want 1", got)
	}
}

func TestRefreshFailureSynthesizes502(t *testing.T) {
	userErr := customerrors.NewUserErrorWithHint(
		"Authentication session has expired or refresh token is no longer valid.",
		"Please re-authenticate using: `datumctl login`",
	)
	source := &fakeTokenSource{token: "stale", expired: true, refreshErr: userErr}
	upstream := newRecordingUpstream(t, nil)
	proxy := newTestProxy(t, upstream.server.URL, func(c *Config) { c.TokenSource = source })

	const parallel = 8
	var wg sync.WaitGroup
	failures := make(chan error, parallel)
	for range parallel {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(proxy.URL + "/apis/foo")
			if err != nil {
				failures <- err
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusBadGateway {
				failures <- fmt.Errorf("status = %d, want 502", resp.StatusCode)
				return
			}
			if resp.Header.Get(proxyErrorHeader) != "true" {
				failures <- errors.New("502 must carry the proxy error marker")
				return
			}
			if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
				failures <- fmt.Errorf("Content-Type = %q, want application/json", ct)
				return
			}
			status := kubeStatus{}
			if err := jsonDecode(resp.Body, &status); err != nil {
				failures <- err
				return
			}
			if status.Kind != "Status" || status.APIVersion != "v1" || status.Status != "Failure" ||
				status.Code != http.StatusBadGateway {
				failures <- fmt.Errorf("body = %+v, want a 502 Failure Status", status)
				return
			}
			if status.Reason != "ProxyAuthenticationFailed" {
				failures <- fmt.Errorf("Status.reason = %q, want ProxyAuthenticationFailed", status.Reason)
				return
			}
			if !strings.Contains(status.Message, userErr.Message) || !strings.Contains(status.Message, userErr.Hint) {
				failures <- fmt.Errorf("Status.message = %q, want the UserError message and hint", status.Message)
			}
		}()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}

	if got := source.refreshCount.Load(); got != 1 {
		t.Fatalf("refresh attempts = %d, want 1 per cooldown window under concurrent load", got)
	}
	if n := upstream.count(); n != 0 {
		t.Fatalf("upstream saw %d request(s); token failures must never reach it", n)
	}
}

func TestCooldownWindowGatesRefreshRetries(t *testing.T) {
	source := &fakeTokenSource{token: "stale", expired: true, refreshErr: errors.New("boom")}
	current := time.Unix(1_000_000, 0)
	cooled := newCooldownTokenSource(source, 5*time.Second, io.Discard)
	cooled.now = func() time.Time { return current }

	_, err := cooled.Token()
	if err == nil {
		t.Fatal("want an error from the failing source")
	}
	var refreshErr *tokenRefreshError
	if !errors.As(err, &refreshErr) {
		t.Fatalf("error %T does not mark a token refresh failure", err)
	}
	if got := source.refreshCount.Load(); got != 1 {
		t.Fatalf("refresh attempts = %d, want 1", got)
	}

	// Inside the window: fail fast with the cached error, no new attempt.
	if _, err2 := cooled.Token(); err2 != err {
		t.Fatalf("in-window error = %v, want the cached %v", err2, err)
	}
	current = current.Add(4 * time.Second)
	if _, _ = cooled.Token(); source.refreshCount.Load() != 1 {
		t.Fatalf("refresh attempts = %d after 4s, want still 1", source.refreshCount.Load())
	}

	// Past the window: one new attempt is allowed — and it can recover.
	current = current.Add(2 * time.Second)
	source.mu.Lock()
	source.refreshErr = nil
	source.next = "fresh"
	source.mu.Unlock()
	token, err := cooled.Token()
	if err != nil {
		t.Fatalf("post-window Token: %v", err)
	}
	if token.AccessToken != "fresh" {
		t.Fatalf("post-window token = %q, want %q", token.AccessToken, "fresh")
	}
	if got := source.refreshCount.Load(); got != 2 {
		t.Fatalf("refresh attempts = %d, want 2 (one per window)", got)
	}
}

func jsonDecode(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}
