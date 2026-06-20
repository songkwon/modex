package api

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"modex/backend/internal/deploy"
	"modex/backend/internal/store"
)

func (s *Server) handleDeploy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	report := newDeployReport()
	artifact, err := deploy.ParseZip(r.Body, envInt64("DOCS_DEPLOY_MAX_BYTES", 100*1024*1024))
	if err != nil {
		report.fail("parse_artifact", err)
		writeDeployError(w, http.StatusBadRequest, "invalid_artifact", err.Error(), report)
		return
	}
	report.ok("parse_artifact")

	// Deploy auth (GitLab CI / docsctl integration). Each document source owns
	// an independent token; the artifact module key selects the token to verify.
	// Token can be sent as X-Modex-Deploy-Token header or Authorization: Bearer <token>
	moduleKey := artifact.Metadata.ModuleKey
	provided := r.Header.Get("X-Modex-Deploy-Token")
	if provided == "" {
		provided = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	}
	m, moduleErr := s.app.Store().Module(moduleKey)
	if moduleErr != nil {
		report.fail("authenticate", moduleErr)
		writeDeployError(w, http.StatusNotFound, "module_not_found", "document source not found", report)
		return
	}
	if m.DeployToken == "" {
		report.fail("authenticate", errors.New("deploy token is not configured"))
		writeDeployError(w, http.StatusForbidden, "deploy_token_not_configured", "generate a deploy token for this document source first", report)
		return
	}
	if subtle.ConstantTimeCompare([]byte(provided), []byte(m.DeployToken)) != 1 {
		report.fail("authenticate", errors.New("deploy token mismatch"))
		writeDeployError(w, http.StatusForbidden, "invalid_deploy_token", "deploy token required or invalid for this module", report)
		return
	}
	report.ok("authenticate")

	uploadedSiteFiles := artifact.SiteFiles
	if s.minioClient != nil {
		if err := s.uploadSiteFilesToMinIO(r.Context(), artifact, moduleKey, artifact.Metadata.DocsVersion); err != nil {
			report.fail("upload_assets", err)
			writeDeployError(w, http.StatusBadGateway, "site_upload_failed", err.Error(), report)
			return
		}
		report.ok("upload_assets")
		// MinIO is the source of truth for static site assets in deployed
		// environments. Keeping the same bytes in the in-memory fallback can
		// push the backend into multi-GB RSS for image-heavy documentation.
		artifact.SiteFiles = nil
		artifact.SiteHTML = nil
	} else {
		report.skip("upload_assets", "object storage is not configured; keeping assets in memory")
	}
	if err := s.app.Search().DeleteModuleVersionEmbeddings(r.Context(), moduleKey, artifact.Metadata.DocsVersion); err != nil {
		report.fail("clear_embeddings", err)
		if s.minioClient != nil {
			s.cleanupUploadedSiteFiles(moduleKey, artifact.Metadata.DocsVersion, uploadedSiteFiles)
		}
		writeDeployError(w, http.StatusBadGateway, "embedding_cleanup_failed", err.Error(), report)
		return
	}
	report.ok("clear_embeddings")

	result, err := s.app.Store().IngestArtifact(toStoreArtifact(artifact))
	if err != nil {
		report.fail("ingest_metadata", err)
		if s.minioClient != nil {
			s.cleanupUploadedSiteFiles(moduleKey, artifact.Metadata.DocsVersion, uploadedSiteFiles)
		}
		writeDeployError(w, http.StatusBadRequest, "deploy_failed", err.Error(), report)
		return
	}
	report.ok("ingest_metadata")
	s.writeMutation(w, map[string]any{"status": "published", "result": result, "deploy": report}, http.StatusAccepted, nil)
}

func (s *Server) handleReleases(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireConsole(w, r)
	if !ok {
		return
	}
	releases := s.app.Store().Releases()
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

func (s *Server) handlePageView(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	var req struct {
		DocID     string `json:"doc_id"`
		SessionID string `json:"session_id"`
		ReadID    string `json:"read_id"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	req.DocID = strings.TrimSpace(req.DocID)
	if req.DocID == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "doc_id is required")
		return
	}
	user, _ := s.currentUser(r)
	pv := s.app.Store().RecordPageView(store.PageView{DocID: req.DocID, UserID: user.ID, SessionID: req.SessionID, ReadID: req.ReadID})
	s.writeMutation(w, map[string]any{"status": "recorded", "page_view": pv}, http.StatusAccepted, nil)
}

func (s *Server) handleReadProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	var req struct {
		DocID           string  `json:"doc_id"`
		SessionID       string  `json:"session_id"`
		ReadID          string  `json:"read_id"`
		DurationSeconds int     `json:"duration_seconds"`
		ScrollDepth     float64 `json:"scroll_depth"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	req.DocID = strings.TrimSpace(req.DocID)
	if req.DocID == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "doc_id is required")
		return
	}
	pv := s.app.Store().RecordReadProgress(req.DocID, req.SessionID, req.ReadID, req.DurationSeconds, req.ScrollDepth)
	s.writeMutation(w, map[string]any{"status": "recorded", "page_view": pv}, http.StatusAccepted, nil)
}

