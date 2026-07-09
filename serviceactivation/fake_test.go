package serviceactivation

import (
	"bytes"
	"context"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apimachinery/pkg/watch"

	servicesv1alpha1 "go.miloapis.com/service-catalog/api/v1alpha1"
)

var entitlementGR = schema.GroupResource{Group: "services.miloapis.com", Resource: "serviceentitlements"}

// catalogAbsentErr is the real discovery failure returned when the services API
// group is not served, so catalogAbsent (apimeta.IsNoMatchError) recognizes it.
func catalogAbsentErr() error {
	return &apimeta.NoResourceMatchError{
		PartialResource: schema.GroupVersionResource{
			Group: "services.miloapis.com", Version: "v1alpha1", Resource: "serviceentitlements",
		},
	}
}

// fakeClient is a scriptable EntitlementClient for exercising the flow without a
// real API server. Each response can be configured, and mutating calls are
// recorded so tests can assert that the non-interactive paths never mutate.
type fakeClient struct {
	listResult *servicesv1alpha1.ServiceEntitlementList
	listErr    error

	getResult *servicesv1alpha1.ServiceEntitlement
	getErr    error

	createErr error
	watchEmit []watch.Event

	creates []*servicesv1alpha1.ServiceEntitlement
	deletes []*servicesv1alpha1.ServiceEntitlement
}

func (f *fakeClient) List(ctx context.Context) (*servicesv1alpha1.ServiceEntitlementList, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.listResult != nil {
		return f.listResult, nil
	}
	return &servicesv1alpha1.ServiceEntitlementList{}, nil
}

func (f *fakeClient) Get(ctx context.Context, name string) (*servicesv1alpha1.ServiceEntitlement, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.getResult != nil {
		return f.getResult, nil
	}
	return nil, apierrors.NewNotFound(entitlementGR, name)
}

func (f *fakeClient) Create(ctx context.Context, e *servicesv1alpha1.ServiceEntitlement) error {
	f.creates = append(f.creates, e.DeepCopy())
	if f.createErr != nil {
		return f.createErr
	}
	e.ResourceVersion = "1"
	return nil
}

func (f *fakeClient) Delete(ctx context.Context, e *servicesv1alpha1.ServiceEntitlement) error {
	f.deletes = append(f.deletes, e.DeepCopy())
	return nil
}

func (f *fakeClient) Watch(ctx context.Context, name, resourceVersion string) (watch.Interface, error) {
	ch := make(chan watch.Event, len(f.watchEmit)+1)
	for _, ev := range f.watchEmit {
		ch <- ev
	}
	return &bufferedWatch{ch: ch}, nil
}

// bufferedWatch replays a fixed set of events and then blocks (its channel stays
// open) so a bounded wait resolves on the first matching event rather than
// falling through to the closed-channel re-read path.
type bufferedWatch struct {
	ch chan watch.Event
}

func (b *bufferedWatch) Stop()                          {}
func (b *bufferedWatch) ResultChan() <-chan watch.Event { return b.ch }

// modifiedEvent wraps an entitlement in a MODIFIED watch event.
func modifiedEvent(e *servicesv1alpha1.ServiceEntitlement) watch.Event {
	return watch.Event{Type: watch.Modified, Object: e}
}

// alreadyExistsErr mimics a concurrent create winning the race.
func alreadyExistsErr() error {
	return apierrors.NewAlreadyExists(entitlementGR, "compute")
}

// invalidCreateErr mimics the admission webhook rejecting a create for a missing
// or unpublished service (apierrors.NewInvalid).
func invalidCreateErr() error {
	return apierrors.NewInvalid(
		schema.GroupKind{Group: "services.miloapis.com", Kind: "ServiceEntitlement"},
		"compute",
		field.ErrorList{field.NotFound(field.NewPath("spec", "serviceRef", "name"), "compute")},
	)
}

// listOf wraps entitlements in a list.
func listOf(items ...*servicesv1alpha1.ServiceEntitlement) *servicesv1alpha1.ServiceEntitlementList {
	list := &servicesv1alpha1.ServiceEntitlementList{}
	for _, it := range items {
		list.Items = append(list.Items, *it)
	}
	return list
}

// testIO returns IOStreams over buffers with a scripted stdin, plus the stderr
// buffer for assertions.
func testIO(stdin string, interactive bool) (IOStreams, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	io := IOStreams{In: strings.NewReader(stdin), Out: &out, Err: &errb}.WithInteractive(interactive)
	return io, &out, &errb
}

// withCreationAge stamps a creation timestamp for age-sensitive assertions.
func withCreationAge(e *servicesv1alpha1.ServiceEntitlement, t metav1.Time) *servicesv1alpha1.ServiceEntitlement {
	e.CreationTimestamp = t
	return e
}
