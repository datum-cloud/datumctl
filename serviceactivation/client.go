package serviceactivation

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"

	servicesv1alpha1 "go.miloapis.com/service-catalog/api/v1alpha1"
	versioned "go.miloapis.com/service-catalog/pkg/generated/clientset/versioned"
)

// EntitlementClient is the narrow surface the flow needs against a project's
// virtual control plane. Keeping it minimal lets tests inject the generated
// fake clientset and simulate catalog-absent, already-exists, and watch-event
// sequences without a real API server. ServiceEntitlements are cluster-scoped,
// so no namespace is carried.
type EntitlementClient interface {
	// List returns every ServiceEntitlement visible in the control plane.
	List(ctx context.Context) (*servicesv1alpha1.ServiceEntitlementList, error)
	// Get fetches a single entitlement by name.
	Get(ctx context.Context, name string) (*servicesv1alpha1.ServiceEntitlement, error)
	// Create submits a new entitlement, writing the server's response back into e.
	Create(ctx context.Context, e *servicesv1alpha1.ServiceEntitlement) error
	// Delete removes an entitlement.
	Delete(ctx context.Context, e *servicesv1alpha1.ServiceEntitlement) error
	// Watch returns a watcher scoped to a single entitlement by name, seeded at
	// resourceVersion (empty means "from now").
	Watch(ctx context.Context, name, resourceVersion string) (watch.Interface, error)
}

// clientsetAdapter maps the generated services clientset onto EntitlementClient.
type clientsetAdapter struct {
	cs versioned.Interface
}

// NewClient wraps a generated services clientset (real or fake) as an
// EntitlementClient. Tests pass the generated fake; production passes a
// clientset built from a REST config.
func NewClient(cs versioned.Interface) EntitlementClient {
	return &clientsetAdapter{cs: cs}
}

// NewRESTClient builds an EntitlementClient from a Kubernetes REST config. This
// is the auth seam: callers supply a config already carrying the control-plane
// host and a bearer token (plugins from the datumctl credentials helper, core
// from its native config).
func NewRESTClient(cfg *rest.Config) (EntitlementClient, error) {
	cs, err := versioned.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("building services clientset: %w", err)
	}
	return NewClient(cs), nil
}

// entitlements is the subset of the generated ServiceEntitlementInterface the
// adapter calls. Naming it keeps the methods readable and documents the exact
// surface the flow depends on.
func (a *clientsetAdapter) entitlements() clientEntitlements {
	return a.cs.ServicesV1alpha1().ServiceEntitlements()
}

func (a *clientsetAdapter) List(ctx context.Context) (*servicesv1alpha1.ServiceEntitlementList, error) {
	return a.entitlements().List(ctx, metav1.ListOptions{})
}

func (a *clientsetAdapter) Get(ctx context.Context, name string) (*servicesv1alpha1.ServiceEntitlement, error) {
	return a.entitlements().Get(ctx, name, metav1.GetOptions{})
}

func (a *clientsetAdapter) Create(ctx context.Context, e *servicesv1alpha1.ServiceEntitlement) error {
	created, err := a.entitlements().Create(ctx, e, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	// The clientset returns a fresh object; copy it back so callers see the
	// server-assigned resourceVersion (used to seed the post-create watch).
	*e = *created
	return nil
}

func (a *clientsetAdapter) Delete(ctx context.Context, e *servicesv1alpha1.ServiceEntitlement) error {
	return a.entitlements().Delete(ctx, e.Name, metav1.DeleteOptions{})
}

func (a *clientsetAdapter) Watch(ctx context.Context, name, resourceVersion string) (watch.Interface, error) {
	return a.entitlements().Watch(ctx, metav1.ListOptions{
		// Scope the watch to the single named entitlement and resume from the
		// create response's resourceVersion so the first status write is not
		// missed between the create and the watch establishing.
		FieldSelector:   "metadata.name=" + name,
		ResourceVersion: resourceVersion,
	})
}

// clientEntitlements is the subset of the generated ServiceEntitlementInterface
// the adapter uses. The generated interface is a superset, so it satisfies this.
type clientEntitlements interface {
	List(ctx context.Context, opts metav1.ListOptions) (*servicesv1alpha1.ServiceEntitlementList, error)
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*servicesv1alpha1.ServiceEntitlement, error)
	Create(ctx context.Context, e *servicesv1alpha1.ServiceEntitlement, opts metav1.CreateOptions) (*servicesv1alpha1.ServiceEntitlement, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
}
