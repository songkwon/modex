package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

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
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		var req rpcRequest
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			respond(nil, nil, err)
			continue
		}
		result, err := handle(c, req)
		respond(req.ID, result, err)
	}
	if err := scanner.Err(); err != nil {
		respond(nil, nil, fmt.Errorf("stdin read failed: %w", err))
	}
}

func handle(c client.Client, req rpcRequest) (any, error) {
	switch req.Method {
	case "initialize":
		return map[string]any{"protocolVersion": "2024-11-05", "serverInfo": map[string]any{"name": "modex-mcp-server", "version": "0.1.0"}, "capabilities": map[string]any{"tools": map[string]any{}}}, nil
	case "tools/list":
		return map[string]any{"tools": []map[string]any{
			tool("list_modules", "List documentation modules by category or keyword", map[string]any{
				"category_id": propString("Optional category/platform id."),
				"keyword":     propString("Optional module keyword."),
			}, nil),
			tool("list_versions", "List versions for one module", map[string]any{
				"module_key": propString("Module key."),
			}, []string{"module_key"}),
			tool("search_docs", "Search docs with keyword, semantic, or hybrid mode", map[string]any{
				"query":        propString("Search query."),
				"mode":         map[string]any{"type": "string", "enum": []string{"keyword", "semantic", "hybrid"}, "description": "Search mode. Defaults to hybrid."},
				"limit":        map[string]any{"type": "integer", "minimum": 1, "maximum": 20, "description": "Maximum results. Defaults to 5."},
				"module_key":   propString("Optional module key filter."),
				"docs_version": propString("Optional docs version filter."),
			}, []string{"query"}),
			tool("get_doc_page", "Get one document page by doc_id", map[string]any{
				"doc_id": propString("Document id returned by search_docs."),
			}, []string{"doc_id"}),
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
	if args == nil {
		args = map[string]any{}
	}
	var result any
	var err error
	switch name {
	case "list_modules":
		result, err = c.ListModules(str(args["category_id"]), str(args["keyword"]))
	case "list_versions":
		moduleKey := str(args["module_key"])
		if moduleKey == "" {
			return nil, fmt.Errorf("module_key is required")
		}
		result, err = c.ListVersions(moduleKey)
	case "search_docs":
		query := str(args["query"])
		if query == "" {
			return nil, fmt.Errorf("query is required")
		}
		limit := clampInt(intVal(args["limit"], 5), 1, 20)
		mode := defaultStr(str(args["mode"]), "hybrid")
		if mode != "keyword" && mode != "semantic" && mode != "hybrid" {
			return nil, fmt.Errorf("mode must be keyword, semantic, or hybrid")
		}
		body := map[string]any{
			"query": query,
			"mode":  mode,
			"page":  1, "page_size": limit,
			"filters": map[string]any{
				"modules":       optionalList(str(args["module_key"])),
				"docs_versions": optionalList(str(args["docs_version"])),
			},
		}
		result, err = c.SearchDocs(body)
	case "get_doc_page":
		docID := str(args["doc_id"])
		if docID == "" {
			return nil, fmt.Errorf("doc_id is required")
		}
		result, err = c.GetDocPage(docID)
	default:
		return nil, fmt.Errorf("unknown tool %s", name)
	}
	if err == nil {
		c.LogMCP(name, str(args["query"]), args, resultCount(result))
	}
	return result, err
}

func tool(name, description string, properties map[string]any, required []string) map[string]any {
	return map[string]any{
		"name": name, "description": description,
		"inputSchema": map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false},
	}
}

func propString(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
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
		return strings.TrimSpace(s)
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

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
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
