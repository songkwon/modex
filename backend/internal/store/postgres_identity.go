package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type rowQueryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func queryJSONOne[T any](ctx context.Context, q rowQueryer, query string, args ...any) (T, error) {
	var zero T
	var raw []byte
	if err := q.QueryRow(ctx, query, args...).Scan(&raw); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return zero, ErrNotFound
		}
		return zero, err
	}
	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		return zero, err
	}
	return value, nil
}

func postgresError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrConflict
	}
	return err
}

func databaseID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UTC().UnixNano())
}

func (p *PostgresRepository) CurrentUser() User {
	user, err := queryJSONOne[User](context.Background(), p.pool, `SELECT to_jsonb(u) FROM users u ORDER BY username LIMIT 1`)
	if err != nil {
		return User{}
	}
	return user
}

func (p *PostgresRepository) Users(keyword string) []User {
	ctx := context.Background()
	q := "%" + strings.TrimSpace(keyword) + "%"
	var users []User
	if err := loadQuery(ctx, p.pool, `SELECT to_jsonb(u) FROM users u
		WHERE $1='' OR username ILIKE $1 OR display_name ILIKE $1 OR email ILIKE $1 OR department ILIKE $1
		ORDER BY username`, &users, q); err != nil {
		return []User{}
	}
	return users
}

func (p *PostgresRepository) UserByID(id string) (User, error) {
	var raw []byte
	var mcpToken string
	err := p.pool.QueryRow(context.Background(), `SELECT to_jsonb(u),COALESCE(mcp_token,'') FROM users u WHERE id=$1`, id).Scan(&raw,&mcpToken)
	if errors.Is(err,pgx.ErrNoRows){return User{},ErrNotFound}
	if err!=nil{return User{},err}
	var user User
	if err=json.Unmarshal(raw,&user);err!=nil{return User{},err}
	user.MCPToken=mcpToken
	return user,nil
}

func (p *PostgresRepository) UserByMCPToken(token string) (User, error) {
	user, err := queryJSONOne[User](context.Background(), p.pool, `SELECT to_jsonb(u) FROM users u WHERE mcp_token=$1`, token)
	if err == nil {
		user.MCPToken = token
	}
	return user, err
}

func (p *PostgresRepository) SetUserMCPToken(id, token string) (User, error) {
	user, err := queryJSONOne[User](context.Background(), p.pool, `UPDATE users SET mcp_token=$2,updated_at=now() WHERE id=$1 RETURNING to_jsonb(users)`, id, token)
	if err == nil {
		user.MCPToken = token
	}
	return user, err
}

func (p *PostgresRepository) CreateUser(u User) (User, error) {
	u.Username = strings.TrimSpace(u.Username)
	if u.Username == "" {
		return User{}, ErrInvalid
	}
	if u.ID == "" {
		u.ID = databaseID("u")
	}
	if u.DisplayName == "" {
		u.DisplayName = u.Username
	}
	if u.Source == "" {
		u.Source = "manual"
	}
	if u.Status == "" {
		u.Status = "active"
	}
	ctx := context.Background()
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)
	created, err := queryJSONOne[User](ctx, tx, `INSERT INTO users
		(id,username,display_name,email,department,avatar,roles_json,managed_categories_json,source,status,is_super_admin,mcp_token,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7::jsonb,$8::jsonb,$9,$10,$11,$12,now(),now()) RETURNING to_jsonb(users)`,
		u.ID, u.Username, u.DisplayName, u.Email, u.Department, u.Avatar, mustJSON(u.Roles), mustJSON(u.ManagedCategories), u.Source, u.Status, u.SuperAdmin, u.MCPToken)
	if err != nil {
		return User{}, postgresError(err)
	}
	return created, tx.Commit(ctx)
}

