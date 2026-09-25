package logs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/util/templates"

	"go.datum.net/datumctl/internal/client"
	customerrors "go.datum.net/datumctl/internal/errors"
	"go.datum.net/datumctl/internal/miloapi"
)

// Command returns the top-level "logs" command.
func Command(factory *client.DatumCloudFactory) *cobra.Command {
	var (
		since     string
		start     string
		end       string
		limit     int32
		direction string
	)

	cmd := &cobra.Command{
		Use:   "logs QUERY",
		Short: "Query telemetry logs for the active project",
		Long: templates.LongDesc(`
			Query telemetry logs for the active project.

			A query is required and must start with a stream selector containing at
			least one label matcher. The accepted query language is the subset of
			LogQL made up of label matchers and line filters, for example:

			    {service_name="checkout"} |= "error"

			Metric queries and aggregations are not supported, and neither are the
			parser and formatting stages (json, logfmt, line_format and friends).

			Logs are scoped to the active project on the server side, resolved from
			your context or from --project/--organization. Live tailing is not
			available yet, so each invocation returns a bounded window.`),
		Example: templates.Examples(`
			# Show the last 5 minutes of logs for a service
			datumctl logs '{service_name="checkout"}' --since 5m

			# Filter by a line substring over a 1h window
			datumctl logs '{service_name="checkout"} |= "500"' --since 1h --limit 500

			# Query an explicit window, oldest line first
			datumctl logs '{service_name="checkout"}' \
			  --start 2026-09-09T00:00:00Z --end 2026-09-09T01:00:00Z --direction forward`),
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.TrimSpace(strings.Join(args, " "))
			if query == "" {
				return customerrors.NewUserErrorWithHint(
					"a log query is required",
					`Start with a stream selector, e.g. datumctl logs '{service_name="checkout"}'`,
				)
			}
			return run(cmd.Context(), factory, runOptions{
				query:     query,
				since:     since,
				start:     start,
				end:       end,
				limit:     limit,
				direction: direction,
			})
		},
	}

	f := cmd.Flags()
	f.StringVar(&since, "since", "", "return logs newer than a relative duration, e.g. 5m, 1h (sets --start implicitly)")
	f.StringVar(&start, "start", "", "start of the window as RFC3339 timestamp or unix seconds")
	f.StringVar(&end, "end", "", "end of the window as RFC3339 timestamp or unix seconds")
	f.Int32Var(&limit, "limit", 100, "maximum number of log lines to return (clamped server-side to 5000)")
	f.StringVar(&direction, "direction", "backward", "order to return lines in: forward or backward")
	return cmd
}

type runOptions struct {
	query     string
	since     string
	start     string
	end       string
	limit     int32
	direction string
}

func run(ctx context.Context, factory *client.DatumCloudFactory, opts runOptions) error {
	restConfig, err := factory.ConfigFlags.ToRESTConfig()
	if err != nil {
		return customerrors.WrapUserError("failed to build API client from your context", err)
	}

	httpClient, err := rest.HTTPClientFor(restConfig)
	if err != nil {
		return customerrors.WrapUserError("failed to build authenticated HTTP client", err)
	}

	start, end, err := resolveWindow(opts)
	if err != nil {
		return err
	}

	base := strings.TrimSuffix(restConfig.Host, "/") + miloapi.ProjectLogsAPIPrefix()
	u := base + "/loki/api/v1/query_range"
	reqURL, err := buildQueryURL(u, opts.query, start, end, opts.limit, opts.direction)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return customerrors.WrapUserError("telemetry logs request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return queryError(resp)
	}

	var out lokiQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return customerrors.WrapUserError("failed to decode queryapi response", err)
	}

	printStreams(out, opts.direction)
	return nil
}

// queryError turns a non-200 from queryapi into a user-facing error. The
// response body carries the reason the server rejected the request -- a LogQL
// parse error, an authorization denial -- so it is always surfaced; without it
// a 400 and a 403 are indistinguishable to the caller.
func queryError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
	detail := strings.TrimSpace(string(body))

	// Errors arrive either in the Loki envelope (the handler's own rejections)
	// or as a Kubernetes Status (the aggregator's and the authorizer's).
	var envelope lokiQueryResponse
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Error != "" {
		detail = envelope.Error
	} else {
		var status struct{ Message string }
		if err := json.Unmarshal(body, &status); err == nil && status.Message != "" {
			detail = status.Message
		}
	}

	msg := fmt.Sprintf("telemetry logs query returned %s", resp.Status)
	if detail != "" {
		msg = fmt.Sprintf("%s: %s", msg, detail)
	}

	var hint string
	switch resp.StatusCode {
	case http.StatusBadRequest:
		hint = `Queries accept only label matchers and line filters, e.g. '{service_name="checkout"} |= "error"'.`
	case http.StatusUnauthorized:
		hint = "Run 'datumctl login' to refresh your credentials."
	case http.StatusForbidden:
		hint = "Querying logs requires the o11y.miloapis.com/logs.query permission on the active project."
	case http.StatusNotFound:
		hint = "Check that your context resolves to a project and that the queryapi APIService is registered in this environment."
	default:
		hint = "Run 'datumctl ctx' to confirm the active project."
	}
	return customerrors.NewUserErrorWithHint(msg, hint)
}

