package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"go.datum.net/datumctl/internal/client"
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
			// ------ Phase-1 tools ------
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

			// ------ New tools (CRUD + context + list) ------
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

				obj, err := s.K.Apply(context.Background(), client.ApplyOptions{
					Kind:        kind,
					APIVersion:  apiVersion,
					Name:        name,
					Namespace:   ns,
					Labels:      labels,
					Annotations: ann,
					Spec:        spec,
					DryRun:      dry,
					Force:       force,
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
		// phase-1
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
		// new
		{
			"name":        "datum_change_context",
			"description": "Switch project/org/namespace for this MCP session.",
			"inputSchema": map[string]any{
				"type":       "object",
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
