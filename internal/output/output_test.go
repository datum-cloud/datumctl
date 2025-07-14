package output

import (
	"bytes"
	"strings"
	"testing"

	resourcemanagerv1alpha1 "go.miloapis.com/milo/pkg/apis/resourcemanager/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestCLIPrint(t *testing.T) {
	testOrgK8s := &resourcemanagerv1alpha1.Organization{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "resourcemanager.miloapis.com/v1alpha1",
			Kind:       "Organization",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "1234",
		},
		Spec: resourcemanagerv1alpha1.OrganizationSpec{
			Type: "Business",
		},
	}

	tests := []struct {
		name       string
		format     string
		data       runtime.Object
		headers    []any
		rowData    [][]any
		wantErr    bool
		wantOutput []string
	}{
		{
			name:    "Print YAML",
			format:  "yaml",
			data:    testOrgK8s,
			wantErr: false,
			wantOutput: []string{
				"apiVersion: resourcemanager.miloapis.com/v1alpha1",
				"kind: Organization",
				`name: "1234"`,
				"type: Business",
			},
		},
		{
			name:    "Print JSON",
			format:  "json",
			data:    testOrgK8s,
			wantErr: false,
			wantOutput: []string{
				`"apiVersion":"resourcemanager.miloapis.com/v1alpha1"`,
				`"kind":"Organization"`,
				`"name":"1234"`,
				`"type":"Business"`,
			},
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
			data:    testOrgK8s,
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
				out := buf.String()
				if tt.format == "table" {
					if !strings.Contains(out, tt.headers[0].(string)) || !strings.Contains(out, tt.headers[1].(string)) {
						t.Errorf("CLIPrint() output = %v, does not have correct headers", out)
					}
				} else {
					for _, want := range tt.wantOutput {
						if !strings.Contains(out, want) {
							t.Errorf("CLIPrint() output = \n%v, want to contain \n%v", out, want)
						}
					}
				}
			}
		})
	}
}
