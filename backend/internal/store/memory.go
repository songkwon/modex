package store

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	mu         sync.RWMutex
	user       User
	categories []Category
	modules    []Module
	versions   []Version
	entries    []Entry
	releases   []Release
	pages      []Page
	searchLogs []SearchLog
	mcpLogs    []MCPLog
}

func NewSeeded() *Store {
	now := time.Now().UTC()
	s := &Store{
		user: User{ID: "u-dev", Username: "dev", DisplayName: "研发用户", Email: "dev@example.com", Department: "工程化", Groups: []string{"cad-team", "engineering"}, Roles: []string{"admin"}},
		categories: []Category{
			{ID: "engineering", Key: "engineering", Name: "工程化", Description: "研发效能、构建、CI/CD 与质量平台", Icon: "wrench", SortOrder: 10, Status: "active"},
			{ID: "engineering.cbb", ParentID: "engineering", Key: "engineering.cbb", Name: "CBB", Description: "CBB 构建与模块治理", Icon: "package", SortOrder: 11, Status: "active"},
			{ID: "cad", Key: "cad", Name: "CAD", Description: "CAD 内核、插件与图形渲染", Icon: "box", SortOrder: 20, Status: "active"},
			{ID: "cad.demo", ParentID: "cad", Key: "cad.demo", Name: "示例模块", Description: "示例与接入指南", Icon: "book", SortOrder: 21, Status: "active"},
			{ID: "app", Key: "app", Name: "应用", Description: "业务应用、订单、设备联网", Icon: "layers", SortOrder: 30, Status: "active"},
		},
		modules: []Module{
			{ID: "m-demo", ModuleKey: "DemoModule", Name: "DemoModule", Description: "示例模块文档，覆盖模块落地、维护和发布流程。", OwnerGroup: "cad-team", RepoType: "gitlab", RepoURL: "https://gitlab.example.com/cad/demo-module", DefaultVersion: "latest", Visibility: "internal", Status: "active", PackageName: "DemoModule", PackageVersion: "1.2.3", Channel: "default", Edition: "2025", Keywords: []string{"demo", "cad", "markdown"}, Maintainers: []string{"alice", "bob"}, CategoryIDs: []string{"cad", "cad.demo"}, CategoryPath: "CAD / 示例模块", UpdatedAt: now, Reads7d: 128, Reads30d: 520},
			{ID: "m-cbb", ModuleKey: "CBB", Name: "CBB 文档", Description: "工程化模块构建、依赖分析、发布规范与常见问题。", OwnerGroup: "engineering", RepoType: "gitlab", RepoURL: "https://gitlab.example.com/devops/cbb", DefaultVersion: "latest", Visibility: "internal", Status: "active", PackageName: "CBB", PackageVersion: "2.8.0", Channel: "stable", Edition: "2025", Keywords: []string{"cbb", "ci", "build"}, Maintainers: []string{"platform-team"}, CategoryIDs: []string{"engineering", "engineering.cbb"}, CategoryPath: "工程化 / CBB", UpdatedAt: now.Add(-4 * time.Hour), Reads7d: 342, Reads30d: 1430},
		},
	}
	s.versions = []Version{
		{ID: "v-demo-latest", ModuleKey: "DemoModule", DocsVersion: "latest", DisplayName: "latest", VersionType: "branch", IsDefault: true, Status: "active", SourceBranch: "main", PackageVersion: "1.2.3", Channel: "default", Edition: "2025", SupportStatus: "supported", CreatedAt: now},
		{ID: "v-demo-v12", ModuleKey: "DemoModule", DocsVersion: "v1.2", DisplayName: "v1.2", VersionType: "release", IsDefault: false, Status: "active", SourceBranch: "release/1.2", PackageVersion: "1.2.0", Channel: "default", Edition: "2025", SupportStatus: "supported", CreatedAt: now.AddDate(0, -1, 0)},
		{ID: "v-cbb-latest", ModuleKey: "CBB", DocsVersion: "latest", DisplayName: "latest", VersionType: "branch", IsDefault: true, Status: "active", SourceBranch: "main", PackageVersion: "2.8.0", Channel: "stable", Edition: "2025", SupportStatus: "supported", CreatedAt: now},
	}
	s.entries = []Entry{
		{ID: "e-demo-guide", ModuleKey: "DemoModule", DocsVersion: "latest", EntryKey: "guide", Title: "模块落地指导", EntryType: "markdown", Builder: "markdown", Source: "docs/integration-guide.md", StorageURI: "minio://modex/DemoModule/latest/site/guide/index.html", NavURI: "minio://modex/DemoModule/latest/nav.json", IndexStatus: "indexed", IsPrimary: true, SortOrder: 1, Status: "active", CreatedAt: now},
		{ID: "e-demo-maintenance", ModuleKey: "DemoModule", DocsVersion: "latest", EntryKey: "maintenance", Title: "模块维护说明", EntryType: "markdown", Builder: "markdown", Source: "docs/maintenance-guide.md", StorageURI: "minio://modex/DemoModule/latest/site/maintenance/index.html", NavURI: "minio://modex/DemoModule/latest/nav.json", IndexStatus: "indexed", SortOrder: 2, Status: "active", CreatedAt: now},
		{ID: "e-cbb-build", ModuleKey: "CBB", DocsVersion: "latest", EntryKey: "build-cache", Title: "构建缓存清理", EntryType: "markdown", Builder: "markdown", Source: "docs/build-cache.md", StorageURI: "minio://modex/CBB/latest/site/build-cache/index.html", NavURI: "minio://modex/CBB/latest/nav.json", IndexStatus: "indexed", IsPrimary: true, SortOrder: 1, Status: "active", CreatedAt: now},
	}
	s.pages = []Page{
		{ID: "p-demo-guide", DocID: "DemoModule:latest:guide", ModuleKey: "DemoModule", ModuleName: "DemoModule", DocsVersion: "latest", PackageVersion: "1.2.3", EntryKey: "guide", EntryType: "markdown", Title: "模块落地指导", Description: "面向业务开发人员的模块接入、部署、接口和异常处理说明。", Path: "/docs/DemoModule/latest/guide", SourceFile: "docs/integration-guide.md", DocType: "markdown", Status: "active", OwnerGroup: "cad-team", CategoryIDs: []string{"cad", "cad.demo"}, Tags: []string{"demo", "cad"}, ContentText: "模块落地指导说明如何接入 DemoModule，包括接口设计、部署运行、异常处理、风险影响面和发布检查。", UpdatedAt: now},
		{ID: "p-demo-maintenance", DocID: "DemoModule:latest:maintenance", ModuleKey: "DemoModule", ModuleName: "DemoModule", DocsVersion: "latest", PackageVersion: "1.2.3", EntryKey: "maintenance", EntryType: "markdown", Title: "模块维护说明", Description: "面向维护开发人员的架构、设计、流程和维护说明。", Path: "/docs/DemoModule/latest/maintenance", SourceFile: "docs/maintenance-guide.md", DocType: "markdown", Status: "active", OwnerGroup: "cad-team", CategoryIDs: []string{"cad", "cad.demo"}, Tags: []string{"demo", "cad", "architecture"}, ContentText: "模块维护说明包含总体架构、设计原则、模块结构、核心流程、时序逻辑、前后端设计和质量可维护性要求。", UpdatedAt: now},
		{ID: "p-cbb-build", DocID: "CBB:latest:build-cache", ModuleKey: "CBB", ModuleName: "CBB 文档", DocsVersion: "latest", PackageVersion: "2.8.0", EntryKey: "build-cache", EntryType: "markdown", Title: "构建缓存清理", Description: "CBB 构建缓存清理和常见构建问题排查。", Path: "/docs/CBB/latest/build-cache", SourceFile: "docs/build-cache.md", DocType: "markdown", Status: "active", OwnerGroup: "engineering", CategoryIDs: []string{"engineering", "engineering.cbb"}, Tags: []string{"cbb", "ci", "build"}, ContentText: "构建缓存清理用于解决依赖缓存、编译缓存和 CI 工作区残留导致的构建异常。可以重新拉取依赖并清理本地缓存。", UpdatedAt: now},
	}
	s.releases = []Release{{ID: "r-demo-1", ReleaseID: "rel-demo-latest-001", ModuleKey: "DemoModule", DocsVersion: "latest", CommitSHA: "d34db33f", Branch: "main", Publisher: "alice", PipelineURL: "https://gitlab.example.com/cad/demo-module/-/pipelines/1", BuildSystem: "gitlab", BuildID: "1", ArtifactVersion: "20260609.1", PackageVersion: "1.2.3", StorageURI: "minio://modex/DemoModule/latest/docs-artifact.zip", Status: "published", PublishedAt: now, CreatedAt: now}}
	return s
}

