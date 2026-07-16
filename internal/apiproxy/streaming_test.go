package apiproxy

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// lineReader delivers response lines on a channel so tests can assert each
// line the moment it clears the proxy. The timeouts below are hang guards
// that turn a buffering proxy into a loud failure, not timing assertions:
// the pass path never waits on a clock.
type lineReader struct {
	lines chan string
	errs  chan error
}

func newLineReader(r io.Reader) *lineReader {
	lr := &lineReader{lines: make(chan string), errs: make(chan error, 1)}
	go func() {
		reader := bufio.NewReader(r)
		for {
			line, err := reader.ReadString('\n')
			if line != "" {
				lr.lines <- line
			}
			if err != nil {
				lr.errs <- err
				return
			}
		}
	}()
	return lr
}

func (lr *lineReader) next(t *testing.T) string {
	t.Helper()
	select {
	case line := <-lr.lines:
		return line
	case err := <-lr.errs:
		t.Fatalf("stream ended before the expected line: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for a line: the proxy is buffering the stream")
	}
	return ""
}

func (lr *lineReader) expectEOF(t *testing.T) {
	t.Helper()
	select {
	case err := <-lr.errs:
		if err != io.EOF {
			t.Fatalf("stream ended with %v, want io.EOF", err)
		}
	case line := <-lr.lines:
		t.Fatalf("unexpected extra line %q", line)
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for stream end")
	}
}

// TestWatchStreamingIsUnbuffered is the load-bearing streaming test. The
// upstream writes one watch event, flushes, then blocks until the test has
// read that event through the proxy. Receiving event N while the upstream is
// still blocked before producing event N+1 proves zero proxy-side buffering
// with no timing dependence. It runs twice against the same proxy to cover
// the repeat-watch case.
func TestWatchStreamingIsUnbuffered(t *testing.T) {
	events := []string{
		`{"type":"ADDED","object":{"metadata":{"name":"zone-a"}}}`,
		`{"type":"MODIFIED","object":{"metadata":{"name":"zone-a"}}}`,
		`{"type":"DELETED","object":{"metadata":{"name":"zone-a"}}}`,
	}
	proceed := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		flusher := w.(http.Flusher)
		for i, event := range events {
			io.WriteString(w, event+"\n")
			flusher.Flush()
			// Block until the test confirms the event arrived through the
			// proxy: a buffering proxy can never unblock this.
			if i < len(events)-1 {
				<-proceed
			}
		}
	}))
	defer upstream.Close()
	proxy := newTestProxy(t, upstream.URL)

	runWatch := func() {
		resp, err := http.Get(proxy.URL + "/apis/networking.datumapis.com/v1alpha/dnszones?watch=true")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		if cl := resp.Header.Get("Content-Length"); cl != "" {
			t.Fatalf("watch response carries Content-Length %q; it must stream", cl)
		}
		reader := newLineReader(resp.Body)
		for i, want := range events {
			if got := reader.next(t); got != want+"\n" {
				t.Fatalf("event %d = %q, want %q", i, got, want)
			}
			if i < len(events)-1 {
				proceed <- struct{}{}
			}
		}
		reader.expectEOF(t)
	}

	runWatch()
	// A second watch through the same proxy must stream just as well.
	runWatch()
}

func TestSSEStreamingIsUnbuffered(t *testing.T) {
	events := []string{"one", "two", "three"}
	proceed := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		for i, event := range events {
			fmt.Fprintf(w, "data: %s\n\n", event)
			flusher.Flush()
			if i < len(events)-1 {
				<-proceed
			}
		}
	}))
	defer upstream.Close()
	proxy := newTestProxy(t, upstream.URL)

	resp, err := http.Get(proxy.URL + "/events")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream passthrough", ct)
	}

	reader := newLineReader(resp.Body)
	for i, event := range events {
		if got, want := reader.next(t), "data: "+event+"\n"; got != want {
			t.Fatalf("event %d = %q, want %q", i, got, want)
		}
		if got := reader.next(t); got != "\n" {
			t.Fatalf("event %d separator = %q, want blank line", i, got)
		}
		if i < len(events)-1 {
			proceed <- struct{}{}
		}
	}
	reader.expectEOF(t)
}

