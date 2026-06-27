package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"modex/mcp/internal/client"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

type listModulesInput struct {
	CategoryID string `json:"category_id,omitempty" jsonschema:"Optional category/platform id."`
	Keyword    string `json:"keyword,omitempty" jsonschema:"Optional module keyword."`
}

type listVersionsInput struct {
	ModuleKey string `json:"module_key" jsonschema:"Module key."`
}

type searchDocsInput struct {
	Query       string `json:"query" jsonschema:"Search query."`
	Mode        string `json:"mode,omitempty" jsonschema:"Search mode: keyword, semantic, or hybrid. Defaults to hybrid."`
	Limit       int    `json:"limit,omitempty" jsonschema:"Maximum results. Defaults to 5."`
	ModuleKey   string `json:"module_key,omitempty" jsonschema:"Optional module key filter."`
	DocsVersion string `json:"docs_version,omitempty" jsonschema:"Optional docs version filter."`
}

type getDocPageInput struct {
	DocID string `json:"doc_id" jsonschema:"Document id returned by search_docs."`
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
		if err := newMCPServer(c).Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("unsupported transport %q", *transport)
	}
}

func newMCPServer(c client.Client) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "modex-mcp-server", Version: "0.1.0"}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_modules",
		Description: "List documentation modules by category or keyword",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input listModulesInput) (*mcp.CallToolResult, any, error) {
		result, err := c.ListModules(input.CategoryID, input.Keyword)
		return toolCallResult(c, "list_modules", input.Keyword, input, result, err)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_versions",
		Description: "List versions for one module",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input listVersionsInput) (*mcp.CallToolResult, any, error) {
		if input.ModuleKey == "" {
			return nil, nil, fmt.Errorf("module_key is required")
		}
		result, err := c.ListVersions(input.ModuleKey)
		return toolCallResult(c, "list_versions", "", input, result, err)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_docs",
		Description: "Search docs with keyword, semantic, or hybrid mode",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input searchDocsInput) (*mcp.CallToolResult, any, error) {
		if input.Query == "" {
			return nil, nil, fmt.Errorf("query is required")
		}
		limit := input.Limit
		if limit == 0 {
			limit = 5
		}
		limit = clampInt(limit, 1, 20)
		mode := defaultStr(input.Mode, "hybrid")
		if mode != "keyword" && mode != "semantic" && mode != "hybrid" {
			return nil, nil, fmt.Errorf("mode must be keyword, semantic, or hybrid")
		}
		body := map[string]any{
			"query": input.Query,
			"mode":  mode,
			"page":  1, "page_size": limit,
			"filters": map[string]any{
				"modules":       optionalList(input.ModuleKey),
				"docs_versions": optionalList(input.DocsVersion),
			},
		}
		result, err := c.SearchDocs(body)
		return toolCallResult(c, "search_docs", input.Query, input, result, err)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_doc_page",
		Description: "Get one document page by doc_id",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input getDocPageInput) (*mcp.CallToolResult, any, error) {
		if input.DocID == "" {
			return nil, nil, fmt.Errorf("doc_id is required")
		}
		result, err := c.GetDocPage(input.DocID)
		return toolCallResult(c, "get_doc_page", "", input, result, err)
	})

	return server
}

func toolCallResult(c client.Client, name, query string, input any, result any, err error) (*mcp.CallToolResult, any, error) {
	if err != nil {
		return nil, nil, err
	}
	c.LogMCP(name, query, input, resultCount(result))
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: prettyJSON(result)},
		},
	}, nil, nil
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
		writeOAuthAuthorizationMetadata(w, r)
	})
	mux.HandleFunc("/.well-known/oauth-protected-resource", func(w http.ResponseWriter, r *http.Request) {
		writeOAuthProtectedResourceMetadata(w, r)
	})
	mux.HandleFunc("/.well-known/oauth-protected-resource"+path, func(w http.ResponseWriter, r *http.Request) {
		writeOAuthProtectedResourceMetadata(w, r)
	})
	mux.Handle(path, mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		if token := bearerToken(r); token != "" {
			return newMCPServer(c.WithToken(token))
		}
		return newMCPServer(c)
	}, nil))
	log.Printf("modex MCP streamable HTTP listening on %s%s", addr, path)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func writeOAuthAuthorizationMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "use GET", http.StatusMethodNotAllowed)
		return
	}
	base := requestOrigin(r)
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

func writeOAuthProtectedResourceMetadata(w http.ResponseWriter, r *http.Request) {
	auth.ProtectedResourceMetadataHandler(&oauthex.ProtectedResourceMetadata{
		Resource:               resourceURL(r),
		AuthorizationServers:   []string{requestOrigin(r)},
		ScopesSupported:        []string{"modex:mcp:read", "modex:docs:read"},
		BearerMethodsSupported: []string{"header"},
		ResourceName:           "Modex MCP",
	}).ServeHTTP(w, r)
}

func resourceURL(r *http.Request) string {
	return strings.TrimRight(requestOrigin(r)+r.URL.Path, "/")
}

func requestOrigin(r *http.Request) string {
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
	return strings.TrimRight(scheme+"://"+strings.TrimSpace(host), "/")
}

func bearerToken(r *http.Request) string {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[len("Bearer "):])
	}
	return ""
}

func prettyJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
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

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func resultCount(v any) int {
	switch x := v.(type) {
	case []any:
		return len(x)
	case map[string]any:
		if total, ok := x["total"].(float64); ok {
			return int(total)
		}
		if items, ok := x["items"].([]any); ok {
			return len(items)
		}
		if results, ok := x["results"].([]any); ok {
			return len(results)
		}
		return 1
	default:
		if v == nil {
			return 0
		}
		return 1
	}
}
