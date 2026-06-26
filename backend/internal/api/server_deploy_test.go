package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"modex/backend/internal/deploy"
	"modex/backend/internal/store"
)

func TestHealthIncludesOperationalSnapshot(t *testing.T) {
	srv := New(store.NewSeededTestStore())
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{`"dependencies"`, `"counts"`, `"modules"`, `"pages"`, `"object_storage"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("health response missing %s: %s", want, body)
		}
	}
}

func TestRewriteServedSiteRootRefsHandlesLegacyVitePressBase(t *testing.T) {
	input := []byte(`<link href="/internal-tools/assets/style.css"><script src="/internal-tools/assets/app.js"></script><a href="/internal-tools/posts/cbb-shelf/faq.html">FAQ</a><script>const route="/internal-tools/posts/cbb-shelf/faq.html";const keep="/standards/sidebar-state";history.pushState(null,"",'/internal-tools/posts/cbb-shelf/system.html')</script><img srcset="/internal-tools/assets/a.png 1x, /internal-tools/assets/b.png 2x"><style>.x{background:url('/internal-tools/assets/bg.png')}</style>`)

	out := string(rewriteServedSiteRootRefs(input, "internal-wiki", "latest", "guide"))

	base := "/api/docs/internal-wiki/latest/guide/site/"
	for _, want := range []string{
		`href="` + base + `assets/style.css"`,
		`src="` + base + `assets/app.js"`,
		base + `assets/a.png 1x`,
		base + `assets/b.png 2x`,
		`url('` + base + `assets/bg.png')`,
		`href="` + base + `posts/cbb-shelf/faq.html"`,
		`"` + base + `posts/cbb-shelf/faq.html"`,
		`'` + base + `posts/cbb-shelf/system.html'`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("rewritten output missing %q: %s", want, out)
		}
	}
	if strings.Contains(out, "/internal-tools/") {
		t.Fatalf("legacy base still present: %s", out)
	}
	if !strings.Contains(out, `"/standards/sidebar-state"`) {
		t.Fatalf("unrelated JS string path was rewritten: %s", out)
	}
}

func TestSiteFileCandidatesSupportVitePressRoutes(t *testing.T) {
	cases := map[string][]string{
		"":                                     {"index.html"},
		"posts/cbb-shelf/system-overview":      {"posts/cbb-shelf/system-overview", "posts/cbb-shelf/system-overview.html", "posts/cbb-shelf/system-overview/index.html", "index.html"},
		"posts/cbb-shelf/system-overview.md":   {"posts/cbb-shelf/system-overview.md", "posts/cbb-shelf/system-overview.html"},
		"assets/style.css":                     {"assets/style.css"},
		"/posts/process-tools/itr/user-guide/": {"posts/process-tools/itr/user-guide", "posts/process-tools/itr/user-guide.html", "posts/process-tools/itr/user-guide/index.html", "index.html"},
	}
	for input, want := range cases {
		got := siteFileCandidates(input)
		if strings.Join(got, "|") != strings.Join(want, "|") {
			t.Fatalf("siteFileCandidates(%q) = %#v, want %#v", input, got, want)
		}
	}
}

func TestDeployErrorIncludesStageReport(t *testing.T) {
	st := store.NewSeededTestStore()
	if _, err := st.UpdateModule("DemoModule", store.Module{DeployToken: "secret"}); err != nil {
		t.Fatalf("UpdateModule: %v", err)
	}
	srv := New(st)
	req := httptest.NewRequest(http.MethodPost, "/api/deploy", bytes.NewReader(testDeployZip(t)))
	req.Header.Set("Content-Type", "application/zip")
	req.Header.Set("X-Modex-Deploy-Token", "wrong")
	rr := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403: %s", rr.Code, rr.Body.String())
	}
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
		Deploy struct {
			Stages []struct {
				Name   string `json:"name"`
				Status string `json:"status"`
			} `json:"stages"`
		} `json:"deploy"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Error.Code != "invalid_deploy_token" {
		t.Fatalf("error code = %q", payload.Error.Code)
	}
	if len(payload.Deploy.Stages) != 2 || payload.Deploy.Stages[0].Name != "parse_artifact" || payload.Deploy.Stages[0].Status != "ok" || payload.Deploy.Stages[1].Name != "authenticate" || payload.Deploy.Stages[1].Status != "failed" {
		t.Fatalf("unexpected deploy stages: %+v", payload.Deploy.Stages)
	}
}

func TestDeployTokenSelectsDocumentSource(t *testing.T) {
	st := store.NewSeededTestStore()
	if _, err := st.UpdateModule("DemoModule", store.Module{DeployToken: "secret"}); err != nil {
		t.Fatalf("UpdateModule: %v", err)
	}
	srv := New(st)
	req := httptest.NewRequest(http.MethodPost, "/api/deploy", bytes.NewReader(testDeployZipForModule(t, "WrongModule")))
	req.Header.Set("Content-Type", "application/zip")
	req.Header.Set("X-Modex-Deploy-Token", "secret")
	rr := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202: %s", rr.Code, rr.Body.String())
	}
	if _, err := st.Page("DemoModule:latest:guide"); err != nil {
		t.Fatalf("expected page to be indexed under token-owned module: %v", err)
	}
	if _, err := st.Page("WrongModule:latest:guide"); err == nil {
		t.Fatal("page was indexed under artifact module instead of token-owned module")
	}
}

