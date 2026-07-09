package serviceactivation

// State is the CLI-facing activation state derived from a project's
// ServiceEntitlement (or its absence). It is a pure reduction of phase, the
// Ready condition, and timestamps — never of the root Service's enablement
// policy — so the first status write answers the regime authoritatively.
type State string

const (
	// StateCatalogUnavailable means the services.miloapis.com API group is not
	// served in this control plane. It is the one unavailability detectable
	// before a create, so the gate never prompts on it.
	StateCatalogUnavailable State = "CatalogUnavailable"

	// StateNotRequested means the catalog is reachable but the project has no
	// entitlement for the service yet.
	StateNotRequested State = "NotRequested"

	// StateProcessing means the entitlement exists but the operator has not
	// written status yet (empty phase / no Ready condition). Normally seconds.
	StateProcessing State = "Processing"

	// StatePendingApproval means the request awaits a human provider decision,
	// with unbounded latency.
	StatePendingApproval State = "PendingApproval"

	// StateActive means the service is usable now.
	StateActive State = "Active"

	// StateDenied means the provider rejected a request that was never active
	// (entitledAt unset). Terminal for this object; recovery is delete + recreate.
	StateDenied State = "Denied"

	// StateRevoked means a previously-active entitlement was rejected
	// (entitledAt set). The server's message is identical to a denial's; the CLI
	// frames it as revocation from the entitledAt derivation.
	StateRevoked State = "Revoked"

	// StateUnavailable means the service is missing or unpublished — reached
	// either from a Rejected entitlement whose reason is ServiceNotPublished, or
	// at create time via the create-error mapping.
	StateUnavailable State = "Unavailable"
)

// Usable reports whether the service can be used right now.
func (s State) Usable() bool { return s == StateActive }

// Terminal reports whether the state will not resolve without user action
// (delete + recreate for Denied/Revoked, or a platform/provider change for
// Unavailable/CatalogUnavailable).
func (s State) Terminal() bool {
	switch s {
	case StateDenied, StateRevoked, StateUnavailable, StateCatalogUnavailable:
		return true
	default:
		return false
	}
}
