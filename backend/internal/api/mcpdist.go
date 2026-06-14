package api

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// MCP tool distribution: serve the zero-dependency npx package (mcp/npx) straight
// from this deployment so users can install the MCP server without public npm
// (intranet-friendly). The per-user MCP token still gates real API access.

var (
	mcpTgzOnce sync.Once
	mcpTgzData []byte
	mcpTgzErr  error
)

// mcpDistDir resolves the directory holding the npx package. Configured via
// MCP_DIST_DIR (set in Docker); falls back to common dev locations.
func mcpDistDir() string {
	if d := strings.TrimSpace(os.Getenv("MCP_DIST_DIR")); d != "" {
		return d
	}
	for _, c := range []string{"../mcp/npx", "mcp/npx", "../../mcp/npx"} {
		if st, err := os.Stat(filepath.Join(c, "package.json")); err == nil && !st.IsDir() {
			return c
		}
	}
	return ""
}

// mcpDistFiles is the fixed allowlist of files we expose (no traversal).
func mcpDistFiles(dir string) []string {
	out := []string{}
	for _, n := range []string{"index.mjs", "package.json", "README.md"} {
		if st, err := os.Stat(filepath.Join(dir, n)); err == nil && !st.IsDir() {
			out = append(out, n)
		}
	}
	return out
}

// handleMcpDist serves the package listing, individual files, and an npm-style
// tarball. Public (the tool source is not secret).
func (s *Server) handleMcpDist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}
	dir := mcpDistDir()
	if dir == "" {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "MCP 工具产物未配置（设置 MCP_DIST_DIR）")
		return
	}
	rest := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/api/mcp/dist"), "/")

	if rest == "" {
		writeJSON(w, http.StatusOK, map[string]any{
			"package": "modex-docs-mcp",
			"files":   mcpDistFiles(dir),
			"tarball": "/api/mcp/dist/modex-docs-mcp.tgz",
		})
		return
	}

	if rest == "modex-docs-mcp.tgz" {
		data, err := mcpTarball(dir)
		if err != nil || len(data) == 0 {
			writeError(w, http.StatusInternalServerError, "tarball_failed", "无法生成 MCP 工具压缩包")
			return
		}
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Disposition", `attachment; filename="modex-docs-mcp.tgz"`)
		w.Header().Set("Cache-Control", "public, max-age=300")
		_, _ = w.Write(data)
		return
	}

	// Single file from the allowlist; reject any path tricks.
	if strings.ContainsAny(rest, "/\\") || strings.Contains(rest, "..") {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid file name")
		return
	}
	allowed := false
	for _, n := range mcpDistFiles(dir) {
		if n == rest {
			allowed = true
			break
		}
	}
	if !allowed {
		writeError(w, http.StatusNotFound, "not_found", "file not found")
		return
	}
	b, err := os.ReadFile(filepath.Join(dir, rest))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "file not found")
		return
	}
	w.Header().Set("Content-Type", contentTypeForName(rest, b))
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write(b)
}

// mcpTarball builds (once) an npm-installable gzip tarball: a gzipped tar whose
// entries are prefixed with "package/", which `npx -y <tarball-url>` accepts.
func mcpTarball(dir string) ([]byte, error) {
	mcpTgzOnce.Do(func() {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gz)
		for _, n := range mcpDistFiles(dir) {
			b, err := os.ReadFile(filepath.Join(dir, n))
			if err != nil {
				mcpTgzErr = err
				return
			}
			if err := tw.WriteHeader(&tar.Header{Name: "package/" + n, Mode: 0o644, Size: int64(len(b))}); err != nil {
				mcpTgzErr = err
				return
			}
			if _, err := tw.Write(b); err != nil {
				mcpTgzErr = err
				return
			}
		}
		if err := tw.Close(); err != nil {
			mcpTgzErr = err
			return
		}
		if err := gz.Close(); err != nil {
			mcpTgzErr = err
			return
		}
		mcpTgzData = buf.Bytes()
	})
	return mcpTgzData, mcpTgzErr
}
