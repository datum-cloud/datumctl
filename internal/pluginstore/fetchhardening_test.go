package pluginstore

import (
	"bytes"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func mustRequest(t *testing.T, rawURL string) *http.Request {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse %q: %v", rawURL, err)
	}
	return &http.Request{URL: u}
}

func TestSafeCheckRedirect_RejectsHTTPDowngrade(t *testing.T) {
	// A third-party HTTPS catalog 302-redirecting the manifest fetch to http://
	// must be rejected (TLS downgrade).
	err := SafeCheckRedirect(mustRequest(t, "http://example.com/index.yaml"), nil)
	if err == nil {
		t.Fatal("expected redirect to http:// to be rejected, got nil")
	}
	if !strings.Contains(err.Error(), "non-HTTPS") {
		t.Fatalf("expected a non-HTTPS error, got %q", err)
	}
}

func TestSafeCheckRedirect_RejectsLoopbackSSRF(t *testing.T) {
	// A redirect to a loopback IP literal must be rejected (SSRF).
	err := SafeCheckRedirect(mustRequest(t, "https://127.0.0.1/index.yaml"), nil)
	if err == nil {
		t.Fatal("expected redirect to 127.0.0.1 to be rejected, got nil")
	}
	if !strings.Contains(err.Error(), "private or non-routable") {
		t.Fatalf("expected an SSRF rejection error, got %q", err)
	}
}

func TestSafeCheckRedirect_AllowsPublicHTTPS(t *testing.T) {
	// A public HTTPS IP literal (no DNS lookup) should be allowed.
	if err := SafeCheckRedirect(mustRequest(t, "https://93.184.216.34/index.yaml"), nil); err != nil {
		t.Fatalf("expected public HTTPS target to be allowed, got %q", err)
	}
}

func TestSafeCheckRedirect_StopsAfterTooManyHops(t *testing.T) {
	via := make([]*http.Request, 10)
	err := SafeCheckRedirect(mustRequest(t, "https://93.184.216.34/x"), via)
	if err == nil || !strings.Contains(err.Error(), "stopped after 10 redirects") {
		t.Fatalf("expected redirect-limit error, got %v", err)
	}
}

func TestIsBlockedIP(t *testing.T) {
	blocked := []string{
		"127.0.0.1",    // loopback v4
		"::1",          // loopback v6
		"10.0.0.5",     // RFC1918
		"172.16.0.1",   // RFC1918
		"192.168.1.1",  // RFC1918
		"169.254.10.1", // link-local v4
		"fe80::1",      // link-local v6
		"fc00::1",      // ULA v6
		"0.0.0.0",      // unspecified v4
		"::",           // unspecified v6
		"224.0.0.1",    // multicast
	}
	for _, s := range blocked {
		if !isBlockedIP(net.ParseIP(s)) {
			t.Errorf("expected %s to be blocked", s)
		}
	}
	allowed := []string{
		"93.184.216.34",                      // example.com public
		"8.8.8.8",                            // public
		"2606:2800:220:1:248:1893:25c8:1946", // public v6
	}
	for _, s := range allowed {
		if isBlockedIP(net.ParseIP(s)) {
			t.Errorf("expected %s to be allowed", s)
		}
	}
}

func TestValidateHostNotBlocked_IPLiterals(t *testing.T) {
	if err := validateHostNotBlocked("127.0.0.1"); err == nil {
		t.Error("expected loopback literal to be rejected")
	}
	if err := validateHostNotBlocked("93.184.216.34"); err != nil {
		t.Errorf("expected public literal to be allowed, got %v", err)
	}
	if err := validateHostNotBlocked(""); err == nil {
		t.Error("expected empty host to be rejected")
	}
}

func TestReadCapped(t *testing.T) {
	// Under the cap: returned verbatim.
	got, err := ReadCapped(bytes.NewReader([]byte("hello")), 16)
	if err != nil {
		t.Fatalf("unexpected error under cap: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("expected %q, got %q", "hello", got)
	}

	// Exactly at the cap is allowed.
	if _, err := ReadCapped(bytes.NewReader(bytes.Repeat([]byte("a"), 8)), 8); err != nil {
		t.Fatalf("expected exact-cap read to succeed, got %v", err)
	}

	// Over the cap: rejected with a clear error.
	_, err = ReadCapped(bytes.NewReader(bytes.Repeat([]byte("a"), 100)), 8)
	if err == nil || !strings.Contains(err.Error(), "maximum allowed size") {
		t.Fatalf("expected over-cap rejection, got %v", err)
	}
}

func TestManifestCapConstantSane(t *testing.T) {
	// Guard against an accidental zero/negative cap that would reject everything.
	if MaxManifestBytes <= 0 || MaxArchiveBytes <= 0 || MaxDecompressedFileBytes <= 0 {
		t.Fatal("size caps must be positive")
	}
	if MaxArchiveBytes < MaxManifestBytes {
		t.Fatal("archive cap should be at least the manifest cap")
	}
}
