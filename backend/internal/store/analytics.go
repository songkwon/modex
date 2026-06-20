package store

import (
	"sort"
	"strings"
	"time"
)

func (s *Store) AddSearchLog(log SearchLog) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.searchLogs = append(s.searchLogs, log)
}

func (s *Store) SearchLogs() []SearchLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]SearchLog(nil), s.searchLogs...)
}

func (s *Store) AddMCPLog(log MCPLog) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mcpLogs = append(s.mcpLogs, log)
}

func (s *Store) MCPLogs() []MCPLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]MCPLog(nil), s.mcpLogs...)
}

func (s *Store) AddDocFeedback(f DocFeedback) DocFeedback {
	s.mu.Lock()
	defer s.mu.Unlock()
	if f.CreatedAt.IsZero() {
		f.CreatedAt = time.Now().UTC()
	}
	if f.ID == "" {
		f.ID = s.nextIDLocked("df")
	}
	for _, p := range s.pages {
		if p.DocID == f.DocID {
			f.PageID = p.ID
			f.ModuleKey = p.ModuleKey
			f.Title = p.Title
			break
		}
	}
	s.feedbacks = append(s.feedbacks, f)
	return f
}

func (s *Store) DocFeedbacks() []DocFeedback {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]DocFeedback(nil), s.feedbacks...)
}

// RecordPageView appends a page view and returns the stored record.
func (s *Store) RecordPageView(pv PageView) PageView {
	s.mu.Lock()
	defer s.mu.Unlock()
	if pv.ViewedAt.IsZero() {
		pv.ViewedAt = time.Now().UTC()
	}
	if pv.ID == "" {
		pv.ID = s.nextIDLocked("pv")
	}
	for _, p := range s.pages {
		if p.DocID == pv.DocID {
			pv.PageID = p.ID
			pv.ModuleKey = p.ModuleKey
			pv.ModuleName = p.ModuleName
			pv.DocsVersion = p.DocsVersion
			pv.EntryKey = p.EntryKey
			pv.Title = p.Title
			pv.Path = p.Path
			break
		}
	}
	s.pageViews = append(s.pageViews, pv)
	return pv
}

// RecordReadProgress updates the latest matching page view with duration and
// scroll depth for the given session and doc, or records a new view if none.
func (s *Store) RecordReadProgress(docID, sessionID, readID string, durationSeconds int, scrollDepth float64) PageView {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := len(s.pageViews) - 1; i >= 0; i-- {
		pv := &s.pageViews[i]
		if pv.DocID == docID && ((readID != "" && pv.ReadID == readID) || (readID == "" && pv.SessionID == sessionID)) {
			if durationSeconds > pv.DurationSeconds {
				pv.DurationSeconds = durationSeconds
			}
			if scrollDepth > pv.ScrollDepth {
				pv.ScrollDepth = scrollDepth
			}
			return *pv
		}
	}
	pv := PageView{
		ID: s.nextIDLocked("pv"), DocID: docID, SessionID: sessionID, ReadID: readID,
		DurationSeconds: durationSeconds, ScrollDepth: scrollDepth, ViewedAt: time.Now().UTC(),
	}
	for _, p := range s.pages {
		if p.DocID == pv.DocID {
			pv.PageID = p.ID
			pv.ModuleKey = p.ModuleKey
			pv.ModuleName = p.ModuleName
			pv.DocsVersion = p.DocsVersion
			pv.EntryKey = p.EntryKey
			pv.Title = p.Title
			pv.Path = p.Path
			break
		}
	}
	s.pageViews = append(s.pageViews, pv)
	return pv
}

