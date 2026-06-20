package embedding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSettingsProviderUsesCurrentAdminSettings(t *testing.T) {
	var requestedModel string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings" {
			t.Fatalf("path = %q, want /v1/embeddings", r.URL.Path)
		}
		var body struct {
			Model string   `json:"model"`
			Input []string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		requestedModel = body.Model
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []any{map[string]any{"index": 0, "embedding": []float32{0.1, 0.2}}},
		})
	}))
	defer server.Close()

	cfg := Settings{Dim: 3}
	provider := SettingsProvider{Load: func() Settings { return cfg }}
	if provider.Name() != "fallback" {
		t.Fatalf("provider name = %q, want fallback", provider.Name())
	}
	fallbackVector, err := provider.EmbedText(context.Background(), "hello")
	if err != nil || len(fallbackVector) != 3 {
		t.Fatalf("fallback vector len = %d, err = %v", len(fallbackVector), err)
	}

	cfg = Settings{BaseURL: server.URL + "/v1", Model: "embed-v2", APIKey: "secret"}
	vector, err := provider.EmbedText(context.Background(), "hello")
	if err != nil {
		t.Fatal(err)
	}
	if provider.Name() != "admin" || requestedModel != "embed-v2" || len(vector) != 2 {
		t.Fatalf("name=%q model=%q vector=%v", provider.Name(), requestedModel, vector)
	}
}
