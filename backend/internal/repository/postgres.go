package repository

import (
	"context"

	"modex/backend/internal/store"
)

// PostgresRepository is the formal-table business repository used in
// production. It is exposed from this package so application assembly does not
// depend on store implementation details directly.
type PostgresRepository = store.PostgresRepository

func OpenPostgres(ctx context.Context, databaseURL string) (*PostgresRepository, error) {
	return store.OpenPostgresRepository(ctx, databaseURL)
}
