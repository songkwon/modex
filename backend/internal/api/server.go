package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"modex/backend/internal/application"
	"modex/backend/internal/redisurl"
	"modex/backend/internal/search"
	"modex/backend/internal/store"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
)

// dialRedis builds the Redis client shared by the rate and deploy limiters,
// or returns nil (so both fall back to in-process behavior) when Redis is
// unconfigured, malformed, or unreachable.
func dialRedis() *redis.Client {
	rawURL := redisurl.FromEnv()
	if rawURL == "" {
		return nil
	}
	options, err := redis.ParseURL(rawURL)
	if err != nil {
		log.Printf("redis: invalid connection settings (%v); using in-process limiters", err)
		return nil
	}
	client := redis.NewClient(options)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("redis: ping failed (%v); using in-process limiters", err)
		_ = client.Close()
		return nil
	}
	return client
}

type Server struct {
	app         *application.Service
	minioClient *minio.Client
	minioBucket string
	limiter     rateLimiter
	// deploy bounds how many artifacts are ingested at once so a burst of large
	// uploads can't exhaust memory/CPU. Redis-backed when Redis is configured (a
	// single global bound across replicas), per-process otherwise. Requests
	// beyond the bound wait briefly, then get 503. nil means unbounded.
	deploy deployLimiter
}

func New(st store.DataStore) *Server {
	return NewWithVectorStore(st, nil)
}

func NewWithVectorStore(st store.DataStore, vectors search.VectorStore) *Server {
	return NewWithApplication(application.New(st, vectors, nil))
}

