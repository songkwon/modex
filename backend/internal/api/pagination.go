package api

import (
	"net/http"
	"strconv"
	"strings"
)

// pageResult is the envelope returned by admin list endpoints when the client
// opts into server-side pagination via ?page= / ?limit=. Endpoints stay
// backward-compatible: without those params they return the plain array so
// existing callers (dropdowns, homepage, identity derivation) keep working.
type pageResult[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

// wantsPage reports whether the request opted into pagination.
func wantsPage(r *http.Request) bool {
	q := r.URL.Query()
	return q.Has("page") || q.Has("limit")
}

func pageParams(r *http.Request) (page, limit int) {
	page = atoiOr(r.URL.Query().Get("page"), 1)
	limit = atoiOr(r.URL.Query().Get("limit"), 20)
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	return page, limit
}

func atoiOr(s string, def int) int {
	if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
		return n
	}
	return def
}

// paginate slices items for the given 1-based page and returns the envelope.
func paginate[T any](items []T, page, limit int) pageResult[T] {
	total := len(items)
	start := (page - 1) * limit
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	out := append([]T{}, items[start:end]...)
	return pageResult[T]{Items: out, Total: total, Page: page, Limit: limit}
}

// keywordOf returns the normalized (trimmed, lowercased) keyword query param.
func keywordOf(r *http.Request) string {
	return strings.ToLower(strings.TrimSpace(r.URL.Query().Get("keyword")))
}

func containsFold(haystack, needleLower string) bool {
	if needleLower == "" {
		return true
	}
	return strings.Contains(strings.ToLower(haystack), needleLower)
}
