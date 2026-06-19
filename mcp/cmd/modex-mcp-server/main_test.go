package main

import (
	"encoding/json"
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
