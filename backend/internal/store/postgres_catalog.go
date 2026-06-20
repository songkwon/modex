package store

import (
	"context"
	"encoding/json"
	"errors"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (p *PostgresRepository) AllCategories() []Category {
	var categories []Category
	if err := loadQuery(context.Background(), p.pool, `SELECT to_jsonb(c)-'created_at'-'updated_at' FROM docs_category c ORDER BY sort_order,id`, &categories); err != nil {
		return []Category{}
	}
	return categories
}

func (p *PostgresRepository) CategoryName(id string) string {
	var name string
	if err := p.pool.QueryRow(context.Background(), `SELECT name FROM docs_category WHERE id=$1`, id).Scan(&name); err != nil {
		return id
	}
	return name
}

func (p *PostgresRepository) CategoryTree() []Category {
	byParent := map[string][]Category{}
	for _, category := range p.AllCategories() {
		category.Children = nil
		byParent[category.ParentID] = append(byParent[category.ParentID], category)
	}
	var attach func(string) []Category
	attach = func(parent string) []Category {
		nodes := byParent[parent]
		sort.SliceStable(nodes, func(i, j int) bool { return nodes[i].SortOrder < nodes[j].SortOrder })
		for i := range nodes {
			nodes[i].Children = attach(nodes[i].ID)
		}
		return nodes
	}
	result := attach("")
	if result == nil {
		return []Category{}
	}
	return result
}

func (p *PostgresRepository) createCategoryKey(ctx context.Context, name, parentID string) string {
	base := slugifyKey(name)
	if base == "" {
		base = "d" + strconv.FormatInt(time.Now().UnixNano()%1_000_000, 36)
	}
	prefix := ""
	_ = p.pool.QueryRow(ctx, `SELECT key||'.' FROM docs_category WHERE id=$1`, parentID).Scan(&prefix)
	for index := 1; ; index++ {
		key := prefix + base
		if index > 1 {
			key += "-" + strconv.Itoa(index)
		}
		var exists bool
		_ = p.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM docs_category WHERE key=$1 OR id=$1)`, key).Scan(&exists)
		if !exists {
			return key
		}
	}
}

func (p *PostgresRepository) CreateCategory(category Category) (Category, error) {
	ctx := context.Background()
	if strings.TrimSpace(category.Key) == "" {
		if strings.TrimSpace(category.Name) == "" {
			return Category{}, ErrInvalid
		}
		category.Key = p.createCategoryKey(ctx, category.Name, category.ParentID)
	}
	if category.ID == "" {
		category.ID = category.Key
	}
	if category.Status == "" {
		category.Status = "active"
	}
	created, err := queryJSONOne[Category](ctx, p.pool, `INSERT INTO docs_category(id,parent_id,key,name,description,icon,sort_order,status,responsible_team) VALUES($1,NULLIF($2,''),$3,$4,$5,$6,$7,$8,$9) RETURNING to_jsonb(docs_category)-'created_at'-'updated_at'`, category.ID, category.ParentID, category.Key, category.Name, category.Description, category.Icon, category.SortOrder, category.Status, category.ResponsibleTeam)
	return created, postgresError(err)
}

func (p *PostgresRepository) MoveCategory(id, parentID string, index int) (Category, error) {
	ctx := context.Background()
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return Category{}, err
	}
	defer tx.Rollback(ctx)
	category, err := queryJSONOne[Category](ctx, tx, `SELECT to_jsonb(c)-'created_at'-'updated_at' FROM docs_category c WHERE id=$1 FOR UPDATE`, id)
	if err != nil {
		return Category{}, err
	}
	if parentID == id {
		return Category{}, ErrInvalid
	}
	if parentID != "" {
		var valid bool
		if err = tx.QueryRow(ctx, `WITH RECURSIVE parents AS (SELECT id,parent_id FROM docs_category WHERE id=$1 UNION ALL SELECT c.id,c.parent_id FROM docs_category c JOIN parents p ON c.id=p.parent_id) SELECT EXISTS(SELECT 1 FROM docs_category WHERE id=$1) AND NOT EXISTS(SELECT 1 FROM parents WHERE id=$2)`, parentID, id).Scan(&valid); err != nil || !valid {
			return Category{}, ErrInvalid
		}
	}
	rows, err := tx.Query(ctx, `SELECT id FROM docs_category WHERE parent_id IS NOT DISTINCT FROM NULLIF($1,'') AND id<>$2 ORDER BY sort_order,id FOR UPDATE`, parentID, id)
	if err != nil {
		return Category{}, err
	}
	var siblings []string
	for rows.Next() {
		var sibling string
		if err = rows.Scan(&sibling); err != nil {
			rows.Close()
			return Category{}, err
		}
		siblings = append(siblings, sibling)
	}
	rows.Close()
	if index < 0 {
		index = 0
	}
	if index > len(siblings) {
		index = len(siblings)
	}
	siblings = append(siblings, "")
	copy(siblings[index+1:], siblings[index:])
	siblings[index] = id
	if _, err = tx.Exec(ctx, `UPDATE docs_category SET parent_id=NULLIF($2,'') WHERE id=$1`, id, parentID); err != nil {
		return Category{}, err
	}
	for position, sibling := range siblings {
		if _, err = tx.Exec(ctx, `UPDATE docs_category SET sort_order=$2 WHERE id=$1`, sibling, (position+1)*10); err != nil {
			return Category{}, err
		}
	}
	category.ParentID, category.SortOrder = parentID, (index+1)*10
	return category, tx.Commit(ctx)
}

func (p *PostgresRepository) UpdateCategory(id string, patch Category) (Category, error) {
	current, err := queryJSONOne[Category](context.Background(), p.pool, `SELECT to_jsonb(c)-'created_at'-'updated_at' FROM docs_category c WHERE id=$1`, id)
	if err != nil {
		return Category{}, err
	}
	if patch.Name != "" {
		current.Name = patch.Name
	}
	if patch.Description != "" {
		current.Description = patch.Description
	}
	if patch.Icon != "" {
		current.Icon = patch.Icon
	}
	if patch.SortOrder != 0 {
		current.SortOrder = patch.SortOrder
	}
	if patch.Status != "" {
		current.Status = patch.Status
	}
	if patch.ParentID != "" {
		current.ParentID = patch.ParentID
	}
	current.ResponsibleTeam = patch.ResponsibleTeam
	return queryJSONOne[Category](context.Background(), p.pool, `UPDATE docs_category SET parent_id=NULLIF($2,''),name=$3,description=$4,icon=$5,sort_order=$6,status=$7,responsible_team=$8 WHERE id=$1 RETURNING to_jsonb(docs_category)-'created_at'-'updated_at'`, id, current.ParentID, current.Name, current.Description, current.Icon, current.SortOrder, current.Status, current.ResponsibleTeam)
}

func (p *PostgresRepository) DeleteCategory(id string) error {
	command, err := p.pool.Exec(context.Background(), `DELETE FROM docs_category WHERE id=$1 AND NOT EXISTS(SELECT 1 FROM docs_category WHERE parent_id=$1)`, id)
	if err != nil {
		return postgresError(err)
	}
	if command.RowsAffected() == 0 {
		var exists bool
		_ = p.pool.QueryRow(context.Background(), `SELECT EXISTS(SELECT 1 FROM docs_category WHERE id=$1)`, id).Scan(&exists)
		if exists {
			return ErrConflict
		}
		return ErrNotFound
	}
	return nil
}

func (p *PostgresRepository) Module(moduleKey string) (Module, error) {
	var raw []byte
	var deployToken string
	err := p.pool.QueryRow(context.Background(), `SELECT (to_jsonb(m)-'default_version_id'-'created_at'-'deploy_token') || jsonb_build_object('category_ids',COALESCE((SELECT jsonb_agg(mc.category_id ORDER BY mc.is_primary DESC,mc.category_id) FROM docs_module_category mc WHERE mc.module_id=m.id),'[]'::jsonb),'deploy_token_set',COALESCE(m.deploy_token,'')<>''),COALESCE(m.deploy_token,'') FROM docs_module m WHERE lower(module_key)=lower($1)`, moduleKey).Scan(&raw, &deployToken)
	if errors.Is(err, pgx.ErrNoRows) {
		return Module{}, ErrNotFound
	}
	if err != nil {
		return Module{}, err
	}
	var module Module
	if err = json.Unmarshal(raw, &module); err != nil {
		return Module{}, err
	}
	module.DeployToken = deployToken
	module.AvailableVers = p.Versions(module.ModuleKey)
	return module, nil
}

func (p *PostgresRepository) uniqueModuleKey(name string) string {
	base := slugifyKey(name)
	if base == "" {
		base = "doc-" + strconv.FormatInt(time.Now().UnixNano()%1_000_000, 36)
	}
	for index := 1; ; index++ {
		key := base
		if index > 1 {
			key += "-" + strconv.Itoa(index)
		}
		var exists bool
		_ = p.pool.QueryRow(context.Background(), `SELECT EXISTS(SELECT 1 FROM docs_module WHERE lower(module_key)=lower($1))`, key).Scan(&exists)
		if !exists {
			return key
		}
	}
}

func (p *PostgresRepository) CreateModule(module Module) (Module, error) {
	if strings.TrimSpace(module.ModuleKey) == "" {
		if strings.TrimSpace(module.Name) == "" {
			return Module{}, ErrInvalid
		}
		module.ModuleKey = p.uniqueModuleKey(module.Name)
	}
	if module.ID == "" {
		module.ID = databaseID("m")
	}
	if module.DeployToken == "" {
		module.DeployToken = "mdx_" + strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	if module.Name == "" {
		module.Name = module.ModuleKey
	}
	if module.Status == "" {
		module.Status = "active"
	}
	if module.DefaultVersion == "" {
		module.DefaultVersion = "latest"
	}
	ctx := context.Background()
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return Module{}, err
	}
	defer tx.Rollback(ctx)
	created, err := queryJSONOne[Module](ctx, tx, `INSERT INTO docs_module(id,module_key,name,description,owner_group,repo_type,repo_url,default_version,visibility,status,package_name,package_version,channel,edition,keywords,maintainers,category_path,source_type,doc_type,mount,gitlab_branch,gitlab_path,deploy_token,last_synced_commit,last_synced_at,reads_7d,reads_30d,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15::jsonb,$16::jsonb,$17,$18,$19,$20,$21,$22,$23,$24,NULLIF($25,'')::timestamptz,$26,$27,now()) RETURNING to_jsonb(docs_module)-'default_version_id'-'created_at'`, module.ID, module.ModuleKey, module.Name, module.Description, module.OwnerGroup, module.RepoType, module.RepoURL, module.DefaultVersion, module.Visibility, module.Status, module.PackageName, module.PackageVersion, module.Channel, module.Edition, mustJSON(module.Keywords), mustJSON(module.Maintainers), module.CategoryPath, module.SourceType, module.DocType, module.Mount, module.GitLabBranch, module.GitLabPath, module.DeployToken, module.LastSyncedCommit, timeText(module.LastSyncedAt), module.Reads7d, module.Reads30d)
	if err != nil {
		return Module{}, postgresError(err)
	}
	if err = p.syncModuleCategories(ctx, tx, created.ID, module.CategoryIDs); err != nil {
		return Module{}, err
	}
	created.CategoryIDs = cloneStrings(module.CategoryIDs)
	created.DeployToken = module.DeployToken
	created.DeployTokenSet = created.DeployToken != ""
	return created, tx.Commit(ctx)
}

func (p *PostgresRepository) syncModuleCategories(ctx context.Context, tx interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}, moduleID string, ids []string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM docs_module_category WHERE module_id=$1`, moduleID); err != nil {
		return err
	}
	for index, id := range ids {
		if _, err := tx.Exec(ctx, `INSERT INTO docs_module_category(module_id,category_id,is_primary) VALUES($1,$2,$3)`, moduleID, id, index == 0); err != nil {
			return err
		}
	}
	return nil
}

