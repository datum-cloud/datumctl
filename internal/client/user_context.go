package client

import (
	"context"
	"fmt"
	"net/http"

	resourcemanagerv1alpha1 "go.miloapis.com/milo/pkg/apis/resourcemanager/v1alpha1"
	"golang.org/x/oauth2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/datumconfig"
	"go.datum.net/datumctl/internal/miloapi"
)

// NewUserContextualClient creates a new controller-runtime client configured for the current user's context.
func NewUserContextualClient(ctx context.Context) (client.Client, error) {
	config, err := NewRestConfig(ctx)
	if err != nil {
		return nil, err
	}
	// Create a scheme and add the resourcemanager types
	scheme := runtime.NewScheme()
	if err := resourcemanagerv1alpha1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add resourcemanager types to scheme: %w", err)
	}
	// Create a new controller-runtime client
	return client.New(config, client.Options{Scheme: scheme})
}

func NewRestConfig(ctx context.Context) (*rest.Config, error) {
	userKey, session, err := authutil.GetUserKeyForCurrentSession()
	if err != nil {
		return nil, fmt.Errorf("failed to get user key: %w", err)
	}
	tknSrc, err := authutil.GetTokenSourceForUser(ctx, userKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get token source: %w", err)
	}
	// Get user ID from stored credentials
	userID, err := authutil.GetUserIDFromTokenForUser(userKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID from token: %w", err)
	}

	// Get API hostname — prefer session endpoint, fall back to credentials.
	var apiHostname string
	if session != nil && session.Endpoint.Server != "" {
		apiHostname = datumconfig.StripScheme(session.Endpoint.Server)
	} else {
		apiHostname, err = authutil.GetAPIHostnameForUser(userKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get API hostname: %w", err)
		}
	}

	userContextAPI := miloapi.UserControlPlaneURL(apiHostname, userID)

	// Create Kubernetes client configuration with scheme
	config := &rest.Config{
		Host:      userContextAPI,
		UserAgent: "datumctl",
		WrapTransport: func(rt http.RoundTripper) http.RoundTripper {
			return &oauth2.Transport{
				Source: tknSrc,
				Base:   rt,
			}
		},
	}

	return config, nil
}

