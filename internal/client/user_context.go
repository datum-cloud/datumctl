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
	tknSrc, err := authutil.GetTokenSource(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token source: %w", err)
	}
	// Get user ID from stored credentials
	userID, err := authutil.GetUserIDFromToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID from token: %w", err)
	}

	// Get API hostname from stored credentials
	apiHostname, err := authutil.GetAPIHostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get API hostname: %w", err)
	}

	// Build the user-contextual API endpoint
	userContextAPI := fmt.Sprintf("https://%s/apis/iam.miloapis.com/v1alpha1/users/%s/control-plane", apiHostname, userID)

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