func resolveWindow(opts runOptions) (start, end string, err error) {
	// --start and --since both set the lower bound. Honouring one and dropping
	// the other silently would return a window the caller did not ask for.
	if opts.start != "" && opts.since != "" {
		return "", "", customerrors.NewUserError("--since and --start are mutually exclusive; pass only one")
	}

	end = opts.end
	if end == "" {
		end = time.Now().UTC().Format(time.RFC3339)
	}
	switch {
	case opts.start != "":
		start = opts.start
	case opts.since != "":
		d, perr := time.ParseDuration(opts.since)
		if perr != nil {
			return "", "", customerrors.WrapUserError("invalid --since value", perr)
		}
		e, eerr := parseTime(end)
		if eerr != nil {
			return "", "", eerr
		}
		start = e.Add(-d).UTC().Format(time.RFC3339)
	default:
		start = time.Now().UTC().Add(-10 * time.Minute).Format(time.RFC3339)
	}
	return start, end, nil
}

func parseTime(s string) (time.Time, error) {
	if secs, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Unix(secs, 0).UTC(), nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, customerrors.NewUserError(`--start/--end must be an RFC3339 timestamp or unix seconds`)
	}
	return t, nil
}

func buildQueryURL(base, query, start, end string, limit int32, direction string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", customerrors.WrapUserError("failed to parse queryapi URL", err)
	}
	q := u.Query()
	q.Set("query", query)
	q.Set("start", start)
	q.Set("end", end)
	if limit > 0 {
		q.Set("limit", strconv.FormatInt(int64(limit), 10))
	}
	if direction != "" {
		q.Set("direction", direction)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// lokiQueryResponse mirrors openapi.yaml's LokiQueryResponse envelope.
type lokiQueryResponse struct {
	Status    string        `json:"status"`
	Data      lokiQueryData `json:"data"`
	Error     string        `json:"error"`
	ErrorType string        `json:"errorType"`
}

type lokiQueryData struct {
	ResultType string       `json:"resultType"`
	Result     []lokiStream `json:"result"`
}

type lokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][2]string       `json:"values"`
}

// entry is one log line lifted out of the stream it arrived in, so that lines
// from every stream can be ordered against each other.
type entry struct {
	nanos  int64
	hasTS  bool
	ts     string
	labels string
	line   string
}

// printStreams renders the response newest-first, or oldest-first when
// direction is forward.
//
// queryapi returns one stream per distinct label set, each internally ordered.
// Printing them stream by stream would walk the clock backwards at every stream
// boundary, which for a single service spread over several label sets is most
// of the output. The lines are flattened and re-ordered so that the sequence on
// screen is the sequence the caller asked for.
func printStreams(out lokiQueryResponse, direction string) {
	var entries []entry
	for _, stream := range out.Data.Result {
		labels := formatLabels(stream.Stream)
		for _, pair := range stream.Values {
			n, err := strconv.ParseInt(pair[0], 10, 64)
			entries = append(entries, entry{
				nanos:  n,
				hasTS:  err == nil,
				ts:     sanitizeTS(pair[0]),
				labels: labels,
				line:   pair[1],
			})
		}
	}

	forward := direction == "forward"
	// Stable, so lines sharing a timestamp keep the order the server sent them
	// in, and unparseable timestamps sort together at the end rather than
	// masquerading as the epoch.
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.hasTS != b.hasTS {
			return a.hasTS
		}
		if !a.hasTS {
			return false
		}
		if forward {
			return a.nanos < b.nanos
		}
		return a.nanos > b.nanos
	})

	for _, e := range entries {
		fmt.Printf("%s | %s | %s\n", e.ts, e.labels, e.line)
	}
}

// formatLabels renders a label set in sorted key order. Ranging over the map
// directly would emit a different permutation on every run, so two identical
// queries would produce text that does not compare equal.
func formatLabels(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%q", k, labels[k]))
	}
	return strings.Join(parts, ",")
}

// sanitizeTS renders a nanosecond string timestamp as a human-readable time.
func sanitizeTS(ns string) string {
	n, err := strconv.ParseInt(ns, 10, 64)
	if err != nil {
		return ns
	}
	return time.Unix(0, n).UTC().Format(time.RFC3339)
}
