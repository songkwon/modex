package store

import "testing"

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
