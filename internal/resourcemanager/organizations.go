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
	// HTTPClient is the client used to make requests, pre-configured with auth.
	HTTPClient *http.Client

	// The hostname to use when connecting to the upstream API to retrieve
	// organizations.
	Hostname string
}

func (o *OrganizationsAPI) ListOrganizations(ctx context.Context, _ *resourcemanagerpb.ListOrganizationsRequest) (*resourcemanagerpb.ListOrganizationsResponse, error) {
	body := strings.NewReader(`{
		"query": "query { organizations { edges { node { id name userEntityID createdAt description } } } }"
	}`) // TODO: Use proper GraphQL query generation

	url := fmt.Sprintf("https://%s/graphql", o.Hostname)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	// Authentication is now handled by the HTTPClient's Transport (e.g., OAuth2)

	client := o.HTTPClient
	if client == nil {
		// Fallback to default client if none provided, though this might lack auth
		// Consider returning an error if HTTPClient is nil and required.
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d from GraphQL endpoint: %s", resp.StatusCode, string(bodyBytes))
	}

	var gqlResp listOrganizationsGraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	// Convert GraphQL response to protobuf response
	pbResp := &resourcemanagerpb.ListOrganizationsResponse{}
	for _, edge := range gqlResp.Data.Organizations.Edges {
		node := edge.Node
		pbResp.Organizations = append(pbResp.Organizations, &resourcemanagerpb.Organization{
			OrganizationId: node.ID,   // Assuming ID is the resource ID
			DisplayName:    node.Name, // Assuming Name is the display name
			CreateTime:     timestamppb.New(node.CreatedAt),
			// Add other fields if necessary and available in GraphQL response
		})
	}

	return pbResp, nil
}