func NewWithApplication(app *application.Service) *Server {
	redisClient := dialRedis()
	s := &Server{
		app:     app,
		limiter: newRateLimiter(redisClient),
		deploy:  newDeployLimiter(redisClient),
	}
	// init MinIO for real site file storage (upload SiteFiles from deploys)
	if endpoint := os.Getenv("MINIO_ENDPOINT"); endpoint != "" {
		accessKey := os.Getenv("MINIO_ROOT_USER")
		secretKey := os.Getenv("MINIO_ROOT_PASSWORD")
		secure := strings.HasPrefix(strings.ToLower(endpoint), "https://")
		// minio-go expects a bare host:port; strip any scheme/trailing slash so
		// MINIO_ENDPOINT=http://minio:9000 doesn't fail init ("Endpoint url
		// cannot have fully qualified paths") and fall back to database assets.
		host := endpoint
		if i := strings.Index(host, "://"); i >= 0 {
			host = host[i+3:]
		}
		host = strings.TrimRight(host, "/")
		client, err := minio.New(host, &minio.Options{
			Creds:        credentials.NewStaticV4(accessKey, secretKey, ""),
			Secure:       secure,
			Region:       os.Getenv("MINIO_REGION"),
			BucketLookup: minioBucketLookup(),
		})
		if err == nil {
			if envBool("MINIO_TRACE", false) {
				client.TraceOn(os.Stderr)
			}
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
	return s
}

func minioBucketLookup() minio.BucketLookupType {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("MINIO_BUCKET_LOOKUP"))) {
	case "", "path":
		return minio.BucketLookupPath
	case "auto":
		return minio.BucketLookupAuto
	case "dns", "virtual-host", "virtualhost":
		return minio.BucketLookupDNS
	default:
		log.Printf("minio: unknown MINIO_BUCKET_LOOKUP=%q; using path-style bucket lookup", os.Getenv("MINIO_BUCKET_LOOKUP"))
		return minio.BucketLookupPath
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/api/auth/me", s.handleMe)
	mux.HandleFunc("/api/me/mcp-token", s.handleMeMCPToken)
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
	mux.HandleFunc("/api/analytics/page-view", s.handlePageView)
	mux.HandleFunc("/api/analytics/read-progress", s.handleReadProgress)
	mux.HandleFunc("/api/analytics/feedback", s.handleDocFeedback)
	mux.HandleFunc("/api/analytics/doc", s.handleDocAnalytics)
	mux.HandleFunc("/api/admin/releases", s.handleReleases)
	mux.HandleFunc("/api/admin/releases/", s.handleReleaseRoutes)
	mux.HandleFunc("/api/admin/analytics/feedback", s.handleDocFeedbackLogs)
	mux.HandleFunc("/api/admin/analytics/search", s.handleSearchLogs)
	mux.HandleFunc("/api/admin/analytics/mcp", s.handleMCPLogs)
	mux.HandleFunc("/api/admin/analytics/pages", http.NotFound)
	mux.HandleFunc("/api/mcp/log", s.handleMCPLog)
	mux.HandleFunc("/api/me/favorites", s.handleMeFavorites)
	mux.HandleFunc("/api/me/recent", s.handleMeRecent)
	mux.HandleFunc("/api/mcp/dist", s.handleMcpDist)
	mux.HandleFunc("/api/mcp/dist/", s.handleMcpDist)
	mux.HandleFunc("/.well-known/agent-skills/index.json", s.handleSkillDiscovery)
	mux.HandleFunc("/.well-known/skills/index.json", s.handleSkillDiscovery)
	mux.HandleFunc("/api/admin/settings/models", s.handleAdminModels)
	mux.HandleFunc("/api/admin/settings/test-connection", s.handleAdminModelConnectionTest)
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
	mux.HandleFunc("/api/admin/teams", s.handleAdminTeams)
	mux.HandleFunc("/api/admin/teams/", s.handleAdminTeamRoutes)
	mux.HandleFunc("/api/admin/", s.handleAdminAccepted)
	mux.HandleFunc("/api/webhooks/gitlab", s.handleGitLabWebhook)
	return s.cors(accessLogger(recoverer(s.requestGuards(mux))))
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	embeddingCount, embeddingErr := s.app.Search().EmbeddingCount(ctx)
	sessionErr := s.app.Auth().Healthy(ctx)
	searchStatus := "ok"
	if embeddingErr != nil {
		searchStatus = "degraded"
	}
	status := "ok"
	statusCode := http.StatusOK
	if sessionErr != nil {
		status = "unavailable"
		statusCode = http.StatusServiceUnavailable
	}
	writeJSON(w, statusCode, map[string]any{
		"status":  status,
		"service": "modex-api",
		"dependencies": map[string]any{
			"repository": map[string]any{"configured": true},
			"sessions":   map[string]any{"status": ternary(sessionErr == nil, "ok", "unavailable"), "error": errorString(sessionErr)},
			"object_storage": map[string]any{
				"mode":   ternary(s.minioClient != nil, "minio", "postgres"),
				"bucket": s.minioBucket,
			},
			"search": map[string]any{
				"status":          searchStatus,
				"provider":        s.app.Search().Embedder.Name(),
				"external_vector": s.app.Search().Vectors != nil,
				"embedding_count": embeddingCount,
				"error":           errorString(embeddingErr),
			},
		},
		"counts": map[string]int{
			"modules":  len(s.app.Store().Modules("", "")),
			"pages":    len(s.app.Store().Pages()),
			"releases": len(s.app.Store().Releases()),
		},
	})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	if user, ok := s.currentUser(r); ok {
		writeJSON(w, http.StatusOK, struct {
			store.User
			IsSuperAdmin bool `json:"is_super_admin"`
			IsTeamAdmin  bool `json:"is_team_admin"`
		}{User: user, IsSuperAdmin: s.app.Auth().IsSuperAdmin(user), IsTeamAdmin: s.isTeamAdmin(user)})
		return
	}
	if r.URL.Query().Get("optional") == "1" {
		writeJSON(w, http.StatusOK, nil)
		return
	}
	writeError(w, http.StatusUnauthorized, "unauthorized", "not logged in")
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	loginURL, err := s.app.Auth().BeginLogin(w)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "oidc_not_configured", err.Error())
		return
	}
	http.Redirect(w, r, loginURL, http.StatusFound)
}

func (s *Server) handleCallback(w http.ResponseWriter, r *http.Request) {
	user, err := s.app.Auth().CompleteLogin(r.Context(), r, w)
	frontend := s.app.Auth().Config().FrontendBaseURL
	if err != nil {
		// Surface the failure to the user in the portal rather than a bare JSON
		// 400, and log the detail server-side for diagnostics.
		log.Printf("oidc callback failed: %v", err)
		http.Redirect(w, r, frontend+"/?login_error="+url.QueryEscape(err.Error()), http.StatusFound)
		return
	}
	// Sync the SSO identity into the user directory.
	s.app.Store().UpsertUser(user)
	http.Redirect(w, r, frontend, http.StatusFound)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.app.Auth().Logout(w, r)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	cfg := s.app.Auth().Config()
	loginURL := ""
	// Advertise the login URL only once the OIDC provider is fully configured.
	if cfg.LoginReady() {
		loginURL = cfg.AppBaseURL + "/api/auth/login"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"oidc_login_enabled": cfg.LoginReady(),
		"login_url":          loginURL,
		"app_base_url":       cfg.AppBaseURL,
		"frontend_base_url":  cfg.FrontendBaseURL,
		"auto_login":         cfg.AutoLogin,
		// Effective doc-engine plugin state (enabled + non-secret config) so the
		// renderer can conditionally apply plugins without admin rights.
		"plugins": s.app.Store().PluginEffective(),
	})
}

