package api

import (
	"encoding/base64"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"

	"modex/backend/internal/store"
)

const (
	oauthCodeTTL    = 10 * time.Minute
	oauthAccessTTL  = 60 * time.Minute
	oauthRefreshTTL = 90 * 24 * time.Hour
)

type connectedAppResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	ClientID     string    `json:"client_id"`
	ClientSecret string    `json:"client_secret,omitempty"`
	RedirectURIs []string  `json:"redirect_uris"`
	Scopes       []string  `json:"scopes"`
	Trusted      bool      `json:"trusted"`
	Enabled      bool      `json:"enabled"`
	CreatedBy    string    `json:"created_by,omitempty"`
	LastUsedAt   time.Time `json:"last_used_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func connectedAppOut(app store.ConnectedApp, secret string) connectedAppResponse {
	return connectedAppResponse{
		ID: app.ID, Name: app.Name, Description: app.Description, ClientID: app.ClientID, ClientSecret: secret,
		RedirectURIs: app.RedirectURIs, Scopes: app.Scopes, Trusted: app.Trusted, Enabled: app.Enabled,
		CreatedBy: app.CreatedBy, LastUsedAt: app.LastUsedAt, CreatedAt: app.CreatedAt, UpdatedAt: app.UpdatedAt,
	}
}

func (s *Server) handleOAuthMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}
	base := strings.TrimRight(s.auth.Config().AppBaseURL, "/")
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                base,
		"authorization_endpoint":                base + "/oauth/authorize",
		"token_endpoint":                        base + "/oauth/token",
		"revocation_endpoint":                   base + "/oauth/revoke",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post"},
		"scopes_supported":                      []string{"modex:mcp:read", "modex:docs:read"},
	})
}

func (s *Server) handleAdminConnectedApps(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireSuperAdmin(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		apps := s.store.ConnectedApps()
		out := make([]connectedAppResponse, 0, len(apps))
		for _, app := range apps {
			out = append(out, connectedAppOut(app, ""))
		}
		writeJSON(w, http.StatusOK, map[string]any{"apps": out})
	case http.MethodPost:
		var req struct {
			Name         string   `json:"name"`
			Description  string   `json:"description"`
			ClientID     string   `json:"client_id"`
			RedirectURIs []string `json:"redirect_uris"`
			Scopes       []string `json:"scopes"`
			Trusted      bool     `json:"trusted"`
			Enabled      *bool    `json:"enabled"`
		}
		if err := decodeBody(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		clientID := strings.TrimSpace(req.ClientID)
		if clientID == "" {
			suffix, err := randomToken(12)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "token_gen_failed", err.Error())
				return
			}
			clientID = "modex_" + suffix
		}
		secret, err := randomToken(32)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "token_gen_failed", err.Error())
			return
		}
		enabled := true
		if req.Enabled != nil {
			enabled = *req.Enabled
		}
		app, err := s.store.CreateConnectedApp(store.ConnectedApp{
			Name: req.Name, Description: req.Description, ClientID: clientID, RedirectURIs: req.RedirectURIs,
			Scopes: req.Scopes, Trusted: req.Trusted, Enabled: enabled, CreatedBy: user.ID,
		}, secret)
		writeMutation(w, connectedAppOut(app, secret), http.StatusCreated, err)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET or POST")
	}
}

func (s *Server) handleAdminConnectedAppByID(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSuperAdmin(w, r); !ok {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/admin/connected-apps/")
	if id == "" {
		writeError(w, http.StatusNotFound, "not_found", "connected app not found")
		return
	}
	switch r.Method {
	case http.MethodPut:
		var req struct {
			Name         string   `json:"name"`
			Description  string   `json:"description"`
			RedirectURIs []string `json:"redirect_uris"`
			Scopes       []string `json:"scopes"`
			Trusted      bool     `json:"trusted"`
			Enabled      bool     `json:"enabled"`
		}
		if err := decodeBody(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		app, err := s.store.UpdateConnectedApp(id, store.ConnectedApp{
			Name: req.Name, Description: req.Description, RedirectURIs: req.RedirectURIs,
			Scopes: req.Scopes, Trusted: req.Trusted, Enabled: req.Enabled,
		})
		writeMutation(w, connectedAppOut(app, ""), http.StatusOK, err)
	case http.MethodDelete:
		writeMutation(w, map[string]string{"status": "deleted"}, http.StatusOK, s.store.DeleteConnectedApp(id))
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use PUT or DELETE")
	}
}

func (s *Server) handleOAuthAuthorize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET or POST")
		return
	}
	user, ok := s.auth.CurrentUser(r)
	if !ok {
		login := s.auth.Config().AppBaseURL + "/api/auth/login?next=" + url.QueryEscape(r.URL.RequestURI())
		http.Redirect(w, r, login, http.StatusFound)
		return
	}
	q := r.URL.Query()
	clientID := q.Get("client_id")
	redirectURI := q.Get("redirect_uri")
	state := q.Get("state")
	if q.Get("response_type") != "code" {
		oauthRedirectError(w, r, redirectURI, state, "unsupported_response_type")
		return
	}
	app, err := s.store.ConnectedAppByClientID(clientID)
	if err != nil || !app.Enabled || !redirectURIAllowed(app.RedirectURIs, redirectURI) {
		oauthRedirectError(w, r, redirectURI, state, "invalid_client")
		return
	}
	scopes, ok := requestedScopes(q.Get("scope"), app.Scopes)
	if !ok {
		oauthRedirectError(w, r, redirectURI, state, "invalid_scope")
		return
	}
	if r.Method == http.MethodGet && !app.Trusted && q.Get("approve") != "1" {
		writeOAuthConsentPage(w, r, app, scopes)
		return
	}
	code, err := randomToken(32)
	if err != nil {
		oauthRedirectError(w, r, redirectURI, state, "server_error")
		return
	}
	if _, err := s.store.CreateOAuthCode(app.ID, user.ID, redirectURI, scopes, code, oauthCodeTTL); err != nil {
		oauthRedirectError(w, r, redirectURI, state, "server_error")
		return
	}
	u, _ := url.Parse(redirectURI)
	out := u.Query()
	out.Set("code", code)
	if state != "" {
		out.Set("state", state)
	}
	u.RawQuery = out.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func (s *Server) handleOAuthToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeOAuthTokenError(w, http.StatusMethodNotAllowed, "invalid_request", "use POST")
		return
	}
	if err := r.ParseForm(); err != nil {
		writeOAuthTokenError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	clientID, secret := oauthClientCredentials(r)
	app, err := s.store.VerifyConnectedAppSecret(clientID, secret)
	if err != nil {
		writeOAuthTokenError(w, http.StatusUnauthorized, "invalid_client", "client authentication failed")
		return
	}
	access, err := randomToken(32)
	if err != nil {
		writeOAuthTokenError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	refresh, err := randomToken(32)
	if err != nil {
		writeOAuthTokenError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	switch r.Form.Get("grant_type") {
	case "authorization_code":
		grant, _, _, err := s.store.RedeemOAuthCode(app.ClientID, r.Form.Get("code"), r.Form.Get("redirect_uri"), access, refresh, oauthAccessTTL, oauthRefreshTTL)
		if err != nil {
			writeOAuthTokenError(w, http.StatusBadRequest, "invalid_grant", "authorization code is invalid or expired")
			return
		}
		writeOAuthTokenResponse(w, access, refresh, grant.Scopes)
	case "refresh_token":
		grant, _, _, err := s.store.RefreshOAuthToken(app.ClientID, r.Form.Get("refresh_token"), access, refresh, oauthAccessTTL, oauthRefreshTTL)
		if err != nil {
			writeOAuthTokenError(w, http.StatusBadRequest, "invalid_grant", "refresh token is invalid or expired")
			return
		}
		writeOAuthTokenResponse(w, access, refresh, grant.Scopes)
	default:
		writeOAuthTokenError(w, http.StatusBadRequest, "unsupported_grant_type", "grant_type must be authorization_code or refresh_token")
	}
}

func (s *Server) handleOAuthRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeOAuthTokenError(w, http.StatusMethodNotAllowed, "invalid_request", "use POST")
		return
	}
	if err := r.ParseForm(); err != nil {
		writeOAuthTokenError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	clientID, secret := oauthClientCredentials(r)
	if _, err := s.store.VerifyConnectedAppSecret(clientID, secret); err != nil {
		writeOAuthTokenError(w, http.StatusUnauthorized, "invalid_client", "client authentication failed")
		return
	}
	s.store.RevokeOAuthToken(clientID, r.Form.Get("token"))
	writeJSON(w, http.StatusOK, map[string]any{"revoked": true})
}

func oauthClientCredentials(r *http.Request) (string, string) {
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Basic ") {
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(strings.TrimPrefix(auth, "Basic ")))
		if err == nil {
			parts := strings.SplitN(string(raw), ":", 2)
			if len(parts) == 2 {
				id, _ := url.QueryUnescape(parts[0])
				secret, _ := url.QueryUnescape(parts[1])
				return id, secret
			}
		}
	}
	return r.Form.Get("client_id"), r.Form.Get("client_secret")
}

func requestedScopes(raw string, allowed []string) ([]string, bool) {
	if strings.TrimSpace(raw) == "" {
		return allowed, true
	}
	allowedSet := map[string]struct{}{}
	for _, s := range allowed {
		allowedSet[s] = struct{}{}
	}
	var out []string
	for _, scope := range strings.Fields(raw) {
		if _, ok := allowedSet[scope]; !ok {
			return nil, false
		}
		out = append(out, scope)
	}
	return out, true
}

func redirectURIAllowed(allowed []string, redirectURI string) bool {
	for _, u := range allowed {
		if u == redirectURI {
			return true
		}
	}
	return false
}

func writeOAuthConsentPage(w http.ResponseWriter, r *http.Request, app store.ConnectedApp, scopes []string) {
	q := r.URL.Query()
	q.Set("approve", "1")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, `<!doctype html><html><head><meta charset="utf-8"><title>Authorize %s</title></head><body style="font-family:system-ui,sans-serif;max-width:560px;margin:64px auto;line-height:1.5"><h1>Authorize %s</h1><p>This app wants to access Modex with these scopes:</p><pre>%s</pre><form method="get" action="%s">`, html.EscapeString(app.Name), html.EscapeString(app.Name), html.EscapeString(strings.Join(scopes, "\n")), html.EscapeString(r.URL.Path))
	for key, vals := range q {
		for _, val := range vals {
			_, _ = fmt.Fprintf(w, `<input type="hidden" name="%s" value="%s">`, html.EscapeString(key), html.EscapeString(val))
		}
	}
	_, _ = fmt.Fprint(w, `<button type="submit" style="padding:8px 14px">Authorize</button></form></body></html>`)
}

func oauthRedirectError(w http.ResponseWriter, r *http.Request, redirectURI, state, code string) {
	if redirectURI == "" {
		writeError(w, http.StatusBadRequest, code, code)
		return
	}
	u, err := url.Parse(redirectURI)
	if err != nil {
		writeError(w, http.StatusBadRequest, code, code)
		return
	}
	q := u.Query()
	q.Set("error", code)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func writeOAuthTokenResponse(w http.ResponseWriter, access, refresh string, scopes []string) {
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  access,
		"token_type":    "Bearer",
		"expires_in":    int(oauthAccessTTL.Seconds()),
		"refresh_token": refresh,
		"scope":         strings.Join(scopes, " "),
	})
}

func writeOAuthTokenError(w http.ResponseWriter, status int, code, desc string) {
	writeJSON(w, status, map[string]any{"error": code, "error_description": desc})
}
