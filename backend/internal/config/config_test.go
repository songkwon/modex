package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRejectsMissingExplicitConfig(t *testing.T) {
	t.Setenv("CONFIG_FILE", filepath.Join(t.TempDir(), "missing.yaml"))
	_, err := Load()
	if !errors.Is(err, ErrConfigNotFound) {
		t.Fatalf("Load error = %v, want ErrConfigNotFound", err)
	}
}

func TestLoadExplicitConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("auth:\n  user_mapping:\n    unique_id_claim: sub\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CONFIG_FILE", path)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Auth.UserMapping.UniqueIDClaim != "sub" {
		t.Fatalf("unique id claim = %q", cfg.Auth.UserMapping.UniqueIDClaim)
	}
}

func TestLoadUserMappingIgnoresOIDCClaimEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("auth:\n  user_mapping:\n    unique_id_claim: sub\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CONFIG_FILE", path)
	t.Setenv("OIDC_CLAIM_UNIQUE_ID", "email")

	mapping := LoadUserMapping()
	if mapping.UniqueIDClaim != "sub" {
		t.Fatalf("unique id claim = %q, want config file value", mapping.UniqueIDClaim)
	}
}

func TestLoadSearchScoring(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("search:\n  scoring:\n    keyword_weight: 0.7\n    semantic_weight: 0.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CONFIG_FILE", path)

	scoring := LoadSearchScoring()
	if scoring.KeywordWeight != 0.7 || scoring.SemanticWeight != 0.3 {
		t.Fatalf("search scoring = %+v", scoring)
	}
}
