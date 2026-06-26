package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"modex/backend/internal/embedding"
	"modex/backend/internal/store"
)

const defaultEmbeddingInputRunes = 800
const maxEmbeddingChunksPerDoc = 512

type Mode string

const (
	ModeKeyword  Mode = "keyword"
	ModeSemantic Mode = "semantic"
	ModeHybrid   Mode = "hybrid"
)

type Filters struct {
	CategoryIDs  []string `json:"category_ids"`
	Modules      []string `json:"modules"`
	DocsVersions []string `json:"docs_versions"`
	EntryTypes   []string `json:"entry_types"`
	Keywords     []string `json:"keywords"`
	Owners       []string `json:"owners"`
	Status       []string `json:"status"`
}

type Request struct {
	Query    string  `json:"query"`
	Mode     Mode    `json:"mode"`
	Filters  Filters `json:"filters"`
	Page     int     `json:"page"`
	PageSize int     `json:"page_size"`
	// DefaultVersionsOnly limits results to each module's configured default
	// docs version when the caller has not explicitly requested versions.
	DefaultVersionsOnly bool `json:"default_versions_only"`
	// Log marks an explicit, user-committed search (Enter / search button /
	// result click) that should be persisted to the search log. Live
	// as-you-type queries leave this false so the log isn't flooded.
	Log          bool   `json:"log"`
	ClickedDocID string `json:"clicked_doc_id"`
}

type Result struct {
	DocID          string   `json:"doc_id"`
	Title          string   `json:"title"`
	Snippet        string   `json:"snippet"`
	Path           string   `json:"path"`
	Score          float64  `json:"score"`
	SearchMode     Mode     `json:"search_mode"`
	ModuleKey      string   `json:"module_key"`
	ModuleName     string   `json:"module_name"`
	DocsVersion    string   `json:"docs_version"`
	PackageVersion string   `json:"package_version"`
	EntryType      string   `json:"entry_type"`
	OwnerGroup     string   `json:"owner_group"`
	Status         string   `json:"status"`
	UpdatedAt      string   `json:"updated_at"`
	Keywords       []string `json:"keywords"`
	Breadcrumb     string   `json:"breadcrumb"`
	EntryKey       string   `json:"entry_key"`
	MatchTerms     []string `json:"match_terms"`
}

type Response struct {
	Query    string                    `json:"query"`
	Mode     Mode                      `json:"mode"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"page_size"`
	Total    int                       `json:"total"`
	Results  []Result                  `json:"results"`
	Facets   map[string]map[string]int `json:"facets"`
}

// ContentStore is the search-facing data boundary. Production uses the
// PostgreSQL repository; tests may use MemoryStore as an explicit fake.
type ContentStore interface {
	Pages() []store.Page
	Modules(categoryID, keyword string) []store.Module
	Settings() store.Settings
	Embedding(docID string) ([]float32, bool)
	SetEmbedding(docID string, vector []float32)
	ClearEmbeddings()
	EmbeddingCount() int
}

type Service struct {
	Store          ContentStore
	Embedder       embedding.Provider
	Vectors        VectorStore
	KeywordWeight  float64
	SemanticWeight float64
}

