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
	queryVec, _ := s.Embedder.EmbedText(ctx, req.Query)
	var scored []Result
	for _, p := range pages {
		if !matchFilters(p, req.Filters) {
			continue
		}
		kScore := keywordScore(req.Query, p)
		sScore := semanticScore(queryVec, p.ContentText)
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
		scored = append(scored, Result{
			DocID: p.DocID, Title: p.Title, Snippet: snippet(req.Query, p.ContentText), Path: p.Path,
			Score: final, SearchMode: req.Mode, ModuleKey: p.ModuleKey, ModuleName: p.ModuleName,
			DocsVersion: p.DocsVersion, PackageVersion: p.PackageVersion, EntryType: p.EntryType,
			OwnerGroup: p.OwnerGroup, Status: p.Status, UpdatedAt: p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"), Keywords: p.Tags,
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
	score := 0.0
	for _, term := range q {
		if strings.Contains(strings.ToLower(p.Title), term) {
			score += 3
		}
		if strings.Contains(hay, term) {
			score += 1
		}
	}
	return score / float64(len(q)*4)
}

func semanticScore(queryVec []float32, text string) float64 {
	if len(queryVec) == 0 || text == "" {
		return 0
	}
	// Deterministic MVP approximation: compare hashed text vector to query vector.
	mock := embedding.MockProvider{Dim: len(queryVec)}
	vec, _ := mock.EmbedText(context.Background(), text)
	var dot, qn, vn float64
	for i := range queryVec {
		dot += float64(queryVec[i] * vec[i])
		qn += float64(queryVec[i] * queryVec[i])
		vn += float64(vec[i] * vec[i])
	}
	if qn == 0 || vn == 0 {
		return 0
	}
	return (dot/math.Sqrt(qn*vn) + 1) / 2
}

func snippet(query, content string) string {
	content = strings.TrimSpace(content)
	if len(content) <= 160 {
		return content
	}
	q := strings.Fields(strings.ToLower(query))
	lower := strings.ToLower(content)
	pos := 0
	if len(q) > 0 {
		if idx := strings.Index(lower, q[0]); idx > 0 {
			pos = idx
		}
	}
	start := pos - 40
	if start < 0 {
		start = 0
	}
	end := start + 160
	if end > len(content) {
		end = len(content)
	}
	return strings.TrimSpace(content[start:end])
}

func facets(pages []store.Page) map[string]map[string]int {
	out := map[string]map[string]int{
		"modules": {}, "docs_versions": {}, "entry_types": {}, "keywords": {}, "owners": {}, "status": {},
	}
	for _, p := range pages {
		out["modules"][p.ModuleKey]++
		out["docs_versions"][p.DocsVersion]++
		out["entry_types"][p.EntryType]++
		out["owners"][p.OwnerGroup]++
		out["status"][p.Status]++
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
