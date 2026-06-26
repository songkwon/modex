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
	if err := os.WriteFile(filepath.Join(docsDir, "CLAUDE.md"), []byte("# Agent instructions\n\nDo not publish."), 0o644); err != nil {
		t.Fatalf("WriteFile CLAUDE: %v", err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "runtime", "AGENTS.md"), []byte("# Agent instructions\n\nDo not publish."), 0o644); err != nil {
		t.Fatalf("WriteFile AGENTS: %v", err)
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
	for _, r := range records {
		if strings.Contains(r.SourceFile, "CLAUDE.md") || strings.Contains(r.SourceFile, "AGENTS.md") || strings.Contains(r.Content, "Do not publish") {
			t.Fatalf("agent metadata markdown was indexed: %#v", r)
		}
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

func TestBuildMarkdownDirectoryHonorsModexIgnore(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs", "drafts"), 0o755); err != nil {
		t.Fatalf("MkdirAll drafts: %v", err)
	}
	files := map[string]string{
		"docs/README.md":           "# Overview\n",
		"docs/drafts/hidden.md":    "# Hidden\n",
		"docs/keep.draft.md":       "# Keep\n",
		"docs/notes.local.md":      "# Local\n",
		"docs/runtime/visible.md":  "# Visible\n",
		"docs/runtime/hidden.mdx":  "# Hidden MDX\n",
		"docs/runtime/private.tmp": "private",
	}
	for name, content := range files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0o755); err != nil {
			t.Fatalf("MkdirAll %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".modexignore"), []byte("drafts/\n*.local.md\n**/*.draft.md\ndocs/runtime/*.mdx\n!docs/keep.draft.md\n"), 0o644); err != nil {
		t.Fatalf("WriteFile .modexignore: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs.yaml"), []byte(`entries:
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
	if got, want := len(records), 3; got != want {
		t.Fatalf("records = %d, want %d: %#v", got, want, records)
	}
	var sources []string
	for _, r := range records {
		sources = append(sources, filepath.ToSlash(r.SourceFile))
	}
	joined := strings.Join(sources, "\n")
	for _, want := range []string{"docs/README.md", "docs/keep.draft.md", "docs/runtime/visible.md"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("sources missing %s: %s", want, joined)
		}
	}
	for _, hidden := range []string{"hidden.md", "notes.local.md", "hidden.mdx"} {
		if strings.Contains(joined, hidden) {
			t.Fatalf("ignored source %s was indexed: %s", hidden, joined)
		}
	}
}

func TestBuildSplitMountsProduceUniqueDocIDs(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"a", "b"} {
		if err := os.MkdirAll(filepath.Join(root, "docs", dir), 0o755); err != nil {
			t.Fatalf("MkdirAll %s: %v", dir, err)
		}
		// Each mount has a top-level README and an identically-named nested file,
		// which previously collided into the same doc_id across mounts.
		if err := os.WriteFile(filepath.Join(root, "docs", dir, "README.md"), []byte("# Root\n\nroot"), 0o644); err != nil {
			t.Fatalf("WriteFile README %s: %v", dir, err)
		}
		if err := os.WriteFile(filepath.Join(root, "docs", dir, "intro.md"), []byte("# Intro\n\nintro"), 0o644); err != nil {
			t.Fatalf("WriteFile intro %s: %v", dir, err)
		}
	}
	config := `metadata:
  module_key: demo
  docs_version: v1
entries:
  - key: guide
    title: Guide A
    type: markdown
    source: docs/a
  - key: guide
    title: Guide B
    type: markdown
    source: docs/b
`
	if err := os.WriteFile(filepath.Join(root, "docs.yaml"), []byte(config), 0o644); err != nil {
		t.Fatalf("WriteFile docs.yaml: %v", err)
	}

	out := filepath.Join(t.TempDir(), "build")
	if err := Build(root, out); err != nil {
		t.Fatalf("Build: %v", err)
	}

	records := readDocumentsJSONL(t, filepath.Join(out, "documents.jsonl"))
	if len(records) != 4 {
		t.Fatalf("documents = %d, want 4", len(records))
	}
	seen := map[string]bool{}
	for _, r := range records {
		if r.DocID == "" {
			t.Fatalf("empty doc id: %#v", r)
		}
		if seen[r.DocID] {
			t.Fatalf("duplicate doc id %q across split mounts", r.DocID)
		}
		seen[r.DocID] = true
	}
}

func TestBuildMarkdownRewritesInternalDocLinks(t *testing.T) {
	t.Setenv("DOCS_MODULE", "standards")
	t.Setenv("DOCS_VERSION", "latest")
	root := t.TempDir()
	mkdir := func(p string) {
		if err := os.MkdirAll(filepath.Join(root, p), 0o755); err != nil {
			t.Fatalf("MkdirAll %s: %v", p, err)
		}
	}
	write := func(p, body string) {
		if err := os.WriteFile(filepath.Join(root, p), []byte(body), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", p, err)
		}
	}
	mkdir("coding/general-programming")
	// Root README links to a nested doc (exists) and a not-yet-written doc.
	write("README.md", "# Standards\n\n- [mem](./coding/general-programming/memory-safety.md) - safety\n- [todo](./coding/todo.md) - later\n")
	// Nested README uses a sibling-relative link and an anchor.
	write("coding/README.md", "# Coding\n\n- [mem](./general-programming/memory-safety.md#p0) - safety\n")
	write("coding/general-programming/memory-safety.md", "# Memory Safety\n\nbody")

	out := filepath.Join(t.TempDir(), "build")
	if err := Build(root, out); err != nil {
		t.Fatalf("Build: %v", err)
	}
	records := readDocumentsJSONL(t, filepath.Join(out, "documents.jsonl"))
	byKey := map[string]DocumentRecord{}
	for _, r := range records {
		byKey[r.EntryKey] = r
	}

	root0, ok := byKey["guide"]
	if !ok {
		t.Fatalf("missing root README record; keys: %v", byKey)
	}
	wantRoute := "/docs/standards/latest/guide-coding-general-programming-memory-safety"
	if !contains(root0.ContentMD, "("+wantRoute+")") {
		t.Fatalf("root README not rewritten to %q:\n%s", wantRoute, root0.ContentMD)
	}
	// A link to a doc that does not exist must be left untouched (no false route).
	if !contains(root0.ContentMD, "./coding/todo.md") {
		t.Fatalf("link to missing doc should be left as-is:\n%s", root0.ContentMD)
	}

	coding, ok := byKey["guide-coding"]
	if !ok {
		t.Fatalf("missing coding README record; keys: %v", byKey)
	}
	if !contains(coding.ContentMD, wantRoute+"#p0)") {
		t.Fatalf("sibling-relative link with anchor not rewritten:\n%s", coding.ContentMD)
	}
}

func TestBuildCommandEntryDetectsCustomNestedOutput(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("MkdirAll docs: %v", err)
	}
	config := `entries:
  - key: guide
    title: Guide
    type: vitepress
    source: docs
    build: mkdir -p docs/dist/rd/standards && printf '<h1>Standards</h1>' > docs/dist/rd/standards/index.html
    output: docs/.vitepress/dist
`
	if err := os.WriteFile(filepath.Join(root, "docs.yaml"), []byte(config), 0o644); err != nil {
		t.Fatalf("WriteFile docs.yaml: %v", err)
	}

	out := filepath.Join(t.TempDir(), "build")
	if err := Build(root, out); err != nil {
		t.Fatalf("Build: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "site", "guide", "index.html")); err != nil {
		t.Fatalf("expected detected custom output to be copied: %v", err)
	}
}

func TestBuildCommandFailureIncludesActionableContext(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	config := `entries:
  - key: guide
    title: Guide
    type: mkdocs
    source: docs
    build: printf 'missing plugin: mkdocs-material\n'; exit 7
    output: site
`
	if err := os.WriteFile(filepath.Join(root, "docs.yaml"), []byte(config), 0o644); err != nil {
		t.Fatalf("WriteFile docs.yaml: %v", err)
	}

	err := Build(root, filepath.Join(t.TempDir(), "build"))
	if err == nil {
		t.Fatal("Build succeeded, want failure")
	}
	message := err.Error()
	for _, want := range []string{
		"documentation build failed",
		"entry: guide (mkdocs)",
		"printf 'missing plugin: mkdocs-material",
		"working directory: " + root,
		"exit code: 7",
		"missing plugin: mkdocs-material",
		"DOCS_BASE=/api/docs/UnknownModule/latest/guide/site/",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("error missing %q:\n%s", want, message)
		}
	}
}

func TestRewriteStaticSiteBaseOnlyRewritesExistingAssets(t *testing.T) {
	site := t.TempDir()
	if err := os.MkdirAll(filepath.Join(site, "assets"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	for name, content := range map[string]string{
		"assets/app.js":   "console.log('ok')",
		"assets/app.css":  "body{background:url('/assets/bg.png')}",
		"assets/bg.png":   "png",
		"assets/icon.png": "icon",
		"assets/one.png":  "one",
		"assets/two.png":  "two",
	} {
		if err := os.WriteFile(filepath.Join(site, filepath.FromSlash(name)), []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
	}
	html := `<html><head><link href="/assets/app.css"><script src="/assets/app.js"></script><script src="/missing.js"></script><script src="//cdn.example.com/lib.js"></script></head><body><img srcset="/assets/one.png 1x, /assets/two.png 2x"><a href="/guide/start">Guide</a></body></html>`
	if err := os.WriteFile(filepath.Join(site, "index.html"), []byte(html), 0o644); err != nil {
		t.Fatalf("WriteFile index: %v", err)
	}
	manifest := `{"icons":[{"src":"/assets/icon.png"}],"start_url":"/"}`
	if err := os.WriteFile(filepath.Join(site, "manifest.webmanifest"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile manifest: %v", err)
	}

	base := "/api/docs/demo/v1/guide/site/"
	count, err := rewriteStaticSiteBase(site, base)
	if err != nil {
		t.Fatalf("rewriteStaticSiteBase: %v", err)
	}
	if count != 7 {
		t.Fatalf("rewritten count = %d, want 7", count)
	}
	indexBytes, _ := os.ReadFile(filepath.Join(site, "index.html"))
	index := string(indexBytes)
	for _, want := range []string{base + "assets/app.css", base + "assets/app.js", base + "assets/one.png 1x", base + "assets/two.png 2x"} {
		if !strings.Contains(index, want) {
			t.Fatalf("index missing %q: %s", want, index)
		}
	}
	for _, untouched := range []string{`src="/missing.js"`, `src="//cdn.example.com/lib.js"`, `href="/guide/start"`} {
		if !strings.Contains(index, untouched) {
			t.Fatalf("index unexpectedly rewrote %q: %s", untouched, index)
		}
	}
	cssBytes, _ := os.ReadFile(filepath.Join(site, "assets", "app.css"))
	if !strings.Contains(string(cssBytes), base+"assets/bg.png") {
		t.Fatalf("css root asset was not rewritten: %s", cssBytes)
	}
	manifestBytes, _ := os.ReadFile(filepath.Join(site, "manifest.webmanifest"))
	if !strings.Contains(string(manifestBytes), base+"assets/icon.png") || !strings.Contains(string(manifestBytes), base) {
		t.Fatalf("manifest root URLs were not rewritten: %s", manifestBytes)
	}
	secondCount, err := rewriteStaticSiteBase(site, base)
	if err != nil {
		t.Fatalf("second rewriteStaticSiteBase: %v", err)
	}
	if secondCount != 0 {
		t.Fatalf("second rewrite changed %d URLs, want idempotent result", secondCount)
	}
	if _, ok := rewriteRootRef("/%2e%2e/assets/app.js", site, base); ok {
		t.Fatal("encoded parent traversal was rewritten")
	}
}

func TestRewriteStaticSiteBaseToRelative(t *testing.T) {
	site := t.TempDir()
	if err := os.MkdirAll(filepath.Join(site, "assets"), 0o755); err != nil {
		t.Fatalf("MkdirAll assets: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(site, "posts", "guide"), 0o755); err != nil {
		t.Fatalf("MkdirAll posts: %v", err)
	}
	for name, content := range map[string]string{
		"assets/app.css": "body{background:url('/api/docs/demo/v1/guide/site/assets/bg.png')}",
		"assets/app.js":  "console.log('ok')",
		"assets/bg.png":  "png",
	} {
		if err := os.WriteFile(filepath.Join(site, filepath.FromSlash(name)), []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
	}
	rootHTML := `<link href="/api/docs/demo/v1/guide/site/assets/app.css"><script src="/api/docs/demo/v1/guide/site/assets/app.js"></script>`
	if err := os.WriteFile(filepath.Join(site, "index.html"), []byte(rootHTML), 0o644); err != nil {
		t.Fatalf("WriteFile index: %v", err)
	}
	nestedHTML := `<link href="/api/docs/demo/v1/guide/site/assets/app.css">`
	if err := os.WriteFile(filepath.Join(site, "posts", "guide", "intro.html"), []byte(nestedHTML), 0o644); err != nil {
		t.Fatalf("WriteFile nested: %v", err)
	}

	count, err := rewriteStaticSiteBaseToRelative(site, "/api/docs/demo/v1/guide/site/")
	if err != nil {
		t.Fatalf("rewriteStaticSiteBaseToRelative: %v", err)
	}
	if count != 4 {
		t.Fatalf("rewritten count = %d, want 4", count)
	}
	rootBytes, _ := os.ReadFile(filepath.Join(site, "index.html"))
	if root := string(rootBytes); !strings.Contains(root, `href="./assets/app.css"`) || !strings.Contains(root, `src="./assets/app.js"`) {
		t.Fatalf("root html not relativized: %s", root)
	}
	nestedBytes, _ := os.ReadFile(filepath.Join(site, "posts", "guide", "intro.html"))
	if nested := string(nestedBytes); !strings.Contains(nested, `href="../../assets/app.css"`) {
		t.Fatalf("nested html not relativized: %s", nested)
	}
	cssBytes, _ := os.ReadFile(filepath.Join(site, "assets", "app.css"))
	if css := string(cssBytes); !strings.Contains(css, `url('./bg.png')`) {
		t.Fatalf("css not relativized: %s", css)
	}
}

func TestRewriteStaticSiteBaseStripsLegacyBasePrefix(t *testing.T) {
	site := t.TempDir()
	if err := os.MkdirAll(filepath.Join(site, "assets"), 0o755); err != nil {
		t.Fatalf("MkdirAll assets: %v", err)
	}
	for name, content := range map[string]string{
		"assets/style.css": "body{background:url('/internal-tools/assets/bg.png')}",
		"assets/app.js":    "console.log('ok')",
		"assets/bg.png":    "png",
	} {
		if err := os.WriteFile(filepath.Join(site, filepath.FromSlash(name)), []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
	}
	html := `<link href="/internal-tools/assets/style.css"><script src="/internal-tools/assets/app.js"></script>`
	if err := os.WriteFile(filepath.Join(site, "index.html"), []byte(html), 0o644); err != nil {
		t.Fatalf("WriteFile index: %v", err)
	}

	base := "/api/docs/internal-wiki/latest/guide/site/"
	count, err := rewriteStaticSiteBase(site, base)
	if err != nil {
		t.Fatalf("rewriteStaticSiteBase: %v", err)
	}
	if count != 3 {
		t.Fatalf("rewritten count = %d, want 3", count)
	}
	indexBytes, _ := os.ReadFile(filepath.Join(site, "index.html"))
	if index := string(indexBytes); strings.Contains(index, "/internal-tools/") || !strings.Contains(index, base+"assets/style.css") {
		t.Fatalf("legacy base was not normalized: %s", index)
	}
	cssBytes, _ := os.ReadFile(filepath.Join(site, "assets", "style.css"))
	if css := string(cssBytes); strings.Contains(css, "/internal-tools/") || !strings.Contains(css, base+"assets/bg.png") {
		t.Fatalf("legacy css base was not normalized: %s", css)
	}

	relative, err := rewriteStaticSiteBaseToRelative(site, base)
	if err != nil {
		t.Fatalf("rewriteStaticSiteBaseToRelative: %v", err)
	}
	if relative != 3 {
		t.Fatalf("relative count = %d, want 3", relative)
	}
	indexBytes, _ = os.ReadFile(filepath.Join(site, "index.html"))
	if index := string(indexBytes); !strings.Contains(index, `href="./assets/style.css"`) || !strings.Contains(index, `src="./assets/app.js"`) {
		t.Fatalf("legacy base was not relativized: %s", index)
	}
	cssBytes, _ = os.ReadFile(filepath.Join(site, "assets", "style.css"))
	if css := string(cssBytes); !strings.Contains(css, `url('./bg.png')`) {
		t.Fatalf("legacy css base was not relativized: %s", css)
	}
}

func TestBuildLogHelpersBoundAndRedactOutput(t *testing.T) {
	buffer := newTailBuffer(8)
	if _, err := buffer.Write([]byte("0123456789abcdef")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if got := buffer.String(); got != "89abcdef" {
		t.Fatalf("tail = %q, want 89abcdef", got)
	}
	redacted := redactBuildOutput("TOKEN=secret API_KEY:abc Authorization: Bearer xyz")
	if strings.Contains(redacted, "secret") || strings.Contains(redacted, "abc") || strings.Contains(redacted, "xyz") {
		t.Fatalf("sensitive values were not redacted: %s", redacted)
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
