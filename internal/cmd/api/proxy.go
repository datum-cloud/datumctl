package api

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/templates"

	"go.datum.net/datumctl/internal/apiproxy"
	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/client"
	"go.datum.net/datumctl/internal/datumconfig"
	customerrors "go.datum.net/datumctl/internal/errors"
	"go.datum.net/datumctl/internal/miloapi"
)

// shutdownGrace is how long in-flight requests get to finish after the first
// interrupt before their connections are cut.
const shutdownGrace = 2 * time.Second

func proxyCommand(factory *client.DatumCloudFactory) *cobra.Command {
	var (
		port        int
		sessionName string
		quiet       bool
	)

	cmd := &cobra.Command{
		Use:   "proxy",
		Short: "Start a local proxy that authenticates requests to the Datum Cloud API",
		Long: templates.LongDesc(`
			Start a local proxy that authenticates requests to the Datum Cloud API.

			The proxy listens on 127.0.0.1 and forwards every request to the API
			endpoint of your datumctl session, adding your credentials automatically
			and refreshing them as needed. Point any local tool at the printed URL —
			no tokens to copy, no expiry to manage.

			By default the proxy serves the full API endpoint, so requests use the
			same paths as the real API. Pass --project or --organization to serve a
			single control plane instead, with shorter paths.

			The session and scope are pinned when the proxy starts. Switching your
			active account or context does not affect a running proxy.`),
		Example: templates.Examples(`
			# Start a proxy on a fixed port for a dev server
			datumctl api proxy --port 8001

			# Start on a random free port; the URL is printed on the first stdout line
			datumctl api proxy

			# List organizations through the proxy
			curl http://127.0.0.1:8001/apis/resourcemanager.miloapis.com/v1alpha1/organizations

			# Watch DNS zones on a project control plane through the proxy
			curl "http://127.0.0.1:8001/apis/resourcemanager.miloapis.com/v1alpha1/projects/my-project/control-plane/apis/networking.datumapis.com/v1alpha/dnszones?watch=true"

			# Serve one project's control plane directly, for shorter URLs
			datumctl api proxy --port 8001 --project my-project
			curl "http://127.0.0.1:8001/apis/networking.datumapis.com/v1alpha/dnszones?watch=true"

			# Pin a non-active session
			datumctl api proxy --session sam@datum.net@api.staging.env.datum.net`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProxy(cmd, factory, port, sessionName, quiet)
		},
	}

	cmd.Flags().IntVar(&port, "port", 0, "Local port to listen on (default: a random free port)")
	cmd.Flags().StringVar(&sessionName, "session", "", "Pin a specific session by name (defaults to the active session; see 'datumctl auth list')")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Suppress per-request log lines")
	return cmd
}

// proxyTarget is everything about the upstream that gets pinned when the
// proxy starts: the session it serves, the resolved endpoint identity, the
// upstream root, and the human-readable scope shown in the banner.
type proxyTarget struct {
	session  *datumconfig.Session
	endpoint *client.SessionEndpoint
	upstream *url.URL
	scope    string
}

// resolveProxyTarget resolves the pinned session and upstream for the proxy.
//
// Unlike resource commands, scope comes only from the explicit flags passed
// here: with no scope flag the upstream is the endpoint root — never the
// active context and never DATUM_* environment variables. Tools cache the
// proxy URL, so its meaning must not depend on ambient state at launch (see
// the api-proxy proposal, "Path semantics").
func resolveProxyTarget(cfg *datumconfig.ConfigV1Beta1, sessionName, project, organization string, platformWide bool) (*proxyTarget, error) {
	if platformWide && (project != "" || organization != "") {
		return nil, customerrors.NewUserError("--platform-wide cannot be used with --project or --organization")
	}
	if project != "" && organization != "" {
		return nil, customerrors.NewUserError("only one of --project or --organization may be set")
	}

	session, endpoint, err := client.ResolveSessionEndpoint(cfg, sessionName)
	if err != nil {
		if authutil.IsNoActiveUser(err) {
			return nil, customerrors.NewUserErrorWithHint(
				"No datumctl session found.",
				"Run 'datumctl login' to authenticate, then start the proxy again.",
			)
		}
		return nil, err
	}

	upstream := endpoint.BaseServer
	scope := "full endpoint (use --project/--organization to serve one control plane)"
	switch {
	case platformWide:
		scope = "platform-wide"
	case organization != "":
		upstream = miloapi.OrgControlPlaneURL(endpoint.BaseServer, organization)
		scope = "organization " + organization
	case project != "":
		upstream = miloapi.ProjectControlPlaneURL(endpoint.BaseServer, project)
		scope = "project " + project
	}

	upstreamURL, err := url.Parse(upstream)
	if err != nil {
		return nil, customerrors.WrapUserErrorWithHint(
			fmt.Sprintf("The session's API endpoint produced an invalid upstream URL (%q).", upstream),
			"Run 'datumctl login' again to refresh the session's endpoint, or check the session with 'datumctl auth list'.",
			err,
		)
	}

	return &proxyTarget{
		session:  session,
		endpoint: endpoint,
		upstream: upstreamURL,
		scope:    scope,
	}, nil
}

