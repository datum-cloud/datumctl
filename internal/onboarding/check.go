package onboarding

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	resourcemanagerv1alpha1 "go.miloapis.com/milo/pkg/apis/resourcemanager/v1alpha1"
	"golang.org/x/oauth2"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	customerrors "go.datum.net/datumctl/internal/errors"
	"go.datum.net/datumctl/internal/miloapi"
)

const displayNameAnnotation = "kubernetes.io/display-name"

// State describes the user's onboarding readiness.
type State int

const (
	Complete State = iota
	NeedsOnboarding
	OrgIncomplete
)

// Result holds the outcome of an onboarding status check.
type Result struct {
	State          State
	PortalURL      string
	ActionURL      string
	OrgID          string
	OrgDisplayName string
	Reason         string
	Message        string
}

// CheckOrg evaluates whether a specific organization has completed cloud-portal
// onboarding by reading its OnboardingComplete condition from the user control
// plane (same path cloud-portal uses).
func CheckOrg(
	ctx context.Context,
	apiHostname string,
	tokenSource oauth2.TokenSource,
	userID, orgID, orgDisplayName string,
) (Result, error) {
	if orgID == "" {
		return Result{}, fmt.Errorf("organization ID is required")
	}
	if userID == "" {
		return Result{}, fmt.Errorf("user ID is required")
	}

	portalBase, err := DerivePortalURL(apiHostname)
	if err != nil {
		return Result{}, err
	}

	org, err := fetchOrganization(ctx, apiHostname, tokenSource, userID, orgID)
	if err != nil {
		return Result{}, err
	}

	displayName := orgDisplayName
	if displayName == "" {
		displayName = org.Annotations[displayNameAnnotation]
	}
	if displayName == "" {
		displayName = orgID
	}

	if complete, reason, message := orgOnboardingStatus(&org); !complete {
		return Result{
			State:          OrgIncomplete,
			PortalURL:      portalBase,
			ActionURL:      OrgProjectsURL(portalBase, orgID),
			OrgID:          orgID,
			OrgDisplayName: displayName,
			Reason:         reason,
			Message:        message,
		}, nil
	}

	return Result{State: Complete, PortalURL: portalBase, OrgID: orgID, OrgDisplayName: displayName}, nil
}

func orgOnboardingStatus(org *resourcemanagerv1alpha1.Organization) (complete bool, reason, message string) {
	cond := apimeta.FindStatusCondition(org.Status.Conditions, resourcemanagerv1alpha1.OrganizationConditionOnboardingComplete)
	if cond == nil {
		return false,
			resourcemanagerv1alpha1.OrganizationOnboardingCompleteReasonContactInfoIncomplete,
			"We couldn't tell if setup is finished yet"
	}
	if cond.Status == metav1.ConditionTrue &&
		cond.Reason == resourcemanagerv1alpha1.OrganizationOnboardingCompleteReasonReady {
		return true, cond.Reason, cond.Message
	}
	reason = cond.Reason
	if reason == "" {
		reason = resourcemanagerv1alpha1.OrganizationOnboardingCompleteReasonContactInfoIncomplete
	}
	message = cond.Message
	if message == "" {
		message = "Setup isn't finished yet"
	}
	return false, reason, message
}

func portalHint(actionURL string) string {
	return fmt.Sprintf("Head to %s to finish up.", actionURL)
}

func orgIncompleteMessage(displayName, reason, apiMessage string) string {
	name := displayName
	if name == "" {
		name = "This organization"
	}

	if apiMessage != "" && apiMessage != humanReason(reason) && !strings.HasPrefix(apiMessage, "Organization ") {
		return fmt.Sprintf("%s still needs setup: %s.", name, apiMessage)
	}

	switch reason {
	case resourcemanagerv1alpha1.OrganizationOnboardingCompleteReasonContactInfoIncomplete:
		return fmt.Sprintf("%s still needs your contact details before you can use datumctl here.", name)
	case resourcemanagerv1alpha1.OrganizationOnboardingCompleteReasonBillingAccountMissing:
		return fmt.Sprintf("%s still needs billing set up before you can use datumctl here.", name)
	case resourcemanagerv1alpha1.OrganizationOnboardingCompleteReasonPaymentMethodNotReady:
		return fmt.Sprintf("%s still needs a payment method on file before you can use datumctl here.", name)
	default:
		return fmt.Sprintf("%s still needs a little setup in the portal before you can use datumctl here.", name)
	}
}

