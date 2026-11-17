package datumconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultKind      = "DatumctlConfig"
	DefaultNamespace = "default"
)

// OrganizationNamespace returns the namespace that holds an organization's own
// namespaced resources in its control plane. The authorizer refuses
// organization-scope lists in any other namespace, and lists across all
// namespaces, whatever the caller's role.
func OrganizationNamespace(orgID string) string {
	return "organization-" + orgID
}

// DefaultPath returns the canonical config file path (~/.datumctl/config).
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".datumctl", "config"), nil
}

// EnsureScheme adds an https:// scheme to server if none is present.
func EnsureScheme(server string) string {
	if server == "" {
		return server
	}
	if strings.HasPrefix(server, "http://") || strings.HasPrefix(server, "https://") {
		return server
	}
	return "https://" + server
}

// CleanBaseServer strips a trailing slash from server.
func CleanBaseServer(server string) string {
	if server == "" {
		return server
	}
	return strings.TrimRight(server, "/")
}

// StripScheme removes the https:// or http:// prefix and any trailing slash,
// returning the bare host (and optional path).
func StripScheme(s string) string {
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimSuffix(s, "/")
	return s
}
