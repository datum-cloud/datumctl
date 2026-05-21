package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	syaml "sigs.k8s.io/yaml"

	"go.datum.net/datumctl/internal/ai/llm"
	"go.datum.net/datumctl/internal/client"
)

// Tool is a single capability exposed to the LLM.
type Tool struct {
	Def             llm.ToolDef
	RequiresConfirm bool
	Execute         func(ctx context.Context, args map[string]any) (string, error)
}

// Registry holds the set of tools available in an agentic session.
type Registry struct {
	tools []Tool
}

// NewRegistry builds the full tool set backed by a DatumCloudFactory.
func NewRegistry(factory *client.DatumCloudFactory) *Registry {
	r := &Registry{}

	r.add(Tool{
		Def: llm.ToolDef{
			Name:        "list_resource_types",
			Description: "List all Datum Cloud resource types available in the current context.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		Execute: func(ctx context.Context, _ map[string]any) (string, error) {
			dc, err := factory.ToDiscoveryClient()
			if err != nil {
				return "", fmt.Errorf("discovery client: %w", err)
			}
			lists, err := dc.ServerPreferredResources()
			if err != nil && lists == nil {
				return "", fmt.Errorf("list resource types: %w", err)
			}
			type item struct {
				Name       string `json:"name"`
				Kind       string `json:"kind"`
				Group      string `json:"group"`
				Version    string `json:"version"`
				Namespaced bool   `json:"namespaced"`
			}
			var items []item
			skipGroups := map[string]bool{
				"events.k8s.io": true, "authentication.k8s.io": true,
				"authorization.k8s.io": true, "coordination.k8s.io": true,
			}
			for _, list := range lists {
				gv, err := schema.ParseGroupVersion(list.GroupVersion)
				if err != nil || skipGroups[gv.Group] {
					continue
				}
				for _, res := range list.APIResources {
					items = append(items, item{
						Name: res.Name, Kind: res.Kind,
						Group: gv.Group, Version: gv.Version,
						Namespaced: res.Namespaced,
					})
				}
			}
			b, _ := json.Marshal(map[string]any{"items": items})
			return string(b), nil
		},
	})

	r.add(Tool{
		Def: llm.ToolDef{
			Name:        "get_resource_schema",
			Description: "Get the OpenAPI schema for a Datum Cloud resource type. Pass the CRD name (e.g. dnszones.networking.datumapis.com) or kind name.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type":        "string",
						"description": "CRD name (e.g. dnszones.networking.datumapis.com) or kind (e.g. DNSZone)",
					},
				},
				"required": []string{"name"},
			},
		},
		Execute: func(ctx context.Context, args map[string]any) (string, error) {
			name := stringArg(args, "name")
			dc, err := factory.ToDiscoveryClient()
			if err != nil {
				return "", fmt.Errorf("discovery client: %w", err)
			}
			// Resolve group from CRD name (plural.group format) or kind.
			group := ""
			lists, _ := dc.ServerPreferredResources()
			for _, list := range lists {
				gv, err := schema.ParseGroupVersion(list.GroupVersion)
				if err != nil {
					continue
				}
				for _, res := range list.APIResources {
					if res.Name == name || res.Kind == name ||
						res.Name+"."+gv.Group == name {
						group = gv.Group
						break
					}
				}
				if group != "" {
					break
				}
			}
			if group == "" {
				// Try treating name as "plural.group"
				if idx := strings.Index(name, "."); idx > 0 {
					group = name[idx+1:]
				}
			}
			if group == "" {
				return fmt.Sprintf(`{"error":"resource type %q not found","hint":"use list_resource_types to see available types"}`, name), nil
			}
			schema, err := dc.OpenAPIV3().Paths()
			if err != nil {
				return fmt.Sprintf(`{"error":"schema not available","group":"%s"}`, group), nil
			}
			for p, gv := range schema {
				if strings.Contains(p, group) {
					data, err := gv.Schema("application/json")
					if err != nil {
						continue
					}
					return string(data), nil
				}
			}
			return fmt.Sprintf(`{"error":"schema not available","hint":"use list_resources to inspect existing resources and infer field structure","group":"%s"}`, group), nil
		},
	})

	r.add(Tool{
		Def: llm.ToolDef{
			Name:        "list_resources",
			Description: "List Datum Cloud resources of a given type.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"kind": map[string]any{
						"type":        "string",
						"description": "Resource kind, e.g. DNSZone",
					},
					"apiVersion": map[string]any{
						"type":        "string",
						"description": "API version, e.g. networking.datumapis.com/v1alpha (optional)",
					},
					"namespace": map[string]any{
						"type":        "string",
						"description": "Namespace (optional)",
					},
					"labelSelector": map[string]any{
						"type":        "string",
						"description": "Label selector to filter results (optional)",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum number of results (optional)",
					},
				},
				"required": []string{"kind"},
			},
		},
		Execute: func(ctx context.Context, args map[string]any) (string, error) {
			kind := stringArg(args, "kind")
			apiVersion := stringArg(args, "apiVersion")
			namespace := stringArg(args, "namespace")
			labelSelector := stringArg(args, "labelSelector")
			var limit int64
			if v, ok := args["limit"]; ok {
				switch n := v.(type) {
				case float64:
					limit = int64(n)
				case int64:
					limit = n
				}
			}

			gvr, namespaced, err := resolveGVR(factory, kind, apiVersion)
			if err != nil {
				return "", err
			}

			dc, err := factory.DynamicClient()
			if err != nil {
				return "", fmt.Errorf("dynamic client: %w", err)
			}

			opts := metav1.ListOptions{LabelSelector: labelSelector}
			if limit > 0 {
				opts.Limit = limit
			}

			// Fall back to the session namespace rather than NamespaceAll so the
			// API doesn't need to support cross-namespace listing.
			if namespaced && namespace == "" && factory.ConfigFlags.Namespace != nil && *factory.ConfigFlags.Namespace != "" {
				namespace = *factory.ConfigFlags.Namespace
			}

			var lst *unstructured.UnstructuredList
			if namespaced && namespace != "" {
				lst, err = dc.Resource(gvr).Namespace(namespace).List(ctx, opts)
			} else if namespaced {
				lst, err = dc.Resource(gvr).Namespace(metav1.NamespaceAll).List(ctx, opts)
			} else {
				lst, err = dc.Resource(gvr).List(ctx, opts)
			}

			if err != nil {
				// Only mask errors that mean the resource type itself doesn't exist.
				// Propagate everything else (namespace not found, access denied, etc.)
				// so the AI can surface the real problem to the user.
				if strings.Contains(err.Error(), "no matches for kind") {
					return "No resources found", nil
				}
				return "", err
			}

			if len(lst.Items) == 0 {
				return fmt.Sprintf("No %s resources found in namespace %q", gvr.Resource, namespace), nil
			}

			b, err := syaml.Marshal(lst)
			if err != nil {
				return "", err
			}
			return string(b), nil
		},
	})

	r.add(Tool{
		Def: llm.ToolDef{
			Name:        "get_resource",
			Description: "Get a single Datum Cloud resource by kind and name.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"kind": map[string]any{
						"type":        "string",
						"description": "Resource kind, e.g. DNSZone",
					},
					"name": map[string]any{
						"type":        "string",
						"description": "Resource name",
					},
					"apiVersion": map[string]any{
						"type":        "string",
						"description": "API version (optional)",
					},
					"namespace": map[string]any{
						"type":        "string",
						"description": "Namespace (optional)",
					},
					"format": map[string]any{
						"type":        "string",
						"description": "Output format: yaml (default) or json",
						"enum":        []string{"yaml", "json"},
					},
				},
				"required": []string{"kind", "name"},
			},
		},
		Execute: func(ctx context.Context, args map[string]any) (string, error) {
			kind := stringArg(args, "kind")
			name := stringArg(args, "name")
			apiVersion := stringArg(args, "apiVersion")
			namespace := stringArg(args, "namespace")
			format := stringArg(args, "format")

			gvr, namespaced, err := resolveGVR(factory, kind, apiVersion)
			if err != nil {
				return "", err
			}

			dc, err := factory.DynamicClient()
			if err != nil {
				return "", fmt.Errorf("dynamic client: %w", err)
			}

			if namespaced && namespace == "" && factory.ConfigFlags.Namespace != nil && *factory.ConfigFlags.Namespace != "" {
				namespace = *factory.ConfigFlags.Namespace
			}

			var obj *unstructured.Unstructured
			if namespaced && namespace != "" {
				obj, err = dc.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
			} else {
				obj, err = dc.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
			}
			if err != nil {
				return "", err
			}

			if strings.ToLower(format) == "json" {
				b, _ := json.MarshalIndent(obj.Object, "", "  ")
				return string(b), nil
			}
			b, err := syaml.Marshal(obj.Object)
			if err != nil {
				return "", err
			}
			return string(b), nil
		},
	})

	r.add(Tool{
		Def: llm.ToolDef{
			Name:        "validate_manifest",
			Description: "Validate a resource manifest via server-side dry run without persisting changes.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"yaml": map[string]any{
						"type":        "string",
						"description": "YAML manifest to validate",
					},
				},
				"required": []string{"yaml"},
			},
		},
		Execute: func(ctx context.Context, args map[string]any) (string, error) {
			rawYAML := stringArg(args, "yaml")
			obj, err := parseYAMLManifest(rawYAML)
			if err != nil {
				return "", fmt.Errorf("parse manifest: %w", err)
			}

			gvr, namespaced, err := resolveGVR(factory, obj.GetKind(), obj.GetAPIVersion())
			if err != nil {
				return "", err
			}

			dc, err := factory.DynamicClient()
			if err != nil {
				return "", fmt.Errorf("dynamic client: %w", err)
			}

			applyOpts := metav1.ApplyOptions{FieldManager: "datumctl-ai", DryRun: []string{metav1.DryRunAll}}
			var applyErr error
			if namespaced && obj.GetNamespace() != "" {
				_, applyErr = dc.Resource(gvr).Namespace(obj.GetNamespace()).Apply(ctx, obj.GetName(), obj, applyOpts)
			} else {
				_, applyErr = dc.Resource(gvr).Apply(ctx, obj.GetName(), obj, applyOpts)
			}

			type result struct {
				Valid  bool   `json:"valid"`
				Output string `json:"output,omitempty"`
			}
			if applyErr != nil {
				b, _ := json.Marshal(result{Valid: false, Output: applyErr.Error()})
				return string(b), nil
			}
			b, _ := json.Marshal(result{Valid: true})
			return string(b), nil
		},
	})

	r.add(Tool{
		Def: llm.ToolDef{
			Name:        "apply_manifest",
			Description: "Create or update a Datum Cloud resource by applying a YAML manifest (server-side apply / upsert). Requires confirmation.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"yaml": map[string]any{
						"type":        "string",
						"description": "YAML manifest describing the desired resource state",
					},
					"dryRun": map[string]any{
						"type":        "boolean",
						"description": "If true, validate only without persisting (default false)",
					},
				},
				"required": []string{"yaml"},
			},
		},
		RequiresConfirm: true,
		Execute: func(ctx context.Context, args map[string]any) (string, error) {
			rawYAML := stringArg(args, "yaml")
			dryRun, _ := args["dryRun"].(bool)

			obj, err := parseYAMLManifest(rawYAML)
			if err != nil {
				return "", fmt.Errorf("parse manifest: %w", err)
			}

			gvr, namespaced, err := resolveGVR(factory, obj.GetKind(), obj.GetAPIVersion())
			if err != nil {
				return "", err
			}

			dc, err := factory.DynamicClient()
			if err != nil {
				return "", fmt.Errorf("dynamic client: %w", err)
			}

			applyOpts := metav1.ApplyOptions{FieldManager: "datumctl-ai", Force: true}
			if dryRun {
				applyOpts.DryRun = []string{metav1.DryRunAll}
			}

			var result *unstructured.Unstructured
			if namespaced && obj.GetNamespace() != "" {
				result, err = dc.Resource(gvr).Namespace(obj.GetNamespace()).Apply(ctx, obj.GetName(), obj, applyOpts)
			} else {
				result, err = dc.Resource(gvr).Apply(ctx, obj.GetName(), obj, applyOpts)
			}
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("applied %s/%s (resourceVersion: %s)",
				result.GetKind(), result.GetName(), result.GetResourceVersion()), nil
		},
	})

	r.add(Tool{
		Def: llm.ToolDef{
			Name:        "delete_resource",
			Description: "Delete a Datum Cloud resource by kind and name. Always requires confirmation.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"kind": map[string]any{
						"type":        "string",
						"description": "Resource kind, e.g. DNSZone",
					},
					"name": map[string]any{
						"type":        "string",
						"description": "Resource name",
					},
					"apiVersion": map[string]any{
						"type":        "string",
						"description": "API version (optional)",
					},
					"namespace": map[string]any{
						"type":        "string",
						"description": "Namespace (optional)",
					},
				},
				"required": []string{"kind", "name"},
			},
		},
		RequiresConfirm: true,
		Execute: func(ctx context.Context, args map[string]any) (string, error) {
			kind := stringArg(args, "kind")
			name := stringArg(args, "name")
			apiVersion := stringArg(args, "apiVersion")
			namespace := stringArg(args, "namespace")

			gvr, namespaced, err := resolveGVR(factory, kind, apiVersion)
			if err != nil {
				return "", err
			}

			dc, err := factory.DynamicClient()
			if err != nil {
				return "", fmt.Errorf("dynamic client: %w", err)
			}

			policy := metav1.DeletePropagationBackground
			opts := metav1.DeleteOptions{PropagationPolicy: &policy}

			if namespaced && namespace != "" {
				err = dc.Resource(gvr).Namespace(namespace).Delete(ctx, name, opts)
			} else {
				err = dc.Resource(gvr).Delete(ctx, name, opts)
			}
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("deleted %s/%s", kind, name), nil
		},
	})

	r.add(Tool{
		Def: llm.ToolDef{
			Name:        "change_context",
			Description: "Switch the active organization, project, or namespace for the current session.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"org": map[string]any{
						"type":        "string",
						"description": "New organization ID (mutually exclusive with project)",
					},
					"project": map[string]any{
						"type":        "string",
						"description": "New project ID (mutually exclusive with org)",
					},
				},
			},
		},
		Execute: func(ctx context.Context, args map[string]any) (string, error) {
			org := stringArg(args, "org")
			project := stringArg(args, "project")

			if org != "" && project != "" {
				return "", fmt.Errorf("org and project are mutually exclusive")
			}
			if org == "" && project == "" {
				return "", fmt.Errorf("one of org or project is required")
			}

			if org != "" {
				*factory.ConfigFlags.Organization = org
				*factory.ConfigFlags.Project = ""
			} else {
				*factory.ConfigFlags.Project = project
				*factory.ConfigFlags.Organization = ""
			}

			// Invalidate the REST mapper cache so the next tool call uses the new context.
			factory.ConfigFlags.ToDiscoveryClient() //nolint:errcheck

			b, _ := json.Marshal(map[string]any{"ok": true, "org": org, "project": project})
			return string(b), nil
		},
	})

	return r
}

