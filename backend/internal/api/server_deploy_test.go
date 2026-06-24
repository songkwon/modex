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
