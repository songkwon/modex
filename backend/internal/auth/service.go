package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"modex/backend/internal/store"
)

type Service struct {
	cfg      Config
	client   *http.Client
	mu       sync.RWMutex
	sessions map[string]store.User
}

func NewService(cfg Config) *Service {
	return &Service{
		cfg:      cfg,
		client:   &http.Client{Timeout: 20 * time.Second},
		sessions: map[string]store.User{},
	}
}

func (s *Service) Config() Config {
	return s.cfg
}

// IsSuperAdmin reports whether the user matches a SUPER_ADMIN_USERS entry
// (matched against username or email, case-insensitive).
func (s *Service) IsSuperAdmin(user store.User) bool {
	for _, id := range s.cfg.SuperAdmins {
		if id == "" {
			continue
		}
		if strings.EqualFold(id, user.Username) || strings.EqualFold(id, user.Email) {
			return true
		}
	}
	return false
}

// applySuperAdmin grants the admin role to a configured super admin so the role
// travels with the session and the persisted user record.
func (s *Service) applySuperAdmin(user store.User) store.User {
	if !s.IsSuperAdmin(user) {
		return user
	}
	for _, r := range user.Roles {
		if r == "admin" {
			return user
		}
	}
	user.Roles = append(user.Roles, "admin")
	return user
}

func (s *Service) BeginLogin(w http.ResponseWriter) (string, error) {
	if !s.cfg.LoginReady() {
		return "", errors.New("OIDC login is not configured")
	}
	state, err := randomToken(32)
	if err != nil {
		return "", err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     s.cfg.StateCookie,
		Value:    state,
		Domain:   s.cfg.CookieDomain,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		SameSite: sameSiteMode(s.cfg.CookieSameSite),
		Secure:   s.cfg.CookieSecure,
	})
	return s.cfg.LoginURL(state), nil
}

func (s *Service) CompleteLogin(ctx context.Context, r *http.Request, w http.ResponseWriter) (store.User, error) {
	if providerErr := r.URL.Query().Get("error"); providerErr != "" {
		desc := r.URL.Query().Get("error_description")
		if desc == "" {
			desc = providerErr
		}
		return store.User{}, fmt.Errorf("identity provider returned error %q: %s", providerErr, desc)
	}
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		return store.User{}, errors.New("missing OAuth2 code or state")
	}
	stateCookie, err := r.Cookie(s.cfg.StateCookie)
	if err != nil || stateCookie.Value != state {
		return store.User{}, errors.New("invalid OAuth2 state")
	}
	token, err := s.exchangeCode(ctx, code)
	if err != nil {
		return store.User{}, err
	}
	user, err := s.fetchUserInfo(ctx, token.AccessToken, token.IDToken)
	if err != nil {
		return store.User{}, err
	}
	if err := s.CreateSession(w, user); err != nil {
		return store.User{}, err
	}
	http.SetCookie(w, &http.Cookie{Name: s.cfg.StateCookie, Value: "", Domain: s.cfg.CookieDomain, Path: "/", MaxAge: -1, HttpOnly: true, SameSite: sameSiteMode(s.cfg.CookieSameSite), Secure: s.cfg.CookieSecure})
	return user, nil
}

// CreateSession issues a session cookie for the given user. It is shared by the
// OIDC callback and the local mock-login endpoint so both paths produce a real,
// cookie-backed session.
func (s *Service) CreateSession(w http.ResponseWriter, user store.User) error {
	user = s.applySuperAdmin(user)
	sessionID, err := randomToken(32)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.sessions[sessionID] = user
	s.mu.Unlock()
	http.SetCookie(w, &http.Cookie{
		Name:     s.cfg.SessionCookie,
		Value:    sessionID,
		Domain:   s.cfg.CookieDomain,
		Path:     "/",
		MaxAge:   int((8 * time.Hour).Seconds()),
		HttpOnly: true,
		SameSite: sameSiteMode(s.cfg.CookieSameSite),
		Secure:   s.cfg.CookieSecure,
	})
	return nil
}

func (s *Service) CurrentUser(r *http.Request) (store.User, bool) {
	cookie, err := r.Cookie(s.cfg.SessionCookie)
	if err != nil || cookie.Value == "" {
		return store.User{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.sessions[cookie.Value]
	return user, ok
}

func (s *Service) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(s.cfg.SessionCookie); err == nil {
		s.mu.Lock()
		delete(s.sessions, cookie.Value)
		s.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: s.cfg.SessionCookie, Value: "", Domain: s.cfg.CookieDomain, Path: "/", MaxAge: -1, HttpOnly: true, SameSite: sameSiteMode(s.cfg.CookieSameSite), Secure: s.cfg.CookieSecure})
}

