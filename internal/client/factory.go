package client

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/pflag"
	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/datumconfig"
	"go.datum.net/datumctl/internal/miloapi"
	"golang.org/x/oauth2"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	diskcached "k8s.io/client-go/discovery/cached/disk"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
	"k8s.io/kubectl/pkg/cmd/util"
)

type DatumCloudFactory struct {
	util.Factory
	ConfigFlags *CustomConfigFlags
}

type CustomConfigFlags struct {
	*genericclioptions.ConfigFlags
	Project      *string
	Organization *string
	PlatformWide *bool
	Context      context.Context
}

func (factory *DatumCloudFactory) AddFlags(flags *pflag.FlagSet) {
	factory.ConfigFlags.AddFlags(flags)
	flags.StringVar(factory.ConfigFlags.Project, "project", "", "project name")
	flags.StringVar(factory.ConfigFlags.Organization, "organization", "", "organization name")
	flags.BoolVar(factory.ConfigFlags.PlatformWide, "platform-wide", false, "access the platform root instead of a project or organization control plane")
}

func (factory *DatumCloudFactory) AddFlagMutualExclusions(cmd interface{ MarkFlagsMutuallyExclusive(...string) }) {
	cmd.MarkFlagsMutuallyExclusive("project", "organization", "platform-wide")
}

func (c *CustomConfigFlags) ToRESTConfig() (*rest.Config, error) {
	config, err := c.ConfigFlags.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	ctxEntry, session, err := c.loadDatumContext()
	if err != nil {
		return nil, err
	}
	if c.APIServer != nil && *c.APIServer != "" {
		config.Host = *c.APIServer
	}
	if c.Insecure != nil && *c.Insecure {
		config.Insecure = true
	}
	if c.TLSServerName != nil && *c.TLSServerName != "" {
		config.ServerName = *c.TLSServerName
	}

	userKey, err := c.resolveUserKey(session)
	if err != nil {
		return nil, err
	}
	tknSrc, err := authutil.GetTokenSourceForUser(c.Context, userKey)
	if err != nil {
		return nil, err
	}

	config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		return &oauth2.Transport{Source: tknSrc, Base: rt}
	}

	baseServer, err := c.resolveBaseServer(userKey, session)
	if err != nil {
		return nil, err
	}

	projectID, organizationID, platformWide, err := c.resolveScope(ctxEntry)
	if err != nil {
		return nil, err
	}

	switch {
	case platformWide:
		config.Host = baseServer
	case organizationID != "":
		config.Host = miloapi.OrgControlPlaneURL(baseServer, organizationID)
	case projectID != "":
		config.Host = miloapi.ProjectControlPlaneURL(baseServer, projectID)
	default:
		userID, err := authutil.GetUserIDFromTokenForUser(userKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get user ID from token: %w", err)
		}
		config.Host = miloapi.UserControlPlaneURL(baseServer, userID)
	}

	if session != nil {
		ep := session.Endpoint
		if (c.TLSServerName == nil || *c.TLSServerName == "") && ep.TLSServerName != "" {
			config.ServerName = ep.TLSServerName
		}
		if (c.Insecure == nil || !*c.Insecure) && ep.InsecureSkipTLSVerify {
			config.Insecure = true
		}
		if len(config.CAData) == 0 && ep.CertificateAuthorityData != "" {
			decoded, err := base64.StdEncoding.DecodeString(ep.CertificateAuthorityData)
			if err != nil {
				return nil, fmt.Errorf("decode certificate authority data for session %q: %w", session.Name, err)
			}
			config.CAData = decoded
		}
	}

	return config, nil
}

func (c *CustomConfigFlags) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	restConfig, err := c.ToRESTConfig()
	if err != nil {
		panic(err)
	}
	kubeConfig := &api.Config{
		Clusters: map[string]*api.Cluster{
			"inmemory": {
				Server:                   restConfig.Host,
				CertificateAuthorityData: restConfig.CAData,
				InsecureSkipTLSVerify:    restConfig.Insecure,
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			"inmemory": {
				Token:                 restConfig.BearerToken,
				ClientCertificateData: restConfig.CertData,
				ClientKeyData:         restConfig.KeyData,
			},
		},
		Contexts: map[string]*api.Context{
			"inmemory": {
				Cluster:   "inmemory",
				AuthInfo:  "inmemory",
				Namespace: "",
			},
		},
		CurrentContext: "inmemory",
	}

	overrides := &clientcmd.ConfigOverrides{}

	if c.ConfigFlags.Namespace != nil && *c.ConfigFlags.Namespace != "" {
		overrides.Context.Namespace = *c.ConfigFlags.Namespace
	} else {
		ctxEntry, _, err := c.loadDatumContext()
		if err == nil && ctxEntry != nil && ctxEntry.Namespace != "" {
			overrides.Context.Namespace = ctxEntry.Namespace
		}
	}

	if c.APIServer != nil && *c.APIServer != "" {
		overrides.ClusterInfo.Server = *c.APIServer
	}

	if c.Insecure != nil && *c.Insecure {
		overrides.ClusterInfo.InsecureSkipTLSVerify = true
	}

	if c.Impersonate != nil && *c.Impersonate != "" {
		overrides.AuthInfo.Impersonate = *c.Impersonate
	}

	if c.ImpersonateGroup != nil && len(*c.ImpersonateGroup) > 0 {
		overrides.AuthInfo.ImpersonateGroups = *c.ImpersonateGroup
	}

	if c.ImpersonateUID != nil && *c.ImpersonateUID != "" {
		overrides.AuthInfo.ImpersonateUID = *c.ImpersonateUID
	}

	return clientcmd.NewDefaultClientConfig(*kubeConfig, overrides)
}

