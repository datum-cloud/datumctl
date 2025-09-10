package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextcs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
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

type CreateOptions struct {
	Kind         string
	APIVersion   string // optional: "group/version" or "v1"
	Name         string // set either Name or GenerateName
	GenerateName string
	Namespace    string
	Labels       map[string]string
	Annotations  map[string]string
	Spec         map[string]any
	DryRun       bool // default true at the tool layer
}

type GetOptions struct {
	Kind       string
	APIVersion string // optional
	Name       string
	Namespace  string
}

type ApplyOptions struct {
	Kind        string
	APIVersion  string // optional
	Name        string
	Namespace   string
	Labels      map[string]string
	Annotations map[string]string
	Spec        map[string]any
	DryRun      bool
	Force       bool // SSA: claim field ownership on conflicts
}

type DeleteOptions struct {
	Kind       string
	APIVersion string // optional
	Name       string
	Namespace  string
	DryRun     bool
}

func (c *K8sClient) Create(ctx context.Context, opt CreateOptions) (*unstructured.Unstructured, error) {
	rk, err := c.resolveKind(ctx, opt.Kind, opt.APIVersion)
	if err != nil {
		return nil, err
	}
	obj := buildObject(buildObjectOpts{
		APIVersion:   rk.GVK.GroupVersion().String(),
		Kind:         rk.GVK.Kind,
		Name:         opt.Name,
		GenerateName: opt.GenerateName,
		Namespace:    firstNonEmpty(opt.Namespace, c.Namespace),
		Labels:       opt.Labels,
		Annotations:  opt.Annotations,
		Spec:         opt.Spec,
	})

	// keep namespaceable and resource interfaces separate
	nsable := c.dyn.Resource(rk.GVR) // dynamic.NamespaceableResourceInterface
	var ri dynamic.ResourceInterface
	if rk.Namespaced {
		ri = nsable.Namespace(obj.GetNamespace())
	} else {
		ri = nsable // Namespaceable implements ResourceInterface
	}

	return ri.Create(ctx, obj, metav1.CreateOptions{
		DryRun: ternary(opt.DryRun, []string{metav1.DryRunAll}, nil),
	})
}

func (c *K8sClient) Get(ctx context.Context, opt GetOptions) (*unstructured.Unstructured, error) {
	rk, err := c.resolveKind(ctx, opt.Kind, opt.APIVersion)
	if err != nil {
		return nil, err
	}

	nsable := c.dyn.Resource(rk.GVR)
	var ri dynamic.ResourceInterface
	if rk.Namespaced {
		ri = nsable.Namespace(firstNonEmpty(opt.Namespace, c.Namespace))
	} else {
		ri = nsable
	}

	return ri.Get(ctx, opt.Name, metav1.GetOptions{})
}

func (c *K8sClient) Apply(ctx context.Context, opt ApplyOptions) (*unstructured.Unstructured, error) {
	rk, err := c.resolveKind(ctx, opt.Kind, opt.APIVersion)
	if err != nil {
		return nil, err
	}
	obj := buildObject(buildObjectOpts{
		APIVersion:  rk.GVK.GroupVersion().String(),
		Kind:        rk.GVK.Kind,
		Name:        opt.Name,
		Namespace:   firstNonEmpty(opt.Namespace, c.Namespace),
		Labels:      opt.Labels,
		Annotations: opt.Annotations,
		Spec:        opt.Spec,
	})
	jb, _ := json.Marshal(obj.Object)

	nsable := c.dyn.Resource(rk.GVR)
	var ri dynamic.ResourceInterface
	if rk.Namespaced {
		ri = nsable.Namespace(obj.GetNamespace())
	} else {
		ri = nsable
	}

	return ri.Patch(ctx, opt.Name, types.ApplyPatchType, jb, metav1.PatchOptions{
		FieldManager: "datumctl-mcp",
		Force:        &opt.Force,
		DryRun:       ternary(opt.DryRun, []string{metav1.DryRunAll}, nil),
	})
}

func (c *K8sClient) Delete(ctx context.Context, opt DeleteOptions) error {
	rk, err := c.resolveKind(ctx, opt.Kind, opt.APIVersion)
	if err != nil {
		return err
	}

	nsable := c.dyn.Resource(rk.GVR)
	var ri dynamic.ResourceInterface
	if rk.Namespaced {
		ri = nsable.Namespace(firstNonEmpty(opt.Namespace, c.Namespace))
	} else {
		ri = nsable
	}

	return ri.Delete(ctx, opt.Name, metav1.DeleteOptions{
		DryRun: ternary(opt.DryRun, []string{metav1.DryRunAll}, nil),
	})
}

// ListOptions lists instances of a Kind.
type ListOptions struct {
	Kind          string
	APIVersion    string // optional
	Namespace     string
	LabelSelector string
	FieldSelector string
	Limit         int64
	Continue      string
}