func (s *Store) CurrentUser() User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.user
}

func (s *Store) CategoryTree() []Category {
	s.mu.RLock()
	defer s.mu.RUnlock()
	byParent := map[string][]Category{}
	for _, c := range s.categories {
		cp := c
		cp.Children = nil
		byParent[c.ParentID] = append(byParent[c.ParentID], cp)
	}
	var attach func(parent string) []Category
	attach = func(parent string) []Category {
		nodes := byParent[parent]
		sort.Slice(nodes, func(i, j int) bool { return nodes[i].SortOrder < nodes[j].SortOrder })
		for i := range nodes {
			nodes[i].Children = attach(nodes[i].ID)
		}
		return nodes
	}
	return attach("")
}

func (s *Store) Modules(categoryID, keyword string) []Module {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Module
	q := strings.ToLower(keyword)
	for _, m := range s.modules {
		if categoryID != "" && !contains(m.CategoryIDs, categoryID) {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(m.Name+" "+m.Description+" "+strings.Join(m.Keywords, " ")), q) {
			continue
		}
		cp := m
		cp.AvailableVers = s.versionsForLocked(m.ModuleKey)
		out = append(out, cp)
	}
	return out
}

func (s *Store) Module(moduleKey string) (Module, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, m := range s.modules {
		if strings.EqualFold(m.ModuleKey, moduleKey) {
			m.AvailableVers = s.versionsForLocked(m.ModuleKey)
			return m, nil
		}
	}
	return Module{}, ErrNotFound
}

