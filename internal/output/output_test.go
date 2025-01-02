package output

import (
	"bytes"
	"strings"
	"testing"

	resourcemanagerv1alpha "buf.build/gen/go/datum-cloud/datum-os/protocolbuffers/go/datum/os/resourcemanager/v1alpha"
	"google.golang.org/protobuf/proto"
)

func TestCLIPrint(t *testing.T) {

	testOrgProto := &resourcemanagerv1alpha.Organization{
		DisplayName:    "Test Organization",
		OrganizationId: "1234",
	}

	tests := []struct {
		name       string
		format     string
		data       proto.Message
		headers    []any
		rowData    [][]any
		wantErr    bool
		wantOutput string
	}{
		{
			name:       "Print YAML",
			format:     "yaml",
			data:       testOrgProto,
			wantErr:    false,
			wantOutput: "organizationId: \"1234\"\ndisplayName: Test Organization\n",
		},
		{
			name:       "Print JSON",
			format:     "json",
			data:       testOrgProto,
			wantErr:    false,
			wantOutput: "{\"organizationId\":\"1234\",\"displayName\":\"Test Organization\"}",
		},
		{
			name:    "Print Table",
			format:  "table",
			headers: []any{"Header1", "Header2"},
			rowData: [][]any{{"Row1Col1", "Row1Col2"}, {"Row2Col1", "Row2Col2"}},
			wantErr: false,
		},
		{
			name:    "Unsupported Format",
			format:  "unsupported",
			data:    testOrgProto,
			wantErr: true,
		},
		{
			name:    "Table without headers and rowData",
			format:  "table",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			err := CLIPrint(&buf, tt.format, tt.data, func() (ColumnFormatter, RowFormatterFunc) {
				return tt.headers, func() RowFormatter {
					return tt.rowData
				}
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("CLIPrint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if tt.format == "table" {
					out := buf.String()
					if !strings.Contains(out, tt.headers[0].(string)) || !strings.Contains(out, tt.headers[1].(string)) {
						t.Errorf("CLIPrint() output = %v, does not have correct headers", out)
					}
				} else {
					if gotOutput := buf.String(); gotOutput != tt.wantOutput {
						t.Errorf("CLIPrint() output = \n%v, want \n%v", gotOutput, tt.wantOutput)
					}
				}
			}
		})
	}
}
