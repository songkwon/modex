package dburl

import (
	"net"
	"net/url"
	"os"
	"strings"
)

// FromEnv returns DATABASE_URL when explicitly set; otherwise it builds a
// PostgreSQL URL from POSTGRES_* connection settings.
func FromEnv() string {
	if databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL")); databaseURL != "" {
		return databaseURL
	}
	host := env("POSTGRES_HOST", "postgres")
	port := env("POSTGRES_PORT", "5432")
	database := env("POSTGRES_DB", "modex")
	user := env("POSTGRES_USER", "modex")
	password := env("POSTGRES_PASSWORD", "modex")
	sslMode := env("POSTGRES_SSLMODE", "disable")

	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   net.JoinHostPort(host, port),
		Path:   "/" + database,
	}
	q := u.Query()
	q.Set("sslmode", sslMode)
	u.RawQuery = q.Encode()
	return u.String()
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
