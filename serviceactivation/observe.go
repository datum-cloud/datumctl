package serviceactivation

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	servicesv1alpha1 "go.miloapis.com/service-catalog/api/v1alpha1"
)

// Observe lists entitlements, selects the one for the configured service, and
// classifies it. A CatalogUnavailable result is returned as a State, not an
// error; only genuine transport/API failures come back as err.
func Observe(ctx context.Context, ec EntitlementClient, cfg Config) (State, *servicesv1alpha1.ServiceEntitlement, error) {
	list, err := ec.List(ctx)
	if err != nil {
		if catalogAbsent(err) {
			return StateCatalogUnavailable, nil, nil
		}
		return "", nil, err
	}
	e := SelectEntitlement(list, cfg)
	return Classify(Observation{Entitlement: e}), e, nil
}

// getAndClassify re-reads a single entitlement by name and classifies it. Used
// as the re-GET fallback after a watch times out or errors. A NotFound resolves
// to NotRequested; a catalog-absent error to CatalogUnavailable.
func getAndClassify(ctx context.Context, ec EntitlementClient, name string) (State, *servicesv1alpha1.ServiceEntitlement, error) {
	e, err := ec.Get(ctx, name)
	if err != nil {
		switch {
		case apierrors.IsNotFound(err):
			return StateNotRequested, nil, nil
		case catalogAbsent(err):
			return StateCatalogUnavailable, nil, nil
		default:
			return "", nil, err
		}
	}
	return Classify(Observation{Entitlement: e}), e, nil
}
