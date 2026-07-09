package serviceactivation

import (
	"fmt"

	servicesv1alpha1 "go.miloapis.com/service-catalog/api/v1alpha1"
)

// This file holds the user-facing copy for the gate and request verbs. Every
// string routes to stderr (the caller's IOStreams.Err); stdout stays a clean
// data channel. The server explains "why" (the Ready Message, verbatim) and the
// CLI explains "what next" (exactly one plugin-local command) — never a latency
// or notification promise the platform does not keep.

// pendingNextSteps is the two-line footer offering the status and wait verbs.
func (c Config) pendingNextSteps() string {
	return fmt.Sprintf("Check progress with:   %s\nWait for activation:   %s --wait",
		c.AccessCommand, c.requestCommand())
}

// reportSubmittedPending prints the clean "submitted" copy for a fresh request
// that landed in PendingApproval. There is deliberately no "Error:" prefix — the
// one thing the user consented to succeeded — but the gated command did not run,
// so the exit code is still ExitPending.
func reportSubmittedPending(io IOStreams, cfg Config, project string, e *servicesv1alpha1.ServiceEntitlement) *Error {
	fmt.Fprintf(io.Err, "\nYour request to enable %s for project %q has been submitted.\n\n", cfg.noun(), project)
	fmt.Fprintf(io.Err, "  Status:  Pending approval — %s\n", serverMessage(StatePendingApproval, e))
	fmt.Fprintf(io.Err, "           Approval is a manual step by the service provider and may take a while.\n\n")
	fmt.Fprintf(io.Err, "%s\n", cfg.pendingNextSteps())
	return newError(ExitPending, StatePendingApproval, "request submitted; awaiting provider approval", nil)
}

// reportSubmittedProcessing prints the "submitted" copy when the platform has
// not recorded a decision within the bounded wait. Same shape as pending, with a
// degraded detail line; exit ExitPending.
func reportSubmittedProcessing(io IOStreams, cfg Config, project string) *Error {
	fmt.Fprintf(io.Err, "\nYour request to enable %s for project %q has been submitted.\n\n", cfg.noun(), project)
	fmt.Fprintf(io.Err, "  Status:  Processing — the platform hasn't recorded a decision yet.\n")
	fmt.Fprintf(io.Err, "           This normally takes only a few seconds.\n\n")
	fmt.Fprintf(io.Err, "%s\n", cfg.pendingNextSteps())
	return newError(ExitPending, StateProcessing, "request submitted; awaiting first status", nil)
}

// reportPendingReentry prints the awaiting-approval copy when a gated command is
// re-invoked while the request is already pending. Instant: no watch, no wait.
func reportPendingReentry(io IOStreams, cfg Config, project string, e *servicesv1alpha1.ServiceEntitlement) *Error {
	fmt.Fprintf(io.Err, "Error: %s access for project %q is awaiting provider approval (requested %s).\n",
		cfg.noun(), project, requestedAge(e))
	fmt.Fprintf(io.Err, "  %s\n\n", serverMessage(StatePendingApproval, e))
	fmt.Fprintf(io.Err, "%s\n", cfg.pendingNextSteps())
	return newError(ExitPending, StatePendingApproval, "awaiting provider approval", nil)
}

// reportProcessingReentry prints the still-processing copy for an older
// entitlement whose status the operator has not written. A wedged operator must
// not tax every invocation with the bounded wait, so this reports immediately.
func reportProcessingReentry(io IOStreams, cfg Config, project string, e *servicesv1alpha1.ServiceEntitlement) *Error {
	fmt.Fprintf(io.Err, "Error: %s access for project %q is still being processed (requested %s).\n",
		cfg.noun(), project, requestedAge(e))
	fmt.Fprintf(io.Err, "  This normally takes only a few seconds — if it persists, contact support.\n\n")
	fmt.Fprintf(io.Err, "Check progress with:   %s\n", cfg.AccessCommand)
	return newError(ExitPending, StateProcessing, "still being processed", nil)
}

