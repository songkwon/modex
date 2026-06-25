package vectorstore

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"modex/backend/internal/embedding"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func Open(ctx context.Context, databaseURL string) (*Postgres, error) {
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
	if _, err := pool.Exec(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_docs_embedding_doc_id ON docs_embedding(doc_id)`); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ensure docs_embedding index: %w", err)
	}
	return &Postgres{pool: pool}, nil
}

func (p *Postgres) Close() {
	p.pool.Close()
}

func (p *Postgres) Existing(ctx context.Context, docIDs []string) (map[string]bool, error) {
	out := make(map[string]bool, len(docIDs))
	if len(docIDs) == 0 {
		return out, nil
	}
	rows, err := p.pool.Query(ctx, `SELECT doc_id FROM docs_embedding WHERE doc_id = ANY($1)`, docIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var docID string
		if err := rows.Scan(&docID); err != nil {
			return nil, err
		}
		out[docID] = true
	}
	return out, rows.Err()
}

func (p *Postgres) Similarities(ctx context.Context, query []float32, docIDs []string, limit int) (map[string]float64, error) {
	out := make(map[string]float64, len(docIDs))
	if len(query) == 0 || len(docIDs) == 0 || limit <= 0 {
		return out, nil
	}
	if len(query) != embedding.Dim {
		return nil, fmt.Errorf("query embedding dimension mismatch: got %d, want %d", len(query), embedding.Dim)
	}
	rows, err := p.pool.Query(ctx, `
		SELECT doc_id, 1 - ((embedding <=> $1::vector) / 2.0) AS score
		FROM docs_embedding
		WHERE doc_id = ANY($2) AND embedding IS NOT NULL
		ORDER BY embedding <=> $1::vector
		LIMIT $3`, vectorLiteral(query), docIDs, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var docID string
		var score float64
		if err := rows.Scan(&docID, &score); err != nil {
			return nil, err
		}
		out[docID] = score
	}
	return out, rows.Err()
}

func (p *Postgres) Upsert(ctx context.Context, docID string, vector []float32) error {
	if len(vector) != embedding.Dim {
		return fmt.Errorf("embedding dimension mismatch: model produced %d dims but the vector store requires %d; configure an embedding model with %d-dimensional output", len(vector), embedding.Dim, embedding.Dim)
	}
	raw, err := json.Marshal(vector)
	if err != nil {
		return err
	}
	vectorValue := vectorLiteral(vector)
	tag, err := p.pool.Exec(ctx, `
		INSERT INTO docs_embedding (
			page_id, doc_id, module_id, version_id, entry_id, content,
			embedding, embedding_json, metadata_json, updated_at
		)
		SELECT
			p.id, p.doc_id, p.module_id, p.version_id, p.entry_id,
			trim(concat_ws(E'\n', p.title, p.description, p.content_text)),
			$2::vector,
			$3::jsonb,
			jsonb_build_object(
				'title', p.title,
				'path', p.path,
				'source_file', p.source_file,
				'doc_type', p.doc_type
			),
			now()
		FROM docs_page p
		WHERE p.doc_id=$1
		ON CONFLICT (doc_id) DO UPDATE
		SET page_id = EXCLUDED.page_id,
		    module_id = EXCLUDED.module_id,
		    version_id = EXCLUDED.version_id,
		    entry_id = EXCLUDED.entry_id,
		    content = EXCLUDED.content,
		    embedding = EXCLUDED.embedding,
		    embedding_json = EXCLUDED.embedding_json,
		    metadata_json = EXCLUDED.metadata_json,
		    updated_at = now()`, docID, vectorValue, string(raw))
	if err != nil || tag.RowsAffected() > 0 {
		return err
	}
	_, err = p.pool.Exec(ctx, `
		INSERT INTO docs_embedding (doc_id, embedding, embedding_json, updated_at)
		VALUES ($1, $2::vector, $3::jsonb, now())
		ON CONFLICT (doc_id) DO UPDATE
		SET embedding = EXCLUDED.embedding,
		    embedding_json = EXCLUDED.embedding_json,
		    updated_at = now()`, docID, vectorValue, string(raw))
	return err
}

func (p *Postgres) Clear(ctx context.Context) error {
	_, err := p.pool.Exec(ctx, `DELETE FROM docs_embedding`)
	return err
}

func (p *Postgres) DeletePrefix(ctx context.Context, docIDPrefix string) error {
	_, err := p.pool.Exec(ctx, `DELETE FROM docs_embedding WHERE doc_id LIKE $1 ESCAPE E'\\'`, escapeLike(docIDPrefix)+"%")
	return err
}

func (p *Postgres) Count(ctx context.Context) (int, error) {
	var count int
	err := p.pool.QueryRow(ctx, `SELECT count(*) FROM docs_embedding`).Scan(&count)
	return count, err
}

func vectorLiteral(vector []float32) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, value := range vector {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(value), 'g', -1, 32))
	}
	b.WriteByte(']')
	return b.String()
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	return strings.ReplaceAll(value, `_`, `\_`)
}
