package store

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

func (s *MemoryStore) ConnectedApps() []ConnectedApp {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := append([]ConnectedApp{}, s.apps...)
	return out
}

func (s *MemoryStore) ConnectedAppByClientID(clientID string) (ConnectedApp, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, app := range s.apps {
		if app.ClientID == clientID {
			return app, nil
		}
	}
	return ConnectedApp{}, ErrNotFound
}

func (s *MemoryStore) CreateConnectedApp(app ConnectedApp, clientSecret string) (ConnectedApp, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(app.Name) == "" || strings.TrimSpace(app.ClientID) == "" || len(app.RedirectURIs) == 0 {
		return ConnectedApp{}, ErrInvalid
	}
	for _, existing := range s.apps {
		if strings.EqualFold(existing.ClientID, app.ClientID) {
			return ConnectedApp{}, ErrConflict
		}
	}
	if app.ID == "" {
		app.ID = s.nextIDLocked("app")
	}
	app.Name = strings.TrimSpace(app.Name)
	app.ClientID = strings.TrimSpace(app.ClientID)
	app.RedirectURIs = cleanNonEmpty(app.RedirectURIs)
	app.Scopes = normalizeScopes(app.Scopes)
	if len(app.Scopes) == 0 {
		app.Scopes = []string{"modex:mcp:read"}
	}
	app.ClientSecretHash = hashToken(clientSecret)
	now := time.Now().UTC()
	app.CreatedAt = now
	app.UpdatedAt = now
	s.apps = append(s.apps, app)
	return app, nil
}

func (s *MemoryStore) UpdateConnectedApp(id string, patch ConnectedApp) (ConnectedApp, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.apps {
		if s.apps[i].ID != id {
			continue
		}
		app := &s.apps[i]
		if strings.TrimSpace(patch.Name) != "" {
			app.Name = strings.TrimSpace(patch.Name)
		}
		app.Description = patch.Description
		if patch.RedirectURIs != nil {
			uris := cleanNonEmpty(patch.RedirectURIs)
			if len(uris) == 0 {
				return ConnectedApp{}, ErrInvalid
			}
			app.RedirectURIs = uris
		}
		if patch.Scopes != nil {
			app.Scopes = normalizeScopes(patch.Scopes)
		}
		app.Trusted = patch.Trusted
		app.Enabled = patch.Enabled
		app.UpdatedAt = time.Now().UTC()
		return *app, nil
	}
	return ConnectedApp{}, ErrNotFound
}

func (s *MemoryStore) DeleteConnectedApp(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.apps {
		if s.apps[i].ID == id {
			s.apps = append(s.apps[:i], s.apps[i+1:]...)
			now := time.Now().UTC()
			for j := range s.grants {
				if s.grants[j].AppID == id && s.grants[j].RevokedAt.IsZero() {
					s.grants[j].RevokedAt = now
					s.grants[j].UpdatedAt = now
				}
			}
			return nil
		}
	}
	return ErrNotFound
}

func (s *MemoryStore) VerifyConnectedAppSecret(clientID, clientSecret string) (ConnectedApp, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h := hashToken(clientSecret)
	for _, app := range s.apps {
		if app.ClientID == clientID && app.ClientSecretHash == h && app.Enabled {
			return app, nil
		}
	}
	return ConnectedApp{}, ErrNotFound
}

