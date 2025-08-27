// internal/kube/client.go
// Package kube provides a client-go–backed Kubernetes helper used by the MCP service.
// It exposes small, deterministic functions for:
//   • Listing served CRDs
//   • Validating manifests via server-side dry-run apply
//   • Fetching a CRD in yaml/json or a short "describe" view
//   • Preflight connectivity/version check
package kube

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"

	apixclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apixv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"

	_ "k8s.io/client-go/plugin/pkg/client/auth" // auth providers (GCP, Azure, OIDC, exec)
	"sigs.k8s.io/yaml"
)

// Kubectl is a thin wrapper around client-go components used by datumctl.
// Connections are created lazily on first use.
type Kubectl struct {
	// Optional kubeconfig context override.
	Context string

	// Default namespace used when a namespaced object omits metadata.namespace.
	Namespace string

	// Reserved for API stability; no effect.
	Path string
	Env  []string

	// internal clients (initialized on first use)
	once    sync.Once
	initErr error

	cfg    *rest.Config
	kube   *kubernetes.Clientset
	apix   *apixclient.Clientset
	dyn    dynamic.Interface
	disco  discovery.DiscoveryInterface
	mapper *restmapper.DeferredDiscoveryRESTMapper
}

// New returns a new helper; clients are initialized lazily.
func New() *Kubectl { return &Kubectl{} }

// CRDItem is a compact view of a CustomResourceDefinition.
type CRDItem struct {
	Name     string   `json:"name"`     // e.g. "httpproxies.networking.datumapis.com"
	Group    string   `json:"group"`    // e.g. "networking.datumapis.com"
	Kind     string   `json:"kind"`     // Kind from CRD spec
	Versions []string `json:"versions"` // served versions
}

// ListCRDs returns the set of served CRDs in the cluster.
func (k *Kubectl) ListCRDs(ctx context.Context) ([]CRDItem, error) {
	if err := k.ensureClusterReachable(ctx); err != nil {
		return nil, err
	}

	crds, err := k.apix.ApiextensionsV1().CustomResourceDefinitions().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list CRDs: %w", err)
	}

	out := make([]CRDItem, 0, len(crds.Items))
	for _, crd := range crds.Items {
		versions := make([]string, 0, len(crd.Spec.Versions))
		for _, v := range crd.Spec.Versions {
			if v.Served {
				versions = append(versions, v.Name)
			}
		}
		sort.Strings(versions)
		out = append(out, CRDItem{
			Name:     crd.Name,
			Group:    crd.Spec.Group,
			Kind:     crd.Spec.Names.Kind,
			Versions: versions,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Group == out[j].Group {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Group < out[j].Group
	})
	return out, nil
}

// ValidateYAML validates a multi-document YAML/JSON manifest by issuing a
// server-side apply (DryRun=All) for each object. It returns (ok, message, err):
//   - ok=false, err=nil   → validation failed; message explains why.
//   - ok=true,  err=nil   → validation succeeded; message summarizes the result.
//   - err!=nil            → transport/config error.
func (k *Kubectl) ValidateYAML(ctx context.Context, manifest string) (bool, string, error) {
	if err := k.ensureClusterReachable(ctx); err != nil {
		return false, "", err
	}

	dec := k8syaml.NewYAMLOrJSONDecoder(strings.NewReader(manifest), 4096)
	docN := 0
	for {
		var raw map[string]any
		if err := dec.Decode(&raw); err != nil {
			if err == io.EOF {
				break
			}
			return false, fmt.Sprintf("YAML parse error: %v", err), nil
		}
		if len(raw) == 0 {
			continue
		}
		docN++

		u := &unstructured.Unstructured{Object: raw}
		gvk := u.GroupVersionKind()
		if gvk.Empty() {
			apiVersion, _, _ := unstructured.NestedString(raw, "apiVersion")
			kind, _, _ := unstructured.NestedString(raw, "kind")
			if apiVersion == "" || kind == "" {
				return false, "object is missing apiVersion and/or kind", nil
			}
			gv, err := schema.ParseGroupVersion(apiVersion)
			if err != nil {
				return false, fmt.Sprintf("invalid apiVersion %q: %v", apiVersion, err), nil
			}
			gvk = gv.WithKind(kind)
			u.SetAPIVersion(apiVersion)
			u.SetKind(kind)
		}

		// Resolve resource (GVR) via RESTMapper.
		mapping, err := k.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return false, fmt.Sprintf("unable to map %s: %v", gvk.String(), err), nil
		}

		// Choose the correct resource interface (namespaced or cluster-scoped).
		var ri dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			ns := u.GetNamespace()
			if ns == "" {
				if k.Namespace != "" {
					ns = k.Namespace
				} else {
					ns = "default"
				}
			}
			ri = k.dyn.Resource(mapping.Resource).Namespace(ns)
		} else {
			ri = k.dyn.Resource(mapping.Resource)
		}

		// Server-side apply (dry run) performs full API validation without persisting.
		data, err := json.Marshal(u.Object)
		if err != nil {
			return false, fmt.Sprintf("encode %s/%s: %v", u.GetKind(), u.GetName(), err), nil
		}
		if u.GetName() == "" {
			return false, fmt.Sprintf("%s is missing metadata.name", gvk.Kind), nil
		}

		_, err = ri.Patch(
			ctx,
			u.GetName(),
			types.ApplyPatchType,
			data,
			metav1.PatchOptions{
				DryRun:       []string{metav1.DryRunAll},
				FieldManager: "datumctl-validate",
			},
		)
		if err != nil {
			// Treat API validation failures as ok=false with a descriptive message.
			if statusErr, ok := err.(*apierrors.StatusError); ok {
				status := statusErr.ErrStatus
				reason := status.Message
				if status.Details != nil && len(status.Details.Causes) > 0 {
					var b strings.Builder
					_, _ = fmt.Fprintf(&b, "%s:", status.Message)
					for _, c := range status.Details.Causes {
						_, _ = fmt.Fprintf(&b, " %s", c.Message)
					}
					reason = b.String()
				}
				return false, reason, nil
			}
			return false, err.Error(), nil
		}
	}

	return true, fmt.Sprintf("validated %d object(s) (server-side dry-run)", docN), nil
}