func (c *K8sClient) List(ctx context.Context, opt ListOptions) (*unstructured.UnstructuredList, error) {
	rk, err := c.resolveKind(ctx, opt.Kind, opt.APIVersion)
	if err != nil {
		return nil, err
	}
	nsable := c.dyn.Resource(rk.GVR)
	var ri dynamic.ResourceInterface
	if rk.Namespaced {
		ri = nsable.Namespace(firstNonEmpty(opt.Namespace, c.Namespace))
	} else {
		ri = nsable
	}
	return ri.List(ctx, metav1.ListOptions{
		LabelSelector: opt.LabelSelector,
		FieldSelector: opt.FieldSelector,
		Limit:         opt.Limit,
		Continue:      opt.Continue,
	})
}

// ToYAML returns a YAML string for a k8s object
func ToYAML(u *unstructured.Unstructured) (string, error) {
	jb, err := json.Marshal(u.Object)
	if err != nil {
		return "", err
	}
	yb, err := syaml.JSONToYAML(jb)
	if err != nil {
		return "", err
	}
	return string(yb), nil
}

// ToYAMLList renders a list to YAML (like `kubectl get -o yaml`).
func ToYAMLList(l *unstructured.UnstructuredList) (string, error) {
	jb, err := json.Marshal(l)
	if err != nil {
		return "", err
	}
	yb, err := syaml.JSONToYAML(jb)
	if err != nil {
		return "", err
	}
	return string(yb), nil
}

// ----- internal helpers (keep local to this file) -----

type ResolvedKind struct {
	GVR        schema.GroupVersionResource
	GVK        schema.GroupVersionKind
	Namespaced bool
}

// ResolveKind resolves a kind and optional apiVersion to a ResolvedKind
func (c *K8sClient) ResolveKind(ctx context.Context, kind, apiVersion string) (*ResolvedKind, error) {
	return c.resolveKind(ctx, kind, apiVersion)
}

// GetDynamicClient returns the dynamic client interface
func (c *K8sClient) GetDynamicClient() dynamic.Interface {
	return c.dyn
}

func (c *K8sClient) resolveKind(ctx context.Context, kind, apiVersion string) (*ResolvedKind, error) {
	gr, err := restmapper.GetAPIGroupResources(c.disco)
	if err != nil {
		return nil, fmt.Errorf("discovery: %w", err)
	}
	var candidates []ResolvedKind
	for _, grs := range gr {
		group := grs.Group.Name
		for ver, resources := range grs.VersionedResources {
			for _, r := range resources {
				if r.Kind != kind || strings.Contains(r.Name, "/") {
					continue
				}
				gv := schema.GroupVersion{Group: group, Version: ver}
				rk := ResolvedKind{
					GVR:        gv.WithResource(r.Name),
					GVK:        gv.WithKind(kind),
					Namespaced: r.Namespaced,
				}
				// include if no apiVersion filter, or matches explicit apiVersion ("group/version" or "v1")
				if apiVersion == "" || gv.String() == apiVersion || (group == "" && ver == apiVersion) {
					candidates = append(candidates, rk)
				}
			}
		}
	}
	if len(candidates) == 0 {
		return nil, apierrs.NewNotFound(schema.GroupResource{Resource: strings.ToLower(kind)}, kind)
	}
	// If multiple apiVersions provide this Kind and none specified, ask the caller to disambiguate.
	uniq := map[string]struct{}{}
	for _, cnd := range candidates {
		uniq[cnd.GVK.Group+"/"+cnd.GVK.Version] = struct{}{}
	}
	if apiVersion == "" && len(uniq) > 1 {
		return nil, fmt.Errorf("kind %q is available in multiple apiVersions; please specify apiVersion", kind)
	}
	return &candidates[0], nil
}

type buildObjectOpts struct {
	APIVersion   string
	Kind         string
	Name         string
	GenerateName string
	Namespace    string
	Labels       map[string]string
	Annotations  map[string]string
	Spec         map[string]any
}

func buildObject(opts buildObjectOpts) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": opts.APIVersion,
			"kind":       opts.Kind,
			"metadata": map[string]any{
				"labels":      map[string]string{},
				"annotations": map[string]string{},
			},
			"spec": map[string]any{},
		},
	}
	md := obj.Object["metadata"].(map[string]any)
	if opts.Name != "" {
		md["name"] = opts.Name
	}
	if opts.GenerateName != "" && opts.Name == "" {
		md["generateName"] = opts.GenerateName
	}
	if opts.Namespace != "" {
		md["namespace"] = opts.Namespace
	}
	if len(opts.Labels) > 0 {
		md["labels"] = opts.Labels
	}
	if len(opts.Annotations) > 0 {
		md["annotations"] = opts.Annotations
	}
	if opts.Spec != nil {
		obj.Object["spec"] = opts.Spec
	}
	return obj
}

func ternary[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}
func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
