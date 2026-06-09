package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"modex/backend/internal/auth"
	"modex/backend/internal/embedding"
	"modex/backend/internal/search"
	"modex/backend/internal/store"
)

type Server struct {
	store  *store.Store
	auth   *auth.Service
	search search.Service
}

func New(st *store.Store) *Server {
	provider := embedding.FromEnv()
	authSvc := auth.NewService(auth.FromEnv())
	return &Server{
		store: st,
		auth:  authSvc,
		search: search.Service{
			Store:          st,
			Embedder:       provider,
			KeywordWeight:  envFloat("HYBRID_KEYWORD_WEIGHT", 0.6),
			SemanticWeight: envFloat("HYBRID_SEMANTIC_WEIGHT", 0.4),
		},
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/api/auth/me", s.handleMe)
	mux.HandleFunc("/api/auth/mock-login", s.handleMockLogin)
	mux.HandleFunc("/api/auth/login", s.handleLogin)
	mux.HandleFunc("/api/auth/callback", s.handleCallback)
	mux.HandleFunc("/api/auth/logout", s.handleLogout)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/categories/tree", s.handleCategories)
	mux.HandleFunc("/api/modules", s.handleModules)
	mux.HandleFunc("/api/modules/", s.handleModuleRoutes)
	mux.HandleFunc("/api/docs/page/", s.handleDocPage)
	mux.HandleFunc("/api/docs/", s.handleDocRoutes)
	mux.HandleFunc("/api/search", s.handleSearch)
	mux.HandleFunc("/api/search/facets", s.handleFacets)
	mux.HandleFunc("/api/search/reindex", s.handleAccepted("search_reindex_started"))
	mux.HandleFunc("/api/embeddings/embed-text", s.handleEmbedText)
	mux.HandleFunc("/api/embeddings/reindex", s.handleAccepted("embedding_reindex_started"))
	mux.HandleFunc("/api/deploy", s.handleDeploy)
	mux.HandleFunc("/api/analytics/page-view", s.handlePageView)
	mux.HandleFunc("/api/analytics/read-progress", s.handleReadProgress)
	mux.HandleFunc("/api/admin/releases", s.handleReleases)
	mux.HandleFunc("/api/admin/releases/", s.handleReleaseRoutes)
	mux.HandleFunc("/api/admin/analytics/pages", s.handlePageAnalytics)
	mux.HandleFunc("/api/admin/analytics/search", s.handleSearchLogs)
	mux.HandleFunc("/api/admin/analytics/mcp", s.handleMCPLogs)
	mux.HandleFunc("/api/mcp/log", s.handleMCPLog)
	mux.HandleFunc("/api/admin/categories", s.handleAdminCategories)
	mux.HandleFunc("/api/admin/categories/", s.handleAdminCategoryByID)
	mux.HandleFunc("/api/admin/modules", s.handleAdminModules)
	mux.HandleFunc("/api/admin/modules/", s.handleAdminModuleRoutes)
	mux.HandleFunc("/api/admin/entries/", s.handleAdminEntryByID)
	mux.HandleFunc("/api/admin/", s.handleAdminAccepted)
	return s.cors(recoverer(mux))
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "modex-api"})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	if user, ok := s.currentUser(r); ok {
		writeJSON(w, http.StatusOK, user)
		return
	}
	writeError(w, http.StatusUnauthorized, "unauthorized", "not logged in")
}

func (s *Server) handleMockLogin(w http.ResponseWriter, r *http.Request) {
	if s.auth.Config().Mode == "oidc" {
		writeError(w, http.StatusForbidden, "mock_login_disabled", "mock login is disabled when AUTH_MODE=oidc")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": s.store.CurrentUser(), "session": "mock-session"})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	loginURL, err := s.auth.BeginLogin(w)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "oidc_not_configured", err.Error())
		return
	}
	http.Redirect(w, r, loginURL, http.StatusFound)
}