func (p *PostgresRepository) UpdateModule(moduleKey string, patch Module) (Module, error) {
	current, err := p.Module(moduleKey)
	if err != nil {
		return Module{}, err
	}
	if patch.Name != "" {
		current.Name = patch.Name
	}
	if patch.Description != "" {
		current.Description = patch.Description
	}
	if patch.OwnerGroup != "" {
		current.OwnerGroup = patch.OwnerGroup
	}
	if patch.RepoType != "" {
		current.RepoType = patch.RepoType
	}
	if patch.RepoURL != "" {
		current.RepoURL = patch.RepoURL
	}
	if patch.DefaultVersion != "" {
		current.DefaultVersion = patch.DefaultVersion
	}
	if patch.Visibility != "" {
		current.Visibility = patch.Visibility
	}
	if patch.Status != "" {
		current.Status = patch.Status
	}
	if patch.PackageVersion != "" {
		current.PackageVersion = patch.PackageVersion
	}
	if patch.Channel != "" {
		current.Channel = patch.Channel
	}
	if patch.Edition != "" {
		current.Edition = patch.Edition
	}
	if patch.Keywords != nil {
		current.Keywords = patch.Keywords
	}
	if patch.Maintainers != nil {
		current.Maintainers = patch.Maintainers
	}
	if patch.CategoryIDs != nil {
		current.CategoryIDs = patch.CategoryIDs
	}
	if patch.CategoryPath != "" {
		current.CategoryPath = patch.CategoryPath
	}
	if patch.SourceType != "" {
		current.SourceType = patch.SourceType
	}
	if patch.DocType != "" {
		current.DocType = patch.DocType
	}
	if patch.Mount != "" {
		current.Mount = patch.Mount
	}
	if patch.GitLabBranch != "" {
		current.GitLabBranch = patch.GitLabBranch
	}
	if patch.GitLabPath != "" {
		current.GitLabPath = patch.GitLabPath
	}
	if patch.DeployToken != "" {
		current.DeployToken = patch.DeployToken
	}
	ctx := context.Background()
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return Module{}, err
	}
	defer tx.Rollback(ctx)
	updated, err := queryJSONOne[Module](ctx, tx, `UPDATE docs_module SET name=$2,description=$3,owner_group=$4,repo_type=$5,repo_url=$6,default_version=$7,visibility=$8,status=$9,package_version=$10,channel=$11,edition=$12,keywords=$13::jsonb,maintainers=$14::jsonb,category_path=$15,source_type=$16,doc_type=$17,mount=$18,gitlab_branch=$19,gitlab_path=$20,deploy_token=$21,updated_at=now() WHERE id=$1 RETURNING to_jsonb(docs_module)-'default_version_id'-'created_at'`, current.ID, current.Name, current.Description, current.OwnerGroup, current.RepoType, current.RepoURL, current.DefaultVersion, current.Visibility, current.Status, current.PackageVersion, current.Channel, current.Edition, mustJSON(current.Keywords), mustJSON(current.Maintainers), current.CategoryPath, current.SourceType, current.DocType, current.Mount, current.GitLabBranch, current.GitLabPath, current.DeployToken)
	if err != nil {
		return Module{}, err
	}
	if patch.CategoryIDs != nil {
		if err = p.syncModuleCategories(ctx, tx, current.ID, current.CategoryIDs); err != nil {
			return Module{}, err
		}
	}
	updated.CategoryIDs = current.CategoryIDs
	updated.AvailableVers = current.AvailableVers
	updated.DeployToken = current.DeployToken
	updated.DeployTokenSet = updated.DeployToken != ""
	return updated, tx.Commit(ctx)
}

