package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/spf13/pflag"
	"go.datum.net/datumctl/internal/authutil"
	"golang.org/x/oauth2"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
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
	if c.APIServer != nil && *c.APIServer != "" {
		config.Host = *c.APIServer
	}
	if c.Insecure != nil && *c.Insecure {
		config.Insecure = true
	}
	if c.TLSServerName != nil && *c.TLSServerName != "" {
		config.ServerName = *c.TLSServerName
	}

	tknSrc, err := authutil.GetTokenSource(c.Context)
	if err != nil {
		return nil, err
	}

	config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		return &oauth2.Transport{Source: tknSrc, Base: rt}
	}

	apiHostname, err := authutil.GetAPIHostname()
	if err != nil {
		return nil, err
	}

	// Handle platform-wide mode
	isPlatformWide := c.PlatformWide != nil && *c.PlatformWide
	hasProject := c.Project != nil && *c.Project != ""
	hasOrganization := c.Organization != nil && *c.Organization != ""

	switch {
	case isPlatformWide:
		// Platform-wide mode: access the root of the platform
		if hasProject || hasOrganization {
			return nil, fmt.Errorf("--platform-wide cannot be used with --project or --organization")
		}
		config.Host = fmt.Sprintf("https://%s", apiHostname)
	case !hasProject && !hasOrganization:
		// No context specified - default behavior
	case hasOrganization && !hasProject:
		// Organization context
		config.Host = fmt.Sprintf("https://%s/apis/resourcemanager.miloapis.com/v1alpha1/organizations/%s/control-plane",
			apiHostname, *c.Organization)
	case hasProject && !hasOrganization:
		// Project context
		config.Host = fmt.Sprintf("https://%s/apis/resourcemanager.miloapis.com/v1alpha1/projects/%s/control-plane",
			apiHostname, *c.Project)
	default:
		return nil, fmt.Errorf("exactly one of organizationID or projectID must be provided")
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

	// Create overrides from ConfigFlags - THIS IS THE KEY
	overrides := &clientcmd.ConfigOverrides{}

	if c.ConfigFlags.Namespace != nil && *c.ConfigFlags.Namespace != "" {
		overrides.Context.Namespace = *c.ConfigFlags.Namespace
	}

	// Apply cluster overrides if set
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