func (s *Server) handleCallback(w http.ResponseWriter, r *http.Request) {
	if _, err := s.auth.CompleteLogin(r.Context(), r, w); err != nil {
		writeError(w, http.StatusBadRequest, "oidc_callback_failed", err.Error())
		return
	}
	http.Redirect(w, r, s.auth.Config().FrontendBaseURL, http.StatusFound)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.auth.Logout(w, r)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	cfg := s.auth.Config()
	writeJSON(w, http.StatusOK, map[string]any{
		"auth_mode":          cfg.Mode,
		"oidc_login_enabled": cfg.LoginReady(),
		"login_url":          cfg.AppBaseURL + "/api/auth/login",
		"frontend_base_url":  cfg.FrontendBaseURL,
	})
}

func (s *Server) handleCategories(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.CategoryTree())
}

func (s *Server) handleModules(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Modules(r.URL.Query().Get("category_id"), r.URL.Query().Get("keyword")))
}

func (s *Server) handleModuleRoutes(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(strings.TrimPrefix(r.URL.Path, "/api/modules/"))
	if len(parts) == 0 {
		writeError(w, http.StatusNotFound, "not_found", "module route not found")
		return
	}
	moduleKey := parts[0]
	if len(parts) == 1 || (len(parts) == 2 && parts[1] == "info") {
		m, err := s.store.Module(moduleKey)
		writeResult(w, m, err)
		return
	}
	if len(parts) == 2 && parts[1] == "versions" {
		writeJSON(w, http.StatusOK, s.store.Versions(moduleKey))
		return
	}
	if len(parts) == 4 && parts[1] == "versions" && parts[3] == "entries" {
		writeJSON(w, http.StatusOK, s.store.Entries(moduleKey, parts[2]))
		return
	}
	writeError(w, http.StatusNotFound, "not_found", "module route not found")
}

func (s *Server) handleDocRoutes(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(strings.TrimPrefix(r.URL.Path, "/api/docs/"))
	if len(parts) < 1 {
		writeError(w, http.StatusNotFound, "not_found", "docs route not found")
		return
	}
	module, err := s.store.Module(parts[0])
	if err != nil {
		writeResult(w, module, err)
		return
	}
	if len(parts) == 1 {
		writeJSON(w, http.StatusOK, map[string]any{"module": module, "redirect_to": "/docs/" + module.ModuleKey + "/" + module.DefaultVersion})
		return
	}
	if len(parts) == 2 {
		writeJSON(w, http.StatusOK, map[string]any{"module": module, "version": parts[1], "entries": s.store.Entries(module.ModuleKey, parts[1])})
		return
	}
	if len(parts) >= 3 {
		if len(parts) == 4 && parts[3] == "nav" {
			writeJSON(w, http.StatusOK, []map[string]string{{"title": "概览", "href": "#overview"}, {"title": "正文", "href": "#content"}, {"title": "元数据", "href": "#metadata"}})
			return
		}
		page, err := s.store.PageByRoute(module.ModuleKey, parts[1], parts[2])
		writeResult(w, page, err)
		return
	}
}

func (s *Server) handleDocPage(w http.ResponseWriter, r *http.Request) {
	docID := strings.TrimPrefix(r.URL.Path, "/api/docs/page/")
	page, err := s.store.Page(docID)
	writeResult(w, page, err)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	var req search.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	resp, err := s.search.Search(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "search_failed", err.Error())
		return
	}
	filters, _ := json.Marshal(req.Filters)
	user, _ := s.currentUser(r)
	s.store.AddSearchLog(store.SearchLog{ID: fmt.Sprintf("sl-%d", time.Now().UnixNano()), UserID: user.ID, Query: req.Query, Mode: string(resp.Mode), FiltersJSON: string(filters), ResultCount: resp.Total, SearchedAt: time.Now().UTC()})
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleFacets(w http.ResponseWriter, r *http.Request) {
	resp, _ := s.search.Search(r.Context(), search.Request{Mode: search.ModeKeyword, PageSize: 1})
	writeJSON(w, http.StatusOK, resp.Facets)
}

