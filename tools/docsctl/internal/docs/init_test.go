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
