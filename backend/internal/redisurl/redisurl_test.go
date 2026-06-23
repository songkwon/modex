package redisurl

import "testing"

func TestFromEnvPrefersRedisURL(t *testing.T) {
	t.Setenv("REDIS_URL", "redis://custom:6380/2")
	t.Setenv("REDIS_HOST", "ignored")

	if got := FromEnv(); got != "redis://custom:6380/2" {
		t.Fatalf("FromEnv() = %q", got)
	}
}

func TestFromEnvBuildsFromRedisParts(t *testing.T) {
	t.Setenv("REDIS_HOST", "redis.internal")
	t.Setenv("REDIS_PORT", "6380")
	t.Setenv("REDIS_DB", "3")
	t.Setenv("REDIS_USER", "modex")
	t.Setenv("REDIS_PASSWORD", "p@ss/word")

	got := FromEnv()
	want := "redis://modex:p%40ss%2Fword@redis.internal:6380/3"
	if got != want {
		t.Fatalf("FromEnv() = %q, want %q", got, want)
	}
}

func TestFromEnvBuildsPasswordOnlyURL(t *testing.T) {
	t.Setenv("REDIS_HOST", "redis")
	t.Setenv("REDIS_PASSWORD", "secret")

	got := FromEnv()
	want := "redis://:secret@redis:6379/0"
	if got != want {
		t.Fatalf("FromEnv() = %q, want %q", got, want)
	}
}

func TestFromEnvDisabledWithoutHost(t *testing.T) {
	if got := FromEnv(); got != "" {
		t.Fatalf("FromEnv() = %q, want empty", got)
	}
}