// PageAnalytics aggregates recorded views into per-page reading statistics.
// Pages with no recorded views fall back to seeded read counts so the admin
// dashboard is populated on a fresh start.
func (s *Store) PageAnalytics() []PageStat {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now().UTC()
	week := now.AddDate(0, 0, -7)
	month := now.AddDate(0, 0, -30)
	type agg struct {
		pv, reads7, reads30, durSum, durCount int
		users                                 map[string]struct{}
		last                                  time.Time
	}
	byDoc := map[string]*agg{}
	for _, pv := range s.pageViews {
		a := byDoc[pv.DocID]
		if a == nil {
			a = &agg{users: map[string]struct{}{}}
			byDoc[pv.DocID] = a
		}
		a.pv++
		uid := pv.UserID
		if uid == "" {
			uid = pv.SessionID
		}
		if uid != "" {
			a.users[uid] = struct{}{}
		}
		if pv.ViewedAt.After(week) {
			a.reads7++
		}
		if pv.ViewedAt.After(month) {
			a.reads30++
		}
		if pv.DurationSeconds > 0 {
			a.durSum += pv.DurationSeconds
			a.durCount++
		}
		if pv.ViewedAt.After(a.last) {
			a.last = pv.ViewedAt
		}
	}
	var out []PageStat
	for _, p := range s.pages {
		stat := PageStat{DocID: p.DocID, Title: p.Title, ModuleKey: p.ModuleKey, ModuleName: p.ModuleName, DocsVersion: p.DocsVersion, Path: p.Path, LastViewedAt: p.UpdatedAt}
		if a := byDoc[p.DocID]; a != nil {
			stat.PV = a.pv
			stat.UV = len(a.users)
			stat.Reads7d = a.reads7
			stat.Reads30d = a.reads30
			stat.LastViewedAt = a.last
			if a.durCount > 0 {
				stat.AvgDurationSec = a.durSum / a.durCount
			}
		} else {
			stat.Reads7d = s.seedReadsLocked(p.ModuleKey, true)
			stat.Reads30d = s.seedReadsLocked(p.ModuleKey, false)
		}
		out = append(out, stat)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].PV != out[j].PV {
			return out[i].PV > out[j].PV
		}
		return out[i].Reads30d > out[j].Reads30d
	})
	return out
}

func (s *Store) seedReadsLocked(moduleKey string, week bool) int {
	for _, m := range s.modules {
		if strings.EqualFold(m.ModuleKey, moduleKey) {
			if week {
				return m.Reads7d
			}
			return m.Reads30d
		}
	}
	return 0
}

