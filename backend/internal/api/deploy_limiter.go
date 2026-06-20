package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// deployLimiter bounds how many artifacts are ingested concurrently. acquire
// blocks until a slot is free, the wait budget runs out, or the client
// disconnects; the returned release frees the slot and must be deferred by the
// caller. ok is false when no slot could be acquired.
type deployLimiter interface {
	acquire(ctx context.Context) (release func(), ok bool)
}

// newDeployLimiter returns a Redis-backed distributed limiter when a shared
// client is available so the bound is global across replicas; otherwise it
// returns a per-process limiter. A non-positive max disables limiting.
func newDeployLimiter(client *redis.Client) deployLimiter {
	max := envPositiveInt("DEPLOY_MAX_CONCURRENT", 2)
	if max <= 0 {
		return nil
	}
	if client != nil {
		log.Printf("deploy limiter: using shared Redis backend (global max %d)", max)
		return &redisDeployLimiter{client: client, key: "modex:deploysem", max: max}
	}
	return &localDeployLimiter{sem: make(chan struct{}, max)}
}

func deployWait() time.Duration {
	return time.Duration(envPositiveInt("DEPLOY_QUEUE_WAIT_SECONDS", 15)) * time.Second
}

// localDeployLimiter bounds concurrency within a single process via a buffered
// channel acting as a semaphore.
type localDeployLimiter struct {
	sem chan struct{}
}

func (l *localDeployLimiter) acquire(ctx context.Context) (func(), bool) {
	timer := time.NewTimer(deployWait())
	defer timer.Stop()
	select {
	case l.sem <- struct{}{}:
		return func() { <-l.sem }, true
	case <-ctx.Done():
		return nil, false
	case <-timer.C:
		return nil, false
	}
}

// redisDeployLimiter is a distributed semaphore backed by a Redis sorted set.
// Each holder is a unique member scored by its acquisition time. Acquisition
// first evicts holders older than the lease TTL, so a replica that crashes
// mid-deploy never holds a slot forever; release removes the member.
type redisDeployLimiter struct {
	client *redis.Client
	key    string
	max    int
}

// acquireScript atomically reclaims expired holders and, if there is room,
// admits the caller. Returns 1 when admitted, 0 when full.
//
//	KEYS[1] = sorted set key
//	ARGV[1] = now (unix ms)    ARGV[2] = lease TTL (ms)
//	ARGV[3] = max holders      ARGV[4] = caller's unique member token
var deployAcquireScript = redis.NewScript(`
local now = tonumber(ARGV[1])
local ttl = tonumber(ARGV[2])
local max = tonumber(ARGV[3])
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', now - ttl)
if redis.call('ZCARD', KEYS[1]) < max then
  redis.call('ZADD', KEYS[1], now, ARGV[4])
  redis.call('PEXPIRE', KEYS[1], ttl)
  return 1
end
return 0
`)

func (l *redisDeployLimiter) leaseTTL() time.Duration {
	// Must exceed the longest possible deploy so an in-flight holder is not
	// evicted while still working (which would over-admit). Defaults to 10m.
	return time.Duration(envPositiveInt("DEPLOY_SLOT_TTL_SECONDS", 600)) * time.Second
}

func (l *redisDeployLimiter) acquire(ctx context.Context) (func(), bool) {
	member := deployMemberToken()
	ttl := l.leaseTTL()
	deadline := time.Now().Add(deployWait())
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		if l.tryAcquire(ctx, member, ttl) {
			return func() { l.release(member) }, true
		}
		if time.Now().After(deadline) {
			return nil, false
		}
		select {
		case <-ctx.Done():
			return nil, false
		case <-ticker.C:
		}
	}
}

func (l *redisDeployLimiter) tryAcquire(ctx context.Context, member string, ttl time.Duration) bool {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	now := time.Now().UnixMilli()
	admitted, err := deployAcquireScript.Run(ctx, l.client, []string{l.key},
		now, ttl.Milliseconds(), l.max, member).Int64()
	if err != nil {
		// Fail open: Redis trouble should not block deploys entirely. The local
		// HTTP timeouts and per-instance request limits still cap load.
		log.Printf("deploy limiter: redis acquire failed (%v); admitting", err)
		return true
	}
	return admitted == 1
}

func (l *redisDeployLimiter) release(member string) {
	// Detached from the request context, which is already cancelled by the time
	// the deferred release runs.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := l.client.ZRem(ctx, l.key, member).Err(); err != nil {
		// The lease TTL reclaims the slot anyway; just record the slower path.
		log.Printf("deploy limiter: redis release failed (%v); slot reclaims on TTL", err)
	}
}

func deployMemberToken() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		// Fall back to a timestamp; collisions only risk a transient miscount.
		return "t-" + time.Now().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(buf)
}
