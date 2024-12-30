package output

import (
	"fmt"
	"io"

	"buf.build/go/protoyaml"
	"github.com/rodaine/table"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type ClIPrinter struct {
	Writer       io.Writer
	TableColumns []string
}

func CLIPrint(w io.Writer, format string, data proto.Message, headers []any, rowData [][]any) error {
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

func printYAML(w io.Writer, data proto.Message) error {
	marshaller := protoyaml.MarshalOptions{
		Indent: 2,
	}

	output, err := marshaller.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data to YAML: %w", err)
	}
	fmt.Fprint(w, string(output))
	return nil
}

func printJSON(w io.Writer, data proto.Message) error {
	output, err := protojson.Marshal(data)
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
