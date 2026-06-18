package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"modex/backend/internal/auth"
	"modex/backend/internal/deploy"
	"modex/backend/internal/embedding"
	"modex/backend/internal/search"
	"modex/backend/internal/store"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Server struct {
	store       *store.Store
	auth        *auth.Service
	search      search.Service
	minioClient *minio.Client
	minioBucket string
}

func New(st *store.Store) *Server {
	return NewWithVectorStore(st, nil)
}

func NewWithVectorStore(st *store.Store, vectors search.VectorStore) *Server {
	provider := embedding.SettingsProvider{Load: func() embedding.Settings {
		ai := st.Settings().AI
		return embedding.Settings{
			BaseURL: ai.EmbeddingBaseURL,
			Model:   ai.EmbeddingModel,
			APIKey:  ai.EmbeddingAPIKey,
			Dim:     ai.EmbeddingDim,
		}
	}}
	authSvc := auth.NewService(auth.FromEnv())
	s := &Server{
		store: st,
		auth:  authSvc,
		search: search.Service{
			Store:          st,
			Embedder:       provider,
			Vectors:        vectors,
			KeywordWeight:  envFloat("HYBRID_KEYWORD_WEIGHT", 0.6),
			SemanticWeight: envFloat("HYBRID_SEMANTIC_WEIGHT", 0.4),
		},
	}
	// init MinIO for real site file storage (upload SiteFiles from deploys)
	if endpoint := os.Getenv("MINIO_ENDPOINT"); endpoint != "" {
		accessKey := os.Getenv("MINIO_ROOT_USER")
		secretKey := os.Getenv("MINIO_ROOT_PASSWORD")
		secure := strings.HasPrefix(strings.ToLower(endpoint), "https://")
		// minio-go expects a bare host:port; strip any scheme/trailing slash so
		// MINIO_ENDPOINT=http://minio:9000 doesn't fail init ("Endpoint url
		// cannot have fully qualified paths"), which would silently drop us to
		// the in-memory fallback and never create the bucket.
		host := endpoint
		if i := strings.Index(host, "://"); i >= 0 {
			host = host[i+3:]
		}
		host = strings.TrimRight(host, "/")
		client, err := minio.New(host, &minio.Options{
			Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
			Secure: secure,
		})
		if err == nil {
			s.minioClient = client
			s.minioBucket = os.Getenv("MINIO_BUCKET")
			if s.minioBucket == "" {
				s.minioBucket = "modex"
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			exists, bucketErr := client.BucketExists(ctx, s.minioBucket)
			if bucketErr == nil && !exists {
				bucketErr = client.MakeBucket(ctx, s.minioBucket, minio.MakeBucketOptions{})
			}
			if bucketErr != nil {
				log.Printf("minio bucket init failed: %v", bucketErr)
				s.minioClient = nil
			}
		} else {
			log.Printf("minio client init failed: %v", err)
		}
	}
	if s.minioClient != nil && s.migrateSiteAssetsToMinIO() {
		s.store.ClearSiteAssets()
	}
	if vectors != nil {
		// A legacy snapshot may still contain vectors from an older release.
		st.ClearEmbeddings()
	}
	return s
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/api/auth/me", s.handleMe)
	mux.HandleFunc("/api/me/mcp-token", s.handleMeMCPToken)
	mux.HandleFunc("/api/auth/mock-login", s.handleMockLogin)
	mux.HandleFunc("/api/auth/login", s.handleLogin)
	mux.HandleFunc("/api/auth/callback", s.handleCallback)
	mux.HandleFunc("/api/auth/logout", s.handleLogout)
	mux.HandleFunc("/.well-known/oauth-authorization-server", s.handleOAuthMetadata)
	mux.HandleFunc("/oauth/authorize", s.handleOAuthAuthorize)
	mux.HandleFunc("/oauth/token", s.handleOAuthToken)
	mux.HandleFunc("/oauth/revoke", s.handleOAuthRevoke)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/categories/tree", s.handleCategories)
	mux.HandleFunc("/api/modules", s.handleModules)
	mux.HandleFunc("/api/modules/", s.handleModuleRoutes)
	mux.HandleFunc("/api/docs/page/", s.handleDocPage)
	mux.HandleFunc("/api/docs/", s.handleDocRoutes)
	mux.HandleFunc("/api/search", s.handleSearch)
	mux.HandleFunc("/api/ask", s.handleAsk)
	mux.HandleFunc("/api/search/facets", s.handleFacets)
	mux.HandleFunc("/api/search/reindex", s.handleSearchReindex)
	mux.HandleFunc("/api/embeddings/embed-text", s.handleEmbedText)
	mux.HandleFunc("/api/embeddings/reindex", s.handleEmbeddingReindex)
	mux.HandleFunc("/api/deploy", s.handleDeploy)
	mux.HandleFunc("/api/analytics/feedback", s.handleDocFeedback)
	mux.HandleFunc("/api/analytics/doc", s.handleDocAnalytics)
	mux.HandleFunc("/api/admin/releases", s.handleReleases)
	mux.HandleFunc("/api/admin/releases/", s.handleReleaseRoutes)
	mux.HandleFunc("/api/admin/analytics/feedback", s.handleDocFeedbackLogs)
	mux.HandleFunc("/api/admin/analytics/search", s.handleSearchLogs)
	mux.HandleFunc("/api/admin/analytics/mcp", s.handleMCPLogs)
	mux.HandleFunc("/api/admin/analytics/pages", http.NotFound)
	mux.HandleFunc("/api/mcp/log", s.handleMCPLog)
	mux.HandleFunc("/api/mcp/dist", s.handleMcpDist)
	mux.HandleFunc("/api/mcp/dist/", s.handleMcpDist)
	mux.HandleFunc("/.well-known/agent-skills/index.json", s.handleSkillDiscovery)
	mux.HandleFunc("/.well-known/skills/index.json", s.handleSkillDiscovery)
	mux.HandleFunc("/api/admin/settings/models", s.handleAdminModels)
	mux.HandleFunc("/api/admin/settings/recall-test", s.handleAdminRecallTest)
	mux.HandleFunc("/api/admin/settings", s.handleAdminSettings)
	mux.HandleFunc("/api/admin/plugins", s.handleAdminPlugins)
	mux.HandleFunc("/api/admin/plugins/import", s.handleAdminPluginImport)
	mux.HandleFunc("/api/admin/plugins/import/", s.handleAdminPluginImport)
	mux.HandleFunc("/api/admin/snippets", s.handleAdminSnippets)
	mux.HandleFunc("/api/admin/connected-apps", s.handleAdminConnectedApps)
	mux.HandleFunc("/api/admin/connected-apps/", s.handleAdminConnectedAppByID)
	mux.HandleFunc("/api/docs/snippets", s.handleDocsSnippets)
	mux.HandleFunc("/api/docs/plugins", s.handleDocsPlugins)
	mux.HandleFunc("/api/admin/categories", s.handleAdminCategories)
	mux.HandleFunc("/api/admin/categories/", s.handleAdminCategoryByID)
	mux.HandleFunc("/api/admin/modules", s.handleAdminModules)
	mux.HandleFunc("/api/admin/modules/", s.handleAdminModuleRoutes)
	mux.HandleFunc("/api/admin/entries/", s.handleAdminEntryByID)
	mux.HandleFunc("/api/admin/users", s.handleAdminUsers)
	mux.HandleFunc("/api/admin/users/", s.handleAdminUserByID)
	mux.HandleFunc("/api/admin/groups", s.handleAdminGroups)
	mux.HandleFunc("/api/admin/teams", s.handleAdminTeams)
	mux.HandleFunc("/api/admin/teams/", s.handleAdminTeamRoutes)
	mux.HandleFunc("/api/admin/", s.handleAdminAccepted)
	mux.HandleFunc("/api/webhooks/gitlab", s.handleGitLabWebhook)
	return s.cors(recoverer(mux))
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "modex-api"})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	if user, ok := s.currentUser(r); ok {
		writeJSON(w, http.StatusOK, struct {
			store.User
			IsSuperAdmin bool `json:"is_super_admin"`
			IsTeamAdmin  bool `json:"is_team_admin"`
		}{User: user, IsSuperAdmin: s.auth.IsSuperAdmin(user), IsTeamAdmin: s.isTeamAdmin(user)})
		return
	}
	writeError(w, http.StatusUnauthorized, "unauthorized", "not logged in")
}