func (s Service) Search(ctx context.Context, req Request) (Response, error) {
	if req.Mode == "" {
		req.Mode = ModeHybrid
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}
	kw := s.KeywordWeight
	sw := s.SemanticWeight
	if kw == 0 {
		kw = 0.6
	}
	if sw == 0 {
		sw = 0.4
	}
	pages := s.Store.Pages()
	// Resolve module -> category breadcrumb once so each result carries a path.
	breadcrumbs := map[string]string{}
	defaultVersions := map[string]string{}
	for _, m := range s.Store.Modules("", "") {
		breadcrumbs[m.ModuleKey] = m.CategoryPath
		defaultVersions[m.ModuleKey] = defaultVersionForModule(m)
	}
	candidates := make([]store.Page, 0, len(pages))
	for _, p := range pages {
		if !matchFilters(p, req.Filters) {
			continue
		}
		if req.DefaultVersionsOnly && len(req.Filters.DocsVersions) == 0 {
			if def := defaultVersions[p.ModuleKey]; def != "" && p.DocsVersion != def {
				continue
			}
		}
		candidates = append(candidates, p)
	}
	terms := matchTerms(req.Query)
	// Semantic and hybrid modes need a query embedding; keyword mode skips it to
	// avoid an unnecessary embedding-provider call.
	var queryVec []float32
	if req.Mode != ModeKeyword && strings.TrimSpace(req.Query) != "" {
		var err error
		queryVec, err = s.Embedder.EmbedText(ctx, s.truncateEmbeddingInput(req.Query))
		if err != nil {
			return Response{}, fmt.Errorf("embed query: %w", err)
		}
	}
	// The fallback provider returns hash-based vectors (no real semantics). In
	// hybrid mode that noise floats weak docs up and buries strong keyword hits
	// (e.g. an "EventBus" page for the query "EventBus怎么用"), so when embeddings
	// come from the fallback we score hybrid by keyword only. Explicit semantic
	// mode still uses the vectors — the caller asked for vector ranking.
	fallbackEmbed := s.Embedder.Name() == "fallback"
	vectors := map[string][]float32{}
	semanticScores := map[string]float64{}
	if len(queryVec) > 0 {
		docIDs := make([]string, 0, len(candidates))
		for _, p := range candidates {
			docIDs = append(docIDs, p.DocID)
		}
		if s.Vectors != nil {
			var err error
			semanticScores, err = s.Vectors.Similarities(ctx, queryVec, docIDs, len(docIDs))
			if err != nil {
				return Response{}, err
			}
		} else {
			vectors, _ = s.embeddingBatch(docIDs)
		}
	}
	var scored []Result
	for _, p := range candidates {
		kScore := keywordScore(req.Query, p)
		var sScore float64
		if len(queryVec) > 0 {
			if s.Vectors != nil {
				sScore = semanticScores[p.DocID]
			} else {
				var err error
				sScore, err = s.pageSemanticScore(ctx, queryVec, p, vectors)
				if err != nil {
					return Response{}, err
				}
			}
		}
		var final float64
		switch req.Mode {
		case ModeKeyword:
			final = kScore
		case ModeSemantic:
			final = sScore
		default:
			if fallbackEmbed {
				final = kScore
			} else {
				final = kScore*kw + sScore*sw
			}
			req.Mode = ModeHybrid
		}
		if strings.TrimSpace(req.Query) != "" && final <= 0 {
			continue
		}
		breadcrumb := breadcrumbs[p.ModuleKey]
		if breadcrumb != "" {
			breadcrumb = breadcrumb + " / " + p.ModuleName
		} else {
			breadcrumb = p.ModuleName
		}
		scored = append(scored, Result{
			DocID: p.DocID, Title: plainText(p.Title), Snippet: snippet(req.Query, plainText(p.ContentText)), Path: p.Path,
			Score: final, SearchMode: req.Mode, ModuleKey: p.ModuleKey, ModuleName: p.ModuleName,
			DocsVersion: p.DocsVersion, PackageVersion: p.PackageVersion, EntryType: p.EntryType,
			OwnerGroup: p.OwnerGroup, Status: p.Status, UpdatedAt: p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Keywords: p.Tags, Breadcrumb: breadcrumb, EntryKey: p.EntryKey, MatchTerms: terms,
		})
	}
	sort.Slice(scored, func(i, j int) bool { return scored[i].Score > scored[j].Score })
	if reranked, err := s.rerank(ctx, req.Query, scored); err == nil {
		scored = reranked
	} else {
		log.Printf("search rerank skipped: %v", err)
	}
	total := len(scored)
	start := (req.Page - 1) * req.PageSize
	if start > len(scored) {
		start = len(scored)
	}
	end := start + req.PageSize
	if end > len(scored) {
		end = len(scored)
	}
	return Response{Query: req.Query, Mode: req.Mode, Page: req.Page, PageSize: req.PageSize, Total: total, Results: scored[start:end], Facets: facets(pages)}, nil
}

