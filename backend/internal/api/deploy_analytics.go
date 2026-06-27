package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"modex/backend/internal/deploy"
	"modex/backend/internal/store"
)

// acquireDeploySlot blocks until an ingest slot is free, the wait budget runs
// out, or the client disconnects. The returned release frees the slot and must
// be deferred by the caller; ok is false when the slot could not be acquired.
func (s *Server) acquireDeploySlot(ctx context.Context) (release func(), ok bool) {
	if s.deploy == nil {
		return func() {}, true
	}
	return s.deploy.acquire(ctx)
}

func (s *Server) handleDeploy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	// Bound concurrent ingests so simultaneous large artifacts can't exhaust the
	// instance. Wait briefly (CI tolerates a short queue), then shed load with 503.
	if release, ok := s.acquireDeploySlot(r.Context()); ok {
		defer release()
	} else {
		w.Header().Set("Retry-After", strconv.Itoa(envPositiveInt("DEPLOY_BUSY_RETRY_SECONDS", 30)))
		writeError(w, http.StatusServiceUnavailable, "deploy_busy", "too many concurrent deployments; retry shortly")
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
	// an independent token, and the token selects the target document source.
	// Token can be sent as X-Modex-Deploy-Token header or Authorization: Bearer <token>
	provided := r.Header.Get("X-Modex-Deploy-Token")
	if provided == "" {
		provided = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	}
	provided = strings.TrimSpace(provided)
	if provided == "" {
		report.fail("authenticate", errors.New("deploy token is required"))
		writeDeployError(w, http.StatusForbidden, "invalid_deploy_token", "deploy token required or invalid", report)
		return
	}
	m, moduleErr := s.app.Store().ModuleByDeployToken(provided)
	if moduleErr != nil {
		report.fail("authenticate", moduleErr)
		writeDeployError(w, http.StatusForbidden, "invalid_deploy_token", "deploy token required or invalid", report)
		return
	}
	artifact = canonicalizeDeployArtifact(artifact, m)
	moduleKey := m.ModuleKey
	report.ok("authenticate")

	uploadedSiteFiles := artifact.SiteFiles
	if s.minioClient != nil {
		if err := s.uploadSiteFilesToMinIO(r.Context(), artifact, moduleKey, artifact.Metadata.DocsVersion); err != nil {
			report.fail("upload_assets", err)
			writeDeployError(w, http.StatusBadGateway, "site_upload_failed", err.Error(), report)
			return
		}
		report.ok("upload_assets")
		// MinIO is the source of truth for static site assets when configured;
		// avoid duplicating the same bytes in PostgreSQL.
		artifact.SiteFiles = nil
		artifact.SiteHTML = nil
	} else {
		report.skip("upload_assets", "object storage is not configured; storing assets in PostgreSQL")
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

	storeArtifact := toStoreArtifact(artifact)
	storeArtifact.SourceIP = clientIP(r)
	storeArtifact.TriggerType = deployTriggerType(r)
	result, err := s.app.Store().IngestArtifact(storeArtifact)
	if err != nil {
		report.fail("ingest_metadata", err)
		if s.minioClient != nil {
			s.cleanupUploadedSiteFiles(moduleKey, artifact.Metadata.DocsVersion, uploadedSiteFiles)
		}
		writeDeployError(w, http.StatusBadRequest, "deploy_failed", err.Error(), report)
		return
	}
	report.ok("ingest_metadata")
	if count, err := s.app.Search().ReindexModuleVersion(r.Context(), moduleKey, artifact.Metadata.DocsVersion); err != nil {
		report.fail("rebuild_embeddings", err)
		if s.minioClient != nil {
			s.cleanupUploadedSiteFiles(moduleKey, artifact.Metadata.DocsVersion, uploadedSiteFiles)
		}
		writeDeployError(w, http.StatusBadGateway, "embedding_rebuild_failed", err.Error(), report)
		return
	} else if count > 0 {
		report.ok("rebuild_embeddings")
	} else {
		report.skip("rebuild_embeddings", "no indexable document chunks")
	}
	s.writeMutation(w, map[string]any{"status": "published", "result": result, "deploy": report}, http.StatusAccepted, nil)
}

func canonicalizeDeployArtifact(artifact deploy.Artifact, module store.Module) deploy.Artifact {
	artifactModuleKey := strings.TrimSpace(artifact.Metadata.ModuleKey)
	artifactDocsVersion := firstNonEmptyStr(artifact.Metadata.DocsVersion, "latest")
	moduleKey := module.ModuleKey
	moduleName := firstNonEmptyStr(module.Name, module.ModuleKey)
	docsVersion := firstNonEmptyStr(artifact.Metadata.DocsVersion, module.DefaultVersion, "latest")
	artifact.Metadata.ModuleKey = moduleKey
	artifact.Metadata.ModuleName = moduleName
	artifact.Metadata.DocsVersion = docsVersion
	if artifact.Metadata.Description == "" {
		artifact.Metadata.Description = module.Description
	}
	artifact = applyModuleMount(artifact, module)
	// docsctl builds without knowing the target module (the deploy token selects
	// it), so resource URLs in document content are baked with the placeholder
	// module/version. Rewrite them to the resolved module so embedded images and
	// attachments resolve under the real /api/docs/<module>/<version>/ path.
	rewriteBase := artifactModuleKey != "" && (artifactModuleKey != moduleKey || artifactDocsVersion != docsVersion)
	for i := range artifact.Documents {
		entryKey := firstNonEmptyStr(artifact.Documents[i].EntryKey, entryKeyFromDocID(artifact.Documents[i].DocID))
		artifact.Documents[i].ModuleKey = moduleKey
		artifact.Documents[i].ModuleName = moduleName
		artifact.Documents[i].DocsVersion = docsVersion
		if artifact.Documents[i].DocID != "" {
			artifact.Documents[i].DocID = rewriteDocIDModuleVersion(artifact.Documents[i].DocID, artifactModuleKey, moduleKey, artifactDocsVersion, docsVersion)
		} else if entryKey != "" {
			artifact.Documents[i].DocID = moduleKey + ":" + docsVersion + ":" + entryKey
		}
		if rewriteBase {
			artifact.Documents[i].Content = rewriteDeployAssetBaseString(artifact.Documents[i].Content, artifactModuleKey, moduleKey, artifactDocsVersion, docsVersion)
			artifact.Documents[i].ContentMD = rewriteDeployAssetBaseString(artifact.Documents[i].ContentMD, artifactModuleKey, moduleKey, artifactDocsVersion, docsVersion)
		}
	}
	artifact.SiteHTML = rewriteDeployAssetBases(artifact.SiteHTML, artifactModuleKey, moduleKey, artifactDocsVersion, docsVersion)
	for name, content := range artifact.SiteFiles {
		rewritten := rewriteDeployAssetBaseBytes(content, artifactModuleKey, moduleKey, artifactDocsVersion, docsVersion)
		if rewritten != nil {
			artifact.SiteFiles[name] = rewritten
		}
	}
	return artifact
}

func rewriteDocIDModuleVersion(docID, fromModule, toModule, fromVersion, toVersion string) string {
	parts := strings.SplitN(docID, ":", 3)
	if len(parts) != 3 {
		return docID
	}
	if fromModule != "" && !strings.EqualFold(parts[0], fromModule) {
		return docID
	}
	if fromVersion != "" && parts[1] != fromVersion {
		return docID
	}
	return toModule + ":" + toVersion + ":" + parts[2]
}

func applyModuleMount(artifact deploy.Artifact, module store.Module) deploy.Artifact {
	if !strings.EqualFold(strings.TrimSpace(module.Mount), "split") || !moduleUsesMarkdown(module, artifact) || len(artifact.Documents) <= 1 {
		return artifact
	}
	return splitMarkdownArtifactByTopLevel(artifact)
}

func moduleUsesMarkdown(module store.Module, artifact deploy.Artifact) bool {
	if strings.EqualFold(strings.TrimSpace(module.DocType), "markdown") {
		return true
	}
	for _, entry := range artifact.Manifest.Entries {
		if !strings.EqualFold(strings.TrimSpace(entry.Type), "markdown") {
			return false
		}
	}
	return len(artifact.Manifest.Entries) > 0
}

func splitMarkdownArtifactByTopLevel(artifact deploy.Artifact) deploy.Artifact {
	prefix := commonSourcePrefix(artifact.Documents)
	type group struct {
		key    string
		title  string
		source string
		docs   []deploy.DocumentRecord
	}
	groups := map[string]*group{}
	var order []string
	for _, doc := range artifact.Documents {
		rel := trimSourcePrefix(doc.SourceFile, prefix)
		key, title, source := splitGroupForSource(rel, doc)
		g := groups[key]
		if g == nil {
			g = &group{key: key, title: title, source: source}
			groups[key] = g
			order = append(order, key)
		}
		g.docs = append(g.docs, doc)
	}
	sort.SliceStable(order, func(i, j int) bool {
		if order[i] == "guide" {
			return true
		}
		if order[j] == "guide" {
			return false
		}
		return order[i] < order[j]
	})

	entries := make([]deploy.Entry, 0, len(order))
	nav := make([]deploy.NavItem, 0, len(order))
	documents := make([]deploy.DocumentRecord, 0, len(order))
	for _, key := range order {
		g := groups[key]
		entries = append(entries, deploy.Entry{Key: g.key, Title: g.title, Type: "markdown", Source: g.source})
		nav = append(nav, deploy.NavItem{Title: g.title, Path: "/" + g.key})
		documents = append(documents, mergeMarkdownGroup(artifact, *g))
	}
	artifact.Manifest.Entries = entries
	artifact.Nav = nav
	artifact.Documents = documents
	return artifact
}

func mergeMarkdownGroup(artifact deploy.Artifact, g struct {
	key    string
	title  string
	source string
	docs   []deploy.DocumentRecord
}) deploy.DocumentRecord {
	var content strings.Builder
	var contentMD strings.Builder
	for _, doc := range g.docs {
		title := firstNonEmptyStr(doc.Title, doc.SourceFile)
		content.WriteString("# " + title + "\n\n" + strings.TrimSpace(doc.Content) + "\n\n")
		md := firstNonEmptyStr(doc.ContentMD, doc.Content)
		contentMD.WriteString("# " + title + "\n\n" + strings.TrimSpace(md) + "\n\n")
	}
	text := strings.TrimSpace(content.String())
	desc := text
	if len(desc) > 140 {
		desc = desc[:140]
	}
	return deploy.DocumentRecord{
		DocID:          artifact.Metadata.ModuleKey + ":" + artifact.Metadata.DocsVersion + ":" + g.key,
		ModuleKey:      artifact.Metadata.ModuleKey,
		ModuleName:     artifact.Metadata.ModuleName,
		DocsVersion:    artifact.Metadata.DocsVersion,
		PackageVersion: artifact.Metadata.PackageVersion,
		EntryKey:       g.key,
		EntryType:      "markdown",
		Title:          g.title,
		Description:    desc,
		Content:        text,
		ContentMD:      strings.TrimSpace(contentMD.String()),
		Path:           "/" + g.key,
		SourceFile:     g.source,
		Status:         "active",
		Keywords:       artifact.Metadata.Keywords,
	}
}

func commonSourcePrefix(docs []deploy.DocumentRecord) string {
	counts := map[string]int{}
	for _, doc := range docs {
		parts := splitSourcePath(doc.SourceFile)
		if len(parts) > 1 {
			counts[parts[0]]++
		}
	}
	for first, count := range counts {
		if count == len(docs) {
			return first
		}
	}
	return ""
}

func trimSourcePrefix(source, prefix string) string {
	source = strings.Trim(strings.ReplaceAll(source, "\\", "/"), "/")
	if prefix != "" && (source == prefix || strings.HasPrefix(source, prefix+"/")) {
		return strings.TrimPrefix(strings.TrimPrefix(source, prefix), "/")
	}
	return source
}

func splitGroupForSource(rel string, doc deploy.DocumentRecord) (string, string, string) {
	parts := splitSourcePath(rel)
	if len(parts) == 0 {
		return firstNonEmptyStr(doc.EntryKey, "guide"), firstNonEmptyStr(doc.Title, "Guide"), doc.SourceFile
	}
	if len(parts) == 1 {
		name := strings.TrimSuffix(parts[0], sourceExt(parts[0]))
		if strings.EqualFold(name, "readme") || strings.EqualFold(name, "index") {
			return "guide", "Guide", doc.SourceFile
		}
		key := slugForMount(name)
		return key, firstNonEmptyStr(doc.Title, name), doc.SourceFile
	}
	key := slugForMount(parts[0])
	return key, titleForMount(parts[0]), strings.Trim(strings.TrimSuffix(doc.SourceFile, strings.Join(parts[1:], "/")), "/")
}

func splitSourcePath(source string) []string {
	source = strings.Trim(strings.ReplaceAll(source, "\\", "/"), "/")
	if source == "" {
		return nil
	}
	return strings.Split(source, "/")
}

func sourceExt(name string) string {
	lower := strings.ToLower(name)
	for _, ext := range []string{".mdx", ".md"} {
		if strings.HasSuffix(lower, ext) {
			return name[len(name)-len(ext):]
		}
	}
	return ""
}

func slugForMount(s string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "guide"
	}
	return out
}

