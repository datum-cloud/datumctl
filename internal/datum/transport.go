package datum

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// wrappedRoundTripper wraps the default transport and modifies request bodies transparently
type wrappedRoundTripper struct {
	Transport        http.RoundTripper
	AdditionalFields map[string]string
}

// RoundTrip modifies the request before sending it
func (c *wrappedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Read the existing body
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
	}
	req.Body.Close() // Close the original body to avoid leaks

	// Parse the body as URL-encoded form
	form, err := url.ParseQuery(string(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to parse request body: %w", err)
	}

	// Add additional fields to the form
	for key, value := range c.AdditionalFields {
		form.Add(key, value)
	}

	// Encode the updated form back into the request body
	encodedForm := form.Encode()
	req.Body = io.NopCloser(bytes.NewBufferString(encodedForm))
	req.ContentLength = int64(len(encodedForm))

	// Ensure Content-Type is correct
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Pass the modified request to the underlying transport
	return c.Transport.RoundTrip(req)
}