func (s *Server) handleMockLogin(w http.ResponseWriter, r *http.Request) {
	if s.auth.Config().Mode == "oidc" {
		writeError(w, http.StatusForbidden, "mock_login_disabled", "mock login is disabled when AUTH_MODE=oidc")
		return
	}
	// Allow developers to choose which seeded identity to log in as.
	var req struct {
		Username string `json:"username"`
	}
	_ = decodeBody(r, &req)
	user := s.store.CurrentUser()
	if req.Username != "" {
		for _, u := range s.store.Users("") {
			if strings.EqualFold(u.Username, req.Username) {
				user = u
				break
			}
		}
	}
	user = s.store.UpsertUser(user)
	if err := s.auth.CreateSession(w, user); err != nil {
		writeError(w, http.StatusInternalServerError, "session_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user})
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
	user, err := s.auth.CompleteLogin(r.Context(), r, w)
	frontend := s.auth.Config().FrontendBaseURL
	if err != nil {
		// Surface the failure to the user in the portal rather than a bare JSON
		// 400, and log the detail server-side for diagnostics.
		log.Printf("oidc callback failed: %v", err)
		http.Redirect(w, r, frontend+"/?login_error="+url.QueryEscape(err.Error()), http.StatusFound)
		return
	}
	// Sync the SSO identity (and its groups) into the user directory.
	s.store.UpsertUser(user)
	http.Redirect(w, r, frontend, http.StatusFound)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.auth.Logout(w, r)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	cfg := s.auth.Config()
	loginURL := ""
	// Advertise the login URL when a real login is available: always in mock
	// mode, and in OIDC mode only once the provider is fully configured.
	if cfg.Mode != "oidc" || cfg.LoginReady() {
		loginURL = cfg.AppBaseURL + "/api/auth/login"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"auth_mode":          cfg.Mode,
		"oidc_login_enabled": cfg.LoginReady(),
		"login_url":          loginURL,
		"frontend_base_url":  cfg.FrontendBaseURL,
		// Effective doc-engine plugin state (enabled + non-secret config) so the
		// renderer can conditionally apply plugins without admin rights.
		"plugins": s.store.PluginEffective(),
	})
}

func (s *Server) handleCategories(w http.ResponseWriter, r *http.Request) {
	tree := s.store.CategoryTree()
	// Admin views pass ?scope=managed to get only the subtrees a team admin may
	// manage. Public/browse calls (no scope) always get the full tree.
	if r.URL.Query().Get("scope") == "managed" {
		if user, ok := s.currentUser(r); ok {
			if set, all := s.accessibleCategoryIDs(user); !all {
				tree = filterCategoryTree(tree, set)
			}
		}
	}
	writeJSON(w, http.StatusOK, tree)
}

// filterCategoryTree keeps only nodes whose id is in the set, preserving any
// ancestors needed to reach them.
func filterCategoryTree(nodes []store.Category, set map[string]bool) []store.Category {
	out := make([]store.Category, 0, len(nodes))
	for _, n := range nodes {
		children := filterCategoryTree(n.Children, set)
		if set[n.ID] || len(children) > 0 {
			n.Children = children
			out = append(out, n)
		}
	}
	return out
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
		if len(parts) >= 4 && parts[3] == "site" {
			s.handleDocSiteFile(w, r, module.ModuleKey, parts[1], parts[2], strings.Join(parts[4:], "/"))
			return
		}
		if len(parts) == 4 && parts[3] == "nav" {
			nav := s.store.Nav(module.ModuleKey, parts[1])
			if len(nav) == 0 {
				writeJSON(w, http.StatusOK, []map[string]string{{"title": "概览", "path": "#overview"}, {"title": "正文", "path": "#content"}, {"title": "元数据", "path": "#metadata"}})
				return
			}
			writeJSON(w, http.StatusOK, nav)
			return
		}
		page, err := s.store.PageByRoute(module.ModuleKey, parts[1], parts[2])
		if err != nil {
			writeResult(w, page, err)
			return
		}
		page.ContentHTML, err = s.docPageHTML(r.Context(), module.ModuleKey, parts[1], parts[2])
		if err != nil {
			writeError(w, http.StatusBadGateway, "site_read_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, page)
		return
	}
}

func (s *Server) handleDocSiteFile(w http.ResponseWriter, r *http.Request, moduleKey, docsVersion, entryKey, name string) {
	if name == "" {
		name = "index.html"
	}
	if s.minioClient != nil {
		zipName := fmt.Sprintf("site/%s/%s", entryKey, name)
		key := fmt.Sprintf("modules/%s/%s/%s", moduleKey, docsVersion, zipName)
		obj, err := s.minioClient.GetObject(r.Context(), s.minioBucket, key, minio.GetObjectOptions{})
		if err == nil {
			defer obj.Close()
			stat, statErr := obj.Stat()
			if statErr == nil && stat.Size > 0 {
				ct := stat.ContentType
				if ct == "" {
					ct = contentTypeForName(name, nil)
				}
				w.Header().Set("Content-Type", ct)
				w.Header().Set("Cache-Control", "private, max-age=60")
				w.WriteHeader(http.StatusOK)
				io.Copy(w, obj)
				return
			}
		}
	}
	// fallback to in-memory
	f, err := s.store.SiteFile(moduleKey, docsVersion, entryKey, name)
	if err != nil {
		writeResult(w, nil, err)
		return
	}
	w.Header().Set("Content-Type", f.ContentType)
	w.Header().Set("Cache-Control", "private, max-age=60")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(f.Content)
}

func (s *Server) handleDocPage(w http.ResponseWriter, r *http.Request) {
	docID := strings.TrimPrefix(r.URL.Path, "/api/docs/page/")
	page, err := s.store.Page(docID)
	if err != nil {
		writeResult(w, page, err)
		return
	}
	page.ContentHTML, err = s.docPageHTML(r.Context(), page.ModuleKey, page.DocsVersion, page.EntryKey)
	if err != nil {
		writeError(w, http.StatusBadGateway, "site_read_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *Server) docPageHTML(ctx context.Context, moduleKey, docsVersion, entryKey string) (string, error) {
	if s.minioClient != nil {
		key := fmt.Sprintf("modules/%s/%s/site/%s/index.html", moduleKey, docsVersion, entryKey)
		obj, err := s.minioClient.GetObject(ctx, s.minioBucket, key, minio.GetObjectOptions{})
		if err == nil {
			defer obj.Close()
			stat, statErr := obj.Stat()
			if statErr == nil && stat.Size > 0 {
				b, readErr := io.ReadAll(obj)
				if readErr == nil && len(b) > 0 {
					return string(b), nil
				}
				if readErr != nil {
					err = readErr
				}
			} else if statErr != nil {
				err = statErr
			}
		}
		if fallback := s.store.PageHTML(moduleKey, docsVersion, entryKey); fallback != "" {
			return fallback, nil
		}
		if err == nil {
			err = store.ErrNotFound
		}
		return "", fmt.Errorf("read MinIO object %s: %w", key, err)
	}
	return s.store.PageHTML(moduleKey, docsVersion, entryKey), nil
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
	if len(req.Filters.DocsVersions) == 0 {
		req.DefaultVersionsOnly = true
	}
	resp, err := s.search.Search(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "search_failed", err.Error())
		return
	}
	// Only persist explicit, user-committed searches (Enter / search button /
	// result click). Live as-you-type queries set Log=false to avoid flooding
	// the log with one row per keystroke.
	if req.Log && strings.TrimSpace(req.Query) != "" {
		filters, _ := json.Marshal(req.Filters)
		user, _ := s.currentUser(r)
		s.store.AddSearchLog(store.SearchLog{ID: fmt.Sprintf("sl-%d", time.Now().UnixNano()), UserID: user.ID, IPAddress: clientIP(r), Query: req.Query, Mode: string(resp.Mode), FiltersJSON: string(filters), ResultCount: resp.Total, SearchedAt: time.Now().UTC()})
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleAsk answers a natural-language question using retrieval over the docs
// (RAG). When ASK_HTTP_URL is configured it forwards the question plus retrieved
// context to an external LLM; otherwise it returns an extractive answer built
// from the top matches so the "Ask AI" flow works without an LLM key.
func (s *Server) handleAsk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	var req struct {
		Query       string   `json:"query"`
		ModuleKey   string   `json:"module_key"`
		CategoryIDs []string `json:"category_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if strings.TrimSpace(req.Query) == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "query is required")
		return
	}
	// Optional scope: constrain retrieval to a module (doc-level chat) or a
	// platform/category (domain-level ask).
	filters := search.Filters{}
	if req.ModuleKey != "" {
		filters.Modules = []string{req.ModuleKey}
	}
	if len(req.CategoryIDs) > 0 {
		filters.CategoryIDs = req.CategoryIDs
	}
	resp, err := s.search.Search(r.Context(), search.Request{Query: req.Query, Mode: search.ModeHybrid, Filters: filters, Page: 1, PageSize: 8, DefaultVersionsOnly: len(filters.DocsVersions) == 0})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ask_failed", err.Error())
		return
	}
	answer, provider := s.synthesizeAnswer(r.Context(), req.Query, resp.Results)
	user, _ := s.currentUser(r)
	s.store.AddSearchLog(store.SearchLog{ID: fmt.Sprintf("ask-%d", time.Now().UnixNano()), UserID: user.ID, IPAddress: clientIP(r), Query: req.Query, Mode: "ask", ResultCount: len(resp.Results), SearchedAt: time.Now().UTC()})
	writeJSON(w, http.StatusOK, map[string]any{
		"query":    req.Query,
		"answer":   answer,
		"provider": provider,
		"sources":  resp.Results,
	})
}

func (s *Server) synthesizeAnswer(ctx context.Context, query string, results []search.Result) (string, string) {
	// 1) Admin-configured OpenAI-compatible chat model (preferred).
	if ai := s.store.Settings().AI; strings.TrimSpace(ai.AskBaseURL) != "" && strings.TrimSpace(ai.AskModel) != "" {
		if answer, err := s.askOpenAICompatible(ctx, ai, query, results); err == nil && strings.TrimSpace(answer) != "" {
			return answer, "llm"
		} else if err != nil {
			log.Printf("ask llm failed: %v", err)
		}
	}
	// 2) Legacy custom {query,context}->{answer} proxy via env.
	if url := os.Getenv("ASK_HTTP_URL"); url != "" {
		if answer, err := s.askExternalLLM(ctx, url, query, results); err == nil && strings.TrimSpace(answer) != "" {
			return answer, "http"
		}
	}
	if len(results) == 0 {
		return "未在文档库中找到与该问题相关的内容。可以换个关键词，或在左侧按平台浏览。", "extractive"
	}
	var b strings.Builder
	b.WriteString("根据文档库中最相关的内容，整理如下：\n\n")
	for i, r := range results {
		if i >= 3 {
			break
		}
		b.WriteString(fmt.Sprintf("%d. 【%s】%s\n", i+1, r.ModuleName, r.Title))
		if r.Snippet != "" {
			b.WriteString("   " + r.Snippet + "\n")
		}
	}
	b.WriteString("\n以上内容来自下方引用文档，点击可查看完整页面。")
	return b.String(), "extractive"
}

func (s *Server) askExternalLLM(ctx context.Context, url, query string, results []search.Result) (string, error) {
	var ctxBuilder strings.Builder
	for i, r := range results {
		if i >= 6 {
			break
		}
		ctxBuilder.WriteString(s.askContextForResult(i+1, r))
	}
	payload, _ := json.Marshal(map[string]any{"query": query, "context": ctxBuilder.String()})
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(payload)))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if key := os.Getenv("ASK_HTTP_API_KEY"); key != "" {
		httpReq.Header.Set("Authorization", "Bearer "+key)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode >= 300 {
		return "", fmt.Errorf("ask endpoint returned %d", httpResp.StatusCode)
	}
	var out struct {
		Answer string `json:"answer"`
	}
	if err := json.NewDecoder(httpResp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.Answer, nil
}

// askOpenAICompatible calls any OpenAI-compatible /chat/completions endpoint
// with a retrieval-augmented prompt built from the top search results.
func (s *Server) askOpenAICompatible(ctx context.Context, ai store.AISettings, query string, results []search.Result) (string, error) {
	var ctxBuilder strings.Builder
	for i, r := range results {
		if i >= 6 {
			break
		}
		ctxBuilder.WriteString(s.askContextForResult(i+1, r))
	}
	system := strings.TrimSpace(ai.AskSystemPrompt)
	if system == "" {
		system = store.DefaultAskSystemPrompt
	}
	userMsg := fmt.Sprintf("文档片段：\n%s\n问题：%s", ctxBuilder.String(), query)
	// Dispatch to the configured API format (OpenAI / Anthropic / Gemini / …).
	return chatComplete(ctx, ai, system, userMsg)
}

func (s *Server) askContextForResult(i int, r search.Result) string {
	content := firstNonEmptyStr(r.Snippet, r.Title)
	if p, err := s.store.Page(r.DocID); err == nil {
		content = firstNonEmptyStr(p.ContentMD, p.ContentText, p.Description, r.Snippet)
	}
	content = truncateRunes(strings.TrimSpace(content), 4200)
	return fmt.Sprintf("[%d] %s\n路径：%s\n分类：%s\n正文：\n%s\n\n", i, r.Title, r.Path, r.Breadcrumb, content)
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	rs := []rune(s)
	if len(rs) <= max {
		return s
	}
	return string(rs[:max]) + "\n……"
}

func firstNonEmptyStr(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// handleAdminSettings exposes the admin-editable platform settings (AI model).
func (s *Server) handleAdminSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSuperAdmin(w, r); !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, maskedSettings(s.store.Settings()))
	case http.MethodPut, http.MethodPost:
		var body store.AISettings
		if err := decodeBody(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		saved := s.store.SaveAISettings(body)
		writeJSON(w, http.StatusOK, maskedSettings(saved))
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET or PUT")
	}
}

func (s *Server) handleAdminRecallTest(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSuperAdmin(w, r); !ok {
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	ai := s.store.Settings().AI
	query := strings.TrimSpace(ai.RecallTestQuery)
	if query == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "recall_test_query is required")
		return
	}
	topK := ai.RecallTestTopK
	if topK <= 0 {
		topK = 10
	}
	if topK > 100 {
		topK = 100
	}
	expected := parseDocIDList(ai.RecallTestDocIDs)
	if len(expected) == 0 {
		writeError(w, http.StatusBadRequest, "bad_request", "recall_test_doc_ids is required")
		return
	}
	resp, err := s.search.Search(r.Context(), search.Request{
		Query:               query,
		Mode:                search.ModeHybrid,
		Page:                1,
		PageSize:            topK,
		DefaultVersionsOnly: true,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "search_failed", err.Error())
		return
	}
	actual := make([]string, 0, len(resp.Results))
	for _, r := range resp.Results {
		actual = append(actual, r.DocID)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"query":            query,
		"top_k":            topK,
		"expected_doc_ids": expected,
		"actual_doc_ids":   actual,
		"recall_at_k":      recallAtK(expected, actual),
		"mrr":              reciprocalRank(expected, actual),
		"ndcg":             ndcg(expected, actual),
		"results":          resp.Results,
		"note":             "当前版本评估 hybrid 召回；重排序模型接入后可在此返回 rerank 前后对比。",
	})
}

// handleAdminPlugins exposes the built-in doc-engine plugin registry. GET
// returns the catalog merged with saved overrides; PUT persists enable/config
// overrides. Super-admin only; effective state is served to viewers via /api/config.
func (s *Server) handleAdminPlugins(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSuperAdmin(w, r); !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"plugins": s.store.PluginStates()})
	case http.MethodPut, http.MethodPost:
		var body struct {
			Plugins map[string]store.PluginSetting `json:"plugins"`
		}
		if err := decodeBody(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"plugins": s.store.SavePluginSettings(body.Plugins)})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET or PUT")
	}
}

// handleAdminPluginImport imports (POST) or removes (DELETE) a sandbox-rendered
// JSX plugin. Super-admin only. Imports stay disabled until toggled on.
func (s *Server) handleAdminPluginImport(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSuperAdmin(w, r); !ok {
		return
	}
	key := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/api/admin/plugins/import"), "/")
	switch r.Method {
	case http.MethodPost, http.MethodPut:
		var body store.UploadedPlugin
		if err := decodeBody(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		saved, err := s.store.SaveUploadedPlugin(body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_plugin", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"plugin": saved, "plugins": s.store.PluginStates()})
	case http.MethodDelete:
		if key == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "plugin key required")
			return
		}
		if !s.store.DeleteUploadedPlugin(key) {
			writeError(w, http.StatusNotFound, "not_found", "plugin not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "deleted", "key": key, "plugins": s.store.PluginStates()})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST or DELETE")
	}
}

// handleDocsPlugins exposes enabled uploaded plugins (with their JSX source) to
// the renderer. Read-only and un-gated, like /api/docs/snippets.
func (s *Server) handleDocsPlugins(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"plugins": s.store.EnabledUploadedPlugins()})
}

// handleAdminSnippets manages the reusable snippet library and variables.
// Super-admin only. GET returns the current set; PUT replaces it.
func (s *Server) handleAdminSnippets(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSuperAdmin(w, r); !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		snips, vars := s.store.SnippetData()
		writeJSON(w, http.StatusOK, map[string]any{"snippets": snips, "variables": vars})
	case http.MethodPut, http.MethodPost:
		var body struct {
			Snippets  []store.Snippet   `json:"snippets"`
			Variables map[string]string `json:"variables"`
		}
		if err := decodeBody(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		snips, vars := s.store.SaveSnippetData(body.Snippets, body.Variables)
		writeJSON(w, http.StatusOK, map[string]any{"snippets": snips, "variables": vars})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET or PUT")
	}
}

// handleDocsSnippets exposes the snippet library + variables to the renderer.
// Read-only and un-gated (no secrets), mirroring /api/config.
func (s *Server) handleDocsSnippets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}
	snips, vars := s.store.SnippetData()
	writeJSON(w, http.StatusOK, map[string]any{"snippets": snips, "variables": vars})
}

// handleAdminModels proxies GET {base}/models so the admin UI can populate a
// model picker. Uses the request's api_key, falling back to the stored key.
func (s *Server) handleAdminModels(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSuperAdmin(w, r); !ok {
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	var body struct {
		Protocol string `json:"protocol"`
		BaseURL  string `json:"base_url"`
		APIKey   string `json:"api_key"`
	}
	if err := decodeBody(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	cur := s.store.Settings().AI
	base := strings.TrimSpace(body.BaseURL)
	if base == "" {
		base = cur.AskBaseURL
	}
	if base == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "base_url required")
		return
	}
	key := strings.TrimSpace(body.APIKey)
	if key == "" {
		key = cur.AskAPIKey
	}
	protocol := strings.TrimSpace(body.Protocol)
	if protocol == "" {
		protocol = cur.AskProtocol
	}
	ids, err := listModels(r.Context(), protocol, base, key)
	if err != nil {
		writeError(w, http.StatusBadGateway, "models_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"models": ids})
}

// maskedSettings hides the stored API key but reports whether one is set.
func maskedSettings(set store.Settings) map[string]any {
	ai := set.AI
	keySet := strings.TrimSpace(ai.AskAPIKey) != ""
	embeddingKeySet := strings.TrimSpace(ai.EmbeddingAPIKey) != ""
	rerankKeySet := strings.TrimSpace(ai.RerankAPIKey) != ""
	ai.AskAPIKey = ""
	ai.EmbeddingAPIKey = ""
	ai.RerankAPIKey = ""
	return map[string]any{
		"ai":                        ai,
		"ask_api_key_set":           keySet,
		"embedding_api_key_set":     embeddingKeySet,
		"rerank_api_key_set":        rerankKeySet,
		"ask_system_prompt_default": store.DefaultAskSystemPrompt,
	}
}

func parseDocIDList(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == '\r' || r == '\t' || r == ',' || r == '，' || r == ';' || r == '；'
	})
	seen := map[string]bool{}
	out := []string{}
	for _, f := range fields {
		v := strings.TrimSpace(f)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func expectedSet(expected []string) map[string]bool {
	set := make(map[string]bool, len(expected))
	for _, id := range expected {
		set[id] = true
	}
	return set
}

func recallAtK(expected, actual []string) float64 {
	if len(expected) == 0 {
		return 0
	}
	set := expectedSet(expected)
	hits := 0
	for _, id := range actual {
		if set[id] {
			hits++
		}
	}
	return float64(hits) / float64(len(expected))
}

func reciprocalRank(expected, actual []string) float64 {
	set := expectedSet(expected)
	for i, id := range actual {
		if set[id] {
			return 1 / float64(i+1)
		}
	}
	return 0
}

func ndcg(expected, actual []string) float64 {
	if len(expected) == 0 || len(actual) == 0 {
		return 0
	}
	set := expectedSet(expected)
	dcg := 0.0
	for i, id := range actual {
		if set[id] {
			dcg += 1 / math.Log2(float64(i+2))
		}
	}
	idealHits := len(expected)
	if idealHits > len(actual) {
		idealHits = len(actual)
	}
	idcg := 0.0
	for i := 0; i < idealHits; i++ {
		idcg += 1 / math.Log2(float64(i+2))
	}
	if idcg == 0 {
		return 0
	}
	return dcg / idcg
}

func (s *Server) handleFacets(w http.ResponseWriter, r *http.Request) {
	resp, _ := s.search.Search(r.Context(), search.Request{Mode: search.ModeKeyword, PageSize: 1})
	writeJSON(w, http.StatusOK, resp.Facets)
}

func (s *Server) handleEmbedText(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	if user, ok := s.requireUser(w, r); !ok {
		return
	} else if !isAdmin(user) && !s.auth.IsSuperAdmin(user) {
		writeError(w, http.StatusForbidden, "forbidden", "admin required")
		return
	}
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

func (s *Server) handleEmbeddingReindex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	if user, ok := s.requireUser(w, r); !ok {
		return
	} else if !isAdmin(user) && !s.auth.IsSuperAdmin(user) {
		writeError(w, http.StatusForbidden, "forbidden", "admin required")
		return
	}
	count, err := s.search.Reindex(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "embedding_reindex_failed", err.Error())
		return
	}
	storedCount, err := s.search.EmbeddingCount(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "embedding_count_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":           "reindexed",
		"provider":         s.search.Embedder.Name(),
		"embedded_pages":   count,
		"cached_documents": storedCount,
	})
}

func (s *Server) handleSearchReindex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	if user, ok := s.requireUser(w, r); !ok {
		return
	} else if !isAdmin(user) && !s.auth.IsSuperAdmin(user) {
		writeError(w, http.StatusForbidden, "forbidden", "admin required")
		return
	}
	// The keyword index is computed from the in-memory page set, so reindexing
	// primarily (re)builds the embedding cache used by semantic/hybrid search.
	count, err := s.search.Reindex(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "search_reindex_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":             "reindexed",
		"indexed_documents":  len(s.store.Pages()),
		"embedded_documents": count,
	})
}

func (s *Server) handleDeploy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	artifact, err := deploy.ParseZip(r.Body, envInt64("DOCS_DEPLOY_MAX_BYTES", 100*1024*1024))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_artifact", err.Error())
		return
	}

	// Deploy auth (GitLab CI / docsctl integration). Each document source owns
	// an independent token; the artifact module key selects the token to verify.
	// Token can be sent as X-Modex-Deploy-Token header or Authorization: Bearer <token>
	moduleKey := artifact.Metadata.ModuleKey
	provided := r.Header.Get("X-Modex-Deploy-Token")
	if provided == "" {
		provided = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	}
	m, moduleErr := s.store.Module(moduleKey)
	if moduleErr != nil {
		writeError(w, http.StatusNotFound, "module_not_found", "document source not found")
		return
	}
	if m.DeployToken == "" {
		writeError(w, http.StatusForbidden, "deploy_token_not_configured", "generate a deploy token for this document source first")
		return
	}
	if subtle.ConstantTimeCompare([]byte(provided), []byte(m.DeployToken)) != 1 {
		writeError(w, http.StatusForbidden, "invalid_deploy_token", "deploy token required or invalid for this module")
		return
	}

	if s.minioClient != nil {
		if err := s.uploadSiteFilesToMinIO(r.Context(), artifact, moduleKey, artifact.Metadata.DocsVersion); err != nil {
			writeError(w, http.StatusBadGateway, "site_upload_failed", err.Error())
			return
		}
		// MinIO is the source of truth for static site assets in deployed
		// environments. Keeping the same bytes in the in-memory fallback can
		// push the backend into multi-GB RSS for image-heavy documentation.
		artifact.SiteFiles = nil
		artifact.SiteHTML = nil
	}
	if err := s.search.DeleteModuleVersionEmbeddings(r.Context(), moduleKey, artifact.Metadata.DocsVersion); err != nil {
		writeError(w, http.StatusBadGateway, "embedding_cleanup_failed", err.Error())
		return
	}

	result, err := s.store.IngestArtifact(toStoreArtifact(artifact))
	if err != nil {
		writeError(w, http.StatusBadRequest, "deploy_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"status": "published", "result": result})
}

func (s *Server) handleReleases(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireConsole(w, r)
	if !ok {
		return
	}
	releases := s.store.Releases()
	if set, all := s.accessibleCategoryIDs(user); !all {
		scoped := releases[:0:0]
		for _, rel := range releases {
			if categoriesIntersect(s.moduleCategories(rel.ModuleKey), set) {
				scoped = append(scoped, rel)
			}
		}
		releases = scoped
	}
	if kw := keywordOf(r); kw != "" {
		filtered := releases[:0:0]
		for _, rel := range releases {
			if containsFold(rel.ModuleKey, kw) || containsFold(rel.Publisher, kw) || containsFold(rel.DocsVersion, kw) || containsFold(rel.ReleaseID, kw) {
				filtered = append(filtered, rel)
			}
		}
		releases = filtered
	}
	if wantsPage(r) {
		page, limit := pageParams(r)
		writeJSON(w, http.StatusOK, paginate(releases, page, limit))
		return
	}
	writeJSON(w, http.StatusOK, releases)
}

// handleDocAnalytics powers the doc-page "eye" popover: a daily read trend and
// a per-reader breakdown for one document. Reading statistics are available
// only when the server-side PostHog query credentials are configured.
func (s *Server) handleDocAnalytics(w http.ResponseWriter, r *http.Request) {
	docID := strings.TrimSpace(r.URL.Query().Get("doc_id"))
	if docID == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "doc_id is required")
		return
	}
	days := 30
	if v := r.URL.Query().Get("days"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 180 {
			days = n
		}
	}
	stats, err := posthogDocStats(docID, days)
	if err != nil {
		if errors.Is(err, errPosthogNotConfigured) {
			writeError(w, http.StatusServiceUnavailable, "posthog_not_configured", "reading statistics are unavailable because PostHog is not configured")
			return
		}
		writeError(w, http.StatusBadGateway, "posthog_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"source": "posthog", "stats": stats})
}

func (s *Server) handleDocFeedback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	var req struct {
		DocID     string `json:"doc_id"`
		Rating    string `json:"rating"`
		Comment   string `json:"comment"`
		SessionID string `json:"session_id"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	req.DocID = strings.TrimSpace(req.DocID)
	req.Rating = strings.TrimSpace(req.Rating)
	req.Comment = strings.TrimSpace(req.Comment)
	if req.DocID == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "doc_id is required")
		return
	}
	if req.Rating != "good" && req.Rating != "bad" {
		writeError(w, http.StatusBadRequest, "bad_request", "rating must be good or bad")
		return
	}
	user, _ := s.currentUser(r)
	f := s.store.AddDocFeedback(store.DocFeedback{
		DocID: req.DocID, Rating: req.Rating, Comment: req.Comment, UserID: user.ID, SessionID: req.SessionID,
	})
	writeJSON(w, http.StatusAccepted, map[string]any{"status": "recorded", "feedback": f})
}

func (s *Server) handleDocFeedbackLogs(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireConsole(w, r)
	if !ok {
		return
	}
	logs := s.store.DocFeedbacks()
	type out struct {
		store.DocFeedback
		DisplayName string `json:"display_name"`
	}
	res := make([]out, 0, len(logs))
	set, all := s.accessibleCategoryIDs(user)
	for _, log := range logs {
		if !all && !categoriesIntersect(s.docCategoryIDs(log.DocID), set) {
			continue
		}
		dn := ""
		if log.UserID != "" {
			if u, err := s.store.UserByID(log.UserID); err == nil {
				dn = u.DisplayName
				if dn == "" {
					dn = u.Username
				}
			}
		}
		res = append(res, out{DocFeedback: log, DisplayName: dn})
	}
	if kw := keywordOf(r); kw != "" {
		filtered := res[:0:0]
		for _, l := range res {
			if containsFold(l.DocID, kw) || containsFold(l.Title, kw) || containsFold(l.ModuleKey, kw) || containsFold(l.Rating, kw) || containsFold(l.Comment, kw) || containsFold(l.DisplayName, kw) || containsFold(l.UserID, kw) {
				filtered = append(filtered, l)
			}
		}
		res = filtered
	}
	if wantsPage(r) {
		page, limit := pageParams(r)
		writeJSON(w, http.StatusOK, paginate(res, page, limit))
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleSearchLogs(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireConsole(w, r)
	if !ok {
		return
	}
	logs := s.store.SearchLogs()
	type out struct {
		store.SearchLog
		DisplayName string `json:"display_name"`
	}
	res := make([]out, 0, len(logs))
	set, all := s.accessibleCategoryIDs(user)
	for _, log := range logs {
		// Team admins only see searches whose clicked doc maps to an owned
		// category; searches with no resolvable category are hidden.
		if !all && !categoriesIntersect(s.docCategoryIDs(log.ClickedDocID), set) {
			continue
		}
		dn := ""
		if log.UserID != "" {
			if u, err := s.store.UserByID(log.UserID); err == nil {
				dn = u.DisplayName
				if dn == "" {
					dn = u.Username
				}
			}
		}
		res = append(res, out{SearchLog: log, DisplayName: dn})
	}
	if kw := keywordOf(r); kw != "" {
		filtered := res[:0:0]
		for _, l := range res {
			if containsFold(l.Query, kw) || containsFold(l.DisplayName, kw) || containsFold(l.UserID, kw) || containsFold(l.IPAddress, kw) {
				filtered = append(filtered, l)
			}
		}
		res = filtered
	}
	if wantsPage(r) {
		page, limit := pageParams(r)
		writeJSON(w, http.StatusOK, paginate(res, page, limit))
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleMCPLogs(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireConsole(w, r)
	if !ok {
		return
	}
	logs := s.store.MCPLogs()
	type out struct {
		store.MCPLog
		DisplayName string `json:"display_name"`
	}
	res := make([]out, 0, len(logs))
	set, all := s.accessibleCategoryIDs(user)
	for _, log := range logs {
		// Team admins only see MCP calls resolvable to an owned category.
		if !all && !categoriesIntersect(s.mcpLogCategoryIDs(log.InputJSON), set) {
			continue
		}
		dn := ""
		if log.UserID != "" {
			if u, err := s.store.UserByID(log.UserID); err == nil {
				dn = u.DisplayName
				if dn == "" {
					dn = u.Username
				}
			}
		}
		res = append(res, out{MCPLog: log, DisplayName: dn})
	}
	if kw := keywordOf(r); kw != "" {
		filtered := res[:0:0]
		for _, l := range res {
			if containsFold(l.ToolName, kw) || containsFold(l.Query, kw) || containsFold(l.DisplayName, kw) || containsFold(l.UserID, kw) {
				filtered = append(filtered, l)
			}
		}
		res = filtered
	}
	if wantsPage(r) {
		page, limit := pageParams(r)
		writeJSON(w, http.StatusOK, paginate(res, page, limit))
		return
	}
	writeJSON(w, http.StatusOK, res)
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
	if user.ID == "" {
		if tok := bearerToken(r); tok != "" {
			if u, err := s.store.UserByMCPToken(tok); err == nil {
				user = u
			}
		}
	}
	s.store.AddMCPLog(store.MCPLog{ID: fmt.Sprintf("ml-%d", time.Now().UnixNano()), ToolName: req.ToolName, UserID: user.ID, Query: req.Query, InputJSON: req.InputJSON, ResultCount: req.ResultCount, CreatedAt: time.Now().UTC()})
	writeJSON(w, http.StatusAccepted, map[string]any{"status": "logged"})
}

func (s *Server) handleMeMCPToken(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "not logged in")
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]string{"mcp_token": user.MCPToken})
	case http.MethodPost:
		tok, err := randomToken(32)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "token_gen_failed", err.Error())
			return
		}
		updated, err := s.store.SetUserMCPToken(user.ID, tok)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "update_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"mcp_token": updated.MCPToken})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET or POST")
	}
}

