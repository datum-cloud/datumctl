package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	resourcemanagerv1alpha1 "go.miloapis.com/milo/pkg/apis/resourcemanager/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"go.datum.net/datumctl/internal/onboarding"
)

func TestIsOrganizationsAlias(t *testing.T) {
	tests := []struct {
		args []string
		want bool
	}{
		{nil, false},
		{[]string{}, false},
		{[]string{"organizations"}, true},
		{[]string{"organization"}, true},
		{[]string{"organizationmemberships"}, false},
		{[]string{"projects"}, false},
	}
	for _, tt := range tests {
		if got := isOrganizationsAlias(tt.args); got != tt.want {
			t.Fatalf("isOrganizationsAlias(%v) = %v, want %v", tt.args, got, tt.want)
		}
	}
}

func TestPrintOrganizationsTable(t *testing.T) {
	now := metav1.NewTime(time.Now().Add(-2 * time.Hour))
	items := []resourcemanagerv1alpha1.OrganizationMembership{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "m-ready", CreationTimestamp: now},
			Spec: resourcemanagerv1alpha1.OrganizationMembershipSpec{
				OrganizationRef: resourcemanagerv1alpha1.OrganizationReference{Name: "org-ready"},
			},
			Status: resourcemanagerv1alpha1.OrganizationMembershipStatus{
				Organization: resourcemanagerv1alpha1.OrganizationMembershipOrganizationStatus{
					DisplayName: "Ready Org",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "m-billing", CreationTimestamp: now},
			Spec: resourcemanagerv1alpha1.OrganizationMembershipSpec{
				OrganizationRef: resourcemanagerv1alpha1.OrganizationReference{Name: "org-billing"},
			},
			Status: resourcemanagerv1alpha1.OrganizationMembershipStatus{
				Organization: resourcemanagerv1alpha1.OrganizationMembershipOrganizationStatus{
					DisplayName: "Billing Org",
				},
			},
		},
	}
	statuses := map[string]onboarding.Result{
		"org-ready": {
			State:          onboarding.Complete,
			OrgID:          "org-ready",
			OrgDisplayName: "Ready Org",
		},
		"org-billing": {
			State:          onboarding.OrgIncomplete,
			OrgID:          "org-billing",
			OrgDisplayName: "Billing Org",
			Reason:         resourcemanagerv1alpha1.OrganizationOnboardingCompleteReasonBillingAccountMissing,
			ActionURL:      "https://cloud.datum.net/org/org-billing/projects",
		},
	}

	var buf bytes.Buffer
	if err := printOrganizationsTable(&buf, items, statuses, false); err != nil {
		t.Fatalf("printOrganizationsTable: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"ORGANIZATION",
		"STATUS",
		"org-ready",
		"ready",
		"org-billing",
		"billing setup needed",
		"Some organizations still need setup",
		"https://cloud.datum.net/org/org-billing/projects",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}
