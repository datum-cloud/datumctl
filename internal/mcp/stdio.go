package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"go.datum.net/datumctl/internal/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

// JSON-RPC constants and types
const (
	jsonrpcVersion        = "2.0"
	JSONRPCMethodNotFound = -32601
	JSONRPCInvalidParams  = -32602
	JSONRPCInternalError  = -32603
)

type jsonrpcReq struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      any            `json:"id,omitempty"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params,omitempty"`
}

type jsonrpcResp struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      any            `json:"id,omitempty"`
	Result  any            `json:"result,omitempty"`
	Error   *jsonrpcError  `json:"error,omitempty"`
	Method  string         `json:"method,omitempty"` // for notifications
	Params  map[string]any `json:"params,omitempty"` // for notifications
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

var ignored = map[string]bool{
	"resources/list":            true,
	"prompts/list":              true,
	"notifications/cancelled":   true,
	"notifications/initialized": true,
}

func (s *Service) RunSTDIO(port int) {
	fmt.Fprintf(os.Stderr, "[datum-mcp] STDIO mode ready\n")
	// Optional HTTP for manual testing.
	if port > 0 {
		go func() {
			if err := ServeHTTP(s, port); err != nil {
				fmt.Fprintf(os.Stderr, "[datum-mcp] HTTP server error: %v\n", err)
			}
		}()
	}

	sc := bufio.NewScanner(os.Stdin)
	buf := make([]byte, 0, 1024*1024)
	sc.Buffer(buf, 10*1024*1024)

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var req jsonrpcReq
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			continue
		}
		switch req.Method {
		case "initialize":
			reply(jsonrpcResp{
				JSONRPC: jsonrpcVersion,
				ID:      req.ID,
				Result: map[string]any{
					"protocolVersion": "2025-06-18",
					"serverInfo": map[string]any{
						"name":    "datum-mcp",
						"version": "3.0.0",
					},
					"capabilities": map[string]any{},
				},
			})
			notify("notifications/initialized", map[string]any{})
			continue

		case "tools/list":
			reply(jsonrpcResp{
				JSONRPC: jsonrpcVersion,
				ID:      req.ID,
				Result: map[string]any{
					"tools": toolsList(),
				},
			})
			continue

		case "tools/call":
			name, _ := req.Params["name"].(string)
			args, _ := req.Params["arguments"].(map[string]any)
			if name == "" {
				replyErr(req.ID, JSONRPCInvalidParams, "Missing tool name")
				continue
			}

			// friendly readiness check (no auto-login)
			if err := s.K.Preflight(context.Background()); err != nil {
				replyErr(req.ID, JSONRPCInternalError, err.Error())
				continue
			}

			switch name {
			case "datum_list_crds":
				res, err := s.ListCRDs(context.Background())
				if err != nil {
					replyErr(req.ID, JSONRPCInternalError, err.Error())
					continue
				}
				replyToolOK(req.ID, res)

			case "datum_get_crd":
				var r GetCRDReq
				if args != nil {
					r.Name, _ = args["name"].(string)
					r.Mode, _ = args["mode"].(string)
				}
				res, err := s.GetCRD(context.Background(), r)
				if err != nil {
					replyErr(req.ID, JSONRPCInternalError, err.Error())
					continue
				}
				replyToolOK(req.ID, res)

			case "datum_validate_yaml":
				var r ValidateReq
				if args != nil {
					r.YAML, _ = args["yaml"].(string)
				}
				res := s.ValidateYAML(context.Background(), r)
				replyToolOK(req.ID, res)

			case "datum_change_context":
				var project, org, ns string
				if args != nil {
					project, _ = args["project"].(string)
					org, _ = args["org"].(string)
					ns, _ = args["namespace"].(string)
				}
				// namespace-only change
				if project == "" && org == "" && ns != "" {
					s.K.Namespace = ns
					replyToolOK(req.ID, map[string]any{"ok": true, "namespace": s.K.Namespace})
					continue
				}
				if (project == "") == (org == "") {
					replyErr(req.ID, JSONRPCInvalidParams, "exactly one of project or org is required")
					continue
				}
				var (
					nc  *client.K8sClient
					err error
				)
				if project != "" {
					nc, err = client.NewForProject(context.Background(), project, firstNonEmpty(ns, s.K.Namespace))
				} else {
					nc, err = client.NewForOrg(context.Background(), org, firstNonEmpty(ns, s.K.Namespace))
				}
				if err != nil {
					replyErr(req.ID, JSONRPCInternalError, err.Error())
					continue
				}
				if err := nc.Preflight(context.Background()); err != nil {
					replyErr(req.ID, JSONRPCInternalError, err.Error())
					continue
				}
				s.K = nc
				replyToolOK(req.ID, map[string]any{"ok": true, "project": project, "org": org, "namespace": s.K.Namespace})

			case "datum_create_resource":
				kind, _ := getString(args, "kind")
				if kind == "" {
					replyErr(req.ID, JSONRPCInvalidParams, "kind is required")
					continue
				}
				apiVersion, _ := getString(args, "apiVersion")
				name, _ := getString(args, "name")
				genName, _ := getString(args, "generateName")
				ns := firstNonEmpty(getStringDefault(args, "namespace"), s.K.Namespace)
				spec := getMapAny(args, "spec")
				labels := getMapString(args, "labels")
				ann := getMapString(args, "annotations")
				dry := getBoolDefault(args, "dryRun", true)

				obj, err := s.K.Create(context.Background(), client.CreateOptions{
					Kind:         kind,
					APIVersion:   apiVersion,
					Name:         name,
					GenerateName: genName,
					Namespace:    ns,
					Labels:       labels,
					Annotations:  ann,
					Spec:         spec,
					DryRun:       dry,
				})
				if err != nil {
					replyErr(req.ID, JSONRPCInternalError, err.Error())
					continue
				}
				if dry {
					replyToolOK(req.ID, map[string]any{"ok": true, "dryRun": true, "validated": true})
				} else {
					replyToolOK(req.ID, map[string]any{"ok": true, "resourceVersion": obj.GetResourceVersion()})
				}

			case "datum_get_resource":
				kind, _ := getString(args, "kind")
				name, _ := getString(args, "name")
				if kind == "" || name == "" {
					replyErr(req.ID, JSONRPCInvalidParams, "kind and name are required")
					continue
				}
				apiVersion, _ := getString(args, "apiVersion")
				ns := firstNonEmpty(getStringDefault(args, "namespace"), s.K.Namespace)
				format := strings.ToLower(firstNonEmpty(getStringDefault(args, "format"), "yaml"))

				obj, err := s.K.Get(context.Background(), client.GetOptions{
					Kind:       kind,
					APIVersion: apiVersion,
					Name:       name,
					Namespace:  ns,
				})
				if err != nil {
					replyErr(req.ID, JSONRPCInternalError, err.Error())
					continue
				}
				if format == "json" {
					jb, _ := json.MarshalIndent(obj.Object, "", "  ")
					replyToolOK(req.ID, string(jb))
				} else {
					yb, err := client.ToYAML(obj)
					if err != nil {
						replyErr(req.ID, JSONRPCInternalError, err.Error())
						continue
					}
					replyToolOK(req.ID, yb)
				}

			case "datum_update_resource":
				kind, _ := getString(args, "kind")
				name, _ := getString(args, "name")
				if kind == "" || name == "" {
					replyErr(req.ID, JSONRPCInvalidParams, "kind and name are required")
					continue
				}
				apiVersion, _ := getString(args, "apiVersion")
				ns := firstNonEmpty(getStringDefault(args, "namespace"), s.K.Namespace)
				spec := getMapAny(args, "spec")
				labels := getMapString(args, "labels")
				ann := getMapString(args, "annotations")
				dry := getBoolDefault(args, "dryRun", true)
				force := getBoolDefault(args, "force", false)

				// For partial updates, we need to fetch the current resource first
				// and merge the changes rather than replacing the entire spec
				var currentObj *unstructured.Unstructured
				var err error

				// First, try to get the current resource to determine the correct API version
				// if not provided, we'll discover it from the existing resource
				if apiVersion == "" {
					// Try to find the resource without specifying apiVersion first
					// This will help us discover the correct API version
					currentObj, err = s.K.Get(context.Background(), client.GetOptions{
						Kind:      kind,
						Name:      name,
						Namespace: ns,
					})
					if err != nil {
						// If that fails, try to resolve the kind to get available API versions
						rk, resolveErr := s.K.ResolveKind(context.Background(), kind, "")
						if resolveErr != nil {
							replyErr(req.ID, JSONRPCInternalError, fmt.Sprintf("failed to resolve kind %s: %v", kind, resolveErr))
							continue
						}
						apiVersion = rk.GVK.GroupVersion().String()
					} else {
						// Use the API version from the existing resource
						apiVersion = currentObj.GetAPIVersion()
					}
				} else {
					// API version was provided, fetch the resource
					currentObj, err = s.K.Get(context.Background(), client.GetOptions{
						Kind:       kind,
						APIVersion: apiVersion,
						Name:       name,
						Namespace:  ns,
					})
					if err != nil {
						replyErr(req.ID, JSONRPCInternalError, fmt.Sprintf("failed to get current resource: %v", err))
						continue
					}
				}

				// Build the update object with merged changes
				var updateObj *unstructured.Unstructured
				if currentObj != nil {
					// Start with current object and merge changes
					updateObj = currentObj.DeepCopy()

					// Merge spec changes
					if spec != nil {
						currentSpec, _, _ := unstructured.NestedMap(updateObj.Object, "spec")
						if currentSpec == nil {
							currentSpec = make(map[string]any)
						}
						mergedSpec := mergeMaps(currentSpec, spec)
						unstructured.SetNestedMap(updateObj.Object, mergedSpec, "spec")
					}

					// Merge labels
					if labels != nil {
						currentLabels := updateObj.GetLabels()
						if currentLabels == nil {
							currentLabels = make(map[string]string)
						}
						for k, v := range labels {
							currentLabels[k] = v
						}
						updateObj.SetLabels(currentLabels)
					}

					// Merge annotations
					if ann != nil {
						currentAnn := updateObj.GetAnnotations()
						if currentAnn == nil {
							currentAnn = make(map[string]string)
						}
						for k, v := range ann {
							currentAnn[k] = v
						}
						updateObj.SetAnnotations(currentAnn)
					}
				} else {
					// No current resource found, this is an error for update operations
					replyErr(req.ID, JSONRPCInternalError, fmt.Sprintf("resource %s/%s not found in namespace %s", kind, name, ns))
					continue
				}

				// Apply the merged object using Server-Side Apply
				jb, _ := json.Marshal(updateObj.Object)
				rk, err := s.K.ResolveKind(context.Background(), kind, apiVersion)
				if err != nil {
					replyErr(req.ID, JSONRPCInternalError, err.Error())
					continue
				}

				nsable := s.K.GetDynamicClient().Resource(rk.GVR)
				var ri dynamic.ResourceInterface
				if rk.Namespaced {
					ri = nsable.Namespace(updateObj.GetNamespace())
				} else {
					ri = nsable
				}

				obj, err := ri.Patch(context.Background(), name, types.ApplyPatchType, jb, metav1.PatchOptions{
					FieldManager: "datumctl-mcp",
					Force:        &force,
					DryRun:       ternary(dry, []string{metav1.DryRunAll}, nil),
				})
				if err != nil {
					replyErr(req.ID, JSONRPCInternalError, err.Error())
					continue
				}
				if dry {
					replyToolOK(req.ID, map[string]any{"ok": true, "dryRun": true, "validated": true})
				} else {
					replyToolOK(req.ID, map[string]any{"ok": true, "resourceVersion": obj.GetResourceVersion()})
				}

			case "datum_delete_resource":
				kind, _ := getString(args, "kind")
				name, _ := getString(args, "name")
				if kind == "" || name == "" {
					replyErr(req.ID, JSONRPCInvalidParams, "kind and name are required")
					continue
				}
				apiVersion, _ := getString(args, "apiVersion")
				ns := firstNonEmpty(getStringDefault(args, "namespace"), s.K.Namespace)
				dry := getBoolDefault(args, "dryRun", true)

				if err := s.K.Delete(context.Background(), client.DeleteOptions{
					Kind:       kind,
					APIVersion: apiVersion,
					Name:       name,
					Namespace:  ns,
					DryRun:     dry,
				}); err != nil {
					replyErr(req.ID, JSONRPCInternalError, err.Error())
					continue
				}
				if dry {
					replyToolOK(req.ID, map[string]any{"ok": true, "dryRun": true, "validated": true})
				} else {
					replyToolOK(req.ID, map[string]any{"ok": true})
				}

			case "datum_list_resources":
				kind, _ := getString(args, "kind")
				if kind == "" {
					replyErr(req.ID, JSONRPCInvalidParams, "kind is required")
					continue
				}
				apiVersion := getStringDefault(args, "apiVersion")
				ns := firstNonEmpty(getStringDefault(args, "namespace"), s.K.Namespace)
				lbl := getStringDefault(args, "labelSelector")
				fld := getStringDefault(args, "fieldSelector")
				var limPtr *int64
				if v, ok := args["limit"].(float64); ok {
					v64 := int64(v)
					limPtr = &v64
				}
				cont := getStringDefault(args, "continue")
				format := getStringDefault(args, "format")

				res, err := s.ListResources(context.Background(), ListResourcesReq{
					Kind:          kind,
					APIVersion:    apiVersion,
					Namespace:     ns,
					LabelSelector: lbl,
					FieldSelector: fld,
					Limit:         limPtr,
					Continue:      cont,
					Format:        format,
				})
				if err != nil {
					replyErr(req.ID, JSONRPCInternalError, err.Error())
					continue
				}
				replyToolOK(req.ID, res)

			default:
				replyErr(req.ID, JSONRPCMethodNotFound, fmt.Sprintf("Unknown tool %s", name))
			}
			continue

		default:
			if ignored[req.Method] {
				if req.ID != nil {
					root := strings.SplitN(req.Method, "/", 2)[0]
					reply(jsonrpcResp{
						JSONRPC: jsonrpcVersion,
						ID:      req.ID,
						Result:  map[string]any{root: []any{}},
					})
				}
				continue
			}
			if req.ID != nil {
				replyErr(req.ID, JSONRPCMethodNotFound, "Unknown method "+req.Method)
			}
		}
	}
}

