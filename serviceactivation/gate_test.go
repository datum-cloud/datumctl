package serviceactivation

import (
	"context"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	servicesv1alpha1 "go.miloapis.com/service-catalog/api/v1alpha1"
)

func gateWith(t *testing.T, ec EntitlementClient, io IOStreams) Gate {
	t.Helper()
	return Gate{Config: testConfig, Client: ec, IO: io, Project: "datum-cloud"}
}

// wantExit asserts err is an *Error with the expected code and state.
func wantExit(t *testing.T, err error, code int, state State) {
	t.Helper()
	se, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}
	if se.Code != code {
		t.Fatalf("exit code = %d, want %d (message: %q)", se.Code, code, se.Message)
	}
	if state != "" && se.State != state {
		t.Fatalf("state = %q, want %q", se.State, state)
	}
}

func TestGateActiveProceeds(t *testing.T) {
	active := entitlement("compute", servicesv1alpha1.EntitlementPhaseActive, reasonEntitlementActive, "This service is enabled and ready to use.", ptrNow())
	ec, _ := newFake(withObjects(active))
	io, _, errb := testIO("", false)

	if err := gateWith(t, ec, io).Run(context.Background()); err != nil {
		t.Fatalf("Run() = %v, want nil (command should proceed)", err)
	}
	if errb.Len() != 0 {
		t.Fatalf("expected no stderr for an active service, got %q", errb.String())
	}
}

func TestGateNonInteractiveNeverMutates(t *testing.T) {
	ec, cs := newFake() // empty list → NotRequested
	io, _, errb := testIO("", false)

	err := gateWith(t, ec, io).Run(context.Background())
	wantExit(t, err, ExitNotEnabled, StateNotRequested)
	if got := countVerb(cs, "create"); got != 0 {
		t.Fatalf("non-interactive gate created %d entitlements, want 0", got)
	}
	if !strings.Contains(errb.String(), "Request access with: datumctl compute access request") {
		t.Fatalf("missing opt-in command in output:\n%s", errb.String())
	}
}

func TestGateInteractiveSubmitLandsPending(t *testing.T) {
	pending := entitlement("compute", servicesv1alpha1.EntitlementPhasePendingApproval, reasonEntitlementPendingApproval, "Waiting for the service provider to approve this request.", nil)
	ec, cs := newFake(withWatch(modifiedEvent(pending)))
	io, _, errb := testIO("y\n", true)

	err := gateWith(t, ec, io).Run(context.Background())
	wantExit(t, err, ExitPending, StatePendingApproval)
	if got := countVerb(cs, "create"); got != 1 {
		t.Fatalf("expected exactly 1 create, got %d", got)
	}
	out := errb.String()
	if !strings.Contains(out, "has been submitted") {
		t.Fatalf("missing submitted copy:\n%s", out)
	}
	if strings.Contains(out, "try again in a moment") {
		t.Fatalf("the deleted 'try again in a moment' string is back:\n%s", out)
	}
	if strings.Contains(out, "Error:") {
		t.Fatalf("printed Error: after a successful submit:\n%s", out)
	}
	if !strings.Contains(out, "datumctl compute access") {
		t.Fatalf("missing plugin-local next-step verb:\n%s", out)
	}
}

func TestGateInteractiveDecline(t *testing.T) {
	ec, cs := newFake()
	io, _, errb := testIO("n\n", true)

	err := gateWith(t, ec, io).Run(context.Background())
	wantExit(t, err, ExitDeclined, StateNotRequested)
	if got := countVerb(cs, "create"); got != 0 {
		t.Fatalf("declining still created an entitlement")
	}
	if !strings.Contains(errb.String(), "No request was made") {
		t.Fatalf("missing decline copy:\n%s", errb.String())
	}
}

func TestGateCreateAlreadyExistsFallsThrough(t *testing.T) {
	pending := entitlement("compute", servicesv1alpha1.EntitlementPhasePendingApproval, reasonEntitlementPendingApproval, "Waiting for the service provider to approve this request.", nil)
	ec, _ := newFake(withCreateErr(alreadyExistsErr()), withWatch(modifiedEvent(pending)))
	io, _, errb := testIO("y\n", true)

	err := gateWith(t, ec, io).Run(context.Background())
	// AlreadyExists is a won race, not a failure: fall through to the wait.
	wantExit(t, err, ExitPending, StatePendingApproval)
	if !strings.Contains(errb.String(), "has been submitted") {
		t.Fatalf("AlreadyExists did not fall through to the submitted/pending path:\n%s", errb.String())
	}
}

func TestGateCreateAdmissionRejectionMapsToUnavailable(t *testing.T) {
	ec, _ := newFake(withCreateErr(invalidCreateErr()))
	io, _, errb := testIO("y\n", true)

	err := gateWith(t, ec, io).Run(context.Background())
	wantExit(t, err, ExitUnavailable, StateUnavailable)
	if !strings.Contains(errb.String(), "not available on this platform environment") {
		t.Fatalf("missing unavailable copy:\n%s", errb.String())
	}
}

func TestGatePendingReentryIsInstant(t *testing.T) {
	pending := withCreationAge(
		entitlement("compute", servicesv1alpha1.EntitlementPhasePendingApproval, reasonEntitlementPendingApproval, "Waiting for the service provider to approve this request.", nil),
		metav1.NewTime(time.Now().Add(-2*time.Hour)),
	)
	ec, cs := newFake(withObjects(pending))
	io, _, errb := testIO("", false)

	err := gateWith(t, ec, io).Run(context.Background())
	wantExit(t, err, ExitPending, StatePendingApproval)
	if got := countVerb(cs, "create"); got != 0 {
		t.Fatalf("pending re-entry must not create")
	}
	if !strings.Contains(errb.String(), "awaiting provider approval") {
		t.Fatalf("missing pending re-entry copy:\n%s", errb.String())
	}
}

func TestGateDeniedReentry(t *testing.T) {
	denied := entitlement("compute", servicesv1alpha1.EntitlementPhaseRejected, reasonEntitlementRejected, "The service provider denied this request.", nil)
	ec, _ := newFake(withObjects(denied))
	io, _, errb := testIO("", false)

	err := gateWith(t, ec, io).Run(context.Background())
	wantExit(t, err, ExitDeniedOrRevoked, StateDenied)
	if !strings.Contains(errb.String(), "--renew") {
		t.Fatalf("missing recovery command:\n%s", errb.String())
	}
}

func TestGateCatalogUnavailable(t *testing.T) {
	ec, cs := newFake(withListErr(catalogAbsentErr()))
	io, _, errb := testIO("", false)

	err := gateWith(t, ec, io).Run(context.Background())
	wantExit(t, err, ExitUnavailable, StateCatalogUnavailable)
	if got := countVerb(cs, "create"); got != 0 {
		t.Fatalf("catalog-absent must never create")
	}
	if !strings.Contains(errb.String(), "not available on this platform environment") {
		t.Fatalf("missing unavailable copy:\n%s", errb.String())
	}
}

func TestGateEmptyProjectIsNoop(t *testing.T) {
	ec, _ := newFake(withListErr(catalogAbsentErr()))
	io, _, _ := testIO("", false)
	g := Gate{Config: testConfig, Client: ec, IO: io, Project: ""}
	if err := g.Run(context.Background()); err != nil {
		t.Fatalf("empty project should be a no-op, got %v", err)
	}
}

func ptrNow() *metav1.Time {
	n := metav1.Now()
	return &n
}
