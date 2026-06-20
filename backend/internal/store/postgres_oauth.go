package store

import (
	"context"
	"strings"
	"time"
)

func (p *PostgresRepository) ConnectedApps() []ConnectedApp {
	var apps []ConnectedApp
	if err := loadQuery(context.Background(), p.pool, `SELECT to_jsonb(a) FROM connected_app a ORDER BY created_at DESC,id`, &apps); err != nil {
		return []ConnectedApp{}
	}
	return apps
}
func (p *PostgresRepository) ConnectedAppByClientID(clientID string) (ConnectedApp, error) {
	return queryJSONOne[ConnectedApp](context.Background(), p.pool, `SELECT to_jsonb(a) FROM connected_app a WHERE client_id=$1`, clientID)
}
func (p *PostgresRepository) CreateConnectedApp(app ConnectedApp, clientSecret string) (ConnectedApp, error) {
	app.Name = strings.TrimSpace(app.Name)
	app.ClientID = strings.TrimSpace(app.ClientID)
	app.RedirectURIs = cleanNonEmpty(app.RedirectURIs)
	app.Scopes = normalizeScopes(app.Scopes)
	if app.Name == "" || app.ClientID == "" || len(app.RedirectURIs) == 0 {
		return ConnectedApp{}, ErrInvalid
	}
	if len(app.Scopes) == 0 {
		app.Scopes = []string{"modex:mcp:read"}
	}
	if app.ID == "" {
		app.ID = databaseID("app")
	}
	app.ClientSecretHash = hashToken(clientSecret)
	created, err := queryJSONOne[ConnectedApp](context.Background(), p.pool, `INSERT INTO connected_app(id,name,description,client_id,client_secret_hash,redirect_uris,scopes,trusted,enabled,created_by,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6::jsonb,$7::jsonb,$8,$9,NULLIF($10,''),now(),now()) RETURNING to_jsonb(connected_app)`, app.ID, app.Name, app.Description, app.ClientID, app.ClientSecretHash, mustJSON(app.RedirectURIs), mustJSON(app.Scopes), app.Trusted, app.Enabled, app.CreatedBy)
	return created, postgresError(err)
}
func (p *PostgresRepository) UpdateConnectedApp(id string, patch ConnectedApp) (ConnectedApp, error) {
	app, err := queryJSONOne[ConnectedApp](context.Background(), p.pool, `SELECT to_jsonb(a) FROM connected_app a WHERE id=$1`, id)
	if err != nil {
		return ConnectedApp{}, err
	}
	if strings.TrimSpace(patch.Name) != "" {
		app.Name = strings.TrimSpace(patch.Name)
	}
	app.Description = patch.Description
	if patch.RedirectURIs != nil {
		app.RedirectURIs = cleanNonEmpty(patch.RedirectURIs)
		if len(app.RedirectURIs) == 0 {
			return ConnectedApp{}, ErrInvalid
		}
	}
	if patch.Scopes != nil {
		app.Scopes = normalizeScopes(patch.Scopes)
	}
	app.Trusted = patch.Trusted
	app.Enabled = patch.Enabled
	return queryJSONOne[ConnectedApp](context.Background(), p.pool, `UPDATE connected_app SET name=$2,description=$3,redirect_uris=$4::jsonb,scopes=$5::jsonb,trusted=$6,enabled=$7,updated_at=now() WHERE id=$1 RETURNING to_jsonb(connected_app)`, id, app.Name, app.Description, mustJSON(app.RedirectURIs), mustJSON(app.Scopes), app.Trusted, app.Enabled)
}
func (p *PostgresRepository) DeleteConnectedApp(id string) error {
	ctx := context.Background()
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `DELETE FROM oauth_grant WHERE app_id=$1`, id); err != nil {
		return err
	}
	command, err := tx.Exec(ctx, `DELETE FROM connected_app WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}
func (p *PostgresRepository) VerifyConnectedAppSecret(clientID, clientSecret string) (ConnectedApp, error) {
	return queryJSONOne[ConnectedApp](context.Background(), p.pool, `SELECT to_jsonb(a) FROM connected_app a WHERE client_id=$1 AND client_secret_hash=$2 AND enabled=true`, clientID, hashToken(clientSecret))
}

func (p *PostgresRepository) CreateOAuthCode(appID, userID, redirectURI string, scopes []string, code string, ttl time.Duration) (OAuthGrant, error) {
	if appID == "" || userID == "" || redirectURI == "" || code == "" {
		return OAuthGrant{}, ErrInvalid
	}
	return queryJSONOne[OAuthGrant](context.Background(), p.pool, `INSERT INTO oauth_grant(id,app_id,user_id,code_hash,redirect_uri,scopes,code_expires_at,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6::jsonb,$7,now(),now()) RETURNING to_jsonb(oauth_grant)`, databaseID("grant"), appID, userID, hashToken(code), redirectURI, mustJSON(normalizeScopes(scopes)), time.Now().UTC().Add(ttl))
}

func (p *PostgresRepository) RedeemOAuthCode(clientID, code, redirectURI, accessToken, refreshToken string, accessTTL, refreshTTL time.Duration) (OAuthGrant, ConnectedApp, User, error) {
	ctx := context.Background()
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return OAuthGrant{}, ConnectedApp{}, User{}, err
	}
	defer tx.Rollback(ctx)
	grant, err := queryJSONOne[OAuthGrant](ctx, tx, `SELECT to_jsonb(g) FROM oauth_grant g JOIN connected_app a ON a.id=g.app_id WHERE g.code_hash=$1 AND g.redirect_uri=$2 AND g.revoked_at IS NULL AND g.code_expires_at>now() AND a.client_id=$3 AND a.enabled=true FOR UPDATE OF g`, hashToken(code), redirectURI, clientID)
	if err != nil {
		return OAuthGrant{}, ConnectedApp{}, User{}, err
	}
	app, err := queryJSONOne[ConnectedApp](ctx, tx, `SELECT to_jsonb(a) FROM connected_app a WHERE id=$1`, grant.AppID)
	if err != nil {
		return OAuthGrant{}, ConnectedApp{}, User{}, err
	}
	user, err := queryJSONOne[User](ctx, tx, `SELECT to_jsonb(u) FROM users u WHERE id=$1`, grant.UserID)
	if err != nil {
		return OAuthGrant{}, ConnectedApp{}, User{}, err
	}
	now := time.Now().UTC()
	grant, err = queryJSONOne[OAuthGrant](ctx, tx, `UPDATE oauth_grant SET code_hash='',access_token_hash=$2,refresh_token_hash=$3,access_expires_at=$4,refresh_expires_at=$5,updated_at=$6 WHERE id=$1 RETURNING to_jsonb(oauth_grant)`, grant.ID, hashToken(accessToken), hashToken(refreshToken), now.Add(accessTTL), now.Add(refreshTTL), now)
	if err != nil {
		return OAuthGrant{}, ConnectedApp{}, User{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE connected_app SET last_used_at=$2,updated_at=$2 WHERE id=$1`, app.ID, now); err != nil {
		return OAuthGrant{}, ConnectedApp{}, User{}, err
	}
	return grant, app, user, tx.Commit(ctx)
}