func notify(method string, params map[string]any) {
	emit(jsonrpcResp{
		JSONRPC: jsonrpcVersion,
		Method:  method,
		Params:  params,
	})
}

func reply(resp jsonrpcResp) { emit(resp) }

func replyErr(id any, code int, msg string) {
	reply(jsonrpcResp{
		JSONRPC: jsonrpcVersion,
		ID:      id,
		Error:   &jsonrpcError{Code: code, Message: msg},
	})
}

func replyToolOK(id any, payload any) {
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		b, _ = json.Marshal(payload)
	}
	reply(jsonrpcResp{
		JSONRPC: jsonrpcVersion,
		ID:      id,
		Result: map[string]any{
			"content": []any{
				map[string]any{
					"type": "text",
					"text": string(b),
				},
			},
		},
	})
}

func emit(resp jsonrpcResp) {
	enc := json.NewEncoder(os.Stdout)
	_ = enc.Encode(resp)
}

func toolsList() []map[string]any {
	return []map[string]any{
		{
			"name":        "datum_list_crds",
			"description": "List CustomResourceDefinitions in the current cluster.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}, "required": []any{}},
		},
		{
			"name":        "datum_get_crd",
			"description": "Get or describe a CRD by name. Mode: yaml|json|describe (default yaml).",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
					"mode": map[string]any{"type": "string"},
				},
				"required": []any{"name"},
			},
		},
		{
			"name":        "datum_validate_yaml",
			"description": "Validate a manifest with server-side dry-run via the Kubernetes API.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"yaml": map[string]any{"type": "string"},
				},
				"required": []any{"yaml"},
			},
		},
		{
			"name":        "datum_change_context",
			"description": "Switch project/org/namespace for this MCP session.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project":   map[string]any{"type": "string"},
					"org":       map[string]any{"type": "string"},
					"namespace": map[string]any{"type": "string"},
				},
			},
		},
		{
			"name":        "datum_create_resource",
			"description": "Create a resource (server-side dry-run by default).",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"kind":         map[string]any{"type": "string"},
					"apiVersion":   map[string]any{"type": "string"},
					"name":         map[string]any{"type": "string"},
					"generateName": map[string]any{"type": "string"},
					"namespace":    map[string]any{"type": "string"},
					"spec":         map[string]any{"type": "object"},
					"labels":       map[string]any{"type": "object"},
					"annotations":  map[string]any{"type": "object"},
					"dryRun":       map[string]any{"type": "boolean"},
				},
				"required": []any{"kind"},
			},
		},
		{
			"name":        "datum_get_resource",
			"description": "Get a resource by name (yaml by default).",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"kind":       map[string]any{"type": "string"},
					"apiVersion": map[string]any{"type": "string"},
					"name":       map[string]any{"type": "string"},
					"namespace":  map[string]any{"type": "string"},
					"format":     map[string]any{"type": "string"},
				},
				"required": []any{"kind", "name"},
			},
		},
		{
			"name":        "datum_update_resource",
			"description": "Update a resource using Server-Side Apply.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"kind":        map[string]any{"type": "string"},
					"apiVersion":  map[string]any{"type": "string"},
					"name":        map[string]any{"type": "string"},
					"namespace":   map[string]any{"type": "string"},
					"spec":        map[string]any{"type": "object"},
					"labels":      map[string]any{"type": "object"},
					"annotations": map[string]any{"type": "object"},
					"dryRun":      map[string]any{"type": "boolean"},
					"force":       map[string]any{"type": "boolean"},
				},
				"required": []any{"kind", "name"},
			},
		},
		{
			"name":        "datum_delete_resource",
			"description": "Delete a resource (server-side dry-run by default).",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"kind":       map[string]any{"type": "string"},
					"apiVersion": map[string]any{"type": "string"},
					"name":       map[string]any{"type": "string"},
					"namespace":  map[string]any{"type": "string"},
					"dryRun":     map[string]any{"type": "boolean"},
				},
				"required": []any{"kind", "name"},
			},
		},
		{
			"name":        "datum_list_resources",
			"description": "List instances of a Kind (optionally filter by namespace/labels).",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"kind":          map[string]any{"type": "string"},
					"apiVersion":    map[string]any{"type": "string"},
					"namespace":     map[string]any{"type": "string"},
					"labelSelector": map[string]any{"type": "string"},
					"fieldSelector": map[string]any{"type": "string"},
					"limit":         map[string]any{"type": "number"},
					"continue":      map[string]any{"type": "string"},
					"format":        map[string]any{"type": "string"}, // yaml|names
				},
				"required": []any{"kind"},
			},
		},
	}
}

