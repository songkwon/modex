package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"modex/backend/internal/store"
)

func TestDocAnalyticsUnavailableWithoutPostHog(t *testing.T) {
	t.Setenv("POSTHOG_PERSONAL_API_KEY", "")
	t.Setenv("POSTHOG_PROJECT_ID", "")

	srv := New(store.New())
	req := httptest.NewRequest(http.MethodGet, "/api/analytics/doc?doc_id=guide", nil)
	rr := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(rr.Body.String(), `"code":"posthog_not_configured"`) {
		t.Fatalf("response does not explain unavailable analytics: %s", rr.Body.String())
	}
}

func TestBuiltInReadingAnalyticsRoutesAreRemoved(t *testing.T) {
	srv := New(store.New())
	tests := []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/analytics/page-view"},
		{method: http.MethodPost, path: "/api/analytics/read-progress"},
		{method: http.MethodGet, path: "/api/admin/analytics/pages"},
	}
	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(`{"doc_id":"guide"}`))
		rr := httptest.NewRecorder()

		srv.Handler().ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("%s status = %d, want %d", tt.path, rr.Code, http.StatusNotFound)
		}
	}
}