func (s *Store) Versions(moduleKey string) []Version {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.versionsForLocked(moduleKey)
}

func (s *Store) Entries(moduleKey, docsVersion string) []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Entry
	for _, e := range s.entries {
		if strings.EqualFold(e.ModuleKey, moduleKey) && e.DocsVersion == docsVersion {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SortOrder < out[j].SortOrder })
	return out
}

func (s *Store) Releases() []Release {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := append([]Release(nil), s.releases...)
	sort.Slice(out, func(i, j int) bool { return out[i].PublishedAt.After(out[j].PublishedAt) })
	return out
}

func (s *Store) Page(docID string) (Page, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.pages {
		if p.DocID == docID {
			return p, nil
		}
	}
	return Page{}, ErrNotFound
}

func (s *Store) PageByRoute(moduleKey, docsVersion, entryKey string) (Page, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.pages {
		if strings.EqualFold(p.ModuleKey, moduleKey) && p.DocsVersion == docsVersion && p.EntryKey == entryKey {
			return p, nil
		}
	}
	return Page{}, ErrNotFound
}

func (s *Store) Pages() []Page {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Page(nil), s.pages...)
}

func (s *Store) AddSearchLog(log SearchLog) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.searchLogs = append(s.searchLogs, log)
}

func (s *Store) SearchLogs() []SearchLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]SearchLog(nil), s.searchLogs...)
}

func (s *Store) AddMCPLog(log MCPLog) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mcpLogs = append(s.mcpLogs, log)
}

func (s *Store) MCPLogs() []MCPLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]MCPLog(nil), s.mcpLogs...)
}

func (s *Store) versionsForLocked(moduleKey string) []Version {
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
