package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"modex/backend/internal/store"
)

// posthogConfigured reports whether the server-side PostHog query credentials
// are present. This is distinct from the frontend NEXT_PUBLIC_POSTHOG_KEY
// (capture key); querying read stats needs a personal/project API key.
func posthogConfigured() bool {
	return os.Getenv("POSTHOG_PERSONAL_API_KEY") != "" && os.Getenv("POSTHOG_PROJECT_ID") != ""
}

// posthogDocStats queries PostHog (HogQL) for the daily read trend and per-user
// breakdown of one document. It returns ok=false when PostHog is not configured
// or any error occurs, so the caller can fall back to the built-in store. The
// event/property names match what the frontend captures: a "docs_page_view"
// event carrying a "doc_id" property.
func posthogDocStats(docID string, days int) (store.PageReadStats, bool) {
	if !posthogConfigured() {
		return store.PageReadStats{}, false
	}
	host := strings.TrimRight(os.Getenv("POSTHOG_API_HOST"), "/")
	if host == "" {
		host = "https://app.posthog.com"
	}
	projectID := os.Getenv("POSTHOG_PROJECT_ID")
	apiKey := os.Getenv("POSTHOG_PERSONAL_API_KEY")

	query := func(hogql string) ([][]any, bool) {
		body, _ := json.Marshal(map[string]any{
			"query": map[string]any{"kind": "HogQLQuery", "query": hogql},
		})
		url := fmt.Sprintf("%s/api/projects/%s/query/", host, projectID)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, false
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 8 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, false
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, false
		}
		var out struct {
			Results [][]any `json:"results"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return nil, false
		}
		return out.Results, true
	}

	esc := strings.ReplaceAll(docID, "'", "\\'")
	dailyHogQL := fmt.Sprintf(
		"SELECT toDate(timestamp) AS d, count() AS c FROM events "+
			"WHERE event = 'docs_page_view' AND properties.doc_id = '%s' "+
			"AND timestamp >= now() - INTERVAL %d DAY GROUP BY d ORDER BY d",
		esc, days)
	readersHogQL := fmt.Sprintf(
		"SELECT coalesce(person.properties.name, person.properties.email, distinct_id) AS reader, "+
			"count() AS c, max(timestamp) AS last FROM events "+
			"WHERE event = 'docs_page_view' AND properties.doc_id = '%s' "+
			"GROUP BY reader ORDER BY c DESC LIMIT 200",
		esc)

	dailyRows, ok1 := query(dailyHogQL)
	readerRows, ok2 := query(readersHogQL)
	if !ok1 || !ok2 {
		return store.PageReadStats{}, false
	}

	// Build a zero-filled day window so the chart has no gaps.
	today := time.Now().UTC().Truncate(24 * time.Hour)
	idx := map[string]int{}
	daily := make([]store.DailyReadPoint, days)
	for i := 0; i < days; i++ {
		key := today.AddDate(0, 0, -(days-1-i)).Format("2006-01-02")
		daily[i] = store.DailyReadPoint{Date: key, Count: 0}
		idx[key] = i
	}
	total := 0
	for _, row := range dailyRows {
		if len(row) < 2 {
			continue
		}
		date := fmt.Sprintf("%v", row[0])
		if len(date) > 10 {
			date = date[:10]
		}
		c := toInt(row[1])
		if i, ok := idx[date]; ok {
			daily[i].Count = c
		}
		total += c
	}

	readers := make([]store.ReaderStat, 0, len(readerRows))
	for _, row := range readerRows {
		if len(row) < 3 {
			continue
		}
		name := fmt.Sprintf("%v", row[0])
		if name == "" || name == "<nil>" {
			name = "匿名"
		}
		last, _ := time.Parse(time.RFC3339, fmt.Sprintf("%v", row[2]))
		readers = append(readers, store.ReaderStat{Reader: name, Count: toInt(row[1]), LastReadAt: last})
	}
	return store.PageReadStats{DocID: docID, Total: total, Daily: daily, Readers: readers}, true
}

func toInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	}
	return 0
}
