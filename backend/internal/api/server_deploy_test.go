package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"modex/backend/internal/store"
)

func TestHealthIncludesOperationalSnapshot(t *testing.T) {
	srv := New(store.NewSeeded())
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
	st := store.NewSeeded()
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

func testDeployZip(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	files := map[string]string{
		"metadata.json":   `{"module_key":"DemoModule","module_name":"DemoModule","docs_version":"latest","package_version":"1.2.3"}`,
		"manifest.json":   `{"schema_version":"modex.docs/v1","generated_by":"test","entries":[{"key":"guide","title":"Guide","type":"markdown","source":"README.md"}]}`,
		"nav.json":        `[{"title":"Guide","path":"/guide"}]`,
		"documents.jsonl": `{"doc_id":"DemoModule:latest:guide","module_key":"DemoModule","module_name":"DemoModule","docs_version":"latest","entry_key":"guide","entry_type":"markdown","title":"Guide","content":"Hello","path":"/docs/DemoModule/latest/guide"}` + "\n",
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