func (p *PostgresRepository) UpdateUser(id string, patch User) (User, error) {
	ctx := context.Background()
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)
	current, err := queryJSONOne[User](ctx, tx, `SELECT to_jsonb(u) FROM users u WHERE id=$1 FOR UPDATE`, id)
	if err != nil {
		return User{}, err
	}
	if patch.DisplayName != "" {
		current.DisplayName = patch.DisplayName
	}
	if patch.Email != "" {
		current.Email = patch.Email
	}
	if patch.Department != "" {
		current.Department = patch.Department
	}
	if patch.Roles != nil {
		current.Roles = patch.Roles
	}
	if patch.ManagedCategories != nil {
		current.ManagedCategories = patch.ManagedCategories
	}
	if patch.Status != "" {
		current.Status = patch.Status
	}
	current.SuperAdmin = patch.SuperAdmin
	updated, err := queryJSONOne[User](ctx, tx, `UPDATE users SET display_name=$2,email=$3,department=$4,roles_json=$5::jsonb,managed_categories_json=$6::jsonb,status=$7,is_super_admin=$8,updated_at=now() WHERE id=$1 RETURNING to_jsonb(users)`, id, current.DisplayName, current.Email, current.Department, mustJSON(current.Roles), mustJSON(current.ManagedCategories), current.Status, current.SuperAdmin)
	if err != nil {
		return User{}, err
	}
	return updated, tx.Commit(ctx)
}

func (p *PostgresRepository) DeleteUser(id string) error {
	command, err := p.pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresRepository) UpsertUser(u User) User {
	u.Username = strings.TrimSpace(u.Username)
	if u.ID == "" {
		u.ID = databaseID("u")
	}
	if u.Status == "" {
		u.Status = "active"
	}
	ctx := context.Background()
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return u
	}
	defer tx.Rollback(ctx)
	result, err := queryJSONOne[User](ctx, tx, `INSERT INTO users
		(id,username,display_name,email,department,avatar,roles_json,managed_categories_json,source,status,is_super_admin,last_login_at,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7::jsonb,$8::jsonb,'oidc','active',$9,now(),now(),now())
		ON CONFLICT(username) DO UPDATE SET display_name=COALESCE(NULLIF(EXCLUDED.display_name,''),users.display_name),email=COALESCE(NULLIF(EXCLUDED.email,''),users.email),department=COALESCE(NULLIF(EXCLUDED.department,''),users.department),avatar=COALESCE(NULLIF(EXCLUDED.avatar,''),users.avatar),roles_json=CASE WHEN jsonb_array_length(EXCLUDED.roles_json)>0 THEN EXCLUDED.roles_json ELSE users.roles_json END,source='oidc',status='active',last_login_at=now(),updated_at=now()
		RETURNING to_jsonb(users)`, u.ID, u.Username, u.DisplayName, u.Email, u.Department, u.Avatar, mustJSON(u.Roles), mustJSON(u.ManagedCategories), u.SuperAdmin)
	if err != nil {
		return u
	}
	if tx.Commit(ctx) != nil {
		return u
	}
	return result
}

func (p *PostgresRepository) Teams() []Team {
	var teams []Team
	if err := loadQuery(context.Background(), p.pool, `SELECT to_jsonb(t) FROM teams t ORDER BY key`, &teams); err != nil {
		return []Team{}
	}
	return teams
}

func (p *PostgresRepository) Team(key string) (Team, error) {
	return queryJSONOne[Team](context.Background(), p.pool, `SELECT to_jsonb(t) FROM teams t WHERE lower(key)=lower($1) OR id=$1`, key)
}

func normalizeTeam(t Team) Team {
	if t.Name == "" {
		t.Name = t.Key
	}
	for _, leader := range t.Leaders {
		if leader != "" && !contains(t.Members, leader) {
			t.Members = append(t.Members, leader)
		}
	}
	return t
}

func (p *PostgresRepository) CreateTeam(t Team) (Team, error) {
	t.Key = strings.TrimSpace(t.Key)
	if t.Key == "" {
		return Team{}, ErrInvalid
	}
	if t.ID == "" {
		t.ID = databaseID("t")
	}
	t = normalizeTeam(t)
	ctx := context.Background()
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return Team{}, err
	}
	defer tx.Rollback(ctx)
	created, err := queryJSONOne[Team](ctx, tx, `INSERT INTO teams(id,key,name,description,leaders,members,created_at,updated_at) VALUES($1,$2,$3,$4,$5::jsonb,$6::jsonb,now(),now()) RETURNING to_jsonb(teams)`, t.ID, t.Key, t.Name, t.Description, mustJSON(t.Leaders), mustJSON(t.Members))
	if err != nil {
		return Team{}, postgresError(err)
	}
	return created, tx.Commit(ctx)
}

