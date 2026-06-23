package dburl

import (
	"strings"
	"testing"
)

func TestFromEnvPrefersDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://custom:secret@db.example.com:6543/custom?sslmode=require")
	t.Setenv("POSTGRES_HOST", "ignored")

	if got := FromEnv(); got != "postgres://custom:secret@db.example.com:6543/custom?sslmode=require" {
		t.Fatalf("FromEnv() = %q", got)
	}
}

func TestFromEnvBuildsFromPostgresParts(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "10.0.0.5")
	t.Setenv("POSTGRES_PORT", "15432")
	t.Setenv("POSTGRES_DB", "modex_prod")
	t.Setenv("POSTGRES_USER", "modex_user")
	t.Setenv("POSTGRES_PASSWORD", "p@ss/word")
	t.Setenv("POSTGRES_SSLMODE", "require")

	got := FromEnv()
	want := "postgres://modex_user:p%40ss%2Fword@10.0.0.5:15432/modex_prod?sslmode=require"
	if got != want {
		t.Fatalf("FromEnv() = %q, want %q", got, want)
	}
}

func TestFromEnvDefaultsToComposePostgres(t *testing.T) {
	got := FromEnv()
	if !strings.HasPrefix(got, "postgres://modex:") {
		t.Fatalf("FromEnv() = %q, want default modex credentials", got)
	}
	if !strings.Contains(got, "@postgres:5432/modex?") {
		t.Fatalf("FromEnv() = %q, want default compose host and database", got)
	}
	if !strings.Contains(got, "sslmode=disable") {
		t.Fatalf("FromEnv() = %q, want sslmode=disable", got)
	}
}
