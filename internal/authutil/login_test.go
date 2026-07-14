package authutil

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

// TestPollDeviceToken_ContextCancel verifies that a canceled context aborts the
// device-code poll promptly with context.Canceled. This is the cancellation
// path that #252 makes reachable: before that fix the login flow ran with a
// non-cancellable context.Background(), so ^C/^D could never interrupt the
// wait. The token server here always answers "authorization_pending", so the
// only way the poll returns is via context cancellation.
func TestPollDeviceToken_ContextCancel(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"authorization_pending"}`))
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel shortly after the poll starts, while it is sleeping between
	// attempts (interval is 1s below).
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	// intervalSeconds=1 keeps the between-attempt sleep long enough that the
	// select lands on ctx.Done() rather than the timer.
	token, err := pollDeviceToken(ctx, srv.URL, "client-id", "device-code", 1, 0)
	elapsed := time.Since(start)

	if token != nil {
		t.Fatalf("expected nil token on cancellation, got %#v", token)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if elapsed > 900*time.Millisecond {
		t.Fatalf("poll did not abort promptly on cancel: took %s", elapsed)
	}
	if atomic.LoadInt32(&calls) == 0 {
		t.Fatal("expected at least one poll attempt before cancel")
	}
}

// TestWatchReaderEOF_CancelsOnEOF verifies the ^D fix: reaching EOF on the
// watched stream cancels the derived context, which aborts the login wait.
func TestWatchReaderEOF_CancelsOnEOF(t *testing.T) {
	r, w := mustPipe(t)
	defer r.Close()

	ctx, stop := watchReaderEOF(context.Background(), r)
	defer stop()

	// Closing the write end delivers EOF to the reader, as ^D does at a tty.
	_ = w.Close()

	select {
	case <-ctx.Done():
		if !errors.Is(ctx.Err(), context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", ctx.Err())
		}
	case <-time.After(3 * time.Second):
		t.Fatal("context was not canceled after EOF")
	}
}

// TestWatchReaderEOF_DiscardsInputThenEOF verifies that input received during
// the wait is consumed and discarded, and a subsequent EOF still cancels.
func TestWatchReaderEOF_DiscardsInputThenEOF(t *testing.T) {
	r, w := mustPipe(t)
	defer r.Close()

	ctx, stop := watchReaderEOF(context.Background(), r)
	defer stop()

	if _, err := w.Write([]byte("noise typed during the wait\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	_ = w.Close()

	select {
	case <-ctx.Done():
		if !errors.Is(ctx.Err(), context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", ctx.Err())
		}
	case <-time.After(3 * time.Second):
		t.Fatal("context was not canceled after input+EOF")
	}
}

// TestWatchReaderEOF_NoCancelBeforeEOFOrStop verifies the watcher does not
// cancel while the stream is open and idle (no ^D), so a successful login is
// not spuriously aborted; stop() then releases it.
func TestWatchReaderEOF_NoCancelBeforeEOFOrStop(t *testing.T) {
	r, w := mustPipe(t)
	defer r.Close()
	defer w.Close()

	ctx, stop := watchReaderEOF(context.Background(), r)

	select {
	case <-ctx.Done():
		t.Fatal("context canceled while stream open and idle")
	case <-time.After(600 * time.Millisecond): // spans a couple read-deadline cycles
	}

	stop()
	select {
	case <-ctx.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("stop() did not cancel the context")
	}
}

func mustPipe(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	return r, w
}
