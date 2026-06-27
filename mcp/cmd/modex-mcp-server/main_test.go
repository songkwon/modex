package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"modex/mcp/internal/client"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestToolValidationUsesSDKCallPath(t *testing.T) {
	session := newTestSession(t, client.Client{})

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_doc_page",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !res.IsError || !strings.Contains(strings.ToLower(toolText(res)), "required") {
		t.Fatalf("result = %+v, want doc_id validation error", res)
	}

	res, err = session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "search_docs",
		Arguments: map[string]any{"query": "guide", "mode": "magic"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !res.IsError || !strings.Contains(toolText(res), "mode must be") {
		t.Fatalf("result = %+v, want mode validation error", res)
	}
}

func TestToolsListExposesModexTools(t *testing.T) {
	session := newTestSession(t, client.Client{})

	result, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	got := map[string]bool{}
	for _, tool := range result.Tools {
		got[tool.Name] = true
		if tool.InputSchema == nil {
			t.Fatalf("tool %s missing input schema", tool.Name)
		}
		if tool.OutputSchema == nil {
			t.Fatalf("tool %s missing output schema", tool.Name)
		}
	}
	for _, name := range []string{"list_modules", "list_versions", "search_docs", "get_doc_page"} {
		if !got[name] {
			t.Fatalf("tool %s missing from list: %+v", name, got)
		}
	}
}

func TestOAuthMetadataForCodex(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "modex.example.com")
	rr := httptest.NewRecorder()

	writeOAuthAuthorizationMetadata(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rr.Code, http.StatusOK, rr.Body.String())
	}
	body := rr.Body.String()
	for _, want := range []string{
		`"authorization_endpoint":"https://modex.example.com/oauth/authorize"`,
		`"token_endpoint_auth_methods_supported":["none","client_secret_basic","client_secret_post"]`,
		`"modex:mcp:read"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metadata missing %s: %s", want, body)
		}
	}
}

func TestOAuthProtectedResourceMetadataUsesPublicAuthBase(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://mcp:8787/.well-known/oauth-protected-resource/mcp", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "modex.example.com")
	rr := httptest.NewRecorder()

	writeOAuthProtectedResourceMetadata(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rr.Code, http.StatusOK, rr.Body.String())
	}
	body := rr.Body.String()
	for _, want := range []string{
		`"resource":"https://modex.example.com/.well-known/oauth-protected-resource/mcp"`,
		`"authorization_servers":["https://modex.example.com"]`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metadata missing %s: %s", want, body)
		}
	}
}

func TestBearerMiddlewareUsesSDKAuthentication(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/mcp/token-info" || r.Header.Get("Authorization") != "Bearer oauth-access-token" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(client.TokenInfo{
			UserID: "user-1", Scopes: []string{"modex:mcp:read"}, ExpiresAt: time.Now().Add(time.Hour),
		})
	}))
	defer backend.Close()

	c := client.Client{BaseURL: backend.URL, HTTP: backend.Client()}
	protected := requireBearerToken(c, "/mcp")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := auth.TokenInfoFromContext(r.Context())
		if info == nil || info.UserID != "user-1" || info.Extra["access_token"] != "oauth-access-token" {
			http.Error(w, "missing token info", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	missing := httptest.NewRecorder()
	protected.ServeHTTP(missing, httptest.NewRequest(http.MethodPost, "https://modex.example.com/mcp", nil))
	if missing.Code != http.StatusUnauthorized || !strings.Contains(missing.Header().Get("WWW-Authenticate"), "resource_metadata=") {
		t.Fatalf("missing token response = %d %q", missing.Code, missing.Header().Get("WWW-Authenticate"))
	}

	validReq := httptest.NewRequest(http.MethodPost, "https://modex.example.com/mcp", nil)
	validReq.Header.Set("Authorization", "Bearer oauth-access-token")
	valid := httptest.NewRecorder()
	protected.ServeHTTP(valid, validReq)
	if valid.Code != http.StatusNoContent {
		t.Fatalf("valid token status = %d: %s", valid.Code, valid.Body.String())
	}
}

func TestStructuredToolOutputAndDocumentResource(t *testing.T) {
	logged := make(chan map[string]any, 1)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/modules":
			_, _ = w.Write([]byte(`[{"module_key":"guide","name":"Guide"}]`))
		case strings.HasPrefix(r.URL.Path, "/api/docs/page/"):
			_, _ = w.Write([]byte(`{"doc_id":"doc-1","title":"Guide","content_md":"# Guide\n\nHello"}`))
		case r.URL.Path == "/api/mcp/log":
			var entry map[string]any
			_ = json.NewDecoder(r.Body).Decode(&entry)
			logged <- entry
			w.WriteHeader(http.StatusAccepted)
		default:
			http.NotFound(w, r)
		}
	}))
	defer backend.Close()

	session := newTestSession(t, client.Client{BaseURL: backend.URL, Token: "token", HTTP: backend.Client()})
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "list_modules"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	structured, ok := result.StructuredContent.(map[string]any)
	if !ok || structured["modules"] == nil {
		t.Fatalf("structured output = %#v", result.StructuredContent)
	}
	entry := <-logged
	if entry["tool_name"] != "list_modules" || entry["result_count"] != float64(1) {
		t.Fatalf("middleware log = %#v", entry)
	}

	templates, err := session.ListResourceTemplates(context.Background(), nil)
	if err != nil || len(templates.ResourceTemplates) != 1 {
		t.Fatalf("resource templates = %#v, %v", templates, err)
	}
	resource, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: "modex://docs/doc-1"})
	if err != nil {
		t.Fatalf("ReadResource: %v", err)
	}
	if len(resource.Contents) != 1 || resource.Contents[0].Text != "# Guide\n\nHello" {
		t.Fatalf("resource = %#v", resource)
	}
}

func newTestSession(t *testing.T, c client.Client) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	server := newMCPServer(c)
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	t.Cleanup(func() {
		serverSession.Close()
	})

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.1.0"}, nil)
	session, err := mcpClient.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() {
		session.Close()
		serverSession.Wait()
	})
	return session
}

func toolText(res *mcp.CallToolResult) string {
	var b strings.Builder
	for _, content := range res.Content {
		if text, ok := content.(*mcp.TextContent); ok {
			b.WriteString(text.Text)
		}
	}
	return b.String()
}