// currentUser returns the authenticated user from the session cookie. Login is
// a real, cookie-backed action in both mock and OIDC modes, so there is no
// silent impersonation; anonymous callers simply get ok == false.
func (s *Server) currentUser(r *http.Request) (store.User, bool) {
	if user, ok := s.auth.CurrentUser(r); ok {
		return user, true
	}
	if tok := bearerToken(r); tok != "" {
		if user, _, _, err := s.store.UserByOAuthAccessToken(tok); err == nil {
			return user, true
		}
	}
	return store.User{}, false
}

func isAdmin(u store.User) bool {
	for _, r := range u.Roles {
		if r == "admin" {
			return true
		}
	}
	return false
}

// canManageCategory reports whether the user may manage the given platform.
// A managed category id covers its descendants (e.g. "engineering" covers
// "engineering.cbb").
func canManageCategory(u store.User, categoryID string) bool {
	for _, m := range u.ManagedCategories {
		if m == categoryID || strings.HasPrefix(categoryID, m+".") {
			return true
		}
	}
	return false
}

func canManageCategories(u store.User, categoryIDs []string) bool {
	for _, id := range categoryIDs {
		if canManageCategory(u, id) {
			return true
		}
	}
	return false
}

// isTeamLeader checks if the user is the designated leader of the team.
func (s *Server) isTeamLeader(u store.User, teamKey string) bool {
	if teamKey == "" {
		return false
	}
	t, err := s.store.Team(teamKey)
	if err != nil {
		return false
	}
	for _, l := range t.Leaders {
		if strings.EqualFold(l, u.Username) || strings.EqualFold(l, u.ID) {
			return true
		}
	}
	return false
}

