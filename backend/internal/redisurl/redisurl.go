package redisurl

import (
	"net"
	"net/url"
	"os"
	"strings"
)

// FromEnv returns REDIS_URL when explicitly set. When REDIS_URL is empty it
// builds a Redis URL from REDIS_HOST/REDIS_PORT/REDIS_DB/REDIS_USER/REDIS_PASSWORD.
// If REDIS_HOST is not set, Redis remains disabled and callers can fall back.
func FromEnv() string {
	if redisURL := strings.TrimSpace(os.Getenv("REDIS_URL")); redisURL != "" {
		return redisURL
	}
	host := strings.TrimSpace(os.Getenv("REDIS_HOST"))
	if host == "" {
		return ""
	}
	port := env("REDIS_PORT", "6379")
	db := strings.Trim(strings.TrimSpace(env("REDIS_DB", "0")), "/")
	user := strings.TrimSpace(os.Getenv("REDIS_USER"))
	password := os.Getenv("REDIS_PASSWORD")

	u := url.URL{
		Scheme: "redis",
		Host:   net.JoinHostPort(host, port),
		Path:   "/" + db,
	}
	switch {
	case user != "" && password != "":
		u.User = url.UserPassword(user, password)
	case user != "":
		u.User = url.User(user)
	case password != "":
		u.User = url.UserPassword("", password)
	}
	return u.String()
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
