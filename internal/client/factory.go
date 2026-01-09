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
	RestConfig  *rest.Config
}

type CustomConfigFlags struct {
	*genericclioptions.ConfigFlags
	Project      *string
	Organization *string
	restConfig   *rest.Config
	tokenSrc     oauth2.TokenSource
}

func (factory *DatumCloudFactory) AddFlags(flags *pflag.FlagSet) {
	factory.ConfigFlags.AddFlags(flags)
	flags.StringVar(factory.ConfigFlags.Project, "project", "", "project name")
	flags.StringVar(factory.ConfigFlags.Organization, "organization", "", "organization name")
}

func (c *CustomConfigFlags) ToRESTConfig() (*rest.Config, error) {
	config := rest.CopyConfig(c.restConfig)
	if c.APIServer != nil && *c.APIServer != "" {
		config.Host = *c.APIServer
	}
	if c.Insecure != nil && *c.Insecure {
		config.Insecure = true
	}
	if c.TLSServerName != nil && *c.TLSServerName != "" {
		config.ServerName = *c.TLSServerName
	}

	config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		return &oauth2.Transport{Source: c.tokenSrc, Base: rt}
	}

	apiHostname, err := authutil.GetAPIHostname()
	if err != nil {
		return nil, err
	}
	switch {
	case (c.Project == nil || *c.Project == "") && (c.Organization == nil || *c.Organization == ""):
	case (c.Project == nil || *c.Project == "") && (c.Organization != nil || *c.Organization != ""):
		config.Host = fmt.Sprintf("https://%s/apis/resourcemanager.miloapis.com/v1alpha1/organizations/%s/control-plane",
			apiHostname, *c.Organization)
	case (c.Project != nil || *c.Project != "") && (c.Organization == nil || *c.Organization == ""):
		config.Host = fmt.Sprintf("https://%s/apis/resourcemanager.miloapis.com/v1alpha1/projects/%s/control-plane",
			apiHostname, *c.Project)
	default:
		return nil, fmt.Errorf("exactly one of organizationID or projectID must be provided")
	}

	return config, nil
}

func (c *CustomConfigFlags) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	kubeConfig := &api.Config{
		Clusters: map[string]*api.Cluster{
			"inmemory": {
				Server:                   c.restConfig.Host,
				CertificateAuthorityData: c.restConfig.CAData,
				InsecureSkipTLSVerify:    c.restConfig.Insecure,
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			"inmemory": {
				Token:                 c.restConfig.BearerToken,
				ClientCertificateData: c.restConfig.CertData,
				ClientKeyData:         c.restConfig.KeyData,
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

func NewDatumFactory(ctx context.Context, restConfig *rest.Config) (*DatumCloudFactory, error) {
	baseConfigFlags := genericclioptions.NewConfigFlags(true)
	baseConfigFlags = baseConfigFlags.WithWrapConfigFn(func(*rest.Config) *rest.Config {
		return restConfig
	})
	baseConfigFlags.KubeConfig = nil
	baseConfigFlags.CacheDir = nil
	baseConfigFlags.KeyFile = nil
	baseConfigFlags.CertFile = nil
	baseConfigFlags.ClusterName = nil
	baseConfigFlags.Context = nil
	tknSrc, err := authutil.GetTokenSource(ctx)
	if err != nil {
		return nil, err
	}
	configFlags := &CustomConfigFlags{
		ConfigFlags: baseConfigFlags,
		restConfig:  restConfig,
		tokenSrc:    tknSrc,
		Project: func() *string {
			m := ""
			return &m
		}(),
		Organization: func() *string {
			m := ""
			return &m
		}(),
	}
	f := util.NewFactory(configFlags)
	return &DatumCloudFactory{
		Factory:     f,
		ConfigFlags: configFlags,
		RestConfig:  restConfig,
	}, nil
}