// teamMembers returns usernames/ids in the team (for ownership checks).
func (s *Server) teamMembers(teamKey string) []string {
	return s.store.TeamMembers(teamKey)
}

// canManageViaResponsibleTeam allows members (incl. leader) of a category's responsible team
// to manage that category's resources (generic domain ownership).
func (s *Server) canManageViaResponsibleTeam(u store.User, categoryIDs []string) bool {
	for _, cid := range categoryIDs {
		// Check direct; for hierarchy the responsible on parent covers subs conceptually,
		// but we also check the specific id's assignment.
		resp := s.categoryResponsible(cid)
		if resp == "" {
			continue
		}
		for _, m := range s.teamMembers(resp) {
			if strings.EqualFold(m, u.Username) || strings.EqualFold(m, u.ID) {
				return true
			}
		}
	}
	return false
}

func (s *Server) categoryResponsible(id string) string {
	// Walk the tree (small data) to find assignment for id or nearest ancestor with one.
	tree := s.store.CategoryTree()
	var find func([]store.Category) string
	find = func(cats []store.Category) string {
		for _, c := range cats {
			if c.ID == id {
				if c.ResponsibleTeam != "" {
					return c.ResponsibleTeam
				}
				// inherit from parent? caller walks up if needed; here return what we have at leaf.
				return ""
			}
			if hit := find(c.Children); hit != "" {
				return hit
			}
		}
		return ""
	}
	// Also try ancestor match for sub-ids (e.g. "standards.foo" covered by "standards" team)
	for _, c := range tree {
		if c.ID == id || strings.HasPrefix(id, c.ID+".") {
			if c.ResponsibleTeam != "" {
				return c.ResponsibleTeam
			}
		}
		for _, ch := range c.Children {
			if ch.ID == id || strings.HasPrefix(id, ch.ID+".") {
				if ch.ResponsibleTeam != "" {
					return ch.ResponsibleTeam
				}
			}
		}
	}
	return find(tree)
}

