package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestPosthogDocStatsIncludesRangeAndReaderDuration(t *testing.T) {
	var mu sync.Mutex
	queries := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Query struct {
				Query string `json:"query"`
			} `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		mu.Lock()
		queries = append(queries, body.Query.Query)
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(body.Query.Query, "docs_page_read") {
			_, _ = w.Write([]byte(`{"results":[["Alice","user-1",2,75.5,"2026-06-19T10:00:00Z"]]}`))
			return
		}
		_, _ = w.Write([]byte(`{"results":[["2026-06-19",3]]}`))
	}))
	defer server.Close()

	t.Setenv("POSTHOG_HOST", server.URL)
	t.Setenv("POSTHOG_PERSONAL_API_KEY", "secret")
	t.Setenv("POSTHOG_PROJECT_ID", "42")

	stats, err := posthogDocStats("guide", 30)
	if err != nil {
		t.Fatalf("posthogDocStats: %v", err)
	}
	if stats.Total != 3 || stats.AvgDurationSec != 75 {
		t.Fatalf("stats totals = %+v", stats)
	}
	if len(stats.Readers) != 1 || stats.Readers[0].Count != 2 || stats.Readers[0].AvgDurationSec != 75 {
		t.Fatalf("reader stats = %+v", stats.Readers)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(queries) != 2 {
		t.Fatalf("queries = %d, want 2", len(queries))
	}
	for _, query := range queries {
		if !strings.Contains(query, "INTERVAL 30 DAY") {
			t.Errorf("query missing selected range: %s", query)
		}
	}
	readerQuery := queries[1]
	if !strings.Contains(readerQuery, "GROUP BY reader, user_id, read_id") || !strings.Contains(readerQuery, "avg(duration)") {
		t.Errorf("reader query does not deduplicate reads and average duration: %s", readerQuery)
	}
}
