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

func TestAdminPluginsRequiresLogin(t *testing.T) {
	srv := New(store.NewSeeded())
	req := httptest.NewRequest(http.MethodGet, "/api/admin/plugins", nil)
	rr := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestConfigExposesPluginDefaults(t *testing.T) {
	srv := New(store.NewSeeded())
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	rr := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"kroki"`) || !strings.Contains(rr.Body.String(), `"plugins"`) {
		t.Fatalf("config missing plugin defaults: %s", rr.Body.String())
	}
}
