package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"modex/backend/internal/store"
)

func TestAdminReleaseRollbackRequiresLogin(t *testing.T) {
	srv := New(store.NewSeededTestStore())
	req := httptest.NewRequest(http.MethodPost, "/api/admin/releases/rel-demo-latest-001/rollback", nil)
	rr := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestOptionalCurrentUserProbeIsAnonymousWithout401(t *testing.T) {
	srv := New(store.NewTestStore())
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me?optional=1", nil)
	rr := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if strings.TrimSpace(rr.Body.String()) != "null" {
		t.Fatalf("body = %q, want null", rr.Body.String())
	}
}

func TestEmptyPublicModulesUsesJSONArray(t *testing.T) {
	srv := New(store.NewTestStore())
	req := httptest.NewRequest(http.MethodGet, "/api/modules", nil)
	rr := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if strings.TrimSpace(rr.Body.String()) != "[]" {
		t.Fatalf("body = %q, want []", rr.Body.String())
	}
}

func TestEmbedTextRequiresAdminSession(t *testing.T) {
	srv := New(store.NewSeededTestStore())
	req := httptest.NewRequest(http.MethodPost, "/api/embeddings/embed-text", strings.NewReader(`{"text":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestAdminPluginsRequiresLogin(t *testing.T) {
	srv := New(store.NewSeededTestStore())
	req := httptest.NewRequest(http.MethodGet, "/api/admin/plugins", nil)
	rr := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestTeamLeaderCanMoveChildCategoryWithinResponsibleParent(t *testing.T) {
	t.Setenv("SUPER_ADMIN_USERS", "")
	st := store.NewSeededTestStore()
	srv := New(st)
	alice, err := st.UserByID("u-alice")
	if err != nil {
		t.Fatalf("UserByID: %v", err)
	}
	login := httptest.NewRecorder()
	if err := srv.app.Auth().CreateSession(login, alice); err != nil {
		t.Fatalf("create session: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/admin/categories/standards.tools/move", strings.NewReader(`{"parent_id":"standards","index":0}`))
	req.Header.Set("Content-Type", "application/json")
	for _, cookie := range login.Result().Cookies() {
		req.AddCookie(cookie)
	}
	rr := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var moved store.Category
	if err := json.Unmarshal(rr.Body.Bytes(), &moved); err != nil {
		t.Fatal(err)
	}
	if moved.ID != "standards.tools" || moved.ParentID != "standards" || moved.SortOrder != 10 {
		t.Fatalf("moved category = %+v", moved)
	}
}

func TestMCPLogRequiresAuthenticatedCaller(t *testing.T) {
	srv := New(store.NewSeededTestStore())
	req := httptest.NewRequest(http.MethodPost, "/api/mcp/log", strings.NewReader(`{"tool_name":"search_docs"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestMCPLogAcceptsPersonalMCPToken(t *testing.T) {
	st := store.NewSeededTestStore()
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

func TestMCPTokenInfoAcceptsPersonalMCPToken(t *testing.T) {
	st := store.NewSeededTestStore()
	current := st.CurrentUser()
	if _, err := st.SetUserMCPToken(current.ID, "mcp-test-token"); err != nil {
		t.Fatalf("SetUserMCPToken: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/mcp/token-info", nil)
	req.Header.Set("Authorization", "Bearer mcp-test-token")
	rr := httptest.NewRecorder()

	New(st).Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rr.Code, rr.Body.String())
	}
	var info struct {
		UserID string   `json:"user_id"`
		Scopes []string `json:"scopes"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &info); err != nil {
		t.Fatal(err)
	}
	if info.UserID != current.ID || len(info.Scopes) != 2 {
		t.Fatalf("token info = %+v", info)
	}
}

func TestUnknownAdminRouteReturnsNotFound(t *testing.T) {
	srv := New(store.NewSeededTestStore())
	req := httptest.NewRequest(http.MethodPost, "/api/admin/unknown-placeholder", nil)
	rr := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestConfigExposesPluginDefaults(t *testing.T) {
	srv := New(store.NewSeededTestStore())
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