func (s *Server) handleEmbedText(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	vec, err := s.search.Embedder.EmbedText(r.Context(), req.Text)
	if err != nil {
		writeError(w, http.StatusBadGateway, "embedding_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"provider": s.search.Embedder.Name(), "dimension": len(vec), "embedding": vec})
}

func (s *Server) handleDeploy(w http.ResponseWriter, r *http.Request) {
	n, _ := io.Copy(io.Discard, r.Body)
	writeJSON(w, http.StatusAccepted, map[string]any{"status": "accepted", "bytes_received": n})
}

func (s *Server) handleReleases(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Releases())
}

func (s *Server) handlePageAnalytics(w http.ResponseWriter, r *http.Request) {
	stats := s.store.PageAnalytics()
	var totalPV, totalReads7d int
	for _, st := range stats {
		totalPV += st.PV
		totalReads7d += st.Reads7d
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"popular_pages": stats,
		"total_pv":      totalPV,
		"reads_7d":      totalReads7d,
		"events":        []string{"docs_page_view", "docs_search_result_click"},
	})
}

func (s *Server) handlePageView(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	var req struct {
		DocID       string  `json:"doc_id"`
		SessionID   string  `json:"session_id"`
		Duration    int     `json:"duration_seconds"`
		ScrollDepth float64 `json:"scroll_depth"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if strings.TrimSpace(req.DocID) == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "doc_id is required")
		return
	}
	user, _ := s.currentUser(r)
	pv := s.store.RecordPageView(store.PageView{
		DocID: req.DocID, UserID: user.ID, SessionID: req.SessionID,
		DurationSeconds: req.Duration, ScrollDepth: req.ScrollDepth,
	})
	writeJSON(w, http.StatusAccepted, map[string]any{"status": "recorded", "view_id": pv.ID})
}

func (s *Server) handleReadProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	var req struct {
		DocID       string  `json:"doc_id"`
		SessionID   string  `json:"session_id"`
		Duration    int     `json:"duration_seconds"`
		ScrollDepth float64 `json:"scroll_depth"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if strings.TrimSpace(req.DocID) == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "doc_id is required")
		return
	}
	s.store.RecordReadProgress(req.DocID, req.SessionID, req.Duration, req.ScrollDepth)
	writeJSON(w, http.StatusAccepted, map[string]any{"status": "recorded"})
}

func (s *Server) handleSearchLogs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.SearchLogs())
}

func (s *Server) handleMCPLogs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.MCPLogs())
}

