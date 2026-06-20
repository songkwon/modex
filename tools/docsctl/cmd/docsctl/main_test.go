package main

import (
	"archive/zip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeployRequiresTokenBeforeUploading(t *testing.T) {
	dir := t.TempDir()
	artifact := filepath.Join(dir, "docs.zip")
	if err := os.WriteFile(artifact, []byte("zip-bytes"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	err := deploy(dir, filepath.Join(dir, "build"), artifact, "http://127.0.0.1:1/api/deploy", "")

	if err == nil || !strings.Contains(err.Error(), "deploy token is required") {
		t.Fatalf("deploy error = %v, want token guidance", err)
	}
}

func TestDeployRebuildsExistingArtifactBeforeUpload(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "index.md"), []byte("# Guide\n\nFresh docs."), 0o644); err != nil {
		t.Fatalf("write doc: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs.yaml"), []byte(`entries:
  - key: guide
    title: Guide
    type: markdown
    source: docs
`), 0o644); err != nil {
		t.Fatalf("write docs.yaml: %v", err)
	}

	artifact := filepath.Join(root, ".modex", "docs-artifact.zip")
	if err := writeArtifactMetadata(t, artifact, "v1.0.0"); err != nil {
		t.Fatalf("write stale artifact: %v", err)
	}
	t.Setenv("DOCS_MODULE", "demo")
	t.Setenv("DOCS_VERSION", "v1.0.3")

	var uploadedVersion string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read upload: %v", err)
		}
		uploadedVersion = readArtifactVersion(t, body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	err := deploy(root, filepath.Join(root, ".modex", "build"), artifact, server.URL, "secret")

	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if uploadedVersion != "v1.0.3" {
		t.Fatalf("uploaded docs_version = %q, want v1.0.3", uploadedVersion)
	}
}

func TestFormatAPIErrorIncludesFailedDeployStage(t *testing.T) {
	body := []byte(`{"error":{"code":"invalid_deploy_token","message":"deploy token required or invalid for this module"},"deploy":{"stages":[{"name":"parse_artifact","status":"ok"},{"name":"authenticate","status":"failed","error":"deploy token mismatch"}]}}`)

	got := formatAPIError(body)

	if !strings.Contains(got, "invalid_deploy_token") || !strings.Contains(got, "failed stage: authenticate") {
		t.Fatalf("formatted error = %q", got)
	}
}

func writeArtifactMetadata(t *testing.T, artifact, version string) error {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(artifact), 0o755); err != nil {
		return err
	}
	f, err := os.Create(artifact)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	w, err := zw.Create("metadata.json")
	if err != nil {
		_ = zw.Close()
		return err
	}
	if _, err = w.Write([]byte(`{"module_key":"demo","docs_version":"` + version + `"}`)); err != nil {
		_ = zw.Close()
		return err
	}
	return zw.Close()
}

func readArtifactVersion(t *testing.T, body []byte) string {
	t.Helper()
	zr, err := zip.NewReader(strings.NewReader(string(body)), int64(len(body)))
	if err != nil {
		t.Fatalf("open uploaded zip: %v", err)
	}
	for _, f := range zr.File {
		if f.Name != "metadata.json" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open metadata: %v", err)
		}
		defer rc.Close()
		var metadata struct {
			DocsVersion string `json:"docs_version"`
		}
		if err := json.NewDecoder(rc).Decode(&metadata); err != nil {
			t.Fatalf("decode metadata: %v", err)
		}
		return metadata.DocsVersion
	}
	t.Fatal("uploaded artifact missing metadata.json")
	return ""
}
