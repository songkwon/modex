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
	user, err := s.fetchUserInfo(ctx, token.AccessToken)
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

func (s *Service) fetchUserInfo(ctx context.Context, accessToken string) (store.User, error) {
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
	var info userInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return store.User{}, err
	}
	if resp.StatusCode >= 300 {
		return store.User{}, fmt.Errorf("OIDC userinfo endpoint returned %d", resp.StatusCode)
	}
	username := first(info.PreferredUsername, info.Email, info.Sub)
	displayName := first(info.Name, info.PreferredUsername, info.Email, info.Sub)
	return store.User{
		ID:          first(info.Sub, username),
		Username:    username,
		DisplayName: displayName,
		Email:       info.Email,
		Department:  info.Department,
		Groups:      info.Groups,
		Roles:       info.Roles,
	}, nil
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Error       string `json:"error"`
}

type userInfo struct {
	Sub               string   `json:"sub"`
	PreferredUsername string   `json:"preferred_username"`
	Name              string   `json:"name"`
	Email             string   `json:"email"`
	Department        string   `json:"department"`
	Groups            []string `json:"groups"`
	Roles             []string `json:"roles"`
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
