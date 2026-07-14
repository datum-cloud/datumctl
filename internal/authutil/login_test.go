package authutil

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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
