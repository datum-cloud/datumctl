package output

import (
	"fmt"
	"io"

	"github.com/rodaine/table"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

type TableFormatterFunc func() (ColumnFormatter, RowFormatterFunc)
type RowFormatterFunc func() RowFormatter
type RowFormatter [][]any
type ColumnFormatter []any

// CLIPrint outputs a Kubernetes runtime.Object in the specified format using kubectl-compatible marshalling
func CLIPrint(w io.Writer, format string, data runtime.Object, tableFormatterFunc TableFormatterFunc) error {
	switch format {
	case "yaml":
		return printYAML(w, data)
	case "json":
		return printJSON(w, data)
	case "table":
		headers, rowDataFunc := tableFormatterFunc()
		if headers == nil {
			return fmt.Errorf("headers must be provided for table output")
		}
		return printTable(w, headers, rowDataFunc())
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func printYAML(w io.Writer, data runtime.Object) error {
	// Create a YAML serializer using the default Kubernetes scheme
	// This uses the same approach as kubectl for marshalling objects
	serializer := json.NewYAMLSerializer(json.DefaultMetaFactory, clientgoscheme.Scheme, clientgoscheme.Scheme)

	return serializer.Encode(data, w)
}

func printJSON(w io.Writer, data runtime.Object) error {
	// Create a JSON serializer using the default Kubernetes scheme
	// This uses the same approach as kubectl for marshalling objects
	serializer := json.NewSerializer(json.DefaultMetaFactory, clientgoscheme.Scheme, clientgoscheme.Scheme, false)

	return serializer.Encode(data, w)
}

func printTable(w io.Writer, headers ColumnFormatter, rowData [][]any) error {
	t := table.New(headers...)
	t.WithWriter(w)
	for _, row := range rowData {
		t.AddRow(row...)
	}
	t.Print()
	return nil
}