// GetCRD returns the CRD by name formatted as yaml|json|describe (default: yaml).
func (k *Kubectl) GetCRD(ctx context.Context, name, mode string) (string, error) {
	if err := k.ensureClusterReachable(ctx); err != nil {
		return "", err
	}
	crd, err := k.apix.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get CRD %q: %w", name, err)
	}
	switch strings.ToLower(mode) {
	case "", "yaml":
		js, err := json.Marshal(crd)
		if err != nil {
			return "", err
		}
		yb, err := yaml.JSONToYAML(js)
		if err != nil {
			return "", err
		}
		return string(yb), nil
	case "json":
		js, err := json.MarshalIndent(crd, "", "  ")
		if err != nil {
			return "", err
		}
		return string(js), nil
	case "describe":
		return formatCRDDescribe(crd), nil
	default:
		return "", fmt.Errorf("unsupported mode %q (use yaml|json|describe)", mode)
	}
}

// Preflight verifies API connectivity (no return payload).
func (k *Kubectl) Preflight(ctx context.Context) error {
	if err := k.ensureClusterReachable(ctx); err != nil {
		return err
	}
	if _, err := k.disco.ServerVersion(); err != nil {
		return fmt.Errorf("server version: %w", err)
	}
	return nil
}


// ensureClusterReachable initializes clients and verifies the API server is reachable.
func (k *Kubectl) ensureClusterReachable(ctx context.Context) error {
	if err := k.initClients(); err != nil {
		return err
	}
	if _, err := k.disco.ServerVersion(); err != nil {
		return fmt.Errorf("cannot reach Kubernetes API for the selected context: %w", err)
	}
	return nil
}

// initClients loads kubeconfig and constructs client-go clients on first use.
func (k *Kubectl) initClients() error {
	k.once.Do(func() {
		rules := clientcmd.NewDefaultClientConfigLoadingRules()
		overrides := &clientcmd.ConfigOverrides{}
		if k.Context != "" {
			overrides.CurrentContext = k.Context
		}
		cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
		if err != nil {
			k.initErr = fmt.Errorf("load kubeconfig: %w", err)
			return
		}
		k.cfg = cfg

		if k.kube, err = kubernetes.NewForConfig(cfg); err != nil {
			k.initErr = fmt.Errorf("kubernetes client: %w", err)
			return
		}
		if k.apix, err = apixclient.NewForConfig(cfg); err != nil {
			k.initErr = fmt.Errorf("apiextensions client: %w", err)
			return
		}
		if k.dyn, err = dynamic.NewForConfig(cfg); err != nil {
			k.initErr = fmt.Errorf("dynamic client: %w", err)
			return
		}
		disco, err := discovery.NewDiscoveryClientForConfig(cfg)
		if err != nil {
			k.initErr = fmt.Errorf("discovery client: %w", err)
			return
		}
		k.disco = disco
		k.mapper = restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(disco))
	})
	return k.initErr
}

// formatCRDDescribe renders a concise, kubectl-like CRD summary.
func formatCRDDescribe(crd *apixv1.CustomResourceDefinition) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Name:        %s\n", crd.Name)
	fmt.Fprintf(&b, "Group:       %s\n", crd.Spec.Group)
	fmt.Fprintf(&b, "Kind:        %s\n", crd.Spec.Names.Kind)
	fmt.Fprintf(&b, "Plural:      %s\n", crd.Spec.Names.Plural)
	fmt.Fprintf(&b, "Scope:       %s\n", crd.Spec.Scope)
	fmt.Fprintf(&b, "Versions:\n")
	for _, v := range crd.Spec.Versions {
		fmt.Fprintf(&b, "  - %s (served=%t, storage=%t)\n", v.Name, v.Served, v.Storage)
		if len(v.AdditionalPrinterColumns) > 0 {
			fmt.Fprintf(&b, "    AdditionalPrinterColumns:\n")
			for _, c := range v.AdditionalPrinterColumns {
				fmt.Fprintf(&b, "      - %s (%s) %s\n", c.Name, c.Type, c.JSONPath)
			}
		}
	}
	return b.String()
}