// tlsClientConfig converts the session endpoint's TLS settings into a
// *tls.Config for the upstream transport. Returns nil when the endpoint
// declares nothing, keeping Go's defaults.
func tlsClientConfig(epTLS client.EndpointTLS) (*tls.Config, error) {
	if epTLS.ServerName == "" && !epTLS.InsecureSkipTLSVerify && len(epTLS.CAData) == 0 {
		return nil, nil
	}
	cfg := &tls.Config{
		ServerName:         epTLS.ServerName,
		InsecureSkipVerify: epTLS.InsecureSkipTLSVerify,
	}
	if len(epTLS.CAData) > 0 {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(epTLS.CAData) {
			return nil, customerrors.NewUserErrorWithHint(
				"No certificates could be parsed from the session's certificate authority data.",
				"Run 'datumctl login' again to refresh the session's endpoint settings.",
			)
		}
		cfg.RootCAs = pool
	}
	return cfg, nil
}

// sessionLabel names the identity the proxy serves, for the startup banner.
func sessionLabel(target *proxyTarget) string {
	if target.session != nil {
		return fmt.Sprintf("%s (%s)", target.session.UserEmail, datumconfig.StripScheme(target.endpoint.BaseServer))
	}
	return target.endpoint.UserKey
}

func runProxy(cmd *cobra.Command, factory *client.DatumCloudFactory, port int, sessionName string, quiet bool) error {
	cfg, err := datumconfig.LoadAuto()
	if err != nil {
		return err
	}
	if err := authutil.EnsureUserKeysMigrated(cfg); err != nil {
		return err
	}

	flags := factory.ConfigFlags
	target, err := resolveProxyTarget(cfg, sessionName,
		stringValue(flags.Project), stringValue(flags.Organization), boolValue(flags.PlatformWide))
	if err != nil {
		return err
	}

	tokenSource, err := authutil.GetTokenSourceForUser(cmd.Context(), target.endpoint.UserKey)
	if err != nil {
		if _, isUser := customerrors.IsUserError(err); isUser {
			return err
		}
		return customerrors.WrapUserErrorWithHint(
			fmt.Sprintf("No stored credentials found for %s.", sessionLabel(target)),
			"Run 'datumctl login' to authenticate, then start the proxy again.",
			err,
		)
	}

	tlsConfig, err := tlsClientConfig(target.endpoint.TLS)
	if err != nil {
		return err
	}

	errOut := cmd.ErrOrStderr()
	server, err := apiproxy.New(apiproxy.Config{
		Upstream:        target.upstream,
		TokenSource:     tokenSource,
		TLSClientConfig: tlsConfig,
		LogWriter:       errOut,
		Quiet:           quiet,
	})
	if err != nil {
		return err
	}

	// Loopback only, by design: there is no flag to bind other addresses.
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		if port != 0 {
			return customerrors.WrapUserErrorWithHint(
				fmt.Sprintf("Could not listen on 127.0.0.1:%d.", port),
				"The port may already be in use — pass a different --port, or omit --port to pick a random free port.",
				err,
			)
		}
		return customerrors.WrapUserError("Could not open a local listener on 127.0.0.1.", err)
	}
	localURL := "http://" + listener.Addr().String()

	// Register the signal handler before advertising readiness, so a harness
	// that reads the URL line and later interrupts the proxy can never signal
	// an unhandled process.
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	fmt.Fprintf(errOut, "  Session:    %s\n", sessionLabel(target))
	fmt.Fprintf(errOut, "  Upstream:   %s\n", target.upstream)
	fmt.Fprintf(errOut, "  Scope:      %s\n", target.scope)
	fmt.Fprintf(errOut, "  Listening:  %s\n", localURL)
	fmt.Fprintln(errOut)
	if quiet {
		fmt.Fprintln(errOut, "  Press Ctrl+C to stop.")
	} else {
		fmt.Fprintln(errOut, "  Press Ctrl+C to stop. Requests are logged below (silence with --quiet).")
	}
	fmt.Fprintln(errOut)

	// Machine-readable readiness contract: the bare URL is the first and only
	// stdout line, printed only after the listener is bound.
	fmt.Fprintln(cmd.OutOrStdout(), localURL)

	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(listener) }()

	select {
	case err := <-serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-signals:
	}

	// Graceful shutdown: the listener closes and in-flight requests get the
	// grace period before their connections are cut. A second signal exits
	// immediately.
	fmt.Fprintln(errOut, "Shutting down...")
	shutdownDone := make(chan struct{})
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
		defer cancel()
		_ = server.Shutdown(ctx)
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
	case <-signals:
	}
	return nil
}

func stringValue(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func boolValue(p *bool) bool {
	return p != nil && *p
}
