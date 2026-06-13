package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"modex/backend/internal/store"
)

func TestAdminReleaseRollbackRequiresLogin(t *testing.T) {
	srv := New(store.NewSeeded())
	req := httptest.NewRequest(http.MethodPost, "/api/admin/releases/rel-demo-latest-001/rollback", nil)
	rr := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestAdminPageAnalyticsRequiresLogin(t *testing.T) {
	srv := New(store.NewSeeded())
	req := httptest.NewRequest(http.MethodGet, "/api/admin/analytics/pages", nil)
	rr := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestEmbedTextRequiresAdminSession(t *testing.T) {
	srv := New(store.NewSeeded())
	req := httptest.NewRequest(http.MethodPost, "/api/embeddings/embed-text", strings.NewReader(`{"text":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}
