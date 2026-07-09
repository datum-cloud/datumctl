package serviceactivation

import (
	"fmt"
	"io"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	servicesv1alpha1 "go.miloapis.com/service-catalog/api/v1alpha1"
)

// StatusReport is the machine-readable form emitted by the access status verb
// under -o json|yaml. It pairs the derived state enum with the raw entitlement
// so automation branches without scraping prose.
type StatusReport struct {
	Service     string                               `json:"service"`
	Project     string                               `json:"project"`
	State       State                                `json:"state"`
	Entitlement *servicesv1alpha1.ServiceEntitlement `json:"entitlement,omitempty"`
}

// NewStatusReport assembles a StatusReport from a classification result.
func NewStatusReport(cfg Config, project string, state State, e *servicesv1alpha1.ServiceEntitlement) StatusReport {
	return StatusReport{
		Service:     cfg.CanonicalName,
		Project:     project,
		State:       state,
		Entitlement: e,
	}
}

// RenderStatus writes the human-readable status block for the access verb to w.
// It never returns an error and never exits; state is data, not a failure.
func RenderStatus(w io.Writer, cfg Config, project string, state State, e *servicesv1alpha1.ServiceEntitlement) {
	fmt.Fprintf(w, "Service:  %s (%s)\n", cfg.DisplayName, cfg.CanonicalName)
	fmt.Fprintf(w, "Project:  %s\n", project)
	fmt.Fprintf(w, "Status:   %s\n", statusLine(cfg, state, e))
	if msg := serverMessage(state, e); msg != "" {
		fmt.Fprintf(w, "          %s\n", msg)
	}
	if hint := nextStepHint(cfg, state); hint != "" {
		fmt.Fprintf(w, "\n%s\n", hint)
	}
}

// statusLine is the one-line derived-state summary, with an age suffix where a
// timestamp gives one meaning.
func statusLine(cfg Config, state State, e *servicesv1alpha1.ServiceEntitlement) string {
	switch state {
	case StateActive:
		if e != nil && e.Status.EntitledAt != nil {
			return fmt.Sprintf("Active (enabled %s)", ageAgo(*e.Status.EntitledAt))
		}
		return "Active"
	case StatePendingApproval:
		return fmt.Sprintf("Pending approval (requested %s)", requestedAge(e))
	case StateProcessing:
		return fmt.Sprintf("Processing (requested %s)", requestedAge(e))
	case StateDenied:
		return "Denied"
	case StateRevoked:
		return "Revoked"
	case StateUnavailable:
		return "Unavailable"
	case StateCatalogUnavailable:
		return "Unavailable"
	case StateNotRequested:
		return "Not requested"
	default:
		return string(state)
	}
}

// nextStepHint is the single copy-pasteable command suggested for a state in the
// status view, or "" when none applies. It references only the plugin-local verb.
func nextStepHint(cfg Config, state State) string {
	switch state {
	case StateNotRequested:
		return "Request access with: " + cfg.requestCommand()
	case StatePendingApproval, StateProcessing:
		return "Wait for activation: " + cfg.requestCommand() + " --wait"
	case StateDenied, StateRevoked:
		return "Request access again with: " + cfg.requestCommand() + " --renew"
	default:
		return ""
	}
}

// serverMessage returns the platform's Ready condition Message verbatim, or a
// state default when the condition carries none. The server explains "why"; the
// CLI must not paraphrase it.
func serverMessage(state State, e *servicesv1alpha1.ServiceEntitlement) string {
	if c := readyCondition(e); c != nil && c.Message != "" {
		return c.Message
	}
	switch state {
	case StateProcessing:
		return "The request is being processed."
	case StateNotRequested:
		return "This service is not enabled for this project."
	default:
		return ""
	}
}

// requestedAge returns the age of the entitlement's creation, or "just now".
func requestedAge(e *servicesv1alpha1.ServiceEntitlement) string {
	if e == nil {
		return "just now"
	}
	return ageAgo(e.CreationTimestamp)
}

// ageAgo formats a timestamp as a compact relative age with an "ago" suffix.
func ageAgo(t metav1.Time) string {
	if t.IsZero() {
		return "just now"
	}
	d := time.Since(t.Time)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
