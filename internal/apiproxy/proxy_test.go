package apiproxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"golang.org/x/oauth2"
)

// corsHeaders must never appear on any proxy response, synthesized or
// passed through.
var corsHeaders = []string{
	"Access-Control-Allow-Origin",
	"Access-Control-Allow-Methods",
	"Access-Control-Allow-Headers",
	"Access-Control-Allow-Credentials",
}

func assertNoCORS(t *testing.T, h http.Header) {
	t.Helper()
	for _, name := range corsHeaders {
		if v := h.Get(name); v != "" {
			t.Errorf("response carries CORS header %s: %q; the proxy must never emit CORS headers", name, v)
		}
	}
}

// upstreamRequest is one request as observed by the fake upstream.
type upstreamRequest struct {
	method   string
	path     string
	rawQuery string
	host     string
	header   http.Header
	body     []byte
}

// recordingUpstream is an httptest upstream that records every request it
// receives before delegating to respond (200 "ok" if nil).
type recordingUpstream struct {
	server *httptest.Server

	mu       sync.Mutex
	requests []upstreamRequest
}

func newRecordingUpstream(t *testing.T, respond http.HandlerFunc) *recordingUpstream {
	t.Helper()
	u := &recordingUpstream{}
	u.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		u.mu.Lock()
		u.requests = append(u.requests, upstreamRequest{
			method:   r.Method,
			path:     r.URL.Path,
			rawQuery: r.URL.RawQuery,
			host:     r.Host,
			header:   r.Header.Clone(),
			body:     body,
		})
		u.mu.Unlock()
		if respond != nil {
			respond(w, r)
			return
		}
		io.WriteString(w, "ok")
	}))
	t.Cleanup(u.server.Close)
	return u
}

func (u *recordingUpstream) count() int {
	u.mu.Lock()
	defer u.mu.Unlock()
	return len(u.requests)
}

func (u *recordingUpstream) last(t *testing.T) upstreamRequest {
	t.Helper()
	u.mu.Lock()
	defer u.mu.Unlock()
	if len(u.requests) == 0 {
		t.Fatal("upstream received no requests")
	}
	return u.requests[len(u.requests)-1]
}

func (u *recordingUpstream) all() []upstreamRequest {
	u.mu.Lock()
	defer u.mu.Unlock()
	return append([]upstreamRequest(nil), u.requests...)
}

func parseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}
	return u
}

func staticToken(token string) oauth2.TokenSource {
	return oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
}

// newTestProxy builds a proxy engine against upstream and serves its handler
// on an httptest server (which binds 127.0.0.1, so Host validation passes).
func newTestProxy(t *testing.T, upstream string, mutate ...func(*Config)) *httptest.Server {
	t.Helper()
	cfg := Config{
		Upstream:    parseURL(t, upstream),
		TokenSource: staticToken("good-token"),
	}
	for _, m := range mutate {
		m(&cfg)
	}
	server, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	proxy := httptest.NewServer(server.Handler())
	t.Cleanup(proxy.Close)
	return proxy
}

func decodeStatus(t *testing.T, body io.Reader) kubeStatus {
	t.Helper()
	var status kubeStatus
	if err := json.NewDecoder(body).Decode(&status); err != nil {
		t.Fatalf("decode Status body: %v", err)
	}
	return status
}