// handleDocAnalytics powers the doc-page "eye" popover: a daily read trend and
// a per-reader breakdown for one document. PostHog is preferred when configured;
// otherwise the built-in first-party page-view store returns the same shape.
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
			writeJSON(w, http.StatusOK, map[string]any{"source": "builtin", "stats": s.app.Store().PageReadStats(docID, days)})
			return
		}
		writeError(w, http.StatusBadGateway, "posthog_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"source": "posthog", "stats": stats})
}

func (s *Server) handleMeFavorites(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"favorites": s.app.Store().UserFavorites(user.ID)})
	case http.MethodPost:
		var req struct {
			ModuleKey string `json:"module_key"`
			Favorite  bool   `json:"favorite"`
		}
		if err := decodeBody(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		favorites, err := s.app.Store().SetUserFavorite(user.ID, req.ModuleKey, req.Favorite)
		s.writeMutation(w, map[string]any{"favorites": favorites}, http.StatusOK, err)
	case http.MethodDelete:
		var req struct {
			ModuleKey string `json:"module_key"`
		}
		if err := decodeBody(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		favorites, err := s.app.Store().SetUserFavorite(user.ID, req.ModuleKey, false)
		s.writeMutation(w, map[string]any{"favorites": favorites}, http.StatusOK, err)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET, POST or DELETE")
	}
}

func (s *Server) handleMeRecent(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"recent": s.app.Store().UserRecentDocs(user.ID, 30)})
	case http.MethodPost:
		var req store.UserRecentDoc
		if err := decodeBody(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		recent, err := s.app.Store().RecordUserRecentDoc(user.ID, req)
		s.writeMutation(w, map[string]any{"recent": recent}, http.StatusAccepted, err)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET or POST")
	}
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
	f := s.app.Store().AddDocFeedback(store.DocFeedback{
		DocID: req.DocID, Rating: req.Rating, Comment: req.Comment, UserID: user.ID, SessionID: req.SessionID,
	})
	s.writeMutation(w, map[string]any{"status": "recorded", "feedback": f}, http.StatusAccepted, nil)
}

func (s *Server) handleDocFeedbackLogs(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireConsole(w, r)
	if !ok {
		return
	}
	logs := s.app.Store().DocFeedbacks()
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
			if u, err := s.app.Store().UserByID(log.UserID); err == nil {
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
	logs := s.app.Store().SearchLogs()
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
			if u, err := s.app.Store().UserByID(log.UserID); err == nil {
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
	logs := s.app.Store().MCPLogs()
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
			if u, err := s.app.Store().UserByID(log.UserID); err == nil {
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
	user, ok := s.mcpLogUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "MCP log requires a session, OAuth access token, or personal MCP token")
		return
	}
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
	s.app.Store().AddMCPLog(store.MCPLog{ID: fmt.Sprintf("ml-%d", time.Now().UnixNano()), ToolName: req.ToolName, UserID: user.ID, Query: req.Query, InputJSON: req.InputJSON, ResultCount: req.ResultCount, CreatedAt: time.Now().UTC()})
	s.writeMutation(w, map[string]any{"status": "logged"}, http.StatusAccepted, nil)
}

func (s *Server) mcpLogUser(r *http.Request) (store.User, bool) {
	if user, ok := s.currentUser(r); ok {
		return user, true
	}
	if tok := bearerToken(r); tok != "" {
		if user, err := s.app.Store().UserByMCPToken(tok); err == nil {
			return user, true
		}
	}
	return store.User{}, false
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
		updated, err := s.app.Store().SetUserMCPToken(user.ID, tok)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "update_failed", err.Error())
			return
		}
		s.writeMutation(w, map[string]string{"mcp_token": updated.MCPToken}, http.StatusOK, nil)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET or POST")
	}
}

// currentUser returns the authenticated user from the session cookie. Login is
// a real, cookie-backed action in both mock and OIDC modes, so there is no
// silent impersonation; anonymous callers simply get ok == false.
