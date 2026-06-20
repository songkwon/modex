package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"modex/backend/internal/store"
)

func TestConnectedAppOAuthAuthorizationCodeFlow(t *testing.T) {
	t.Setenv("SUPER_ADMIN_USERS", "dev")
	t.Setenv("APP_BASE_URL", "http://modex.test")

	srv := New(store.NewSeededTestStore())
	handler := srv.Handler()

	login := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/mock-login", strings.NewReader(`{"username":"dev"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(login, loginReq)
	if login.Code != http.StatusOK {
		t.Fatalf("login status = %d, body=%s", login.Code, login.Body.String())
	}
	cookies := login.Result().Cookies()

	createBody := `{"name":"External MCP Client","redirect_uris":["https://client.example.com/oauth/modex/callback"],"scopes":["modex:mcp:read","modex:docs:read"],"trusted":true}`
	create := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/admin/connected-apps", strings.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		createReq.AddCookie(c)
	}
	handler.ServeHTTP(create, createReq)
	if create.Code != http.StatusCreated {
		t.Fatalf("create app status = %d, body=%s", create.Code, create.Body.String())
	}
	var app connectedAppResponse
	if err := json.Unmarshal(create.Body.Bytes(), &app); err != nil {
		t.Fatal(err)
	}
	if app.ClientID == "" || app.ClientSecret == "" {
		t.Fatalf("missing client credentials: %+v", app)
	}

	authURL := "/oauth/authorize?response_type=code&client_id=" + url.QueryEscape(app.ClientID) +
		"&redirect_uri=" + url.QueryEscape("https://client.example.com/oauth/modex/callback") +
		"&scope=" + url.QueryEscape("modex:mcp:read modex:docs:read") +
		"&state=s1"
	auth := httptest.NewRecorder()
	authReq := httptest.NewRequest(http.MethodGet, authURL, nil)
	for _, c := range cookies {
		authReq.AddCookie(c)
	}
	handler.ServeHTTP(auth, authReq)
	if auth.Code != http.StatusFound {
		t.Fatalf("authorize status = %d, body=%s", auth.Code, auth.Body.String())
	}
	loc, err := url.Parse(auth.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	code := loc.Query().Get("code")
	if code == "" || loc.Query().Get("state") != "s1" {
		t.Fatalf("bad authorize redirect: %s", loc.String())
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", "https://client.example.com/oauth/modex/callback")
	token := httptest.NewRecorder()
	tokenReq := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokenReq.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(url.QueryEscape(app.ClientID)+":"+url.QueryEscape(app.ClientSecret))))
	handler.ServeHTTP(token, tokenReq)
	if token.Code != http.StatusOK {
		t.Fatalf("token status = %d, body=%s", token.Code, token.Body.String())
	}
	var tok struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		Scope        string `json:"scope"`
	}
	if err := json.Unmarshal(token.Body.Bytes(), &tok); err != nil {
		t.Fatal(err)
	}
	if tok.AccessToken == "" || tok.RefreshToken == "" || !strings.Contains(tok.Scope, "modex:mcp:read") {
		t.Fatalf("bad token response: %+v", tok)
	}

	me := httptest.NewRecorder()
	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	handler.ServeHTTP(me, meReq)
	if me.Code != http.StatusOK || !strings.Contains(me.Body.String(), `"username":"dev"`) {
		t.Fatalf("bearer me status = %d, body=%s", me.Code, me.Body.String())
	}

	admin := httptest.NewRecorder()
	adminReq := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	adminReq.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	handler.ServeHTTP(admin, adminReq)
	if admin.Code != http.StatusUnauthorized {
		t.Fatalf("oauth bearer should not enter admin APIs, status = %d, body=%s", admin.Code, admin.Body.String())
	}
}
