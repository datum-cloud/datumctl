package updatecheck

import (
	"context"
	"net/http"
	"path"
	"strings"
)

// fetchLatest issues a HEAD against the releases/latest URL and parses the
// tag from the redirect Location header. Returns "" with no error when the
// response is not a redirect or the Location is unparseable; returns an
// error only on transport failures.
func (c *Checker) fetchLatest(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, c.url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "datumctl-update-check")
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 300 || resp.StatusCode >= 400 {
		return "", nil
	}
	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", nil
	}
	tag := path.Base(loc)
	if tag == "" || tag == "/" || tag == "." {
		return "", nil
	}
	if !strings.HasPrefix(tag, "v") {
		return "", nil
	}
	return tag, nil
}
