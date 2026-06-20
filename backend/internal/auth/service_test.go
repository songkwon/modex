package auth

import (
	"context"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"modex/backend/internal/store"
)

func TestBeginLoginUsesNonceAndPKCE(t *testing.T) {
	service := NewService(Config{
		Mode:          "oidc",
		IssuerURL:     "https://issuer.example.com",
		AuthURL:       "https://issuer.example.com/authorize",
		TokenURL:      "https://issuer.example.com/token",
		ClientID:      "modex",
		RedirectURL:   "https://modex.example.com/callback",
		Scopes:        []string{"openid"},
		StateCookie:   "oauth_state",
		SessionCookie: "session",
		SessionTTL:    time.Hour,
	})
	recorder := httptest.NewRecorder()
	loginURL, err := service.BeginLogin(recorder)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(loginURL)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	for _, key := range []string{"state", "nonce", "code_challenge"} {
		if query.Get(key) == "" {
			t.Fatalf("missing %s in login URL", key)
		}
	}
	if query.Get("code_challenge_method") != "S256" {
		t.Fatalf("code challenge method = %q", query.Get("code_challenge_method"))
	}
	if !strings.Contains(recorder.Header().Get("Set-Cookie"), "HttpOnly") {
		t.Fatal("state cookie must be HttpOnly")
	}
}

func TestMemoryBackedSessionLifecycle(t *testing.T) {
	service := NewService(Config{SessionCookie: "session", SessionTTL: time.Hour})
	recorder := httptest.NewRecorder()
	user := store.User{ID: "user-1", Username: "alice"}
	if err := service.CreateSession(recorder, user); err != nil {
		t.Fatal(err)
	}
	response := recorder.Result()
	cookies := response.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d", len(cookies))
	}
	request := httptest.NewRequest("GET", "/", nil)
	request.AddCookie(cookies[0])
	got, ok := service.CurrentUser(request)
	if !ok || got.ID != user.ID {
		t.Fatalf("CurrentUser = %#v, %v", got, ok)
	}
	logout := httptest.NewRecorder()
	service.Logout(logout, request)
	if _, ok := service.CurrentUser(request); ok {
		t.Fatal("session should be deleted on logout")
	}
}

func TestMemorySessionExpires(t *testing.T) {
	sessions := newMemorySessionStore()
	if err := sessions.Set(context.Background(), "key", map[string]string{"id": "1"}, -time.Second); err != nil {
		t.Fatal(err)
	}
	var value map[string]string
	if err := sessions.Get(context.Background(), "key", &value); err != errSessionNotFound {
		t.Fatalf("Get error = %v", err)
	}
}
