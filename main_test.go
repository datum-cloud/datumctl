package main

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// TestExitCodeForError_Interrupt verifies that a canceled context — how a
// ^C/SIGTERM interrupt surfaces once the signal-derived context is wired into
// the root command — maps to a quiet exit with code 130, including when the
// cancellation is wrapped by intervening error handling.
func TestExitCodeForError_Interrupt(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{"direct", context.Canceled},
		{"wrapped", fmt.Errorf("authentication failed: %w", context.Canceled)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, interrupted := exitCodeForError(tc.err)
			if !interrupted {
				t.Fatalf("expected interrupted=true for %v", tc.err)
			}
			if code != 130 {
				t.Fatalf("expected exit code 130, got %d", code)
			}
		})
	}
}

// TestExitCodeForError_NonInterrupt verifies a normal error is not treated as
// an interrupt and yields a non-zero exit code.
func TestExitCodeForError_NonInterrupt(t *testing.T) {
	code, interrupted := exitCodeForError(errors.New("boom"))
	if interrupted {
		t.Fatal("expected interrupted=false for a non-cancellation error")
	}
	if code == 0 {
		t.Fatal("expected a non-zero exit code for a failing command")
	}
}