func (p *PostgresRepository) Versions(moduleKey string) []Version {
	var values []Version
	if err := loadQuery(context.Background(), p.pool, `SELECT (to_jsonb(v)-'module_id'-'updated_at')||jsonb_build_object('module_key',m.module_key) FROM docs_version v JOIN docs_module m ON m.id=v.module_id WHERE lower(m.module_key)=lower($1) ORDER BY v.created_at`, &values, moduleKey); err != nil {
		return []Version{}
	}
	return values
}
func (p *PostgresRepository) Entries(moduleKey, docsVersion string) []Entry {
	var values []Entry
	if err := loadQuery(context.Background(), p.pool, `SELECT (to_jsonb(e)-'module_id'-'version_id'-'updated_at')||jsonb_build_object('module_key',m.module_key,'docs_version',v.docs_version) FROM docs_entry e JOIN docs_module m ON m.id=e.module_id JOIN docs_version v ON v.id=e.version_id WHERE lower(m.module_key)=lower($1) AND v.docs_version=$2 ORDER BY e.sort_order,e.id`, &values, moduleKey, docsVersion); err != nil {
		return []Entry{}
	}
	return values
}
func (p *PostgresRepository) EntryModuleKey(entryID string) (string, bool) {
	var key string
	err := p.pool.QueryRow(context.Background(), `SELECT m.module_key FROM docs_entry e JOIN docs_module m ON m.id=e.module_id WHERE e.id=$1`, entryID).Scan(&key)
	return key, err == nil
}

