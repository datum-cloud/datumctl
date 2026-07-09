package serviceactivation

import (
	"context"
	"errors"
	"time"

	"k8s.io/apimachinery/pkg/watch"

	servicesv1alpha1 "go.miloapis.com/service-catalog/api/v1alpha1"
)

const (
	// firstStatusTimeout bounds the post-create wait for the platform's first
	// status write. Timing out is not an error — it resolves to Processing.
	firstStatusTimeout = 30 * time.Second

	// reentryWaitWindow is how young a Processing entitlement may be for the gate
	// to re-enter the bounded wait rather than reporting immediately. A wedged
	// operator must not become a per-invocation tax.
	reentryWaitWindow = 2 * time.Minute

	// defaultApprovalTimeout bounds `--wait` by default. Approval is a manual
	// provider step with no time bound, so this is a safety ceiling, not an SLA.
	defaultApprovalTimeout = 30 * time.Minute

	// approvalRereadInterval re-reads the object periodically during `--wait` to
	// cover a deployed operator that predates the pending-requeue hardening.
	approvalRereadInterval = 60 * time.Second
)

// waitForFirstStatus watches a single entitlement for its first status write,
// bounded by firstStatusTimeout. It returns the first non-Processing state, or —
// on watch timeout, closure, or error — the result of a single re-read. A
// timeout therefore surfaces as Processing, not an error.
func waitForFirstStatus(ctx context.Context, ec EntitlementClient, name, resourceVersion string) (State, *servicesv1alpha1.ServiceEntitlement, error) {
	watchCtx, cancel := context.WithTimeout(ctx, firstStatusTimeout)
	defer cancel()

	w, err := ec.Watch(watchCtx, name, resourceVersion)
	if err != nil {
		return getAndClassify(ctx, ec, name)
	}
	defer w.Stop()

	for {
		select {
		case <-watchCtx.Done():
			return getAndClassify(ctx, ec, name)
		case ev, ok := <-w.ResultChan():
			if !ok {
				return getAndClassify(ctx, ec, name)
			}
			e, matched := entitlementFromEvent(ev, name)
			if !matched {
				continue
			}
			if st := Classify(Observation{Entitlement: e}); st != StateProcessing {
				return st, e, nil
			}
		}
	}
}

// waitUntilResolved blocks until the entitlement reaches a resolved state
// (Active, Denied, Revoked, or Unavailable) or ctx expires. It prefers a watch
// but re-reads on approvalRereadInterval so a stale-pending operator still
// converges. On ctx deadline it returns timedOut=true with the last-known state;
// on cancellation it returns ctx.Err().
func waitUntilResolved(ctx context.Context, ec EntitlementClient, name string) (state State, e *servicesv1alpha1.ServiceEntitlement, timedOut bool, err error) {
	state, e, err = getAndClassify(ctx, ec, name)
	if err != nil {
		return state, e, false, err
	}
	if isResolved(state) {
		return state, e, false, nil
	}

	ticker := time.NewTicker(approvalRereadInterval)
	defer ticker.Stop()

	for {
		w, werr := ec.Watch(ctx, name, "")
		var events <-chan watch.Event
		if werr == nil {
			events = w.ResultChan()
		}

		reestablish := false
		for !reestablish {
			select {
			case <-ctx.Done():
				if w != nil {
					w.Stop()
				}
				if errors.Is(ctx.Err(), context.DeadlineExceeded) {
					return state, e, true, nil
				}
				return state, e, false, ctx.Err()

			case <-ticker.C:
				st, ent, gerr := getAndClassify(ctx, ec, name)
				if gerr == nil {
					state, e = st, ent
					if isResolved(state) {
						if w != nil {
							w.Stop()
						}
						return state, e, false, nil
					}
				}

			case ev, ok := <-events:
				if !ok {
					// Watch closed (or was never established); re-establish after
					// the next tick to avoid a hot loop when watch is unsupported.
					if w != nil {
						w.Stop()
					}
					reestablish = true
					continue
				}
				ent, matched := entitlementFromEvent(ev, name)
				if !matched {
					continue
				}
				state, e = Classify(Observation{Entitlement: ent}), ent
				if isResolved(state) {
					w.Stop()
					return state, e, false, nil
				}
			}
		}

		if werr != nil {
			// No watch available: wait out one re-read interval before retrying so
			// the loop degrades to polling rather than spinning.
			select {
			case <-ctx.Done():
				if errors.Is(ctx.Err(), context.DeadlineExceeded) {
					return state, e, true, nil
				}
				return state, e, false, ctx.Err()
			case <-ticker.C:
				st, ent, gerr := getAndClassify(ctx, ec, name)
				if gerr == nil {
					state, e = st, ent
					if isResolved(state) {
						return state, e, false, nil
					}
				}
			}
		}
	}
}

// isResolved reports whether a state will not change without user or provider
// action — the terminal outcomes plus Active.
func isResolved(s State) bool {
	return s == StateActive || s.Terminal()
}

// entitlementFromEvent extracts a matching entitlement from a watch event,
// ignoring deletes, bookmarks, and events for other objects.
func entitlementFromEvent(ev watch.Event, name string) (*servicesv1alpha1.ServiceEntitlement, bool) {
	if ev.Type != watch.Added && ev.Type != watch.Modified {
		return nil, false
	}
	e, ok := ev.Object.(*servicesv1alpha1.ServiceEntitlement)
	if !ok || e.Name != name {
		return nil, false
	}
	return e, true
}
