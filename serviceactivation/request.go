package serviceactivation

import (
	"context"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	servicesv1alpha1 "go.miloapis.com/service-catalog/api/v1alpha1"
)

// Requester backs the explicit `access request` verb. Invoking it is consent, so
// it never prompts (except a --renew confirmation on a TTY) and is safe in
// scripts. Its exit-code contract differs from the gate: a submitted request
// that stands pending exits 0 here, because the request itself succeeded.
type Requester struct {
	Config  Config
	Client  EntitlementClient
	IO      IOStreams
	Project string
}

// RequestOptions parameterizes a request.
type RequestOptions struct {
	// Message populates spec.requestMessage on create.
	Message string
	// Renew deletes a Denied/Revoked entitlement and recreates it.
	Renew bool
	// Wait blocks until the entitlement is Active, Rejected, or Timeout expires.
	Wait bool
	// Timeout bounds --wait; zero means defaultApprovalTimeout.
	Timeout time.Duration
}

// tombstoneDrainBudget bounds how long a renew retries create while the deleted
// entitlement's finalizer teardown drains.
const tombstoneDrainBudget = 30 * time.Second

// Request runs the idempotent request/renew/wait flow.
func (r Requester) Request(ctx context.Context, opts RequestOptions) error {
	if err := r.Config.Validate(); err != nil {
		return err
	}
	if r.Project == "" {
		return reportGeneric(r.IO, nil, "no project set — pass --project or select one with datumctl")
	}

	state, e, err := Observe(ctx, r.Client, r.Config)
	if err != nil {
		return reportGeneric(r.IO, err, "checking %s access: %v", r.Config.noun(), err)
	}

	switch state {
	case StateCatalogUnavailable:
		return reportCatalogUnavailable(r.IO, r.Config)
	case StateUnavailable:
		return reportUnavailable(r.IO, r.Config, e)

	case StateDenied, StateRevoked:
		if !opts.Renew {
			if state == StateRevoked {
				return reportRevoked(r.IO, r.Config, r.Project, e)
			}
			return reportDenied(r.IO, r.Config, r.Project, e)
		}
		if derr := r.renewDelete(ctx, e); derr != nil {
			return derr
		}
		// The rejected object is deleting; fall through to a tombstone-aware create.
		return r.create(ctx, opts, true)

	case StateNotRequested:
		return r.create(ctx, opts, false)

	case StateActive:
		// Already usable; --wait has nothing to wait for.
		r.printStanding(state, e)
		return nil

	case StatePendingApproval, StateProcessing:
		if opts.Wait {
			return r.waitLoop(ctx, opts.Timeout)
		}
		r.printStanding(state, e)
		return nil

	default:
		return reportGeneric(r.IO, nil, "%s access is in an unexpected state (%s)", r.Config.noun(), state)
	}
}

// create submits a new entitlement (retrying past a draining tombstone when
// renewing), then either waits or does the bounded first-status wait.
func (r Requester) create(ctx context.Context, opts RequestOptions, renewing bool) error {
	fmt.Fprintf(r.IO.Err, "Requesting access to %s for project %q...\n", r.Config.noun(), r.Project)

	created, unavailable, cerr := r.submit(ctx, opts.Message, renewing)
	if cerr != nil {
		return reportGeneric(r.IO, cerr, "requesting %s access: %v", r.Config.noun(), cerr)
	}
	if unavailable {
		return reportUnavailable(r.IO, r.Config, nil)
	}

	if opts.Wait {
		return r.waitLoop(ctx, opts.Timeout)
	}

	state, ent, werr := r.waitFirstStatus(ctx, resourceVersionOf(created))
	if werr != nil {
		return reportGeneric(r.IO, werr, "waiting for %s status: %v", r.Config.noun(), werr)
	}
	switch state {
	case StateActive:
		r.printActive()
		return nil
	case StatePendingApproval, StateProcessing:
		// The request stands; the explicit verb exits 0 even while pending.
		r.printSubmitted(state, ent)
		return nil
	case StateDenied:
		return reportDenied(r.IO, r.Config, r.Project, ent)
	case StateRevoked:
		return reportRevoked(r.IO, r.Config, r.Project, ent)
	case StateUnavailable:
		return reportUnavailable(r.IO, r.Config, ent)
	default:
		r.printSubmitted(StateProcessing, ent)
		return nil
	}
}

// submit creates the entitlement. When renewing, it retries past AlreadyExists
// while the previous object's finalizer teardown drains, failing with a clear
// terminal error if the teardown wedges.
func (r Requester) submit(ctx context.Context, message string, renewing bool) (*servicesv1alpha1.ServiceEntitlement, bool, error) {
	if !renewing {
		return createEntitlement(ctx, r.Client, r.Config, message)
	}

	deadline := time.Now().Add(tombstoneDrainBudget)
	for {
		created, unavailable, err := createEntitlement(ctx, r.Client, r.Config, message)
		if err != nil {
			return nil, false, err
		}
		if unavailable {
			return nil, true, nil
		}
		if created != nil {
			return created, false, nil
		}
		// createEntitlement mapped AlreadyExists to (nil, false, nil): the old
		// object is still draining. Retry until the tombstone budget is spent.
		if time.Now().After(deadline) {
			return nil, false, fmt.Errorf("the previous request is still being cleaned up; try again shortly")
		}
		if serr := sleepCtx(ctx, time.Second); serr != nil {
			return nil, false, serr
		}
	}
}