func (s Service) rerank(ctx context.Context, query string, results []Result) ([]Result, error) {
	ai := s.Store.Settings().AI
	if strings.TrimSpace(query) == "" || ai.RerankBaseURL == "" || ai.RerankModel == "" || len(results) < 2 {
		return results, nil
	}
	topK := ai.RerankTopK
	if topK <= 0 || topK > len(results) {
		topK = len(results)
	}
	documents := make([]string, topK)
	for i := 0; i < topK; i++ {
		documents[i] = results[i].Title + "\n" + results[i].Snippet
	}
	payload, _ := json.Marshal(map[string]any{
		"model": ai.RerankModel, "query": query, "documents": documents, "top_n": topK,
	})
	endpoint := strings.TrimRight(ai.RerankBaseURL, "/")
	if !strings.HasSuffix(endpoint, "/rerank") {
		endpoint += "/rerank"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if ai.RerankAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+ai.RerankAPIKey)
	}
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("rerank http status %d", resp.StatusCode)
	}
	var decoded struct {
		Results []struct {
			Index          int     `json:"index"`
			RelevanceScore float64 `json:"relevance_score"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	if len(decoded.Results) == 0 {
		return nil, fmt.Errorf("rerank provider returned no results")
	}
	reranked := make([]Result, 0, len(results))
	seen := make(map[int]bool, topK)
	for _, item := range decoded.Results {
		if item.Index < 0 || item.Index >= topK || seen[item.Index] {
			continue
		}
		result := results[item.Index]
		result.Score = item.RelevanceScore
		reranked = append(reranked, result)
		seen[item.Index] = true
	}
	for i := 0; i < topK; i++ {
		if !seen[i] {
			reranked = append(reranked, results[i])
		}
	}
	return append(reranked, results[topK:]...), nil
}

func defaultVersionForModule(m store.Module) string {
	for _, v := range m.AvailableVers {
		if v.IsDefault && v.Status != "archived" {
			return v.DocsVersion
		}
	}
	if strings.TrimSpace(m.DefaultVersion) != "" {
		return m.DefaultVersion
	}
	for _, v := range m.AvailableVers {
		if v.Status != "archived" {
			return v.DocsVersion
		}
	}
	return ""
}

func matchFilters(p store.Page, f Filters) bool {
	return inAny(p.CategoryIDs, f.CategoryIDs) &&
		inValue(p.ModuleKey, f.Modules) &&
		inValue(p.DocsVersion, f.DocsVersions) &&
		inValue(p.EntryType, f.EntryTypes) &&
		inAny(p.Tags, f.Keywords) &&
		inValue(p.OwnerGroup, f.Owners) &&
		inValue(p.Status, f.Status)
}

// queryTokenRe splits a query into searchable tokens: runs of ASCII
// letters/digits and runs of CJK (Han) characters. Plain whitespace splitting
// fails on mixed input like "EventBus怎么用" (no spaces), collapsing it to one
// token that matches nothing; this separates it into "eventbus" + "怎么用".
var queryTokenRe = regexp.MustCompile(`[a-z0-9]+|\p{Han}+`)

func queryTokens(query string) []string {
	var out []string
	for _, tok := range queryTokenRe.FindAllString(strings.ToLower(query), -1) {
		r := []rune(tok)
		// ASCII alnum run: keep whole (e.g. "eventbus").
		if r[0] < 128 || len(r) == 1 {
			out = append(out, tok)
			continue
		}
		// CJK run: emit 2-char shingles so a query like "如何下载插件" matches docs
		// containing "下载"/"插件" without a full word segmenter. Single chars would
		// over-match; whole runs (the previous behaviour) under-match.
		for i := 0; i+1 < len(r); i++ {
			out = append(out, string(r[i:i+2]))
		}
	}
	return out
}

func keywordScore(query string, p store.Page) float64 {
	q := queryTokens(query)
	if len(q) == 0 {
		return 1
	}
	hay := strings.ToLower(p.Title + " " + p.Description + " " + p.ContentText + " " + strings.Join(p.Tags, " "))
	normalizedHay := normalizeText(hay)
	score := 0.0
	for _, term := range q {
		normalizedTerm := normalizeText(term)
		if strings.Contains(strings.ToLower(p.Title), term) {
			score += 3
		}
		if strings.Contains(hay, term) {
			score += 1
			continue
		}
		if normalizedTerm != "" && strings.Contains(normalizedHay, normalizedTerm) {
			score += 0.8
		}
	}
	return score / float64(len(q)*4)
}

// Reindex (re)computes and caches embeddings for every page chunk using the
// configured provider. It powers POST /api/embeddings/reindex and pre-warms the
// cache so semantic search does not pay an embedding call on the hot path.
func (s Service) Reindex(ctx context.Context) (int, error) {
	if err := s.clearEmbeddings(ctx); err != nil {
		return 0, err
	}
	return s.reindexPages(ctx, s.Store.Pages())
}

func (s Service) ReindexModuleVersion(ctx context.Context, moduleKey, docsVersion string) (int, error) {
	prefix := moduleKey + ":" + docsVersion + ":"
	if err := s.DeleteModuleVersionEmbeddings(ctx, moduleKey, docsVersion); err != nil {
		return 0, err
	}
	pages := make([]store.Page, 0)
	for _, p := range s.Store.Pages() {
		if strings.HasPrefix(p.DocID, prefix) {
			pages = append(pages, p)
		}
	}
	return s.reindexPages(ctx, pages)
}

func (s Service) reindexPages(ctx context.Context, pages []store.Page) (int, error) {
	count := 0
	for _, p := range pages {
		for _, chunk := range s.embeddingChunks(p) {
			vec, err := s.Embedder.EmbedText(ctx, chunk.Content)
			if err != nil {
				return count, fmt.Errorf("embed page %s chunk %s: %w", p.DocID, chunk.ID, err)
			}
			if err := s.upsertEmbedding(ctx, p.DocID, chunk.ID, chunk.Content, vec); err != nil {
				return count, err
			}
			count++
		}
	}
	return count, nil
}

// pageSemanticScore returns the best precomputed chunk score for the page.
// Missing embeddings are skipped; indexing is done by Reindex/ReindexModuleVersion
// during document publish or explicit admin maintenance, not on the search path.
func (s Service) pageSemanticScore(ctx context.Context, queryVec []float32, p store.Page, vectors map[string][]float32) (float64, error) {
	best := 0.0
	for _, chunk := range s.embeddingChunks(p) {
		vec, ok := vectors[chunk.ID]
		if !ok {
			continue
		}
		if score := cosine(queryVec, vec); score > best {
			best = score
		}
	}
	return best, nil
}

func (s Service) embeddingBatch(docIDs []string) (map[string][]float32, error) {
	out := make(map[string][]float32, len(docIDs))
	for _, p := range s.Store.Pages() {
		for _, chunk := range s.embeddingChunks(p) {
			if vector, ok := s.Store.Embedding(chunk.ID); ok {
				out[chunk.ID] = vector
			}
		}
	}
	return out, nil
}

func (s Service) upsertEmbedding(ctx context.Context, docID, chunkID, content string, vector []float32) error {
	if s.Vectors != nil {
		return s.Vectors.UpsertChunk(ctx, docID, chunkID, content, vector)
	}
	s.Store.SetEmbedding(chunkID, vector)
	return nil
}

func (s Service) clearEmbeddings(ctx context.Context) error {
	if s.Vectors != nil {
		return s.Vectors.Clear(ctx)
	}
	s.Store.ClearEmbeddings()
	return nil
}

func (s Service) EmbeddingCount(ctx context.Context) (int, error) {
	if s.Vectors != nil {
		return s.Vectors.Count(ctx)
	}
	return s.Store.EmbeddingCount(), nil
}

func (s Service) DeleteModuleVersionEmbeddings(ctx context.Context, moduleKey, docsVersion string) error {
	prefix := moduleKey + ":" + docsVersion + ":"
	if s.Vectors != nil {
		return s.Vectors.DeletePrefix(ctx, prefix)
	}
	return nil
}

type embeddingChunk struct {
	ID      string
	Content string
}

func (s Service) embeddingChunks(p store.Page) []embeddingChunk {
	text := plainText(strings.TrimSpace(p.ContentText))
	prefix := strings.TrimSpace(p.Title + "\n" + p.Description)
	if text == "" {
		text = prefix
	}
	if strings.TrimSpace(text) == "" {
		return nil
	}
	limit := s.embeddingInputLimit()
	overlap := s.Store.Settings().AI.ChunkOverlap
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= limit {
		overlap = limit / 5
	}
	prefixRunes := []rune(prefix)
	if len(prefixRunes) > limit/4 {
		prefix = string(prefixRunes[:limit/4])
		prefixRunes = []rune(prefix)
	}
	bodyLimit := limit
	if prefix != "" {
		bodyLimit = limit - len(prefixRunes) - 1
		if bodyLimit <= 0 {
			bodyLimit = limit
			prefix = ""
		}
	}
	bodyRunes := []rune(text)
	step := bodyLimit - overlap
	if step <= 0 {
		step = bodyLimit
	}
	chunks := make([]embeddingChunk, 0, (len(bodyRunes)/step)+1)
	for start, index := 0, 0; start < len(bodyRunes) && index < maxEmbeddingChunksPerDoc; index++ {
		end := start + bodyLimit
		if end > len(bodyRunes) {
			end = len(bodyRunes)
		}
		content := strings.TrimSpace(string(bodyRunes[start:end]))
		if prefix != "" && content != prefix {
			content = strings.TrimSpace(prefix + "\n" + content)
		}
		content = s.truncateEmbeddingInput(content)
		if content != "" {
			chunks = append(chunks, embeddingChunk{
				ID:      fmt.Sprintf("%s#chunk-%04d", p.DocID, index),
				Content: content,
			})
		}
		if end == len(bodyRunes) {
			break
		}
		start += step
	}
	return chunks
}

func (s Service) embeddingInputLimit() int {
	limit := s.Store.Settings().AI.ChunkSize
	if limit <= 0 {
		return defaultEmbeddingInputRunes
	}
	return limit
}

func (s Service) truncateEmbeddingInput(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	limit := s.embeddingInputLimit()
	if limit <= 0 {
		limit = defaultEmbeddingInputRunes
	}
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit])
}

// cosine returns a 0..1 similarity (cosine distance shifted into [0,1]).
func cosine(a, b []float32) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, an, bn float64
	for i := range a {
		dot += float64(a[i] * b[i])
		an += float64(a[i] * a[i])
		bn += float64(b[i] * b[i])
	}
	if an == 0 || bn == 0 {
		return 0
	}
	return (dot/math.Sqrt(an*bn) + 1) / 2
}

// Markdown-stripping patterns for plainText. Applied in order so links/images
// resolve to their text before stray emphasis markers are removed.
var (
	mdCodeFence   = regexp.MustCompile("(?s)```.*?```")
	mdImage       = regexp.MustCompile(`!\[([^\]]*)\]\([^)]*\)`)
	mdLink        = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	mdInlineCode  = regexp.MustCompile("`([^`]*)`")
	mdTableSep    = regexp.MustCompile(`(?m)^\s*\|?[\s:|-]{3,}\|?\s*$`)
	mdHeadingHash = regexp.MustCompile(`(?m)(^|\s)#{1,6}\s+`)
	mdBlockquote  = regexp.MustCompile(`(?m)^\s{0,3}>\s?`)
	mdListMarker  = regexp.MustCompile(`(?m)^\s{0,3}([-*+]|\d+\.)\s+`)
	mdHTMLTag     = regexp.MustCompile(`<[^>]+>`)
	mdEmphasis    = regexp.MustCompile(`[*_~]{1,3}`)
	mdWhitespace  = regexp.MustCompile(`\s+`)
)

// plainText strips common Markdown syntax so search titles and snippets read as
// clean prose instead of raw markup (** , #, [text](url), ``` ... ```, tables).
// It is display-only; keyword scoring and embeddings still use the raw content.
func plainText(s string) string {
	if s == "" {
		return ""
	}
	s = mdCodeFence.ReplaceAllString(s, " ")
	s = mdImage.ReplaceAllString(s, "$1")
	s = mdLink.ReplaceAllString(s, "$1")
	s = mdInlineCode.ReplaceAllString(s, "$1")
	s = mdTableSep.ReplaceAllString(s, " ")
	s = mdHeadingHash.ReplaceAllString(s, "$1")
	s = mdBlockquote.ReplaceAllString(s, "")
	s = mdListMarker.ReplaceAllString(s, "")
	s = mdHTMLTag.ReplaceAllString(s, "")
	s = mdEmphasis.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "|", " ")
	s = mdWhitespace.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// snippet returns a rune-safe excerpt centered on the first matched term, with
// surrounding context before/after and ellipses, so callers can highlight the
// matched keywords. It never splits a multibyte (e.g. CJK) character.
func snippet(query, content string) string {
	runes := []rune(strings.TrimSpace(content))
	const window = 140
	if len(runes) <= window {
		return string(runes)
	}
	lower := strings.ToLower(string(runes))
	matchByte := -1
	for _, term := range matchTerms(query) {
		if idx := strings.Index(lower, term); idx >= 0 {
			matchByte = idx
			break
		}
	}
	center := 0
	if matchByte >= 0 {
		center = len([]rune(lower[:matchByte]))
	}
	start := center - 50
	if start < 0 {
		start = 0
	}
	end := start + window
	if end > len(runes) {
		end = len(runes)
		start = end - window
		if start < 0 {
			start = 0
		}
	}
	out := strings.TrimSpace(string(runes[start:end]))
	if start > 0 {
		out = "…" + out
	}
	if end < len(runes) {
		out = out + "…"
	}
	return out
}

// matchTerms extracts the distinct lowercase query tokens used both for snippet
// centering and client-side keyword highlighting.
func matchTerms(query string) []string {
	seen := map[string]bool{}
	var out []string
	for _, f := range queryTokens(query) {
		if f == "" || seen[f] {
			continue
		}
		seen[f] = true
		out = append(out, f)
	}
	return out
}

func normalizeText(s string) string {
	replacer := strings.NewReplacer(" ", "", "\t", "", "\n", "", "\r", "", "，", "", "。", "", "、", "", "；", "", "：", "", ",", "", ".", "", ";", "", ":", "", "/", "", "-", "", "_", "", "与", "", "和", "")
	return replacer.Replace(strings.ToLower(s))
}

func facets(pages []store.Page) map[string]map[string]int {
	out := map[string]map[string]int{
		"modules": {}, "docs_versions": {}, "entry_types": {}, "keywords": {}, "owners": {}, "status": {}, "categories": {},
	}
	for _, p := range pages {
		out["modules"][p.ModuleKey]++
		out["docs_versions"][p.DocsVersion]++
		out["entry_types"][p.EntryType]++
		out["owners"][p.OwnerGroup]++
		out["status"][p.Status]++
		for _, c := range p.CategoryIDs {
			out["categories"][c]++
		}
		for _, tag := range p.Tags {
			out["keywords"][tag]++
		}
	}
	return out
}

func inValue(value string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, x := range allowed {
		if strings.EqualFold(value, x) {
			return true
		}
	}
	return false
}

func inAny(values, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, a := range allowed {
		for _, v := range values {
			if strings.EqualFold(v, a) {
				return true
			}
		}
	}
	return false
}
