package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestCLIPrint(t *testing.T) {
	tests := []struct {
		name       string
		format     string
		data       interface{}
		headers    []any
		rowData    [][]any
		wantErr    bool
		wantOutput string
	}{
		{
			name:       "Print YAML",
			format:     "yaml",
			data:       map[string]string{"key": "value"},
			wantErr:    false,
			wantOutput: "key: value\n",
		},
		{
			name:       "Print JSON",
			format:     "json",
			data:       map[string]string{"key": "value"},
			wantErr:    false,
			wantOutput: "{\n  \"key\": \"value\"\n}",
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
			data:    map[string]string{"key": "value"},
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

			err := CLIPrint(&buf, tt.format, tt.data, tt.headers, tt.rowData)
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
						t.Errorf("CLIPrint() output = %v, want %v", gotOutput, tt.wantOutput)
					}
				}
			}
		})
	}
}