func (s *Service) exchangeCode(ctx context.Context, code string) (tokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", s.cfg.ClientID)
	form.Set("code", code)
	form.Set("redirect_uri", s.cfg.RedirectURL)
	if s.cfg.ClientSecret != "" {
		form.Set("client_secret", s.cfg.ClientSecret)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return tokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.client.Do(req)
	if err != nil {
		return tokenResponse{}, err
	}
	defer resp.Body.Close()
	var token tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return tokenResponse{}, err
	}
	if resp.StatusCode >= 300 {
		return tokenResponse{}, fmt.Errorf("OIDC token endpoint returned %d: %s", resp.StatusCode, token.Error)
	}
	if token.AccessToken == "" {
		return tokenResponse{}, errors.New("OIDC token endpoint returned empty access_token")
	}
	return token, nil
}

func (s *Service) fetchUserInfo(ctx context.Context, accessToken, idToken string) (store.User, error) {
	// Collect claims from both ID token (often has custom mappers like wxPhotoURL) and userinfo endpoint.
	claims := map[string]any{}

	if idToken != "" {
		if idClaims, err := parseJWTClaims(idToken); err == nil {
			mergeInto(claims, idClaims)
		}
	}

	if s.cfg.UserInfoURL != "" && accessToken != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.UserInfoURL, nil)
		if err != nil {
			return store.User{}, err
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)
		resp, err := s.client.Do(req)
		if err != nil {
			return store.User{}, err
		}
		defer resp.Body.Close()
		var ui map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&ui); err != nil {
			return store.User{}, err
		}
		if resp.StatusCode >= 300 {
			return store.User{}, fmt.Errorf("OIDC userinfo endpoint returned %d", resp.StatusCode)
		}
		mergeInto(claims, ui)
	}

	m := s.cfg.UserMapping

	// Unique identifier for this user (company uses email as the key).
	uniqueID := pickClaim(claims, m.UniqueIDClaim, "sub", "email", "preferred_username")

	// Stable internal ID: prefer the configured unique claim value; fall back to sub.
	userID := uniqueID
	if userID == "" {
		userID = getString(claims, "sub")
	}

	// Username for login/display fallback.
	username := pickClaim(claims, "preferred_username", m.UniqueIDClaim, "email", "sub")
	if username == "" {
		username = userID
	}

	// Primary display name (1级显示的名字).
	displayName := pickClaim(claims, m.DisplayNameClaim, "name", "given_name", "preferred_username", m.UniqueIDClaim, "email")

	// Secondary info line under the name (名字下方的信息字段), currently surfaced as Department.
	secondary := pickClaim(claims, m.SecondaryInfoClaim, "department", "org", "title", "organization")

	email := getString(claims, "email")
	if email == "" && strings.EqualFold(m.UniqueIDClaim, "email") {
		email = uniqueID
	}

	avatar := pickClaim(claims, m.AvatarClaim, "picture", "avatar", "photo", "wxPhotoURL")

	groups := extractStringSlice(claims, "groups")
	roles := extractStringSlice(claims, "roles")

	return store.User{
		ID:          userID,
		Username:    username,
		DisplayName: displayName,
		Email:       email,
		Department:  secondary,
		Avatar:      avatar,
		Groups:      groups,
		Roles:       roles,
	}, nil
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Error       string `json:"error"`
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func first(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// parseJWTClaims extracts the payload claims from a JWT without signature validation.
// This is safe here because the token was obtained directly from the token endpoint
// after a successful authorization code exchange.
func parseJWTClaims(token string) (map[string]any, error) {
	if token == "" {
		return nil, nil
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid jwt format")
	}
	payload := parts[1]
	// Pad base64 if needed
	if pad := len(payload) % 4; pad != 0 {
		payload += strings.Repeat("=", 4-pad)
	}
	b, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		// try raw encoding (no padding)
		b, err = base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			return nil, err
		}
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// getString safely extracts a string value from claims (handles string, number, single-element slice).
func getString(m map[string]any, key string) string {
	if m == nil || key == "" {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []any:
		for _, x := range val {
			if s, ok := x.(string); ok && s != "" {
				return s
			}
		}
	case float64:
		return fmt.Sprintf("%v", val)
	default:
		return fmt.Sprintf("%v", val)
	}
	return ""
}

// pickClaim tries the primary configured claim first, then the provided fallbacks.
func pickClaim(all map[string]any, primary string, fallbacks ...string) string {
	if s := getString(all, primary); s != "" {
		return s
	}
	for _, f := range fallbacks {
		if s := getString(all, f); s != "" {
			return s
		}
	}
	return ""
}

// mergeInto copies src into dst (dst is mutated). Later values win.
func mergeInto(dst, src map[string]any) {
	if src == nil {
		return
	}
	for k, v := range src {
		dst[k] = v
	}
}

// extractStringSlice returns a []string for common group/role claims that may be string or []string.
func extractStringSlice(m map[string]any, key string) []string {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	switch val := v.(type) {
	case []string:
		return append([]string(nil), val...)
	case []any:
		out := make([]string, 0, len(val))
		for _, x := range val {
			if s, ok := x.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		if val != "" {
			return []string{val}
		}
	}
	return nil
}

func sameSiteMode(value string) http.SameSite {
	switch strings.ToLower(value) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
