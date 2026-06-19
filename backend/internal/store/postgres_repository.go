package store

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var relationalSchema string

// PostgresRepository persists each business entity in its own relational table.
// JSONB is limited to record-local arrays and configuration; IDs, relationships,
// and queryable business fields remain typed columns.
type PostgresRepository struct {
	pool   *pgxpool.Pool
	saveMu sync.Mutex
}

func OpenPostgresRepository(ctx context.Context, databaseURL string) (*PostgresRepository, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	for {
		if err := pool.Ping(ctx); err == nil {
			break
		} else {
			select {
			case <-ctx.Done():
				pool.Close()
				return nil, fmt.Errorf("connect PostgreSQL: %w", err)
			case <-time.After(500 * time.Millisecond):
			}
		}
	}
	if _, err := pool.Exec(ctx, relationalSchema); err != nil {
		pool.Close()
		return nil, fmt.Errorf("apply relational schema: %w", err)
	}
	return &PostgresRepository{pool: pool}, nil
}

func (p *PostgresRepository) Close() {
	if p != nil && p.pool != nil {
		p.pool.Close()
	}
}

// LoadOrMigrate loads the relational store. On the first run it imports the old
// database snapshot or filesystem JSON once, writes normalized rows, and drops
// the obsolete snapshot table.
func (p *PostgresRepository) LoadOrMigrate(ctx context.Context, legacyDataDir string) (*Store, bool, error) {
	st, found, err := p.load(ctx)
	if err != nil {
		return nil, false, err
	}
	if found {
		if err := p.dropLegacySnapshotTable(ctx); err != nil {
			return nil, false, err
		}
		return st, false, nil
	}
	st, err = p.loadLegacyDatabaseSnapshot(ctx)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, false, err
	}
	migrated := err == nil
	if errors.Is(err, ErrNotFound) && legacyDataDir != "" {
		st, err = LoadWithoutEmbeddings(filepath.Join(legacyDataDir, "modex-store.json"))
		if err == nil {
			migrated = true
		} else if !errors.Is(err, ErrNotFound) {
			return nil, false, err
		}
	}
	if st == nil {
		st = New()
	}
	if err := p.Save(ctx, st); err != nil {
		return nil, false, err
	}
	if err := p.dropLegacySnapshotTable(ctx); err != nil {
		return nil, false, err
	}
	return st, migrated, nil
}

func (p *PostgresRepository) dropLegacySnapshotTable(ctx context.Context) error {
	if _, err := p.pool.Exec(ctx, `DROP TABLE IF EXISTS modex_store_snapshot`); err != nil {
		return fmt.Errorf("drop obsolete snapshot table: %w", err)
	}
	return nil
}

func (p *PostgresRepository) loadLegacyDatabaseSnapshot(ctx context.Context) (*Store, error) {
	var exists bool
	if err := p.pool.QueryRow(ctx, `SELECT to_regclass('public.modex_store_snapshot') IS NOT NULL`).Scan(&exists); err != nil || !exists {
		return nil, ErrNotFound
	}
	var raw []byte
	if err := p.pool.QueryRow(ctx, `SELECT snapshot FROM modex_store_snapshot WHERE key = 'main'`).Scan(&raw); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var snap snapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		return nil, fmt.Errorf("decode legacy database snapshot: %w", err)
	}
	snap.HTML, snap.SiteFiles, snap.Embeddings = nil, nil, nil
	return storeFromSnapshot(snap), nil
}

