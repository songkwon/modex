package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"modex/backend/internal/deploy"
	"modex/backend/internal/store"

	"github.com/minio/minio-go/v7"
)

func (s *Server) handleAdminAccepted(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "not_found", "admin route not found")
}

func decodeBody(r *http.Request, v any) error {
	return json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(v)
}

func (s *Server) writeMutation(w http.ResponseWriter, v any, successStatus int, err error) {
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
	for _, allowed := range s.app.Auth().Config().CORSAllowOrigins {
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

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func accessLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rr := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rr, r)
		if r.URL.Path == "/healthz" {
			return
		}
		log.Printf("http method=%s path=%s status=%d duration_ms=%d remote=%s", r.Method, r.URL.Path, rr.status, time.Since(start).Milliseconds(), r.RemoteAddr)
	})
}

type deployStep struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
	Note   string `json:"note,omitempty"`
}

type deployReport struct {
	Stages []deployStep `json:"stages"`
}

func newDeployReport() *deployReport {
	return &deployReport{Stages: []deployStep{}}
}

func (r *deployReport) ok(name string) {
	r.Stages = append(r.Stages, deployStep{Name: name, Status: "ok"})
}

func (r *deployReport) skip(name, note string) {
	r.Stages = append(r.Stages, deployStep{Name: name, Status: "skipped", Note: note})
}

func (r *deployReport) fail(name string, err error) {
	r.Stages = append(r.Stages, deployStep{Name: name, Status: "failed", Error: errorString(err)})
	log.Printf("deploy stage failed stage=%s error=%v", name, err)
}

func writeDeployError(w http.ResponseWriter, status int, code, message string, report *deployReport) {
	writeJSON(w, status, map[string]any{
		"error":  map[string]string{"code": code, "message": message},
		"deploy": report,
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

func (s *Server) cleanupUploadedSiteFiles(moduleKey, docsVersion string, files map[string][]byte) {
	if s.minioClient == nil || len(files) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for name := range files {
		key := fmt.Sprintf("modules/%s/%s/%s", moduleKey, docsVersion, name)
		if err := s.minioClient.RemoveObject(ctx, s.minioBucket, key, minio.RemoveObjectOptions{}); err != nil {
			log.Printf("cleanup uploaded site file failed key=%s error=%v", key, err)
		}
	}
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

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func ternary[T any](cond bool, yes, no T) T {
	if cond {
		return yes
	}
	return no
}

func envInt64(key string, fallback int64) int64 {
	v, err := strconv.ParseInt(os.Getenv(key), 10, 64)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
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
