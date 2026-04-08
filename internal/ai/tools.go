package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	syaml "sigs.k8s.io/yaml"

	"go.datum.net/datumctl/internal/ai/llm"
	"go.datum.net/datumctl/internal/client"
	mcpsvc "go.datum.net/datumctl/internal/mcp"
)

// Tool is a single capability exposed to the LLM. The Execute function is
// called when the LLM requests invocation; RequiresConfirm gates writes.
type Tool struct {
	Def             llm.ToolDef
	RequiresConfirm bool
	Execute         func(ctx context.Context, args map[string]any) (string, error)
}

// Registry holds the set of tools available in an agentic session.
type Registry struct {
	tools []Tool
}

// NewRegistry builds the full tool set by wrapping mcp.Service methods.
// apply_manifest calls svc.K.Apply directly since Service.K is exported.
func NewRegistry(svc *mcpsvc.Service) *Registry {
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
			resp, err := svc.ListCRDs(ctx)
			if err != nil {
				// Fall back to the discovery API when CRD access is forbidden.
				items, discErr := svc.K.DiscoverResourceTypes()
				if discErr != nil {
					return "", fmt.Errorf("list resource types (CRD forbidden: %v; discovery: %v)", err, discErr)
				}
				b, _ := json.Marshal(map[string]any{"items": items})
				return string(b), nil
			}
			b, _ := json.Marshal(resp)
			return string(b), nil
		},
	})

	r.add(Tool{
		Def: llm.ToolDef{
			Name:        "get_resource_schema",
			Description: "Get the full schema for a Datum Cloud resource type by CRD name.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type":        "string",
						"description": "CRD name, e.g. dnszones.networking.datumapis.com",
					},
					"mode": map[string]any{
						"type":        "string",
						"description": "Output format: yaml (default), json, or describe",
						"enum":        []string{"yaml", "json", "describe"},
					},
				},
				"required": []string{"name"},
			},
		},
		Execute: func(ctx context.Context, args map[string]any) (string, error) {
			name, _ := args["name"].(string)
			mode, _ := args["mode"].(string)
			resp, err := svc.GetCRD(ctx, mcpsvc.GetCRDReq{Name: name, Mode: mode})
			if err != nil {
				// CRD access is restricted — fall back to the OpenAPI V3 schema.
				// CRD names are in the form "plural.group", e.g. "dnszones.networking.datumapis.com"
				// Extract the group and try to find the API version from discovery.
				parts := strings.SplitN(name, ".", 2)
				if len(parts) == 2 {
					group := parts[1]
					// Find the version for this group via discovery.
					lists, _ := svc.K.DiscoverResourceTypes()
					version := ""
					for _, item := range lists {
						if g, _ := item["group"].(string); g == group {
							if vs, ok := item["versions"].([]string); ok && len(vs) > 0 {
								version = vs[0]
								break
							}
						}
					}
					if version != "" {
						schema, oaErr := svc.K.GetOpenAPISchema(group + "/" + version)
						if oaErr == nil {
							return schema, nil
						}
					}
				}
				return fmt.Sprintf(`{"error":"schema not available","hint":"CRD and OpenAPI access both failed. Use list_resources to inspect existing resources of this type and infer field structure from their spec.","crd":"%s"}`, name), nil
			}
			return resp.Text, nil
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
						"description": "Namespace (optional, uses session default if omitted)",
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
			req := mcpsvc.ListResourcesReq{
				Kind:          stringArg(args, "kind"),
				APIVersion:    stringArg(args, "apiVersion"),
				Namespace:     stringArg(args, "namespace"),
				LabelSelector: stringArg(args, "labelSelector"),
			}
			if v, ok := args["limit"]; ok {
				switch n := v.(type) {
				case float64:
					i := int64(n)
					req.Limit = &i
				case int64:
					req.Limit = &n
				}
			}
			resp, err := svc.ListResources(ctx, req)
			if err != nil {
				return "", err
			}
			return resp.Text, nil
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
						"description": "Namespace (optional, uses session default if omitted)",
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
			resp, err := svc.GetResource(ctx, mcpsvc.GetResourceReq{
				Kind:       stringArg(args, "kind"),
				Name:       stringArg(args, "name"),
				APIVersion: stringArg(args, "apiVersion"),
				Namespace:  stringArg(args, "namespace"),
				Format:     stringArg(args, "format"),
			})
			if err != nil {
				return "", err
			}
			return resp.Text, nil
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
			resp := svc.ValidateYAML(ctx, mcpsvc.ValidateReq{
				YAML: stringArg(args, "yaml"),
			})
			b, _ := json.Marshal(resp)
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

			spec, _, _ := unstructured.NestedMap(obj.Object, "spec")
			labels := obj.GetLabels()
			annotations := obj.GetAnnotations()

			result, err := svc.K.Apply(ctx, client.ApplyOptions{
				Kind:        obj.GetKind(),
				APIVersion:  obj.GetAPIVersion(),
				Name:        obj.GetName(),
				Namespace:   obj.GetNamespace(),
				Labels:      labels,
				Annotations: annotations,
				Spec:        spec,
				DryRun:      dryRun,
			})
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
						"description": "Namespace (optional, uses session default if omitted)",
					},
				},
				"required": []string{"kind", "name"},
			},
		},
		RequiresConfirm: true,
		Execute: func(ctx context.Context, args map[string]any) (string, error) {
			dryRun := false
			_, err := svc.DeleteResource(ctx, mcpsvc.DeleteResourceReq{
				Kind:       stringArg(args, "kind"),
				Name:       stringArg(args, "name"),
				APIVersion: stringArg(args, "apiVersion"),
				Namespace:  stringArg(args, "namespace"),
				DryRun:     &dryRun,
			})
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("deleted %s/%s", stringArg(args, "kind"), stringArg(args, "name")), nil
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
					"namespace": map[string]any{
						"type":        "string",
						"description": "New default namespace",
					},
				},
			},
		},
		Execute: func(ctx context.Context, args map[string]any) (string, error) {
			resp, err := svc.ChangeContext(ctx, mcpsvc.ChangeContextReq{
				Org:       stringArg(args, "org"),
				Project:   stringArg(args, "project"),
				Namespace: stringArg(args, "namespace"),
			})
			if err != nil {
				return "", err
			}
			b, _ := json.Marshal(resp)
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

// Find looks up a tool by name. Returns nil, false if not found.
func (r *Registry) Find(name string) (*Tool, bool) {
	for i := range r.tools {
		if r.tools[i].Def.Name == name {
			return &r.tools[i], true
		}
	}
	return nil, false
}

func (r *Registry) add(t Tool) { r.tools = append(r.tools, t) }

// NewEmptyRegistry returns a Registry with no tools, used when no org/project
// context is set and the agent is answering general questions only.
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

// parseYAMLManifest parses a raw YAML string into an unstructured.Unstructured object.
func parseYAMLManifest(rawYAML string) (*unstructured.Unstructured, error) {
	// sigs.k8s.io/yaml converts YAML to JSON first, then unmarshals.
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
