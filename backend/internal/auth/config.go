package auth

import (
	"net/url"
	"os"
	"strings"

	"modex/backend/internal/config"
)

type Config struct {
	Mode             string
	AppBaseURL       string
	FrontendBaseURL  string
	IssuerURL        string
	AuthURL          string
	TokenURL         string
	UserInfoURL      string
	EndSessionURL    string
	ClientID         string
	ClientSecret     string
	RedirectURL      string
	Scopes           []string
	SessionCookie    string
	StateCookie      string
	CookieDomain     string
	CookieSameSite   string
	CookieSecure     bool
	CORSAllowOrigins []string
	SuperAdmins      []string

	// UserMapping controls which OIDC claims are used for core user identity fields.
	// Values are resolved from an optional config file + environment variable overrides.
	UserMapping config.UserMapping
}

func FromEnv() Config {
	keycloakBase := strings.TrimRight(os.Getenv("KEYCLOAK_BASE_URL"), "/")
	realm := os.Getenv("KEYCLOAK_REALM")
	issuer := strings.TrimRight(os.Getenv("OIDC_ISSUER_URL"), "/")
	if issuer == "" && keycloakBase != "" && realm != "" {
		issuer = keycloakBase + "/realms/" + realm
	}
	appBase := strings.TrimRight(env("APP_BASE_URL", "http://localhost:8671"), "/")
	mode := env("AUTH_MODE", "mock")
	if mode == "keycloak" {
		mode = "oidc"
	}
	cfg := Config{
		Mode:             mode,
		AppBaseURL:       appBase,
		FrontendBaseURL:  strings.TrimRight(env("FRONTEND_BASE_URL", "http://localhost:3456"), "/"),
		IssuerURL:        issuer,
		AuthURL:          os.Getenv("OIDC_AUTH_URL"),
		TokenURL:         os.Getenv("OIDC_TOKEN_URL"),
		UserInfoURL:      os.Getenv("OIDC_USERINFO_URL"),
		EndSessionURL:    os.Getenv("OIDC_END_SESSION_URL"),
		ClientID:         os.Getenv("OIDC_CLIENT_ID"),
		ClientSecret:     os.Getenv("OIDC_CLIENT_SECRET"),
		RedirectURL:      os.Getenv("OIDC_REDIRECT_URL"),
		Scopes:           splitList(env("OIDC_SCOPES", "openid profile email")),
		SessionCookie:    env("SESSION_COOKIE_NAME", "modex_session"),
		StateCookie:      env("OAUTH_STATE_COOKIE_NAME", "modex_oauth_state"),
		CookieDomain:     os.Getenv("COOKIE_DOMAIN"),
		CookieSameSite:   env("COOKIE_SAME_SITE", "lax"),
		CookieSecure:     env("COOKIE_SECURE", "false") == "true",
		CORSAllowOrigins: splitList(env("CORS_ALLOW_ORIGINS", "http://localhost:3456")),
		SuperAdmins:      splitList(os.Getenv("SUPER_ADMIN_USERS")),

		// UserMapping comes from a combination of (optional) config file + env overrides.
		// See internal/config for precedence rules and why some settings live in files
		// while connection secrets stay in the environment.
		UserMapping: resolveUserMapping(),
	}
	if cfg.AuthURL == "" && issuer != "" {
		cfg.AuthURL = issuer + "/protocol/openid-connect/auth"
	}
	if cfg.TokenURL == "" && issuer != "" {
		cfg.TokenURL = issuer + "/protocol/openid-connect/token"
	}
	if cfg.UserInfoURL == "" && issuer != "" {
		cfg.UserInfoURL = issuer + "/protocol/openid-connect/userinfo"
	}
	if cfg.EndSessionURL == "" && issuer != "" {
		cfg.EndSessionURL = issuer + "/protocol/openid-connect/logout"
	}
	if cfg.RedirectURL == "" {
		cfg.RedirectURL = appBase + "/api/auth/callback"
	}
	return cfg
}

func (c Config) LoginReady() bool {
	return c.Mode == "oidc" && c.AuthURL != "" && c.TokenURL != "" && c.UserInfoURL != "" && c.ClientID != ""
}

func (c Config) LoginURL(state string) string {
	u, _ := url.Parse(c.AuthURL)
	q := u.Query()
	q.Set("client_id", c.ClientID)
	q.Set("redirect_uri", c.RedirectURL)
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(c.Scopes, " "))
	q.Set("state", state)
	u.RawQuery = q.Encode()
	return u.String()
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitList(v string) []string {
	var out []string
	for _, item := range strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == ' ' }) {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

// resolveUserMapping loads the user attribute mapping with the following precedence:
//  1. Values from config file (if CONFIG_FILE or conventional locations exist)
//  2. OIDC_CLAIM_* environment variables (explicit overrides)
//  3. Reasonable defaults (company prefers email as unique id)
func resolveUserMapping() config.UserMapping {
	m := config.LoadUserMapping()

	if m.UniqueIDClaim == "" {
		m.UniqueIDClaim = "email"
	}
	if m.AvatarClaim == "" {
		m.AvatarClaim = "picture"
	}
	if m.DisplayNameClaim == "" {
		m.DisplayNameClaim = "name"
	}
	if m.SecondaryInfoClaim == "" {
		m.SecondaryInfoClaim = "department"
	}
	return m
}
