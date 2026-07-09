package serviceactivation

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	servicesv1alpha1 "go.miloapis.com/service-catalog/api/v1alpha1"
)

// createEntitlement submits a new entitlement for the configured service and
// applies the create-error mapping:
//
//   - success       → the created object (its resourceVersion seeds the watch)
//   - AlreadyExists  → (nil, false, nil): a concurrent create won; the caller
//     falls through to the bounded wait and reclassifies
//   - Invalid        → (nil, true, nil): the admission webhook rejected a
//     missing or unpublished service; the caller reports Unavailable (exit 13)
//   - anything else  → (nil, false, err): generic failure (exit 1)
//
// The entitlement is created with the object name in both metadata.name and
// spec.serviceRef.name — admission rejects the canonical name in the spec.
func createEntitlement(ctx context.Context, ec EntitlementClient, cfg Config, message string) (created *servicesv1alpha1.ServiceEntitlement, unavailable bool, err error) {
	e := &servicesv1alpha1.ServiceEntitlement{
		ObjectMeta: metav1.ObjectMeta{Name: cfg.ObjectName},
		Spec: servicesv1alpha1.ServiceEntitlementSpec{
			ServiceRef:     servicesv1alpha1.ServiceRef{Name: cfg.ObjectName},
			RequestMessage: message,
		},
	}
	switch cerr := ec.Create(ctx, e); {
	case cerr == nil:
		return e, false, nil
	case apierrors.IsAlreadyExists(cerr):
		return nil, false, nil
	case apierrors.IsInvalid(cerr):
		return nil, true, nil
	default:
		return nil, false, cerr
	}
}

// resourceVersionOf returns an object's resourceVersion, or "" when the object
// is nil (an AlreadyExists race), in which case the watch starts from now and
// the re-read fallback covers any status written in between.
func resourceVersionOf(e *servicesv1alpha1.ServiceEntitlement) string {
	if e == nil {
		return ""
	}
	return e.ResourceVersion
}
