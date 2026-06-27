package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
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

type listModulesOutput struct {
	Modules any `json:"modules" jsonschema:"Documentation modules matching the requested filters."`
}

type listVersionsOutput struct {
	Versions any `json:"versions" jsonschema:"Available documentation versions for the module."`
}

type searchDocsOutput struct {
	Search any `json:"search" jsonschema:"Search response containing results, total count, and facets."`
}

type getDocPageOutput struct {
	Page any `json:"page" jsonschema:"The requested documentation page."`
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
	}, func(ctx context.Context, req *mcp.CallToolRequest, input listModulesInput) (*mcp.CallToolResult, listModulesOutput, error) {
		result, err := c.ListModules(input.CategoryID, input.Keyword)
		return nil, listModulesOutput{Modules: result}, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_versions",
		Description: "List versions for one module",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input listVersionsInput) (*mcp.CallToolResult, listVersionsOutput, error) {
		if input.ModuleKey == "" {
			return nil, listVersionsOutput{}, fmt.Errorf("module_key is required")
		}
		result, err := c.ListVersions(input.ModuleKey)
		return nil, listVersionsOutput{Versions: result}, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_docs",
		Description: "Search docs with keyword, semantic, or hybrid mode",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input searchDocsInput) (*mcp.CallToolResult, searchDocsOutput, error) {
		if input.Query == "" {
			return nil, searchDocsOutput{}, fmt.Errorf("query is required")
		}
		limit := input.Limit
		if limit == 0 {
			limit = 5
		}
		limit = clampInt(limit, 1, 20)
		mode := defaultStr(input.Mode, "hybrid")
		if mode != "keyword" && mode != "semantic" && mode != "hybrid" {
			return nil, searchDocsOutput{}, fmt.Errorf("mode must be keyword, semantic, or hybrid")
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
		return nil, searchDocsOutput{Search: result}, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_doc_page",
		Description: "Get one document page by doc_id",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input getDocPageInput) (*mcp.CallToolResult, getDocPageOutput, error) {
		if input.DocID == "" {
			return nil, getDocPageOutput{}, fmt.Errorf("doc_id is required")
		}
		result, err := c.GetDocPage(input.DocID)
		return nil, getDocPageOutput{Page: result}, err
	})

	server.AddReceivingMiddleware(toolLoggingMiddleware(c))
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "modex://docs/{doc_id}",
		Name:        "modex_document",
		Title:       "Modex document",
		Description: "Read a Modex documentation page by the doc_id returned from search_docs.",
		MIMEType:    "text/markdown",
	}, docResourceHandler(c))

	return server
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
	mcpHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		if info := auth.TokenInfoFromContext(r.Context()); info != nil {
			token, _ := info.Extra["access_token"].(string)
			return newMCPServer(c.WithToken(token))
		}
		return newMCPServer(c)
	}, nil)
	mux.Handle(path, requireBearerToken(c, path)(mcpHandler))
	log.Printf("modex MCP streamable HTTP listening on %s%s", addr, path)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func requireBearerToken(c client.Client, path string) func(http.Handler) http.Handler {
	verifier := func(ctx context.Context, token string, req *http.Request) (*auth.TokenInfo, error) {
		info, err := c.VerifyToken(token)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", auth.ErrInvalidToken, err)
		}
		return &auth.TokenInfo{
			UserID:     info.UserID,
			Scopes:     info.Scopes,
			Expiration: info.ExpiresAt,
			Extra:      map[string]any{"access_token": token},
		}, nil
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth.RequireBearerToken(verifier, &auth.RequireBearerTokenOptions{
				ResourceMetadataURL: requestOrigin(r) + "/.well-known/oauth-protected-resource" + path,
				Scopes:              []string{"modex:mcp:read"},
			})(next).ServeHTTP(w, r)
		})
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

func toolLoggingMiddleware(c client.Client) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			result, err := next(ctx, method, req)
			if method != "tools/call" || err != nil {
				return result, err
			}
			call, ok := req.(*mcp.CallToolRequest)
			if !ok || call.Params == nil {
				return result, nil
			}
			var input map[string]any
			_ = json.Unmarshal(call.Params.Arguments, &input)
			query, _ := input["query"].(string)
			count := 0
			if toolResult, ok := result.(*mcp.CallToolResult); ok && !toolResult.IsError {
				count = resultCount(toolResult.StructuredContent)
			}
			c.LogMCP(call.Params.Name, query, input, count)
			return result, nil
		}
	}
}

func docResourceHandler(c client.Client) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		parsed, err := url.Parse(req.Params.URI)
		if err != nil || parsed.Scheme != "modex" || parsed.Host != "docs" {
			return nil, mcp.ResourceNotFoundError(req.Params.URI)
		}
		docID, err := url.PathUnescape(strings.TrimPrefix(parsed.Path, "/"))
		if err != nil || docID == "" {
			return nil, mcp.ResourceNotFoundError(req.Params.URI)
		}
		result, err := c.GetDocPage(docID)
		if err != nil {
			return nil, err
		}
		page, ok := result.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("unexpected document response")
		}
		text, _ := page["content_md"].(string)
		if text == "" {
			text, _ = page["content_text"].(string)
		}
		mimeType := "text/markdown"
		if text == "" {
			text = prettyJSON(page)
			mimeType = "application/json"
		}
		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{
			URI: req.Params.URI, MIMEType: mimeType, Text: text,
		}}}, nil
	}
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
		for _, key := range []string{"modules", "versions", "search", "page"} {
			if nested, ok := x[key]; ok {
				return resultCount(nested)
			}
		}
		return 1
	default:
		if v == nil {
			return 0
		}
		return 1
	}
}