func TestCanonicalizeDeployArtifactAppliesSplitMount(t *testing.T) {
	artifact := deployArtifactForSplitTest("WrongModule")

	got := canonicalizeDeployArtifact(artifact, store.Module{
		ModuleKey: "DemoModule",
		Name:      "Demo Module",
		DocType:   "markdown",
		Mount:     "split",
	})

	if len(got.Manifest.Entries) != 2 {
		t.Fatalf("entries = %#v, want two top-level groups", got.Manifest.Entries)
	}
	if got.Manifest.Entries[0].Key != "standard" || got.Manifest.Entries[1].Key != "tools" {
		t.Fatalf("entry keys = %#v, want standard/tools", got.Manifest.Entries)
	}
	if len(got.Documents) != 2 {
		t.Fatalf("documents = %#v, want two grouped documents", got.Documents)
	}
	if got.Documents[0].DocID != "DemoModule:latest:standard" || got.Documents[1].DocID != "DemoModule:latest:tools" {
		t.Fatalf("doc ids = %#v", got.Documents)
	}
	if !strings.Contains(got.Documents[0].ContentMD, "Standard A") || !strings.Contains(got.Documents[1].ContentMD, "Tools B") {
		t.Fatalf("grouped markdown content missing source text: %#v", got.Documents)
	}
}

func TestCanonicalizeDeployArtifactRewritesContentResourceBase(t *testing.T) {
	artifact := deploy.Artifact{
		Metadata: deploy.Metadata{ModuleKey: "UnknownModule", ModuleName: "UnknownModule", DocsVersion: "latest", PackageVersion: "1.0.0"},
		Manifest: deploy.Manifest{Entries: []deploy.Entry{{Key: "guide", Title: "Guide", Type: "markdown", Source: "docs"}}},
		Documents: []deploy.DocumentRecord{{
			DocID:       "UnknownModule:latest:guide",
			ModuleKey:   "UnknownModule",
			DocsVersion: "latest",
			EntryKey:    "guide",
			EntryType:   "markdown",
			Title:       "Guide",
			Content:     "see /api/docs/UnknownModule/latest/guide/site/images/shot.png",
			ContentMD:   "![shot](/api/docs/UnknownModule/latest/guide/site/images/shot.png)",
		}},
	}

	got := canonicalizeDeployArtifact(artifact, store.Module{ModuleKey: "standards", Name: "Standards"})

	if strings.Contains(got.Documents[0].ContentMD, "UnknownModule") || strings.Contains(got.Documents[0].Content, "UnknownModule") {
		t.Fatalf("placeholder module not rewritten: %#v", got.Documents[0])
	}
	if !strings.Contains(got.Documents[0].ContentMD, "/api/docs/standards/latest/guide/site/images/shot.png") {
		t.Fatalf("content not rewritten to resolved module: %q", got.Documents[0].ContentMD)
	}
}

func TestCanonicalizeDeployArtifactPreservesSitePageDocIDs(t *testing.T) {
	artifact := deploy.Artifact{
		Metadata: deploy.Metadata{ModuleKey: "UnknownModule", ModuleName: "UnknownModule", DocsVersion: "latest"},
		Manifest: deploy.Manifest{Entries: []deploy.Entry{{Key: "guide", Title: "Guide", Type: "vitepress", Source: "."}}},
		Documents: []deploy.DocumentRecord{
			{DocID: "UnknownModule:latest:guide/", ModuleKey: "UnknownModule", DocsVersion: "latest", EntryKey: "guide", EntryType: "vitepress", Title: "Home", Content: "home"},
			{DocID: "UnknownModule:latest:guide/standards/coding/memory-safety", ModuleKey: "UnknownModule", DocsVersion: "latest", EntryKey: "guide", EntryType: "vitepress", Title: "Memory Safety", Content: "memory safety"},
		},
	}

	got := canonicalizeDeployArtifact(artifact, store.Module{ModuleKey: "standards", Name: "Standards"})

	if len(got.Documents) != 2 {
		t.Fatalf("documents = %#v, want two pages", got.Documents)
	}
	if got.Documents[0].DocID != "standards:latest:guide/" {
		t.Fatalf("first doc id = %q", got.Documents[0].DocID)
	}
	if got.Documents[1].DocID != "standards:latest:guide/standards/coding/memory-safety" {
		t.Fatalf("second doc id = %q", got.Documents[1].DocID)
	}
}

