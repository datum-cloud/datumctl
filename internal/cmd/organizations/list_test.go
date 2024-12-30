package organizations

import (
	"testing"

	resourcemanagerv1alpha "buf.build/gen/go/datum-cloud/datum-os/protocolbuffers/go/datum/os/resourcemanager/v1alpha"
)

func TestGetListOrganizationsTableOutputData(t *testing.T) {
	listOrgs := &resourcemanagerv1alpha.ListOrganizationsResponse{
		Organizations: []*resourcemanagerv1alpha.Organization{
			{
				DisplayName:    "Org1",
				OrganizationId: "org1",
			},
			{
				DisplayName:    "Org2",
				OrganizationId: "org2",
			},
		},
	}

	headers, rowData := getListOrganizationsTableOutputData(listOrgs)
	if len(headers) != 2 {
		t.Errorf("Expected 2 headers, got %d", len(headers))
	}
	if headers[0] != "DISPLAY NAME" {
		t.Errorf("Expected DISPLAY NAME, got %s", headers[0])
	}
	if headers[1] != "RESOURCE ID" {
		t.Errorf("Expected RESOURCE ID, got %s", headers[1])
	}
	if len(rowData) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(rowData))
	}
	if rowData[0][0] != "Org1" {
		t.Errorf("Expected Org1, got %s", rowData[0][0])
	}
	if rowData[0][1] != "org1" {
		t.Errorf("Expected org1, got %s", rowData[0][1])
	}
	if rowData[1][0] != "Org2" {
		t.Errorf("Expected Org2, got %s", rowData[1][0])
	}
	if rowData[1][1] != "org2" {
		t.Errorf("Expected org2, got %s", rowData[1][1])
	}
}
