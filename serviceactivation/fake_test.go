package serviceactivation

import (
	"bytes"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apimachinery/pkg/watch"
	k8stesting "k8s.io/client-go/testing"

	servicesv1alpha1 "go.miloapis.com/service-catalog/api/v1alpha1"
	svcfake "go.miloapis.com/service-catalog/pkg/generated/clientset/versioned/fake"
)

const entitlementResource = "serviceentitlements"

var entitlementGR = schema.GroupResource{Group: "services.miloapis.com", Resource: entitlementResource}

// newFake builds an EntitlementClient backed by the generated fake clientset,
// configured by opts, and returns the fake so tests can assert on recorded
// actions. Exercising the real clientsetAdapter over the generated fake is the
// point: it covers the production code path, not a stand-in.
func newFake(opts ...fakeOpt) (EntitlementClient, *svcfake.Clientset) {
	cfg := fakeConfig{}
	for _, o := range opts {
		o(&cfg)
	}
	cs := svcfake.NewSimpleClientset(cfg.seed...)
	if cfg.listErr != nil {
		cs.PrependReactor("list", entitlementResource, func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, cfg.listErr
		})
	}
	if cfg.createErr != nil {
		cs.PrependReactor("create", entitlementResource, func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, cfg.createErr
		})
	}
	if cfg.watchEvents != nil {
		cs.PrependWatchReactor(entitlementResource, func(k8stesting.Action) (bool, watch.Interface, error) {
			fw := watch.NewFakeWithChanSize(len(cfg.watchEvents)+1, false)
			for _, ev := range cfg.watchEvents {
				if ev.Type == watch.Added {
					fw.Add(ev.Object)
				} else {
					fw.Modify(ev.Object)
				}
			}
			return true, fw, nil
		})
	}
	return NewClient(cs), cs
}

type fakeConfig struct {
	seed        []runtime.Object
	listErr     error
	createErr   error
	watchEvents []watch.Event
}

type fakeOpt func(*fakeConfig)

func withObjects(objs ...*servicesv1alpha1.ServiceEntitlement) fakeOpt {
	return func(c *fakeConfig) {
		for _, o := range objs {
			c.seed = append(c.seed, o)
		}
	}
}

func withListErr(err error) fakeOpt        { return func(c *fakeConfig) { c.listErr = err } }
func withCreateErr(err error) fakeOpt      { return func(c *fakeConfig) { c.createErr = err } }
func withWatch(evs ...watch.Event) fakeOpt { return func(c *fakeConfig) { c.watchEvents = evs } }

// countVerb returns how many actions of the given verb were recorded against
// serviceentitlements.
func countVerb(cs *svcfake.Clientset, verb string) int {
	n := 0
	for _, a := range cs.Actions() {
		if a.GetVerb() == verb && a.GetResource().Resource == entitlementResource {
			n++
		}
	}
	return n
}

// catalogAbsentErr is the real discovery failure returned when the services API
// group is not served, so catalogAbsent (apimeta.IsNoMatchError) recognizes it.
func catalogAbsentErr() error {
	return &apimeta.NoResourceMatchError{
		PartialResource: schema.GroupVersionResource{
			Group: "services.miloapis.com", Version: "v1alpha1", Resource: entitlementResource,
		},
	}
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

// modifiedEvent wraps an entitlement in a MODIFIED watch event.
func modifiedEvent(e *servicesv1alpha1.ServiceEntitlement) watch.Event {
	return watch.Event{Type: watch.Modified, Object: e}
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
