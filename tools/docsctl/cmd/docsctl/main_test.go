package main

import (
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

func TestFormatAPIErrorIncludesFailedDeployStage(t *testing.T) {
	body := []byte(`{"error":{"code":"invalid_deploy_token","message":"deploy token required or invalid for this module"},"deploy":{"stages":[{"name":"parse_artifact","status":"ok"},{"name":"authenticate","status":"failed","error":"deploy token mismatch"}]}}`)

	got := formatAPIError(body)

	if !strings.Contains(got, "invalid_deploy_token") || !strings.Contains(got, "failed stage: authenticate") {
		t.Fatalf("formatted error = %q", got)
	}
}
