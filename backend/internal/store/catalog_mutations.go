package store

import (
	"mime"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (s *MemoryStore) CreateCategory(c Category) (Category, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Key is system-generated from the name (dotted under the parent's key) so
	// users never have to invent one. A user-supplied key is still honored.
	if strings.TrimSpace(c.Key) == "" {
		if strings.TrimSpace(c.Name) == "" {
			return Category{}, ErrInvalid
		}
		c.Key = s.generateCategoryKeyLocked(c.Name, c.ParentID)
	}
	if c.ID == "" {
		c.ID = c.Key
	}
	for _, existing := range s.categories {
		if existing.ID == c.ID {
			return Category{}, ErrConflict
		}
	}
	if c.Status == "" {
		c.Status = "active"
	}
	c.Children = nil
	s.categories = append(s.categories, c)
	return c, nil
}

// slugifyKey lowercases and keeps [a-z0-9-]; non-ASCII (e.g. Chinese) collapses
// to empty, in which case callers fall back to a short unique token.
func slugifyKey(name string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevDash = false
		case r == ' ' || r == '-' || r == '_' || r == '.' || r == '/':
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func (s *MemoryStore) generateCategoryKeyLocked(name, parentID string) string {
	base := slugifyKey(name)
	if base == "" {
		base = "d" + strconv.FormatInt(time.Now().UnixNano()%1_000_000, 36)
	}
	prefix := ""
	for _, c := range s.categories {
		if c.ID == parentID {
			prefix = c.Key + "."
			break
		}
	}
	taken := func(k string) bool {
		for _, c := range s.categories {
			if c.Key == k || c.ID == k {
				return true
			}
		}
		return false
	}
	key := prefix + base
	for i := 2; taken(key); i++ {
		key = prefix + base + "-" + strconv.Itoa(i)
	}
	return key
}

func (s *MemoryStore) generateTeamKeyLocked(name string) string {
	base := slugifyKey(name)
	if base == "" {
		base = "team-" + strconv.FormatInt(time.Now().UnixNano()%1_000_000, 36)
	}
	taken := func(k string) bool {
		for _, t := range s.teams {
			if strings.EqualFold(t.Key, k) {
				return true
			}
		}
		return false
	}
	key := base
	for i := 2; taken(key); i++ {
		key = base + "-" + strconv.Itoa(i)
	}
	return key
}

// MoveCategory reparents a category and positions it at `index` among its new
// siblings, renumbering sibling SortOrder so the tree order is stable. Rejects
// moves that would create a cycle (into the node's own subtree).
func (s *MemoryStore) MoveCategory(id, parentID string, index int) (Category, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i := range s.categories {
		if s.categories[i].ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return Category{}, ErrNotFound
	}
	if parentID == id {
		return Category{}, ErrInvalid
	}
	// Walk parent chain to reject cycles.
	for p := parentID; p != ""; {
		if p == id {
			return Category{}, ErrInvalid
		}
		next := ""
		for i := range s.categories {
			if s.categories[i].ID == p {
				next = s.categories[i].ParentID
				break
			}
		}
		p = next
	}
	if parentID != "" {
		found := false
		for i := range s.categories {
			if s.categories[i].ID == parentID {
				found = true
				break
			}
		}
		if !found {
			return Category{}, ErrInvalid
		}
	}

	s.categories[idx].ParentID = parentID

	// Collect new siblings (same parent) in current order, excluding the moved
	// node, then insert it at the requested index and renumber.
	var sibs []int
	for i := range s.categories {
		if s.categories[i].ParentID == parentID && s.categories[i].ID != id {
			sibs = append(sibs, i)
		}
	}
	sort.SliceStable(sibs, func(a, b int) bool { return s.categories[sibs[a]].SortOrder < s.categories[sibs[b]].SortOrder })
	order := make([]int, 0, len(sibs)+1)
	if index < 0 {
		index = 0
	}
	if index > len(sibs) {
		index = len(sibs)
	}
	order = append(order, sibs[:index]...)
	order = append(order, idx)
	order = append(order, sibs[index:]...)
	for pos, ci := range order {
		s.categories[ci].SortOrder = (pos + 1) * 10
	}
	out := s.categories[idx]
	out.Children = nil
	return out, nil
}

func (s *MemoryStore) UpdateCategory(id string, c Category) (Category, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.categories {
		if s.categories[i].ID == id {
			if c.Name != "" {
				s.categories[i].Name = c.Name
			}
			if c.Description != "" {
				s.categories[i].Description = c.Description
			}
			if c.Icon != "" {
				s.categories[i].Icon = c.Icon
			}
			if c.SortOrder != 0 {
				s.categories[i].SortOrder = c.SortOrder
			}
			if c.Status != "" {
				s.categories[i].Status = c.Status
			}
			if c.ParentID != "" {
				s.categories[i].ParentID = c.ParentID
			}
			// Always accept ResponsibleTeam from patch (send "" explicitly to clear assignment to a team).
			s.categories[i].ResponsibleTeam = c.ResponsibleTeam
			out := s.categories[i]
			out.Children = nil
			return out, nil
		}
	}
	return Category{}, ErrNotFound
}

func (s *MemoryStore) DeleteCategory(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.categories {
		if c.ParentID == id {
			return ErrConflict
		}
	}
	for i := range s.categories {
		if s.categories[i].ID == id {
			s.categories = append(s.categories[:i], s.categories[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

func (s *MemoryStore) moduleKeyTakenLocked(key string) bool {
	for _, m := range s.modules {
		if strings.EqualFold(m.ModuleKey, key) {
			return true
		}
	}
	return false
}

func (s *MemoryStore) CreateModule(m Module) (Module, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Auto-generate module_key from the name (slug, unique) so admins never type one.
	if strings.TrimSpace(m.ModuleKey) == "" {
		if strings.TrimSpace(m.Name) == "" {
			return Module{}, ErrInvalid
		}
		base := slugifyKey(m.Name)
		if base == "" {
			base = "doc-" + strconv.FormatInt(time.Now().UnixNano()%1_000_000, 36)
		}
		key := base
		for i := 2; s.moduleKeyTakenLocked(key); i++ {
			key = base + "-" + strconv.Itoa(i)
		}
		m.ModuleKey = key
	}
	for _, existing := range s.modules {
		if strings.EqualFold(existing.ModuleKey, m.ModuleKey) {
			return Module{}, ErrConflict
		}
	}
	if m.ID == "" {
		m.ID = s.nextIDLocked("m")
	}
	// Every doc source gets a deploy token for CI push auth.
	if strings.TrimSpace(m.DeployToken) == "" {
		m.DeployToken = "mdx_" + strconv.FormatInt(time.Now().UnixNano(), 36) + strconv.FormatInt(int64(len(s.modules)+1), 36)
	}
	if m.Name == "" {
		m.Name = m.ModuleKey
	}
	if m.Status == "" {
		m.Status = "active"
	}
	if m.DefaultVersion == "" {
		m.DefaultVersion = "latest"
	}
	m.UpdatedAt = time.Now().UTC()
	m.AvailableVers = nil
	m.DeployTokenSet = m.DeployToken != ""
	s.modules = append(s.modules, m)
	return m, nil
}

func (s *MemoryStore) UpdateModule(moduleKey string, patch Module) (Module, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.modules {
		if strings.EqualFold(s.modules[i].ModuleKey, moduleKey) {
			m := &s.modules[i]
			if patch.Name != "" {
				m.Name = patch.Name
			}
			if patch.Description != "" {
				m.Description = patch.Description
			}
			if patch.OwnerGroup != "" {
				m.OwnerGroup = patch.OwnerGroup
			}
			if patch.RepoType != "" {
				m.RepoType = patch.RepoType
			}
			if patch.RepoURL != "" {
				m.RepoURL = patch.RepoURL
			}
			if patch.DefaultVersion != "" {
				m.DefaultVersion = patch.DefaultVersion
			}
			if patch.Visibility != "" {
				m.Visibility = patch.Visibility
			}
			if patch.Status != "" {
				m.Status = patch.Status
			}
			if patch.PackageVersion != "" {
				m.PackageVersion = patch.PackageVersion
			}
			if patch.Channel != "" {
				m.Channel = patch.Channel
			}
			if patch.Edition != "" {
				m.Edition = patch.Edition
			}
			if patch.Keywords != nil {
				m.Keywords = patch.Keywords
			}
			if patch.Maintainers != nil {
				m.Maintainers = patch.Maintainers
			}
			if patch.CategoryIDs != nil {
				m.CategoryIDs = patch.CategoryIDs
			}
			if patch.CategoryPath != "" {
				m.CategoryPath = patch.CategoryPath
			}
			if patch.SourceType != "" {
				m.SourceType = patch.SourceType
			}
			if patch.DocType != "" {
				m.DocType = patch.DocType
			}
			if patch.Mount != "" {
				m.Mount = patch.Mount
			}
			if patch.GitLabBranch != "" {
				m.GitLabBranch = patch.GitLabBranch
			}
			if patch.GitLabPath != "" {
				m.GitLabPath = patch.GitLabPath
			}
			if patch.DeployToken != "" {
				m.DeployToken = patch.DeployToken
			}
			m.UpdatedAt = time.Now().UTC()
			out := *m
			out.AvailableVers = s.versionsForLocked(m.ModuleKey)
			out.DeployTokenSet = out.DeployToken != ""
			return out, nil
		}
	}
	return Module{}, ErrNotFound
}

func (s *MemoryStore) CreateVersion(moduleKey string, v Version) (Version, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.moduleIndexLocked(moduleKey); err != nil {
		return Version{}, err
	}
	if strings.TrimSpace(v.DocsVersion) == "" {
		return Version{}, ErrInvalid
	}
	for _, existing := range s.versions {
		if strings.EqualFold(existing.ModuleKey, moduleKey) && existing.DocsVersion == v.DocsVersion {
			return Version{}, ErrConflict
		}
	}
	v.ModuleKey = moduleKey
	if v.ID == "" {
		v.ID = s.nextIDLocked("v")
	}
	if v.DisplayName == "" {
		v.DisplayName = v.DocsVersion
	}
	if v.Status == "" {
		v.Status = "active"
	}
	v.CreatedAt = time.Now().UTC()
	if v.IsDefault {
		for i := range s.versions {
			if strings.EqualFold(s.versions[i].ModuleKey, moduleKey) {
				s.versions[i].IsDefault = false
			}
		}
		if idx, err := s.moduleIndexLocked(moduleKey); err == nil {
			s.modules[idx].DefaultVersion = v.DocsVersion
		}
	}
	s.versions = append(s.versions, v)
	return v, nil
}

func (s *MemoryStore) UpdateVersion(moduleKey, docsVersion string, patch Version) (Version, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.versions {
		if strings.EqualFold(s.versions[i].ModuleKey, moduleKey) && s.versions[i].DocsVersion == docsVersion {
			v := &s.versions[i]
			if patch.DisplayName != "" {
				v.DisplayName = patch.DisplayName
			}
			if patch.VersionType != "" {
				v.VersionType = patch.VersionType
			}
			if patch.Status != "" {
				v.Status = patch.Status
			}
			if patch.SourceBranch != "" {
				v.SourceBranch = patch.SourceBranch
			}
			if patch.PackageVersion != "" {
				v.PackageVersion = patch.PackageVersion
			}
			if patch.Channel != "" {
				v.Channel = patch.Channel
			}
			if patch.Edition != "" {
				v.Edition = patch.Edition
			}
			if patch.SupportStatus != "" {
				v.SupportStatus = patch.SupportStatus
			}
			if patch.IsDefault {
				for j := range s.versions {
					if strings.EqualFold(s.versions[j].ModuleKey, moduleKey) {
						s.versions[j].IsDefault = false
					}
				}
				v.IsDefault = true
				if idx, err := s.moduleIndexLocked(moduleKey); err == nil {
					s.modules[idx].DefaultVersion = v.DocsVersion
				}
			}
			return *v, nil
		}
	}
	return Version{}, ErrNotFound
}

func (s *MemoryStore) CreateEntry(moduleKey, docsVersion string, e Entry) (Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.moduleIndexLocked(moduleKey); err != nil {
		return Entry{}, err
	}
	if strings.TrimSpace(e.EntryKey) == "" {
		return Entry{}, ErrInvalid
	}
	for _, existing := range s.entries {
		if strings.EqualFold(existing.ModuleKey, moduleKey) && existing.DocsVersion == docsVersion && existing.EntryKey == e.EntryKey {
			return Entry{}, ErrConflict
		}
	}
	e.ModuleKey = moduleKey
	e.DocsVersion = docsVersion
	if e.ID == "" {
		e.ID = s.nextIDLocked("e")
	}
	if e.EntryType == "" {
		e.EntryType = "markdown"
	}
	if e.Builder == "" {
		e.Builder = e.EntryType
	}
	if e.IndexStatus == "" {
		e.IndexStatus = "pending"
	}
	if e.Status == "" {
		e.Status = "active"
	}
	e.CreatedAt = time.Now().UTC()
	s.entries = append(s.entries, e)
	return e, nil
}

func (s *MemoryStore) UpdateEntry(entryID string, patch Entry) (Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.entries {
		if s.entries[i].ID == entryID {
			e := &s.entries[i]
			if patch.Title != "" {
				e.Title = patch.Title
			}
			if patch.EntryType != "" {
				e.EntryType = patch.EntryType
			}
			if patch.Builder != "" {
				e.Builder = patch.Builder
			}
			if patch.Source != "" {
				e.Source = patch.Source
			}
			if patch.StorageURI != "" {
				e.StorageURI = patch.StorageURI
			}
			if patch.NavURI != "" {
				e.NavURI = patch.NavURI
			}
			if patch.IndexStatus != "" {
				e.IndexStatus = patch.IndexStatus
			}
			if patch.SortOrder != 0 {
				e.SortOrder = patch.SortOrder
			}
			if patch.Status != "" {
				e.Status = patch.Status
			}
			e.IsPrimary = patch.IsPrimary
			return *e, nil
		}
	}
	return Entry{}, ErrNotFound
}

func (s *MemoryStore) DeleteEntry(entryID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.entries {
		if s.entries[i].ID == entryID {
			s.entries = append(s.entries[:i], s.entries[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

func (s *MemoryStore) Release(releaseID string) (Release, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, r := range s.releases {
		if r.ReleaseID == releaseID || r.ID == releaseID {
			return r, nil
		}
	}
	return Release{}, ErrNotFound
}

// RollbackRelease marks the target release as rolled back. A real
// implementation would also re-point storage and search to the prior artifact.
func (s *MemoryStore) RollbackRelease(releaseID string) (Release, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.releases {
		if s.releases[i].ReleaseID == releaseID || s.releases[i].ID == releaseID {
			s.releases[i].Status = "rolled_back"
			return s.releases[i], nil
		}
	}
	return Release{}, ErrNotFound
}

func (s *MemoryStore) moduleIndexLocked(moduleKey string) (int, error) {
	for i := range s.modules {
		if strings.EqualFold(s.modules[i].ModuleKey, moduleKey) {
			return i, nil
		}
	}
	return -1, ErrNotFound
}

func (s *MemoryStore) nextIDLocked(prefix string) string {
	s.seq++
	return prefix + "-" + strconv.FormatInt(s.seq, 10) + "-" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

func (s *MemoryStore) versionsForLocked(moduleKey string) []Version {
	var out []Version
	for _, v := range s.versions {
		if strings.EqualFold(v.ModuleKey, moduleKey) {
			out = append(out, v)
		}
	}
	return out
}

func contains(xs []string, target string) bool {
	for _, x := range xs {
		if x == target {
			return true
		}
	}
	return false
}

func routeKey(moduleKey, docsVersion, entryKey string) string {
	if entryKey == "" {
		return strings.ToLower(moduleKey) + ":" + docsVersion
	}
	return strings.ToLower(moduleKey) + ":" + docsVersion + ":" + entryKey
}

func siteFileKey(moduleKey, docsVersion, entryKey, name string) string {
	return routeKey(moduleKey, docsVersion, entryKey) + ":" + path.Clean(strings.TrimPrefix(name, "/"))
}

// isSiteBuilderType reports whether an entry ships a pre-built static site
// (VitePress/VuePress/Fumadocs) rather than Markdown rendered by Modex.
func isSiteBuilderType(entryType string) bool {
	switch strings.ToLower(entryType) {
	case "vitepress", "vuepress", "fumadocs":
		return true
	default:
		return false
	}
}

// docPagePath builds the in-app URL for an indexed page. For site-builder
// entries it preserves the per-page route as a ?p= deep link so a search hit
// opens the matched page inside the embedded site, instead of collapsing every
// page to the entry root. Markdown/static entries keep the plain entry URL.
func docPagePath(moduleKey, docsVersion, entryKey, entryType, route string) string {
	base := "/docs/" + moduleKey + "/" + docsVersion + "/" + entryKey
	route = strings.TrimSpace(route)
	if isSiteBuilderType(entryType) && route != "" && route != "/" && !strings.HasPrefix(route, "/docs/") {
		return base + "?p=" + url.QueryEscape(route)
	}
	return base
}

func cloneStrings(xs []string) []string {
	if xs == nil {
		return nil
	}
	return append([]string(nil), xs...)
}

func cloneNav(xs []NavItem) []NavItem {
	if xs == nil {
		return nil
	}
	out := make([]NavItem, len(xs))
	for i, x := range xs {
		out[i] = x
		out[i].Children = cloneNav(x.Children)
	}
	return out
}

func sortOrderForEntryLocked(entries []Entry, moduleKey, docsVersion, entryKey string) int {
	for _, e := range entries {
		if strings.EqualFold(e.ModuleKey, moduleKey) && e.DocsVersion == docsVersion && e.EntryKey == entryKey {
			return e.SortOrder
		}
	}
	return int(^uint(0) >> 1)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func firstString(xs []string) string {
	if len(xs) == 0 {
		return ""
	}
	return xs[0]
}

func coalesceStrings(primary, fallback []string) []string {
	if len(primary) > 0 {
		return primary
	}
	return fallback
}

func removeEntries(entries []Entry, moduleKey, docsVersion string) []Entry {
	out := entries[:0]
	for _, e := range entries {
		if strings.EqualFold(e.ModuleKey, moduleKey) && e.DocsVersion == docsVersion {
			continue
		}
		out = append(out, e)
	}
	return out
}

func removePages(pages []Page, moduleKey, docsVersion string) []Page {
	out := pages[:0]
	for _, p := range pages {
		if strings.EqualFold(p.ModuleKey, moduleKey) && p.DocsVersion == docsVersion {
			continue
		}
		out = append(out, p)
	}
	return out
}

func entryKeyFromDocID(docID string) string {
	parts := strings.Split(docID, ":")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func entryTypeForEntry(entries []DeployEntry, entryKey string) string {
	for _, e := range entries {
		if e.Key == entryKey {
			return e.Type
		}
	}
	return "markdown"
}

func titleForEntry(entries []DeployEntry, entryKey string) string {
	for _, e := range entries {
		if e.Key == entryKey {
			return e.Title
		}
	}
	return entryKey
}

func htmlForEntry(files map[string]string, entryKey string) string {
	for _, name := range []string{"site/" + entryKey + "/index.html", "site/" + entryKey + ".html"} {
		if html := files[name]; html != "" {
			return html
		}
	}
	prefix := "site/" + entryKey + "/"
	for name, html := range files {
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, ".html") {
			return html
		}
	}
	return ""
}

func splitSiteFile(name string) (string, string, bool) {
	name = path.Clean(strings.TrimPrefix(name, "/"))
	parts := strings.Split(name, "/")
	if len(parts) < 3 || parts[0] != "site" || parts[1] == "" {
		return "", "", false
	}
	rel := path.Join(parts[2:]...)
	if rel == "." || rel == "" {
		rel = "index.html"
	}
	return parts[1], rel, true
}

func contentTypeForName(name string, content []byte) string {
	if ct := mime.TypeByExtension(path.Ext(name)); ct != "" {
		return ct
	}
	if len(content) > 0 {
		return http.DetectContentType(content)
	}
	return "application/octet-stream"
}
