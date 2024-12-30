package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/rodaine/table"
	"gopkg.in/yaml.v3"
)

func CLIPrint(w io.Writer, format string, data interface{}, headers []any, rowData [][]any) error {
	switch format {
	case "yaml":
		return printYAML(w, data)
	case "json":
		return printJSON(w, data)
	case "table":
		if headers == nil || rowData == nil {
			return fmt.Errorf("headers and rowData must be provided for table output")
		}
		return printTable(w, headers, rowData)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func printYAML(w io.Writer, data interface{}) error {
	output, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data to YAML: %w", err)
	}
	fmt.Fprint(w, string(output))
	return nil
}

func printJSON(w io.Writer, data interface{}) error {
	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data to JSON: %w", err)
	}
	fmt.Fprint(w, string(output))
	return nil
}

func printTable(w io.Writer, headers []any, rowData [][]any) error {
	t := table.New(headers...)
	t.WithWriter(w)
	for _, row := range rowData {
		t.AddRow(row)
	}
	t.Print()
	return nil
}