func titleForMount(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "-", " "))
	if s == "" {
		return "Guide"
	}
	runes := []rune(s)
	runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
	return string(runes)
}

func rewriteDeployAssetBases(files map[string]string, fromModule, toModule, fromVersion, toVersion string) map[string]string {
	if len(files) == 0 || fromModule == "" || toModule == "" || (fromModule == toModule && fromVersion == toVersion) {
		return files
	}
	out := make(map[string]string, len(files))
	for name, content := range files {
		out[name] = rewriteDeployAssetBaseString(content, fromModule, toModule, fromVersion, toVersion)
	}
	return out
}

func rewriteDeployAssetBaseBytes(content []byte, fromModule, toModule, fromVersion, toVersion string) []byte {
	if len(content) == 0 || fromModule == "" || toModule == "" || (fromModule == toModule && fromVersion == toVersion) {
		return nil
	}
	rewritten := rewriteDeployAssetBaseString(string(content), fromModule, toModule, fromVersion, toVersion)
	if rewritten == string(content) {
		return nil
	}
	return []byte(rewritten)
}

func rewriteDeployAssetBaseString(content, fromModule, toModule, fromVersion, toVersion string) string {
	from := "/api/docs/" + fromModule + "/" + fromVersion + "/"
	to := "/api/docs/" + toModule + "/" + toVersion + "/"
	return strings.ReplaceAll(content, from, to)
}

