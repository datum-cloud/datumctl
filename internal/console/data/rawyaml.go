package data

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

// RawYAML marshals an unstructured object to YAML, matching kubectl -o yaml output.
func RawYAML(obj *unstructured.Unstructured) (string, error) {
	if obj == nil {
		return "", nil
	}
	b, err := yaml.Marshal(obj.Object)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
