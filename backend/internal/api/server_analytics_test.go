package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"modex/backend/internal/store"
)

func TestDocAnalyticsFallsBackToBuiltinWithoutPostHog(t *testing.T) {
	t.Setenv("POSTHOG_PERSONAL_API_KEY", "")
	t.Setenv("POSTHOG_PROJECT_ID", "")

	srv := New(store.NewSeededTestStore())
	viewReq := httptest.NewRequest(http.MethodPost, "/api/analytics/page-view", strings.NewReader(`{"doc_id":"DemoModule:latest:guide","session_id":"s1","read_id":"r1"}`))
	viewReq.Header.Set("Content-Type", "application/json")
	viewRR := httptest.NewRecorder()
	srv.Handler().ServeHTTP(viewRR, viewReq)
	if viewRR.Code != http.StatusAccepted {
		t.Fatalf("page-view status = %d, want %d: %s", viewRR.Code, http.StatusAccepted, viewRR.Body.String())
	}
	progressReq := httptest.NewRequest(http.MethodPost, "/api/analytics/read-progress", strings.NewReader(`{"doc_id":"DemoModule:latest:guide","session_id":"s1","read_id":"r1","duration_seconds":42,"scroll_depth":0.8}`))
	progressReq.Header.Set("Content-Type", "application/json")
	progressRR := httptest.NewRecorder()
	srv.Handler().ServeHTTP(progressRR, progressReq)
	if progressRR.Code != http.StatusAccepted {
		t.Fatalf("read-progress status = %d, want %d: %s", progressRR.Code, http.StatusAccepted, progressRR.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/api/analytics/doc?doc_id=DemoModule:latest:guide", nil)
	rr := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), `"source":"builtin"`) {
		t.Fatalf("response does not use builtin analytics: %s", rr.Body.String())
	}
}

func TestAdminPageAnalyticsRouteIsNotPublic(t *testing.T) {
	srv := New(store.NewTestStore())
	req := httptest.NewRequest(http.MethodGet, "/api/admin/analytics/pages", nil)
	rr := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}
