package auth

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

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