func TestPassthrough(t *testing.T) {
	upstream := newRecordingUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Upstream", "yes")
		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, "created")
	})
	proxy := newTestProxy(t, upstream.server.URL)

	req, err := http.NewRequest(http.MethodPost,
		proxy.URL+"/apis/example.com/v1/things?labelSelector=a%3Db&limit=2",
		strings.NewReader("hello body"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer locally-smuggled")
	req.Header.Set("X-Request-ID", "rid-123")
	req.Header.Set("Proxy-Connection", "keep-alive")
	req.Header.Set("Keep-Alive", "timeout=5")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	if string(body) != "created" {
		t.Errorf("body = %q, want %q", body, "created")
	}
	if resp.Header.Get("X-Upstream") != "yes" {
		t.Error("upstream response header did not pass through")
	}
	if resp.Header.Get(proxyErrorHeader) != "" {
		t.Error("passthrough response must not carry the proxy error marker")
	}
	assertNoCORS(t, resp.Header)

	got := upstream.last(t)
	if got.method != http.MethodPost {
		t.Errorf("upstream method = %q, want POST", got.method)
	}
	if got.path != "/apis/example.com/v1/things" {
		t.Errorf("upstream path = %q", got.path)
	}
	if got.rawQuery != "labelSelector=a%3Db&limit=2" {
		t.Errorf("upstream query = %q, want it forwarded verbatim", got.rawQuery)
	}
	if string(got.body) != "hello body" {
		t.Errorf("upstream body = %q", got.body)
	}
	if auth := got.header.Get("Authorization"); auth != "Bearer good-token" {
		t.Errorf("upstream Authorization = %q, want the injected token, never the inbound one", auth)
	}
	if rid := got.header.Get("X-Request-ID"); rid != "rid-123" {
		t.Errorf("upstream X-Request-ID = %q, want passthrough", rid)
	}
	for _, hop := range []string{"Proxy-Connection", "Keep-Alive"} {
		if v := got.header.Get(hop); v != "" {
			t.Errorf("hop-by-hop header %s reached upstream: %q", hop, v)
		}
	}
	wantHost := parseURL(t, upstream.server.URL).Host
	if got.host != wantHost {
		t.Errorf("upstream Host = %q, want %q", got.host, wantHost)
	}
}

func TestHostValidationRejectsNonLocal(t *testing.T) {
	upstream := newRecordingUpstream(t, nil)
	proxy := newTestProxy(t, upstream.server.URL)

	req, err := http.NewRequest(http.MethodGet, proxy.URL+"/apis/foo", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "evil.example"

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
	if resp.Header.Get(proxyErrorHeader) != "true" {
		t.Error("proxy-synthesized 403 must carry the proxy error marker")
	}
	assertNoCORS(t, resp.Header)
	if status := decodeStatus(t, resp.Body); status.Kind != "Status" || status.Code != http.StatusForbidden {
		t.Errorf("403 body = %+v, want a Status object with code 403", status)
	}
	if n := upstream.count(); n != 0 {
		t.Fatalf("upstream saw %d request(s); a rejected Host must never reach it", n)
	}
}

func TestHostValidationAllowsLocalAliases(t *testing.T) {
	upstream := newRecordingUpstream(t, nil)
	proxy := newTestProxy(t, upstream.server.URL)

	for _, host := range []string{
		"localhost", "localhost:9999", "LOCALHOST",
		"127.0.0.1", "127.0.0.1:8001",
		"[::1]", "[::1]:8001",
	} {
		req, err := http.NewRequest(http.MethodGet, proxy.URL+"/apis/foo", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Host = host
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Host %q: %v", host, err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Host %q: status = %d, want 200", host, resp.StatusCode)
		}
	}
}

func TestAllowedHost(t *testing.T) {
	cases := []struct {
		host string
		want bool
	}{
		{"localhost", true},
		{"localhost:8001", true},
		{"LocalHost:8001", true},
		{"127.0.0.1", true},
		{"127.0.0.1:52347", true},
		{"[::1]", true},
		{"[::1]:8001", true},
		{"::1", true},
		{"evil.example", false},
		{"evil.example:80", false},
		{"127.0.0.2", false},
		{"127.0.0.1.evil.example", false},
		{"localhost.evil.example", false},
		{"[2001:db8::1]:8001", false},
		{"", false},
	}
	for _, c := range cases {
		if got := allowedHost(c.host); got != c.want {
			t.Errorf("allowedHost(%q) = %v, want %v", c.host, got, c.want)
		}
	}
}

func TestUpstreamAuthErrorsPassThroughUnmarked(t *testing.T) {
	for _, code := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			upstreamBody := fmt.Sprintf(`{"kind":"Status","apiVersion":"v1","status":"Failure","code":%d,"reason":"upstream-said-no"}`, code)
			upstream := newRecordingUpstream(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(code)
				io.WriteString(w, upstreamBody)
			})
			proxy := newTestProxy(t, upstream.server.URL)

			resp, err := http.Get(proxy.URL + "/apis/foo")
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != code {
				t.Fatalf("status = %d, want %d", resp.StatusCode, code)
			}
			if string(body) != upstreamBody {
				t.Errorf("body = %q, want the upstream body byte-for-byte", body)
			}
			if resp.Header.Get(proxyErrorHeader) != "" {
				t.Errorf("upstream %d must pass through without the proxy error marker", code)
			}
			assertNoCORS(t, resp.Header)
		})
	}
}