func (p *PostgresRepository) CreateVersion(moduleKey string, v Version) (Version, error) {
	if strings.TrimSpace(v.DocsVersion) == "" {
		return Version{}, ErrInvalid
	}
	m, err := p.Module(moduleKey)
	if err != nil {
		return Version{}, err
	}
	if v.ID == "" {
		v.ID = databaseID("v")
	}
	if v.DisplayName == "" {
		v.DisplayName = v.DocsVersion
	}
	if v.Status == "" {
		v.Status = "active"
	}
	ctx := context.Background()
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return Version{}, err
	}
	defer tx.Rollback(ctx)
	if v.IsDefault {
		_, err = tx.Exec(ctx, `UPDATE docs_version SET is_default=false WHERE module_id=$1`, m.ID)
		if err == nil {
			_, err = tx.Exec(ctx, `UPDATE docs_module SET default_version=$2 WHERE id=$1`, m.ID, v.DocsVersion)
		}
		if err != nil {
			return Version{}, err
		}
	}
	created, err := queryJSONOne[Version](ctx, tx, `WITH changed AS (INSERT INTO docs_version(id,module_id,docs_version,display_name,version_type,is_default,status,source_branch,package_version,channel,edition,support_status,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,now()) RETURNING *) SELECT (to_jsonb(changed)-'module_id'-'updated_at')||jsonb_build_object('module_key',$13::text) FROM changed`, v.ID, m.ID, v.DocsVersion, v.DisplayName, v.VersionType, v.IsDefault, v.Status, v.SourceBranch, v.PackageVersion, v.Channel, v.Edition, v.SupportStatus, m.ModuleKey)
	if err != nil {
		return Version{}, postgresError(err)
	}
	return created, tx.Commit(ctx)
}

