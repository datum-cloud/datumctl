package resourcemanager

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	resourcemanagerpb "buf.build/gen/go/datum-cloud/datum-os/protocolbuffers/go/datum/os/resourcemanager/v1alpha"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type listOrganizationsGraphQLResponse struct {
	Data struct {
		Organizations struct {
			Edges []struct {
				Node struct {
					ID           string    `json:"id"`
					Name         string    `json:"name"`
					UserEntityID string    `json:"userEntityID"`
					CreatedAt    time.Time `json:"createdAt"`
					Description  string    `json:"description"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"organizations"`
	} `json:"data"`
}

type OrganizationsAPI struct {
	// The personal access token to use when authenticating with the API.
	PAT string

	// The hostname to use when connecting to the upstream API to retrieve
	// organizations.
	Hostname string
}

func (r *OrganizationsAPI) ListOrganizations(ctx context.Context, req *resourcemanagerpb.ListOrganizationsRequest) (*resourcemanagerpb.ListOrganizationsResponse, error) {
	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("https://%s/datum-os/query", r.Hostname),
		strings.NewReader(getAllOrganizationsRequest),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+r.PAT)

	client := http.DefaultClient

	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		payload, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("unexpected status code %d from graphql endpoint: %s", httpResp.StatusCode, string(payload))
	}

	var listResp listOrganizationsGraphQLResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	resp := &resourcemanagerpb.ListOrganizationsResponse{}
	for _, org := range listResp.Data.Organizations.Edges {
		resp.Organizations = append(resp.Organizations, &resourcemanagerpb.Organization{
			Name:           "organizations/" + org.Node.UserEntityID,
			DisplayName:    org.Node.Name,
			Uid:            org.Node.ID,
			OrganizationId: org.Node.UserEntityID,
			CreateTime:     timestamppb.New(org.Node.CreatedAt),
			Annotations: map[string]string{
				"meta.datum.net/description": org.Node.Description,
			},
		})
	}

	return resp, nil
}

const getAllOrganizationsRequest = `{
  "operationName": "GetAllOrganizations",
  "query": "query GetAllOrganizations {organizations {edges {node {id name userEntityID createdAt description}}}}"
}`