func TestModelEndpointAppendsExpectedSuffixOnce(t *testing.T) {
	tests := []struct {
		name   string
		base   string
		suffix string
		want   string
	}{
		{name: "embedding base", base: "https://api.example.com/v1", suffix: "/embeddings", want: "https://api.example.com/v1/embeddings"},
		{name: "embedding endpoint", base: "https://api.example.com/v1/embeddings", suffix: "/embeddings", want: "https://api.example.com/v1/embeddings"},
		{name: "rerank base", base: "https://api.example.com/v1/", suffix: "/rerank", want: "https://api.example.com/v1/rerank"},
		{name: "rerank endpoint", base: "https://api.example.com/v1/rerank", suffix: "/rerank", want: "https://api.example.com/v1/rerank"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := modelEndpoint(tt.base, tt.suffix); got != tt.want {
				t.Fatalf("modelEndpoint() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestChatEndpointUsesSelectedProtocol(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		base     string
		model    string
		want     string
	}{
		{name: "openai chat", protocol: "openai-chat", base: "https://api.example.com/v1", model: "gpt", want: "https://api.example.com/v1/chat/completions"},
		{name: "responses", protocol: "openai-responses", base: "https://api.example.com/v1/", model: "gpt", want: "https://api.example.com/v1/responses"},
		{name: "anthropic", protocol: "anthropic", base: "https://api.anthropic.com", model: "claude", want: "https://api.anthropic.com/v1/messages"},
		{name: "gemini", protocol: "gemini", base: "https://generativelanguage.googleapis.com", model: "gemini pro", want: "https://generativelanguage.googleapis.com/v1beta/models/gemini%20pro:generateContent"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := chatEndpoint(tt.base, tt.protocol, tt.model); got != tt.want {
				t.Fatalf("chatEndpoint() = %q, want %q", got, tt.want)
			}
		})
	}
}

func testDeployZip(t *testing.T) []byte {
	return testDeployZipForModule(t, "DemoModule")
}

func deployArtifactForSplitTest(moduleKey string) deploy.Artifact {
	return deploy.Artifact{
		Metadata: deploy.Metadata{ModuleKey: moduleKey, ModuleName: moduleKey, DocsVersion: "latest", PackageVersion: "1.0.0"},
		Manifest: deploy.Manifest{Entries: []deploy.Entry{
			{Key: "guide-standard-a", Title: "Standard A", Type: "markdown", Source: "docs/standard/a.md"},
			{Key: "guide-tools-b", Title: "Tools B", Type: "markdown", Source: "docs/tools/b.md"},
		}},
		Documents: []deploy.DocumentRecord{
			{DocID: moduleKey + ":latest:guide-standard-a", ModuleKey: moduleKey, ModuleName: moduleKey, DocsVersion: "latest", EntryKey: "guide-standard-a", EntryType: "markdown", Title: "Standard A", SourceFile: "docs/standard/a.md", Content: "Standard text", ContentMD: "# Standard A"},
			{DocID: moduleKey + ":latest:guide-tools-b", ModuleKey: moduleKey, ModuleName: moduleKey, DocsVersion: "latest", EntryKey: "guide-tools-b", EntryType: "markdown", Title: "Tools B", SourceFile: "docs/tools/b.md", Content: "Tools text", ContentMD: "# Tools B"},
		},
	}
}

func testDeployZipForModule(t *testing.T, moduleKey string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	files := map[string]string{
		"metadata.json":   `{"module_key":"` + moduleKey + `","module_name":"` + moduleKey + `","docs_version":"latest","package_version":"1.2.3"}`,
		"manifest.json":   `{"schema_version":"modex.docs/v1","generated_by":"test","entries":[{"key":"guide","title":"Guide","type":"markdown","source":"README.md"}]}`,
		"nav.json":        `[{"title":"Guide","path":"/guide"}]`,
		"documents.jsonl": `{"doc_id":"` + moduleKey + `:latest:guide","module_key":"` + moduleKey + `","module_name":"` + moduleKey + `","docs_version":"latest","entry_key":"guide","entry_type":"markdown","title":"Guide","content":"Hello","path":"/docs/` + moduleKey + `/latest/guide"}` + "\n",
		"llms.txt":        "Guide\n",
	}
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip entry: %v", err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry: %v", err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}
