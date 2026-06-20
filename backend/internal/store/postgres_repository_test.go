package store

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestPostgresRepositoryRequestLevelCRUD(t *testing.T) {
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

	suffix := fmt.Sprint(time.Now().UnixNano())
	user, err := repository.CreateUser(User{ID: "u-db-" + suffix, Username: "db-" + suffix, DisplayName: "DB User"})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	category, err := repository.CreateCategory(Category{ID: "cat-" + suffix, Key: "cat-" + suffix, Name: "DB Category"})
	if err != nil {
		t.Fatalf("CreateCategory: %v", err)
	}
	module, err := repository.CreateModule(Module{ID: "m-db-" + suffix, ModuleKey: "module-" + suffix, Name: "DB Module", CategoryIDs: []string{category.ID}})
	if err != nil {
		t.Fatalf("CreateModule: %v", err)
	}
	version, err := repository.CreateVersion(module.ModuleKey, Version{ID: "v-db-" + suffix, DocsVersion: "latest", IsDefault: true})
	if err != nil {
		t.Fatalf("CreateVersion: %v", err)
	}
	_, err = repository.CreateEntry(module.ModuleKey, version.DocsVersion, Entry{ID: "e-db-" + suffix, EntryKey: "guide", Title: "Guide"})
	if err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}
	secondRepository, err := OpenPostgresRepository(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open second repository: %v", err)
	}
	defer secondRepository.Close()
	if _, err = secondRepository.Module(module.ModuleKey); err != nil {
		t.Fatalf("second repository did not observe committed module: %v", err)
	}
	app, err := repository.CreateConnectedApp(ConnectedApp{ID: "app-db-" + suffix, Name: "Repository Test", ClientID: "repository-" + suffix, RedirectURIs: []string{"http://localhost/callback"}, CreatedBy: user.ID, Enabled: true}, "secret")
	if err != nil {
		t.Fatalf("CreateConnectedApp: %v", err)
	}
	t.Cleanup(func() {
		cleanup, openErr := OpenPostgresRepository(context.Background(), databaseURL)
		if openErr != nil {
			return
		}
		defer cleanup.Close()
		_ = cleanup.DeleteConnectedApp(app.ID)
		_, _ = cleanup.pool.Exec(context.Background(), `DELETE FROM docs_page_view WHERE module_id=$1`, module.ID)
		_, _ = cleanup.pool.Exec(context.Background(), `DELETE FROM user_favorite WHERE module_key=$1`, module.ModuleKey)
		_, _ = cleanup.pool.Exec(context.Background(), `DELETE FROM user_recent_doc WHERE module_key=$1`, module.ModuleKey)
		_, _ = cleanup.pool.Exec(context.Background(), `DELETE FROM docs_feedback WHERE module_key=$1`, module.ModuleKey)
		_, _ = cleanup.pool.Exec(context.Background(), `DELETE FROM docs_embedding WHERE module_id=$1`, module.ID)
		_, _ = cleanup.pool.Exec(context.Background(), `DELETE FROM docs_page WHERE module_id=$1`, module.ID)
		_, _ = cleanup.pool.Exec(context.Background(), `DELETE FROM docs_release WHERE module_id=$1`, module.ID)
		_, _ = cleanup.pool.Exec(context.Background(), `DELETE FROM docs_entry WHERE module_id=$1`, module.ID)
		_, _ = cleanup.pool.Exec(context.Background(), `DELETE FROM docs_site_file WHERE module_key=$1`, module.ModuleKey)
		_, _ = cleanup.pool.Exec(context.Background(), `DELETE FROM docs_nav WHERE module_key=$1`, module.ModuleKey)
		_, _ = cleanup.pool.Exec(context.Background(), `DELETE FROM docs_version WHERE id=$1`, version.ID)
		_, _ = cleanup.pool.Exec(context.Background(), `DELETE FROM docs_module_category WHERE module_id=$1`, module.ID)
		_, _ = cleanup.pool.Exec(context.Background(), `DELETE FROM docs_module WHERE id=$1`, module.ID)
		_ = cleanup.DeleteCategory(category.ID)
		_ = cleanup.DeleteUser(user.ID)
	})

	updatedUser, err := repository.SetUserMCPToken(user.ID, "mcp-direct")
	if err != nil || updatedUser.MCPToken != "mcp-direct" {
		t.Fatalf("SetUserMCPToken = %#v, %v", updatedUser, err)
	}
	loadedModule, err := repository.Module(module.ModuleKey)
	if err != nil {
		t.Fatalf("Module: %v", err)
	}
	if len(loadedModule.CategoryIDs) != 1 || loadedModule.CategoryIDs[0] != category.ID {
		t.Fatalf("module categories = %#v", loadedModule.CategoryIDs)
	}
	if loadedModule.DeployToken == "" || !loadedModule.DeployTokenSet {
		t.Fatal("module deploy token was not loaded for internal authentication")
	}
	if len(repository.Entries(module.ModuleKey, version.DocsVersion)) != 1 {
		t.Fatal("entry was not read directly after insert")
	}
	if _, err := repository.VerifyConnectedAppSecret(app.ClientID, "secret"); err != nil {
		t.Fatalf("VerifyConnectedAppSecret: %v", err)
	}

	result, err := repository.IngestArtifact(DeployArtifact{
		ModuleKey: module.ModuleKey, ModuleName: module.Name, DocsVersion: "latest",
		Authors: []string{user.Username}, CommitSHA: "abc123",
		Entries:   []DeployEntry{{Key: "guide", Title: "Guide", Type: "markdown"}},
		Documents: []DeployDocument{{DocID: module.ModuleKey + ":latest:guide", EntryKey: "guide", Title: "Guide", Content: "database-backed content"}},
		Nav:       []NavItem{{Title: "Guide", Path: "/guide"}},
		SiteFiles: map[string][]byte{"site/guide/assets/app.css": []byte("body{}")},
	})
	if err != nil {
		t.Fatalf("IngestArtifact: %v", err)
	}
	if result.PagesIndexed != 1 || len(repository.Releases()) == 0 {
		t.Fatalf("ingest result = %#v", result)
	}
	docID := module.ModuleKey + ":latest:guide"
	page, err := repository.Page(docID)
	if err != nil || page.ContentText != "database-backed content" {
		t.Fatalf("Page = %#v, %v", page, err)
	}
	if _, err := repository.SiteFile(module.ModuleKey, "latest", "guide", "assets/app.css"); err != nil {
		t.Fatalf("SiteFile: %v", err)
	}
	view := repository.RecordPageView(PageView{DocID: docID, UserID: user.ID, SessionID: "session-" + suffix, ReadID: "read-" + suffix})
	view = repository.RecordReadProgress(docID, view.SessionID, view.ReadID, 42, 0.8)
	if view.DurationSeconds != 42 || view.ScrollDepth != 0.8 {
		t.Fatalf("read progress = %#v", view)
	}
	if _, err := repository.SetUserFavorite(user.ID, module.ModuleKey, true); err != nil {
		t.Fatalf("SetUserFavorite: %v", err)
	}
	if _, err := repository.RecordUserRecentDoc(user.ID, UserRecentDoc{DocID: docID}); err != nil {
		t.Fatalf("RecordUserRecentDoc: %v", err)
	}
	if len(repository.UserFavorites(user.ID)) != 1 || len(repository.UserRecentDocs(user.ID, 10)) != 1 {
		t.Fatal("personal activity was not immediately readable")
	}

	grant, err := repository.CreateOAuthCode(app.ID, user.ID, app.RedirectURIs[0], app.Scopes, "code-"+suffix, time.Minute)
	if err != nil {
		t.Fatalf("CreateOAuthCode: %v", err)
	}
	grant, _, _, err = repository.RedeemOAuthCode(app.ClientID, "code-"+suffix, app.RedirectURIs[0], "access-"+suffix, "refresh-"+suffix, time.Minute, time.Hour)
	if err != nil || grant.AccessTokenHash == "" {
		t.Fatalf("RedeemOAuthCode = %#v, %v", grant, err)
	}
	if _, _, _, err = repository.UserByOAuthAccessToken("access-" + suffix); err != nil {
		t.Fatalf("UserByOAuthAccessToken: %v", err)
	}
	if _, _, _, err = repository.RefreshOAuthToken(app.ClientID, "refresh-"+suffix, "access-next-"+suffix, "refresh-next-"+suffix, time.Minute, time.Hour); err != nil {
		t.Fatalf("RefreshOAuthToken: %v", err)
	}
	if !repository.RevokeOAuthToken(app.ClientID, "refresh-next-"+suffix) {
		t.Fatal("RevokeOAuthToken did not update a row")
	}
}
