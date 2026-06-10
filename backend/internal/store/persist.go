package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// snapshot is the serializable projection of the full in-memory store. It is
// used to persist state to disk so the registry survives process restarts until
// the relational (PostgreSQL) backend lands.
type snapshot struct {
	Version    int                  `json:"version"`
	User       User                 `json:"user"`
	Users      []User               `json:"users"`
	Groups     []Group              `json:"groups"`
	Teams      []Team               `json:"teams"`
	Categories []Category           `json:"categories"`
	Modules    []Module             `json:"modules"`
	Versions   []Version            `json:"versions"`
	Entries    []Entry              `json:"entries"`
	Releases   []Release            `json:"releases"`
	Pages      []Page               `json:"pages"`
	SearchLogs []SearchLog          `json:"search_logs"`
	MCPLogs    []MCPLog             `json:"mcp_logs"`
	PageViews  []PageView           `json:"page_views"`
	Navs       map[string][]NavItem `json:"navs"`
	HTML       map[string]string    `json:"html"`
	SiteFiles  map[string]SiteFile  `json:"site_files"`
	Embeddings map[string][]float32 `json:"embeddings"`
	Seq        int64                `json:"seq"`
}

func (s *Store) toSnapshot() snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return snapshot{
		Version:    1,
		User:       s.user,
		Users:      s.users,
		Groups:     s.groups,
		Teams:      s.teams,
		Categories: s.categories,
		Modules:    s.modules,
		Versions:   s.versions,
		Entries:    s.entries,
		Releases:   s.releases,
		Pages:      s.pages,
		SearchLogs: s.searchLogs,
		MCPLogs:    s.mcpLogs,
		PageViews:  s.pageViews,
		Navs:       s.navs,
		HTML:       s.html,
		SiteFiles:  s.siteFiles,
		Embeddings: s.embeddings,
		Seq:        s.seq,
	}
}

func storeFromSnapshot(snap snapshot) *Store {
	s := &Store{
		user:       snap.User,
		users:      snap.Users,
		groups:     snap.Groups,
		teams:      snap.Teams,
		categories: snap.Categories,
		modules:    snap.Modules,
		versions:   snap.Versions,
		entries:    snap.Entries,
		releases:   snap.Releases,
		pages:      snap.Pages,
		searchLogs: snap.SearchLogs,
		mcpLogs:    snap.MCPLogs,
		pageViews:  snap.PageViews,
		navs:       snap.Navs,
		html:       snap.HTML,
		siteFiles:  snap.SiteFiles,
		embeddings: snap.Embeddings,
		seq:        snap.Seq,
	}
	// Ensure slices are never nil (important for empty start and JSON nulls from old snapshots)
	if s.users == nil {
		s.users = []User{}
	}
	if s.groups == nil {
		s.groups = []Group{}
	}
	if s.teams == nil {
		s.teams = []Team{}
	}
	if s.categories == nil {
		s.categories = []Category{}
	}
	if s.modules == nil {
		s.modules = []Module{}
	}
	if s.versions == nil {
		s.versions = []Version{}
	}
	if s.entries == nil {
		s.entries = []Entry{}
	}
	if s.releases == nil {
		s.releases = []Release{}
	}
	if s.pages == nil {
		s.pages = []Page{}
	}
	if s.searchLogs == nil {
		s.searchLogs = []SearchLog{}
	}
	if s.mcpLogs == nil {
		s.mcpLogs = []MCPLog{}
	}
	if s.pageViews == nil {
		s.pageViews = []PageView{}
	}
	if s.navs == nil {
		s.navs = map[string][]NavItem{}
	}
	if s.html == nil {
		s.html = map[string]string{}
	}
	if s.siteFiles == nil {
		s.siteFiles = map[string]SiteFile{}
	}
	if s.embeddings == nil {
		s.embeddings = map[string][]float32{}
	}
	if s.teams == nil {
		s.teams = []Team{}
	}
	return s
}

// Save atomically writes the store snapshot to path (writing to a temp file and
// renaming to avoid a torn file on crash).
func (s *Store) Save(path string) error {
	if path == "" {
		return fmt.Errorf("empty snapshot path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(s.toSnapshot())
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Load reads a store snapshot from path. It returns ErrNotFound when the file
// does not exist so callers can fall back to a freshly seeded store.
func Load(path string) (*Store, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var snap snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("snapshot %s: %w", path, err)
	}
	return storeFromSnapshot(snap), nil
}