// TestWatchStreamingIsUnbufferedThroughScopedProxy re-runs the channel-gated
// watch streaming test against a scoped proxy: the upstream root carries a
// control-plane path prefix, and each event must still clear the proxy while
// the upstream is blocked before producing the next one.
func TestWatchStreamingIsUnbufferedThroughScopedProxy(t *testing.T) {
	const prefix = "/apis/resourcemanager.miloapis.com/v1alpha1/projects/my-project/control-plane"
	events := []string{
		`{"type":"ADDED","object":{"metadata":{"name":"zone-a"}}}`,
		`{"type":"MODIFIED","object":{"metadata":{"name":"zone-a"}}}`,
	}
	proceed := make(chan struct{})
	var gotPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		flusher := w.(http.Flusher)
		for i, event := range events {
			io.WriteString(w, event+"\n")
			flusher.Flush()
			if i < len(events)-1 {
				<-proceed
			}
		}
	}))
	defer upstream.Close()
	proxy := newTestProxy(t, upstream.URL+prefix)

	resp, err := http.Get(proxy.URL + "/apis/networking.datumapis.com/v1alpha/dnszones?watch=true")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	reader := newLineReader(resp.Body)
	for i, want := range events {
		if got := reader.next(t); got != want+"\n" {
			t.Fatalf("event %d = %q, want %q", i, got, want)
		}
		if i < len(events)-1 {
			proceed <- struct{}{}
		}
	}
	reader.expectEOF(t)

	if want := prefix + "/apis/networking.datumapis.com/v1alpha/dnszones"; gotPath != want {
		t.Errorf("upstream path = %q, want %q (watch path joined under the scoped prefix)", gotPath, want)
	}
}

// TestSlowTrickleSmallWritesAreUnbuffered covers the slow-trickle streaming
// variant: the upstream emits tiny sub-line fragments, flushing after each,
// and blocks until the client has observed the fragment through the proxy. A
// proxy that coalesces or buffers small writes can never unblock it.
func TestSlowTrickleSmallWritesAreUnbuffered(t *testing.T) {
	fragments := []string{"{", `"type":`, `"ADD`, `ED"`, "}\n"}
	proceed := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		flusher := w.(http.Flusher)
		for i, fragment := range fragments {
			io.WriteString(w, fragment)
			flusher.Flush()
			if i < len(fragments)-1 {
				<-proceed
			}
		}
	}))
	defer upstream.Close()
	proxy := newTestProxy(t, upstream.URL)

	resp, err := http.Get(proxy.URL + "/apis/foo?watch=true")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	for i, fragment := range fragments {
		// The upstream is blocked before producing the next fragment, so at
		// most len(fragment) bytes can be in flight; ReadFull returning is
		// itself the proof this fragment cleared the proxy unbuffered.
		buf := make([]byte, len(fragment))
		readDone := make(chan error, 1)
		go func() {
			_, err := io.ReadFull(resp.Body, buf)
			readDone <- err
		}()
		select {
		case err := <-readDone:
			if err != nil {
				t.Fatalf("fragment %d: read error: %v", i, err)
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("fragment %d: timed out — the proxy is buffering small writes", i)
		}
		if got := string(buf); got != fragment {
			t.Fatalf("fragment %d = %q, want %q", i, got, fragment)
		}
		if i < len(fragments)-1 {
			proceed <- struct{}{}
		}
	}
	if n, err := resp.Body.Read(make([]byte, 1)); err != io.EOF {
		t.Fatalf("after the last fragment: read %d bytes, err %v, want io.EOF", n, err)
	}
}

func TestUpstreamClosesMidStream(t *testing.T) {
	event := `{"type":"ADDED","object":{"metadata":{"name":"zone-a"}}}`
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, event+"\n")
		w.(http.Flusher).Flush()
		// Returning here cuts the stream: watch timeout, token expiry,
		// anything. The client must see a normal stream end.
	}))
	defer upstream.Close()
	proxy := newTestProxy(t, upstream.URL)

	resp, err := http.Get(proxy.URL + "/apis/foo?watch=true")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	reader := newLineReader(resp.Body)
	if got := reader.next(t); got != event+"\n" {
		t.Fatalf("event = %q, want %q", got, event)
	}
	reader.expectEOF(t)
}
