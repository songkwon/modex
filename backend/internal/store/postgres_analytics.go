package store

import (
	"context"
	"sort"
	"strings"
	"time"
)

func (p *PostgresRepository) AddSearchLog(log SearchLog) {
	if log.ID == "" {
		log.ID = databaseID("sl")
	}
	if log.SearchedAt.IsZero() {
		log.SearchedAt = time.Now().UTC()
	}
	_, _ = p.pool.Exec(context.Background(), `INSERT INTO docs_search_log(id,user_id,ip_address,query,mode,filters_json,result_count,clicked_doc_id,searched_at) VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::jsonb,$7,$8,$9)`, log.ID, log.UserID, log.IPAddress, log.Query, log.Mode, log.FiltersJSON, log.ResultCount, log.ClickedDocID, log.SearchedAt)
}
func (p *PostgresRepository) SearchLogs() []SearchLog {
	var logs []SearchLog
	if err := loadQuery(context.Background(), p.pool, `SELECT (to_jsonb(t)-'filters_json')||jsonb_build_object('filters_json',COALESCE(filters_json::text,'')) FROM docs_search_log t ORDER BY searched_at DESC,id`, &logs); err != nil {
		return []SearchLog{}
	}
	return logs
}
func (p *PostgresRepository) AddMCPLog(log MCPLog) {
	if log.ID == "" {
		log.ID = databaseID("ml")
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now().UTC()
	}
	_, _ = p.pool.Exec(context.Background(), `INSERT INTO docs_mcp_log(id,tool_name,user_id,query,input_json,result_count,created_at) VALUES($1,$2,$3,$4,NULLIF($5,'')::jsonb,$6,$7)`, log.ID, log.ToolName, log.UserID, log.Query, log.InputJSON, log.ResultCount, log.CreatedAt)
}
func (p *PostgresRepository) MCPLogs() []MCPLog {
	var logs []MCPLog
	if err := loadQuery(context.Background(), p.pool, `SELECT (to_jsonb(t)-'input_json')||jsonb_build_object('input_json',COALESCE(input_json::text,'')) FROM docs_mcp_log t ORDER BY created_at DESC,id`, &logs); err != nil {
		return []MCPLog{}
	}
	return logs
}

func (p *PostgresRepository) AddDocFeedback(feedback DocFeedback) DocFeedback {
	if feedback.ID == "" {
		feedback.ID = databaseID("df")
	}
	if feedback.CreatedAt.IsZero() {
		feedback.CreatedAt = time.Now().UTC()
	}
	if page, err := p.Page(feedback.DocID); err == nil {
		feedback.PageID = page.ID
		feedback.ModuleKey = page.ModuleKey
		feedback.Title = page.Title
	}
	saved, err := queryJSONOne[DocFeedback](context.Background(), p.pool, `INSERT INTO docs_feedback(id,doc_id,page_id,module_key,title,rating,comment,user_id,session_id,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING to_jsonb(docs_feedback)`, feedback.ID, feedback.DocID, feedback.PageID, feedback.ModuleKey, feedback.Title, feedback.Rating, feedback.Comment, feedback.UserID, feedback.SessionID, feedback.CreatedAt)
	if err != nil {
		return feedback
	}
	return saved
}
func (p *PostgresRepository) DocFeedbacks() []DocFeedback {
	var values []DocFeedback
	if err := loadQuery(context.Background(), p.pool, `SELECT to_jsonb(t) FROM docs_feedback t ORDER BY created_at DESC,id`, &values); err != nil {
		return []DocFeedback{}
	}
	return values
}