// PageReadStats aggregates recorded views for one document into a daily read
// trend (last `days` days, inclusive of today) plus a per-reader breakdown.
// Readers are keyed by user id, falling back to session id for anonymous views.
func (s *Store) PageReadStats(docID string, days int) PageReadStats {
	if days <= 0 {
		days = 30
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now().UTC()
	today := now.Truncate(24 * time.Hour)
	// Pre-seed every day in the window so the line chart has no gaps.
	idxByDate := map[string]int{}
	daily := make([]DailyReadPoint, days)
	for i := 0; i < days; i++ {
		d := today.AddDate(0, 0, -(days - 1 - i))
		key := d.Format("2006-01-02")
		daily[i] = DailyReadPoint{Date: key, Count: 0}
		idxByDate[key] = i
	}
	windowStart := today.AddDate(0, 0, -(days - 1))

	type ragg struct {
		userID   string
		count    int
		duration int
		last     time.Time
	}
	readers := map[string]*ragg{}
	total := 0
	totalDuration := 0
	timedReads := 0
	for _, pv := range s.pageViews {
		if pv.DocID != docID {
			continue
		}
		total++
		if pv.DurationSeconds > 0 {
			totalDuration += pv.DurationSeconds
			timedReads++
		}
		if !pv.ViewedAt.Before(windowStart) {
			if i, ok := idxByDate[pv.ViewedAt.UTC().Format("2006-01-02")]; ok {
				daily[i].Count++
			}
		}
		key := pv.UserID
		if key == "" {
			key = "session:" + pv.SessionID
		}
		r := readers[key]
		if r == nil {
			r = &ragg{userID: pv.UserID}
			readers[key] = r
		}
		r.count++
		r.duration += pv.DurationSeconds
		if pv.ViewedAt.After(r.last) {
			r.last = pv.ViewedAt
		}
	}

	out := PageReadStats{DocID: docID, Total: total, Daily: daily, Readers: []ReaderStat{}}
	if timedReads > 0 {
		out.AvgDurationSec = totalDuration / timedReads
	}
	for _, r := range readers {
		name := "匿名"
		if r.userID != "" {
			if u, err := s.userByIDLocked(r.userID); err == nil {
				if u.DisplayName != "" {
					name = u.DisplayName
				} else if u.Username != "" {
					name = u.Username
				}
			} else {
				name = r.userID
			}
		}
		avgDuration := 0
		if r.count > 0 {
			avgDuration = r.duration / r.count
		}
		out.Readers = append(out.Readers, ReaderStat{Reader: name, UserID: r.userID, Count: r.count, AvgDurationSec: avgDuration, LastReadAt: r.last})
	}
	sort.Slice(out.Readers, func(i, j int) bool {
		if out.Readers[i].Count != out.Readers[j].Count {
			return out.Readers[i].Count > out.Readers[j].Count
		}
		return out.Readers[i].LastReadAt.After(out.Readers[j].LastReadAt)
	})
	return out
}

func (s *Store) UserFavorites(userID string) []UserFavorite {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []UserFavorite{}
	for _, f := range s.favorites {
		if f.UserID == userID {
			out = append(out, f)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (s *Store) SetUserFavorite(userID, moduleKey string, favorite bool) ([]UserFavorite, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	userID = strings.TrimSpace(userID)
	moduleKey = strings.TrimSpace(moduleKey)
	if userID == "" || moduleKey == "" {
		return nil, ErrInvalid
	}
	exists := false
	for _, m := range s.modules {
		if m.ModuleKey == moduleKey {
			exists = true
			break
		}
	}
	if !exists {
		return nil, ErrNotFound
	}
	for i := range s.favorites {
		if s.favorites[i].UserID == userID && s.favorites[i].ModuleKey == moduleKey {
			if favorite {
				return s.userFavoritesLocked(userID), nil
			}
			s.favorites = append(s.favorites[:i], s.favorites[i+1:]...)
			return s.userFavoritesLocked(userID), nil
		}
	}
	if favorite {
		s.favorites = append(s.favorites, UserFavorite{ID: s.nextIDLocked("fav"), UserID: userID, ModuleKey: moduleKey, CreatedAt: time.Now().UTC()})
	}
	return s.userFavoritesLocked(userID), nil
}

func (s *Store) userFavoritesLocked(userID string) []UserFavorite {
	out := []UserFavorite{}
	for _, f := range s.favorites {
		if f.UserID == userID {
			out = append(out, f)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (s *Store) UserRecentDocs(userID string, limit int) []UserRecentDoc {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 {
		limit = 30
	}
	out := []UserRecentDoc{}
	for _, r := range s.recentDocs {
		if r.UserID == userID {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ViewedAt.After(out[j].ViewedAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func (s *Store) RecordUserRecentDoc(userID string, recent UserRecentDoc) (UserRecentDoc, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	userID = strings.TrimSpace(userID)
	recent.DocID = strings.TrimSpace(recent.DocID)
	if userID == "" || recent.DocID == "" {
		return UserRecentDoc{}, ErrInvalid
	}
	for _, p := range s.pages {
		if p.DocID == recent.DocID {
			recent.Title = firstNonEmpty(recent.Title, p.Title)
			recent.ModuleKey = firstNonEmpty(recent.ModuleKey, p.ModuleKey)
			recent.ModuleName = firstNonEmpty(recent.ModuleName, p.ModuleName)
			recent.DocsVersion = firstNonEmpty(recent.DocsVersion, p.DocsVersion)
			recent.EntryKey = firstNonEmpty(recent.EntryKey, p.EntryKey)
			recent.Href = firstNonEmpty(recent.Href, "/docs/"+p.ModuleKey+"/"+p.DocsVersion+"/"+p.EntryKey)
			break
		}
	}
	if recent.ID == "" {
		recent.ID = s.nextIDLocked("recent")
	}
	recent.UserID = userID
	if recent.ViewedAt.IsZero() {
		recent.ViewedAt = time.Now().UTC()
	}
	filtered := s.recentDocs[:0]
	for _, item := range s.recentDocs {
		if !(item.UserID == userID && item.DocID == recent.DocID) {
			filtered = append(filtered, item)
		}
	}
	s.recentDocs = append(filtered, recent)
	return recent, nil
}

// CreateCategory adds a new category. Key is required and must be unique.