func (s *Server) handleCategories(w http.ResponseWriter, r *http.Request) {
	tree := s.app.Store().CategoryTree()
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
	writeJSON(w, http.StatusOK, s.app.Store().Modules(r.URL.Query().Get("category_id"), r.URL.Query().Get("keyword")))
}

func (s *Server) handleModuleRoutes(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(strings.TrimPrefix(r.URL.Path, "/api/modules/"))
	if len(parts) == 0 {
		writeError(w, http.StatusNotFound, "not_found", "module route not found")
		return
	}
	moduleKey := parts[0]
	if len(parts) == 1 || (len(parts) == 2 && parts[1] == "info") {
		m, err := s.app.Store().Module(moduleKey)
		writeResult(w, m, err)
		return
	}
	if len(parts) == 2 && parts[1] == "versions" {
		writeJSON(w, http.StatusOK, s.app.Store().Versions(moduleKey))
		return
	}
	if len(parts) == 4 && parts[1] == "versions" && parts[3] == "entries" {
		entries := s.app.Store().Entries(moduleKey, parts[2])
		if entries == nil {
			entries = []store.Entry{}
		}
		writeJSON(w, http.StatusOK, entries)
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
	module, err := s.app.Store().Module(parts[0])
	if err != nil {
		writeResult(w, module, err)
		return
	}
	if len(parts) == 1 {
		writeJSON(w, http.StatusOK, map[string]any{"module": module, "redirect_to": "/docs/" + module.ModuleKey + "/" + module.DefaultVersion})
		return
	}
	if len(parts) == 2 {
		entries := s.app.Store().Entries(module.ModuleKey, parts[1])
		if entries == nil {
			entries = []store.Entry{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"module": module, "version": parts[1], "entries": entries})
		return
	}
	if len(parts) >= 3 {
		if len(parts) >= 4 && parts[3] == "site" {
			s.handleDocSiteFile(w, r, module.ModuleKey, parts[1], parts[2], strings.Join(parts[4:], "/"))
			return
		}
		if len(parts) == 4 && parts[3] == "nav" {
			nav := s.app.Store().Nav(module.ModuleKey, parts[1])
			if len(nav) == 0 {
				writeJSON(w, http.StatusOK, []map[string]string{{"title": "概览", "path": "#overview"}, {"title": "正文", "path": "#content"}, {"title": "元数据", "path": "#metadata"}})
				return
			}
			writeJSON(w, http.StatusOK, nav)
			return
		}
		page, err := s.app.Store().PageByRoute(module.ModuleKey, parts[1], parts[2])
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
	if s.minioClient != nil {
		for _, candidate := range siteFileCandidates(name) {
			zipName := fmt.Sprintf("site/%s/%s", entryKey, candidate)
			key := fmt.Sprintf("modules/%s/%s/%s", moduleKey, docsVersion, zipName)
			obj, err := s.minioClient.GetObject(r.Context(), s.minioBucket, key, minio.GetObjectOptions{})
			if err != nil {
				continue
			}
			defer obj.Close()
			head := make([]byte, 512)
			n, readErr := obj.Read(head)
			if readErr == nil || readErr == io.EOF {
				head = head[:n]
				contentType := contentTypeForName(candidate, head)
				w.Header().Set("Cache-Control", "private, max-age=60")
				if shouldRewriteServedSiteFile(candidate, contentType) {
					var body []byte
					body = append(body, head...)
					if readErr != io.EOF {
						rest, err := io.ReadAll(obj)
						if err != nil {
							log.Printf("read site asset %s failed: %v", name, err)
							writeError(w, http.StatusBadGateway, "site_read_failed", err.Error())
							return
						}
						body = append(body, rest...)
					}
					body = rewriteServedSiteRootRefs(body, moduleKey, docsVersion, entryKey)
					w.Header().Set("Content-Type", contentTypeForName(candidate, body))
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write(body)
					return
				}
				w.Header().Set("Content-Type", contentType)
				w.WriteHeader(http.StatusOK)
				if len(head) > 0 {
					_, _ = w.Write(head)
				}
				if readErr != io.EOF {
					if _, err := io.Copy(w, obj); err != nil {
						log.Printf("stream site asset %s failed: %v", name, err)
					}
				}
				return
			}
			log.Printf("minio site asset read failed bucket=%s key=%s error=%v", s.minioBucket, key, readErr)
		}
	}
	// Without MinIO, static assets are read from PostgreSQL.
	var f store.SiteFile
	var found bool
	for _, candidate := range siteFileCandidates(name) {
		next, err := s.app.Store().SiteFile(moduleKey, docsVersion, entryKey, candidate)
		if err == nil {
			f = next
			found = true
			break
		}
	}
	if !found {
		writeResult(w, nil, store.ErrNotFound)
		return
	}
	w.Header().Set("Content-Type", f.ContentType)
	w.Header().Set("Cache-Control", "private, max-age=60")
	if shouldRewriteServedSiteFile(name, f.ContentType) {
		f.Content = rewriteServedSiteRootRefs(f.Content, moduleKey, docsVersion, entryKey)
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(f.Content)
}

func siteFileCandidates(name string) []string {
	clean := path.Clean(strings.TrimPrefix(name, "/"))
	if clean == "." || clean == "" {
		clean = "index.html"
	}
	candidates := []string{clean}
	add := func(value string) {
		value = path.Clean(strings.TrimPrefix(value, "/"))
		if value == "." || value == "" {
			value = "index.html"
		}
		for _, existing := range candidates {
			if existing == value {
				return
			}
		}
		candidates = append(candidates, value)
	}
	ext := path.Ext(clean)
	if ext == "" {
		add(clean + ".html")
		add(path.Join(clean, "index.html"))
		add("index.html")
	}
	if strings.EqualFold(ext, ".md") {
		add(strings.TrimSuffix(clean, ext) + ".html")
	}
	return candidates
}

func shouldRewriteServedSiteFile(name, contentType string) bool {
	name = strings.ToLower(name)
	contentType = strings.ToLower(contentType)
	if strings.HasSuffix(name, ".html") || strings.HasSuffix(name, ".htm") ||
		strings.HasSuffix(name, ".css") || strings.HasSuffix(name, ".js") ||
		strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".webmanifest") {
		return true
	}
	return strings.HasPrefix(contentType, "text/") ||
		strings.Contains(contentType, "javascript") ||
		strings.Contains(contentType, "json")
}

var (
	servedSiteAttrRootRef = regexp.MustCompile(`(?i)(\b(?:href|src|poster|action)\s*=\s*["'])(/[^"'<>]*)(["'])`)
	servedSiteCSSRootRef  = regexp.MustCompile(`(?i)(url\(\s*["']?)(/[^)'"\s]+)(["']?\s*\))`)
	servedSiteSrcset      = regexp.MustCompile(`(?i)(\bsrcset\s*=\s*["'])([^"'<>]*)(["'])`)
	servedSiteStringRef   = regexp.MustCompile("([\"'`])(/internal-tools/[^\"'`<>\\\\\\s]*)([\"'`])")
)

func rewriteServedSiteRootRefs(content []byte, moduleKey, docsVersion, entryKey string) []byte {
	if len(content) == 0 {
		return content
	}
	siteBase := "/api/docs/" + moduleKey + "/" + docsVersion + "/" + entryKey + "/site/"
	text := string(content)
	text = rewriteServedSiteMatches(text, servedSiteAttrRootRef, siteBase)
	text = rewriteServedSiteMatches(text, servedSiteCSSRootRef, siteBase)
	text = rewriteServedSiteStringRefs(text, siteBase)
	text = servedSiteSrcset.ReplaceAllStringFunc(text, func(match string) string {
		parts := servedSiteSrcset.FindStringSubmatch(match)
		if len(parts) != 4 {
			return match
		}
		candidates := strings.Split(parts[2], ",")
		for i, candidate := range candidates {
			fields := strings.Fields(strings.TrimSpace(candidate))
			if len(fields) == 0 {
				continue
			}
			if rewritten, ok := rewriteServedSiteRootURL(fields[0], siteBase); ok {
				fields[0] = rewritten
				candidates[i] = strings.Join(fields, " ")
			}
		}
		return parts[1] + strings.Join(candidates, ", ") + parts[3]
	})
	return []byte(text)
}

func rewriteServedSiteStringRefs(input, siteBase string) string {
	return servedSiteStringRef.ReplaceAllStringFunc(input, func(match string) string {
		parts := servedSiteStringRef.FindStringSubmatch(match)
		if len(parts) != 4 || parts[1] != parts[3] {
			return match
		}
		rewritten, ok := rewriteServedSiteRootURL(parts[2], siteBase)
		if !ok {
			return match
		}
		return parts[1] + rewritten + parts[3]
	})
}

func rewriteServedSiteMatches(input string, pattern *regexp.Regexp, siteBase string) string {
	return pattern.ReplaceAllStringFunc(input, func(match string) string {
		parts := pattern.FindStringSubmatch(match)
		if len(parts) != 4 {
			return match
		}
		rewritten, ok := rewriteServedSiteRootURL(parts[2], siteBase)
		if !ok {
			return match
		}
		return parts[1] + rewritten + parts[3]
	})
}

func rewriteServedSiteRootURL(raw, siteBase string) (string, bool) {
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") ||
		strings.HasPrefix(raw, siteBase) || strings.HasPrefix(raw, "/api/docs/") ||
		strings.HasPrefix(raw, "/_next/") || strings.HasPrefix(raw, "/brand/") {
		return raw, false
	}
	pathPart, trailer := splitServedSiteURLPath(raw)
	rel := strings.TrimPrefix(pathPart, "/")
	if rel == "" {
		return siteBase, true
	}
	parts := strings.Split(rel, "/")
	if len(parts) >= 2 && (shouldStripServedSiteLegacyBase(parts[0]) || shouldStripServedSiteLegacyPrefix(parts[1])) {
		rel = strings.Join(parts[1:], "/")
	}
	if rel == "" {
		return siteBase, true
	}
	return siteBase + rel + trailer, true
}

func splitServedSiteURLPath(raw string) (string, string) {
	if i := strings.IndexAny(raw, "?#"); i >= 0 {
		return raw[:i], raw[i:]
	}
	return raw, ""
}

func shouldStripServedSiteLegacyPrefix(next string) bool {
	if next == "assets" || next == "images" || next == "posts" {
		return true
	}
	return strings.Contains(next, ".")
}

func shouldStripServedSiteLegacyBase(first string) bool {
	switch strings.Trim(first, "/") {
	case "internal-tools":
		return true
	default:
		return false
	}
}

func (s *Server) handleDocPage(w http.ResponseWriter, r *http.Request) {
	docID := strings.TrimPrefix(r.URL.Path, "/api/docs/page/")
	page, err := s.app.Store().Page(docID)
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
			b, readErr := io.ReadAll(obj)
			if readErr == nil {
				return string(b), nil
			}
			err = readErr
		}
		if fallback := s.app.Store().PageHTML(moduleKey, docsVersion, entryKey); fallback != "" {
			return fallback, nil
		}
		// Markdown/static entries legitimately have no site/index.html object: they
		// render from content_md/content_text on the frontend, and site-builder
		// entries render from the iframe (site route), not this HTML. So a missing
		// object is not fatal — degrade to empty HTML instead of failing the whole
		// page render. Only log so genuine MinIO outages remain visible.
		if err != nil {
			log.Printf("minio page html read failed bucket=%s key=%s error=%v", s.minioBucket, key, err)
		}
		return "", nil
	}
	return s.app.Store().PageHTML(moduleKey, docsVersion, entryKey), nil
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
	resp, err := s.app.Search().Search(r.Context(), req)
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
		s.app.Store().AddSearchLog(store.SearchLog{ID: fmt.Sprintf("sl-%d", time.Now().UnixNano()), UserID: user.ID, IPAddress: clientIP(r), Query: req.Query, Mode: string(resp.Mode), FiltersJSON: string(filters), ResultCount: resp.Total, ClickedDocID: req.ClickedDocID, SearchedAt: time.Now().UTC()})
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleAsk answers a natural-language question using retrieval over the docs
// (RAG). The answer model is configured by an administrator; when it is absent
// or unavailable, the endpoint returns an extractive answer from the top hits.
func (s *Server) handleAsk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	var req struct {
		Query       string   `json:"query"`
		ModuleKey   string   `json:"module_key"`
		CategoryIDs []string `json:"category_ids"`
		Stream      bool     `json:"stream"`
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
	if req.Stream {
		s.handleAskStream(w, r, req.Query, filters)
		return
	}
	resp, err := s.app.Search().Search(r.Context(), search.Request{Query: req.Query, Mode: search.ModeHybrid, Filters: filters, Page: 1, PageSize: 8, DefaultVersionsOnly: len(filters.DocsVersions) == 0})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ask_failed", err.Error())
		return
	}
	answer, provider, warning := s.synthesizeAnswer(r.Context(), req.Query, resp.Results)
	user, _ := s.currentUser(r)
	filtersJSON, _ := json.Marshal(filters)
	s.app.Store().AddSearchLog(store.SearchLog{ID: fmt.Sprintf("ask-%d", time.Now().UnixNano()), UserID: user.ID, IPAddress: clientIP(r), Query: req.Query, Mode: "ask", FiltersJSON: string(filtersJSON), ResultCount: len(resp.Results), SearchedAt: time.Now().UTC()})
	payload := map[string]any{
		"query":    req.Query,
		"answer":   answer,
		"provider": provider,
		"sources":  resp.Results,
	}
	if warning != "" {
		payload["warning"] = warning
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) handleAskStream(w http.ResponseWriter, r *http.Request, query string, filters search.Filters) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "stream_unavailable", "streaming is not supported")
		return
	}
	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	writeEvent := func(event map[string]any) bool {
		if err := json.NewEncoder(w).Encode(event); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}
	resp, err := s.app.Search().Search(r.Context(), search.Request{Query: query, Mode: search.ModeHybrid, Filters: filters, Page: 1, PageSize: 8, DefaultVersionsOnly: len(filters.DocsVersions) == 0})
	if err != nil {
		writeEvent(map[string]any{"type": "error", "error": err.Error()})
		return
	}
	if !writeEvent(map[string]any{"type": "sources", "query": query, "sources": resp.Results}) {
		return
	}
	provider, warning := s.synthesizeAnswerStream(r.Context(), query, resp.Results, func(delta string) bool {
		return writeEvent(map[string]any{"type": "delta", "delta": delta})
	})
	if provider == "" {
		return
	}
	user, _ := s.currentUser(r)
	filtersJSON, _ := json.Marshal(filters)
	s.app.Store().AddSearchLog(store.SearchLog{ID: fmt.Sprintf("ask-%d", time.Now().UnixNano()), UserID: user.ID, IPAddress: clientIP(r), Query: query, Mode: "ask", FiltersJSON: string(filtersJSON), ResultCount: len(resp.Results), SearchedAt: time.Now().UTC()})
	if !writeEvent(map[string]any{"type": "meta", "provider": provider, "warning": warning}) {
		return
	}
	writeEvent(map[string]any{"type": "done"})
}

