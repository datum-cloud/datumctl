package client

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"k8s.io/client-go/rest"

	"go.datum.net/datumctl/internal/authutil"
)

func NewForProject(ctx context.Context, projectID, defaultNamespace string) (*K8sClient, error) {
	cfg, err := restConfigFor(ctx, "", projectID)
	if err != nil {
		return nil, err
	}
	k, err := NewK8sFromRESTConfig(cfg)
	if err != nil {
		return nil, err
	}
	k.Namespace = defaultNamespace
	return k, nil
}

func NewForOrg(ctx context.Context, orgID, defaultNamespace string) (*K8sClient, error) {
	cfg, err := restConfigFor(ctx, orgID, "")
	if err != nil {
		return nil, err
	}
	k, err := NewK8sFromRESTConfig(cfg)
	if err != nil {
		return nil, err
	}
	k.Namespace = defaultNamespace
	return k, nil
}

func restConfigFor(ctx context.Context, organizationID, projectID string) (*rest.Config, error) {
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
	case organizationID != "" && projectID == "":
		host = fmt.Sprintf("https://%s/apis/resourcemanager.miloapis.com/v1alpha1/organizations/%s/control-plane",
			apiHostname, organizationID)
	case projectID != "" && organizationID == "":
		host = fmt.Sprintf("https://%s/apis/resourcemanager.miloapis.com/v1alpha1/projects/%s/control-plane",
			apiHostname, projectID)
	default:
		return nil, fmt.Errorf("exactly one of organizationID or projectID must be provided")
	}

	return &rest.Config{
		Host: host,
		WrapTransport: func(rt http.RoundTripper) http.RoundTripper {
			return &oauth2.Transport{Source: tknSrc, Base: rt}
		},
	}, nil
}
