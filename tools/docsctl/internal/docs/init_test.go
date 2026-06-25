package docs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultMarkdownEntryUsesMarkdownRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "sparklers.md"), []byte("# Sparklers\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	entry := DefaultEntry(root, "markdown")
	if entry.Source != "." {
		t.Fatalf("source = %q, want .", entry.Source)
	}
}

func TestDefaultMarkdownEntryPrefersDocsDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "sparklers.md"), []byte("# Sparklers\n"), 0o644); err != nil {
		t.Fatalf("WriteFile sparklers: %v", err)
	}

	entry := DefaultEntry(root, "markdown")
	if entry.Source != "docs" {
		t.Fatalf("source = %q, want docs", entry.Source)
	}
}

func TestDefaultMarkdownEntryIgnoresAgentMetadataMarkdown(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("# Agent instructions\n"), 0o644); err != nil {
		t.Fatalf("WriteFile CLAUDE: %v", err)
	}

	entry := DefaultEntry(root, "markdown")
	if entry.Source == "." {
		t.Fatalf("source = %q, want a concrete docs entry instead of repository root", entry.Source)
	}
}

func TestDefaultVitePressSourceFollowsConfigLocation(t *testing.T) {
	// Config at repo root => built (and indexed) from root, even when an
	// unrelated docs/ subdir exists. Indexing docs/ here would miss the rest of
	// the site and emit routes missing their real path prefix (search 404s).
	rootCfg := t.TempDir()
	if err := os.MkdirAll(filepath.Join(rootCfg, ".vitepress"), 0o755); err != nil {
		t.Fatalf("MkdirAll .vitepress: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rootCfg, ".vitepress", "config.mjs"), []byte("export default {}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile config: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(rootCfg, "docs"), 0o755); err != nil {
		t.Fatalf("MkdirAll docs: %v", err)
	}
	if entry := DefaultEntry(rootCfg, "vitepress"); entry.Source != "." {
		t.Fatalf("root-config source = %q, want .", entry.Source)
	}

	// Config under docs/ => content root is docs/.
	docsCfg := t.TempDir()
	if err := os.MkdirAll(filepath.Join(docsCfg, "docs", ".vitepress"), 0o755); err != nil {
		t.Fatalf("MkdirAll docs/.vitepress: %v", err)
	}
	if err := os.WriteFile(filepath.Join(docsCfg, "docs", ".vitepress", "config.ts"), []byte("export default {}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile config: %v", err)
	}
	if entry := DefaultEntry(docsCfg, "vitepress"); entry.Source != "docs" {
		t.Fatalf("docs-config source = %q, want docs", entry.Source)
	}
}

func TestDetectProjectKindSupportsCommonStaticSiteGenerators(t *testing.T) {
	cases := []struct {
		name string
		file string
		want string
	}{
		{name: "docusaurus", file: "docusaurus.config.js", want: "docusaurus"},
		{name: "mkdocs", file: "mkdocs.yml", want: "mkdocs"},
		{name: "honkit", file: "book.json", want: "honkit"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, tc.file), []byte("{}\n"), 0o644); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}
			if got := DetectProjectKind(root); got != tc.want {
				t.Fatalf("DetectProjectKind = %q, want %q", got, tc.want)
			}
		})
	}
}
