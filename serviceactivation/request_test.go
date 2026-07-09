package serviceactivation

import (
	"context"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/watch"

	servicesv1alpha1 "go.miloapis.com/service-catalog/api/v1alpha1"
)

func requesterWith(fc *fakeClient, io IOStreams) Requester {
	return Requester{Config: testConfig, Client: fc, IO: io, Project: "datum-cloud"}
}

func TestRequestSubmitPendingExitsZero(t *testing.T) {
	pending := entitlement("compute", servicesv1alpha1.EntitlementPhasePendingApproval, reasonEntitlementPendingApproval, "Waiting for the service provider to approve this request.", nil)
	fc := &fakeClient{watchEmit: []watch.Event{modifiedEvent(pending)}}
	io, _, errb := testIO("", false)

	// A submitted-but-pending request via the explicit verb is success (exit 0).
	if err := requesterWith(fc, io).Request(context.Background(), RequestOptions{}); err != nil {
		t.Fatalf("Request() = %v, want nil (exit 0)", err)
	}
	if len(fc.creates) != 1 {
		t.Fatalf("expected 1 create, got %d", len(fc.creates))
	}
	if !strings.Contains(errb.String(), "has been submitted") {
		t.Fatalf("missing submitted copy:\n%s", errb.String())
	}
}

func TestRequestDeniedWithoutRenewIsTerminal(t *testing.T) {
	denied := entitlement("compute", servicesv1alpha1.EntitlementPhaseRejected, reasonEntitlementRejected, "The service provider denied this request.", nil)
	fc := &fakeClient{listResult: listOf(denied)}
	io, _, _ := testIO("", false)

	err := requesterWith(fc, io).Request(context.Background(), RequestOptions{})
	wantExit(t, err, ExitDeniedOrRevoked, StateDenied)
	if len(fc.deletes) != 0 {
		t.Fatalf("denied without --renew must not delete")
	}
}

func TestRequestRenewDeletesAndRecreates(t *testing.T) {
	denied := entitlement("compute", servicesv1alpha1.EntitlementPhaseRejected, reasonEntitlementRejected, "The service provider denied this request.", nil)
	pending := entitlement("compute", servicesv1alpha1.EntitlementPhasePendingApproval, reasonEntitlementPendingApproval, "Waiting for the service provider to approve this request.", nil)
	fc := &fakeClient{
		listResult: listOf(denied),
		watchEmit:  []watch.Event{modifiedEvent(pending)},
	}
	io, _, _ := testIO("", false) // non-interactive: no renew confirmation prompt

	if err := requesterWith(fc, io).Request(context.Background(), RequestOptions{Renew: true}); err != nil {
		t.Fatalf("Request(--renew) = %v, want nil (exit 0)", err)
	}
	if len(fc.deletes) != 1 {
		t.Fatalf("renew should delete the rejected entitlement once, got %d", len(fc.deletes))
	}
	if len(fc.creates) != 1 {
		t.Fatalf("renew should recreate once, got %d", len(fc.creates))
	}
}

func TestRequestWaitTimesOutWhilePending(t *testing.T) {
	pending := entitlement("compute", servicesv1alpha1.EntitlementPhasePendingApproval, reasonEntitlementPendingApproval, "Waiting for the service provider to approve this request.", nil)
	// List and Get both return pending; the watch never emits a resolving event,
	// so a short --timeout expires while pending → exit 12.
	fc := &fakeClient{listResult: listOf(pending), getResult: pending}
	io, _, errb := testIO("", false)

	err := requesterWith(fc, io).Request(context.Background(), RequestOptions{Wait: true, Timeout: 20 * time.Millisecond})
	wantExit(t, err, ExitPending, StatePendingApproval)
	if !strings.Contains(errb.String(), "Still awaiting provider approval") {
		t.Fatalf("missing timeout copy:\n%s", errb.String())
	}
}

func TestRequestWaitReachesActive(t *testing.T) {
	pending := entitlement("compute", servicesv1alpha1.EntitlementPhasePendingApproval, reasonEntitlementPendingApproval, "Waiting for the service provider to approve this request.", nil)
	active := entitlement("compute", servicesv1alpha1.EntitlementPhaseActive, reasonEntitlementActive, "This service is enabled and ready to use.", ptrNow())
	// Get returns pending initially; the watch then delivers Active.
	fc := &fakeClient{listResult: listOf(pending), getResult: pending, watchEmit: []watch.Event{modifiedEvent(active)}}
	io, _, errb := testIO("", false)

	if err := requesterWith(fc, io).Request(context.Background(), RequestOptions{Wait: true, Timeout: 5 * time.Second}); err != nil {
		t.Fatalf("Request(--wait) = %v, want nil once Active", err)
	}
	if !strings.Contains(errb.String(), "is now enabled") {
		t.Fatalf("missing activation copy:\n%s", errb.String())
	}
}

func TestRequestAlreadyActiveStands(t *testing.T) {
	active := entitlement("compute", servicesv1alpha1.EntitlementPhaseActive, reasonEntitlementActive, "This service is enabled and ready to use.", ptrNow())
	fc := &fakeClient{listResult: listOf(active)}
	io, _, errb := testIO("", false)

	if err := requesterWith(fc, io).Request(context.Background(), RequestOptions{}); err != nil {
		t.Fatalf("Request() on an active service = %v, want nil", err)
	}
	if len(fc.creates) != 0 {
		t.Fatalf("request on an active service must not create")
	}
	if !strings.Contains(errb.String(), "Active") {
		t.Fatalf("missing active status:\n%s", errb.String())
	}
}