// -------- helpers for arg parsing --------

func getString(m map[string]any, k string) (string, bool) {
	if m == nil {
		return "", false
	}
	v, ok := m[k].(string)
	return v, ok
}
func getStringDefault(m map[string]any, k string) string {
	v, _ := getString(m, k)
	return v
}
func getBoolDefault(m map[string]any, k string, def bool) bool {
	if m == nil {
		return def
	}
	if v, ok := m[k].(bool); ok {
		return v
	}
	return def
}
func getMapAny(m map[string]any, k string) map[string]any {
	if m == nil {
		return nil
	}
	if v, ok := m[k].(map[string]any); ok {
		return v
	}
	// tolerate map[string]interface{}
	if v, ok := m[k].(map[string]interface{}); ok {
		out := make(map[string]any, len(v))
		for kk, vv := range v {
			out[kk] = vv
		}
		return out
	}
	return nil
}
func getMapString(m map[string]any, k string) map[string]string {
	if m == nil {
		return nil
	}
	// try direct
	if vs, ok := m[k].(map[string]string); ok {
		return vs
	}
	// convert from map[string]any
	if va, ok := m[k].(map[string]any); ok {
		out := make(map[string]string, len(va))
		for kk, vv := range va {
			if s, ok := vv.(string); ok {
				out[kk] = s
			} else {
				out[kk] = fmt.Sprint(vv)
			}
		}
		return out
	}
	// convert from map[string]interface{}
	if vi, ok := m[k].(map[string]interface{}); ok {
		out := make(map[string]string, len(vi))
		for kk, vv := range vi {
			if s, ok := vv.(string); ok {
				out[kk] = s
			} else {
				out[kk] = fmt.Sprint(vv)
			}
		}
		return out
	}
	return nil
}

// mergeMaps recursively merges map b into map a
func mergeMaps(a, b map[string]any) map[string]any {
	if a == nil {
		a = make(map[string]any)
	}
	for k, v := range b {
		if existing, exists := a[k]; exists {
			// If both values are maps, merge them recursively
			if existingMap, ok := existing.(map[string]any); ok {
				if newMap, ok := v.(map[string]any); ok {
					a[k] = mergeMaps(existingMap, newMap)
					continue
				}
			}
		}
		// Otherwise, replace the value
		a[k] = v
	}
	return a
}

// buildObjectOpts represents options for building a Kubernetes object
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

// buildObject creates an unstructured object from buildObjectOpts
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

// ternary returns a if cond is true, otherwise b
func ternary[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}
