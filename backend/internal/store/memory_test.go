package store

import (
	"path/filepath"
	"testing"
)

func TestSnapshotRoundTrip(t *testing.T) {
	s := NewSeeded()
	if _, err := s.CreateUser(User{Username: "carol", Roles: []string{"viewer"}}); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	s.RecordPageView(PageView{DocID: "DemoModule:latest:guide", SessionID: "x"})
	s.SetEmbedding("DemoModule:latest:guide", []float32{0.1, 0.2, 0.3})

	path := filepath.Join(t.TempDir(), "snap.json")
	if err := s.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got, want := len(loaded.Users("")), len(s.Users("")); got != want {
		t.Fatalf("users after reload = %d, want %d", got, want)
	}
	if _, err := loaded.UserByID(""); err == nil {
		t.Fatal("expected lookup of empty id to fail")
	}
	if loaded.EmbeddingCount() != 1 {
		t.Fatalf("embeddings after reload = %d, want 1", loaded.EmbeddingCount())
	}
	stats := loaded.PageAnalytics()
	var pv int
	for _, st := range stats {
		if st.DocID == "DemoModule:latest:guide" {
			pv = st.PV
		}
	}
	if pv != 1 {
		t.Fatalf("page view after reload = %d, want 1", pv)
	}
	// Verify new IDs continue past the persisted sequence (no collisions).
	created, err := loaded.CreateModule(Module{ModuleKey: "PersistedModule"})
	if err != nil {
		t.Fatalf("CreateModule after reload: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected generated module ID after reload")
	}
}

func TestLoadMissingReturnsNotFound(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nope.json")); err != ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestUserCRUDAndGroupAutoRegister(t *testing.T) {
	s := NewSeeded()

	created, err := s.CreateUser(User{Username: "carol", Department: "测试", Groups: []string{"qa-team"}, Roles: []string{"viewer"}})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if created.DisplayName != "carol" || created.Source != "manual" || created.Status != "active" {
		t.Fatalf("CreateUser defaults wrong: %+v", created)
	}
	if _, err := s.CreateUser(User{Username: "CAROL"}); err != ErrConflict {
		t.Fatalf("duplicate username err = %v, want ErrConflict", err)
	}
	// qa-team should have been auto-registered as a group.
	var hasQA bool
	for _, g := range s.Groups() {
		if g.GroupKey == "qa-team" {
			hasQA = true
		}
	}
	if !hasQA {
		t.Fatal("expected qa-team group to be auto-registered")
	}

	updated, err := s.UpdateUser(created.ID, User{Roles: []string{"maintainer"}, Status: "disabled"})
	if err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}
	if len(updated.Roles) != 1 || updated.Roles[0] != "maintainer" || updated.Status != "disabled" {
		t.Fatalf("UpdateUser result wrong: %+v", updated)
	}

	if err := s.DeleteUser(created.ID); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	if _, err := s.UserByID(created.ID); err != ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestUpsertUserSyncsOnLogin(t *testing.T) {
	s := NewSeeded()
	before := len(s.Users(""))

	// Existing seeded user alice: upsert should update, not duplicate.
	u := s.UpsertUser(User{ID: "u-alice", Username: "alice", Department: "新部门", Groups: []string{"cad-team", "release"}})
	if u.Source != "oidc" || u.Department != "新部门" {
		t.Fatalf("upsert existing wrong: %+v", u)
	}
	if u.LastLoginAt.IsZero() {
		t.Fatal("expected LastLoginAt to be set on login")
	}
	if got := len(s.Users("")); got != before {
		t.Fatalf("user count changed on upsert-existing: %d -> %d", before, got)
	}

	// New identity from provider: should be created.
	s.UpsertUser(User{Username: "dave", Email: "dave@example.com", Groups: []string{"ops"}})
	if got := len(s.Users("")); got != before+1 {
		t.Fatalf("expected new user added, count %d -> %d", before, got)
	}
}

func TestPageAnalyticsAggregatesViews(t *testing.T) {
	s := NewSeeded()
	doc := "DemoModule:latest:guide"

	s.RecordPageView(PageView{DocID: doc, SessionID: "a", DurationSeconds: 10})
	s.RecordPageView(PageView{DocID: doc, SessionID: "a"})
	s.RecordPageView(PageView{DocID: doc, SessionID: "b"})
	s.RecordReadProgress(doc, "b", 30, 0.9)

	var found bool
	for _, st := range s.PageAnalytics() {
		if st.DocID != doc {
			continue
		}
		found = true
		if st.PV != 3 {
			t.Fatalf("PV = %d, want 3", st.PV)
		}
		if st.UV != 2 {
			t.Fatalf("UV = %d, want 2", st.UV)
		}
		if st.Reads7d != 3 {
			t.Fatalf("Reads7d = %d, want 3", st.Reads7d)
		}
		if st.AvgDurationSec != 20 { // (10 + 30) / 2 views with duration
			t.Fatalf("AvgDurationSec = %d, want 20", st.AvgDurationSec)
		}
	}
	if !found {
		t.Fatalf("page %s not present in analytics", doc)
	}
}

