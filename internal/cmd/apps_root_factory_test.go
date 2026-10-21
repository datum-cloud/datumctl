package cmd

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/client"
	"go.datum.net/datumctl/internal/datumconfig"
	customerrors "go.datum.net/datumctl/internal/errors"
	"go.datum.net/datumctl/internal/keyring"
	"go.datum.net/datumctl/internal/updatecheck"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

const (
	rootAppsProject      = "factory-project"
	rootAppsName         = "web-api"
	rootAppsUID          = "fixture-template-uid"
	rootAppsRV           = "opaque-rv-a"
	rootAppsRevisionName = "fixture-template-uid.rev-0001"
	rootAppsToken        = "fixture-root-command-token"
	rootAppsKey          = "fixture-root-command-key"
	rootAppsEmail        = "fixture@example.test"
)

const rootAppsCurrentDefinition = `{"description":"current","inputSchema":{"additionalProperties":false,"properties":{},"type":"object"}}`
const rootAppsRevisionDefinition = `{"description":"previous","inputSchema":{"additionalProperties":false,"properties":{"int53Plus":{"const":9007199254740993,"type":"integer"},"preciseDecimal":{"const":123456789012345678901e-21,"type":"number"},"trailingZeroInteger":{"const":9007199254740993e1,"type":"integer"},"uint64Max":{"const":18446744073709551615,"type":"integer"}},"type":"object"}}`

