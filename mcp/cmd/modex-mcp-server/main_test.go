package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"modex/mcp/internal/client"
)

func TestCallToolValidatesRequiredArguments(t *testing.T) {
	_, err := callTool(client.Client{}, "get_doc_page", map[string]any{})
	if err == nil || !strings.Contains(err.Error(), "doc_id is required") {
		t.Fatalf("error = %v, want doc_id validation", err)
	}

	_, err = callTool(client.Client{}, "search_docs", map[string]any{"query": "guide", "mode": "magic"})
	if err == nil || !strings.Contains(err.Error(), "mode must be") {
		t.Fatalf("error = %v, want mode validation", err)
	}
}

func TestHTTPRPCInitialize(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`))
	rr := httptest.NewRecorder()

	handleHTTPRPC(client.Client{}, rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var resp rpcResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.JSONRPC != "2.0" || resp.ID.(float64) != 1 || resp.Error != nil {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestHTTPRPCNotificationReturnsAccepted(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","method":"notifications/initialized"}`))
	rr := httptest.NewRecorder()

	handleHTTPRPC(client.Client{}, rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d: %s", rr.Code, http.StatusAccepted, rr.Body.String())
	}
}

func TestToolsListExposesStrictSchemas(t *testing.T) {
	result, err := handle(client.Client{}, rpcRequest{Method: "tools/list"})
	if err != nil {
		t.Fatalf("tools/list: %v", err)
	}
	b, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	body := string(b)
	for _, want := range []string{`"required":["query"]`, `"additionalProperties":false`, `"enum":["keyword","semantic","hybrid"]`} {
		if !strings.Contains(body, want) {
			t.Fatalf("tools/list schema missing %s: %s", want, body)
		}
	}
}