func (p *PostgresRepository) UpdateTeam(key string, patch Team) (Team, error) {
	current, err := p.Team(key)
	if err != nil {
		return Team{}, err
	}
	if patch.Name != "" {
		current.Name = patch.Name
	}
	if patch.Description != "" {
		current.Description = patch.Description
	}
	if patch.Leaders != nil {
		current.Leaders = cloneStrings(patch.Leaders)
	}
	if patch.Members != nil {
		current.Members = cloneStrings(patch.Members)
	}
	current = normalizeTeam(current)
	return queryJSONOne[Team](context.Background(), p.pool, `UPDATE teams SET name=$2,description=$3,leaders=$4::jsonb,members=$5::jsonb,updated_at=now() WHERE id=$1 RETURNING to_jsonb(teams)`, current.ID, current.Name, current.Description, mustJSON(current.Leaders), mustJSON(current.Members))
}

func (p *PostgresRepository) DeleteTeam(key string) error {
	command, err := p.pool.Exec(context.Background(), `DELETE FROM teams WHERE lower(key)=lower($1) OR id=$1`, key)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresRepository) AddTeamMember(key, member string) (Team, error) {
	member = strings.TrimSpace(member)
	if member == "" {
		return Team{}, ErrInvalid
	}
	t, err := p.Team(key)
	if err != nil {
		return Team{}, err
	}
	if !contains(t.Members, member) {
		t.Members = append(t.Members, member)
	}
	return queryJSONOne[Team](context.Background(), p.pool, `UPDATE teams SET members=$2::jsonb,updated_at=now() WHERE id=$1 RETURNING to_jsonb(teams)`, t.ID, mustJSON(t.Members))
}

func (p *PostgresRepository) RemoveTeamMember(key, member string) (Team, error) {
	t, err := p.Team(key)
	if err != nil {
		return Team{}, err
	}
	members := make([]string, 0, len(t.Members))
	for _, value := range t.Members {
		if !strings.EqualFold(value, strings.TrimSpace(member)) {
			members = append(members, value)
		}
	}
	return queryJSONOne[Team](context.Background(), p.pool, `UPDATE teams SET members=$2::jsonb,updated_at=now() WHERE id=$1 RETURNING to_jsonb(teams)`, t.ID, mustJSON(members))
}

func (p *PostgresRepository) SetTeamLeader(key, leader string) (Team, error) {
	leader = strings.TrimSpace(leader)
	if leader == "" {
		return Team{}, ErrInvalid
	}
	t, err := p.Team(key)
	if err != nil {
		return Team{}, err
	}
	if !contains(t.Leaders, leader) {
		t.Leaders = append(t.Leaders, leader)
	}
	if !contains(t.Members, leader) {
		t.Members = append(t.Members, leader)
	}
	return queryJSONOne[Team](context.Background(), p.pool, `UPDATE teams SET leaders=$2::jsonb,members=$3::jsonb,updated_at=now() WHERE id=$1 RETURNING to_jsonb(teams)`, t.ID, mustJSON(t.Leaders), mustJSON(t.Members))
}

func (p *PostgresRepository) TeamMembers(key string) []string {
	t, err := p.Team(key)
	if err != nil {
		return nil
	}
	return cloneStrings(t.Members)
}

func (p *PostgresRepository) TeamKeysForUser(u User) []string {
	var keys []string
	rows, err := p.pool.Query(context.Background(), `SELECT key FROM teams WHERE members ? $1 OR members ? $2 ORDER BY key`, u.Username, u.ID)
	if err != nil {
		return keys
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		if rows.Scan(&key) == nil {
			keys = append(keys, key)
		}
	}
	return keys
}
