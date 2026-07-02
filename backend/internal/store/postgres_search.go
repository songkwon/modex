package store

import (
	"context"
	"encoding/json"
	"strings"
)

// Pages reads the searchable document projection directly from PostgreSQL.
// It intentionally does not use PostgresRepository.load: request paths must
// not materialize an in-memory copy of the complete business store.
func (p *PostgresRepository) Pages() []Page {
	var pages []Page
	err := loadQuery(context.Background(), p.pool, `
		SELECT (to_jsonb(p)-'module_id'-'version_id'-'entry_id'-'release_id'-'last_verified_at'-'created_at') ||
			jsonb_build_object(
				'module_key',m.module_key,
				'module_name',m.name,
				'docs_version',v.docs_version,
				'package_version',v.package_version,
				'entry_key',COALESCE(e.entry_key,''),
				'entry_type',COALESCE(e.entry_type,'')
			)
		FROM docs_page p
		JOIN docs_module m ON m.id=p.module_id
		JOIN docs_version v ON v.id=p.version_id
		LEFT JOIN docs_entry e ON e.id=p.entry_id
		ORDER BY p.id`, &pages)
	if err != nil {
		return []Page{}
	}
	return pages
}

// Modules queries the catalog directly. Category membership is assembled by a
// correlated relational query rather than from a process-local snapshot.
func (p *PostgresRepository) Modules(categoryID, keyword string) []Module {
	var modules []Module
	args := []any{}
	where := []string{"1=1"}
	if categoryID != "" {
		args = append(args, categoryID)
		where = append(where, `EXISTS (SELECT 1 FROM docs_module_category f WHERE f.module_id=m.id AND f.category_id=$1)`)
	}
	if strings.TrimSpace(keyword) != "" {
		args = append(args, "%"+strings.TrimSpace(keyword)+"%")
		where = append(where, `(m.name ILIKE $`+itoa(len(args))+` OR m.module_key ILIKE $`+itoa(len(args))+` OR m.repo_url ILIKE $`+itoa(len(args))+`)`)
	}
	query := `SELECT (to_jsonb(m)-'default_version_id'-'created_at'-'deploy_token') ||
		jsonb_build_object(
			'category_ids',COALESCE((SELECT jsonb_agg(mc.category_id ORDER BY mc.is_primary DESC,mc.category_id) FROM docs_module_category mc WHERE mc.module_id=m.id),'[]'::jsonb),
			'category_path',COALESCE(NULLIF(m.category_path,''),(SELECT string_agg(c.name,' / ' ORDER BY mc.is_primary DESC,mc.category_id) FROM docs_module_category mc JOIN docs_category c ON c.id=mc.category_id WHERE mc.module_id=m.id),''),
			'created_by_name',COALESCE(NULLIF(u.display_name,''),NULLIF(u.username,''),''),
			'deploy_token_set',COALESCE(m.deploy_token,'')<>'',
			'reads_7d',CASE WHEN count(pv.id)=0 THEN m.reads_7d ELSE count(pv.id) FILTER(WHERE pv.viewed_at>now()-interval '7 days') END,
			'reads_30d',CASE WHEN count(pv.id)=0 THEN m.reads_30d ELSE count(pv.id) FILTER(WHERE pv.viewed_at>now()-interval '30 days') END
		)
		FROM docs_module m
		LEFT JOIN users u ON u.id=m.created_by
		LEFT JOIN docs_page_view pv ON pv.module_key=m.module_key
		WHERE ` + strings.Join(where, " AND ") + `
		GROUP BY m.id,u.id
		ORDER BY m.updated_at DESC,m.id`
	if err := loadQuery(context.Background(), p.pool, query, &modules, args...); err != nil {
		return []Module{}
	}
	for i := range modules {
		modules[i].AvailableVers = p.Versions(modules[i].ModuleKey)
	}
	return modules
}

func (p *PostgresRepository) Settings() Settings {
	var raw []byte
	if err := p.pool.QueryRow(context.Background(), `SELECT value_json FROM platform_settings WHERE key='main'`).Scan(&raw); err != nil {
		return Settings{}
	}
	var settings Settings
	if err := json.Unmarshal(raw, &settings); err != nil {
		return Settings{}
	}
	return settings
}

// Embeddings are owned by vectorstore.Postgres in production. These methods
// satisfy the fallback contract used by unit tests and non-vector deployments;
// a PostgreSQL application always configures the external vector store.
func (p *PostgresRepository) Embedding(string) ([]float32, bool) { return nil, false }
func (p *PostgresRepository) SetEmbedding(string, []float32)     {}
func (p *PostgresRepository) ClearEmbeddings() {
	_, _ = p.pool.Exec(context.Background(), `TRUNCATE docs_embedding`)
}
func (p *PostgresRepository) EmbeddingCount() int {
	var count int
	_ = p.pool.QueryRow(context.Background(), `SELECT count(*) FROM docs_embedding`).Scan(&count)
	return count
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var digits [20]byte
	i := len(digits)
	for value > 0 {
		i--
		digits[i] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[i:])
}
