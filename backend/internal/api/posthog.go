package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"modex/backend/internal/store"
)

// errPosthogNotConfigured signals that reading statistics are unavailable.
var errPosthogNotConfigured = errors.New("posthog not configured")

// posthogHost returns the configured PostHog API host or the default.
func posthogHost() string {
	host := strings.TrimRight(os.Getenv("POSTHOG_HOST"), "/")
	if host == "" {
		return "https://app.posthog.com"
	}
	return host
}

// posthogConfigured reports whether the server-side PostHog query credentials
// are present. This is distinct from the frontend NEXT_PUBLIC_POSTHOG_KEY
// (capture key); querying read stats needs a personal/project API key.
func posthogConfigured() bool {
	return os.Getenv("POSTHOG_PERSONAL_API_KEY") != "" && os.Getenv("POSTHOG_PROJECT_ID") != ""
}

// PosthogConfigured and PosthogHost are exported wrappers used by package main
// (startup logging).
func PosthogConfigured() bool { return posthogConfigured() }
func PosthogHost() string     { return posthogHost() }

// posthogDocStats queries PostHog (HogQL) for the daily read trend and per-user
// reading totals/duration of one document. It returns an error when configured but
// the query fails, so callers can surface the problem instead of silently
// falling back. The event/property names match what the frontend captures:
// a "docs_page_view" event carrying a "doc_id" property.
func posthogDocStats(docID string, days int) (store.PageReadStats, error) {
	if !posthogConfigured() {
		return store.PageReadStats{}, errPosthogNotConfigured
	}
	host := posthogHost()
	projectID := os.Getenv("POSTHOG_PROJECT_ID")
	apiKey := os.Getenv("POSTHOG_PERSONAL_API_KEY")

	query := func(hogql string) ([][]any, error) {
		body, _ := json.Marshal(map[string]any{
			"query": map[string]any{"kind": "HogQLQuery", "query": hogql},
		})
		url := fmt.Sprintf("%s/api/projects/%s/query/", host, projectID)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 8 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("posthog returned %d", resp.StatusCode)
		}
		var out struct {
			Results [][]any `json:"results"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return nil, err
		}
		return out.Results, nil
	}

	esc := strings.ReplaceAll(docID, "'", "\\'")
	dailyHogQL := fmt.Sprintf(
		"SELECT toDate(timestamp) AS d, count() AS c FROM events "+
			"WHERE event = 'docs_page_view' AND properties.doc_id = '%s' "+
			"AND timestamp >= now() - INTERVAL %d DAY GROUP BY d ORDER BY d",
		esc, days)
	readersHogQL := fmt.Sprintf(
		"SELECT reader, user_id, count() AS c, avg(duration) AS avg_duration, max(last) AS last_read "+
			"FROM (SELECT coalesce(person.properties.name, person.properties.email, distinct_id) AS reader, "+
			"distinct_id AS user_id, properties.read_id AS read_id, "+
			"max(toFloat64OrZero(toString(properties.duration_seconds))) AS duration, max(timestamp) AS last "+
			"FROM events WHERE event = 'docs_page_read' AND properties.doc_id = '%s' "+
			"AND timestamp >= now() - INTERVAL %d DAY GROUP BY reader, user_id, read_id) "+
			"GROUP BY reader, user_id ORDER BY c DESC LIMIT 200",
		esc, days)

	dailyRows, err1 := query(dailyHogQL)
	readerRows, err2 := query(readersHogQL)
	if err1 != nil {
		return store.PageReadStats{}, fmt.Errorf("posthog daily query failed: %w", err1)
	}
	if err2 != nil {
		return store.PageReadStats{}, fmt.Errorf("posthog readers query failed: %w", err2)
	}

	// Build a zero-filled day window so the chart has no gaps.
	today := time.Now().UTC().Truncate(24 * time.Hour)
	idx := map[string]int{}
	daily := make([]store.DailyReadPoint, days)
	for i := 0; i < days; i++ {
		key := today.AddDate(0, 0, -(days - 1 - i)).Format("2006-01-02")
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
	totalDuration := 0
	totalTimedReads := 0
	for _, row := range readerRows {
		if len(row) < 5 {
			continue
		}
		name := fmt.Sprintf("%v", row[0])
		if name == "" || name == "<nil>" {
			name = "匿名"
		}
		count := toInt(row[2])
		avgDuration := toInt(row[3])
		last, _ := time.Parse(time.RFC3339, fmt.Sprintf("%v", row[4]))
		readers = append(readers, store.ReaderStat{
			Reader: name, UserID: fmt.Sprintf("%v", row[1]), Count: count,
			AvgDurationSec: avgDuration, LastReadAt: last,
		})
		totalDuration += count * avgDuration
		totalTimedReads += count
	}
	avgDuration := 0
	if totalTimedReads > 0 {
		avgDuration = totalDuration / totalTimedReads
	}
	return store.PageReadStats{
		DocID: docID, Total: total, AvgDurationSec: avgDuration,
		Daily: daily, Readers: readers,
	}, nil
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
