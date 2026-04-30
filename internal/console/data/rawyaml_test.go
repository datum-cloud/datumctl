package data

import (
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestRawYAML_NilInput_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	got, err := RawYAML(nil)
	if err != nil {
		t.Fatalf("RawYAML(nil): err = %v, want nil", err)
	}
	if got != "" {
		t.Errorf("RawYAML(nil) = %q, want empty string", got)
	}
}

func TestRawYAML_ValidObject_MarshalsPrimaryFields(t *testing.T) {
	t.Parallel()
	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]any{
				"name":      "test-pod",
				"namespace": "default",
			},
			"spec": map[string]any{
				"nodeName": "node-1",
			},
		},
	}
	got, err := RawYAML(obj)
	if err != nil {
		t.Fatalf("RawYAML: unexpected error: %v", err)
	}
	for _, want := range []string{"apiVersion", "v1", "kind", "Pod", "test-pod", "node-1"} {
		if !strings.Contains(got, want) {
			t.Errorf("RawYAML: want %q in YAML output, got:\n%s", want, got)
		}
	}
}

func TestRawYAML_EmptyObject_ReturnsEmptyDoc(t *testing.T) {
	t.Parallel()
	obj := &unstructured.Unstructured{Object: map[string]any{}}
	got, err := RawYAML(obj)
	if err != nil {
		t.Fatalf("RawYAML(empty): unexpected error: %v", err)
	}
	// An empty map marshals to "{}\n" in YAML.
	if got == "" {
		t.Errorf("RawYAML(empty): got empty string, want non-empty YAML doc")
	}
}
