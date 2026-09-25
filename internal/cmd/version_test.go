package cmd

import (
	"errors"
	"fmt"
	"testing"
)

func TestIsVersionSkewParseError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{
			"unresolved git archive placeholder in server version",
			fmt.Errorf("server version error: %w", errors.New(`could not parse pre-release/metadata (-master+$Format:%H$) in version "v0.0.0-master+$Format:%H$"`)),
			true,
		},
		{
			"unparseable client version",
			fmt.Errorf("client version error: %w", errors.New(`could not parse "not-a-version" as version`)),
			true,
		},
		{"unrelated error", errors.New("connection refused"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isVersionSkewParseError(tt.err); got != tt.want {
				t.Fatalf("isVersionSkewParseError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