func rootAppsDigest(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func rootAppsQuoted(raw string) string {
	encoded, _ := json.Marshal(raw)
	return string(encoded)
}

var rootAppsCurrent = `{"apiVersion":"apps.miloapis.com/v1alpha1","kind":"AppTemplate","metadata":{"name":"web-api","uid":"fixture-template-uid","resourceVersion":"opaque-rv-a","generation":2},"spec":{"definitionJSON":` + rootAppsQuoted(rootAppsCurrentDefinition) + `,"digest":"` + rootAppsDigest(rootAppsCurrentDefinition) + `"}}`

var rootAppsRevisionJSON = `{"metadata":{"name":"fixture-template-uid.rev-0001","uid":"fixture-revision-uid"},"spec":{"templateUID":"fixture-template-uid","sequence":1,"digest":"` + rootAppsDigest(rootAppsRevisionDefinition) + `","effectiveActorUID":"actor-fixture","acceptedAt":"2026-10-20T00:00:00Z","definitionJSON":` + rootAppsQuoted(rootAppsRevisionDefinition) + `}}`

type rootAppsExpectedRequest struct {
	method string
	path   string
	query  string
	check  func(*testing.T, *http.Request)
	status int
	body   string
}

func TestRootCmdAppsCommandsUseScopedFactoryAndPreserveWritePreconditions(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		contextProject string
		envProject     string
		expect         []rootAppsExpectedRequest
		wantOutput     []string
	}{
		{
			name:           "history by current name uses active context",
			args:           []string{"apps", "templates", "history", rootAppsName},
			contextProject: "context-project",
			expect: []rootAppsExpectedRequest{
				{method: http.MethodGet, path: "/apis/resourcemanager.miloapis.com/v1alpha1/projects/context-project/control-plane/apis/apps.miloapis.com/v1alpha1/apptemplates/web-api", status: http.StatusOK, body: rootAppsCurrent},
				{method: http.MethodGet, path: "/apis/resourcemanager.miloapis.com/v1alpha1/projects/context-project/control-plane/apis/apps.miloapis.com/v1alpha1/apptemplaterevisions", query: "labelSelector=apps.miloapis.com%2Ftemplate-uid%3Dfixture-template-uid&limit=1", status: http.StatusOK, body: `{"items":[` + rootAppsRevisionJSON + `]}`},
			},
			wantOutput: []string{`"templateUID":"` + rootAppsUID + `"`, rootAppsRevisionName},
		},
		{
			name:           "DATUM_PROJECT overrides active context",
			args:           []string{"apps", "templates", "history", rootAppsName},
			contextProject: "context-project",
			envProject:     "env-project",
			expect: []rootAppsExpectedRequest{
				{method: http.MethodGet, path: "/apis/resourcemanager.miloapis.com/v1alpha1/projects/env-project/control-plane/apis/apps.miloapis.com/v1alpha1/apptemplates/web-api", status: http.StatusOK, body: rootAppsCurrent},
				{method: http.MethodGet, path: "/apis/resourcemanager.miloapis.com/v1alpha1/projects/env-project/control-plane/apis/apps.miloapis.com/v1alpha1/apptemplaterevisions", query: "labelSelector=apps.miloapis.com%2Ftemplate-uid%3Dfixture-template-uid&limit=1", status: http.StatusOK, body: `{"items":[` + rootAppsRevisionJSON + `]}`},
			},
			wantOutput: []string{`"templateUID":"` + rootAppsUID + `"`, rootAppsRevisionName},
		},
		{
			name:           "project flag overrides environment and context",
			args:           []string{"apps", "templates", "history", rootAppsName, "--project", "flag-project"},
			contextProject: "context-project",
			envProject:     "env-project",
			expect: []rootAppsExpectedRequest{
				{method: http.MethodGet, path: "/apis/resourcemanager.miloapis.com/v1alpha1/projects/flag-project/control-plane/apis/apps.miloapis.com/v1alpha1/apptemplates/web-api", status: http.StatusOK, body: rootAppsCurrent},
				{method: http.MethodGet, path: "/apis/resourcemanager.miloapis.com/v1alpha1/projects/flag-project/control-plane/apis/apps.miloapis.com/v1alpha1/apptemplaterevisions", query: "labelSelector=apps.miloapis.com%2Ftemplate-uid%3Dfixture-template-uid&limit=1", status: http.StatusOK, body: `{"items":[` + rootAppsRevisionJSON + `]}`},
			},
			wantOutput: []string{`"templateUID":"` + rootAppsUID + `"`, rootAppsRevisionName},
		},
		{
			name: "history for deleted template by UID",
			args: []string{"apps", "templates", "history", "--uid", "retired-template-uid", "--project", rootAppsProject},
			expect: []rootAppsExpectedRequest{
				{method: http.MethodGet, path: "/apis/resourcemanager.miloapis.com/v1alpha1/projects/factory-project/control-plane/apis/apps.miloapis.com/v1alpha1/apptemplaterevisions", query: "labelSelector=apps.miloapis.com%2Ftemplate-uid%3Dretired-template-uid&limit=1", status: http.StatusOK, body: `{"items":[]}`},
			},
			wantOutput: []string{`"templateUID":"retired-template-uid"`, `"revisions":[]`},
		},
		{
			name: "diff routes through revision lookup and stays read only",
			args: []string{"apps", "templates", "diff", rootAppsName, "--revision", rootAppsRevisionName, "--project", rootAppsProject},
			expect: []rootAppsExpectedRequest{
				{method: http.MethodGet, path: "/apis/resourcemanager.miloapis.com/v1alpha1/projects/factory-project/control-plane/apis/apps.miloapis.com/v1alpha1/apptemplates/web-api", status: http.StatusOK, body: rootAppsCurrent},
				{method: http.MethodGet, path: "/apis/resourcemanager.miloapis.com/v1alpha1/projects/factory-project/control-plane/apis/apps.miloapis.com/v1alpha1/apptemplaterevisions/fixture-template-uid.rev-0001", status: http.StatusOK, body: rootAppsRevisionJSON},
			},
			wantOutput: []string{"previous", "current"},
		},
		{
			name: "restore server dry run preserves exact raw JSON and current identity",
			args: []string{"apps", "templates", "restore", rootAppsName, "--revision", rootAppsRevisionName, "--dry-run=server", "--project", rootAppsProject},
			expect: []rootAppsExpectedRequest{
				{method: http.MethodGet, path: "/apis/resourcemanager.miloapis.com/v1alpha1/projects/factory-project/control-plane/apis/apps.miloapis.com/v1alpha1/apptemplates/web-api", status: http.StatusOK, body: rootAppsCurrent},
				{method: http.MethodGet, path: "/apis/resourcemanager.miloapis.com/v1alpha1/projects/factory-project/control-plane/apis/apps.miloapis.com/v1alpha1/apptemplaterevisions/fixture-template-uid.rev-0001", status: http.StatusOK, body: rootAppsRevisionJSON},
				{method: http.MethodPut, path: "/apis/resourcemanager.miloapis.com/v1alpha1/projects/factory-project/control-plane/apis/apps.miloapis.com/v1alpha1/apptemplates/web-api", query: "dryRun=All", status: http.StatusOK, body: rootAppsCurrent, check: checkRootAppsRestoreBody},
			},
			wantOutput: []string{`"action":"restored"`, `"dryRun":true`},
		},
		{
			name: "delete server dry run carries exact UID and resource version in body",
			args: []string{"apps", "templates", "delete", rootAppsName, "--uid", rootAppsUID, "--resource-version", rootAppsRV, "--dry-run=server", "--project", rootAppsProject},
			expect: []rootAppsExpectedRequest{
				{method: http.MethodDelete, path: "/apis/resourcemanager.miloapis.com/v1alpha1/projects/factory-project/control-plane/apis/apps.miloapis.com/v1alpha1/apptemplates/web-api", status: http.StatusOK, body: rootAppsCurrent, check: checkRootAppsDeleteBody},
			},
			wantOutput: []string{`"action":"deleted"`, `"dryRun":true`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var callsMu sync.Mutex
			calls := 0
			var server *httptest.Server
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/tls-probe" {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				callsMu.Lock()
				index := calls
				calls++
				callsMu.Unlock()
				if r.Header.Get("Authorization") != "Bearer "+rootAppsToken {
					t.Errorf("request %d missing expected root-factory token", index)
				}
				if index >= len(tt.expect) {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.String())
					http.Error(w, "unexpected request", http.StatusInternalServerError)
					return
				}
				expected := tt.expect[index]
				if r.Method != expected.method || r.URL.Path != expected.path || r.URL.RawQuery != expected.query {
					t.Errorf("request %d route mismatch: got %s %s?%s; want %s %s?%s", index, r.Method, r.URL.Path, r.URL.RawQuery, expected.method, expected.path, expected.query)
				}
				if expected.check != nil {
					expected.check(t, r)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(expected.status)
				_, _ = io.WriteString(w, expected.body)
			})
			var caData string
			server, caData = newRootAppsTLSServer(t, handler)
			defer server.Close()
			contextProject := tt.contextProject
			if contextProject == "" {
				contextProject = rootAppsProject
			}
			setupRootAppsFactoryEnv(t, server, caData, contextProject)
			if tt.envProject != "" {
				t.Setenv("DATUM_PROJECT", tt.envProject)
			}
			verifyRootAppsFactoryTLS(t, server)

			output, err := executeRootCommandWithStdout(t, tt.args...)
			if err != nil {
				t.Fatalf("RootCmd execution failed: %v\noutput:\n%s", err, output)
			}
			callsMu.Lock()
			gotCalls := calls
			callsMu.Unlock()
			if gotCalls != len(tt.expect) {
				t.Fatalf("got %d fixture requests, want %d; output=%s", gotCalls, len(tt.expect), output)
			}
			for _, want := range tt.wantOutput {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q:\n%s", want, output)
				}
			}
		})
	}
}