// accessibleCategoryIDs returns the set of category IDs a user may see/manage in
// the admin console. Super admins get (nil, true) meaning "everything". A team
// member gets the categories their team(s) own (Category.ResponsibleTeam) plus
// all descendants. A user with no team gets an empty set.
func (s *Server) accessibleCategoryIDs(u store.User) (set map[string]bool, all bool) {
	if s.auth.IsSuperAdmin(u) {
		return nil, true
	}
	teamKeys := map[string]bool{}
	for _, k := range s.store.TeamKeysForUser(u) {
		teamKeys[strings.ToLower(strings.TrimSpace(k))] = true
	}
	set = map[string]bool{}
	if len(teamKeys) == 0 {
		return set, false
	}
	cats := s.store.AllCategories()
	for _, c := range cats {
		if c.ResponsibleTeam != "" && teamKeys[strings.ToLower(c.ResponsibleTeam)] {
			set[c.ID] = true
		}
	}
	// Expand to all descendants via ParentID closure (a child without its own
	// ResponsibleTeam is still owned through its parent).
	for {
		added := false
		for _, c := range cats {
			if !set[c.ID] && c.ParentID != "" && set[c.ParentID] {
				set[c.ID] = true
				added = true
			}
		}
		if !added {
			break
		}
	}
	return set, false
}

