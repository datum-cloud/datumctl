package errors

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

func TestFormatHuman(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		verbosity int
		want      string
	}{
		{
			name: "user error with hint",
			err:  NewUserErrorWithHint("session expired", "Run `datumctl auth login`."),
			want: "error: session expired\nRun `datumctl auth login`.\n",
		},
		{
			name: "user error without hint",
			err:  NewUserError("not logged in"),
			want: "error: not logged in\n",
		},
		{
			name: "wrapped technical error, low verbosity hides details",
			err:  WrapUserError("login failed", errors.New("oauth2: token expired")),
			want: "error: login failed\n",
		},
		{
			name:      "wrapped technical error, verbosity 4 shows details",
			err:       WrapUserError("login failed", errors.New("oauth2: token expired")),
			verbosity: 4,
			want:      "error: login failed\n\nDetails:\noauth2: token expired\n",
		},
		{
			name: "plain error",
			err:  errors.New("some failure"),
			want: "error: some failure\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			Format(&buf, tt.err, FormatHuman, tt.verbosity)
			if got := buf.String(); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatJSON(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want envelope
	}{
		{
			name: "user error with hint",
			err:  NewUserErrorWithHint("session expired", "Run `datumctl auth login`."),
			want: envelope{Error: envelopeError{
				Message:   "session expired",
				Hint:      "Run `datumctl auth login`.",
				Retryable: false,
			}},
		},
		{
			name: "user error with code and retryable",
			err: &UserError{
				Message:   "rate limited",
				Code:      "RATE_LIMITED",
				Retryable: true,
			},
			want: envelope{Error: envelopeError{
				Code:      "RATE_LIMITED",
				Message:   "rate limited",
				Retryable: true,
			}},
		},
		{
			name: "wrapped technical error includes details",
			err:  WrapUserError("login failed", errors.New("oauth2: token expired")),
			want: envelope{Error: envelopeError{
				Message: "login failed",
				Details: "oauth2: token expired",
			}},
		},
		{
			name: "plain error",
			err:  errors.New("some failure"),
			want: envelope{Error: envelopeError{Message: "some failure"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			Format(&buf, tt.err, FormatJSON, 0)
			out := strings.TrimSpace(buf.String())
			var got envelope
			if err := json.Unmarshal([]byte(out), &got); err != nil {
				t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
			}
			if got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
			if !strings.HasSuffix(buf.String(), "\n") {
				t.Errorf("JSON output should end with newline, got %q", buf.String())
			}
		})
	}
}

func TestFormatJSONOmitsEmptyOptionalFields(t *testing.T) {
	var buf bytes.Buffer
	Format(&buf, NewUserError("simple"), FormatJSON, 0)

	out := buf.String()
	for _, field := range []string{"\"code\"", "\"hint\"", "\"details\""} {
		if strings.Contains(out, field) {
			t.Errorf("expected %s to be omitted, got %s", field, out)
		}
	}
	// retryable must always be present so consumers can branch on it.
	if !strings.Contains(out, "\"retryable\"") {
		t.Errorf("expected retryable to be present, got %s", out)
	}
}

func TestFormatYAML(t *testing.T) {
	var buf bytes.Buffer
	in := &UserError{
		Message:   "session expired",
		Hint:      "Run `datumctl auth login`.",
		Code:      "AUTH_EXPIRED",
		Retryable: false,
		Err:       errors.New("oauth2: token expired"),
	}
	Format(&buf, in, FormatYAML, 0)

	var got envelope
	if err := yaml.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid YAML: %v\noutput: %s", err, buf.String())
	}
	want := envelope{Error: envelopeError{
		Code:      "AUTH_EXPIRED",
		Message:   "session expired",
		Hint:      "Run `datumctl auth login`.",
		Retryable: false,
		Details:   "oauth2: token expired",
	}}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestFormatUnknownFallsBackToHuman(t *testing.T) {
	var buf bytes.Buffer
	Format(&buf, NewUserError("boom"), "xml", 0)
	if got, want := buf.String(), "error: boom\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFormatNilNoOp(t *testing.T) {
	var buf bytes.Buffer
	Format(&buf, nil, FormatJSON, 0)
	if buf.Len() != 0 {
		t.Fatalf("expected no output for nil error, got %q", buf.String())
	}
}
