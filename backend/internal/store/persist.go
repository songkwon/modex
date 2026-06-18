package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// snapshot is the serializable projection of the full in-memory store. It is
// used to persist state to disk so the registry survives process restarts until
// the relational (PostgreSQL) backend lands.
type snapshot struct {
	Version      int                  `json:"version"`
	User         User                 `json:"user"`
	Users        []User               `json:"users"`
	Groups       []Group              `json:"groups"`
	Apps         []ConnectedApp       `json:"connected_apps,omitempty"`
	Grants       []OAuthGrant         `json:"oauth_grants,omitempty"`
	Teams        []Team               `json:"teams"`
	Categories   []Category           `json:"categories"`
	Modules      []Module             `json:"modules"`
	ModuleTokens map[string]string    `json:"module_tokens,omitempty"`
	Versions     []Version            `json:"versions"`
	Entries      []Entry              `json:"entries"`
	Releases     []Release            `json:"releases"`
	Pages        []Page               `json:"pages,omitempty"`
	SearchLogs   []SearchLog          `json:"search_logs"`
	MCPLogs      []MCPLog             `json:"mcp_logs"`
	Feedbacks    []DocFeedback        `json:"feedbacks"`
	PageViews    []PageView           `json:"page_views"`
	Navs         map[string][]NavItem `json:"navs"`
	HTML         map[string]string    `json:"html,omitempty"`
	SiteFiles    map[string]SiteFile  `json:"site_files,omitempty"`
	Embeddings   map[string][]float32 `json:"embeddings,omitempty"`
	Settings     Settings             `json:"settings"`
	Seq          int64                `json:"seq"`
}

func (s *Store) toSnapshot(includeLarge bool) snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	moduleTokens := map[string]string{}
	for _, m := range s.modules {
		if m.DeployToken != "" {
			moduleTokens[m.ModuleKey] = m.DeployToken
		}
	}
	snap := snapshot{
		Version:      1,
		User:         s.user,
		Users:        s.users,
		Groups:       s.groups,
		Apps:         s.apps,
		Grants:       s.grants,
		Teams:        s.teams,
		Categories:   s.categories,
		Modules:      s.modules,
		ModuleTokens: moduleTokens,
		Versions:     s.versions,
		Entries:      s.entries,
		Releases:     s.releases,
		Pages:        nil,
		SearchLogs:   s.searchLogs,
		MCPLogs:      s.mcpLogs,
		Feedbacks:    s.feedbacks,
		PageViews:    s.pageViews,
		Navs:         s.navs,
		HTML:         nil,
		SiteFiles:    nil,
		Embeddings:   nil,
		Settings:     s.settings,
		Seq:          s.seq,
	}
	if includeLarge {
		snap.Pages = s.pages
		snap.HTML = s.html
		snap.SiteFiles = s.siteFiles
		snap.Embeddings = s.embeddings
	}
	return snap
}

func storeFromSnapshot(snap snapshot) *Store {
	s := &Store{
		user:       snap.User,
		users:      snap.Users,
		groups:     snap.Groups,
		apps:       snap.Apps,
		grants:     snap.Grants,
		teams:      snap.Teams,
		categories: snap.Categories,
		modules:    snap.Modules,
		versions:   snap.Versions,
		entries:    snap.Entries,
		releases:   snap.Releases,
		pages:      snap.Pages,
		searchLogs: snap.SearchLogs,
		mcpLogs:    snap.MCPLogs,
		feedbacks:  snap.Feedbacks,
		pageViews:  snap.PageViews,
		navs:       snap.Navs,
		html:       snap.HTML,
		siteFiles:  snap.SiteFiles,
		embeddings: snap.Embeddings,
		settings:   snap.Settings,
		seq:        snap.Seq,
	}
	for i := range s.modules {
		if token := snap.ModuleTokens[s.modules[i].ModuleKey]; token != "" {
			s.modules[i].DeployToken = token
			s.modules[i].DeployTokenSet = true
		}
	}
	// Ensure slices are never nil (important for empty start and JSON nulls from old snapshots)
	if s.users == nil {
		s.users = []User{}
	}
	if s.groups == nil {
		s.groups = []Group{}
	}
	if s.apps == nil {
		s.apps = []ConnectedApp{}
	}
	if s.grants == nil {
		s.grants = []OAuthGrant{}
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
	if s.feedbacks == nil {
		s.feedbacks = []DocFeedback{}
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
	s.ensureMarkdownShowcaseSeed()
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
	if err := writeJSONAtomic(path, s.toSnapshot(false)); err != nil {
		return err
	}
	pages, html, siteFiles := s.largeSnapshotParts()
	if err := writeJSONAtomic(splitPath(path, "pages"), pages); err != nil {
		return err
	}
	if err := writeJSONAtomic(splitPath(path, "html"), html); err != nil {
		return err
	}
	if err := writeJSONAtomic(splitPath(path, "site_files"), siteFiles); err != nil {
		return err
	}
	// Embeddings now live in PostgreSQL/pgvector. Remove the legacy sidecar so a
	// later restart cannot load the full vector set back into process memory.
	if err := os.Remove(splitPath(path, "embeddings")); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Load reads a store snapshot from path. It returns ErrNotFound when the file
// does not exist so callers can fall back to a freshly seeded store.
func Load(path string) (*Store, error) {
	return load(path, true)
}

// LoadWithoutEmbeddings skips the legacy vector sidecar. Production uses this
// when PostgreSQL is configured so startup cannot OOM before the old cache is
// cleared from memory.
func LoadWithoutEmbeddings(path string) (*Store, error) {
	return load(path, false)
}

func load(path string, includeEmbeddings bool) (*Store, error) {
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
	// New snapshots keep large, frequently growing payloads in sidecar files.
	// Old single-file snapshots still load because these fields may already be
	// present in the main JSON.
	if err := readJSONIfExists(splitPath(path, "pages"), &snap.Pages); err != nil {
		return nil, err
	}
	if err := readJSONIfExists(splitPath(path, "html"), &snap.HTML); err != nil {
		return nil, err
	}
	if err := readJSONIfExists(splitPath(path, "site_files"), &snap.SiteFiles); err != nil {
		return nil, err
	}
	if includeEmbeddings {
		if err := readJSONIfExists(splitPath(path, "embeddings"), &snap.Embeddings); err != nil {
			return nil, err
		}
	} else {
		snap.Embeddings = nil
	}
	return storeFromSnapshot(snap), nil
}

func (s *Store) largeSnapshotParts() ([]Page, map[string]string, map[string]SiteFile) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Page(nil), s.pages...), cloneStringMap(s.html), cloneSiteFileMap(s.siteFiles)
}

func splitPath(path, suffix string) string {
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	if ext == "" {
		ext = ".json"
	}
	return base + "." + suffix + ext
}

func writeJSONAtomic(path string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func readJSONIfExists(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, v)
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneSiteFileMap(in map[string]SiteFile) map[string]SiteFile {
	if in == nil {
		return nil
	}
	out := make(map[string]SiteFile, len(in))
	for k, v := range in {
		v.Content = append([]byte(nil), v.Content...)
		out[k] = v
	}
	return out
}
