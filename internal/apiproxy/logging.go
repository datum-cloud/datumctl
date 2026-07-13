package apiproxy

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const timeFormat = "15:04:05"

// redactedQueryKeys are query parameter names whose values are redacted in
// log lines. The platform API never puts credentials in query strings; this
// is a defensive pass.
var redactedQueryKeys = map[string]bool{
	"access_token":  true,
	"token":         true,
	"authorization": true,
}

// requestLogger writes one line per request — and for streaming responses,
// one when headers arrive and one on stream end — recording method, path,
// status, duration, and bytes. Headers and token values are never logged.
type requestLogger struct {
	next  http.Handler
	out   io.Writer
	quiet bool
}

func (l *requestLogger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if l.quiet {
		l.next.ServeHTTP(w, r)
		return
	}
	logged := &loggedResponse{
		ResponseWriter: w,
		out:            l.out,
		method:         r.Method,
		path:           redactedRequestPath(r),
		isHead:         r.Method == http.MethodHead,
		start:          time.Now(),
	}
	// Deferred so an aborted stream (client gone, upstream cut) still gets
	// its completion line.
	defer logged.logCompletion()
	l.next.ServeHTTP(logged, r)
}

// redactedRequestPath returns the request path plus query with token-shaped
// query values redacted, preserving parameter order.
func redactedRequestPath(r *http.Request) string {
	rawQuery := r.URL.RawQuery
	if rawQuery == "" {
		return r.URL.Path
	}
	pairs := strings.Split(rawQuery, "&")
	for i, pair := range pairs {
		key, _, found := strings.Cut(pair, "=")
		if found && redactedQueryKeys[strings.ToLower(key)] {
			pairs[i] = key + "=REDACTED"
		}
	}
	return r.URL.Path + "?" + strings.Join(pairs, "&")
}

// loggedResponse observes the response as it is written. A response with no
// declared Content-Length is treated as streaming: it logs a line the moment
// headers arrive and another with totals on completion.
type loggedResponse struct {
	http.ResponseWriter
	out    io.Writer
	method string
	path   string
	isHead bool
	start  time.Time

	wroteHeader bool
	streaming   bool
	status      int
	bytes       int64
}

func (l *loggedResponse) WriteHeader(code int) {
	if code >= 100 && code < 200 {
		// Informational responses don't settle the final status.
		l.ResponseWriter.WriteHeader(code)
		return
	}
	if !l.wroteHeader {
		l.wroteHeader = true
		l.status = code
		l.streaming = l.detectStreaming(code)
		if l.streaming {
			fmt.Fprintf(l.out, "%s %-4s %s %d …streaming\n",
				time.Now().Format(timeFormat), l.method, l.path, code)
		}
	}
	l.ResponseWriter.WriteHeader(code)
}

func (l *loggedResponse) Write(b []byte) (int, error) {
	if !l.wroteHeader {
		l.WriteHeader(http.StatusOK)
	}
	n, err := l.ResponseWriter.Write(b)
	l.bytes += int64(n)
	return n, err
}

// Flush and Unwrap keep the wrapped writer streamable: the reverse proxy
// reaches the underlying Flusher (and Hijacker, for upgrades) through them.
func (l *loggedResponse) Flush() {
	if flusher, ok := l.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (l *loggedResponse) Unwrap() http.ResponseWriter { return l.ResponseWriter }

// detectStreaming reports whether the response is stream-shaped: no declared
// length on a status/method that can carry a body.
func (l *loggedResponse) detectStreaming(code int) bool {
	if l.isHead || code == http.StatusNoContent || code == http.StatusNotModified {
		return false
	}
	return l.Header().Get("Content-Length") == ""
}

func (l *loggedResponse) logCompletion() {
	status := l.status
	if !l.wroteHeader {
		// The handler wrote nothing; net/http sends an implicit 200.
		status = http.StatusOK
	}
	fmt.Fprintf(l.out, "%s %-4s %s %d %s %s\n",
		time.Now().Format(timeFormat), l.method, l.path, status,
		formatDuration(time.Since(l.start)), formatBytes(l.bytes))
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

func formatBytes(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%dB", n)
	}
	value := float64(n)
	for _, unit := range []string{"kB", "MB", "GB", "TB"} {
		value /= 1000
		if value < 1000 {
			return fmt.Sprintf("%.1f%s", value, unit)
		}
	}
	return fmt.Sprintf("%.1fPB", value/1000)
}