// Defs returns the ToolDef slice to pass to LLMClient.Chat.
func (r *Registry) Defs() []llm.ToolDef {
	defs := make([]llm.ToolDef, len(r.tools))
	for i, t := range r.tools {
		defs[i] = t.Def
	}
	return defs
}

// Find looks up a tool by name.
func (r *Registry) Find(name string) (*Tool, bool) {
	for i := range r.tools {
		if r.tools[i].Def.Name == name {
			return &r.tools[i], true
		}
	}
	return nil, false
}

func (r *Registry) add(t Tool) { r.tools = append(r.tools, t) }

// NewEmptyRegistry returns a Registry with no tools, used when no context is set.
func NewEmptyRegistry() *Registry { return &Registry{} }

// --- helpers ---

func stringArg(args map[string]any, key string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// resolveGVR resolves a kind and optional apiVersion to a GroupVersionResource
// using the factory's discovery client / REST mapper.
func resolveGVR(factory *client.DatumCloudFactory, kind, apiVersion string) (schema.GroupVersionResource, bool, error) {
	// Try REST mapper first — it handles group/version/kind lookup efficiently.
	mapper, err := factory.ToRESTMapper()
	if err != nil {
		return schema.GroupVersionResource{}, false, fmt.Errorf("REST mapper: %w", err)
	}

	var gk schema.GroupKind
	if apiVersion != "" {
		gv, err := schema.ParseGroupVersion(apiVersion)
		if err == nil {
			gk = schema.GroupKind{Group: gv.Group, Kind: kind}
		}
	}
	if gk.Kind == "" {
		gk = schema.GroupKind{Kind: kind}
	}

	mappings, err := mapper.RESTMappings(gk)
	if err != nil || len(mappings) == 0 {
		// Fall back to discovery search by kind name.
		return resolveGVRByDiscovery(factory, kind, apiVersion)
	}

	// Prefer the mapping whose version matches apiVersion if specified.
	m := mappings[0]
	if apiVersion != "" {
		gv, _ := schema.ParseGroupVersion(apiVersion)
		for _, candidate := range mappings {
			if candidate.Resource.Version == gv.Version {
				m = candidate
				break
			}
		}
	}
	namespaced := m.Scope.Name() == "namespace"
	return m.Resource, namespaced, nil
}

// resolveGVRByDiscovery falls back to iterating server-preferred resources.
func resolveGVRByDiscovery(factory *client.DatumCloudFactory, kind, apiVersion string) (schema.GroupVersionResource, bool, error) {
	dc, err := factory.ToDiscoveryClient()
	if err != nil {
		return schema.GroupVersionResource{}, false, fmt.Errorf("discovery client: %w", err)
	}
	lists, err := dc.ServerPreferredResources()
	if err != nil && lists == nil {
		return schema.GroupVersionResource{}, false, fmt.Errorf("server resources: %w", err)
	}
	for _, list := range lists {
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			continue
		}
		if apiVersion != "" {
			wantGV, wantErr := schema.ParseGroupVersion(apiVersion)
			if wantErr == nil && gv != wantGV {
				continue
			}
		}
		for _, res := range list.APIResources {
			if res.Kind == kind {
				gvr := schema.GroupVersionResource{
					Group:    gv.Group,
					Version:  gv.Version,
					Resource: res.Name,
				}
				return gvr, res.Namespaced, nil
			}
		}
	}
	return schema.GroupVersionResource{}, false, fmt.Errorf("resource kind %q not found; use list_resource_types to see available types", kind)
}

// parseYAMLManifest parses a raw YAML string into an unstructured.Unstructured object.
func parseYAMLManifest(rawYAML string) (*unstructured.Unstructured, error) {
	jsonBytes, err := syaml.YAMLToJSON([]byte(rawYAML))
	if err != nil {
		return nil, fmt.Errorf("yaml to json: %w", err)
	}
	var obj map[string]any
	if err := json.Unmarshal(jsonBytes, &obj); err != nil {
		return nil, fmt.Errorf("unmarshal manifest: %w", err)
	}
	return &unstructured.Unstructured{Object: obj}, nil
}