// renewDelete confirms (on a TTY) and deletes a rejected entitlement.
func (r Requester) renewDelete(ctx context.Context, e *servicesv1alpha1.ServiceEntitlement) *Error {
	if r.IO.IsInputTTY() {
		ok, err := r.IO.promptYesNo(fmt.Sprintf("This deletes the rejected %s request and submits a new one. Continue?", r.Config.noun()))
		if err != nil {
			return reportGeneric(r.IO, err, "reading prompt response: %v", err)
		}
		if !ok {
			return reportDeclined(r.IO, r.Config)
		}
	}
	if e != nil {
		if err := r.Client.Delete(ctx, e); err != nil && !apierrors.IsNotFound(err) {
			return reportGeneric(r.IO, err, "removing the rejected %s request: %v", r.Config.noun(), err)
		}
	}
	return nil
}

// waitLoop blocks until the entitlement resolves or the timeout expires, with
// visible progress and up-front "approval is manual" wording.
func (r Requester) waitLoop(ctx context.Context, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = defaultApprovalTimeout
	}
	fmt.Fprintf(r.IO.Err, "Waiting for %s access to become active for project %q.\n", r.Config.noun(), r.Project)
	fmt.Fprintf(r.IO.Err, "Approval is a manual step by the service provider with no time bound; press Ctrl-C to stop waiting.\n")

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var (
		state    State
		ent      *servicesv1alpha1.ServiceEntitlement
		timedOut bool
	)
	err := r.IO.progress(waitCtx, r.IO.IsInputTTY(), "Waiting for provider approval...", func(c context.Context) error {
		var werr error
		state, ent, timedOut, werr = waitUntilResolved(c, r.Client, r.Config.ObjectName)
		return werr
	})
	if err != nil {
		return reportGeneric(r.IO, err, "waiting for %s access: %v", r.Config.noun(), err)
	}
	if timedOut {
		fmt.Fprintf(r.IO.Err, "\nStill awaiting provider approval after %s. Safe to re-run.\n", timeout)
		fmt.Fprintf(r.IO.Err, "%s\n", r.Config.pendingNextSteps())
		return newError(ExitPending, StatePendingApproval, "timed out while pending", nil)
	}
	switch state {
	case StateActive:
		r.printActive()
		return nil
	case StateDenied:
		return reportDenied(r.IO, r.Config, r.Project, ent)
	case StateRevoked:
		return reportRevoked(r.IO, r.Config, r.Project, ent)
	case StateUnavailable:
		return reportUnavailable(r.IO, r.Config, ent)
	default:
		r.printStanding(state, ent)
		return nil
	}
}

// waitFirstStatus runs the bounded first-status wait behind a progress indicator.
func (r Requester) waitFirstStatus(ctx context.Context, resourceVersion string) (State, *servicesv1alpha1.ServiceEntitlement, error) {
	var (
		state State
		ent   *servicesv1alpha1.ServiceEntitlement
	)
	err := r.IO.progress(ctx, r.IO.IsInputTTY(), "Waiting for the platform to process the request...", func(c context.Context) error {
		var e error
		state, ent, e = waitForFirstStatus(c, r.Client, r.Config.ObjectName, resourceVersion)
		return e
	})
	return state, ent, err
}

func (r Requester) printActive() {
	fmt.Fprintf(r.IO.Err, "\n%s is now enabled for project %q.\n", r.Config.DisplayName, r.Project)
}

func (r Requester) printSubmitted(state State, e *servicesv1alpha1.ServiceEntitlement) {
	fmt.Fprintf(r.IO.Err, "\nYour request to enable %s for project %q has been submitted.\n\n", r.Config.noun(), r.Project)
	if state == StatePendingApproval {
		fmt.Fprintf(r.IO.Err, "  Status:  Pending approval — %s\n", serverMessage(StatePendingApproval, e))
		fmt.Fprintf(r.IO.Err, "           Approval is a manual step by the service provider and may take a while.\n\n")
	} else {
		fmt.Fprintf(r.IO.Err, "  Status:  Processing — the platform hasn't recorded a decision yet.\n\n")
	}
	fmt.Fprintf(r.IO.Err, "%s\n", r.Config.pendingNextSteps())
}

func (r Requester) printStanding(state State, e *servicesv1alpha1.ServiceEntitlement) {
	RenderStatus(r.IO.Err, r.Config, r.Project, state, e)
}

// sleepCtx sleeps for d unless ctx is cancelled first.
func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
