package store

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestPostgresRepositoryRoundTrip(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	repository, err := OpenPostgresRepository(ctx, databaseURL)
	if err != nil {
		t.Fatalf("OpenPostgresRepository: %v", err)
	}
	defer repository.Close()

	st := NewSeeded()
	current := st.CurrentUser()
	if _, err = st.SetUserMCPToken(current.ID, "mcp-round-trip"); err != nil {
		t.Fatalf("SetUserMCPToken: %v", err)
	}
	app, err := st.CreateConnectedApp(ConnectedApp{
		Name:         "Repository Test",
		ClientID:     "repository-test",
		RedirectURIs: []string{"http://localhost/callback"},
		CreatedBy:    current.ID,
	}, "secret")
	if err != nil {
		t.Fatalf("CreateConnectedApp: %v", err)
	}
	if err = repository.Save(ctx, st); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, found, err := repository.load(ctx)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !found {
		t.Fatal("load did not find relational state")
	}
	loadedUser, err := loaded.UserByID(current.ID)
	if err != nil {
		t.Fatalf("UserByID: %v", err)
	}
	if loadedUser.MCPToken != "mcp-round-trip" {
		t.Fatalf("MCP token = %q, want round-trip value", loadedUser.MCPToken)
	}
	loadedApp, err := loaded.ConnectedAppByClientID(app.ClientID)
	if err != nil {
		t.Fatalf("ConnectedAppByClientID: %v", err)
	}
	if loadedApp.CreatedBy != current.ID {
		t.Fatalf("connected app created_by = %q, want %q", loadedApp.CreatedBy, current.ID)
	}

	if _, err = repository.pool.Exec(ctx, `CREATE TABLE modex_store_snapshot (key TEXT PRIMARY KEY, snapshot JSONB NOT NULL)`); err != nil {
		t.Fatalf("create obsolete snapshot table: %v", err)
	}
	if _, _, err = repository.LoadOrMigrate(ctx, ""); err != nil {
		t.Fatalf("LoadOrMigrate: %v", err)
	}
	var exists bool
	if err = repository.pool.QueryRow(ctx, `SELECT to_regclass('public.modex_store_snapshot') IS NOT NULL`).Scan(&exists); err != nil {
		t.Fatalf("check obsolete snapshot table: %v", err)
	}
	if exists {
		t.Fatal("modex_store_snapshot still exists after relational load")
	}

	// Recreate the pre-relational state: the legacy table contains the only
	// business data and the relational metadata marker is absent.
	if _, err = repository.pool.Exec(ctx, `DELETE FROM store_metadata`); err != nil {
		t.Fatalf("clear relational marker: %v", err)
	}
	legacyRaw, err := json.Marshal(st.toSnapshot(true))
	if err != nil {
		t.Fatalf("marshal legacy snapshot: %v", err)
	}
	if _, err = repository.pool.Exec(ctx, `CREATE TABLE modex_store_snapshot (key TEXT PRIMARY KEY, snapshot JSONB NOT NULL)`); err != nil {
		t.Fatalf("recreate legacy snapshot table: %v", err)
	}
	if _, err = repository.pool.Exec(ctx, `INSERT INTO modex_store_snapshot(key,snapshot) VALUES('main',$1::jsonb)`, string(legacyRaw)); err != nil {
		t.Fatalf("insert legacy snapshot: %v", err)
	}
	migratedStore, migrated, err := repository.LoadOrMigrate(ctx, "")
	if err != nil {
		t.Fatalf("migrate legacy snapshot: %v", err)
	}
	if !migrated {
		t.Fatal("legacy snapshot was not reported as migrated")
	}
	if _, err = migratedStore.UserByID(current.ID); err != nil {
		t.Fatalf("migrated user missing: %v", err)
	}
	if err = repository.pool.QueryRow(ctx, `SELECT to_regclass('public.modex_store_snapshot') IS NOT NULL`).Scan(&exists); err != nil {
		t.Fatalf("check migrated snapshot table: %v", err)
	}
	if exists {
		t.Fatal("modex_store_snapshot still exists after data migration")
	}
}
