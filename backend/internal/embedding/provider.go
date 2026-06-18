package embedding

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Provider interface {
	Name() string
	EmbedText(ctx context.Context, text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}

type MockProvider struct {
	Dim int
}

func (p MockProvider) Name() string { return "mock" }

func (p MockProvider) EmbedText(ctx context.Context, text string) ([]float32, error) {
	dim := p.Dim
	if dim <= 0 {
		dim = 384
	}
	vec := make([]float32, dim)
	seed := sha256.Sum256([]byte(text))
	for i := 0; i < dim; i++ {
		h := sha256.Sum256(append(seed[:], byte(i), byte(i>>8)))
		n := binary.BigEndian.Uint32(h[:4])
		vec[i] = float32(int(n%2000)-1000) / 1000
	}
	return vec, nil
}

func (p MockProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, 0, len(texts))
	for _, text := range texts {
		vec, err := p.EmbedText(ctx, text)
		if err != nil {
			return nil, err
		}
		out = append(out, vec)
	}
	return out, nil
}

// Settings is the admin-managed embedding configuration needed by Provider.
type Settings struct {
	BaseURL string
	Model   string
	APIKey  string
	Dim     int
}

// SettingsProvider resolves configuration for every request, so model changes
// made in the admin page take effect without restarting the backend.
type SettingsProvider struct {
	Load   func() Settings
	Client *http.Client
}

func (p SettingsProvider) Name() string {
	if cfg := p.settings(); cfg.BaseURL != "" && cfg.Model != "" {
		return "admin"
	}
	return "mock"
}

func (p SettingsProvider) EmbedText(ctx context.Context, text string) ([]float32, error) {
	vectors, err := p.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vectors) == 0 {
		return nil, errors.New("embedding provider returned no vectors")
	}
	return vectors[0], nil
}

func (p SettingsProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	cfg := p.settings()
	if cfg.BaseURL == "" || cfg.Model == "" {
		return MockProvider{Dim: cfg.Dim}.EmbedBatch(ctx, texts)
	}
	endpoint := strings.TrimRight(cfg.BaseURL, "/")
	if !strings.HasSuffix(endpoint, "/embeddings") {
		endpoint += "/embeddings"
	}
	body, _ := json.Marshal(map[string]any{"model": cfg.Model, "input": texts})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("embedding http status %d", resp.StatusCode)
	}
	var decoded struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	out := make([][]float32, len(decoded.Data))
	for i, item := range decoded.Data {
		idx := item.Index
		if idx < 0 || idx >= len(out) {
			idx = i
		}
		out[idx] = item.Embedding
	}
	return out, nil
}

func (p SettingsProvider) settings() Settings {
	if p.Load == nil {
		return Settings{Dim: 384}
	}
	cfg := p.Load()
	if cfg.Dim <= 0 {
		cfg.Dim = 384
	}
	return cfg
}