func setupRootAppsFactoryEnv(t *testing.T, server *httptest.Server, caData, contextProject string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv(updatecheck.EnvDisable, "1")
	t.Setenv(datumconfig.SessionEnvVar, "")
	t.Setenv("DATUM_PROJECT", "")
	t.Setenv("DATUM_ORGANIZATION", "")
	keyring.MockInit()
	t.Cleanup(datumconfig.ClearSessionOverride)

	credentials, err := json.Marshal(authutil.StoredCredentials{
		Hostname:    "auth.fixture.invalid",
		APIHostname: server.URL,
		Subject:     "fixture-subject-uid",
		Token:       &oauth2.Token{AccessToken: rootAppsToken, Expiry: time.Now().Add(time.Hour)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := keyring.Set(authutil.ServiceName, rootAppsKey, string(credentials)); err != nil {
		t.Fatalf("store fixture credential: %v", err)
	}

	sessionName := datumconfig.SessionName(rootAppsEmail, "127.0.0.1")
	cfg := datumconfig.NewV1Beta1()
	cfg.Sessions = []datumconfig.Session{{
		Name:      sessionName,
		UserKey:   rootAppsKey,
		UserEmail: rootAppsEmail,
		Endpoint: datumconfig.Endpoint{
			Server:                   server.URL,
			TLSServerName:            "localhost",
			CertificateAuthorityData: caData,
		},
	}}
	contextName := datumconfig.QualifiedContextName(sessionName, contextProject)
	cfg.Contexts = []datumconfig.DiscoveredContext{{Name: contextName, Session: sessionName, ProjectID: contextProject}}
	cfg.ActiveSession = sessionName
	cfg.CurrentContext = contextName
	if err := datumconfig.SaveV1Beta1(cfg); err != nil {
		t.Fatalf("save isolated CLI config: %v", err)
	}
}

func verifyRootAppsFactoryTLS(t *testing.T, server *httptest.Server) {
	t.Helper()
	factory, err := client.NewDatumFactory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	*factory.ConfigFlags.Project = rootAppsProject
	config, err := factory.ConfigFlags.ToRESTConfig()
	if err != nil {
		t.Fatalf("resolve real Datum factory REST config: %v", err)
	}
	if config.Host != server.URL+rootAppsControlPlanePath(rootAppsProject) {
		t.Fatalf("factory host=%q, want project control-plane URL", config.Host)
	}
	if config.Insecure {
		t.Fatal("factory REST config disabled TLS verification")
	}
	if len(config.CAData) == 0 || config.ServerName != "localhost" {
		t.Fatalf("factory TLS config missing CA or SNI: ca=%d bytes serverName=%q", len(config.CAData), config.ServerName)
	}
	httpClient, err := rest.HTTPClientFor(config)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/tls-probe", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := httpClient.Do(request)
	if err != nil {
		t.Fatalf("real Datum factory TLS probe: %v", err)
	}
	_ = response.Body.Close()
}

func newRootAppsTLSServer(t *testing.T, handler http.Handler) (*httptest.Server, string) {
	t.Helper()
	now := time.Now()
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "datumctl fixture root"},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatal(err)
	}
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:    now.Add(-time.Minute),
		NotAfter:     now.Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	leafKeyDER, err := x509.MarshalPKCS8PrivateKey(leafKey)
	if err != nil {
		t.Fatal(err)
	}
	serverCert, err := tls.X509KeyPair(
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER}),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: leafKeyDER}),
	)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(handler)
	server.TLS = &tls.Config{Certificates: []tls.Certificate{serverCert}, MinVersion: tls.VersionTLS12}
	server.StartTLS()
	return server, base64.StdEncoding.EncodeToString(caPEM)
}