// ToDiscoveryClient overrides the embedded ConfigFlags method so that discovery
// requests are sent to the scope-correct control plane (project, org, etc.)
// rather than always using the user control plane. The embedded method calls
// f.ToRESTConfig() on *ConfigFlags, which bypasses our CustomConfigFlags
// override — this method fixes that by calling c.ToRESTConfig() directly.
func (c *CustomConfigFlags) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	config, err := c.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	cacheDir := filepath.Join(homedir.HomeDir(), ".kube", "cache")
	httpCacheDir := filepath.Join(cacheDir, "http")
	discoveryCacheDir := filepath.Join(cacheDir, "discovery", config.Host)

	return diskcached.NewCachedDiscoveryClientForConfig(config, discoveryCacheDir, httpCacheDir, 6*time.Hour)
}

// ToRESTMapper overrides the embedded ConfigFlags method for the same reason as
// ToDiscoveryClient: the embedded toRESTMapper calls f.ToDiscoveryClient() on
// *ConfigFlags, which would bypass our ToDiscoveryClient override.
func (c *CustomConfigFlags) ToRESTMapper() (meta.RESTMapper, error) {
	discoveryClient, err := c.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(discoveryClient)
	expander := restmapper.NewShortcutExpander(mapper, discoveryClient, nil)
	return expander, nil
}

// loadDatumContext resolves the active v1beta1 session and current context,
// if any. Returns (nil, nil, nil) when no session exists, letting callers
// fall back to the user-key path which bootstraps from keyring if needed.
func (c *CustomConfigFlags) loadDatumContext() (*datumconfig.DiscoveredContext, *datumconfig.Session, error) {
	cfg, err := datumconfig.LoadAuto()
	if err != nil {
		return nil, nil, err
	}
	ctxEntry := cfg.CurrentContextEntry()
	if ctxEntry == nil {
		return nil, nil, nil
	}
	session := cfg.SessionByName(ctxEntry.Session)
	return ctxEntry, session, nil
}

func (c *CustomConfigFlags) resolveUserKey(session *datumconfig.Session) (string, error) {
	if session != nil && session.UserKey != "" {
		return session.UserKey, nil
	}
	return authutil.GetUserKey()
}

func (c *CustomConfigFlags) resolveBaseServer(userKey string, session *datumconfig.Session) (string, error) {
	if c.APIServer != nil && *c.APIServer != "" {
		return datumconfig.CleanBaseServer(datumconfig.EnsureScheme(*c.APIServer)), nil
	}
	if session != nil && session.Endpoint.Server != "" {
		return datumconfig.CleanBaseServer(datumconfig.EnsureScheme(session.Endpoint.Server)), nil
	}
	apiHostname, err := authutil.GetAPIHostnameForUser(userKey)
	if err != nil {
		return "", err
	}
	return datumconfig.CleanBaseServer(datumconfig.EnsureScheme(apiHostname)), nil
}

// resolveScope picks the org/project/platform-wide scope for the request. It
// tries, in order: flags → environment variables → active context. Returns an
// error if the inputs are contradictory.
func (c *CustomConfigFlags) resolveScope(ctxEntry *datumconfig.DiscoveredContext) (string, string, bool, error) {
	platformWide := c.PlatformWide != nil && *c.PlatformWide
	projectID := ""
	organizationID := ""

	if c.Project != nil && *c.Project != "" {
		projectID = *c.Project
	}
	if c.Organization != nil && *c.Organization != "" {
		organizationID = *c.Organization
	}

	if platformWide && (projectID != "" || organizationID != "") {
		return "", "", false, fmt.Errorf("--platform-wide cannot be used with --project or --organization")
	}

	if projectID == "" && organizationID == "" && !platformWide {
		if envProject := os.Getenv("DATUM_PROJECT"); envProject != "" {
			projectID = envProject
		}
		if envOrg := os.Getenv("DATUM_ORGANIZATION"); envOrg != "" {
			organizationID = envOrg
		}
		if projectID != "" && organizationID != "" {
			return "", "", false, fmt.Errorf("DATUM_PROJECT and DATUM_ORGANIZATION cannot both be set")
		}
	}

	// Active context fills in when flags/env don't. Project scope wins over
	// org scope when the context is project-scoped.
	if projectID == "" && organizationID == "" && !platformWide && ctxEntry != nil {
		if ctxEntry.ProjectID != "" {
			projectID = ctxEntry.ProjectID
		} else {
			organizationID = ctxEntry.OrganizationID
		}
	}

	if projectID != "" && organizationID != "" {
		return "", "", false, fmt.Errorf("exactly one of --project or --organization must be provided")
	}

	return projectID, organizationID, platformWide, nil
}

func NewDatumFactory(ctx context.Context) (*DatumCloudFactory, error) {
	baseConfigFlags := genericclioptions.NewConfigFlags(true)
	baseConfigFlags = baseConfigFlags.WithWrapConfigFn(func(*rest.Config) *rest.Config {
		config, err := NewRestConfig(ctx)
		if err != nil {
			panic(err)
		}
		return config
	})
	baseConfigFlags.KubeConfig = nil
	baseConfigFlags.CacheDir = nil
	baseConfigFlags.KeyFile = nil
	baseConfigFlags.CertFile = nil
	baseConfigFlags.ClusterName = nil
	baseConfigFlags.Context = nil
	configFlags := &CustomConfigFlags{
		ConfigFlags: baseConfigFlags,
		Context:     ctx,
		Project: func() *string {
			m := ""
			return &m
		}(),
		Organization: func() *string {
			m := ""
			return &m
		}(),
		PlatformWide: func() *bool {
			b := false
			return &b
		}(),
	}
	f := util.NewFactory(configFlags)
	return &DatumCloudFactory{
		Factory:     f,
		ConfigFlags: configFlags,
	}, nil
}
