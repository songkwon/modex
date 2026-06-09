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
	"os"
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

type HTTPProvider struct {
	URL    string
	APIKey string
	Client *http.Client
}

func (p HTTPProvider) Name() string { return "http" }

func (p HTTPProvider) EmbedText(ctx context.Context, text string) ([]float32, error) {
	batch, err := p.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(batch) == 0 {
		return nil, errors.New("embedding provider returned no vectors")
	}
	return batch[0], nil
}

func (p HTTPProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if p.URL == "" {
		return nil, errors.New("EMBEDDING_HTTP_URL is empty")
	}
	body, _ := json.Marshal(map[string]any{"texts": texts})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
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
		Embeddings [][]float32 `json:"embeddings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	return decoded.Embeddings, nil
}

func FromEnv() Provider {
	dim := envInt("EMBEDDING_DIM", 384)
	if os.Getenv("EMBEDDING_PROVIDER") == "http" {
		return HTTPProvider{URL: os.Getenv("EMBEDDING_HTTP_URL"), APIKey: os.Getenv("EMBEDDING_HTTP_API_KEY")}
	}
	return MockProvider{Dim: dim}
}

func envInt(key string, fallback int) int {
	var v int
	if _, err := fmt.Sscanf(os.Getenv(key), "%d", &v); err == nil && v > 0 {
		return v
	}
	return fallback
}
