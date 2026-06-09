package search

import (
	"context"
	"math"
	"sort"
	"strings"

	"modex/backend/internal/embedding"
	"modex/backend/internal/store"
)

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

type Service struct {
	Store          *store.Store
	Embedder       embedding.Provider
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
	for _, m := range s.Store.Modules("", "") {
		breadcrumbs[m.ModuleKey] = m.CategoryPath
	}
	terms := matchTerms(req.Query)
	// Semantic and hybrid modes need a query embedding; keyword mode skips it to
	// avoid an unnecessary embedding-provider call.
	var queryVec []float32
	if req.Mode != ModeKeyword && strings.TrimSpace(req.Query) != "" {
		queryVec, _ = s.Embedder.EmbedText(ctx, req.Query)
	}
	var scored []Result
	for _, p := range pages {
		if !matchFilters(p, req.Filters) {
			continue
		}
		kScore := keywordScore(req.Query, p)
		var sScore float64
		if len(queryVec) > 0 {
			sScore = cosine(queryVec, s.pageVector(ctx, p))
		}
		var final float64
		switch req.Mode {
		case ModeKeyword:
			final = kScore
		case ModeSemantic:
			final = sScore
		default:
			final = kScore*kw + sScore*sw
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
			DocID: p.DocID, Title: p.Title, Snippet: snippet(req.Query, p.ContentText), Path: p.Path,
			Score: final, SearchMode: req.Mode, ModuleKey: p.ModuleKey, ModuleName: p.ModuleName,
			DocsVersion: p.DocsVersion, PackageVersion: p.PackageVersion, EntryType: p.EntryType,
			OwnerGroup: p.OwnerGroup, Status: p.Status, UpdatedAt: p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Keywords: p.Tags, Breadcrumb: breadcrumb, EntryKey: p.EntryKey, MatchTerms: terms,
		})
	}
	sort.Slice(scored, func(i, j int) bool { return scored[i].Score > scored[j].Score })
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

func matchFilters(p store.Page, f Filters) bool {
	return inAny(p.CategoryIDs, f.CategoryIDs) &&
		inValue(p.ModuleKey, f.Modules) &&
		inValue(p.DocsVersion, f.DocsVersions) &&
		inValue(p.EntryType, f.EntryTypes) &&
		inAny(p.Tags, f.Keywords) &&
		inValue(p.OwnerGroup, f.Owners) &&
		inValue(p.Status, f.Status)
}

func keywordScore(query string, p store.Page) float64 {
	q := strings.Fields(strings.ToLower(query))
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

// Reindex (re)computes and caches an embedding for every page using the
// configured provider. It powers POST /api/embeddings/reindex and pre-warms the
// cache so semantic search does not pay an embedding call on the hot path.
func (s Service) Reindex(ctx context.Context) (int, error) {
	s.Store.ClearEmbeddings()
	pages := s.Store.Pages()
	count := 0
	for _, p := range pages {
		text := embedText(p)
		if text == "" {
			continue
		}
		vec, err := s.Embedder.EmbedText(ctx, text)
		if err != nil {
			return count, err
		}
		s.Store.SetEmbedding(p.DocID, vec)
		count++
	}
	return count, nil
}

// pageVector returns the page's embedding from cache, computing and caching it
// on demand the first time it is needed.
func (s Service) pageVector(ctx context.Context, p store.Page) []float32 {
	if vec, ok := s.Store.Embedding(p.DocID); ok {
		return vec
	}
	text := embedText(p)
	if text == "" {
		return nil
	}
	vec, err := s.Embedder.EmbedText(ctx, text)
	if err != nil {
		return nil
	}
	s.Store.SetEmbedding(p.DocID, vec)
	return vec
}

func embedText(p store.Page) string {
	return strings.TrimSpace(p.Title + "\n" + p.Description + "\n" + p.ContentText)
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
	for _, f := range strings.Fields(strings.ToLower(query)) {
		f = strings.Trim(f, ",.;:!?，。、；：！？\"'")
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
