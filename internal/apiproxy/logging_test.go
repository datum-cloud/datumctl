package apiproxy

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// doRecorded drives the handler synchronously with a recorder, so by the
// time it returns every log line for the request has been written.
func doRecorded(t *testing.T, h http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func logLines(buf *bytes.Buffer) []string {
	var lines []string
	for _, line := range strings.Split(buf.String(), "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func newLoggingProxy(t *testing.T, upstream string, buf *bytes.Buffer, quiet bool, source ...func(*Config)) http.Handler {
	t.Helper()
	cfg := Config{
		Upstream:    parseURL(t, upstream),
		TokenSource: staticToken("tok"),
		LogWriter:   buf,
		Quiet:       quiet,
	}
	for _, m := range source {
		m(&cfg)
	}
	server, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return server.Handler()
}

func TestRequestLogLineRedactsTokenQueryValues(t *testing.T) {
	upstream := newRecordingUpstream(t, nil) // responds 200 "ok" with Content-Length
	var buf bytes.Buffer
	handler := newLoggingProxy(t, upstream.server.URL, &buf, false)

	rec := doRecorded(t, handler,
		"http://127.0.0.1:8001/apis/foo?watch=true&access_token=supersecret&TOKEN=alsosecret&authorization=hush&limit=5")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	lines := logLines(&buf)
	if len(lines) != 1 {
		t.Fatalf("log lines = %q, want exactly one for a non-streaming request", lines)
	}
	line := lines[0]
	for _, want := range []string{
		"GET", "/apis/foo?", "watch=true", "limit=5", " 200 ",
		"access_token=REDACTED", "TOKEN=REDACTED", "authorization=REDACTED",
		" 2B", // "ok"
	} {
		if !strings.Contains(line, want) {
			t.Errorf("log line %q missing %q", line, want)
		}
	}
	for _, banned := range []string{"supersecret", "alsosecret", "hush", "Bearer", "…streaming"} {
		if strings.Contains(line, banned) {
			t.Errorf("log line %q leaks %q", line, banned)
		}
	}
}

func TestStreamingLogsHeaderArrivalAndCompletion(t *testing.T) {
	events := []string{`{"type":"ADDED"}`, `{"type":"DELETED"}`}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		flusher := w.(http.Flusher)
		for _, event := range events {
			io.WriteString(w, event+"\n")
			flusher.Flush()
		}
	}))
	defer upstream.Close()
	var buf bytes.Buffer
	handler := newLoggingProxy(t, upstream.URL, &buf, false)

	rec := doRecorded(t, handler, "http://127.0.0.1:8001/apis/foo?watch=true")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	lines := logLines(&buf)
	if len(lines) != 2 {
		t.Fatalf("log lines = %q, want a streaming-start line and a completion line", lines)
	}
	if !strings.Contains(lines[0], "…streaming") || !strings.Contains(lines[0], " 200") {
		t.Errorf("start line %q must mark the stream and its status", lines[0])
	}
	totalBytes := 0
	for _, event := range events {
		totalBytes += len(event) + 1
	}
	if want := formatBytes(int64(totalBytes)); !strings.Contains(lines[1], " "+want) {
		t.Errorf("completion line %q missing total bytes %q", lines[1], want)
	}
	if strings.Contains(lines[1], "…streaming") {
		t.Errorf("completion line %q must carry totals, not the streaming marker", lines[1])
	}
}

func TestQuietSuppressesRequestLines(t *testing.T) {
	upstream := newRecordingUpstream(t, nil)
	var buf bytes.Buffer
	handler := newLoggingProxy(t, upstream.server.URL, &buf, true)

	rec := doRecorded(t, handler, "http://127.0.0.1:8001/apis/foo?watch=true")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if buf.Len() != 0 {
		t.Fatalf("quiet mode logged %q, want nothing", buf.String())
	}
}

func TestRefreshFailureLoggedOncePerWindowEvenWhenQuiet(t *testing.T) {
	source := &fakeTokenSource{token: "stale", expired: true, refreshErr: errTokenDead}
	upstream := newRecordingUpstream(t, nil)
	var buf bytes.Buffer
	handler := newLoggingProxy(t, upstream.server.URL, &buf, true,
		func(c *Config) { c.TokenSource = source })

	for range 3 {
		rec := doRecorded(t, handler, "http://127.0.0.1:8001/apis/foo")
		if rec.Code != http.StatusBadGateway {
			t.Fatalf("status = %d, want 502", rec.Code)
		}
	}

	lines := logLines(&buf)
	if len(lines) != 1 || !strings.Contains(lines[0], "token refresh failed") {
		t.Fatalf("log lines = %q, want exactly one refresh-failure line for the whole cooldown window", lines)
	}
	if strings.Contains(buf.String(), "/apis/foo") {
		t.Errorf("quiet mode must not log request lines, got %q", buf.String())
	}
}

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{143 * time.Millisecond, "143ms"},
		{0, "0ms"},
		{3200 * time.Millisecond, "3.2s"},
		{time.Minute, "60.0s"},
	}
	for _, c := range cases {
		if got := formatDuration(c.d); got != c.want {
			t.Errorf("formatDuration(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		n    int64
		want string
	}{
		{0, "0B"},
		{999, "999B"},
		{8100, "8.1kB"},
		{1500000, "1.5MB"},
		{2000000000, "2.0GB"},
	}
	for _, c := range cases {
		if got := formatBytes(c.n); got != c.want {
			t.Errorf("formatBytes(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

func TestRedactedRequestPath(t *testing.T) {
	cases := []struct {
		target string
		want   string
	}{
		{"http://localhost/apis/foo", "/apis/foo"},
		{"http://localhost/apis/foo?watch=true", "/apis/foo?watch=true"},
		{
			"http://localhost/apis/foo?a=1&access_token=x&b=2",
			"/apis/foo?a=1&access_token=REDACTED&b=2",
		},
		{
			// Order preserved, case-insensitive keys, valueless keys untouched.
			"http://localhost/apis/foo?Token=x&watch&AUTHORIZATION=y",
			"/apis/foo?Token=REDACTED&watch&AUTHORIZATION=REDACTED",
		},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodGet, c.target, nil)
		if got := redactedRequestPath(req); got != c.want {
			t.Errorf("redactedRequestPath(%q) = %q, want %q", c.target, got, c.want)
		}
	}
}

var errTokenDead = errors.New("refresh token is dead")