func (s *MemoryStore) CreateOAuthCode(appID, userID, redirectURI string, scopes []string, code string, ttl time.Duration) (OAuthGrant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if appID == "" || userID == "" || redirectURI == "" || code == "" {
		return OAuthGrant{}, ErrInvalid
	}
	now := time.Now().UTC()
	g := OAuthGrant{
		ID:            s.nextIDLocked("grant"),
		AppID:         appID,
		UserID:        userID,
		CodeHash:      hashToken(code),
		RedirectURI:   redirectURI,
		Scopes:        normalizeScopes(scopes),
		CodeExpiresAt: now.Add(ttl),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	s.grants = append(s.grants, g)
	return g, nil
}

func (s *MemoryStore) RedeemOAuthCode(clientID, code, redirectURI, accessToken, refreshToken string, accessTTL, refreshTTL time.Duration) (OAuthGrant, ConnectedApp, User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for i := range s.grants {
		g := &s.grants[i]
		if g.CodeHash != hashToken(code) || g.RedirectURI != redirectURI || !g.RevokedAt.IsZero() || now.After(g.CodeExpiresAt) {
			continue
		}
		app, ok := s.appByIDLocked(g.AppID)
		if !ok || app.ClientID != clientID || !app.Enabled {
			return OAuthGrant{}, ConnectedApp{}, User{}, ErrNotFound
		}
		user, err := s.userByIDLocked(g.UserID)
		if err != nil {
			return OAuthGrant{}, ConnectedApp{}, User{}, err
		}
		g.CodeHash = ""
		g.AccessTokenHash = hashToken(accessToken)
		g.RefreshTokenHash = hashToken(refreshToken)
		g.AccessExpiresAt = now.Add(accessTTL)
		g.RefreshExpiresAt = now.Add(refreshTTL)
		g.UpdatedAt = now
		s.touchAppLocked(g.AppID, now)
		return *g, app, user, nil
	}
	return OAuthGrant{}, ConnectedApp{}, User{}, ErrNotFound
}

func (s *MemoryStore) RefreshOAuthToken(clientID, refreshToken, accessToken, nextRefreshToken string, accessTTL, refreshTTL time.Duration) (OAuthGrant, ConnectedApp, User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for i := range s.grants {
		g := &s.grants[i]
		if g.RefreshTokenHash != hashToken(refreshToken) || !g.RevokedAt.IsZero() || now.After(g.RefreshExpiresAt) {
			continue
		}
		app, ok := s.appByIDLocked(g.AppID)
		if !ok || app.ClientID != clientID || !app.Enabled {
			return OAuthGrant{}, ConnectedApp{}, User{}, ErrNotFound
		}
		user, err := s.userByIDLocked(g.UserID)
		if err != nil {
			return OAuthGrant{}, ConnectedApp{}, User{}, err
		}
		g.AccessTokenHash = hashToken(accessToken)
		g.RefreshTokenHash = hashToken(nextRefreshToken)
		g.AccessExpiresAt = now.Add(accessTTL)
		g.RefreshExpiresAt = now.Add(refreshTTL)
		g.UpdatedAt = now
		s.touchAppLocked(g.AppID, now)
		return *g, app, user, nil
	}
	return OAuthGrant{}, ConnectedApp{}, User{}, ErrNotFound
}

func (s *MemoryStore) UserByOAuthAccessToken(token string) (User, ConnectedApp, OAuthGrant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	h := hashToken(token)
	for i := range s.grants {
		g := &s.grants[i]
		if g.AccessTokenHash != h || !g.RevokedAt.IsZero() || now.After(g.AccessExpiresAt) {
			continue
		}
		app, ok := s.appByIDLocked(g.AppID)
		if !ok || !app.Enabled {
			return User{}, ConnectedApp{}, OAuthGrant{}, ErrNotFound
		}
		user, err := s.userByIDLocked(g.UserID)
		if err != nil {
			return User{}, ConnectedApp{}, OAuthGrant{}, err
		}
		g.UpdatedAt = now
		s.touchAppLocked(g.AppID, now)
		return user, app, *g, nil
	}
	return User{}, ConnectedApp{}, OAuthGrant{}, ErrNotFound
}

func (s *MemoryStore) RevokeOAuthToken(clientID, token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	h := hashToken(token)
	revoked := false
	for i := range s.grants {
		g := &s.grants[i]
		app, ok := s.appByIDLocked(g.AppID)
		if !ok || app.ClientID != clientID {
			continue
		}
		if g.AccessTokenHash == h || g.RefreshTokenHash == h || g.CodeHash == h {
			g.RevokedAt = now
			g.UpdatedAt = now
			revoked = true
		}
	}
	return revoked
}

func (s *MemoryStore) appByIDLocked(id string) (ConnectedApp, bool) {
	for _, app := range s.apps {
		if app.ID == id {
			return app, true
		}
	}
	return ConnectedApp{}, false
}

func (s *MemoryStore) touchAppLocked(id string, at time.Time) {
	for i := range s.apps {
		if s.apps[i].ID == id {
			s.apps[i].LastUsedAt = at
			s.apps[i].UpdatedAt = at
			return
		}
	}
}

func hashToken(token string) string {
	if token == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func cleanNonEmpty(items []string) []string {
	var out []string
	for _, item := range items {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func normalizeScopes(scopes []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, scope := range scopes {
		for _, part := range strings.FieldsFunc(scope, func(r rune) bool { return r == ' ' || r == ',' }) {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if _, ok := seen[part]; ok {
				continue
			}
			seen[part] = struct{}{}
			out = append(out, part)
		}
	}
	return out
}