// UserError converts a non-complete Result into a user-facing error with a
// portal link hint. Returns nil when onboarding is complete.
func UserError(result Result) error {
	if result.State == Complete {
		return nil
	}

	hint := portalHint(result.ActionURL)
	switch result.State {
	case NeedsOnboarding:
		return customerrors.NewUserErrorWithHint(
			"You're signed in, but you don't have an organization yet.",
			hint,
		)
	case OrgIncomplete:
		return customerrors.NewUserErrorWithHint(
			orgIncompleteMessage(result.OrgDisplayName, result.Reason, result.Message),
			hint,
		)
	default:
		return customerrors.NewUserErrorWithHint(
			"Your account still needs a little setup in the portal.",
			hint,
		)
	}
}

func humanReason(reason string) string {
	switch reason {
	case resourcemanagerv1alpha1.OrganizationOnboardingCompleteReasonContactInfoIncomplete:
		return "contact details needed"
	case resourcemanagerv1alpha1.OrganizationOnboardingCompleteReasonBillingAccountMissing:
		return "billing setup needed"
	case resourcemanagerv1alpha1.OrganizationOnboardingCompleteReasonPaymentMethodNotReady:
		return "payment method needed"
	case resourcemanagerv1alpha1.OrganizationOnboardingCompleteReasonReady:
		return "ready"
	default:
		if reason == "" {
			return "setup needed"
		}
		return reason
	}
}

// StatusLabel returns a short status string for display in whoami and landing.
func StatusLabel(result Result) string {
	switch result.State {
	case Complete:
		return "ready"
	case NeedsOnboarding:
		return "no org yet"
	case OrgIncomplete:
		if result.OrgDisplayName != "" {
			return fmt.Sprintf("needs setup (%s: %s)", result.OrgDisplayName, humanReason(result.Reason))
		}
		return fmt.Sprintf("needs setup (%s)", humanReason(result.Reason))
	default:
		return "unknown"
	}
}

func organizationRequestURL(apiHostname, userID, orgID string) string {
	userCP := miloapi.UserControlPlaneURL(apiHostname, userID)
	return fmt.Sprintf(
		"%s/apis/resourcemanager.miloapis.com/v1alpha1/organizations/%s",
		userCP,
		url.PathEscape(orgID),
	)
}

func fetchOrganization(
	ctx context.Context,
	apiHostname string,
	tokenSource oauth2.TokenSource,
	userID, orgID string,
) (resourcemanagerv1alpha1.Organization, error) {
	requestURL := organizationRequestURL(apiHostname, userID, orgID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return resourcemanagerv1alpha1.Organization{}, err
	}
	req.Header.Set("Accept", "application/json")

	rt := &oauth2.Transport{Source: tokenSource, Base: http.DefaultTransport}
	resp, err := rt.RoundTrip(req)
	if err != nil {
		return resourcemanagerv1alpha1.Organization{}, fmt.Errorf("get organization %s: %w", orgID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		msg := strings.TrimSpace(string(body))
		if msg != "" {
			return resourcemanagerv1alpha1.Organization{}, fmt.Errorf("get organization %s: HTTP %d: %s", orgID, resp.StatusCode, msg)
		}
		return resourcemanagerv1alpha1.Organization{}, fmt.Errorf("get organization %s: HTTP %d", orgID, resp.StatusCode)
	}

	var org resourcemanagerv1alpha1.Organization
	if err := json.NewDecoder(resp.Body).Decode(&org); err != nil {
		return resourcemanagerv1alpha1.Organization{}, fmt.Errorf("decode organization %s: %w", orgID, err)
	}
	return org, nil
}
