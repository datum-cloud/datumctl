// Package resourcemanager provides a client and methods for interacting
// with the Datum resource manager API.
package resourcemanager

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	iamrmv1alpha "buf.build/gen/go/datum-cloud/iam/protocolbuffers/go/datum/resourcemanager/v1alpha"
	"google.golang.org/protobuf/encoding/protojson"
)

// Client provides methods for interacting with the Datum resource manager API.
// It handles the underlying HTTP requests and authentication.
type Client struct {
	// HTTPClient is the `*http.Client` to use for making API requests.
	// This client should be pre-configured with any necessary authentication,
	// such as an OAuth2 token source.
	HTTPClient *http.Client

	// APIHostname is the hostname of the Datum resource manager API
	// (e.g., "api.example.com").
	APIHostname string
}

// NewClient creates and returns a new Client for interacting with the
// Datum resource manager API. It requires an HTTP client, which should be
// pre-configured for authentication (e.g., with an OAuth2 token source),
// and the API hostname.
func NewClient(httpClient *http.Client, apiHostname string) *Client {
	return &Client{
		HTTPClient:  httpClient,
		APIHostname: apiHostname,
	}
}

// ListOrganizations retrieves a list of organizations accessible by the authenticated user
// from the Datum IAM resource manager API. It makes a POST request to the
// /datum-os/iam/v1alpha/organizations:search endpoint.
//
// The provided context.Context can be used for request cancellation or timeouts.
// It returns a SearchOrganizationsResponse containing the organizations or an
// error if the API request fails or the response cannot be processed.
func (c *Client) ListOrganizations(ctx context.Context) (*iamrmv1alpha.SearchOrganizationsResponse, error) {
	url := fmt.Sprintf("https://%s/datum-os/iam/v1alpha/organizations:search", c.APIHostname)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader("{}"))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request to list organizations: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list organizations, status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var searchRespProto iamrmv1alpha.SearchOrganizationsResponse
	// Using UnmarshalOptions to be more resilient, e.g. if API adds new fields not yet in client's proto.
	unmarshalOpts := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	if err := unmarshalOpts.Unmarshal(bodyBytes, &searchRespProto); err != nil {
		return nil, fmt.Errorf("failed to decode organizations list response into protobuf: %w", err)
	}

	return &searchRespProto, nil
}
