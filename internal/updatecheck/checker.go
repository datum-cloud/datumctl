// Package updatecheck performs a best-effort check for a newer datumctl
// release on GitHub and surfaces a one-line warning to stderr when one is
// available. All errors are silent: the user-facing command must never be
// affected by an update check.
package updatecheck

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

const (
	// EnvDisable, when set to a truthy value, disables the update check.
	EnvDisable = "DATUMCTL_NO_UPDATE_CHECK"

	// CacheTTL is how long a fetched latest version is reused before a new
	// network call is attempted.
	CacheTTL = 24 * time.Hour

	// httpTimeout bounds the network call so a hung connection cannot
	// affect the user-facing command.
	httpTimeout = 2 * time.Second

	releasesLatestURL = "https://github.com/datum-cloud/datumctl/releases/latest"
)

// Checker performs a single asynchronous update check. A zero Checker is not
// usable; construct one with New.
type Checker struct {
	current   string
	cachePath string
	client    *http.Client
	url       string
	now       func() time.Time

	disabled bool
	done     chan struct{}
	warning  string
}

// New constructs a Checker for the given current version (e.g. "v0.13.2").
// If currentVersion is empty, "unknown", or not a vX.Y.Z-style string the
// checker is disabled and Start/Wait become no-ops.
func New(currentVersion string) *Checker {
	c := &Checker{
		current:   currentVersion,
		cachePath: defaultCachePath(),
		client: &http.Client{
			Timeout: httpTimeout,
			// Do not follow redirects: the latest release URL responds
			// with a 302 whose Location contains the tag we want.
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		url:  releasesLatestURL,
		now:  time.Now,
		done: make(chan struct{}),
	}
	if shouldSkip(currentVersion) {
		c.disabled = true
		close(c.done)
	}
	return c
}

// Start launches the check in a goroutine. Safe to call once. If the checker
// is disabled, returns immediately. The provided context bounds the work.
func (c *Checker) Start(ctx context.Context) {
	if c.disabled {
		return
	}
	go func() {
		defer close(c.done)
		c.warning = c.run(ctx)
	}()
}

// Wait blocks up to deadline for the goroutine to finish and returns the
// warning string (empty if no update is available, the check failed, or the
// deadline elapsed).
func (c *Checker) Wait(deadline time.Duration) string {
	if c.disabled {
		return ""
	}
	select {
	case <-c.done:
		return c.warning
	case <-time.After(deadline):
		return ""
	}
}

func (c *Checker) run(ctx context.Context) string {
	latest, ok := c.cachedLatest()
	if !ok {
		fetched, err := c.fetchLatest(ctx)
		if err != nil || fetched == "" {
			return ""
		}
		_ = c.saveCache(fetched)
		latest = fetched
	}
	if !isNewer(c.current, latest) {
		return ""
	}
	return formatWarning(c.current, latest)
}

func formatWarning(current, latest string) string {
	var b strings.Builder
	b.WriteString("A new version of datumctl is available: ")
	b.WriteString(current)
	b.WriteString(" → ")
	b.WriteString(latest)
	b.WriteString("\nhttps://github.com/datum-cloud/datumctl/releases/tag/")
	b.WriteString(latest)
	return b.String()
}

// shouldSkip returns true when the current binary version indicates a
// development build and the check should be a no-op.
func shouldSkip(v string) bool {
	if v == "" || v == "unknown" {
		return true
	}
	if !strings.HasPrefix(v, "v") {
		return true
	}
	return false
}

// SkipFromEnvironment returns true when the update check should be skipped
// based on environment signals (opt-out env var or non-TTY stderr).
func SkipFromEnvironment() bool {
	if v := os.Getenv(EnvDisable); v != "" && v != "0" && strings.ToLower(v) != "false" {
		return true
	}
	if !term.IsTerminal(int(os.Stderr.Fd())) {
		return true
	}
	return false
}
