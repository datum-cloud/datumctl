package client

import (
	"context"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/kubectl/pkg/cmd/util"
)

type MyFactory struct {
	util.Factory
	ConfigFlags *genericclioptions.ConfigFlags
	RestConfig  *rest.Config
}

type customConfigFlags struct {
	*genericclioptions.ConfigFlags
	restConfig *rest.Config
}

func (c *customConfigFlags) ToRESTConfig() (*rest.Config, error) {
	config := rest.CopyConfig(c.restConfig)
	if c.APIServer != nil && *c.APIServer != "" {
		config.Host = *c.APIServer
	}
	if c.BearerToken != nil && *c.BearerToken != "" {
		config.BearerToken = *c.BearerToken
	}
	if c.Insecure != nil && *c.Insecure {
		config.Insecure = true
	}
	if c.TLSServerName != nil && *c.TLSServerName != "" {
		config.ServerName = *c.TLSServerName
	}

	return config, nil
}

func (c *customConfigFlags) ToRawKubeConfigLoader() clientcmd.ClientConfig {
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
				Namespace: *c.Namespace,
			},
		},
		CurrentContext: "inmemory",
	}

	// Create overrides from ConfigFlags - THIS IS THE KEY
	overrides := &clientcmd.ConfigOverrides{}

	if c.Namespace != nil {
		overrides.Context.Namespace = *c.Namespace
	}

	// Apply cluster overrides if set
	if c.APIServer != nil && *c.APIServer != "" {
		overrides.ClusterInfo.Server = *c.APIServer
	}

	if c.Insecure != nil && *c.Insecure {
		overrides.ClusterInfo.InsecureSkipTLSVerify = true
	}

	if c.BearerToken != nil && *c.BearerToken != "" {
		overrides.AuthInfo.Token = *c.BearerToken
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

func NewDatumFactory(ctx context.Context, restConfig *rest.Config) (*MyFactory, error) {
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
	configFlags := &customConfigFlags{
		ConfigFlags: baseConfigFlags,
		restConfig:  restConfig,
	}
	f := util.NewFactory(configFlags)
	return &MyFactory{
		Factory:     f,
		ConfigFlags: baseConfigFlags,
		RestConfig:  restConfig,
	}, nil
}
