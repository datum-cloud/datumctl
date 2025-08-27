package client

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextcs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	syaml "sigs.k8s.io/yaml"
)

// K8sClient is a minimal Kubernetes client wrapper used by MCP.
// It is fully constructed by NewK8sFromRESTConfig and does NOT fall back to kubeconfig.
type K8sClient struct {
	cfg       *rest.Config
	disco     discovery.DiscoveryInterface
	dyn       dynamic.Interface
	mapper    meta.RESTMapper
	Namespace string // optional default for namespaced resources
}

// NewK8sFromRESTConfig constructs a fully-initialized client (no kubeconfig fallback).
func NewK8sFromRESTConfig(cfg *rest.Config) (*K8sClient, error) {
	disco, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("discovery: %w", err)
	}
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("dynamic: %w", err)
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(disco))
	return &K8sClient{cfg: cfg, disco: disco, dyn: dyn, mapper: mapper}, nil
}

// Preflight verifies the API server is reachable for the selected context.
func (c *K8sClient) Preflight(ctx context.Context) error {
	if _, err := c.disco.ServerVersion(); err != nil {
		return fmt.Errorf("kubernetes API unreachable: %w", err)
	}
	return nil
}

// ListCRDs returns native CRDs (the MCP layer shapes any view structs).
func (c *K8sClient) ListCRDs(ctx context.Context) ([]*apiextv1.CustomResourceDefinition, error) {
	cs, err := apiextcs.NewForConfig(c.cfg)
	if err != nil {
		return nil, fmt.Errorf("apiextensions client: %w", err)
	}
	list, err := cs.ApiextensionsV1().CustomResourceDefinitions().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	items := make([]*apiextv1.CustomResourceDefinition, 0, len(list.Items))
	for i := range list.Items {
		crd := list.Items[i]
		items = append(items, &crd)
	}
	return items, nil
}

// GetCRD returns the native CRD by name (no formatting).
func (c *K8sClient) GetCRD(ctx context.Context, name string) (*apiextv1.CustomResourceDefinition, error) {
	cs, err := apiextcs.NewForConfig(c.cfg)
	if err != nil {
		return nil, fmt.Errorf("apiextensions client: %w", err)
	}
	crd, err := cs.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return crd, nil
}

// ValidateYAML validates one or more YAML documents using Create with server-side dry-run.
//
// Semantics:
//   - Always use Create (dryRun=All) to exercise creation-time validation for both named and generateName objects.
//   - For named objects, the API will return an AlreadyExists error if the name is taken.
func (c *K8sClient) ValidateYAML(ctx context.Context, manifest string) (bool, string, error) {
	docs := splitYAMLDocuments([]byte(manifest))
	validated := 0

	for _, d := range docs {
		if len(bytes.TrimSpace(d)) == 0 {
			continue
		}
		var obj map[string]interface{}
		if err := syaml.Unmarshal(d, &obj); err != nil {
			return false, "", fmt.Errorf("decode: %w", err)
		}
		u := unstructured.Unstructured{Object: obj}

		// Determine GVK
		gvk := u.GroupVersionKind()
		if gvk.Empty() {
			apiVersion, _, _ := unstructured.NestedString(u.Object, "apiVersion")
			kind, _, _ := unstructured.NestedString(u.Object, "kind")
			if apiVersion == "" || kind == "" {
				return false, "", fmt.Errorf("apiVersion/kind required")
			}
			parts := strings.Split(apiVersion, "/")
			switch len(parts) {
			case 1:
				gvk = schema.GroupVersionKind{Group: "", Version: parts[0], Kind: kind}
			default:
				gvk = schema.GroupVersionKind{Group: parts[0], Version: parts[1], Kind: kind}
			}
			u.SetGroupVersionKind(gvk)
		}

		// REST mapping
		mapping, err := c.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return false, "", fmt.Errorf("rest mapping for %s: %w", gvk.String(), err)
		}

		// Namespace selection
		ns := u.GetNamespace()
		nsToUse := ""
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			if ns == "" {
				nsToUse = c.Namespace
				u.SetNamespace(nsToUse)
			} else {
				nsToUse = ns
			}
		}

		// Dynamic client interface: resolve to ResourceInterface (ns empty = cluster-scope)
		ri := c.dyn.Resource(mapping.Resource).Namespace(nsToUse)

		// Creation-time validation: Create with dry-run (works for named and generateName)
		if _, err := ri.Create(ctx, &u, metav1.CreateOptions{
			DryRun: []string{metav1.DryRunAll},
		}); err != nil {
			return false, "", err
		}

		validated++
	}

	return true, fmt.Sprintf("validated %d object(s) (server-side dry-run)", validated), nil
}

// splitYAMLDocuments splits multi-doc YAML on '---' boundaries.
func splitYAMLDocuments(in []byte) [][]byte {
	parts := bytes.Split(in, []byte("\n---"))
	out := make([][]byte, 0, len(parts))
	for _, p := range parts {
		if len(bytes.TrimSpace(p)) > 0 {
			out = append(out, p)
		}
	}
	return out
}
