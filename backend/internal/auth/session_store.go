package auth

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var errSessionNotFound = errors.New("session not found")

type sessionStore interface {
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Get(ctx context.Context, key string, value any) error
	Delete(ctx context.Context, key string) error
	Ping(ctx context.Context) error
	Close() error
}

type memorySession struct {
	value     []byte
	expiresAt time.Time
}

type memorySessionStore struct {
	mu    sync.RWMutex
	items map[string]memorySession
}

func newMemorySessionStore() *memorySessionStore {
	return &memorySessionStore{items: map[string]memorySession{}}
}

func (s *memorySessionStore) Set(_ context.Context, key string, value any, ttl time.Duration) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.items[key] = memorySession{value: raw, expiresAt: time.Now().Add(ttl)}
	s.mu.Unlock()
	return nil
}

func (s *memorySessionStore) Get(_ context.Context, key string, value any) error {
	s.mu.RLock()
	item, ok := s.items[key]
	s.mu.RUnlock()
	if !ok || time.Now().After(item.expiresAt) {
		if ok {
			s.mu.Lock()
			delete(s.items, key)
			s.mu.Unlock()
		}
		return errSessionNotFound
	}
	return json.Unmarshal(item.value, value)
}

func (s *memorySessionStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	delete(s.items, key)
	s.mu.Unlock()
	return nil
}

func (s *memorySessionStore) Ping(context.Context) error { return nil }
func (s *memorySessionStore) Close() error               { return nil }

type postgresSessionStore struct {
	pool *pgxpool.Pool
}

func newPostgresSessionStore(databaseURL string) (sessionStore, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	if _, err = pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS auth_session (
		key TEXT PRIMARY KEY,
		value_json JSONB NOT NULL,
		expires_at TIMESTAMPTZ NOT NULL,
		updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`); err != nil {
		pool.Close()
		return nil, err
	}
	return &postgresSessionStore{pool: pool}, nil
}

func (s *postgresSessionStore) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO auth_session(key,value_json,expires_at,updated_at) VALUES($1,$2::jsonb,$3,now()) ON CONFLICT(key) DO UPDATE SET value_json=EXCLUDED.value_json,expires_at=EXCLUDED.expires_at,updated_at=now()`, key, string(raw), time.Now().UTC().Add(ttl))
	return err
}

func (s *postgresSessionStore) Get(ctx context.Context, key string, value any) error {
	var raw []byte
	if _, err := s.pool.Exec(ctx, `DELETE FROM auth_session WHERE key=$1 AND expires_at<=now()`, key); err != nil {
		return err
	}
	err := s.pool.QueryRow(ctx, `SELECT value_json FROM auth_session WHERE key=$1 AND expires_at>now()`, key).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return errSessionNotFound
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, value)
}

func (s *postgresSessionStore) Delete(ctx context.Context, key string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM auth_session WHERE key=$1`, key)
	return err
}

func (s *postgresSessionStore) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }
func (s *postgresSessionStore) Close() error                   { s.pool.Close(); return nil }

type redisSessionStore struct {
	client *redis.Client
}

func newRedisSessionStore(rawURL string) (sessionStore, error) {
	options, err := redis.ParseURL(rawURL)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(options)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	return &redisSessionStore{client: client}, nil
}

func (s *redisSessionStore) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, key, raw, ttl).Err()
}

func (s *redisSessionStore) Get(ctx context.Context, key string, value any) error {
	raw, err := s.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return errSessionNotFound
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, value)
}

func (s *redisSessionStore) Delete(ctx context.Context, key string) error {
	return s.client.Del(ctx, key).Err()
}

func (s *redisSessionStore) Ping(ctx context.Context) error { return s.client.Ping(ctx).Err() }
func (s *redisSessionStore) Close() error                   { return s.client.Close() }