func (p *PostgresRepository) UpdateVersion(moduleKey, docsVersion string, patch Version) (Version, error) {
	m, err := p.Module(moduleKey)
	if err != nil {
		return Version{}, err
	}
	var current Version
	for _, v := range m.AvailableVers {
		if v.DocsVersion == docsVersion {
			current = v
			break
		}
	}
	if current.ID == "" {
		return Version{}, ErrNotFound
	}
	if patch.DisplayName != "" {
		current.DisplayName = patch.DisplayName
	}
	if patch.VersionType != "" {
		current.VersionType = patch.VersionType
	}
	if patch.Status != "" {
		current.Status = patch.Status
	}
	if patch.SourceBranch != "" {
		current.SourceBranch = patch.SourceBranch
	}
	if patch.PackageVersion != "" {
		current.PackageVersion = patch.PackageVersion
	}
	if patch.Channel != "" {
		current.Channel = patch.Channel
	}
	if patch.Edition != "" {
		current.Edition = patch.Edition
	}
	if patch.SupportStatus != "" {
		current.SupportStatus = patch.SupportStatus
	}
	ctx := context.Background()
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return Version{}, err
	}
	defer tx.Rollback(ctx)
	if patch.IsDefault {
		if _, err = tx.Exec(ctx, `UPDATE docs_version SET is_default=false WHERE module_id=$1`, m.ID); err != nil {
			return Version{}, err
		}
		if _, err = tx.Exec(ctx, `UPDATE docs_module SET default_version=$2 WHERE id=$1`, m.ID, docsVersion); err != nil {
			return Version{}, err
		}
		current.IsDefault = true
	}
	updated, err := queryJSONOne[Version](ctx, tx, `WITH changed AS (UPDATE docs_version SET display_name=$2,version_type=$3,is_default=$4,status=$5,source_branch=$6,package_version=$7,channel=$8,edition=$9,support_status=$10 WHERE id=$1 RETURNING *) SELECT (to_jsonb(changed)-'module_id'-'updated_at')||jsonb_build_object('module_key',$11::text) FROM changed`, current.ID, current.DisplayName, current.VersionType, current.IsDefault, current.Status, current.SourceBranch, current.PackageVersion, current.Channel, current.Edition, current.SupportStatus, m.ModuleKey)
	if err != nil {
		return Version{}, err
	}
	return updated, tx.Commit(ctx)
}

