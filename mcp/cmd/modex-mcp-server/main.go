package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"modex/mcp/internal/client"
)

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   any    `json:"error,omitempty"`
}

func main() {
	c := client.FromEnv()
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var req rpcRequest
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			respond(nil, nil, err)
			continue
		}
		result, err := handle(c, req)
		respond(req.ID, result, err)
	}
}

func handle(c client.Client, req rpcRequest) (any, error) {
	switch req.Method {
	case "initialize":
		return map[string]any{"protocolVersion": "2024-11-05", "serverInfo": map[string]any{"name": "modex-mcp-server", "version": "0.1.0"}, "capabilities": map[string]any{"tools": map[string]any{}}}, nil
	case "tools/list":
		return map[string]any{"tools": []map[string]any{
			tool("list_modules", "List documentation modules by category or keyword"),
			tool("list_versions", "List versions for one module"),
			tool("search_docs", "Search docs with keyword, semantic, or hybrid mode"),
			tool("get_doc_page", "Get one document page by doc_id"),
		}}, nil
	case "tools/call":
		var params struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, err
		}
		return callTool(c, params.Name, params.Arguments)
	default:
		return nil, fmt.Errorf("unsupported method %s", req.Method)
	}
}

func callTool(c client.Client, name string, args map[string]any) (any, error) {
	var result any
	var err error
	switch name {
	case "list_modules":
		result, err = c.ListModules(str(args["category_id"]), str(args["keyword"]))
	case "list_versions":
		result, err = c.ListVersions(str(args["module_key"]))
	case "search_docs":
		limit := intVal(args["limit"], 5)
		body := map[string]any{
			"query": args["query"],
			"mode":  defaultStr(str(args["mode"]), "hybrid"),
			"page":  1, "page_size": limit,
			"filters": map[string]any{
				"modules":       optionalList(str(args["module_key"])),
				"docs_versions": optionalList(str(args["docs_version"])),
			},
		}
		result, err = c.SearchDocs(body)
	case "get_doc_page":
		result, err = c.GetDocPage(str(args["doc_id"]))
	default:
		return nil, fmt.Errorf("unknown tool %s", name)
	}
	if err == nil {
		c.LogMCP(name, str(args["query"]), args, resultCount(result))
	}
	return result, err
}

func tool(name, description string) map[string]any {
	return map[string]any{
		"name": name, "description": description,
		"inputSchema": map[string]any{"type": "object", "additionalProperties": true},
	}
}

func respond(id, result any, err error) {
	resp := rpcResponse{JSONRPC: "2.0", ID: id, Result: result}
	if err != nil {
		resp.Result = nil
		resp.Error = map[string]any{"code": -32000, "message": err.Error()}
	}
	_ = json.NewEncoder(os.Stdout).Encode(resp)
}

func str(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func intVal(v any, fallback int) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return fallback
	}
}

func defaultStr(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func optionalList(v string) []string {
	if v == "" {
		return nil
	}
	return []string{v}
}

func resultCount(v any) int {
	switch x := v.(type) {
	case []any:
		return len(x)
	case map[string]any:
		if n, ok := x["total"].(float64); ok {
			return int(n)
		}
		if rs, ok := x["results"].([]any); ok {
			return len(rs)
		}
		return 1
	default:
		return 1
	}
}
