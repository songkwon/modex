package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
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
	transport := flag.String("transport", env("MODEX_MCP_TRANSPORT", "stdio"), "MCP transport: stdio or http")
	addr := flag.String("addr", env("MODEX_MCP_ADDR", ":8787"), "HTTP listen address")
	path := flag.String("path", env("MODEX_MCP_PATH", "/mcp"), "HTTP MCP endpoint path")
	flag.Parse()

	switch strings.ToLower(*transport) {
	case "http", "streamable-http", "streamable_http":
		runHTTP(c, *addr, *path)
	case "stdio":
		runStdio(c)
	default:
		log.Fatalf("unsupported transport %q", *transport)
	}
}

func runStdio(c client.Client) {
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

func runHTTP(c client.Client, addr, path string) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "use GET", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	})
	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		writeOAuthAuthorizationMetadata(c, w, r)
	})
	mux.HandleFunc("/.well-known/oauth-protected-resource", func(w http.ResponseWriter, r *http.Request) {
		writeOAuthProtectedResourceMetadata(c, w, r)
	})
	mux.HandleFunc("/.well-known/oauth-protected-resource"+path, func(w http.ResponseWriter, r *http.Request) {
		writeOAuthProtectedResourceMetadata(c, w, r)
	})
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		handleHTTPRPC(c, w, r)
	})
	log.Printf("modex MCP streamable HTTP listening on %s%s", addr, path)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func handleHTTPRPC(c client.Client, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept, Mcp-Session-Id")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("MCP-Protocol-Version", "2024-11-05")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{
			"name":      "modex-mcp-server",
			"transport": "streamable-http",
			"endpoint":  r.URL.Path,
		})
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}
	if token := bearerToken(r); token != "" {
		c = c.WithToken(token)
	}
	defer r.Body.Close()
	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeJSON(w, http.StatusBadRequest, rpcResponse{JSONRPC: "2.0", ID: nil, Error: rpcError(-32700, "parse error")})
		return
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		writeJSON(w, http.StatusBadRequest, rpcResponse{JSONRPC: "2.0", ID: nil, Error: rpcError(-32700, "parse error")})
		return
	}
	if raw[0] == '[' {
		var reqs []rpcRequest
		if err := json.Unmarshal(raw, &reqs); err != nil {
			writeJSON(w, http.StatusBadRequest, rpcResponse{JSONRPC: "2.0", ID: nil, Error: rpcError(-32600, "invalid request")})
			return
		}
		responses := make([]rpcResponse, 0, len(reqs))
		for _, req := range reqs {
			if isNotification(req) {
				_, _ = handle(c, req)
				continue
			}
			responses = append(responses, handleRPC(c, req))
		}
		if len(responses) == 0 {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		writeJSON(w, http.StatusOK, responses)
		return
	}
	var req rpcRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, rpcResponse{JSONRPC: "2.0", ID: nil, Error: rpcError(-32600, "invalid request")})
		return
	}
	if isNotification(req) {
		_, _ = handle(c, req)
		w.WriteHeader(http.StatusAccepted)
		return
	}
	writeJSON(w, http.StatusOK, handleRPC(c, req))
}

func writeOAuthAuthorizationMetadata(c client.Client, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "use GET", http.StatusMethodNotAllowed)
		return
	}
	base := strings.TrimRight(c.BaseURL, "/")
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                base,
		"authorization_endpoint":                base + "/oauth/authorize",
		"token_endpoint":                        base + "/oauth/token",
		"revocation_endpoint":                   base + "/oauth/revoke",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"token_endpoint_auth_methods_supported": []string{"none", "client_secret_basic", "client_secret_post"},
		"scopes_supported":                      []string{"modex:mcp:read", "modex:docs:read"},
	})
}

func writeOAuthProtectedResourceMetadata(c client.Client, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "use GET", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"resource":              resourceURL(r),
		"authorization_servers": []string{strings.TrimRight(c.BaseURL, "/")},
		"scopes_supported":      []string{"modex:mcp:read", "modex:docs:read"},
	})
}

func resourceURL(r *http.Request) string {
	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = strings.Split(proto, ",")[0]
	}
	host := r.Host
	if forwarded := r.Header.Get("X-Forwarded-Host"); forwarded != "" {
		host = strings.Split(forwarded, ",")[0]
	}
	return strings.TrimRight(scheme+"://"+strings.TrimSpace(host)+r.URL.Path, "/")
}

func bearerToken(r *http.Request) string {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[len("Bearer "):])
	}
	return ""
}

func handleRPC(c client.Client, req rpcRequest) rpcResponse {
	result, err := handle(c, req)
	if err != nil {
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: rpcError(-32000, err.Error())}
	}
	return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: result}
}

func isNotification(req rpcRequest) bool {
	return req.ID == nil
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
		result, err := callTool(c, params.Name, params.Arguments)
		if err != nil {
			return nil, err
		}
		return toolResult(result), nil
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

func toolResult(v any) map[string]any {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		b = []byte(fmt.Sprint(v))
	}
	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": string(b)},
		},
	}
}

func respond(id, result any, err error) {
	resp := rpcResponse{JSONRPC: "2.0", ID: id, Result: result}
	if err != nil {
		resp.Result = nil
		resp.Error = rpcError(-32000, err.Error())
	}
	_ = json.NewEncoder(os.Stdout).Encode(resp)
}

func rpcError(code int, message string) map[string]any {
	return map[string]any{"code": code, "message": message}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
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

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