func (p *PostgresRepository) RefreshOAuthToken(clientID, refreshToken, accessToken, nextRefreshToken string, accessTTL, refreshTTL time.Duration) (OAuthGrant, ConnectedApp, User, error) {
	ctx := context.Background()
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return OAuthGrant{}, ConnectedApp{}, User{}, err
	}
	defer tx.Rollback(ctx)
	grant, err := queryJSONOne[OAuthGrant](ctx, tx, `SELECT to_jsonb(g) FROM oauth_grant g JOIN connected_app a ON a.id=g.app_id WHERE g.refresh_token_hash=$1 AND g.revoked_at IS NULL AND g.refresh_expires_at>now() AND a.client_id=$2 AND a.enabled=true FOR UPDATE OF g`, hashToken(refreshToken), clientID)
	if err != nil {
		return OAuthGrant{}, ConnectedApp{}, User{}, err
	}
	app, err := queryJSONOne[ConnectedApp](ctx, tx, `SELECT to_jsonb(a) FROM connected_app a WHERE id=$1`, grant.AppID)
	if err != nil {
		return OAuthGrant{}, ConnectedApp{}, User{}, err
	}
	user, err := queryJSONOne[User](ctx, tx, `SELECT to_jsonb(u) FROM users u WHERE id=$1`, grant.UserID)
	if err != nil {
		return OAuthGrant{}, ConnectedApp{}, User{}, err
	}
	now := time.Now().UTC()
	grant, err = queryJSONOne[OAuthGrant](ctx, tx, `UPDATE oauth_grant SET access_token_hash=$2,refresh_token_hash=$3,access_expires_at=$4,refresh_expires_at=$5,updated_at=$6 WHERE id=$1 RETURNING to_jsonb(oauth_grant)`, grant.ID, hashToken(accessToken), hashToken(nextRefreshToken), now.Add(accessTTL), now.Add(refreshTTL), now)
	if err != nil {
		return OAuthGrant{}, ConnectedApp{}, User{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE connected_app SET last_used_at=$2,updated_at=$2 WHERE id=$1`, app.ID, now); err != nil {
		return OAuthGrant{}, ConnectedApp{}, User{}, err
	}
	return grant, app, user, tx.Commit(ctx)
}

func (p *PostgresRepository) UserByOAuthAccessToken(token string) (User, ConnectedApp, OAuthGrant, error) {
	ctx := context.Background()
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return User{}, ConnectedApp{}, OAuthGrant{}, err
	}
	defer tx.Rollback(ctx)
	grant, err := queryJSONOne[OAuthGrant](ctx, tx, `SELECT to_jsonb(g) FROM oauth_grant g JOIN connected_app a ON a.id=g.app_id WHERE g.access_token_hash=$1 AND g.revoked_at IS NULL AND g.access_expires_at>now() AND a.enabled=true FOR UPDATE OF g`, hashToken(token))
	if err != nil {
		return User{}, ConnectedApp{}, OAuthGrant{}, err
	}
	app, err := queryJSONOne[ConnectedApp](ctx, tx, `SELECT to_jsonb(a) FROM connected_app a WHERE id=$1`, grant.AppID)
	if err != nil {
		return User{}, ConnectedApp{}, OAuthGrant{}, err
	}
	user, err := queryJSONOne[User](ctx, tx, `SELECT to_jsonb(u) FROM users u WHERE id=$1`, grant.UserID)
	if err != nil {
		return User{}, ConnectedApp{}, OAuthGrant{}, err
	}
	now := time.Now().UTC()
	if _, err = tx.Exec(ctx, `UPDATE oauth_grant SET updated_at=$2 WHERE id=$1`, grant.ID, now); err != nil {
		return User{}, ConnectedApp{}, OAuthGrant{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE connected_app SET last_used_at=$2,updated_at=$2 WHERE id=$1`, app.ID, now); err != nil {
		return User{}, ConnectedApp{}, OAuthGrant{}, err
	}
	return user, app, grant, tx.Commit(ctx)
}
func (p *PostgresRepository) RevokeOAuthToken(clientID, token string) bool {
	command, err := p.pool.Exec(context.Background(), `UPDATE oauth_grant g SET revoked_at=now(),updated_at=now() FROM connected_app a WHERE a.id=g.app_id AND a.client_id=$1 AND (g.access_token_hash=$2 OR g.refresh_token_hash=$2 OR g.code_hash=$2)`, clientID, hashToken(token))
	return err == nil && command.RowsAffected() > 0
}
