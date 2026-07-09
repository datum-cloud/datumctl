package onboarding

import (
	"testing"

	resourcemanagerv1alpha1 "go.miloapis.com/milo/pkg/apis/resourcemanager/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDerivePortalURL(t *testing.T) {
	tests := []struct {
		apiHost string
		want    string
		wantErr bool
	}{
		{"api.datum.net", "https://cloud.datum.net", false},
		{"https://api.datum.net", "https://cloud.datum.net", false},
		{"api.staging.env.datum.net", "https://cloud.staging.env.datum.net", false},
		{"api.example.com", "", true},
	}

	for _, tt := range tests {
		got, err := DerivePortalURL(tt.apiHost)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("DerivePortalURL(%q) expected error", tt.apiHost)
			}
			continue
		}
		if err != nil {
			t.Fatalf("DerivePortalURL(%q) error: %v", tt.apiHost, err)
		}
		if got != tt.want {
			t.Fatalf("DerivePortalURL(%q) = %q, want %q", tt.apiHost, got, tt.want)
		}
	}
}

func TestOrgProjectsURL(t *testing.T) {
	got := OrgProjectsURL("https://cloud.datum.net", "personal-org-86a07d72")
	want := "https://cloud.datum.net/org/personal-org-86a07d72/projects"
	if got != want {
		t.Fatalf("OrgProjectsURL = %q, want %q", got, want)
	}
}

func TestOrganizationRequestURL(t *testing.T) {
	got := organizationRequestURL("api.datum.net", "user-abc", "chips-coding-tuzu8j")
	want := "https://api.datum.net/apis/iam.miloapis.com/v1alpha1/users/user-abc/control-plane/apis/resourcemanager.miloapis.com/v1alpha1/organizations/chips-coding-tuzu8j"
	if got != want {
		t.Fatalf("organizationRequestURL = %q, want %q", got, want)
	}
}

func TestNoOrgsResult(t *testing.T) {
	result, err := NoOrgsResult("api.staging.env.datum.net")
	if err != nil {
		t.Fatalf("NoOrgsResult() error: %v", err)
	}
	if result.State != NeedsOnboarding {
		t.Fatalf("State = %v, want NeedsOnboarding", result.State)
	}
	want := "https://cloud.staging.env.datum.net"
	if result.ActionURL != want {
		t.Fatalf("ActionURL = %q, want %q", result.ActionURL, want)
	}
}

func TestOrgOnboardingStatus(t *testing.T) {
	tests := []struct {
		name       string
		conds      []metav1.Condition
		complete   bool
		wantReason string
	}{
		{
			name: "ready",
			conds: []metav1.Condition{{
				Type:   resourcemanagerv1alpha1.OrganizationConditionOnboardingComplete,
				Status: metav1.ConditionTrue,
				Reason: resourcemanagerv1alpha1.OrganizationOnboardingCompleteReasonReady,
			}},
			complete: true,
		},
		{
			name: "contact info incomplete",
			conds: []metav1.Condition{{
				Type:    resourcemanagerv1alpha1.OrganizationConditionOnboardingComplete,
				Status:  metav1.ConditionFalse,
				Reason:  resourcemanagerv1alpha1.OrganizationOnboardingCompleteReasonContactInfoIncomplete,
				Message: "Organization contact information requires email and name",
			}},
			complete:   false,
			wantReason: resourcemanagerv1alpha1.OrganizationOnboardingCompleteReasonContactInfoIncomplete,
		},
		{
			name: "billing account missing",
			conds: []metav1.Condition{{
				Type:   resourcemanagerv1alpha1.OrganizationConditionOnboardingComplete,
				Status: metav1.ConditionFalse,
				Reason: resourcemanagerv1alpha1.OrganizationOnboardingCompleteReasonBillingAccountMissing,
			}},
			complete:   false,
			wantReason: resourcemanagerv1alpha1.OrganizationOnboardingCompleteReasonBillingAccountMissing,
		},
		{
			name: "payment method not ready",
			conds: []metav1.Condition{{
				Type:   resourcemanagerv1alpha1.OrganizationConditionOnboardingComplete,
				Status: metav1.ConditionFalse,
				Reason: resourcemanagerv1alpha1.OrganizationOnboardingCompleteReasonPaymentMethodNotReady,
			}},
			complete:   false,
			wantReason: resourcemanagerv1alpha1.OrganizationOnboardingCompleteReasonPaymentMethodNotReady,
		},
		{
			name:       "missing condition",
			conds:      nil,
			complete:   false,
			wantReason: resourcemanagerv1alpha1.OrganizationOnboardingCompleteReasonContactInfoIncomplete,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			org := &resourcemanagerv1alpha1.Organization{
				Status: resourcemanagerv1alpha1.OrganizationStatus{Conditions: tt.conds},
			}
			complete, reason, _ := orgOnboardingStatus(org)
			if complete != tt.complete {
				t.Fatalf("complete = %v, want %v", complete, tt.complete)
			}
			if !tt.complete && reason != tt.wantReason {
				t.Fatalf("reason = %q, want %q", reason, tt.wantReason)
			}
		})
	}
}

func TestUserError(t *testing.T) {
	if err := UserError(Result{State: Complete}); err != nil {
		t.Fatalf("UserError(Complete) = %v, want nil", err)
	}

	needs := UserError(Result{
		State:     NeedsOnboarding,
		ActionURL: "https://cloud.datum.net",
	})
	if needs == nil {
		t.Fatal("expected error for NeedsOnboarding")
	}

	incomplete := UserError(Result{
		State:          OrgIncomplete,
		OrgDisplayName: "Acme",
		Reason:         resourcemanagerv1alpha1.OrganizationOnboardingCompleteReasonBillingAccountMissing,
		ActionURL:      "https://cloud.datum.net/org/personal-org-86a07d72/projects",
	})
	if incomplete == nil {
		t.Fatal("expected error for OrgIncomplete")
	}
}

func TestStatusLabel(t *testing.T) {
	if got := StatusLabel(Result{State: Complete}); got != "ready" {
		t.Fatalf("StatusLabel(Complete) = %q", got)
	}
	if got := StatusLabel(Result{State: NeedsOnboarding}); got != "no org yet" {
		t.Fatalf("StatusLabel(NeedsOnboarding) = %q", got)
	}
}