func streamTextChunks(text string, size int) []string {
	if size <= 0 {
		size = 48
	}
	runes := []rune(text)
	if len(runes) == 0 {
		return nil
	}
	chunks := make([]string, 0, (len(runes)/size)+1)
	for start := 0; start < len(runes); start += size {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
	}
	return chunks
}

func (s *Server) synthesizeAnswerStream(ctx context.Context, query string, results []search.Result, onDelta func(string) bool) (string, string) {
	ai := s.app.Store().Settings().AI
	if strings.TrimSpace(ai.AskBaseURL) != "" && strings.TrimSpace(ai.AskModel) != "" {
		if err := s.askOpenAICompatibleStream(ctx, ai, query, results, onDelta); err == nil {
			return "llm", ""
		} else {
			log.Printf("ask llm stream failed: %v", err)
			answer := s.extractiveAnswer(results)
			for _, chunk := range streamTextChunks(answer, 48) {
				if !onDelta(chunk) {
					return "", ""
				}
			}
			return "extractive", "大模型流式调用失败，已退回本地文档摘要：" + err.Error()
		}
	}
	answer := s.extractiveAnswer(results)
	for _, chunk := range streamTextChunks(answer, 48) {
		if !onDelta(chunk) {
			return "", ""
		}
	}
	return "extractive", "问答大模型未配置完整，请在模型设置中配置 API Base URL 和模型名称。"
}

