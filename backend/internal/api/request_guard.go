package api

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type limitPolicy struct {
	requests int
	window   time.Duration
}

// rateLimiter is the abstraction the request guards depend on. The in-memory
// implementation is per-process; the Redis implementation shares counters
// across replicas so a horizontally scaled deployment enforces one global
// limit instead of (limit × replicas).
type rateLimiter interface {
	permit(ctx context.Context, key string, policy limitPolicy) bool
}

// newRateLimiter returns a Redis-backed limiter when a shared client is
// available, otherwise falls back to the in-memory limiter. Falling back keeps
// single-node and local deployments working without Redis.
func newRateLimiter(client *redis.Client) rateLimiter {
	if client != nil {
		log.Printf("rate limiter: using shared Redis backend")
		return &redisRateLimiter{client: client}
	}
	return newRequestLimiter()
}

type limitWindow struct {
	started time.Time
	count   int
}

type requestLimiter struct {
	mu      sync.Mutex
	windows map[string]limitWindow
	lastGC  time.Time
}

func newRequestLimiter() *requestLimiter {
	return &requestLimiter{windows: map[string]limitWindow{}, lastGC: time.Now()}
}

func (l *requestLimiter) permit(_ context.Context, key string, policy limitPolicy) bool {
	return l.allow(key, policy, time.Now())
}

func (l *requestLimiter) allow(key string, policy limitPolicy, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	window := l.windows[key]
	if window.started.IsZero() || now.Sub(window.started) >= policy.window {
		window = limitWindow{started: now}
	}
	if window.count >= policy.requests {
		return false
	}
	window.count++
	l.windows[key] = window
	if now.Sub(l.lastGC) > 10*time.Minute {
		for candidate, item := range l.windows {
			if now.Sub(item.started) > 2*time.Hour {
				delete(l.windows, candidate)
			}
		}
		l.lastGC = now
	}
	return true
}

// rateLimitScript implements an atomic fixed-window counter: the first request
// in a window sets the expiry, and every request returns the running count.
// Running INCR and PEXPIRE as one script avoids a leaked key if the process
// dies between the two commands.
var rateLimitScript = redis.NewScript(`
local current = redis.call('INCR', KEYS[1])
if current == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return current
`)

type redisRateLimiter struct {
	client *redis.Client
}

func (l *redisRateLimiter) permit(ctx context.Context, key string, policy limitPolicy) bool {
	// Cap the time the limiter can add to a request: a slow or unreachable Redis
	// must not become a latency tax on every guarded endpoint.
	ctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	count, err := rateLimitScript.Run(ctx, l.client, []string{"modex:rl:" + key}, policy.window.Milliseconds()).Int64()
	if err != nil {
		// Fail open: a rate limiter is best-effort protection, never a hard
		// dependency that can take the API down when Redis hiccups.
		return true
	}
	return count <= int64(policy.requests)
}

func (s *Server) requestGuards(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if policy, class := requestPolicy(r.URL.Path); policy.requests > 0 {
			key := clientIP(r) + ":" + class
			if !s.limiter.permit(r.Context(), key, policy) {
				w.Header().Set("Retry-After", strconv.Itoa(int(policy.window.Seconds())))
				writeError(w, http.StatusTooManyRequests, "rate_limited", "too many requests")
				return
			}
		}
		if r.Body != nil {
			limit := int64(envPositiveInt("HTTP_MAX_BODY_BYTES", 2<<20))
			if r.URL.Path == "/api/deploy" {
				limit = int64(envPositiveInt("DOCS_DEPLOY_MAX_BYTES", 512<<20))
			}
			r.Body = http.MaxBytesReader(w, r.Body, limit)
		}
		next.ServeHTTP(w, r)
	})
}

func requestPolicy(path string) (limitPolicy, string) {
	switch {
	case path == "/api/auth/login":
		return limitPolicy{requests: envPositiveInt("RATE_LIMIT_AUTH_PER_MINUTE", 10), window: time.Minute}, "auth"
	case path == "/oauth/token":
		return limitPolicy{requests: envPositiveInt("RATE_LIMIT_TOKEN_PER_MINUTE", 30), window: time.Minute}, "token"
	case path == "/api/search":
		return limitPolicy{requests: envPositiveInt("RATE_LIMIT_SEARCH_PER_MINUTE", 60), window: time.Minute}, "search"
	case path == "/api/ask":
		return limitPolicy{requests: envPositiveInt("RATE_LIMIT_AI_PER_MINUTE", 20), window: time.Minute}, "ai"
	case path == "/api/deploy":
		return limitPolicy{requests: envPositiveInt("RATE_LIMIT_DEPLOY_PER_MINUTE", 10), window: time.Minute}, "deploy"
	default:
		return limitPolicy{}, ""
	}
}

func clientIP(r *http.Request) string {
	if envBool("TRUST_PROXY_HEADERS", false) {
		if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
			return forwarded
		}
		if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
			return realIP
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func envPositiveInt(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes"
}