func (s *Server) handleMCPLog(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ToolName    string `json:"tool_name"`
		Query       string `json:"query"`
		InputJSON   string `json:"input_json"`
		ResultCount int    `json:"result_count"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	user, _ := s.currentUser(r)
	s.store.AddMCPLog(store.MCPLog{ID: fmt.Sprintf("ml-%d", time.Now().UnixNano()), ToolName: req.ToolName, UserID: user.ID, Query: req.Query, InputJSON: req.InputJSON, ResultCount: req.ResultCount, CreatedAt: time.Now().UTC()})
	writeJSON(w, http.StatusAccepted, map[string]any{"status": "logged"})
}

func (s *Server) currentUser(r *http.Request) (store.User, bool) {
	if user, ok := s.auth.CurrentUser(r); ok {
		return user, true
	}
	if s.auth.Config().Mode != "oidc" {
		return s.store.CurrentUser(), true
	}
	return store.User{}, false
}

func (s *Server) handleAdminCategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusOK, s.store.CategoryTree())
		return
	}
	var c store.Category
	if err := decodeBody(r, &c); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	created, err := s.store.CreateCategory(c)
	writeMutation(w, created, http.StatusCreated, err)
}

func (s *Server) handleAdminCategoryByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/admin/categories/")
	switch r.Method {
	case http.MethodPut:
		var c store.Category
		if err := decodeBody(r, &c); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		updated, err := s.store.UpdateCategory(id, c)
		writeMutation(w, updated, http.StatusOK, err)
	case http.MethodDelete:
		if err := s.store.DeleteCategory(id); err != nil {
			writeResult(w, nil, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "deleted", "id": id})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use PUT or DELETE")
	}
}

func (s *Server) handleAdminModules(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusOK, s.store.Modules("", ""))
		return
	}
	var m store.Module
	if err := decodeBody(r, &m); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	created, err := s.store.CreateModule(m)
	writeMutation(w, created, http.StatusCreated, err)
}

func (s *Server) handleAdminModuleRoutes(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(strings.TrimPrefix(r.URL.Path, "/api/admin/modules/"))
	if len(parts) == 0 {
		writeError(w, http.StatusNotFound, "not_found", "admin module route not found")
		return
	}
	moduleKey := parts[0]
	switch {
	case len(parts) == 1 && r.Method == http.MethodPut:
		var m store.Module
		if err := decodeBody(r, &m); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		updated, err := s.store.UpdateModule(moduleKey, m)
		writeMutation(w, updated, http.StatusOK, err)
	case len(parts) == 2 && parts[1] == "versions" && r.Method == http.MethodPost:
		var v store.Version
		if err := decodeBody(r, &v); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		created, err := s.store.CreateVersion(moduleKey, v)
		writeMutation(w, created, http.StatusCreated, err)
	case len(parts) == 3 && parts[1] == "versions" && r.Method == http.MethodPut:
		var v store.Version
		if err := decodeBody(r, &v); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		updated, err := s.store.UpdateVersion(moduleKey, parts[2], v)
		writeMutation(w, updated, http.StatusOK, err)
	case len(parts) == 4 && parts[1] == "versions" && parts[3] == "entries" && r.Method == http.MethodPost:
		var e store.Entry
		if err := decodeBody(r, &e); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		created, err := s.store.CreateEntry(moduleKey, parts[2], e)
		writeMutation(w, created, http.StatusCreated, err)
	default:
		writeError(w, http.StatusNotFound, "not_found", "admin module route not found")
	}
}

func (s *Server) handleAdminEntryByID(w http.ResponseWriter, r *http.Request) {
	entryID := strings.TrimPrefix(r.URL.Path, "/api/admin/entries/")
	switch r.Method {
	case http.MethodPut:
		var e store.Entry
		if err := decodeBody(r, &e); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		updated, err := s.store.UpdateEntry(entryID, e)
		writeMutation(w, updated, http.StatusOK, err)
	case http.MethodDelete:
		if err := s.store.DeleteEntry(entryID); err != nil {
			writeResult(w, nil, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "deleted", "id": entryID})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use PUT or DELETE")
	}
}

func (s *Server) handleReleaseRoutes(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(strings.TrimPrefix(r.URL.Path, "/api/admin/releases/"))
	if len(parts) == 0 {
		writeError(w, http.StatusNotFound, "not_found", "release route not found")
		return
	}
	releaseID := parts[0]
	if len(parts) == 2 && parts[1] == "rollback" && r.Method == http.MethodPost {
		rel, err := s.store.RollbackRelease(releaseID)
		writeResult(w, rel, err)
		return
	}
	rel, err := s.store.Release(releaseID)
	writeResult(w, rel, err)
}

func (s *Server) handleAdminAccepted(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusAccepted, map[string]any{"status": "accepted", "note": "MVP admin mutation endpoint placeholder"})
}

func decodeBody(r *http.Request, v any) error {
	return json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(v)
}

func writeMutation(w http.ResponseWriter, v any, successStatus int, err error) {
	if err != nil {
		writeResult(w, nil, err)
		return
	}
	writeJSON(w, successStatus, v)
}

func (s *Server) handleAccepted(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusAccepted, map[string]any{"status": status})
	}
}

func writeResult(w http.ResponseWriter, v any, err error) {
	if err == nil {
		writeJSON(w, http.StatusOK, v)
		return
	}
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "resource not found")
		return
	}
	if errors.Is(err, store.ErrInvalid) {
		writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
		return
	}
	if errors.Is(err, store.ErrConflict) {
		writeError(w, http.StatusConflict, "conflict", err.Error())
		return
	}
	writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func splitPath(path string) []string {
	var out []string
	for _, p := range strings.Split(path, "/") {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if s.originAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		} else if origin == "" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) originAllowed(origin string) bool {
	if origin == "" {
		return false
	}
	for _, allowed := range s.auth.Config().CORSAllowOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				writeError(w, http.StatusInternalServerError, "panic", fmt.Sprint(rec))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func envFloat(key string, fallback float64) float64 {
	v, err := strconv.ParseFloat(os.Getenv(key), 64)
	if err != nil || v == 0 {
		return fallback
	}
	return v
}