func entryKeyFromDocID(docID string) string {
	parts := strings.Split(docID, ":")
	if len(parts) >= 3 {
		return strings.Join(parts[2:], ":")
	}
	return ""
}

func deployTriggerType(r *http.Request) string {
	for _, name := range []string{
		"X-Gitlab-Event",
		"X-GitHub-Event",
		"X-Circleci-Event-Type",
		"X-Jenkins",
		"X-Buildkite-Event",
	} {
		if strings.TrimSpace(r.Header.Get(name)) != "" {
			return "pipeline"
		}
	}
	if strings.TrimSpace(r.Header.Get("X-Modex-Deploy-Trigger")) == "pipeline" ||
		strings.TrimSpace(r.Header.Get("X-CI")) != "" {
		return "pipeline"
	}
	return "manual"
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
			if containsFold(rel.ModuleKey, kw) ||
				containsFold(rel.DocsVersion, kw) ||
				containsFold(rel.PackageVersion, kw) ||
				containsFold(rel.CommitSHA, kw) ||
				containsFold(rel.Branch, kw) ||
				containsFold(rel.SourceIP, kw) ||
				containsFold(rel.TriggerType, kw) ||
				containsFold(rel.ReleaseID, kw) {
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
		// Team admins see searches scoped to, or clicked within, owned categories.
		if !all && !categoriesIntersect(s.searchLogCategoryIDs(log), set) {
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

func (s *Server) searchLogCategoryIDs(log store.SearchLog) []string {
	if cats := s.docCategoryIDs(log.ClickedDocID); len(cats) > 0 {
		return cats
	}
	if strings.TrimSpace(log.FiltersJSON) == "" {
		return nil
	}
	var filters struct {
		CategoryIDs []string `json:"category_ids"`
		Modules     []string `json:"modules"`
	}
	if err := json.Unmarshal([]byte(log.FiltersJSON), &filters); err != nil {
		return nil
	}
	out := append([]string{}, filters.CategoryIDs...)
	for _, moduleKey := range filters.Modules {
		out = append(out, s.moduleCategories(moduleKey)...)
	}
	return out
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

func (s *Server) handleMCPTokenInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}
	token := bearerToken(r)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "invalid_token", "bearer token is required")
		return
	}
	if user, _, grant, err := s.app.Store().UserByOAuthAccessToken(token); err == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"user_id":    user.ID,
			"scopes":     grant.Scopes,
			"expires_at": grant.AccessExpiresAt,
		})
		return
	}
	if user, err := s.app.Store().UserByMCPToken(token); err == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"user_id":    user.ID,
			"scopes":     []string{"modex:mcp:read", "modex:docs:read"},
			"expires_at": time.Now().UTC().Add(24 * time.Hour),
		})
		return
	}
	writeError(w, http.StatusUnauthorized, "invalid_token", "bearer token is invalid or expired")
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
// a real, cookie-backed OIDC action, so there is no silent impersonation;
// anonymous callers simply get ok == false.
