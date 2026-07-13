// Package apiproxy implements the engine behind `datumctl api proxy`: a
// local reverse proxy that forwards every request to the configured Datum
// Cloud upstream, injecting a bearer token from the session's token source
// and streaming responses through unbuffered.
//
// The engine is deliberately self-contained: the upstream URL, token source,
// TLS settings, and log destination are all injected, so it can be tested —
// and reasoned about — without Cobra, the keyring, or a real network.
package apiproxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// Timeouts are asymmetric by design: connection setup is bounded, response
// duration is not — watches are infinite on purpose. Nothing in this package
// may introduce an overall request or response-duration timeout.
const (
	dialTimeout           = 10 * time.Second
	tlsHandshakeTimeout   = 10 * time.Second
	responseHeaderTimeout = 30 * time.Second

	// Local listener hygiene. There is deliberately no ReadTimeout or
	// WriteTimeout on the local server: either one would kill long-lived
	// streams.
	readHeaderTimeout = 10 * time.Second
	idleTimeout       = 2 * time.Minute

	// After a token refresh failure, requests fail fast with the cached
	// error for this long instead of re-attempting a refresh per request.
	refreshCooldown = 5 * time.Second
)

// Config carries everything the proxy engine needs. All dependencies are
// injected; the engine never reads datumctl config or the keyring itself.
type Config struct {
	// Upstream is the proxy root every request is forwarded under: the
	// endpoint root by default, or a control-plane URL whose path prefix
	// local request paths are joined onto.
	Upstream *url.URL

	// TokenSource supplies the bearer token injected into each outbound
	// request (typically authutil.GetTokenSourceForUser).
	TokenSource oauth2.TokenSource

	// TLSClientConfig configures TLS to the upstream. Nil means defaults.
	TLSClientConfig *tls.Config

	// LogWriter is the request log destination (stderr in production).
	// Nil discards log output.
	LogWriter io.Writer

	// Quiet suppresses per-request log lines. Token refresh failures are
	// still logged, once per refresh attempt.
	Quiet bool
}

// Server is a configured proxy engine. Callers either mount Handler on a
// server of their own or hand a listener to Serve.
type Server struct {
	handler    http.Handler
	httpServer *http.Server
}

// New builds a proxy engine from cfg.
func New(cfg Config) (*Server, error) {
	if cfg.Upstream == nil {
		return nil, fmt.Errorf("apiproxy: Upstream is required")
	}
	if cfg.Upstream.Scheme == "" || cfg.Upstream.Host == "" {
		return nil, fmt.Errorf("apiproxy: Upstream must be an absolute URL, got %q", cfg.Upstream)
	}
	if cfg.TokenSource == nil {
		return nil, fmt.Errorf("apiproxy: TokenSource is required")
	}
	logWriter := cfg.LogWriter
	if logWriter == nil {
		logWriter = io.Discard
	}

	upstreamTransport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   dialTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		TLSClientConfig:       cfg.TLSClientConfig,
		TLSHandshakeTimeout:   tlsHandshakeTimeout,
		ResponseHeaderTimeout: responseHeaderTimeout,
		IdleConnTimeout:       90 * time.Second,
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: rewriteTo(cfg.Upstream),
		Transport: &oauth2.Transport{
			Source: newCooldownTokenSource(cfg.TokenSource, refreshCooldown, logWriter),
			Base:   upstreamTransport,
		},
		// Flush every upstream write to the client immediately: unbuffered
		// streaming (watch, SSE, chunked transfer) is the point of the proxy.
		FlushInterval: -1,
		ErrorHandler:  handleProxyError,
		ErrorLog:      log.New(logWriter, "", 0),
	}

	var handler http.Handler = &hostValidator{next: proxy}
	handler = &requestLogger{next: handler, out: logWriter, quiet: cfg.Quiet}

	return &Server{
		handler: handler,
		httpServer: &http.Server{
			Handler:           handler,
			ReadHeaderTimeout: readHeaderTimeout,
			IdleTimeout:       idleTimeout,
			// No ReadTimeout/WriteTimeout: they would cut long-lived streams.
		},
	}, nil
}

// Handler returns the proxy handler: host validation, request logging, and
// the reverse proxy itself.
func (s *Server) Handler() http.Handler { return s.handler }

// Serve accepts connections on l until Shutdown is called, applying
// stream-safe local server settings: ReadHeaderTimeout and a keep-alive
// IdleTimeout only, never a ReadTimeout or WriteTimeout. It returns
// http.ErrServerClosed after Shutdown.
func (s *Server) Serve(l net.Listener) error { return s.httpServer.Serve(l) }

// Shutdown gracefully shuts the proxy down: the listener closes and
// in-flight requests get until ctx expires before their connections are cut.
func (s *Server) Shutdown(ctx context.Context) error { return s.httpServer.Shutdown(ctx) }

// tokenRefreshError marks a failure that came from the token source rather
// than the upstream connection, so the error handler can report
// ProxyAuthenticationFailed instead of a generic upstream error.
type tokenRefreshError struct{ err error }

func (e *tokenRefreshError) Error() string { return e.err.Error() }
func (e *tokenRefreshError) Unwrap() error { return e.err }

// cooldownTokenSource serializes token retrieval — at most one refresh in
// flight — and, after a refresh failure, fails fast with the cached error
// for a cooldown window instead of re-attempting a refresh per proxied
// request. Combined, the auth server sees at most one refresh attempt per
// window no matter how hot the local client polls.
type cooldownTokenSource struct {
	source   oauth2.TokenSource
	cooldown time.Duration
	out      io.Writer
	now      func() time.Time

	mu       sync.Mutex
	lastErr  *tokenRefreshError
	failedAt time.Time
}

func newCooldownTokenSource(source oauth2.TokenSource, cooldown time.Duration, out io.Writer) *cooldownTokenSource {
	return &cooldownTokenSource{source: source, cooldown: cooldown, out: out, now: time.Now}
}

// Token implements oauth2.TokenSource.
func (c *cooldownTokenSource) Token() (*oauth2.Token, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lastErr != nil {
		if c.now().Sub(c.failedAt) < c.cooldown {
			return nil, c.lastErr
		}
		c.lastErr = nil
	}

	token, err := c.source.Token()
	if err != nil {
		c.lastErr = &tokenRefreshError{err: err}
		c.failedAt = c.now()
		// Logged here, once per refresh attempt, rather than once per
		// request in the error handler.
		fmt.Fprintf(c.out, "%s token refresh failed: %s\n",
			c.now().Format(timeFormat), strings.ReplaceAll(err.Error(), "\n", " — "))
		return nil, c.lastErr
	}
	return token, nil
}