func (p *PostgresRepository) CreateEntry(moduleKey, docsVersion string, e Entry) (Entry, error) {
	m, err := p.Module(moduleKey)
	if err != nil {
		return Entry{}, err
	}
	var versionID string
	if err = p.pool.QueryRow(context.Background(), `SELECT id FROM docs_version WHERE module_id=$1 AND docs_version=$2`, m.ID, docsVersion).Scan(&versionID); err != nil {
		return Entry{}, ErrNotFound
	}
	if strings.TrimSpace(e.EntryKey) == "" {
		return Entry{}, ErrInvalid
	}
	if e.ID == "" {
		e.ID = databaseID("e")
	}
	if e.EntryType == "" {
		e.EntryType = "markdown"
	}
	if e.Builder == "" {
		e.Builder = e.EntryType
	}
	if e.IndexStatus == "" {
		e.IndexStatus = "pending"
	}
	if e.Status == "" {
		e.Status = "active"
	}
	created, err := queryJSONOne[Entry](context.Background(), p.pool, `INSERT INTO docs_entry(id,module_id,version_id,entry_key,title,entry_type,builder,source,storage_uri,nav_uri,index_status,is_primary,sort_order,status,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,now()) RETURNING (to_jsonb(docs_entry)-'module_id'-'version_id'-'updated_at')||jsonb_build_object('module_key',$15::text,'docs_version',$16::text)`, e.ID, m.ID, versionID, e.EntryKey, e.Title, e.EntryType, e.Builder, e.Source, e.StorageURI, e.NavURI, e.IndexStatus, e.IsPrimary, e.SortOrder, e.Status, m.ModuleKey, docsVersion)
	return created, postgresError(err)
}

func (p *PostgresRepository) UpdateEntry(entryID string, patch Entry) (Entry, error) {
	key, ok := p.EntryModuleKey(entryID)
	if !ok {
		return Entry{}, ErrNotFound
	}
	var current Entry
	for _, version := range p.Versions(key) {
		for _, entry := range p.Entries(key, version.DocsVersion) {
			if entry.ID == entryID {
				current = entry
				break
			}
		}
	}
	if current.ID == "" {
		return Entry{}, ErrNotFound
	}
	if patch.Title != "" {
		current.Title = patch.Title
	}
	if patch.EntryType != "" {
		current.EntryType = patch.EntryType
	}
	if patch.Builder != "" {
		current.Builder = patch.Builder
	}
	if patch.Source != "" {
		current.Source = patch.Source
	}
	if patch.StorageURI != "" {
		current.StorageURI = patch.StorageURI
	}
	if patch.NavURI != "" {
		current.NavURI = patch.NavURI
	}
	if patch.IndexStatus != "" {
		current.IndexStatus = patch.IndexStatus
	}
	if patch.SortOrder != 0 {
		current.SortOrder = patch.SortOrder
	}
	if patch.Status != "" {
		current.Status = patch.Status
	}
	current.IsPrimary = patch.IsPrimary
	return queryJSONOne[Entry](context.Background(), p.pool, `UPDATE docs_entry SET title=$2,entry_type=$3,builder=$4,source=$5,storage_uri=$6,nav_uri=$7,index_status=$8,is_primary=$9,sort_order=$10,status=$11 WHERE id=$1 RETURNING (to_jsonb(docs_entry)-'module_id'-'version_id'-'updated_at')||jsonb_build_object('module_key',$12::text,'docs_version',$13::text)`, entryID, current.Title, current.EntryType, current.Builder, current.Source, current.StorageURI, current.NavURI, current.IndexStatus, current.IsPrimary, current.SortOrder, current.Status, current.ModuleKey, current.DocsVersion)
}
func (p *PostgresRepository) DeleteEntry(entryID string) error {
	command, err := p.pool.Exec(context.Background(), `DELETE FROM docs_entry WHERE id=$1`, entryID)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresRepository) Releases() []Release {
	var values []Release
	if err := loadQuery(context.Background(), p.pool, `SELECT (to_jsonb(r)-'module_id'-'version_id')||jsonb_build_object('module_key',m.module_key,'docs_version',v.docs_version) FROM docs_release r JOIN docs_module m ON m.id=r.module_id JOIN docs_version v ON v.id=r.version_id ORDER BY r.published_at DESC`, &values); err != nil {
		return []Release{}
	}
	return values
}
func (p *PostgresRepository) Release(releaseID string) (Release, error) {
	return queryJSONOne[Release](context.Background(), p.pool, `SELECT (to_jsonb(r)-'module_id'-'version_id')||jsonb_build_object('module_key',m.module_key,'docs_version',v.docs_version) FROM docs_release r JOIN docs_module m ON m.id=r.module_id JOIN docs_version v ON v.id=r.version_id WHERE r.release_id=$1 OR r.id=$1`, releaseID)
}
func (p *PostgresRepository) RollbackRelease(releaseID string) (Release, error) {
	return queryJSONOne[Release](context.Background(), p.pool, `WITH updated AS (UPDATE docs_release SET status='rolled_back' WHERE release_id=$1 OR id=$1 RETURNING *) SELECT (to_jsonb(r)-'module_id'-'version_id')||jsonb_build_object('module_key',m.module_key,'docs_version',v.docs_version) FROM updated r JOIN docs_module m ON m.id=r.module_id JOIN docs_version v ON v.id=r.version_id`, releaseID)
}

