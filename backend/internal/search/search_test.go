package search

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"modex/backend/internal/embedding"
	"modex/backend/internal/store"
)

func TestRerankUsesAdminSettings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/rerank" {
			t.Fatalf("path = %q, want /v1/rerank", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("authorization = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"results": []any{
			map[string]any{"index": 1, "relevance_score": 0.9},
			map[string]any{"index": 0, "relevance_score": 0.2},
		}})
	}))
	defer server.Close()

	st := store.NewTestStore()
	st.SaveAISettings(store.AISettings{
		RerankBaseURL: server.URL + "/v1",
		RerankModel:   "rerank-v1",
		RerankAPIKey:  "secret",
		RerankTopK:    2,
	})
	results, err := (Service{Store: st}).rerank(context.Background(), "query", []Result{
		{DocID: "first", Title: "First"},
		{DocID: "second", Title: "Second"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if results[0].DocID != "second" || results[0].Score != 0.9 {
		t.Fatalf("results = %#v", results)
	}
}

func TestSearchFallsBackWhenRerankProviderFails(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	st := store.NewSeededTestStore()
	st.SaveAISettings(store.AISettings{
		RerankBaseURL: server.URL + "/v1",
		RerankModel:   "rerank-v1",
		RerankTopK:    2,
	})
	s := Service{Store: st, Embedder: embedding.FallbackProvider{Dim: 256}, KeywordWeight: 0.6, SemanticWeight: 0.4}

	resp, err := s.Search(context.Background(), Request{Query: "构建缓存", Mode: ModeKeyword, PageSize: 5})
	if err != nil {
		t.Fatalf("Search should fall back when rerank fails: %v", err)
	}
	if len(resp.Results) == 0 {
		t.Fatal("expected search results without rerank")
	}
}

type fakeVectorStore struct {
	vectors  map[string][]float32
	simCalls int
}

func (f *fakeVectorStore) Existing(_ context.Context, docIDs []string) (map[string]bool, error) {
	out := map[string]bool{}
	for _, id := range docIDs {
		if _, ok := f.vectors[id]; ok {
			out[id] = true
		}
	}
	return out, nil
}

func (f *fakeVectorStore) Similarities(_ context.Context, query []float32, docIDs []string, _ int) (map[string]float64, error) {
	f.simCalls++
	out := map[string]float64{}
	for _, id := range docIDs {
		out[id] = cosine(query, f.vectors[id])
	}
	return out, nil
}

func (f *fakeVectorStore) Upsert(_ context.Context, docID string, vector []float32) error {
	f.vectors[docID] = append([]float32(nil), vector...)
	return nil
}

func (f *fakeVectorStore) Clear(context.Context) error {
	f.vectors = map[string][]float32{}
	return nil
}

func (f *fakeVectorStore) DeletePrefix(_ context.Context, prefix string) error {
	for id := range f.vectors {
		if strings.HasPrefix(id, prefix) {
			delete(f.vectors, id)
		}
	}
	return nil
}

func (f *fakeVectorStore) Count(context.Context) (int, error) { return len(f.vectors), nil }

func newService() Service {
	return Service{
		Store:          store.NewSeededTestStore(),
		Embedder:       embedding.FallbackProvider{Dim: 256},
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

func TestSemanticSearchUsesExternalVectorStoreWithoutMemoryCache(t *testing.T) {
	s := newService()
	vectors := &fakeVectorStore{vectors: map[string][]float32{}}
	s.Vectors = vectors
	if _, err := s.Reindex(context.Background()); err != nil {
		t.Fatalf("Reindex: %v", err)
	}
	if s.Store.EmbeddingCount() != 0 {
		t.Fatalf("in-memory embeddings = %d, want 0", s.Store.EmbeddingCount())
	}
	if _, err := s.Search(context.Background(), Request{Query: "构建缓存", Mode: ModeSemantic}); err != nil {
		t.Fatalf("Search: %v", err)
	}
	if vectors.simCalls != 1 {
		t.Fatalf("similarity queries = %d, want one pgvector query", vectors.simCalls)
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

func TestDefaultVersionsOnlyFiltersDuplicateOldVersions(t *testing.T) {
	st := store.NewTestStore()
	ingest := func(version, body string) {
		t.Helper()
		_, err := st.IngestArtifact(store.DeployArtifact{
			ModuleKey:   "Threadpool",
			ModuleName:  "Threadpool",
			DocsVersion: version,
			Entries:     []store.DeployEntry{{Key: "guide", Title: "Guide", Type: "markdown"}},
			Documents: []store.DeployDocument{{
				DocID:       "Threadpool:" + version + ":guide",
				EntryKey:    "guide",
				EntryType:   "markdown",
				Title:       "线程池配置",
				Description: "线程池配置",
				Content:     body,
				Status:      "active",
			}},
		})
		if err != nil {
			t.Fatalf("IngestArtifact(%s): %v", version, err)
		}
	}
	ingest("v1.0.0", "线程池配置 max_workers legacy")
	ingest("v2.0.0", "线程池配置 max_workers current")

	s := Service{Store: st, Embedder: embedding.FallbackProvider{Dim: 256}}
	resp, err := s.Search(context.Background(), Request{Query: "max_workers", Mode: ModeKeyword, PageSize: 10, DefaultVersionsOnly: true})
	if err != nil {
		t.Fatalf("Search default versions: %v", err)
	}
	if len(resp.Results) != 1 || resp.Results[0].DocsVersion != "v2.0.0" {
		t.Fatalf("default-version results = %#v, want only v2.0.0", resp.Results)
	}

	resp, err = s.Search(context.Background(), Request{Query: "max_workers", Mode: ModeKeyword, PageSize: 10, Filters: Filters{DocsVersions: []string{"v1.0.0"}}, DefaultVersionsOnly: true})
	if err != nil {
		t.Fatalf("Search explicit version: %v", err)
	}
	if len(resp.Results) != 1 || resp.Results[0].DocsVersion != "v1.0.0" {
		t.Fatalf("explicit-version results = %#v, want v1.0.0", resp.Results)
	}
}

func TestPlainText(t *testing.T) {
	cases := []struct{ in, want string }{
		{"**二、使用指南**", "二、使用指南"},
		{"# **系统概述**\n## **产品定位**\n**[CBB系统](https://cbb.fsdev.cn/#/x)** 是一个集质量管控", "系统概述 产品定位 CBB系统 是一个集质量管控"},
		{"```html\nCBB V25\n```\n- [0.概述](https://x)", "0.概述"},
		{"use C# here", "use C# here"}, // hash not a heading marker
	}
	for _, c := range cases {
		if got := plainText(c.in); got != c.want {
			t.Errorf("plainText(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
