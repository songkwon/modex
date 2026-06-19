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

func TestMCPLogRequiresAuthenticatedCaller(t *testing.T) {
	srv := New(store.NewSeeded())
	req := httptest.NewRequest(http.MethodPost, "/api/mcp/log", strings.NewReader(`{"tool_name":"search_docs"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestMCPLogAcceptsPersonalMCPToken(t *testing.T) {
	st := store.NewSeeded()
	current := st.CurrentUser()
	if _, err := st.SetUserMCPToken(current.ID, "mcp-test-token"); err != nil {
		t.Fatalf("SetUserMCPToken: %v", err)
	}
	srv := New(st)
	req := httptest.NewRequest(http.MethodPost, "/api/mcp/log", strings.NewReader(`{"tool_name":"search_docs","query":"guide","result_count":1}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer mcp-test-token")
	rr := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d: %s", rr.Code, http.StatusAccepted, rr.Body.String())
	}
	logs := st.MCPLogs()
	if len(logs) != 1 || logs[0].UserID != current.ID {
		t.Fatalf("logs = %+v, want one log attributed to %s", logs, current.ID)
	}
}

func TestUnknownAdminRouteReturnsNotFound(t *testing.T) {
	srv := New(store.NewSeeded())
	req := httptest.NewRequest(http.MethodPost, "/api/admin/unknown-placeholder", nil)
	rr := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
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
