package data

import "strings"

// groupDisplayNames maps known Datum API groups to their curated sidebar labels.
// The empty string key represents the Kubernetes core group (no group prefix).
var groupDisplayNames = map[string]string{
	"networking.datumapis.com":      "NETWORKING",
	"iam.datumapis.com":             "IAM",
	"compute.datumapis.com":         "COMPUTE",
	"resourcemanager.datumapis.com": "RESOURCE MGMT",
	"":                              "CORE",
}

// GroupDisplayName returns the curated display name for a known API group.
// For unknown groups it uses only the first domain label, uppercased (e.g.
// "foo.bar.baz" → "FOO"). The core group (empty string) returns "CORE".
func GroupDisplayName(group string) string {
	if name, ok := groupDisplayNames[group]; ok {
		return name
	}
	if i := strings.Index(group, "."); i != -1 {
		return strings.ToUpper(group[:i])
	}
	return strings.ToUpper(group)
}

// hiddenGroups lists Kubernetes platform API groups that are not usable with
// Datum and should be filtered from the sidebar.
var hiddenGroups = map[string]bool{
	"rbac.authorization.k8s.io":       true,
	"coordination.k8s.io":             true,
	"events.k8s.io":                   true,
	"storage.k8s.io":                  true,
	"policy":                          true,
	"authentication.k8s.io":           true,
	"authorization.k8s.io":            true,
	"certificates.k8s.io":             true,
	"scheduling.k8s.io":               true,
	"node.k8s.io":                     true,
	"flowcontrol.apiserver.k8s.io":    true,
	"admissionregistration.k8s.io":    true,
	"apiextensions.k8s.io":            true,
	"apiregistration.k8s.io":          true,
}

// hiddenCoreKinds lists specific core-group kinds that are Kubernetes platform
// internals and should not be shown in the Datum sidebar.
var hiddenCoreKinds = map[string]bool{
	"Node":            true,
	"ComponentStatus": true,
	"Binding":         true,
	"Event":           true,
}

// ShouldHideResourceType reports whether rt is a Kubernetes platform type that
// is not usable with Datum. It returns true for any type whose group is in the
// hidden-group set, and for specific kinds in the core ("") group.
func ShouldHideResourceType(rt ResourceType) bool {
	if hiddenGroups[rt.Group] {
		return true
	}
	if rt.Group == "" && hiddenCoreKinds[rt.Kind] {
		return true
	}
	return false
}