func TestUpstreamUnreachableSynthesizes502(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	deadAddr := listener.Addr().String()
	listener.Close()

	proxy := newTestProxy(t, "http://"+deadAddr)
	resp, err := http.Get(proxy.URL + "/apis/foo")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", resp.StatusCode)
	}
	if resp.Header.Get(proxyErrorHeader) != "true" {
		t.Error("synthesized 502 must carry the proxy error marker")
	}
	assertNoCORS(t, resp.Header)
	status := decodeStatus(t, resp.Body)
	if status.Kind != "Status" || status.APIVersion != "v1" || status.Status != "Failure" {
		t.Errorf("502 body = %+v, want kind Status / apiVersion v1 / status Failure", status)
	}
	if status.Code != http.StatusBadGateway {
		t.Errorf("Status.code = %d, want 502", status.Code)
	}
	if status.Reason != "ProxyUpstreamError" {
		t.Errorf("Status.reason = %q, want ProxyUpstreamError for a non-token failure", status.Reason)
	}
}

func TestScopedUpstreamJoinsPathPrefix(t *testing.T) {
	upstream := newRecordingUpstream(t, nil)
	prefix := "/apis/resourcemanager.miloapis.com/v1alpha1/projects/x/control-plane"
	proxy := newTestProxy(t, upstream.server.URL+prefix)

	resp, err := http.Get(proxy.URL + "/apis/foo?watch=true")
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	got := upstream.last(t)
	if want := prefix + "/apis/foo"; got.path != want {
		t.Errorf("upstream path = %q, want %q (local path joined under the scoped prefix)", got.path, want)
	}
	if got.rawQuery != "watch=true" {
		t.Errorf("upstream query = %q, want %q", got.rawQuery, "watch=true")
	}
}

func TestServerSettingsAreStreamSafe(t *testing.T) {
	server, err := New(Config{
		Upstream:    parseURL(t, "https://api.example.test"),
		TokenSource: staticToken("tok"),
	})
	if err != nil {
		t.Fatal(err)
	}
	hs := server.httpServer
	if hs.ReadHeaderTimeout != readHeaderTimeout {
		t.Errorf("ReadHeaderTimeout = %v, want %v", hs.ReadHeaderTimeout, readHeaderTimeout)
	}
	if hs.IdleTimeout == 0 {
		t.Error("IdleTimeout must be set for keep-alive hygiene")
	}
	if hs.ReadTimeout != 0 || hs.WriteTimeout != 0 {
		t.Errorf("ReadTimeout/WriteTimeout = %v/%v, must stay zero — they would kill streams",
			hs.ReadTimeout, hs.WriteTimeout)
	}
}

func TestServeAndShutdown(t *testing.T) {
	upstream := newRecordingUpstream(t, nil)
	server, err := New(Config{
		Upstream:    parseURL(t, upstream.server.URL),
		TokenSource: staticToken("tok"),
	})
	if err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()

	resp, err := http.Get("http://" + listener.Addr().String() + "/apis/foo")
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	if err := <-served; !errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("Serve returned %v, want http.ErrServerClosed", err)
	}
}

func TestNewValidatesConfig(t *testing.T) {
	valid := func() Config {
		return Config{
			Upstream:    parseURL(t, "https://api.example.test"),
			TokenSource: staticToken("tok"),
		}
	}

	cfg := valid()
	cfg.Upstream = nil
	if _, err := New(cfg); err == nil {
		t.Error("New must reject a nil Upstream")
	}

	cfg = valid()
	cfg.Upstream = parseURL(t, "/relative/only")
	if _, err := New(cfg); err == nil {
		t.Error("New must reject a relative Upstream URL")
	}

	cfg = valid()
	cfg.TokenSource = nil
	if _, err := New(cfg); err == nil {
		t.Error("New must reject a nil TokenSource")
	}

	if _, err := New(valid()); err != nil {
		t.Errorf("New with a valid config: %v", err)
	}
}