// reportDenied prints the provider's denial reason plus the renew recovery path.
func reportDenied(io IOStreams, cfg Config, project string, e *servicesv1alpha1.ServiceEntitlement) *Error {
	fmt.Fprintf(io.Err, "Error: %s access for project %q was denied.\n", cfg.noun(), project)
	if msg := serverMessage(StateDenied, e); msg != "" {
		fmt.Fprintf(io.Err, "  %s\n", msg)
	}
	fmt.Fprintf(io.Err, "\nRequest access again with: %s --renew\n", cfg.requestCommand())
	return newError(ExitDeniedOrRevoked, StateDenied, "access denied", nil)
}

// reportRevoked prints the same shape as a denial, framed as a revocation.
func reportRevoked(io IOStreams, cfg Config, project string, e *servicesv1alpha1.ServiceEntitlement) *Error {
	fmt.Fprintf(io.Err, "Error: %s access for project %q was revoked.\n", cfg.noun(), project)
	if msg := serverMessage(StateRevoked, e); msg != "" {
		fmt.Fprintf(io.Err, "  %s\n", msg)
	}
	fmt.Fprintf(io.Err, "\nRequest access again with: %s --renew\n", cfg.requestCommand())
	return newError(ExitDeniedOrRevoked, StateRevoked, "access revoked", nil)
}

// reportUnavailable prints the not-available-on-this-platform copy. It covers
// both a Rejected/ServiceNotPublished entitlement and the create-time mapping.
func reportUnavailable(io IOStreams, cfg Config, e *servicesv1alpha1.ServiceEntitlement) *Error {
	fmt.Fprintf(io.Err, "Error: the %s service is not available on this platform environment.\n", cfg.noun())
	if msg := serverMessage(StateUnavailable, e); msg != "" {
		fmt.Fprintf(io.Err, "  %s\n", msg)
	}
	if cfg.SupportURL != "" {
		fmt.Fprintf(io.Err, "\nFor help, see: %s\n", cfg.SupportURL)
	}
	return newError(ExitUnavailable, StateUnavailable, "service unavailable on this platform", nil)
}

// reportCatalogUnavailable prints the same copy when the enablement API group is
// absent entirely. Distinguished only by the reported State.
func reportCatalogUnavailable(io IOStreams, cfg Config) *Error {
	fmt.Fprintf(io.Err, "Error: the %s service is not available on this platform environment.\n", cfg.noun())
	if cfg.SupportURL != "" {
		fmt.Fprintf(io.Err, "\nFor help, see: %s\n", cfg.SupportURL)
	}
	return newError(ExitUnavailable, StateCatalogUnavailable, "enablement API not served", nil)
}

// reportNotEnabledNonInteractive prints the deterministic, no-mutation copy for
// the non-interactive gate path.
func reportNotEnabledNonInteractive(io IOStreams, cfg Config, project string) *Error {
	fmt.Fprintf(io.Err, "Error: %s is not enabled for project %q.\n\n", cfg.noun(), project)
	fmt.Fprintf(io.Err, "Request access with: %s\n", cfg.requestCommand())
	return newError(ExitNotEnabled, StateNotRequested, "service not enabled", nil)
}

// reportDeclined prints the copy for a user who declined the interactive prompt.
// Not an error the user needs "Error:" for — they chose it — but the gated
// command did not run, so exit ExitDeclined.
func reportDeclined(io IOStreams, cfg Config) *Error {
	fmt.Fprintf(io.Err, "\nNo request was made.\n\n")
	fmt.Fprintf(io.Err, "Request access later with: %s\n", cfg.requestCommand())
	return newError(ExitDeclined, StateNotRequested, "request declined", nil)
}

// reportGeneric prints a generic failure to stderr and returns an ExitError. The
// SDK owns all of its user-facing output, so it prints here rather than relying
// on the caller to surface *Error messages.
func reportGeneric(io IOStreams, cause error, format string, args ...any) *Error {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(io.Err, "Error: %s\n", msg)
	return newError(ExitError, "", msg, cause)
}