func (p *PostgresRepository) Save(ctx context.Context, s *Store) error {
	p.saveMu.Lock()
	defer p.saveMu.Unlock()

	state := s.toRelationalState()
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Embeddings survive document replacement and are re-linked by doc_id below.
	if _, err = tx.Exec(ctx, `UPDATE docs_embedding SET page_id=NULL,module_id=NULL,version_id=NULL,entry_id=NULL`); err != nil {
		return err
	}
	for _, table := range []string{
		"oauth_grant", "connected_app", "docs_page_view", "docs_feedback",
		"user_recent_doc", "user_favorite", "docs_search_log", "docs_mcp_log", "docs_page", "docs_release",
		"docs_entry", "docs_version", "docs_module_category", "docs_module",
		"docs_category", "user_groups", "teams", "groups", "users",
		"docs_nav", "platform_settings", "store_metadata",
	} {
		if _, err = tx.Exec(ctx, "DELETE FROM "+table); err != nil {
			return fmt.Errorf("clear %s: %w", table, err)
		}
	}

	for _, v := range state.Users {
		_, err = tx.Exec(ctx, `INSERT INTO users
			(id,username,display_name,email,department,avatar,groups_json,roles_json,managed_categories_json,source,status,is_super_admin,mcp_token,last_login_at,created_at,updated_at)
			VALUES($1,$2,$3,$4,$5,$6,$7::jsonb,$8::jsonb,$9::jsonb,$10,$11,$12,$13,NULLIF($14,'')::timestamptz,NULLIF($15,'')::timestamptz,NULLIF($16,'')::timestamptz)`,
			v.ID, v.Username, v.DisplayName, v.Email, v.Department, v.Avatar, mustJSON(v.Groups), mustJSON(v.Roles), mustJSON(v.ManagedCategories), v.Source, v.Status, v.SuperAdmin, v.MCPToken, timeText(v.LastLoginAt), timeText(v.CreatedAt), timeText(v.UpdatedAt))
		if err != nil {
			return fmt.Errorf("insert user %s: %w", v.ID, err)
		}
	}
	for _, v := range state.Groups {
		_, err = tx.Exec(ctx, `INSERT INTO groups(id,group_key,name,source,created_at,updated_at) VALUES($1,$2,$3,$4,NULLIF($5,'')::timestamptz,NULLIF($6,'')::timestamptz)`, v.ID, v.GroupKey, v.Name, v.Source, timeText(v.CreatedAt), timeText(v.UpdatedAt))
		if err != nil {
			return fmt.Errorf("insert group %s: %w", v.ID, err)
		}
	}
	groupIDs := map[string]string{}
	for _, g := range state.Groups {
		groupIDs[g.GroupKey] = g.ID
	}
	for _, u := range state.Users {
		for _, key := range u.Groups {
			if groupID := groupIDs[key]; groupID != "" {
				if _, err = tx.Exec(ctx, `INSERT INTO user_groups(user_id,group_id) VALUES($1,$2)`, u.ID, groupID); err != nil {
					return err
				}
			}
		}
	}
	for _, v := range state.Teams {
		_, err = tx.Exec(ctx, `INSERT INTO teams(id,key,name,description,leaders,members,created_at,updated_at) VALUES($1,$2,$3,$4,$5::jsonb,$6::jsonb,NULLIF($7,'')::timestamptz,NULLIF($8,'')::timestamptz)`, v.ID, v.Key, v.Name, v.Description, mustJSON(v.Leaders), mustJSON(v.Members), timeText(v.CreatedAt), timeText(v.UpdatedAt))
		if err != nil {
			return fmt.Errorf("insert team %s: %w", v.ID, err)
		}
	}
	for _, v := range state.Categories {
		_, err = tx.Exec(ctx, `INSERT INTO docs_category(id,parent_id,key,name,description,icon,sort_order,status,responsible_team) VALUES($1,NULLIF($2,''),$3,$4,$5,$6,$7,$8,$9)`, v.ID, v.ParentID, v.Key, v.Name, v.Description, v.Icon, v.SortOrder, v.Status, v.ResponsibleTeam)
		if err != nil {
			return fmt.Errorf("insert category %s: %w", v.ID, err)
		}
	}
	moduleIDs := map[string]string{}
	for _, v := range state.Modules {
		moduleIDs[v.ModuleKey] = v.ID
		_, err = tx.Exec(ctx, `INSERT INTO docs_module(id,module_key,name,description,owner_group,repo_type,repo_url,default_version,visibility,status,package_name,package_version,channel,edition,keywords,maintainers,category_path,source_type,doc_type,mount,gitlab_branch,gitlab_path,deploy_token,last_synced_commit,last_synced_at,reads_7d,reads_30d,updated_at)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15::jsonb,$16::jsonb,$17,$18,$19,$20,$21,$22,$23,$24,NULLIF($25,'')::timestamptz,$26,$27,NULLIF($28,'')::timestamptz)`, v.ID, v.ModuleKey, v.Name, v.Description, v.OwnerGroup, v.RepoType, v.RepoURL, v.DefaultVersion, v.Visibility, v.Status, v.PackageName, v.PackageVersion, v.Channel, v.Edition, mustJSON(v.Keywords), mustJSON(v.Maintainers), v.CategoryPath, v.SourceType, v.DocType, v.Mount, v.GitLabBranch, v.GitLabPath, v.DeployToken, v.LastSyncedCommit, timeText(v.LastSyncedAt), v.Reads7d, v.Reads30d, timeText(v.UpdatedAt))
		if err != nil {
			return fmt.Errorf("insert module %s: %w", v.ID, err)
		}
		for i, categoryID := range v.CategoryIDs {
			if _, err = tx.Exec(ctx, `INSERT INTO docs_module_category(module_id,category_id,is_primary) VALUES($1,$2,$3)`, v.ID, categoryID, i == 0); err != nil {
				return err
			}
		}
	}
	versionIDs := map[string]string{}
	for _, v := range state.Versions {
		moduleID := moduleIDs[v.ModuleKey]
		versionIDs[v.ModuleKey+"\x00"+v.DocsVersion] = v.ID
		_, err = tx.Exec(ctx, `INSERT INTO docs_version(id,module_id,docs_version,display_name,version_type,is_default,status,source_branch,package_version,channel,edition,support_status,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,NULLIF($13,'')::timestamptz)`, v.ID, moduleID, v.DocsVersion, v.DisplayName, v.VersionType, v.IsDefault, v.Status, v.SourceBranch, v.PackageVersion, v.Channel, v.Edition, v.SupportStatus, timeText(v.CreatedAt))
		if err != nil {
			return fmt.Errorf("insert version %s: %w", v.ID, err)
		}
	}
	entryIDs := map[string]string{}
	for _, v := range state.Entries {
		moduleID := moduleIDs[v.ModuleKey]
		versionID := versionIDs[v.ModuleKey+"\x00"+v.DocsVersion]
		entryIDs[v.ModuleKey+"\x00"+v.DocsVersion+"\x00"+v.EntryKey] = v.ID
		_, err = tx.Exec(ctx, `INSERT INTO docs_entry(id,module_id,version_id,entry_key,title,entry_type,builder,source,storage_uri,nav_uri,index_status,is_primary,sort_order,status,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,NULLIF($15,'')::timestamptz)`, v.ID, moduleID, versionID, v.EntryKey, v.Title, v.EntryType, v.Builder, v.Source, v.StorageURI, v.NavURI, v.IndexStatus, v.IsPrimary, v.SortOrder, v.Status, timeText(v.CreatedAt))
		if err != nil {
			return fmt.Errorf("insert entry %s: %w", v.ID, err)
		}
	}
	for _, v := range state.Releases {
		moduleID := moduleIDs[v.ModuleKey]
		versionID := versionIDs[v.ModuleKey+"\x00"+v.DocsVersion]
		_, err = tx.Exec(ctx, `INSERT INTO docs_release(id,module_id,version_id,release_id,commit_sha,branch,publisher,pipeline_url,build_system,build_id,artifact_version,package_version,storage_uri,status,published_at,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,NULLIF($15,'')::timestamptz,NULLIF($16,'')::timestamptz)`, v.ID, moduleID, versionID, v.ReleaseID, v.CommitSHA, v.Branch, v.Publisher, v.PipelineURL, v.BuildSystem, v.BuildID, v.ArtifactVersion, v.PackageVersion, v.StorageURI, v.Status, timeText(v.PublishedAt), timeText(v.CreatedAt))
		if err != nil {
			return fmt.Errorf("insert release %s: %w", v.ID, err)
		}
	}
	for _, v := range state.Pages {
		moduleID := moduleIDs[v.ModuleKey]
		versionID := versionIDs[v.ModuleKey+"\x00"+v.DocsVersion]
		entryID := entryIDs[v.ModuleKey+"\x00"+v.DocsVersion+"\x00"+v.EntryKey]
		_, err = tx.Exec(ctx, `INSERT INTO docs_page(id,module_id,version_id,entry_id,doc_id,title,description,path,source_file,doc_type,status,owner_group,tags,category_ids,content_text,content_html,content_md,updated_at) VALUES($1,$2,$3,NULLIF($4,''),$5,$6,$7,$8,$9,$10,$11,$12,$13::jsonb,$14::jsonb,$15,$16,$17,NULLIF($18,'')::timestamptz)`, v.ID, moduleID, versionID, entryID, v.DocID, v.Title, v.Description, v.Path, v.SourceFile, v.DocType, v.Status, v.OwnerGroup, mustJSON(v.Tags), mustJSON(v.CategoryIDs), v.ContentText, v.ContentHTML, v.ContentMD, timeText(v.UpdatedAt))
		if err != nil {
			return fmt.Errorf("insert page %s: %w", v.ID, err)
		}
	}
	for _, v := range state.SearchLogs {
		_, err = tx.Exec(ctx, `INSERT INTO docs_search_log(id,user_id,ip_address,query,mode,filters_json,result_count,clicked_doc_id,searched_at) VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::jsonb,$7,$8,NULLIF($9,'')::timestamptz)`, v.ID, v.UserID, v.IPAddress, v.Query, v.Mode, v.FiltersJSON, v.ResultCount, v.ClickedDocID, timeText(v.SearchedAt))
		if err != nil {
			return err
		}
	}
	for _, v := range state.MCPLogs {
		_, err = tx.Exec(ctx, `INSERT INTO docs_mcp_log(id,tool_name,user_id,query,input_json,result_count,created_at) VALUES($1,$2,$3,$4,NULLIF($5,'')::jsonb,$6,NULLIF($7,'')::timestamptz)`, v.ID, v.ToolName, v.UserID, v.Query, v.InputJSON, v.ResultCount, timeText(v.CreatedAt))
		if err != nil {
			return err
		}
	}
	for _, v := range state.Feedbacks {
		_, err = tx.Exec(ctx, `INSERT INTO docs_feedback(id,doc_id,page_id,module_key,title,rating,comment,user_id,session_id,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,NULLIF($10,'')::timestamptz)`, v.ID, v.DocID, v.PageID, v.ModuleKey, v.Title, v.Rating, v.Comment, v.UserID, v.SessionID, timeText(v.CreatedAt))
		if err != nil {
			return err
		}
	}
	for _, v := range state.PageViews {
		_, err = tx.Exec(ctx, `INSERT INTO docs_page_view(id,page_id,module_id,version_id,doc_id,module_key,module_name,docs_version,entry_key,title,path,user_id,session_id,read_id,duration_seconds,scroll_depth,viewed_at) VALUES($1,NULLIF($2,''),NULLIF($3,''),NULLIF($4,''),$5,$6,$7,$8,$9,$10,$11,NULLIF($12,''),$13,$14,$15,$16,NULLIF($17,'')::timestamptz)`, v.ID, v.PageID, moduleIDs[v.ModuleKey], versionIDs[v.ModuleKey+"\x00"+v.DocsVersion], v.DocID, v.ModuleKey, v.ModuleName, v.DocsVersion, v.EntryKey, v.Title, v.Path, v.UserID, v.SessionID, v.ReadID, v.DurationSeconds, v.ScrollDepth, timeText(v.ViewedAt))
		if err != nil {
			return err
		}
	}
	for _, v := range state.Favorites {
		_, err = tx.Exec(ctx, `INSERT INTO user_favorite(id,user_id,module_key,created_at) VALUES($1,$2,$3,NULLIF($4,'')::timestamptz)`, v.ID, v.UserID, v.ModuleKey, timeText(v.CreatedAt))
		if err != nil {
			return err
		}
	}
	for _, v := range state.RecentDocs {
		_, err = tx.Exec(ctx, `INSERT INTO user_recent_doc(id,user_id,doc_id,title,module_key,module_name,docs_version,entry_key,href,viewed_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,NULLIF($10,'')::timestamptz)`, v.ID, v.UserID, v.DocID, v.Title, v.ModuleKey, v.ModuleName, v.DocsVersion, v.EntryKey, v.Href, timeText(v.ViewedAt))
		if err != nil {
			return err
		}
	}
	for _, v := range state.Apps {
		_, err = tx.Exec(ctx, `INSERT INTO connected_app(id,name,description,client_id,client_secret_hash,redirect_uris,scopes,trusted,enabled,created_by,last_used_at,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6::jsonb,$7::jsonb,$8,$9,NULLIF($10,''),NULLIF($11,'')::timestamptz,NULLIF($12,'')::timestamptz,NULLIF($13,'')::timestamptz)`, v.ID, v.Name, v.Description, v.ClientID, v.ClientSecretHash, mustJSON(v.RedirectURIs), mustJSON(v.Scopes), v.Trusted, v.Enabled, v.CreatedBy, timeText(v.LastUsedAt), timeText(v.CreatedAt), timeText(v.UpdatedAt))
		if err != nil {
			return err
		}
	}
	for _, v := range state.Grants {
		_, err = tx.Exec(ctx, `INSERT INTO oauth_grant(id,app_id,user_id,code_hash,access_token_hash,refresh_token_hash,redirect_uri,scopes,code_expires_at,access_expires_at,refresh_expires_at,revoked_at,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8::jsonb,NULLIF($9,'')::timestamptz,NULLIF($10,'')::timestamptz,NULLIF($11,'')::timestamptz,NULLIF($12,'')::timestamptz,NULLIF($13,'')::timestamptz,NULLIF($14,'')::timestamptz)`, v.ID, v.AppID, v.UserID, v.CodeHash, v.AccessTokenHash, v.RefreshTokenHash, v.RedirectURI, mustJSON(v.Scopes), timeText(v.CodeExpiresAt), timeText(v.AccessExpiresAt), timeText(v.RefreshExpiresAt), timeText(v.RevokedAt), timeText(v.CreatedAt), timeText(v.UpdatedAt))
		if err != nil {
			return err
		}
	}
	for key, items := range state.Navs {
		parts := strings.SplitN(key, "@", 2)
		if len(parts) != 2 {
			continue
		}
		_, err = tx.Exec(ctx, `INSERT INTO docs_nav(module_key,docs_version,items_json) VALUES($1,$2,$3::jsonb)`, parts[0], parts[1], mustJSON(items))
		if err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO platform_settings(key,value_json) VALUES('main',$1::jsonb)`, mustJSON(state.Settings))
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO store_metadata(key,value_bigint,value_text) VALUES('state',$1,$2)`, state.Seq, state.User.ID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE docs_embedding e SET page_id=p.id,module_id=p.module_id,version_id=p.version_id,entry_id=p.entry_id FROM docs_page p WHERE e.doc_id=p.doc_id`)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (p *PostgresRepository) load(ctx context.Context) (*Store, bool, error) {
	var count int
	if err := p.pool.QueryRow(ctx, `SELECT count(*) FROM store_metadata WHERE key='state'`).Scan(&count); err != nil || count == 0 {
		return nil, false, err
	}
	snap := snapshot{Version: 3, Navs: map[string][]NavItem{}}
	if err := loadQuery(ctx, p.pool, `SELECT (to_jsonb(t)-'groups_json'-'roles_json'-'managed_categories_json') || jsonb_build_object('groups',groups_json,'roles',roles_json,'managed_categories',managed_categories_json) FROM users t ORDER BY id`, &snap.Users); err != nil {
		return nil, false, err
	}
	rows, err := p.pool.Query(ctx, `SELECT id,mcp_token FROM users WHERE COALESCE(mcp_token,'')<>''`)
	if err != nil {
		return nil, false, err
	}
	userTokens := map[string]string{}
	for rows.Next() {
		var id, token string
		if err = rows.Scan(&id, &token); err != nil {
			rows.Close()
			return nil, false, err
		}
		userTokens[id] = token
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, false, err
	}
	for i := range snap.Users {
		snap.Users[i].MCPToken = userTokens[snap.Users[i].ID]
	}
	if err := loadQuery(ctx, p.pool, `SELECT to_jsonb(t) FROM groups t ORDER BY id`, &snap.Groups); err != nil {
		return nil, false, err
	}
	if err := loadQuery(ctx, p.pool, `SELECT to_jsonb(t) FROM connected_app t ORDER BY id`, &snap.Apps); err != nil {
		return nil, false, err
	}
	if err := loadQuery(ctx, p.pool, `SELECT to_jsonb(t) FROM oauth_grant t ORDER BY id`, &snap.Grants); err != nil {
		return nil, false, err
	}
	if err := loadQuery(ctx, p.pool, `SELECT to_jsonb(t)-'leader' FROM teams t ORDER BY id`, &snap.Teams); err != nil {
		return nil, false, err
	}
	if err := loadQuery(ctx, p.pool, `SELECT to_jsonb(t)-'created_at'-'updated_at' FROM docs_category t ORDER BY sort_order,id`, &snap.Categories); err != nil {
		return nil, false, err
	}
	if err := loadQuery(ctx, p.pool, `SELECT (to_jsonb(m)-'default_version_id'-'created_at'-'deploy_token') || jsonb_build_object('category_ids',COALESCE((SELECT jsonb_agg(mc.category_id ORDER BY mc.is_primary DESC,mc.category_id) FROM docs_module_category mc WHERE mc.module_id=m.id),'[]'::jsonb),'deploy_token_set',COALESCE(m.deploy_token,'')<>'') FROM docs_module m ORDER BY id`, &snap.Modules); err != nil {
		return nil, false, err
	}
	if err := loadQuery(ctx, p.pool, `SELECT (to_jsonb(v)-'module_id'-'updated_at') || jsonb_build_object('module_key',m.module_key) FROM docs_version v JOIN docs_module m ON m.id=v.module_id ORDER BY v.id`, &snap.Versions); err != nil {
		return nil, false, err
	}
	if err := loadQuery(ctx, p.pool, `SELECT (to_jsonb(e)-'module_id'-'version_id'-'updated_at') || jsonb_build_object('module_key',m.module_key,'docs_version',v.docs_version) FROM docs_entry e JOIN docs_module m ON m.id=e.module_id JOIN docs_version v ON v.id=e.version_id ORDER BY e.id`, &snap.Entries); err != nil {
		return nil, false, err
	}
	if err := loadQuery(ctx, p.pool, `SELECT (to_jsonb(r)-'module_id'-'version_id') || jsonb_build_object('module_key',m.module_key,'docs_version',v.docs_version) FROM docs_release r JOIN docs_module m ON m.id=r.module_id JOIN docs_version v ON v.id=r.version_id ORDER BY r.id`, &snap.Releases); err != nil {
		return nil, false, err
	}
	if err := loadQuery(ctx, p.pool, `SELECT (to_jsonb(p)-'module_id'-'version_id'-'entry_id'-'release_id'-'last_verified_at'-'created_at') || jsonb_build_object('module_key',m.module_key,'module_name',m.name,'docs_version',v.docs_version,'package_version',v.package_version,'entry_key',COALESCE(e.entry_key,''),'entry_type',COALESCE(e.entry_type,'')) FROM docs_page p JOIN docs_module m ON m.id=p.module_id JOIN docs_version v ON v.id=p.version_id LEFT JOIN docs_entry e ON e.id=p.entry_id ORDER BY p.id`, &snap.Pages); err != nil {
		return nil, false, err
	}
	if err := loadQuery(ctx, p.pool, `SELECT (to_jsonb(t)-'filters_json') || jsonb_build_object('filters_json',COALESCE(filters_json::text,'')) FROM docs_search_log t ORDER BY searched_at,id`, &snap.SearchLogs); err != nil {
		return nil, false, err
	}
	if err := loadQuery(ctx, p.pool, `SELECT (to_jsonb(t)-'input_json') || jsonb_build_object('input_json',COALESCE(input_json::text,'')) FROM docs_mcp_log t ORDER BY created_at,id`, &snap.MCPLogs); err != nil {
		return nil, false, err
	}
	if err := loadQuery(ctx, p.pool, `SELECT to_jsonb(t) FROM docs_feedback t ORDER BY created_at,id`, &snap.Feedbacks); err != nil {
		return nil, false, err
	}
	if err := loadQuery(ctx, p.pool, `SELECT (to_jsonb(pv)-'module_id'-'version_id') || jsonb_build_object('doc_id',COALESCE(NULLIF(pv.doc_id,''),p.doc_id,''),'module_key',COALESCE(NULLIF(pv.module_key,''),m.module_key,''),'module_name',COALESCE(NULLIF(pv.module_name,''),m.name,''),'docs_version',COALESCE(NULLIF(pv.docs_version,''),v.docs_version,''),'entry_key',COALESCE(NULLIF(pv.entry_key,''),e.entry_key,''),'title',COALESCE(NULLIF(pv.title,''),p.title,''),'path',COALESCE(NULLIF(pv.path,''),p.path,'')) FROM docs_page_view pv LEFT JOIN docs_page p ON p.id=pv.page_id LEFT JOIN docs_module m ON m.id=COALESCE(pv.module_id,p.module_id) LEFT JOIN docs_version v ON v.id=COALESCE(pv.version_id,p.version_id) LEFT JOIN docs_entry e ON e.id=p.entry_id ORDER BY viewed_at,pv.id`, &snap.PageViews); err != nil {
		return nil, false, err
	}
	if err := loadQuery(ctx, p.pool, `SELECT to_jsonb(t) FROM user_favorite t ORDER BY created_at,id`, &snap.Favorites); err != nil {
		return nil, false, err
	}
	if err := loadQuery(ctx, p.pool, `SELECT to_jsonb(t) FROM user_recent_doc t ORDER BY viewed_at,id`, &snap.RecentDocs); err != nil {
		return nil, false, err
	}
	rows, err = p.pool.Query(ctx, `SELECT module_key,deploy_token FROM docs_module WHERE COALESCE(deploy_token,'')<>''`)
	if err != nil {
		return nil, false, err
	}
	tokens := map[string]string{}
	for rows.Next() {
		var key, token string
		if err = rows.Scan(&key, &token); err != nil {
			rows.Close()
			return nil, false, err
		}
		tokens[key] = token
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, false, err
	}
	for i := range snap.Modules {
		snap.Modules[i].DeployToken = tokens[snap.Modules[i].ModuleKey]
		snap.Modules[i].DeployTokenSet = snap.Modules[i].DeployToken != ""
	}
	rows, err = p.pool.Query(ctx, `SELECT module_key,docs_version,items_json FROM docs_nav`)
	if err != nil {
		return nil, false, err
	}
	for rows.Next() {
		var m, v string
		var raw []byte
		if err = rows.Scan(&m, &v, &raw); err != nil {
			rows.Close()
			return nil, false, err
		}
		var items []NavItem
		if err = json.Unmarshal(raw, &items); err != nil {
			rows.Close()
			return nil, false, err
		}
		snap.Navs[m+"@"+v] = items
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, false, err
	}
	var settingsRaw []byte
	if err = p.pool.QueryRow(ctx, `SELECT value_json FROM platform_settings WHERE key='main'`).Scan(&settingsRaw); err == nil {
		if err = json.Unmarshal(settingsRaw, &snap.Settings); err != nil {
			return nil, false, err
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, err
	}
	var currentID string
	if err = p.pool.QueryRow(ctx, `SELECT value_bigint,COALESCE(value_text,'') FROM store_metadata WHERE key='state'`).Scan(&snap.Seq, &currentID); err != nil {
		return nil, false, err
	}
	for _, u := range snap.Users {
		if u.ID == currentID {
			snap.User = u
			break
		}
	}
	return storeFromSnapshot(snap), true, nil
}

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func loadQuery[T any](ctx context.Context, q queryer, query string, out *[]T) error {
	rows, err := q.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return err
		}
		var value T
		if err := json.Unmarshal(raw, &value); err != nil {
			return fmt.Errorf("decode relational record: %w", err)
		}
		*out = append(*out, value)
	}
	return rows.Err()
}

func mustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }
func timeText(v time.Time) string {
	if v.IsZero() {
		return ""
	}
	return v.UTC().Format(time.RFC3339Nano)
}
