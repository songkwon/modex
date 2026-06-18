package api

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
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
	mcpTgzOnce   sync.Once
	mcpTgzData   []byte
	mcpTgzErr    error
	skillTgzOnce sync.Once
	skillTgzData []byte
	skillTgzErr  error
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

// skillDistDir resolves the optional Modex Skill directory served for users who
// install client-side guidance with `npx skills add`.
func skillDistDir() string {
	if d := strings.TrimSpace(os.Getenv("MODEX_SKILL_DIST_DIR")); d != "" {
		return d
	}
	for _, c := range []string{"../mcp/skill", "mcp/skill", "../../mcp/skill", "../../../mcp/skill"} {
		if st, err := os.Stat(filepath.Join(c, "SKILL.md")); err == nil && !st.IsDir() {
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
	rest := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/api/mcp/dist"), "/")

	if rest == "modex-skill.tgz" {
		data, err := skillTarball()
		if err != nil || len(data) == 0 {
			writeError(w, http.StatusInternalServerError, "tarball_failed", "无法生成 Modex Skill 压缩包")
			return
		}
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Disposition", `attachment; filename="modex-skill.tgz"`)
		w.Header().Set("Cache-Control", "public, max-age=300")
		_, _ = w.Write(data)
		return
	}

	dir := mcpDistDir()
	if dir == "" {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "MCP 工具产物未配置（设置 MCP_DIST_DIR）")
		return
	}

	if rest == "" {
		writeJSON(w, http.StatusOK, map[string]any{
			"package":       "modex-mcp",
			"files":         mcpDistFiles(dir),
			"tarball":       "/api/mcp/dist/modex-mcp.tgz",
			"skill_tarball": "/api/mcp/dist/modex-skill.tgz",
		})
		return
	}

	if rest == "modex-mcp.tgz" || rest == "modex-docs-mcp.tgz" {
		data, err := mcpTarball(dir)
		if err != nil || len(data) == 0 {
			writeError(w, http.StatusInternalServerError, "tarball_failed", "无法生成 MCP 工具压缩包")
			return
		}
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Disposition", `attachment; filename="modex-mcp.tgz"`)
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

func (s *Server) handleSkillDiscovery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}
	data, err := skillTarball()
	if err != nil || len(data) == 0 {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "Modex Skill 产物未配置（设置 MODEX_SKILL_DIST_DIR）")
		return
	}
	sum := sha256.Sum256(data)
	writeJSON(w, http.StatusOK, map[string]any{
		"$schema": "https://schemas.agentskills.io/discovery/0.2.0/schema.json",
		"skills": []map[string]any{
			{
				"name":        "modex",
				"description": "Use Modex MCP to search and read a team's live Modex documentation portal before answering module, API, release, architecture, or platform-specific questions.",
				"type":        "archive",
				"url":         "/api/mcp/dist/modex-skill.tgz",
				"digest":      fmt.Sprintf("sha256:%x", sum[:]),
			},
		},
	})
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

func skillTarball() ([]byte, error) {
	skillTgzOnce.Do(func() {
		dir := skillDistDir()
		if dir == "" {
			return
		}
		b, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
		if err != nil {
			skillTgzErr = err
			return
		}
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gz)
		if err := tw.WriteHeader(&tar.Header{Name: "SKILL.md", Mode: 0o644, Size: int64(len(b))}); err != nil {
			skillTgzErr = err
			return
		}
		if _, err := tw.Write(b); err != nil {
			skillTgzErr = err
			return
		}
		if err := tw.Close(); err != nil {
			skillTgzErr = err
			return
		}
		if err := gz.Close(); err != nil {
			skillTgzErr = err
			return
		}
		skillTgzData = buf.Bytes()
	})
	return skillTgzData, skillTgzErr
}
