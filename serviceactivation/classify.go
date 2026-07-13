package serviceactivation

import (
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	servicesv1alpha1 "go.miloapis.com/service-catalog/api/v1alpha1"
)

const (
	// conditionTypeReady is the single condition type the entitlement controller
	// writes. It is unexported by service-catalog today (the RFC's one upstream
	// ask is to promote it to api/v1alpha1); mirror it here until then.
	conditionTypeReady = "Ready"

	// reasonServiceNotPublished is the only Ready reason that branches control
	// flow: it maps a Rejected entitlement to Unavailable rather than Denied.
	// All other reasons refine wording only.
	reasonServiceNotPublished = "ServiceNotPublished"
)

// Observation is the raw input Classify reduces to a State: the selected
// entitlement (nil when none was found) and whether the catalog API group is
// absent. It exists so classification stays a pure, table-testable function.
type Observation struct {
	// Entitlement is the entitlement selected for this service, or nil if the
	// catalog is reachable but no matching entitlement exists.
	Entitlement *servicesv1alpha1.ServiceEntitlement

	// CatalogAbsent is true when the services.miloapis.com API group is not
	// served — the one unavailability knowable before any create.
	CatalogAbsent bool
}

// Classify reduces an Observation to a CLI State. It reads only phase, the Ready
// condition reason, and entitledAt — never the root Service's policy.
func Classify(obs Observation) State {
	if obs.CatalogAbsent {
		return StateCatalogUnavailable
	}
	e := obs.Entitlement
	if e == nil {
		return StateNotRequested
	}

	ready := apimeta.FindStatusCondition(e.Status.Conditions, conditionTypeReady)
	if e.Status.Phase == "" || ready == nil {
		// The operator has not written status yet.
		return StateProcessing
	}

	switch e.Status.Phase {
	case servicesv1alpha1.EntitlementPhaseActive:
		return StateActive
	case servicesv1alpha1.EntitlementPhasePendingApproval:
		return StatePendingApproval
	case servicesv1alpha1.EntitlementPhaseRejected:
		// ServiceNotPublished is checked before the Denied/Revoked split.
		if ready.Reason == reasonServiceNotPublished {
			return StateUnavailable
		}
		// entitledAt is written once on first activation and never cleared, so it
		// — not the reason — distinguishes a revocation from an initial denial.
		if e.Status.EntitledAt != nil {
			return StateRevoked
		}
		return StateDenied
	default:
		// Unknown phase: treat as still processing rather than inventing a state.
		return StateProcessing
	}
}

// SelectEntitlement picks the entitlement representing the configured service
// from a list, preferring a canonical-name match over the object-name fallback.
// It returns nil when none matches.
func SelectEntitlement(list *servicesv1alpha1.ServiceEntitlementList, cfg Config) *servicesv1alpha1.ServiceEntitlement {
	if list == nil {
		return nil
	}
	var fallback *servicesv1alpha1.ServiceEntitlement
	for i := range list.Items {
		item := &list.Items[i]
		if canonicalNameOf(item) == cfg.CanonicalName {
			return item
		}
		if item.Spec.ServiceRef.Name == cfg.ObjectName && fallback == nil {
			fallback = item
		}
	}
	return fallback
}

// canonicalNameOf returns the best-known canonical service identity for an
// entitlement, preferring the controller-stamped status.serviceName and
// falling back to the spec reference for entitlements the controller hasn't
// reconciled yet (status not written).
func canonicalNameOf(e *servicesv1alpha1.ServiceEntitlement) string {
	if e.Status.ServiceName != "" {
		return e.Status.ServiceName
	}
	return e.Spec.ServiceRef.Name
}

// catalogAbsent reports whether a List error means the services API group is
// not served, as opposed to a transient or permission error.
func catalogAbsent(err error) bool {
	return apimeta.IsNoMatchError(err)
}

// readyCondition returns the entitlement's Ready condition, or nil.
func readyCondition(e *servicesv1alpha1.ServiceEntitlement) *metav1.Condition {
	if e == nil {
		return nil
	}
	return apimeta.FindStatusCondition(e.Status.Conditions, conditionTypeReady)
}
