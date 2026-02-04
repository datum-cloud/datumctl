package docs

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/openapi3"
	"k8s.io/client-go/rest"

	"go.datum.net/datumctl/internal/authutil"
)

//go:embed templates/*
var templatesFS embed.FS

type openAPIOptions struct {
	port         int
	noBrowser    bool
	organization string
	project      string
	platformWide bool
}

// OpenAPICmd returns the openapi subcommand.
func OpenAPICmd() *cobra.Command {
	opts := &openAPIOptions{}

	cmd := &cobra.Command{
		Use:   "openapi",
		Short: "Browse OpenAPI specs for platform APIs",
		Long: `Discovers available API groups from the platform and serves
Swagger UI for interactive API exploration.

A dropdown in the UI allows switching between different API groups
without restarting the server.

By default, discovers APIs from the platform root. Use --organization or
--project to browse APIs from a specific control plane.

Examples:
  # Browse platform-wide APIs (default)
  datumctl docs openapi

  # Browse APIs for an organization
  datumctl docs openapi --organization my-org

  # Browse APIs for a project
  datumctl docs openapi --project my-project

  # Use a specific port
  datumctl docs openapi --port 8080`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOpenAPI(cmd.Context(), cmd, opts)
		},
	}

	cmd.Flags().IntVar(&opts.port, "port", 0, "Port for Swagger UI server (default: random available)")
	cmd.Flags().BoolVar(&opts.noBrowser, "no-browser", false, "Don't open browser automatically")
	cmd.Flags().StringVar(&opts.organization, "organization", "", "Organization to target")
	cmd.Flags().StringVar(&opts.project, "project", "", "Project to target")
	cmd.Flags().BoolVar(&opts.platformWide, "platform-wide", false, "Access platform-wide APIs")

	cmd.MarkFlagsMutuallyExclusive("organization", "project", "platform-wide")

	return cmd
}

func runOpenAPI(ctx context.Context, cmd *cobra.Command, opts *openAPIOptions) error {
	// Build REST config based on context flags
	cfg, err := buildRESTConfig(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to create client config: %w", err)
	}

	// Create discovery client
	disco, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create discovery client: %w", err)
	}

	// Create OpenAPI v3 root
	root := openapi3.NewRoot(disco.OpenAPIV3())

	// Discover available API groups
	groupVersions, err := root.GroupVersions()
	if err != nil {
		return fmt.Errorf("failed to discover API groups: %w", err)
	}

	if len(groupVersions) == 0 {
		return fmt.Errorf("no API groups found")
	}

	// Sort for consistent display
	sort.Slice(groupVersions, func(i, j int) bool {
		return groupVersions[i].String() < groupVersions[j].String()
	})

	cmd.Printf("Discovered %d API groups\n", len(groupVersions))

	// Start server with Swagger UI
	return serveSwaggerUI(ctx, cmd, root, groupVersions, opts)
}

func buildRESTConfig(ctx context.Context, opts *openAPIOptions) (*rest.Config, error) {
	tknSrc, err := authutil.GetTokenSource(ctx)
	if err != nil {
		return nil, fmt.Errorf("get token source: %w", err)
	}

	apiHostname, err := authutil.GetAPIHostname()
	if err != nil {
		return nil, fmt.Errorf("get API hostname: %w", err)
	}

	var host string
	switch {
	case opts.organization != "":
		host = fmt.Sprintf("https://%s/apis/resourcemanager.miloapis.com/v1alpha1/organizations/%s/control-plane",
			apiHostname, opts.organization)
	case opts.project != "":
		host = fmt.Sprintf("https://%s/apis/resourcemanager.miloapis.com/v1alpha1/projects/%s/control-plane",
			apiHostname, opts.project)
	default:
		// Default to platform-wide mode
		host = fmt.Sprintf("https://%s", apiHostname)
	}

	return &rest.Config{
		Host: host,
		WrapTransport: func(rt http.RoundTripper) http.RoundTripper {
			return &oauth2.Transport{Source: tknSrc, Base: rt}
		},
	}, nil
}

type apiGroupInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	SpecURL string `json:"specUrl"`
}

func serveSwaggerUI(ctx context.Context, cmd *cobra.Command, root openapi3.Root, gvs []schema.GroupVersion, opts *openAPIOptions) error {
	// Parse template once at startup
	tmpl, err := template.ParseFS(templatesFS, "templates/swagger-ui.html")
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Find available port
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", opts.port))
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	mux := http.NewServeMux()

	// Build API group info for the UI
	apiGroups := make([]apiGroupInfo, 0, len(gvs))
	for _, gv := range gvs {
		id := gvToID(gv)
		apiGroups = append(apiGroups, apiGroupInfo{
			ID:      id,
			Name:    formatGroupVersion(gv),
			SpecURL: fmt.Sprintf("/specs/%s.json", id),
		})
	}

	// Serve the list of API groups
	mux.HandleFunc("/api/groups", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(apiGroups); err != nil {
			// Headers already sent, just log would be ideal but we don't have a logger
			return
		}
	})

	// Serve individual OpenAPI specs
	mux.HandleFunc("/specs/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract the group ID from the path
		path := strings.TrimPrefix(r.URL.Path, "/specs/")
		path = strings.TrimSuffix(path, ".json")

		gv, err := idToGV(path, gvs)
		if err != nil {
			http.Error(w, "API group not found", http.StatusNotFound)
			return
		}

		spec, err := root.GVSpecAsMap(gv)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to fetch spec: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "max-age=300") // Cache for 5 minutes
		if err := json.NewEncoder(w).Encode(spec); err != nil {
			// Headers already sent, just return
			return
		}
	})

	// Serve the main UI page
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		data := struct {
			Title     string
			APIGroups []apiGroupInfo
		}{
			Title:     "Datum Cloud API Explorer",
			APIGroups: apiGroups,
		}

		w.Header().Set("Content-Type", "text/html")
		if err := tmpl.Execute(w, data); err != nil {
			// Template execution failed after headers sent, nothing we can do
			return
		}
	})

	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	cmd.Printf("Swagger UI available at: %s\n", url)
	cmd.Println("Press Ctrl+C to stop the server")

	// Open browser
	if !opts.noBrowser {
		if err := browser.OpenURL(url); err != nil {
			cmd.Printf("Could not open browser automatically. Please visit: %s\n", url)
		}
	}

	// Create server with timeouts
	server := &http.Server{
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown with timeout
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	return server.Serve(listener)
}

func formatGroupVersion(gv schema.GroupVersion) string {
	if gv.Group == "" {
		return fmt.Sprintf("core/%s", gv.Version)
	}
	return gv.String()
}

// gvToID converts a GroupVersion to a URL-safe ID
func gvToID(gv schema.GroupVersion) string {
	if gv.Group == "" {
		return fmt.Sprintf("core_%s", gv.Version)
	}
	// Replace dots and slashes with underscores
	return strings.ReplaceAll(gv.Group, ".", "_") + "_" + gv.Version
}

// idToGV converts a URL-safe ID back to a GroupVersion
func idToGV(id string, gvs []schema.GroupVersion) (schema.GroupVersion, error) {
	for _, gv := range gvs {
		if gvToID(gv) == id {
			return gv, nil
		}
	}
	return schema.GroupVersion{}, fmt.Errorf("not found")
}
