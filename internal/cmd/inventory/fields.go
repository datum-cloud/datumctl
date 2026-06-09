package inventory

import (
	"strconv"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const none = "<none>"

// str reads a nested string field, returning "<none>" when absent or empty.
func str(u unstructured.Unstructured, fields ...string) string {
	v, found, err := unstructured.NestedString(u.Object, fields...)
	if err != nil || !found || v == "" {
		return none
	}
	return v
}

// intStr reads a nested integer field as a string, "<none>" when absent.
func intStr(u unstructured.Unstructured, fields ...string) string {
	v, found, err := unstructured.NestedInt64(u.Object, fields...)
	if err != nil || !found {
		return none
	}
	return strconv.FormatInt(v, 10)
}

// ready returns the status of the "Ready" condition ("True"/"False"), or
// "<none>" when the object carries no such condition yet.
func ready(u unstructured.Unstructured) string {
	conds, found, err := unstructured.NestedSlice(u.Object, "status", "conditions")
	if err != nil || !found {
		return none
	}
	for _, c := range conds {
		m, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		if m["type"] == "Ready" {
			if s, ok := m["status"].(string); ok && s != "" {
				return s
			}
		}
	}
	return none
}
