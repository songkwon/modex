package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"modex/mcp/internal/client"

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

func TestBearerToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer oauth-access-token")

	if got := bearerToken(req); got != "oauth-access-token" {
		t.Fatalf("bearerToken = %q, want oauth-access-token", got)
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