func (p *PostgresRepository) enrichPageView(view *PageView) {
	if page, err := p.Page(view.DocID); err == nil {
		view.PageID = page.ID
		view.ModuleKey = page.ModuleKey
		view.ModuleName = page.ModuleName
		view.DocsVersion = page.DocsVersion
		view.EntryKey = page.EntryKey
		view.Title = page.Title
		view.Path = page.Path
	}
}
func (p *PostgresRepository) RecordPageView(view PageView) PageView {
	if view.ID == "" {
		view.ID = databaseID("pv")
	}
	if view.ViewedAt.IsZero() {
		view.ViewedAt = time.Now().UTC()
	}
	p.enrichPageView(&view)
	moduleID, versionID := "", ""
	if view.ModuleKey != "" {
		if module, err := p.Module(view.ModuleKey); err == nil {
			moduleID = module.ID
			for _, version := range module.AvailableVers {
				if version.DocsVersion == view.DocsVersion {
					versionID = version.ID
					break
				}
			}
		}
	}
	saved, err := queryJSONOne[PageView](context.Background(), p.pool, `INSERT INTO docs_page_view(id,page_id,module_id,version_id,doc_id,module_key,module_name,docs_version,entry_key,title,path,user_id,session_id,read_id,duration_seconds,scroll_depth,viewed_at) VALUES($1,NULLIF($2,''),NULLIF($3,''),NULLIF($4,''),$5,$6,$7,$8,$9,$10,$11,NULLIF($12,''),$13,$14,$15,$16,$17) RETURNING (to_jsonb(docs_page_view)-'module_id'-'version_id')`, view.ID, view.PageID, moduleID, versionID, view.DocID, view.ModuleKey, view.ModuleName, view.DocsVersion, view.EntryKey, view.Title, view.Path, view.UserID, view.SessionID, view.ReadID, view.DurationSeconds, view.ScrollDepth, view.ViewedAt)
	if err != nil {
		return view
	}
	return saved
}
func (p *PostgresRepository) RecordReadProgress(docID, sessionID, readID string, durationSeconds int, scrollDepth float64) PageView {
	ctx := context.Background()
	condition := `doc_id=$1 AND (($2<>'' AND read_id=$2) OR ($2='' AND session_id=$3))`
	view, err := queryJSONOne[PageView](ctx, p.pool, `UPDATE docs_page_view SET duration_seconds=GREATEST(COALESCE(duration_seconds,0),$4),scroll_depth=GREATEST(COALESCE(scroll_depth,0),$5) WHERE id=(SELECT id FROM docs_page_view WHERE `+condition+` ORDER BY viewed_at DESC LIMIT 1) RETURNING (to_jsonb(docs_page_view)-'module_id'-'version_id')`, docID, readID, sessionID, durationSeconds, scrollDepth)
	if err == nil {
		return view
	}
	return p.RecordPageView(PageView{DocID: docID, SessionID: sessionID, ReadID: readID, DurationSeconds: durationSeconds, ScrollDepth: scrollDepth})
}

func (p *PostgresRepository) pageViews() []PageView {
	var values []PageView
	if err := loadQuery(context.Background(), p.pool, `SELECT (to_jsonb(t)-'module_id'-'version_id') FROM docs_page_view t ORDER BY viewed_at,id`, &values); err != nil {
		return []PageView{}
	}
	return values
}
func (p *PostgresRepository) PageAnalytics() []PageStat {
	var values []PageStat
	query := `SELECT jsonb_build_object(
		'doc_id',p.doc_id,'title',p.title,'module_key',m.module_key,'module_name',m.name,
		'docs_version',v.docs_version,'path',p.path,'pv',count(pv.id),
		'uv',count(DISTINCT COALESCE(NULLIF(pv.user_id,''),NULLIF(pv.session_id,''))),
		'reads_7d',CASE WHEN count(pv.id)=0 THEN m.reads_7d ELSE count(pv.id) FILTER(WHERE pv.viewed_at>now()-interval '7 days') END,
		'reads_30d',CASE WHEN count(pv.id)=0 THEN m.reads_30d ELSE count(pv.id) FILTER(WHERE pv.viewed_at>now()-interval '30 days') END,
		'avg_duration_seconds',COALESCE(avg(NULLIF(pv.duration_seconds,0))::int,0),
		'last_viewed_at',COALESCE(max(pv.viewed_at),p.updated_at)
	) FROM docs_page p JOIN docs_module m ON m.id=p.module_id JOIN docs_version v ON v.id=p.version_id LEFT JOIN docs_page_view pv ON pv.doc_id=p.doc_id GROUP BY p.id,m.id,v.id ORDER BY count(pv.id) DESC,m.reads_30d DESC`
	if err := loadQuery(context.Background(), p.pool, query, &values); err != nil {
		return []PageStat{}
	}
	return values
}
func (p *PostgresRepository) PageReadStats(docID string, days int) PageReadStats {
	if days <= 0 {
		days = 30
	}
	now := time.Now().UTC()
	today := now.Truncate(24 * time.Hour)
	daily := make([]DailyReadPoint, days)
	index := map[string]int{}
	for i := 0; i < days; i++ {
		date := today.AddDate(0, 0, -(days - 1 - i)).Format("2006-01-02")
		daily[i] = DailyReadPoint{Date: date}
		index[date] = i
	}
	type readerAggregate struct {
		userID          string
		count, duration int
		last            time.Time
	}
	readers := map[string]*readerAggregate{}
	total, totalDuration, timed := 0, 0, 0
	window := today.AddDate(0, 0, -(days - 1))
	for _, view := range p.pageViews() {
		if view.DocID != docID {
			continue
		}
		total++
		if view.DurationSeconds > 0 {
			totalDuration += view.DurationSeconds
			timed++
		}
		if !view.ViewedAt.Before(window) {
			if i, ok := index[view.ViewedAt.UTC().Format("2006-01-02")]; ok {
				daily[i].Count++
			}
		}
		key := view.UserID
		if key == "" {
			key = "session:" + view.SessionID
		}
		aggregate := readers[key]
		if aggregate == nil {
			aggregate = &readerAggregate{userID: view.UserID}
			readers[key] = aggregate
		}
		aggregate.count++
		aggregate.duration += view.DurationSeconds
		if view.ViewedAt.After(aggregate.last) {
			aggregate.last = view.ViewedAt
		}
	}
	result := PageReadStats{DocID: docID, Total: total, Daily: daily, Readers: []ReaderStat{}}
	if timed > 0 {
		result.AvgDurationSec = totalDuration / timed
	}
	users := map[string]User{}
	for _, user := range p.Users("") {
		users[user.ID] = user
	}
	for _, aggregate := range readers {
		name := "匿名"
		if aggregate.userID != "" {
			if user, ok := users[aggregate.userID]; ok {
				name = firstNonEmpty(user.DisplayName, user.Username)
			} else {
				name = aggregate.userID
			}
		}
		average := 0
		if aggregate.count > 0 {
			average = aggregate.duration / aggregate.count
		}
		result.Readers = append(result.Readers, ReaderStat{Reader: name, UserID: aggregate.userID, Count: aggregate.count, AvgDurationSec: average, LastReadAt: aggregate.last})
	}
	sort.Slice(result.Readers, func(i, j int) bool {
		if result.Readers[i].Count != result.Readers[j].Count {
			return result.Readers[i].Count > result.Readers[j].Count
		}
		return result.Readers[i].LastReadAt.After(result.Readers[j].LastReadAt)
	})
	return result
}

