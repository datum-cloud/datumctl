package serviceactivation

import (
	"context"
	"fmt"
	"time"

	servicesv1alpha1 "go.miloapis.com/service-catalog/api/v1alpha1"
)

// Gate is the preflight run before a gated command. On a TTY it may prompt to
// request access, submit, and wait for the platform's first answer; without one
// it never prompts and never mutates. Run returns nil to let the command
// proceed (the service is usable), or an *Error carrying the exit code.
type Gate struct {
	Config  Config
	Client  EntitlementClient
	IO      IOStreams
	Project string
}

// Run executes the preflight. An empty project is a no-op: there is nothing to
// gate on, and the command's own "no project" error is clearer.
func (g Gate) Run(ctx context.Context) error {
	if g.Project == "" {
		return nil
	}
	if err := g.Config.Validate(); err != nil {
		return err
	}

	state, e, err := Observe(ctx, g.Client, g.Config)
	if err != nil {
		return reportGeneric(g.IO, err, "checking %s access: %v", g.Config.noun(), err)
	}
	return g.handle(ctx, state, e)
}

func (g Gate) handle(ctx context.Context, state State, e *servicesv1alpha1.ServiceEntitlement) error {
	switch state {
	case StateActive:
		return nil
	case StateNotRequested:
		return g.handleNotRequested(ctx)
	case StateProcessing:
		return g.handleProcessing(ctx, e)
	case StatePendingApproval:
		return reportPendingReentry(g.IO, g.Config, g.Project, e)
	case StateDenied:
		return reportDenied(g.IO, g.Config, g.Project, e)
	case StateRevoked:
		return reportRevoked(g.IO, g.Config, g.Project, e)
	case StateUnavailable:
		return reportUnavailable(g.IO, g.Config, e)
	case StateCatalogUnavailable:
		return reportCatalogUnavailable(g.IO, g.Config)
	default:
		return reportGeneric(g.IO, nil, "%s access is in an unexpected state (%s)", g.Config.noun(), state)
	}
}

// handleNotRequested prompts and submits on a TTY; otherwise reports without
// mutating.
func (g Gate) handleNotRequested(ctx context.Context) error {
	if !g.IO.IsInputTTY() {
		return reportNotEnabledNonInteractive(g.IO, g.Config, g.Project)
	}

	fmt.Fprintf(g.IO.Err, "%s is not enabled for project %q.\n", g.Config.DisplayName, g.Project)
	fmt.Fprintf(g.IO.Err, "Requesting access sends an enablement request to the service provider for approval.\n")
	ok, err := g.IO.promptYesNo("Would you like to request access?")
	if err != nil {
		return reportGeneric(g.IO, err, "reading prompt response: %v", err)
	}
	if !ok {
		return reportDeclined(g.IO, g.Config)
	}
	return g.submitAndWait(ctx, "")
}

// handleProcessing re-enters the bounded wait for a young entitlement, then
// reclassifies. An older one (a wedged operator) reports immediately so it does
// not tax every invocation.
func (g Gate) handleProcessing(ctx context.Context, e *servicesv1alpha1.ServiceEntitlement) error {
	if e != nil && time.Since(e.CreationTimestamp.Time) > reentryWaitWindow {
		return reportProcessingReentry(g.IO, g.Config, g.Project, e)
	}

	state, ent, err := g.waitFirstStatus(ctx, resourceVersionOf(e))
	if err != nil {
		return reportGeneric(g.IO, err, "waiting for %s status: %v", g.Config.noun(), err)
	}
	switch state {
	case StateActive:
		return nil
	case StatePendingApproval:
		return reportPendingReentry(g.IO, g.Config, g.Project, ent)
	case StateDenied:
		return reportDenied(g.IO, g.Config, g.Project, ent)
	case StateRevoked:
		return reportRevoked(g.IO, g.Config, g.Project, ent)
	case StateUnavailable:
		return reportUnavailable(g.IO, g.Config, ent)
	default:
		return reportProcessingReentry(g.IO, g.Config, g.Project, ent)
	}
}

// submitAndWait creates the entitlement, waits for the platform's first answer
// with visible progress, and branches on the result. On Active it prints the
// enabled line and returns nil so the original command runs.
func (g Gate) submitAndWait(ctx context.Context, message string) error {
	fmt.Fprintf(g.IO.Err, "Requesting access to %s for project %q...\n", g.Config.noun(), g.Project)

	created, unavailable, cerr := createEntitlement(ctx, g.Client, g.Config, message)
	if cerr != nil {
		return reportGeneric(g.IO, cerr, "requesting %s access: %v", g.Config.noun(), cerr)
	}
	if unavailable {
		return reportUnavailable(g.IO, g.Config, nil)
	}

	state, ent, werr := g.waitFirstStatus(ctx, resourceVersionOf(created))
	if werr != nil {
		return reportGeneric(g.IO, werr, "waiting for %s status: %v", g.Config.noun(), werr)
	}
	switch state {
	case StateActive:
		fmt.Fprintf(g.IO.Err, "\n%s is now enabled for project %q.\n", g.Config.DisplayName, g.Project)
		return nil
	case StatePendingApproval:
		return reportSubmittedPending(g.IO, g.Config, g.Project, ent)
	case StateDenied:
		return reportDenied(g.IO, g.Config, g.Project, ent)
	case StateRevoked:
		return reportRevoked(g.IO, g.Config, g.Project, ent)
	case StateUnavailable:
		return reportUnavailable(g.IO, g.Config, ent)
	default:
		return reportSubmittedProcessing(g.IO, g.Config, g.Project)
	}
}

// waitFirstStatus runs the bounded first-status wait behind a progress
// indicator on the gate's IO streams.
func (g Gate) waitFirstStatus(ctx context.Context, resourceVersion string) (State, *servicesv1alpha1.ServiceEntitlement, error) {
	var (
		state State
		ent   *servicesv1alpha1.ServiceEntitlement
	)
	err := g.IO.progress(ctx, g.IO.IsInputTTY(), "Waiting for the platform to process the request...", func(c context.Context) error {
		var e error
		state, ent, e = waitForFirstStatus(c, g.Client, g.Config.ObjectName, resourceVersion)
		return e
	})
	return state, ent, err
}