// hasConsoleAccess reports whether the user may enter the admin console at all
// (super admin, or a member/leader of any team).
func (s *Server) hasConsoleAccess(u store.User) bool {
	return s.auth.IsSuperAdmin(u) || len(s.store.TeamKeysForUser(u)) > 0
}

// isTeamAdmin reports a non-super-admin who belongs to at least one team (the
// team-scoped admin tier).
func (s *Server) isTeamAdmin(u store.User) bool {
	return !s.auth.IsSuperAdmin(u) && len(s.store.TeamKeysForUser(u)) > 0
}

// requireConsole gates console-scoped reads (logs, releases, module list). Super
// admins and team members pass; everyone else gets 403.
func (s *Server) requireConsole(w http.ResponseWriter, r *http.Request) (store.User, bool) {
	user, ok := s.auth.CurrentUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "login required")
		return store.User{}, false
	}
	if !s.hasConsoleAccess(user) {
		writeError(w, http.StatusForbidden, "forbidden", "console access required")
		return store.User{}, false
	}
	return user, true
}

// docCategoryIDs resolves a document id to the categories of its module.
func (s *Server) docCategoryIDs(docID string) []string {
	if strings.TrimSpace(docID) == "" {
		return nil
	}
	p, err := s.store.Page(docID)
	if err != nil {
		return nil
	}
	return s.moduleCategories(p.ModuleKey)
}

// mcpLogCategoryIDs best-effort resolves an MCP log to categories by parsing its
// free-form input JSON for a doc id or module key. MCP logs carry no structured
// module field, so this is heuristic; unresolved logs return nil (hidden from
// team admins, shown to super admins who bypass scoping).
func (s *Server) mcpLogCategoryIDs(inputJSON string) []string {
	if strings.TrimSpace(inputJSON) == "" {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(inputJSON), &m); err != nil {
		return nil
	}
	str := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := m[k].(string); ok && v != "" {
				return v
			}
		}
		return ""
	}
	if doc := str("doc_id", "docID", "docId"); doc != "" {
		if cats := s.docCategoryIDs(doc); len(cats) > 0 {
			return cats
		}
	}
	if mod := str("module_key", "moduleKey", "module"); mod != "" {
		return s.moduleCategories(mod)
	}
	return nil
}

// categoriesIntersect reports whether any of the ids is in the accessible set.
func categoriesIntersect(ids []string, set map[string]bool) bool {
	for _, id := range ids {
		if set[id] {
			return true
		}
	}
	return false
}

// requireUser writes 401 and returns false when no valid session is present.
func (s *Server) requireUser(w http.ResponseWriter, r *http.Request) (store.User, bool) {
	user, ok := s.auth.CurrentUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "login required")
		return store.User{}, false
	}
	return user, true
}

// requireSuperAdmin gates super-admin-only actions (e.g. user/permission mgmt).
func (s *Server) requireSuperAdmin(w http.ResponseWriter, r *http.Request) (store.User, bool) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return store.User{}, false
	}
	if !s.auth.IsSuperAdmin(user) {
		writeError(w, http.StatusForbidden, "forbidden", "super admin required")
		return store.User{}, false
	}
	return user, true
}

// requirePlatform gates platform-scoped writes: super admins pass; otherwise the
// user must have management rights on at least one of the target categories.
// Team responsible for the domain (via Category.ResponsibleTeam) also grants access
// to its members/leaders (generic for OSS doc maintenance teams owning 领域).
func (s *Server) requirePlatform(w http.ResponseWriter, r *http.Request, categoryIDs []string) (store.User, bool) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return store.User{}, false
	}
	if s.auth.IsSuperAdmin(user) {
		return user, true
	}
	if isAdmin(user) && canManageCategories(user, categoryIDs) {
		return user, true
	}
	if s.canManageViaResponsibleTeam(user, categoryIDs) {
		return user, true
	}
	writeError(w, http.StatusForbidden, "forbidden", "no management permission for this platform")
	return store.User{}, false
}

// moduleCategories resolves the category IDs attached to a module key.
func (s *Server) moduleCategories(moduleKey string) []string {
	if m, err := s.store.Module(moduleKey); err == nil {
		return m.CategoryIDs
	}
	return nil
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
	// Top-level platforms are super-admin only; sub-platforms can be created by
	// a manager of the parent platform.
	if c.ParentID == "" {
		if _, ok := s.requireSuperAdmin(w, r); !ok {
			return
		}
	} else if _, ok := s.requirePlatform(w, r, []string{c.ParentID}); !ok {
		return
	}
	created, err := s.store.CreateCategory(c)
	writeMutation(w, created, http.StatusCreated, err)
}