func (p *PostgresRepository) UserFavorites(userID string) []UserFavorite {
	var values []UserFavorite
	if err := loadQuery(context.Background(), p.pool, `SELECT to_jsonb(t) FROM user_favorite t WHERE user_id=$1 ORDER BY created_at DESC`, &values, userID); err != nil {
		return []UserFavorite{}
	}
	return values
}
func (p *PostgresRepository) SetUserFavorite(userID, moduleKey string, favorite bool) ([]UserFavorite, error) {
	userID = strings.TrimSpace(userID)
	moduleKey = strings.TrimSpace(moduleKey)
	if userID == "" || moduleKey == "" {
		return nil, ErrInvalid
	}
	if _, err := p.Module(moduleKey); err != nil {
		return nil, err
	}
	ctx := context.Background()
	if favorite {
		_, err := p.pool.Exec(ctx, `INSERT INTO user_favorite(id,user_id,module_key,created_at) VALUES($1,$2,$3,now()) ON CONFLICT(user_id,module_key) DO NOTHING`, databaseID("fav"), userID, moduleKey)
		if err != nil {
			return nil, err
		}
	} else {
		if _, err := p.pool.Exec(ctx, `DELETE FROM user_favorite WHERE user_id=$1 AND module_key=$2`, userID, moduleKey); err != nil {
			return nil, err
		}
	}
	return p.UserFavorites(userID), nil
}
func (p *PostgresRepository) UserRecentDocs(userID string, limit int) []UserRecentDoc {
	if limit <= 0 {
		limit = 30
	}
	var values []UserRecentDoc
	if err := loadQuery(context.Background(), p.pool, `SELECT to_jsonb(t) FROM user_recent_doc t WHERE user_id=$1 ORDER BY viewed_at DESC LIMIT $2`, &values, userID, limit); err != nil {
		return []UserRecentDoc{}
	}
	return values
}
func (p *PostgresRepository) RecordUserRecentDoc(userID string, recent UserRecentDoc) (UserRecentDoc, error) {
	userID = strings.TrimSpace(userID)
	recent.DocID = strings.TrimSpace(recent.DocID)
	if userID == "" || recent.DocID == "" {
		return UserRecentDoc{}, ErrInvalid
	}
	if page, err := p.Page(recent.DocID); err == nil {
		recent.Title = firstNonEmpty(recent.Title, page.Title)
		recent.ModuleKey = firstNonEmpty(recent.ModuleKey, page.ModuleKey)
		recent.ModuleName = firstNonEmpty(recent.ModuleName, page.ModuleName)
		recent.DocsVersion = firstNonEmpty(recent.DocsVersion, page.DocsVersion)
		recent.EntryKey = firstNonEmpty(recent.EntryKey, page.EntryKey)
		recent.Href = firstNonEmpty(recent.Href, "/docs/"+page.ModuleKey+"/"+page.DocsVersion+"/"+page.EntryKey)
	}
	if recent.ID == "" {
		recent.ID = databaseID("recent")
	}
	recent.UserID = userID
	if recent.ViewedAt.IsZero() {
		recent.ViewedAt = time.Now().UTC()
	}
	return queryJSONOne[UserRecentDoc](context.Background(), p.pool, `INSERT INTO user_recent_doc(id,user_id,doc_id,title,module_key,module_name,docs_version,entry_key,href,viewed_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT(user_id,doc_id) DO UPDATE SET title=EXCLUDED.title,module_key=EXCLUDED.module_key,module_name=EXCLUDED.module_name,docs_version=EXCLUDED.docs_version,entry_key=EXCLUDED.entry_key,href=EXCLUDED.href,viewed_at=EXCLUDED.viewed_at RETURNING to_jsonb(user_recent_doc)`, recent.ID, recent.UserID, recent.DocID, recent.Title, recent.ModuleKey, recent.ModuleName, recent.DocsVersion, recent.EntryKey, recent.Href, recent.ViewedAt)
}
