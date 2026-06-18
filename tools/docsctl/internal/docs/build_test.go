package docs

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildMarkdownDirectoryCreatesOneDocumentPerFile(t *testing.T) {
	root := t.TempDir()
	docsDir := filepath.Join(root, "docs")
	if err := os.MkdirAll(filepath.Join(docsDir, "runtime"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(docsDir, "images"), 0o755); err != nil {
		t.Fatalf("MkdirAll images: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(docsDir, "media", "screens"), 0o755); err != nil {
		t.Fatalf("MkdirAll media: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(docsDir, "downloads"), 0o755); err != nil {
		t.Fatalf("MkdirAll downloads: %v", err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "images", "screen shot.png"), []byte("png"), 0o644); err != nil {
		t.Fatalf("WriteFile image: %v", err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "media", "screens", "sentry.png"), []byte("sentry"), 0o644); err != nil {
		t.Fatalf("WriteFile nested image: %v", err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "downloads", "runbook.pdf"), []byte("pdf"), 0o644); err != nil {
		t.Fatalf("WriteFile attachment: %v", err)
	}
	readme := `# Overview

Root doc.

![shot](./images/screen shot.png)
![nested](./media/screens/sentry.png)
[runbook](downloads/runbook.pdf)
<img src="./media/screens/sentry.png" />
<a href="downloads/runbook.pdf">Download</a>
`
	if err := os.WriteFile(filepath.Join(docsDir, "README.md"), []byte(readme), 0o644); err != nil {
		t.Fatalf("WriteFile README: %v", err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "runtime", "threadpool.md"), []byte("# Threadpool\n\nPool doc."), 0o644); err != nil {
		t.Fatalf("WriteFile threadpool: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs.yaml"), []byte(`metadata:
  module_key: demo
  docs_version: v1
entries:
  - key: guide
    title: Guide
    type: markdown
    source: docs
`), 0o644); err != nil {
		t.Fatalf("WriteFile docs.yaml: %v", err)
	}

	out := filepath.Join(t.TempDir(), "build")
	if err := Build(root, out); err != nil {
		t.Fatalf("Build: %v", err)
	}

	var manifest Manifest
	readJSONFile(t, filepath.Join(out, "manifest.json"), &manifest)
	if got, want := len(manifest.Entries), 2; got != want {
		t.Fatalf("manifest entries = %d, want %d", got, want)
	}
	if manifest.Entries[0].Key != "guide" || manifest.Entries[0].Title != "README" || manifest.Entries[1].Key != "guide-runtime-threadpool" || manifest.Entries[1].Title != "threadpool" {
		t.Fatalf("entry keys = %#v", manifest.Entries)
	}

	records := readDocumentsJSONL(t, filepath.Join(out, "documents.jsonl"))
	if got, want := len(records), 2; got != want {
		t.Fatalf("documents = %d, want %d", got, want)
	}
	if records[0].Title != "README" || records[1].Title != "threadpool" {
		t.Fatalf("record titles = %#v", records)
	}
	var nav []NavItem
	readJSONFile(t, filepath.Join(out, "nav.json"), &nav)
	if len(nav) != 2 || nav[0].Title != "README" || nav[1].Title != "runtime" || len(nav[1].Children) != 1 || nav[1].Children[0].Title != "threadpool" {
		t.Fatalf("nav = %#v", nav)
	}
	if _, err := os.Stat(filepath.Join(out, "site", "guide", "images", "screen shot.png")); err != nil {
		t.Fatalf("expected bundled image: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "site", "guide", "media", "screens", "sentry.png")); err != nil {
		t.Fatalf("expected bundled nested image: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "site", "guide", "downloads", "runbook.pdf")); err != nil {
		t.Fatalf("expected bundled attachment: %v", err)
	}
	if !contains(records[0].ContentMD, "/api/docs/UnknownModule/latest/guide/site/media/screens/sentry.png") || !contains(records[0].ContentMD, "/api/docs/UnknownModule/latest/guide/site/downloads/runbook.pdf") {
		t.Fatalf("record content was not rewritten: %s", records[0].ContentMD)
	}
}

func TestBuildMarkdownDirectoryBundlesParentRelativeResources(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs", "runtime"), 0o755); err != nil {
		t.Fatalf("MkdirAll runtime: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs", "assets"), 0o755); err != nil {
		t.Fatalf("MkdirAll assets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "assets", "diagram.svg"), []byte("svg"), 0o644); err != nil {
		t.Fatalf("WriteFile asset: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "runtime", "threadpool.md"), []byte("# Threadpool\n\n![diagram](../assets/diagram.svg)"), 0o644); err != nil {
		t.Fatalf("WriteFile threadpool: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs.yaml"), []byte(`metadata:
  module_key: demo
  docs_version: v1
entries:
  - key: guide
    title: Guide
    type: markdown
    source: docs
`), 0o644); err != nil {
		t.Fatalf("WriteFile docs.yaml: %v", err)
	}

	out := filepath.Join(t.TempDir(), "build")
	if err := Build(root, out); err != nil {
		t.Fatalf("Build: %v", err)
	}

	records := readDocumentsJSONL(t, filepath.Join(out, "documents.jsonl"))
	if len(records) != 1 {
		t.Fatalf("records = %d, want 1", len(records))
	}
	if _, err := os.Stat(filepath.Join(out, "site", "guide-runtime-threadpool", "assets", "diagram.svg")); err != nil {
		t.Fatalf("expected parent-relative bundled asset: %v", err)
	}
	if !contains(records[0].ContentMD, "/api/docs/UnknownModule/latest/guide-runtime-threadpool/site/assets/diagram.svg") {
		t.Fatalf("record content was not rewritten: %s", records[0].ContentMD)
	}
}

func readJSONFile(t *testing.T, path string, out any) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile %s: %v", path, err)
	}
	if err := json.Unmarshal(b, out); err != nil {
		t.Fatalf("Unmarshal %s: %v", path, err)
	}
}

func readDocumentsJSONL(t *testing.T, path string) []DocumentRecord {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open %s: %v", path, err)
	}
	defer f.Close()
	var out []DocumentRecord
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var rec DocumentRecord
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			t.Fatalf("Unmarshal jsonl: %v", err)
		}
		out = append(out, rec)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("Scan jsonl: %v", err)
	}
	return out
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
