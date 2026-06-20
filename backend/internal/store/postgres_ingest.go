package store

import (
	"context"
	"strconv"
	"strings"
	"time"
)

func (p *PostgresRepository) IngestArtifact(artifact DeployArtifact) (DeployResult, error) {
	if strings.TrimSpace(artifact.ModuleKey) == "" || strings.TrimSpace(artifact.DocsVersion) == "" || len(artifact.Entries) == 0 || len(artifact.Documents) == 0 {
		return DeployResult{}, ErrInvalid
	}
	ctx := context.Background()
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return DeployResult{}, err
	}
	defer tx.Rollback(ctx)
	now := time.Now().UTC()
	moduleName := firstNonEmpty(artifact.ModuleName, artifact.ModuleKey)
	module, err := queryJSONOne[Module](ctx, tx, `SELECT (to_jsonb(m)-'default_version_id'-'created_at')||jsonb_build_object('category_ids',COALESCE((SELECT jsonb_agg(mc.category_id ORDER BY mc.is_primary DESC,mc.category_id) FROM docs_module_category mc WHERE mc.module_id=m.id),'[]'::jsonb)) FROM docs_module m WHERE lower(module_key)=lower($1) FOR UPDATE`, artifact.ModuleKey)
	if err == ErrNotFound {
		module = Module{ID: databaseID("m"), ModuleKey: artifact.ModuleKey, Name: moduleName, Description: artifact.Description, OwnerGroup: firstNonEmpty(firstString(artifact.Authors), "docs"), RepoType: firstNonEmpty(artifact.RepoType, "git"), RepoURL: artifact.RepoURL, SourceType: "gitlab", GitLabBranch: artifact.Branch, DefaultVersion: artifact.DocsVersion, Visibility: "internal", Status: "active", PackageName: artifact.ModuleKey, PackageVersion: artifact.PackageVersion, Channel: "docs", Edition: artifact.Edition, Keywords: cloneStrings(artifact.Keywords), Maintainers: cloneStrings(artifact.Authors), LastSyncedCommit: artifact.CommitSHA, LastSyncedAt: now}
		module, err = queryJSONOne[Module](ctx, tx, `INSERT INTO docs_module(id,module_key,name,description,owner_group,repo_type,repo_url,default_version,visibility,status,package_name,package_version,channel,edition,keywords,maintainers,source_type,gitlab_branch,last_synced_commit,last_synced_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15::jsonb,$16::jsonb,$17,$18,$19,$20,$20) RETURNING to_jsonb(docs_module)-'default_version_id'-'created_at'`, module.ID, module.ModuleKey, module.Name, module.Description, module.OwnerGroup, module.RepoType, module.RepoURL, module.DefaultVersion, module.Visibility, module.Status, module.PackageName, module.PackageVersion, module.Channel, module.Edition, mustJSON(module.Keywords), mustJSON(module.Maintainers), module.SourceType, module.GitLabBranch, module.LastSyncedCommit, now)
	} else if err == nil {
		module.Name = moduleName
		if artifact.Description != "" {
			module.Description = artifact.Description
		}
		module.DefaultVersion = artifact.DocsVersion
		if artifact.PackageVersion != "" {
			module.PackageVersion = artifact.PackageVersion
		}
		if artifact.Edition != "" {
			module.Edition = artifact.Edition
		}
		if len(artifact.Keywords) > 0 {
			module.Keywords = cloneStrings(artifact.Keywords)
		}
		if len(artifact.Authors) > 0 {
			module.Maintainers = cloneStrings(artifact.Authors)
			if module.OwnerGroup == "" {
				module.OwnerGroup = artifact.Authors[0]
			}
		}
		module.Status = firstNonEmpty(module.Status, "active")
		module.Visibility = firstNonEmpty(module.Visibility, "internal")
		if artifact.RepoURL != "" {
			module.RepoURL = artifact.RepoURL
		}
		if artifact.RepoType != "" {
			module.RepoType = artifact.RepoType
		}
		if artifact.Branch != "" {
			module.GitLabBranch = artifact.Branch
		}
		if artifact.CommitSHA != "" {
			module.LastSyncedCommit = artifact.CommitSHA
		}
		module.LastSyncedAt = now
		module, err = queryJSONOne[Module](ctx, tx, `UPDATE docs_module SET name=$2,description=$3,owner_group=$4,repo_type=$5,repo_url=$6,default_version=$7,visibility=$8,status=$9,package_version=$10,edition=$11,keywords=$12::jsonb,maintainers=$13::jsonb,gitlab_branch=$14,last_synced_commit=$15,last_synced_at=$16,updated_at=$16 WHERE id=$1 RETURNING to_jsonb(docs_module)-'default_version_id'-'created_at'`, module.ID, module.Name, module.Description, module.OwnerGroup, module.RepoType, module.RepoURL, module.DefaultVersion, module.Visibility, module.Status, module.PackageVersion, module.Edition, mustJSON(module.Keywords), mustJSON(module.Maintainers), module.GitLabBranch, module.LastSyncedCommit, now)
	}
	if err != nil {
		return DeployResult{}, err
	}
	var categoryIDs []string
	rows, err := tx.Query(ctx, `SELECT category_id FROM docs_module_category WHERE module_id=$1 ORDER BY is_primary DESC,category_id`, module.ID)
	if err != nil {
		return DeployResult{}, err
	}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return DeployResult{}, err
		}
		categoryIDs = append(categoryIDs, id)
	}
	rows.Close()
	module.CategoryIDs = categoryIDs
	var categoryPath string
	_ = tx.QueryRow(ctx, `SELECT COALESCE(string_agg(c.name,' / ' ORDER BY mc.is_primary DESC,c.sort_order,c.id),'') FROM docs_module_category mc JOIN docs_category c ON c.id=mc.category_id WHERE mc.module_id=$1`, module.ID).Scan(&categoryPath)
	module.CategoryPath = categoryPath
	if _, err = tx.Exec(ctx, `UPDATE docs_module SET category_path=$2 WHERE id=$1`, module.ID, categoryPath); err != nil {
		return DeployResult{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE docs_version SET is_default=false WHERE module_id=$1`, module.ID); err != nil {
		return DeployResult{}, err
	}
	version, err := queryJSONOne[Version](ctx, tx, `WITH changed AS (INSERT INTO docs_version(id,module_id,docs_version,display_name,version_type,is_default,status,package_version,channel,edition,support_status,created_at,updated_at) VALUES($1,$2,$3,$3,'release',true,'active',$4,$5,$6,'supported',$7,$7) ON CONFLICT(module_id,docs_version) DO UPDATE SET display_name=COALESCE(NULLIF(docs_version.display_name,''),EXCLUDED.display_name),version_type=COALESCE(NULLIF(docs_version.version_type,''),'release'),is_default=true,status='active',package_version=EXCLUDED.package_version,edition=EXCLUDED.edition,support_status=COALESCE(NULLIF(docs_version.support_status,''),'supported'),updated_at=EXCLUDED.updated_at RETURNING *) SELECT (to_jsonb(changed)-'module_id'-'updated_at')||jsonb_build_object('module_key',$8::text) FROM changed`, databaseID("v"), module.ID, artifact.DocsVersion, artifact.PackageVersion, firstNonEmpty(module.Channel, "docs"), artifact.Edition, now, module.ModuleKey)
	if err != nil {
		return DeployResult{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE docs_page_view SET page_id=NULL WHERE page_id IN(SELECT id FROM docs_page WHERE module_id=$1 AND version_id=$2)`, module.ID, version.ID); err != nil {
		return DeployResult{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE docs_embedding SET page_id=NULL,entry_id=NULL WHERE module_id=$1 AND version_id=$2`, module.ID, version.ID); err != nil {
		return DeployResult{}, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM docs_page WHERE module_id=$1 AND version_id=$2`, module.ID, version.ID); err != nil {
		return DeployResult{}, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM docs_entry WHERE module_id=$1 AND version_id=$2`, module.ID, version.ID); err != nil {
		return DeployResult{}, err
	}
	entryIDs := map[string]string{}
	for index, entry := range artifact.Entries {
		id := databaseID("e")
		entryIDs[entry.Key] = id
		if _, err = tx.Exec(ctx, `INSERT INTO docs_entry(id,module_id,version_id,entry_key,title,entry_type,builder,source,storage_uri,nav_uri,index_status,is_primary,sort_order,status,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$6,$7,$8,$9,'indexed',$10,$11,'active',$12,$12)`, id, module.ID, version.ID, entry.Key, entry.Title, firstNonEmpty(entry.Type, "markdown"), entry.Source, "minio://modex/modules/"+module.ModuleKey+"/"+artifact.DocsVersion+"/site/"+entry.Key, "minio://modex/modules/"+module.ModuleKey+"/"+artifact.DocsVersion+"/nav.json", index == 0, index+1, now); err != nil {
			return DeployResult{}, err
		}
	}
	for _, document := range artifact.Documents {
		entryKey := firstNonEmpty(document.EntryKey, entryKeyFromDocID(document.DocID))
		docID := firstNonEmpty(document.DocID, artifact.ModuleKey+":"+artifact.DocsVersion+":"+entryKey)
		entryType := firstNonEmpty(document.EntryType, entryTypeForEntry(artifact.Entries, entryKey))
		contentHTML := htmlForEntry(artifact.SiteHTML, entryKey)
		if _, err = tx.Exec(ctx, `INSERT INTO docs_page(id,module_id,version_id,entry_id,doc_id,title,description,path,source_file,doc_type,status,owner_group,tags,category_ids,content_text,content_html,content_md,updated_at,created_at) VALUES($1,$2,$3,NULLIF($4,''),$5,$6,$7,$8,$9,$10,$11,$12,$13::jsonb,$14::jsonb,$15,$16,$17,$18,$18)`, databaseID("p"), module.ID, version.ID, entryIDs[entryKey], docID, firstNonEmpty(document.Title, titleForEntry(artifact.Entries, entryKey)), document.Description, docPagePath(artifact.ModuleKey, artifact.DocsVersion, entryKey, entryType, document.Path), document.SourceFile, entryType, firstNonEmpty(document.Status, "active"), module.OwnerGroup, mustJSON(coalesceStrings(document.Keywords, artifact.Keywords)), mustJSON(categoryIDs), document.Content, contentHTML, document.ContentMD, now); err != nil {
			return DeployResult{}, err
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO docs_nav(module_key,docs_version,items_json,updated_at) VALUES($1,$2,$3::jsonb,$4) ON CONFLICT(module_key,docs_version) DO UPDATE SET items_json=EXCLUDED.items_json,updated_at=EXCLUDED.updated_at`, module.ModuleKey, artifact.DocsVersion, mustJSON(artifact.Nav), now); err != nil {
		return DeployResult{}, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM docs_site_file WHERE lower(module_key)=lower($1) AND docs_version=$2`, module.ModuleKey, artifact.DocsVersion); err != nil {
		return DeployResult{}, err
	}
	for name, content := range artifact.SiteFiles {
		entryKey, relativeName, ok := splitSiteFile(name)
		if !ok {
			continue
		}
		if _, err = tx.Exec(ctx, `INSERT INTO docs_site_file(module_key,docs_version,entry_key,name,content,content_type,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7)`, module.ModuleKey, artifact.DocsVersion, entryKey, relativeName, content, contentTypeForName(relativeName, content), now); err != nil {
			return DeployResult{}, err
		}
	}
	release := Release{ID: databaseID("r"), ReleaseID: "rel-" + strings.ToLower(artifact.ModuleKey) + "-" + strings.ToLower(artifact.DocsVersion) + "-" + strconv.FormatInt(now.UnixNano(), 36), ModuleKey: module.ModuleKey, DocsVersion: artifact.DocsVersion, CommitSHA: artifact.CommitSHA, Branch: artifact.Branch, Publisher: firstNonEmpty(firstString(artifact.Authors), "docsctl"), BuildSystem: "docsctl", ArtifactVersion: now.Format("20060102.150405"), PackageVersion: artifact.PackageVersion, StorageURI: "minio://modex/modules/" + module.ModuleKey + "/" + artifact.DocsVersion + "/docs-artifact.zip", Status: "published", PublishedAt: now, CreatedAt: now}
	if _, err = tx.Exec(ctx, `INSERT INTO docs_release(id,module_id,version_id,release_id,commit_sha,branch,publisher,build_system,artifact_version,package_version,storage_uri,status,published_at,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$13)`, release.ID, module.ID, version.ID, release.ReleaseID, release.CommitSHA, release.Branch, release.Publisher, release.BuildSystem, release.ArtifactVersion, release.PackageVersion, release.StorageURI, release.Status, now); err != nil {
		return DeployResult{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return DeployResult{}, err
	}
	return DeployResult{Release: release, PagesIndexed: len(artifact.Documents), EntriesIndexed: len(artifact.Entries), HTMLFiles: len(artifact.SiteHTML), SiteFiles: len(artifact.SiteFiles), BytesReceived: artifact.Bytes}, nil
}
