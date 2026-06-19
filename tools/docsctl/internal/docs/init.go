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
	if exists(filepath.Join(root, "docusaurus.config.js")) ||
		exists(filepath.Join(root, "docusaurus.config.ts")) ||
		exists(filepath.Join(root, "docusaurus.config.mjs")) {
		return "docusaurus"
	}
	if exists(filepath.Join(root, "mkdocs.yml")) ||
		exists(filepath.Join(root, "mkdocs.yaml")) {
		return "mkdocs"
	}
	if exists(filepath.Join(root, "book.json")) {
		return "honkit"
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
	case "docusaurus":
		return Entry{Key: "guide", Title: "Docusaurus 文档", Type: "docusaurus", Source: firstExistingDir(root, "docs", "."), Build: detectBuildCommand(root, "build"), Output: firstOutput("build", "dist")}
	case "mkdocs":
		return Entry{Key: "guide", Title: "MkDocs 文档", Type: "mkdocs", Source: firstExistingDir(root, "docs", "."), Build: firstCommand("mkdocs build"), Output: firstOutput("site")}
	case "honkit", "gitbook":
		return Entry{Key: "guide", Title: "GitBook 文档", Type: kind, Source: firstExistingDir(root, "docs", "."), Build: detectBuildCommand(root, "build", "docs:build"), Output: firstOutput("_book", "book")}
	case "static":
		return Entry{Key: "legacy", Title: "静态文档", Type: "static", Source: firstExistingDir(root, "dist", "public")}
	default:
		return Entry{Key: "guide", Title: "Markdown 文档", Type: "markdown", Source: firstMarkdownSourceRoot(root)}
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

func firstMarkdownSourceRoot(root string) string {
	if hasMarkdownFiles(filepath.Join(root, "docs")) {
		return "docs"
	}
	if hasMarkdownFiles(root) {
		return "."
	}
	return firstExistingFile(root, "docs/README.md", "README.md", "docs/index.md", "index.md")
}

func hasMarkdownFiles(root string) bool {
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return false
	}
	found := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == ".vitepress" || name == ".vuepress" {
				return filepath.SkipDir
			}
			return nil
		}
		name := strings.ToLower(d.Name())
		if strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".mdx") {
			found = true
		}
		return nil
	})
	return found
}

func firstOutput(candidates ...string) string {
	return candidates[0]
}

func firstCommand(candidates ...string) string {
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
