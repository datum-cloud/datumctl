package auth

import (
	"bytes"
	"strings"
	"testing"

	customerrors "go.datum.net/datumctl/internal/errors"
)

func TestAuthUnknownSubcommand(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantErr  bool
		wantCode string
		wantHint string
	}{
		{
			name: "bare auth shows help",
			args: []string{},
		},
		{
			name:     "moved login errors with top-level hint",
			args:     []string{"login"},
			wantErr:  true,
			wantCode: "AUTH_COMMAND_MOVED",
			wantHint: "datumctl login",
		},
		{
			name:     "moved logout errors with top-level hint",
			args:     []string{"logout"},
			wantErr:  true,
			wantCode: "AUTH_COMMAND_MOVED",
			wantHint: "datumctl logout",
		},
		{
			name:     "unknown subcommand errors generically",
			args:     []string{"bogus"},
			wantErr:  true,
			wantCode: "UNKNOWN_SUBCOMMAND",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := Command()
			cmd.SetArgs(tc.args)
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})

			err := cmd.Execute()

			if !tc.wantErr {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
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
			if userErr.Code != tc.wantCode {
				t.Errorf("Code = %q, want %q", userErr.Code, tc.wantCode)
			}
			if tc.wantHint != "" && !strings.Contains(userErr.Hint, tc.wantHint) {
				t.Errorf("Hint = %q, want it to contain %q", userErr.Hint, tc.wantHint)
			}
		})
	}
}

func TestAuthValidSubcommandStillRoutes(t *testing.T) {
	cmd := Command()
	sub, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("Find(list) returned error: %v", err)
	}
	if sub.Name() != "list" {
		t.Fatalf("expected to resolve the 'list' subcommand, got %q", sub.Name())
	}
}
