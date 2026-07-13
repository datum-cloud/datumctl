package apiproxy

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	customerrors "go.datum.net/datumctl/internal/errors"
)

// proxyErrorHeader marks responses synthesized by the proxy itself, so a
// client can tell a proxy-local failure from something the upstream said.
// Upstream responses — including 401/403 — pass through without it.
const proxyErrorHeader = "X-Datum-Proxy-Error"

// rewriteTo returns the ReverseProxy Rewrite func: the inbound path and
// query are forwarded verbatim, joined under the upstream root (which may
// carry a control-plane path prefix), with the upstream host as the
// outbound Host header.
func rewriteTo(upstream *url.URL) func(*httputil.ProxyRequest) {
	return func(pr *httputil.ProxyRequest) {
		pr.SetURL(upstream)
		pr.Out.Host = upstream.Host
		// Never forward a locally supplied credential upstream — the
		// oauth2.Transport injects the real one. Deleting rather than
		// overwriting also keeps stale tokens baked into client configs
		// from half-working.
		pr.Out.Header.Del("Authorization")
		// No SetXForwarded: the upstream gains nothing from knowing
		// about 127.0.0.1.
	}
}

// hostValidator rejects any request whose Host header is not a local
// address, before the upstream sees anything — the DNS-rebinding defense: a
// malicious page that rebinds its hostname to 127.0.0.1 still sends that
// hostname in Host.
type hostValidator struct {
	next http.Handler
}

func (h *hostValidator) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !allowedHost(r.Host) {
		writeStatus(w, http.StatusForbidden, "Forbidden",
			fmt.Sprintf("host %q is not a local address; the datumctl proxy only serves local clients", r.Host))
		return
	}
	h.next.ServeHTTP(w, r)
}

// allowedHost reports whether hostport is localhost, 127.0.0.1, or [::1],
// with or without a port.
func allowedHost(hostport string) bool {
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	host = strings.TrimPrefix(host, "[")
	host = strings.TrimSuffix(host, "]")
	switch strings.ToLower(host) {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	return false
}

// handleProxyError synthesizes a 502 for failures that happened proxy-side —
// deliberately not 401, so a client's re-auth logic never misfires on a
// proxy-local problem. Token-source failures and upstream connection
// failures get distinct Status reasons.
func handleProxyError(w http.ResponseWriter, r *http.Request, err error) {
	reason := "ProxyUpstreamError"
	var refreshErr *tokenRefreshError
	if errors.As(err, &refreshErr) {
		reason = "ProxyAuthenticationFailed"
	}
	writeStatus(w, http.StatusBadGateway, reason, errorMessage(err))
}

// errorMessage flattens err into a single Status message line, preferring a
// UserError's message and hint so clients see actionable guidance such as
// "run `datumctl login`".
func errorMessage(err error) string {
	if userErr, ok := customerrors.IsUserError(err); ok {
		if userErr.Hint != "" {
			return userErr.Message + " — " + userErr.Hint
		}
		return userErr.Message
	}
	return err.Error()
}

// kubeStatus is the subset of the Kubernetes Status object the proxy
// synthesizes — the dialect the target clients already parse.
type kubeStatus struct {
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
	Status     string `json:"status"`
	Code       int    `json:"code"`
	Reason     string `json:"reason"`
	Message    string `json:"message"`
}

// writeStatus writes a proxy-synthesized error response, carrying the marker
// header that distinguishes it from anything the upstream returned.
func writeStatus(w http.ResponseWriter, code int, reason, message string) {
	body, _ := json.Marshal(kubeStatus{
		Kind:       "Status",
		APIVersion: "v1",
		Status:     "Failure",
		Code:       code,
		Reason:     reason,
		Message:    message,
	})
	h := w.Header()
	h.Set("Content-Type", "application/json")
	h.Set("Content-Length", strconv.Itoa(len(body)))
	h.Set(proxyErrorHeader, "true")
	w.WriteHeader(code)
	_, _ = w.Write(body)
}
