package client

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/pflag"
	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/datumconfig"
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
	ctxEntry, clusterEntry, err := c.loadDatumContext()
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

	userKey, err := c.resolveUserKeyForCluster(clusterEntry)
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

	baseServer, err := c.resolveBaseServer(userKey, clusterEntry)
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
		config.Host = fmt.Sprintf("%s/apis/resourcemanager.miloapis.com/v1alpha1/organizations/%s/control-plane",
			baseServer, organizationID)
	case projectID != "":
		config.Host = fmt.Sprintf("%s/apis/resourcemanager.miloapis.com/v1alpha1/projects/%s/control-plane",
			baseServer, projectID)
	default:
		userID, err := authutil.GetUserIDFromTokenForUser(userKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get user ID from token: %w", err)
		}
		config.Host = fmt.Sprintf("%s/apis/iam.miloapis.com/v1alpha1/users/%s/control-plane", baseServer, userID)
	}

	if clusterEntry != nil {
		if (c.TLSServerName == nil || *c.TLSServerName == "") && clusterEntry.Cluster.TLSServerName != "" {
			config.ServerName = clusterEntry.Cluster.TLSServerName
		}
		if (c.Insecure == nil || !*c.Insecure) && clusterEntry.Cluster.InsecureSkipTLSVerify {
			config.Insecure = true
		}
		if len(config.CAData) == 0 && clusterEntry.Cluster.CertificateAuthorityData != "" {
			decoded, err := base64.StdEncoding.DecodeString(clusterEntry.Cluster.CertificateAuthorityData)
			if err != nil {
				return nil, fmt.Errorf("decode certificate authority data for cluster %q: %w", clusterEntry.Name, err)
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

	// Create overrides from ConfigFlags - THIS IS THE KEY
	overrides := &clientcmd.ConfigOverrides{}

	if c.ConfigFlags.Namespace != nil && *c.ConfigFlags.Namespace != "" {
		overrides.Context.Namespace = *c.ConfigFlags.Namespace
	} else {
		ctxEntry, _, err := c.loadDatumContext()
		if err == nil && ctxEntry != nil && ctxEntry.Context.Namespace != "" {
			overrides.Context.Namespace = ctxEntry.Context.Namespace
		}
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

func (c *CustomConfigFlags) loadDatumContext() (*datumconfig.NamedContext, *datumconfig.NamedCluster, error) {
	_, ctxEntry, clusterEntry, err := datumconfig.LoadCurrentContext()
	if err != nil {
		return nil, nil, err
	}
	return ctxEntry, clusterEntry, nil
}

func (c *CustomConfigFlags) resolveUserKeyForCluster(clusterEntry *datumconfig.NamedCluster) (string, error) {
	if clusterEntry != nil && clusterEntry.Name != "" {
		return authutil.GetActiveUserKeyForCluster(clusterEntry.Name)
	}
	return "", authutil.ErrNoCurrentContext
}

func (c *CustomConfigFlags) resolveBaseServer(userKey string, clusterEntry *datumconfig.NamedCluster) (string, error) {
	if c.APIServer != nil && *c.APIServer != "" {
		return datumconfig.CleanBaseServer(datumconfig.EnsureScheme(*c.APIServer)), nil
	}
	if clusterEntry != nil && clusterEntry.Cluster.Server != "" {
		return datumconfig.CleanBaseServer(datumconfig.EnsureScheme(clusterEntry.Cluster.Server)), nil
	}
	apiHostname, err := authutil.GetAPIHostnameForUser(userKey)
	if err != nil {
		return "", err
	}
	return datumconfig.CleanBaseServer(datumconfig.EnsureScheme(apiHostname)), nil
}

func (c *CustomConfigFlags) resolveScope(ctxEntry *datumconfig.NamedContext) (string, string, bool, error) {
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

	if projectID == "" && organizationID == "" && !platformWide && ctxEntry != nil {
		projectID = ctxEntry.Context.ProjectID
		organizationID = ctxEntry.Context.OrganizationID
	}

	if projectID != "" && organizationID != "" {
		if c.Project != nil && *c.Project != "" || c.Organization != nil && *c.Organization != "" {
			return "", "", false, fmt.Errorf("exactly one of organizationID or projectID must be provided")
		}
		fmt.Fprintf(os.Stderr, "Warning: context has both project_id and organization_id set; using project_id and ignoring organization_id.\n")
		organizationID = ""
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
