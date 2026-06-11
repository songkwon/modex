package docs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Init(root string, force bool) error {
	path := filepath.Join(root, "docs.yaml")
	if _, err := os.Stat(path); err == nil && !force {
		return fmt.Errorf("docs.yaml already exists, set DOCS_INIT_FORCE=true to overwrite")
	}
	kind := DetectProjectKind(root)
	cfg := Config{Entries: []Entry{DefaultEntry(root, kind)}}
	return os.WriteFile(path, []byte(RenderDocsYAML(cfg)), 0o644)
}

func DetectProjectKind(root string) string {
	if existsAny(
		filepath.Join(root, "docs", ".vitepress", "config.mjs"),
		filepath.Join(root, "docs", ".vitepress", "config.mts"),
		filepath.Join(root, "docs", ".vitepress", "config.ts"),
		filepath.Join(root, "docs", ".vitepress", "config.js"),
		filepath.Join(root, ".vitepress", "config.mjs"),
		filepath.Join(root, ".vitepress", "config.mts"),
		filepath.Join(root, ".vitepress", "config.ts"),
		filepath.Join(root, ".vitepress", "config.js"),
	) {
		return "vitepress"
	}
	if exists(filepath.Join(root, "docs", ".vuepress", "config.js")) ||
		exists(filepath.Join(root, "docs", ".vuepress", "config.ts")) ||
		exists(filepath.Join(root, ".vuepress", "config.js")) ||
		exists(filepath.Join(root, ".vuepress", "config.ts")) {
		return "vuepress"
	}
	if exists(filepath.Join(root, "source.config.ts")) ||
		exists(filepath.Join(root, "content", "docs")) {
		return "fumadocs"
	}
	if exists(filepath.Join(root, "dist", "index.html")) ||
		exists(filepath.Join(root, "public", "index.html")) {
		return "static"
	}
	return "markdown"
}

func DefaultEntry(root, kind string) Entry {
	switch kind {
	case "vitepress":
		return Entry{Key: "guide", Title: "VitePress 文档", Type: "vitepress", Source: firstExistingDir(root, "docs", "."), Build: detectBuildCommand(root, "docs:build", "build"), Output: firstOutput("docs/.vitepress/dist", ".vitepress/dist", "dist")}
	case "vuepress":
		return Entry{Key: "guide", Title: "VuePress 文档", Type: "vuepress", Source: firstExistingDir(root, "docs", "."), Build: detectBuildCommand(root, "docs:build", "build"), Output: firstOutput("docs/.vuepress/dist", ".vuepress/dist", "dist")}
	case "fumadocs":
		return Entry{Key: "guide", Title: "Fumadocs 文档", Type: "fumadocs", Source: firstExistingDir(root, "content/docs", "docs"), Build: detectBuildCommand(root, "build", "docs:build"), Output: firstOutput("out", ".next/server/app", "dist")}
	case "static":
		return Entry{Key: "legacy", Title: "静态文档", Type: "static", Source: firstExistingDir(root, "dist", "public")}
	default:
		return Entry{Key: "guide", Title: "Markdown 文档", Type: "markdown", Source: firstExistingFile(root, "docs/README.md", "README.md")}
	}
}

func RenderDocsYAML(cfg Config) string {
	var b strings.Builder
	b.WriteString("entries:\n")
	for _, e := range cfg.Entries {
		b.WriteString("  - key: " + e.Key + "\n")
		b.WriteString("    title: " + e.Title + "\n")
		b.WriteString("    type: " + e.Type + "\n")
		b.WriteString("    source: " + e.Source + "\n")
		if e.Build != "" {
			b.WriteString("    build: " + e.Build + "\n")
		}
		if e.Output != "" {
			b.WriteString("    output: " + e.Output + "\n")
		}
	}
	return b.String()
}

func detectBuildCommand(root string, candidates ...string) string {
	if exists(filepath.Join(root, "package.json")) {
		for _, c := range candidates {
			return "npm run " + c
		}
	}
	return ""
}

func firstExistingDir(root string, candidates ...string) string {
	for _, c := range candidates {
		if info, err := os.Stat(filepath.Join(root, c)); err == nil && info.IsDir() {
			return c
		}
	}
	return candidates[len(candidates)-1]
}

func firstExistingFile(root string, candidates ...string) string {
	for _, c := range candidates {
		if info, err := os.Stat(filepath.Join(root, c)); err == nil && !info.IsDir() {
			return c
		}
	}
	return candidates[len(candidates)-1]
}

func firstOutput(candidates ...string) string {
	return candidates[0]
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func existsAny(paths ...string) bool {
	for _, p := range paths {
		if exists(p) {
			return true
		}
	}
	return false
}