func (p *PostgresRepository) Page(docID string) (Page, error) {
	return queryJSONOne[Page](context.Background(), p.pool, `SELECT (to_jsonb(p)-'module_id'-'version_id'-'entry_id'-'release_id'-'last_verified_at'-'created_at')||jsonb_build_object('module_key',m.module_key,'module_name',m.name,'docs_version',v.docs_version,'package_version',v.package_version,'entry_key',COALESCE(e.entry_key,''),'entry_type',COALESCE(e.entry_type,'')) FROM docs_page p JOIN docs_module m ON m.id=p.module_id JOIN docs_version v ON v.id=p.version_id LEFT JOIN docs_entry e ON e.id=p.entry_id WHERE p.doc_id=$1`, docID)
}
func (p *PostgresRepository) PageByRoute(moduleKey, docsVersion, entryKey string) (Page, error) {
	return queryJSONOne[Page](context.Background(), p.pool, `SELECT (to_jsonb(p)-'module_id'-'version_id'-'entry_id'-'release_id'-'last_verified_at'-'created_at')||jsonb_build_object('module_key',m.module_key,'module_name',m.name,'docs_version',v.docs_version,'package_version',v.package_version,'entry_key',COALESCE(e.entry_key,''),'entry_type',COALESCE(e.entry_type,'')) FROM docs_page p JOIN docs_module m ON m.id=p.module_id JOIN docs_version v ON v.id=p.version_id LEFT JOIN docs_entry e ON e.id=p.entry_id WHERE lower(m.module_key)=lower($1) AND v.docs_version=$2 ORDER BY CASE WHEN e.entry_key=$3 THEN 0 WHEN e.is_primary THEN 1 ELSE 2 END,e.sort_order,p.id LIMIT 1`, moduleKey, docsVersion, entryKey)
}
func (p *PostgresRepository) Nav(moduleKey, docsVersion string) []NavItem {
	var raw []byte
	if err := p.pool.QueryRow(context.Background(), `SELECT items_json FROM docs_nav WHERE lower(module_key)=lower($1) AND docs_version=$2`, moduleKey, docsVersion).Scan(&raw); err != nil {
		return []NavItem{}
	}
	var nav []NavItem
	if json.Unmarshal(raw, &nav) != nil {
		return []NavItem{}
	}
	return nav
}
func (p *PostgresRepository) PageHTML(moduleKey, docsVersion, entryKey string) string {
	page, err := p.PageByRoute(moduleKey, docsVersion, entryKey)
	if err != nil {
		return ""
	}
	return page.ContentHTML
}
func (p *PostgresRepository) SiteFile(moduleKey, docsVersion, entryKey, name string) (SiteFile, error) {
	if name == "" {
		name = "index.html"
	}
	var file SiteFile
	err := p.pool.QueryRow(context.Background(), `SELECT name,content,content_type FROM docs_site_file WHERE lower(module_key)=lower($1) AND docs_version=$2 AND entry_key=$3 AND name=$4`, moduleKey, docsVersion, entryKey, path.Clean(strings.TrimPrefix(name, "/"))).Scan(&file.Name, &file.Content, &file.ContentType)
	if errors.Is(err, pgx.ErrNoRows) {
		return SiteFile{}, ErrNotFound
	}
	return file, err
}
func (p *PostgresRepository) ClearSiteAssets()                 {}
func (p *PostgresRepository) SiteObjects() map[string]SiteFile { return map[string]SiteFile{} }
