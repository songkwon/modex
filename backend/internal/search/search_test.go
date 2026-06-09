package search

import (
	"context"
	"testing"

	"modex/backend/internal/embedding"
	"modex/backend/internal/store"
)

func newService() Service {
	return Service{
		Store:          store.NewSeeded(),
		Embedder:       embedding.MockProvider{Dim: 256},
		KeywordWeight:  0.6,
		SemanticWeight: 0.4,
	}
}

func TestReindexPopulatesEmbeddingCache(t *testing.T) {
	s := newService()
	if s.Store.EmbeddingCount() != 0 {
		t.Fatalf("expected empty cache before reindex")
	}
	n, err := s.Reindex(context.Background())
	if err != nil {
		t.Fatalf("Reindex: %v", err)
	}
	if n == 0 || n != s.Store.EmbeddingCount() {
		t.Fatalf("reindexed %d but cache holds %d", n, s.Store.EmbeddingCount())
	}
}

func TestSemanticSearchReturnsRankedResults(t *testing.T) {
	s := newService()
	if _, err := s.Reindex(context.Background()); err != nil {
		t.Fatalf("Reindex: %v", err)
	}
	resp, err := s.Search(context.Background(), Request{Query: "构建缓存怎么清理", Mode: ModeSemantic, PageSize: 5})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if resp.Mode != ModeSemantic {
		t.Fatalf("mode = %s, want semantic", resp.Mode)
	}
	if len(resp.Results) == 0 {
		t.Fatal("expected semantic results")
	}
	// Results must be sorted by descending score.
	for i := 1; i < len(resp.Results); i++ {
		if resp.Results[i-1].Score < resp.Results[i].Score {
			t.Fatalf("results not sorted by score: %v", resp.Results)
		}
	}
}

func TestKeywordModeSkipsEmbeddingButStillMatches(t *testing.T) {
	s := newService()
	resp, err := s.Search(context.Background(), Request{Query: "构建缓存", Mode: ModeKeyword, PageSize: 5})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(resp.Results) == 0 {
		t.Fatal("expected keyword results")
	}
	// Keyword mode must not have triggered embedding caching.
	if s.Store.EmbeddingCount() != 0 {
		t.Fatalf("keyword search should not populate embedding cache, got %d", s.Store.EmbeddingCount())
	}
}