func (s *Server) handleAdminCategoryByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/admin/categories/")
	// Drag-and-drop move: POST /api/admin/categories/{id}/move {parent_id, index}.
	if strings.HasSuffix(id, "/move") {
		id = strings.TrimSuffix(id, "/move")
		if _, ok := s.requireSuperAdmin(w, r); !ok {
			return
		}
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
			return
		}
		var body struct {
			ParentID string `json:"parent_id"`
			Index    int    `json:"index"`
		}
		if err := decodeBody(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		moved, err := s.store.MoveCategory(id, body.ParentID, body.Index)
		writeMutation(w, moved, http.StatusOK, err)
		return
	}
	if _, ok := s.requirePlatform(w, r, []string{id}); !ok {
		return
	}
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
		user, ok := s.requireConsole(w, r)
		if !ok {
			return
		}
		modules := s.store.Modules("", "")
		// Team admins only see modules attached to a category they own.
		if set, all := s.accessibleCategoryIDs(user); !all {
			scoped := modules[:0:0]
			for _, m := range modules {
				if categoriesIntersect(m.CategoryIDs, set) {
					scoped = append(scoped, m)
				}
			}
			modules = scoped
		}
		if kw := keywordOf(r); kw != "" {
			filtered := modules[:0:0]
			for _, m := range modules {
				if containsFold(m.Name, kw) || containsFold(m.ModuleKey, kw) || containsFold(m.RepoURL, kw) || containsFold(m.Description, kw) {
					filtered = append(filtered, m)
				}
			}
			modules = filtered
		}
		if wantsPage(r) {
			page, limit := pageParams(r)
			writeJSON(w, http.StatusOK, paginate(modules, page, limit))
			return
		}
		writeJSON(w, http.StatusOK, modules)
		return
	}
	var m store.Module
	if err := decodeBody(r, &m); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	// Every doc source must be filed under at least one category.
	if len(m.CategoryIDs) == 0 {
		writeError(w, http.StatusBadRequest, "category_required", "文档源必须关联至少一个分类")
		return
	}
	if _, ok := s.requirePlatform(w, r, m.CategoryIDs); !ok {
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
	// Migration (reassign platform/owner) has its own dual-platform permission
	// check, so handle it before the generic platform gate.
	if len(parts) == 2 && parts[1] == "migrate" && r.Method == http.MethodPost {
		s.handleMigrateModule(w, r, moduleKey)
		return
	}
	user, ok := s.requirePlatform(w, r, s.moduleCategories(moduleKey))
	if !ok {
		return
	}
	// Reveal / rotate the CI deploy token (kept out of normal serialization).
	if len(parts) == 2 && parts[1] == "deploy-token" {
		m, err := s.store.Module(moduleKey)
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", "module not found")
			return
		}
		if r.Method == http.MethodPost { // rotate
			token := "mdx_" + strconv.FormatInt(time.Now().UnixNano(), 36)
			if _, err := s.store.UpdateModule(moduleKey, store.Module{DeployToken: token}); err != nil {
				writeResult(w, nil, err)
				return
			}
			m.DeployToken = token
		} else if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET or POST")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"deploy_token": m.DeployToken,
			"deploy_url":   firstNonEmptyStr(os.Getenv("APP_BASE_URL"), "") + "/api/deploy",
			"module_key":   moduleKey,
		})
		return
	}
	switch {
	case len(parts) == 1 && r.Method == http.MethodPut:
		var m store.Module
		if err := decodeBody(r, &m); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		// A doc source must keep at least one category; reject an explicit clear.
		if m.CategoryIDs != nil && len(m.CategoryIDs) == 0 {
			writeError(w, http.StatusBadRequest, "category_required", "文档源必须关联至少一个分类")
			return
		}
		// Team admins may only file modules under categories they own.
		if len(m.CategoryIDs) > 0 {
			if set, all := s.accessibleCategoryIDs(user); !all {
				for _, cid := range m.CategoryIDs {
					if !set[cid] {
						writeError(w, http.StatusForbidden, "forbidden", "只能选择本团队负责的分类")
						return
					}
				}
			}
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

// handleMigrateModule reassigns a module to different platform(s) (and
// optionally a new owner). The caller must be able to manage both the source
// and destination platforms (super admins bypass the check).
func (s *Server) handleMigrateModule(w http.ResponseWriter, r *http.Request, moduleKey string) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var req struct {
		CategoryIDs []string `json:"category_ids"`
		OwnerGroup  string   `json:"owner_group"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if len(req.CategoryIDs) == 0 {
		writeError(w, http.StatusBadRequest, "bad_request", "category_ids is required")
		return
	}
	if !s.auth.IsSuperAdmin(user) {
		source := s.moduleCategories(moduleKey)
		if !isAdmin(user) || !canManageCategories(user, source) || !canManageCategories(user, req.CategoryIDs) {
			writeError(w, http.StatusForbidden, "forbidden", "need management permission on both source and target platforms")
			return
		}
	}
	names := make([]string, 0, len(req.CategoryIDs))
	for _, id := range req.CategoryIDs {
		names = append(names, s.store.CategoryName(id))
	}
	updated, err := s.store.UpdateModule(moduleKey, store.Module{
		CategoryIDs:  req.CategoryIDs,
		CategoryPath: strings.Join(names, " / "),
		OwnerGroup:   req.OwnerGroup,
	})
	writeMutation(w, updated, http.StatusOK, err)
}

func (s *Server) handleAdminEntryByID(w http.ResponseWriter, r *http.Request) {
	entryID := strings.TrimPrefix(r.URL.Path, "/api/admin/entries/")
	if moduleKey, ok := s.store.EntryModuleKey(entryID); ok {
		if _, ok := s.requirePlatform(w, r, s.moduleCategories(moduleKey)); !ok {
			return
		}
	} else if _, ok := s.requireUser(w, r); !ok {
		return
	}
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
	user, ok := s.requireConsole(w, r)
	if !ok {
		return
	}
	parts := splitPath(strings.TrimPrefix(r.URL.Path, "/api/admin/releases/"))
	if len(parts) == 0 {
		writeError(w, http.StatusNotFound, "not_found", "release route not found")
		return
	}
	releaseID := parts[0]
	rel, err := s.store.Release(releaseID)
	if err != nil {
		writeResult(w, rel, err)
		return
	}
	if set, all := s.accessibleCategoryIDs(user); !all && !categoriesIntersect(s.moduleCategories(rel.ModuleKey), set) {
		writeError(w, http.StatusForbidden, "forbidden", "no access to this release")
		return
	}
	if len(parts) == 2 && parts[1] == "rollback" && r.Method == http.MethodPost {
		rel, err = s.store.RollbackRelease(releaseID)
		writeResult(w, rel, err)
		return
	}
	if len(parts) == 1 && r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, rel)
		return
	}
	writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET or POST /rollback")
}

func (s *Server) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSuperAdmin(w, r); !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		users := s.store.Users(r.URL.Query().Get("keyword"))
		// Enrich with the effective super admin status (persisted flag OR env SUPER_ADMIN_USERS).
		for i := range users {
			users[i].SuperAdmin = s.auth.IsSuperAdmin(users[i])
		}
		if wantsPage(r) {
			page, limit := pageParams(r)
			writeJSON(w, http.StatusOK, paginate(users, page, limit))
			return
		}
		writeJSON(w, http.StatusOK, users)
	case http.MethodPost:
		var u store.User
		if err := decodeBody(r, &u); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		created, err := s.store.CreateUser(u)
		if err == nil {
			created.SuperAdmin = s.auth.IsSuperAdmin(created)
		}
		writeMutation(w, created, http.StatusCreated, err)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET or POST")
	}
}

func (s *Server) handleAdminUserByID(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSuperAdmin(w, r); !ok {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/admin/users/")
	switch r.Method {
	case http.MethodGet:
		u, err := s.store.UserByID(id)
		if err == nil {
			u.SuperAdmin = s.auth.IsSuperAdmin(u)
		}
		writeResult(w, u, err)
	case http.MethodPut:
		var u store.User
		if err := decodeBody(r, &u); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		updated, err := s.store.UpdateUser(id, u)
		if err == nil {
			updated.SuperAdmin = s.auth.IsSuperAdmin(updated)
		}
		writeMutation(w, updated, http.StatusOK, err)
	case http.MethodDelete:
		if err := s.store.DeleteUser(id); err != nil {
			writeResult(w, nil, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "deleted", "id": id})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET, PUT or DELETE")
	}
}

func (s *Server) handleAdminGroups(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSuperAdmin(w, r); !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.store.Groups())
	case http.MethodPost:
		var g store.Group
		if err := decodeBody(r, &g); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		created, err := s.store.CreateGroup(g)
		writeMutation(w, created, http.StatusCreated, err)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET or POST")
	}
}

func (s *Server) handleAdminTeams(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSuperAdmin(w, r); !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		teams := s.store.Teams()
		if kw := keywordOf(r); kw != "" {
			filtered := teams[:0:0]
			for _, t := range teams {
				if containsFold(t.Name, kw) || containsFold(t.Key, kw) || containsFold(strings.Join(t.Leaders, " "), kw) || containsFold(t.Description, kw) || containsFold(strings.Join(t.Members, " "), kw) {
					filtered = append(filtered, t)
				}
			}
			teams = filtered
		}
		if wantsPage(r) {
			page, limit := pageParams(r)
			writeJSON(w, http.StatusOK, paginate(teams, page, limit))
			return
		}
		writeJSON(w, http.StatusOK, teams)
	case http.MethodPost:
		var t store.Team
		if err := decodeBody(r, &t); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		created, err := s.store.CreateTeam(t)
		writeMutation(w, created, http.StatusCreated, err)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET or POST")
	}
}

func (s *Server) handleAdminTeamRoutes(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(strings.TrimPrefix(r.URL.Path, "/api/admin/teams/"))
	if len(parts) == 0 {
		writeError(w, http.StatusNotFound, "not_found", "team route not found")
		return
	}
	key := parts[0]
	switch {
	case len(parts) == 1:
		// View: super or any member of the team. Mutations: leader or super.
		user, ok := s.requireUser(w, r)
		if !ok {
			return
		}
		tm, err := s.store.Team(key)
		if err != nil {
			writeResult(w, tm, err)
			return
		}
		isMember := false
		for _, m := range tm.Members {
			if strings.EqualFold(m, user.Username) || strings.EqualFold(m, user.ID) {
				isMember = true
				break
			}
		}
		if !s.auth.IsSuperAdmin(user) && !isMember {
			writeError(w, http.StatusForbidden, "forbidden", "team membership or super required")
			return
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, tm)
		case http.MethodPut:
			if !s.auth.IsSuperAdmin(user) && !s.isTeamLeader(user, key) {
				writeError(w, http.StatusForbidden, "forbidden", "only leader or super can update team")
				return
			}
			var t store.Team
			if err := decodeBody(r, &t); err != nil {
				writeError(w, http.StatusBadRequest, "bad_request", err.Error())
				return
			}
			updated, err := s.store.UpdateTeam(key, t)
			writeMutation(w, updated, http.StatusOK, err)
		case http.MethodDelete:
			if _, ok := s.requireSuperAdmin(w, r); !ok {
				return
			}
			if err := s.store.DeleteTeam(key); err != nil {
				writeResult(w, nil, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"status": "deleted", "key": key})
		default:
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET/PUT/DELETE")
		}
	case len(parts) == 2 && parts[1] == "members" && r.Method == http.MethodPost:
		// Leader or super can pull (add) members.
		user, ok := s.requireUser(w, r)
		if !ok {
			return
		}
		if !s.auth.IsSuperAdmin(user) && !s.isTeamLeader(user, key) {
			writeError(w, http.StatusForbidden, "forbidden", "only team leader or super can add members")
			return
		}
		var req struct {
			Username string `json:"username"`
		}
		if err := decodeBody(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		if strings.TrimSpace(req.Username) == "" {
			writeError(w, http.StatusBadRequest, "invalid_input", "username required")
			return
		}
		updated, err := s.store.AddTeamMember(key, req.Username)
		writeMutation(w, updated, http.StatusOK, err)
	case len(parts) == 3 && parts[1] == "members" && r.Method == http.MethodDelete:
		// Leader or super can remove member.
		user, ok := s.requireUser(w, r)
		if !ok {
			return
		}
		if !s.auth.IsSuperAdmin(user) && !s.isTeamLeader(user, key) {
			writeError(w, http.StatusForbidden, "forbidden", "only team leader or super can remove members")
			return
		}
		member := parts[2]
		updated, err := s.store.RemoveTeamMember(key, member)
		writeMutation(w, updated, http.StatusOK, err)
	default:
		writeError(w, http.StatusNotFound, "not_found", "team route not found")
	}
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

func (s *Server) handleGitLabWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	secret := r.Header.Get("X-Gitlab-Token")
	globalSecret := os.Getenv("GITLAB_WEBHOOK_SECRET")
	if globalSecret != "" && secret != globalSecret {
		writeError(w, http.StatusForbidden, "forbidden", "invalid webhook secret")
		return
	}
	var payload struct {
		ObjectKind string `json:"object_kind"`
		Project    struct {
			PathWithNamespace string `json:"path_with_namespace"`
		} `json:"project"`
		Commits []struct {
			ID        string `json:"id"`
			Message   string `json:"message"`
			Timestamp string `json:"timestamp"`
		} `json:"commits"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if payload.ObjectKind == "push" && len(payload.Commits) > 0 {
		latest := payload.Commits[0]
		log.Printf("GitLab webhook push for %s, latest commit %s: %s", payload.Project.PathWithNamespace, latest.ID, latest.Message)
		// optionally update module last push info here if repo matches a module
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "received", "kind": payload.ObjectKind})
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

func toStoreArtifact(a deploy.Artifact) store.DeployArtifact {
	out := store.DeployArtifact{
		ModuleKey:      a.Metadata.ModuleKey,
		ModuleName:     a.Metadata.ModuleName,
		DocsVersion:    a.Metadata.DocsVersion,
		PackageVersion: a.Metadata.PackageVersion,
		Description:    a.Metadata.Description,
		Authors:        append([]string(nil), a.Metadata.Authors...),
		Edition:        a.Metadata.Edition,
		Keywords:       append([]string(nil), a.Metadata.Keywords...),
		RepoURL:        a.Metadata.RepoURL,
		RepoType:       a.Metadata.RepoType,
		Branch:         a.Metadata.Branch,
		CommitSHA:      a.Metadata.CommitSHA,
		Bytes:          a.Bytes,
		SiteHTML:       map[string]string{},
		SiteFiles:      map[string][]byte{},
	}
	for _, e := range a.Manifest.Entries {
		out.Entries = append(out.Entries, store.DeployEntry{Key: e.Key, Title: e.Title, Type: e.Type, Source: e.Source, Output: e.Output})
	}
	for _, d := range a.Documents {
		out.Documents = append(out.Documents, store.DeployDocument{
			DocID: d.DocID, ModuleKey: d.ModuleKey, ModuleName: d.ModuleName, DocsVersion: d.DocsVersion,
			PackageVersion: d.PackageVersion, EntryKey: d.EntryKey, EntryType: d.EntryType, Title: d.Title,
			Description: d.Description, Content: d.Content, ContentMD: d.ContentMD, Path: d.Path, SourceFile: d.SourceFile,
			Keywords: append([]string(nil), d.Keywords...), Status: d.Status,
		})
	}
	for _, n := range a.Nav {
		out.Nav = append(out.Nav, toStoreNav(n))
	}
	for name, html := range a.SiteHTML {
		out.SiteHTML[name] = html
	}
	for name, content := range a.SiteFiles {
		// Hand over the bytes directly instead of cloning: the source artifact is
		// discarded right after ingestion, and cloning a large (image/GIF-heavy)
		// site here doubles peak memory and can OOM the backend on big deploys.
		out.SiteFiles[name] = content
	}
	return out
}

func toStoreNav(n deploy.NavItem) store.NavItem {
	out := store.NavItem{Title: n.Title, Path: n.Path}
	for _, child := range n.Children {
		out.Children = append(out.Children, toStoreNav(child))
	}
	return out
}

func (s *Server) uploadSiteFilesToMinIO(ctx context.Context, artifact deploy.Artifact, moduleKey, docsVersion string) error {
	if s.minioClient == nil {
		return nil
	}
	for name, content := range artifact.SiteFiles {
		key := fmt.Sprintf("modules/%s/%s/%s", moduleKey, docsVersion, name)
		ct := contentTypeForName(name, content)
		_, err := s.minioClient.PutObject(ctx, s.minioBucket, key, bytes.NewReader(content), int64(len(content)), minio.PutObjectOptions{
			ContentType: ct,
		})
		if err != nil {
			return fmt.Errorf("upload %s: %w", key, err)
		}
	}
	return nil
}

func (s *Server) migrateSiteAssetsToMinIO() bool {
	objects := s.store.SiteObjects()
	if len(objects) == 0 {
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	for key, file := range objects {
		_, err := s.minioClient.PutObject(ctx, s.minioBucket, key, bytes.NewReader(file.Content), int64(len(file.Content)), minio.PutObjectOptions{ContentType: file.ContentType})
		if err != nil {
			log.Printf("legacy site asset migration failed for %s: %v", key, err)
			return false
		}
	}
	log.Printf("migrated %d legacy site assets to MinIO", len(objects))
	return true
}

func contentTypeForName(name string, content []byte) string {
	if ct := mime.TypeByExtension(filepath.Ext(name)); ct != "" {
		return ct
	}
	if len(content) > 0 {
		return http.DetectContentType(content)
	}
	return "application/octet-stream"
}

func envFloat(key string, fallback float64) float64 {
	v, err := strconv.ParseFloat(os.Getenv(key), 64)
	if err != nil || v == 0 {
		return fallback
	}
	return v
}

func envInt64(key string, fallback int64) int64 {
	v, err := strconv.ParseInt(os.Getenv(key), 10, 64)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.Index(xff, ","); i != -1 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-Ip"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func bearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if strings.HasPrefix(auth, prefix) {
		return strings.TrimSpace(auth[len(prefix):])
	}
	return ""
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