func TestPageAnalyticsFallsBackToSeedReads(t *testing.T) {
	s := NewSeeded()
	for _, st := range s.PageAnalytics() {
		if st.DocID == "CBB:latest:build-cache" {
			if st.Reads30d == 0 {
				t.Fatalf("expected seeded reads_30d fallback for unviewed page, got 0")
			}
			return
		}
	}
	t.Fatal("CBB page missing from analytics")
}

func TestModuleVersionEntryCRUD(t *testing.T) {
	s := NewSeeded()

	if _, err := s.CreateModule(Module{ModuleKey: "NCKernel", CategoryIDs: []string{"nc"}}); err != nil {
		t.Fatalf("CreateModule: %v", err)
	}
	if _, err := s.CreateModule(Module{ModuleKey: "nckernel"}); err != ErrConflict {
		t.Fatalf("duplicate module err = %v, want ErrConflict", err)
	}

	v, err := s.CreateVersion("NCKernel", Version{DocsVersion: "v1.0", IsDefault: true})
	if err != nil {
		t.Fatalf("CreateVersion: %v", err)
	}
	if v.DisplayName != "v1.0" {
		t.Fatalf("version display name = %q, want v1.0", v.DisplayName)
	}
	m, _ := s.Module("NCKernel")
	if m.DefaultVersion != "v1.0" {
		t.Fatalf("default version not updated, got %q", m.DefaultVersion)
	}

	e, err := s.CreateEntry("NCKernel", "v1.0", Entry{EntryKey: "guide", Title: "指南"})
	if err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}
	if e.Builder != "markdown" {
		t.Fatalf("entry builder default = %q, want markdown", e.Builder)
	}
	if got := s.Entries("NCKernel", "v1.0"); len(got) != 1 {
		t.Fatalf("entries count = %d, want 1", len(got))
	}
	if err := s.DeleteEntry(e.ID); err != nil {
		t.Fatalf("DeleteEntry: %v", err)
	}
	if got := s.Entries("NCKernel", "v1.0"); len(got) != 0 {
		t.Fatalf("entries after delete = %d, want 0", len(got))
	}
}

func TestVersionRequiresExistingModule(t *testing.T) {
	s := NewSeeded()
	if _, err := s.CreateVersion("Ghost", Version{DocsVersion: "latest"}); err != ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestRollbackRelease(t *testing.T) {
	s := NewSeeded()
	rel, err := s.RollbackRelease("rel-demo-latest-001")
	if err != nil {
		t.Fatalf("RollbackRelease: %v", err)
	}
	if rel.Status != "rolled_back" {
		t.Fatalf("status = %q, want rolled_back", rel.Status)
	}
	if _, err := s.RollbackRelease("missing"); err != ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestIngestArtifactPublishesPagesNavAndHTML(t *testing.T) {
	s := NewSeeded()
	result, err := s.IngestArtifact(DeployArtifact{
		ModuleKey:      "RuntimeDocs",
		ModuleName:     "Runtime Docs",
		DocsVersion:    "latest",
		PackageVersion: "0.1.0",
		Description:    "Runtime imported documentation.",
		Authors:        []string{"platform"},
		Keywords:       []string{"runtime", "docs"},
		Entries: []DeployEntry{{
			Key: "guide", Title: "Guide", Type: "markdown", Source: "docs/guide.md",
		}},
		Documents: []DeployDocument{{
			DocID: "RuntimeDocs:latest:guide", EntryKey: "guide", EntryType: "markdown", Title: "Guide",
			Description: "Guide page", Content: "runtime documentation content", Keywords: []string{"runtime"}, Status: "active",
		}},
		Nav:      []NavItem{{Title: "Guide", Path: "/guide"}},
		SiteHTML: map[string]string{"site/guide/index.html": "<main><h1>Guide</h1></main>"},
		SiteFiles: map[string][]byte{
			"site/guide/index.html":     []byte("<main><h1>Guide</h1></main>"),
			"site/guide/assets/app.css": []byte("body{color:#111}"),
		},
		Bytes: 123,
	})
	if err != nil {
		t.Fatalf("IngestArtifact: %v", err)
	}
	if result.PagesIndexed != 1 || result.EntriesIndexed != 1 || result.HTMLFiles != 1 || result.SiteFiles != 2 {
		t.Fatalf("unexpected deploy result: %+v", result)
	}
	page, err := s.PageByRoute("RuntimeDocs", "latest", "guide")
	if err != nil {
		t.Fatalf("PageByRoute: %v", err)
	}
	if page.ContentText != "runtime documentation content" {
		t.Fatalf("content = %q", page.ContentText)
	}
	if got := s.PageHTML("RuntimeDocs", "latest", "guide"); got == "" {
		t.Fatal("expected stored html")
	}
	if got := s.Nav("RuntimeDocs", "latest"); len(got) != 1 || got[0].Title != "Guide" {
		t.Fatalf("unexpected nav: %+v", got)
	}
	file, err := s.SiteFile("RuntimeDocs", "latest", "guide", "assets/app.css")
	if err != nil {
		t.Fatalf("SiteFile: %v", err)
	}
	if file.ContentType != "text/css; charset=utf-8" {
		t.Fatalf("content type = %q", file.ContentType)
	}
}
