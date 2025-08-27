package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
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
				JSONRPC: "2.0",
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
				JSONRPC: "2.0",
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
				replyErr(req.ID, -32602, "Missing tool name")
				continue
			}

			// friendly readiness check (no auto-login)
			if err := s.K.Preflight(context.Background()); err != nil {
				replyErr(req.ID, -32603, err.Error())
				continue
			}

			switch name {
			case "datum_list_crds":
				res, err := s.ListCRDs(context.Background())
				if err != nil {
					replyErr(req.ID, -32603, err.Error())
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
					replyErr(req.ID, -32603, err.Error())
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

			default:
				replyErr(req.ID, -32601, fmt.Sprintf("Unknown tool %s", name))
			}
			continue

		default:
			if ignored[req.Method] {
				if req.ID != nil {
					root := strings.SplitN(req.Method, "/", 2)[0]
					reply(jsonrpcResp{
						JSONRPC: "2.0",
						ID:      req.ID,
						Result:  map[string]any{root: []any{}},
					})
				}
				continue
			}
			if req.ID != nil {
				replyErr(req.ID, -32601, "Unknown method "+req.Method)
			}
		}
	}
}

func notify(method string, params map[string]any) {
	emit(jsonrpcResp{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	})
}

func reply(resp jsonrpcResp) { emit(resp) }

func replyErr(id any, code int, msg string) {
	reply(jsonrpcResp{
		JSONRPC: "2.0",
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
		JSONRPC: "2.0",
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
			"description": "Validate a manifest with kubectl server-side dry-run (strict).",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"yaml": map[string]any{"type": "string"},
				},
				"required": []any{"yaml"},
			},
		},
	}
}
