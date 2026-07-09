package serviceactivation

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	servicesv1alpha1 "go.miloapis.com/service-catalog/api/v1alpha1"
)

// EntitlementClient is the narrow surface the flow needs against a project's
// virtual control plane. Keeping it minimal lets tests inject a fake that
// simulates catalog-absent, already-exists, and watch-event sequences without a
// real API server. ServiceEntitlements are cluster-scoped, so no namespace is
// carried.
type EntitlementClient interface {
	// List returns every ServiceEntitlement visible in the control plane.
	List(ctx context.Context) (*servicesv1alpha1.ServiceEntitlementList, error)
	// Get fetches a single entitlement by name.
	Get(ctx context.Context, name string) (*servicesv1alpha1.ServiceEntitlement, error)
	// Create submits a new entitlement.
	Create(ctx context.Context, e *servicesv1alpha1.ServiceEntitlement) error
	// Delete removes an entitlement.
	Delete(ctx context.Context, e *servicesv1alpha1.ServiceEntitlement) error
	// Watch returns a watcher scoped to a single entitlement by name, seeded at
	// resourceVersion (empty means "from now").
	Watch(ctx context.Context, name, resourceVersion string) (watch.Interface, error)
}

// restClient adapts a controller-runtime client.WithWatch to EntitlementClient.
type restClient struct {
	c client.WithWatch
}

// NewRESTClient builds an EntitlementClient from a Kubernetes REST config. This
// is the auth seam: callers supply a config already carrying the control-plane
// host and a bearer token (plugins from the datumctl credentials helper, core
// from its native config).
func NewRESTClient(cfg *rest.Config) (EntitlementClient, error) {
	scheme := runtime.NewScheme()
	if err := servicesv1alpha1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("registering services scheme: %w", err)
	}
	c, err := client.NewWithWatch(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("building entitlement client: %w", err)
	}
	return &restClient{c: c}, nil
}

func (r *restClient) List(ctx context.Context) (*servicesv1alpha1.ServiceEntitlementList, error) {
	var list servicesv1alpha1.ServiceEntitlementList
	if err := r.c.List(ctx, &list); err != nil {
		return nil, err
	}
	return &list, nil
}

func (r *restClient) Get(ctx context.Context, name string) (*servicesv1alpha1.ServiceEntitlement, error) {
	var e servicesv1alpha1.ServiceEntitlement
	if err := r.c.Get(ctx, types.NamespacedName{Name: name}, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *restClient) Create(ctx context.Context, e *servicesv1alpha1.ServiceEntitlement) error {
	return r.c.Create(ctx, e)
}

func (r *restClient) Delete(ctx context.Context, e *servicesv1alpha1.ServiceEntitlement) error {
	return r.c.Delete(ctx, e)
}

func (r *restClient) Watch(ctx context.Context, name, resourceVersion string) (watch.Interface, error) {
	return r.c.Watch(ctx, &servicesv1alpha1.ServiceEntitlementList{}, &client.ListOptions{
		// Scope the watch to the single named entitlement and resume from the
		// create response's resourceVersion so the first status write is not
		// missed between the create and the watch establishing.
		Raw: &metav1.ListOptions{
			FieldSelector:   "metadata.name=" + name,
			ResourceVersion: resourceVersion,
		},
	})
}