func executeRootCommandWithStdout(t *testing.T, args ...string) (string, error) {
	t.Helper()
	previous := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = previous
		_ = writer.Close()
		_ = reader.Close()
	}()
	root := RootCmd()
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.SetArgs(args)
	commandCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	commandErr := root.ExecuteContext(commandCtx)
	_ = writer.Close()
	os.Stdout = previous
	output, readErr := io.ReadAll(reader)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if commandErr != nil {
		var formatted bytes.Buffer
		customerrors.Format(&formatted, commandErr, customerrors.FormatHuman, 0)
		output = append(output, formatted.Bytes()...)
	}
	return string(output), commandErr
}

func rootAppsControlPlanePath(project string) string {
	return "/apis/resourcemanager.miloapis.com/v1alpha1/projects/" + project + "/control-plane"
}

func checkRootAppsRestoreBody(t *testing.T, request *http.Request) {
	t.Helper()
	if got := request.Header.Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Errorf("restore Content-Type=%q, want application/json", got)
	}
	var body struct {
		Metadata struct {
			UID             string `json:"uid"`
			ResourceVersion string `json:"resourceVersion"`
		} `json:"metadata"`
		Spec struct {
			DefinitionJSON string `json:"definitionJSON"`
			Digest         string `json:"digest"`
		} `json:"spec"`
	}
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		t.Errorf("decode restore body: %v", err)
		return
	}
	if body.Metadata.UID != rootAppsUID || body.Metadata.ResourceVersion != rootAppsRV {
		t.Errorf("restore preconditions changed: UID=%q RV=%q", body.Metadata.UID, body.Metadata.ResourceVersion)
	}
	if body.Spec.DefinitionJSON != rootAppsRevisionDefinition ||
		body.Spec.Digest != rootAppsDigest(rootAppsRevisionDefinition) {
		t.Errorf("restore changed exact definition envelope: %#v", body.Spec)
	}
}

func checkRootAppsDeleteBody(t *testing.T, request *http.Request) {
	t.Helper()
	if request.URL.RawQuery != "" {
		t.Errorf("DELETE options must be in the body, got query %q", request.URL.RawQuery)
	}
	var options metav1.DeleteOptions
	if err := json.NewDecoder(request.Body).Decode(&options); err != nil {
		t.Errorf("decode DeleteOptions body: %v", err)
		return
	}
	if options.Preconditions == nil || options.Preconditions.UID == nil || *options.Preconditions.UID != rootAppsUID || options.Preconditions.ResourceVersion == nil || *options.Preconditions.ResourceVersion != rootAppsRV {
		t.Errorf("DELETE preconditions were not preserved: %#v", options.Preconditions)
	}
	if len(options.DryRun) != 1 || options.DryRun[0] != metav1.DryRunAll {
		t.Errorf("DELETE dry-run options were not preserved: %#v", options.DryRun)
	}
}
