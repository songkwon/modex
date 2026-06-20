package store

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var relationalSchema string

// PostgresRepository is the production business store. Every method executes
// its read or write against PostgreSQL; it does not cache a process-local copy
// of application state.
type PostgresRepository struct {
	pool *pgxpool.Pool
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

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func loadQuery[T any](ctx context.Context, q queryer, query string, out *[]T, args ...any) error {
	rows, err := q.Query(ctx, query, args...)
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

func mustJSON(value any) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func timeText(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}