func (s *Server) synthesizeAnswer(ctx context.Context, query string, results []search.Result) (string, string, string) {
	// 1) Admin-configured OpenAI-compatible chat model (preferred).
	if ai := s.app.Store().Settings().AI; strings.TrimSpace(ai.AskBaseURL) != "" && strings.TrimSpace(ai.AskModel) != "" {
		if answer, err := s.askOpenAICompatible(ctx, ai, query, results); err == nil && strings.TrimSpace(answer) != "" {
			return answer, "llm", ""
		} else if err != nil {
			log.Printf("ask llm failed: %v", err)
			return s.extractiveAnswer(results), "extractive", "大模型调用失败，已退回本地文档摘要：" + err.Error()
		}
	} else {
		return s.extractiveAnswer(results), "extractive", "问答大模型未配置完整，请在模型设置中配置 API Base URL 和模型名称。"
	}
	return s.extractiveAnswer(results), "extractive", "大模型返回空答案，已退回本地文档摘要。"
}

func (s *Server) extractiveAnswer(results []search.Result) string {
	if len(results) == 0 {
		return "未在文档库中找到与该问题相关的内容。可以换个关键词，或在左侧按平台浏览。"
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
	return b.String()
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

func (s *Server) askOpenAICompatibleStream(ctx context.Context, ai store.AISettings, query string, results []search.Result, onDelta func(string) bool) error {
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
	return chatCompleteStream(ctx, ai, system, userMsg, onDelta)
}

func (s *Server) askContextForResult(i int, r search.Result) string {
	content := firstNonEmptyStr(r.Snippet, r.Title)
	if p, err := s.app.Store().Page(r.DocID); err == nil {
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
		writeJSON(w, http.StatusOK, maskedSettings(s.app.Store().Settings()))
	case http.MethodPut, http.MethodPost:
		var body store.AISettings
		if err := decodeBody(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		saved := s.app.Store().SaveAISettings(body)
		s.writeMutation(w, maskedSettings(saved), http.StatusOK, nil)
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
	ai := s.app.Store().Settings().AI
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
	resp, err := s.app.Search().Search(r.Context(), search.Request{
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
		writeJSON(w, http.StatusOK, map[string]any{"plugins": s.app.Store().PluginStates()})
	case http.MethodPut, http.MethodPost:
		var body struct {
			Plugins map[string]store.PluginSetting `json:"plugins"`
		}
		if err := decodeBody(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		s.writeMutation(w, map[string]any{"plugins": s.app.Store().SavePluginSettings(body.Plugins)}, http.StatusOK, nil)
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
		saved, err := s.app.Store().SaveUploadedPlugin(body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_plugin", err.Error())
			return
		}
		s.writeMutation(w, map[string]any{"plugin": saved, "plugins": s.app.Store().PluginStates()}, http.StatusOK, nil)
	case http.MethodDelete:
		if key == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "plugin key required")
			return
		}
		if !s.app.Store().DeleteUploadedPlugin(key) {
			writeError(w, http.StatusNotFound, "not_found", "plugin not found")
			return
		}
		s.writeMutation(w, map[string]any{"status": "deleted", "key": key, "plugins": s.app.Store().PluginStates()}, http.StatusOK, nil)
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
	writeJSON(w, http.StatusOK, map[string]any{"plugins": s.app.Store().EnabledUploadedPlugins()})
}

// handleAdminSnippets manages the reusable snippet library and variables.
// Super-admin only. GET returns the current set; PUT replaces it.
func (s *Server) handleAdminSnippets(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSuperAdmin(w, r); !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		snips, vars := s.app.Store().SnippetData()
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
		snips, vars := s.app.Store().SaveSnippetData(body.Snippets, body.Variables)
		s.writeMutation(w, map[string]any{"snippets": snips, "variables": vars}, http.StatusOK, nil)
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
	snips, vars := s.app.Store().SnippetData()
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
	cur := s.app.Store().Settings().AI
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

func (s *Server) handleAdminModelConnectionTest(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSuperAdmin(w, r); !ok {
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	var body struct {
		Kind     string `json:"kind"`
		Protocol string `json:"protocol"`
		BaseURL  string `json:"base_url"`
		Model    string `json:"model"`
		APIKey   string `json:"api_key"`
	}
	if err := decodeBody(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	kind := strings.TrimSpace(body.Kind)
	base := strings.TrimSpace(body.BaseURL)
	model := strings.TrimSpace(body.Model)
	if base == "" || model == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "base_url and model are required")
		return
	}
	ai := s.app.Store().Settings().AI
	key := strings.TrimSpace(body.APIKey)
	switch kind {
	case "chat":
		if key == "" {
			key = ai.AskAPIKey
		}
		protocol := strings.TrimSpace(body.Protocol)
		if protocol == "" {
			protocol = ai.AskProtocol
		}
		result, err := testChatEndpoint(r.Context(), protocol, base, model, key)
		if err != nil {
			writeError(w, http.StatusBadGateway, "chat_test_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, result)
	case "embedding":
		if key == "" {
			key = ai.EmbeddingAPIKey
		}
		result, err := testEmbeddingEndpoint(r.Context(), base, model, key)
		if err != nil {
			writeError(w, http.StatusBadGateway, "embedding_test_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, result)
	case "rerank":
		if key == "" {
			key = ai.RerankAPIKey
		}
		result, err := testRerankEndpoint(r.Context(), base, model, key)
		if err != nil {
			writeError(w, http.StatusBadGateway, "rerank_test_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, result)
	default:
		writeError(w, http.StatusBadRequest, "bad_request", "kind must be chat, embedding or rerank")
	}
}

func testChatEndpoint(ctx context.Context, protocol, base, model, key string) (map[string]any, error) {
	temp := 0.0
	answer, err := chatComplete(ctx, store.AISettings{
		AskProtocol:    protocol,
		AskBaseURL:     base,
		AskModel:       model,
		AskAPIKey:      key,
		AskMaxTokens:   64,
		AskTemperature: &temp,
	}, "你是连接测试助手。", "请只回复 OK。")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(answer) == "" {
		return nil, fmt.Errorf("chat endpoint returned an empty answer")
	}
	return map[string]any{
		"kind":     "chat",
		"status":   "ok",
		"endpoint": chatEndpoint(base, protocol, model),
		"model":    model,
		"sample":   truncateRunes(strings.TrimSpace(answer), 80),
	}, nil
}

func chatEndpoint(base, protocol, model string) string {
	endpoint := strings.TrimRight(strings.TrimSpace(base), "/")
	switch normalizeProtocol(protocol) {
	case protoAnthropic:
		return endpoint + "/v1/messages"
	case protoGemini:
		return endpoint + "/v1beta/models/" + url.PathEscape(model) + ":generateContent"
	case protoOpenAIResponses:
		return endpoint + "/responses"
	default:
		return endpoint + "/chat/completions"
	}
}

func testEmbeddingEndpoint(ctx context.Context, base, model, key string) (map[string]any, error) {
	endpoint := modelEndpoint(base, "/embeddings")
	raw, code, err := httpJSON(ctx, http.MethodPost, endpoint,
		map[string]string{"Content-Type": "application/json", "Authorization": bearer(key)},
		map[string]any{"model": model, "input": []string{"Modex embedding connection test"}})
	if err != nil {
		return nil, err
	}
	if code >= 300 {
		return nil, fmt.Errorf("%s returned HTTP %d: %s", endpoint, code, strings.TrimSpace(string(raw)))
	}
	var out struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode embedding response from %s: %w", endpoint, err)
	}
	if len(out.Data) == 0 || len(out.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("%s returned no embedding vector", endpoint)
	}
	return map[string]any{
		"kind":      "embedding",
		"status":    "ok",
		"endpoint":  endpoint,
		"model":     model,
		"dimension": len(out.Data[0].Embedding),
	}, nil
}

func testRerankEndpoint(ctx context.Context, base, model, key string) (map[string]any, error) {
	endpoint := modelEndpoint(base, "/rerank")
	raw, code, err := httpJSON(ctx, http.MethodPost, endpoint,
		map[string]string{"Content-Type": "application/json", "Authorization": bearer(key)},
		map[string]any{
			"model":     model,
			"query":     "Modex rerank connection test",
			"documents": []string{"Modex rerank connection test document", "unrelated document"},
			"top_n":     2,
		})
	if err != nil {
		return nil, err
	}
	if code >= 300 {
		return nil, fmt.Errorf("%s returned HTTP %d: %s", endpoint, code, strings.TrimSpace(string(raw)))
	}
	var out struct {
		Results []struct {
			Index          int     `json:"index"`
			RelevanceScore float64 `json:"relevance_score"`
		} `json:"results"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode rerank response from %s: %w", endpoint, err)
	}
	if len(out.Results) == 0 {
		return nil, fmt.Errorf("%s returned no rerank results", endpoint)
	}
	return map[string]any{
		"kind":      "rerank",
		"status":    "ok",
		"endpoint":  endpoint,
		"model":     model,
		"top_index": out.Results[0].Index,
		"score":     out.Results[0].RelevanceScore,
	}, nil
}

func modelEndpoint(base, suffix string) string {
	endpoint := strings.TrimRight(strings.TrimSpace(base), "/")
	if !strings.HasSuffix(endpoint, suffix) {
		endpoint += suffix
	}
	return endpoint
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
	resp, _ := s.app.Search().Search(r.Context(), search.Request{Mode: search.ModeKeyword, PageSize: 1})
	writeJSON(w, http.StatusOK, resp.Facets)
}

func (s *Server) handleEmbedText(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	if user, ok := s.requireUser(w, r); !ok {
		return
	} else if !isAdmin(user) && !s.app.Auth().IsSuperAdmin(user) {
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
	vec, err := s.app.Search().Embedder.EmbedText(r.Context(), req.Text)
	if err != nil {
		writeError(w, http.StatusBadGateway, "embedding_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"provider": s.app.Search().Embedder.Name(), "dimension": len(vec), "embedding": vec})
}

func (s *Server) handleEmbeddingReindex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	if _, ok := s.requireSuperAdmin(w, r); !ok {
		return
	}
	count, err := s.app.Search().Reindex(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "embedding_reindex_failed", err.Error())
		return
	}
	storedCount, err := s.app.Search().EmbeddingCount(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "embedding_count_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":           "reindexed",
		"provider":         s.app.Search().Embedder.Name(),
		"embedded_pages":   count,
		"cached_documents": storedCount,
	})
}

func (s *Server) handleSearchReindex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	if _, ok := s.requireSuperAdmin(w, r); !ok {
		return
	}
	// Keyword scoring reads the current PostgreSQL page set; reindexing rebuilds
	// the embedding data used by semantic and hybrid search.
	count, err := s.app.Search().Reindex(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "search_reindex_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":             "reindexed",
		"indexed_documents":  len(s.app.Store().Pages()),
		"embedded_documents": count,
	})
}
