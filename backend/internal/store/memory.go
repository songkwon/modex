package store

import (
	"errors"
	"mime"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotFound = errors.New("not found")
	ErrInvalid  = errors.New("invalid input")
	ErrConflict = errors.New("resource already exists")
)

type Store struct {
	mu         sync.RWMutex
	user       User
	users      []User
	groups     []Group
	teams      []Team
	categories []Category
	modules    []Module
	versions   []Version
	entries    []Entry
	releases   []Release
	pages      []Page
	searchLogs []SearchLog
	mcpLogs    []MCPLog
	pageViews  []PageView
	navs       map[string][]NavItem
	html       map[string]string
	siteFiles  map[string]SiteFile
	embeddings map[string][]float32
	settings   Settings
	seq        int64
}

// Settings returns a copy of the persisted platform settings.
func (s *Store) Settings() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

// SaveAISettings updates the AI connection. An empty AskAPIKey keeps the
// previously stored key (so a masked round-trip from the UI does not wipe it).
func (s *Store) SaveAISettings(ai AISettings) Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(ai.AskAPIKey) == "" {
		ai.AskAPIKey = s.settings.AI.AskAPIKey
	}
	ai.UpdatedAt = time.Now().UTC()
	s.settings.AI = ai
	return s.settings
}

// New returns a completely empty store. This is the default for a fresh start
// ("从0开始"). No demo users, categories, modules or documents are created.
// Use NewSeeded() only for tests or when you explicitly want demo data.
func New() *Store {
	return &Store{
		users:      []User{},
		groups:     []Group{},
		teams:      []Team{},
		categories: []Category{},
		modules:    []Module{},
		versions:   []Version{},
		entries:    []Entry{},
		releases:   []Release{},
		pages:      []Page{},
		searchLogs: []SearchLog{},
		mcpLogs:    []MCPLog{},
		pageViews:  []PageView{},
		navs:       map[string][]NavItem{},
		html:       map[string]string{},
		siteFiles:  map[string]SiteFile{},
		embeddings: map[string][]float32{},
	}
}

func NewSeeded() *Store {
	now := time.Now().UTC()
	s := &Store{
		user: User{ID: "u-dev", Username: "dev", DisplayName: "研发用户", Email: "dev@example.com", Department: "工程化", Groups: []string{"cad-team", "engineering"}, Roles: []string{"admin"}},
		users: []User{
			{ID: "u-dev", Username: "dev", DisplayName: "研发用户", Email: "dev@example.com", Department: "工程化", Groups: []string{"cad-team", "engineering"}, Roles: []string{"admin"}, Source: "seed", Status: "active", CreatedAt: now, UpdatedAt: now},
			{ID: "u-alice", Username: "alice", DisplayName: "Alice", Email: "alice@example.com", Department: "CAD", Groups: []string{"cad-team"}, Roles: []string{"maintainer"}, Source: "seed", Status: "active", CreatedAt: now, UpdatedAt: now},
			{ID: "u-bob", Username: "bob", DisplayName: "Bob", Email: "bob@example.com", Department: "前端", Groups: []string{"frontend-platform"}, Roles: []string{"viewer"}, Source: "seed", Status: "active", CreatedAt: now, UpdatedAt: now},
		},
		groups: []Group{
			{ID: "g-admin", GroupKey: "admin", Name: "平台管理员", Source: "seed", CreatedAt: now, UpdatedAt: now},
			{ID: "g-engineering", GroupKey: "engineering", Name: "工程化", Source: "seed", CreatedAt: now, UpdatedAt: now},
			{ID: "g-cad", GroupKey: "cad-team", Name: "CAD 团队", Source: "seed", CreatedAt: now, UpdatedAt: now},
			{ID: "g-frontend", GroupKey: "frontend-platform", Name: "前端平台", Source: "seed", CreatedAt: now, UpdatedAt: now},
			{ID: "g-standards", GroupKey: "standards", Name: "研发规范", Source: "seed", CreatedAt: now, UpdatedAt: now},
		},
		teams: []Team{
			{ID: "t-cad", Key: "cad-team", Name: "CAD 团队", Description: "CAD 内核与插件文档维护团队", Leaders: []string{"alice"}, Members: []string{"alice", "bob"}, CreatedAt: now, UpdatedAt: now},
			{ID: "t-eng", Key: "engineering", Name: "工程化团队", Description: "工程化平台与 CBB 规范维护", Leaders: []string{"dev"}, Members: []string{"dev", "alice"}, CreatedAt: now, UpdatedAt: now},
			{ID: "t-fe", Key: "frontend-platform", Name: "前端平台团队", Description: "前端文档框架与组件规范", Leaders: []string{"bob"}, Members: []string{"bob", "dev"}, CreatedAt: now, UpdatedAt: now},
			{ID: "t-std", Key: "standards", Name: "研发规范团队", Description: "通用研发规范、流程与工具规范维护团队 (参考 GitBook/Mintlify 层级领域)", Leaders: []string{"alice"}, Members: []string{"alice", "dev"}, CreatedAt: now, UpdatedAt: now},
		},
		navs:       map[string][]NavItem{},
		html:       map[string]string{},
		siteFiles:  map[string]SiteFile{},
		embeddings: map[string][]float32{},
		categories: []Category{
			{ID: "engineering", Key: "engineering", Name: "工程化", Description: "研发效能、构建、CI/CD 与质量平台", Icon: "wrench", SortOrder: 10, Status: "active", ResponsibleTeam: "engineering"},
			{ID: "engineering.cbb", ParentID: "engineering", Key: "engineering.cbb", Name: "CBB", Description: "CBB 构建与模块治理", Icon: "package", SortOrder: 11, Status: "active", ResponsibleTeam: "engineering"},
			{ID: "cad", Key: "cad", Name: "CAD", Description: "CAD 内核、插件与图形渲染", Icon: "box", SortOrder: 20, Status: "active", ResponsibleTeam: "cad-team"},
			{ID: "cad.demo", ParentID: "cad", Key: "cad.demo", Name: "示例模块", Description: "示例与接入指南", Icon: "book", SortOrder: 21, Status: "active", ResponsibleTeam: "cad-team"},
			{ID: "frontend", Key: "frontend", Name: "前端", Description: "前端平台、组件库和文档框架", Icon: "layout", SortOrder: 25, Status: "active", ResponsibleTeam: "frontend-platform"},
			{ID: "frontend.docs", ParentID: "frontend", Key: "frontend.docs", Name: "文档框架", Description: "VuePress、Fumadocs 与静态站点", Icon: "book-open", SortOrder: 26, Status: "active", ResponsibleTeam: "frontend-platform"},
			{ID: "app", Key: "app", Name: "应用", Description: "业务应用、订单、设备联网", Icon: "layers", SortOrder: 30, Status: "active"},
			{ID: "standards", Key: "standards", Name: "研发规范", Description: "编码规范、流程规范、工具规范与领域文档结构 (可由团队负责人维护层级)", Icon: "book", SortOrder: 40, Status: "active", ResponsibleTeam: "standards"},
			{ID: "standards.standard", ParentID: "standards", Key: "standards.standard", Name: "基础规范", Description: "编程规范、设计原则等 (对应 rd-doc/standard)", Icon: "file-text", SortOrder: 41, Status: "active", ResponsibleTeam: "standards"},
			{ID: "standards.tools", ParentID: "standards", Key: "standards.tools", Name: "工具规范", Description: "版本控制、CI 流程、协作工具规范 (对应 rd-doc/tools/*)", Icon: "tool", SortOrder: 42, Status: "active", ResponsibleTeam: "standards"},
			{ID: "standards.tools.version", ParentID: "standards.tools", Key: "standards.tools.version", Name: "代码版本管理", Description: "Git 工作流与版本规范", Icon: "git-branch", SortOrder: 421, Status: "active", ResponsibleTeam: "standards"},
		},
		modules: []Module{
			{ID: "m-demo", ModuleKey: "DemoModule", Name: "DemoModule", Description: "示例模块文档，覆盖模块落地、维护和发布流程。", OwnerGroup: "cad-team", RepoType: "gitlab", RepoURL: "https://gitlab.example.com/cad/demo-module", DefaultVersion: "latest", Visibility: "internal", Status: "active", PackageName: "DemoModule", PackageVersion: "1.2.3", Channel: "default", Edition: "2025", Keywords: []string{"demo", "cad", "markdown"}, Maintainers: []string{"alice", "bob"}, CategoryIDs: []string{"cad", "cad.demo"}, CategoryPath: "CAD / 示例模块", UpdatedAt: now, Reads7d: 128, Reads30d: 520},
			{ID: "m-cbb", ModuleKey: "CBB", Name: "CBB 文档", Description: "工程化模块构建、依赖分析、发布规范与常见问题。", OwnerGroup: "engineering", RepoType: "gitlab", RepoURL: "https://gitlab.example.com/devops/cbb", DefaultVersion: "latest", Visibility: "internal", Status: "active", PackageName: "CBB", PackageVersion: "2.8.0", Channel: "stable", Edition: "2025", Keywords: []string{"cbb", "ci", "build"}, Maintainers: []string{"platform-team"}, CategoryIDs: []string{"engineering", "engineering.cbb"}, CategoryPath: "工程化 / CBB", UpdatedAt: now.Add(-4 * time.Hour), Reads7d: 342, Reads30d: 1430},
			{ID: "m-vuepress", ModuleKey: "VuePressGuide", Name: "VuePressGuide", Description: "VuePress 文档示例，展示传统 Markdown 文档站如何通过 docsctl 打包发布到 Modex。", OwnerGroup: "frontend-platform", RepoType: "gitlab", RepoURL: "https://gitlab.example.com/frontend/vuepress-guide", DefaultVersion: "latest", Visibility: "internal", Status: "active", PackageName: "VuePressGuide", PackageVersion: "0.4.0", Channel: "docs", Edition: "2025", Keywords: []string{"vuepress", "frontend", "markdown"}, Maintainers: []string{"frontend-docs"}, CategoryIDs: []string{"frontend", "frontend.docs"}, CategoryPath: "前端 / 文档框架", UpdatedAt: now.Add(-2 * time.Hour), Reads7d: 86, Reads30d: 310},
			{ID: "m-fumadocs", ModuleKey: "FumadocsKit", Name: "FumadocsKit", Description: "Fumadocs 文档示例，展示基于 Next.js App Router 与 MDX 的现代文档站接入方式。", OwnerGroup: "frontend-platform", RepoType: "gitlab", RepoURL: "https://gitlab.example.com/frontend/fumadocs-kit", DefaultVersion: "latest", Visibility: "internal", Status: "active", PackageName: "FumadocsKit", PackageVersion: "0.2.0", Channel: "docs", Edition: "2025", Keywords: []string{"fumadocs", "nextjs", "mdx", "frontend"}, Maintainers: []string{"frontend-docs"}, CategoryIDs: []string{"frontend", "frontend.docs"}, CategoryPath: "前端 / 文档框架", UpdatedAt: now.Add(-90 * time.Minute), Reads7d: 72, Reads30d: 260},
		},
	}
	s.versions = []Version{
		{ID: "v-demo-latest", ModuleKey: "DemoModule", DocsVersion: "latest", DisplayName: "latest", VersionType: "branch", IsDefault: true, Status: "active", SourceBranch: "main", PackageVersion: "1.2.3", Channel: "default", Edition: "2025", SupportStatus: "supported", CreatedAt: now},
		{ID: "v-demo-v12", ModuleKey: "DemoModule", DocsVersion: "v1.2", DisplayName: "v1.2", VersionType: "release", IsDefault: false, Status: "active", SourceBranch: "release/1.2", PackageVersion: "1.2.0", Channel: "default", Edition: "2025", SupportStatus: "supported", CreatedAt: now.AddDate(0, -1, 0)},
		{ID: "v-cbb-latest", ModuleKey: "CBB", DocsVersion: "latest", DisplayName: "latest", VersionType: "branch", IsDefault: true, Status: "active", SourceBranch: "main", PackageVersion: "2.8.0", Channel: "stable", Edition: "2025", SupportStatus: "supported", CreatedAt: now},
		{ID: "v-vuepress-latest", ModuleKey: "VuePressGuide", DocsVersion: "latest", DisplayName: "latest", VersionType: "branch", IsDefault: true, Status: "active", SourceBranch: "main", PackageVersion: "0.4.0", Channel: "docs", Edition: "2025", SupportStatus: "supported", CreatedAt: now},
		{ID: "v-fumadocs-latest", ModuleKey: "FumadocsKit", DocsVersion: "latest", DisplayName: "latest", VersionType: "branch", IsDefault: true, Status: "active", SourceBranch: "main", PackageVersion: "0.2.0", Channel: "docs", Edition: "2025", SupportStatus: "supported", CreatedAt: now},
	}
	s.entries = []Entry{
		{ID: "e-demo-guide", ModuleKey: "DemoModule", DocsVersion: "latest", EntryKey: "guide", Title: "模块落地指导", EntryType: "markdown", Builder: "markdown", Source: "docs/integration-guide.md", StorageURI: "minio://modex/DemoModule/latest/site/guide/index.html", NavURI: "minio://modex/DemoModule/latest/nav.json", IndexStatus: "indexed", IsPrimary: true, SortOrder: 1, Status: "active", CreatedAt: now},
		{ID: "e-demo-maintenance", ModuleKey: "DemoModule", DocsVersion: "latest", EntryKey: "maintenance", Title: "模块维护说明", EntryType: "markdown", Builder: "markdown", Source: "docs/maintenance-guide.md", StorageURI: "minio://modex/DemoModule/latest/site/maintenance/index.html", NavURI: "minio://modex/DemoModule/latest/nav.json", IndexStatus: "indexed", SortOrder: 2, Status: "active", CreatedAt: now},
		{ID: "e-cbb-build", ModuleKey: "CBB", DocsVersion: "latest", EntryKey: "build-cache", Title: "构建缓存清理", EntryType: "markdown", Builder: "markdown", Source: "docs/build-cache.md", StorageURI: "minio://modex/CBB/latest/site/build-cache/index.html", NavURI: "minio://modex/CBB/latest/nav.json", IndexStatus: "indexed", IsPrimary: true, SortOrder: 1, Status: "active", CreatedAt: now},
		{ID: "e-vuepress-guide", ModuleKey: "VuePressGuide", DocsVersion: "latest", EntryKey: "guide", Title: "VuePress 文档站接入", EntryType: "vuepress", Builder: "vuepress", Source: "docs", StorageURI: "minio://modex/VuePressGuide/latest/site/guide/index.html", NavURI: "minio://modex/VuePressGuide/latest/nav.json", IndexStatus: "indexed", IsPrimary: true, SortOrder: 1, Status: "active", CreatedAt: now},
		{ID: "e-fumadocs-guide", ModuleKey: "FumadocsKit", DocsVersion: "latest", EntryKey: "guide", Title: "Fumadocs 文档站接入", EntryType: "fumadocs", Builder: "fumadocs", Source: "content/docs", StorageURI: "minio://modex/FumadocsKit/latest/site/guide/index.html", NavURI: "minio://modex/FumadocsKit/latest/nav.json", IndexStatus: "indexed", IsPrimary: true, SortOrder: 1, Status: "active", CreatedAt: now},
	}
	s.pages = []Page{
		{ID: "p-demo-guide", DocID: "DemoModule:latest:guide", ModuleKey: "DemoModule", ModuleName: "DemoModule", DocsVersion: "latest", PackageVersion: "1.2.3", EntryKey: "guide", EntryType: "markdown", Title: "模块落地指导", Description: "面向业务开发人员的模块接入、部署、接口和异常处理说明。", Path: "/docs/DemoModule/latest/guide", SourceFile: "docs/integration-guide.md", DocType: "markdown", Status: "active", OwnerGroup: "cad-team", CategoryIDs: []string{"cad", "cad.demo"}, Tags: []string{"demo", "cad"}, ContentText: "模块落地指导说明如何接入 DemoModule，包括接口设计、部署运行、异常处理、风险影响面和发布检查。", ContentMD: seedDemoGuideMD, UpdatedAt: now},
		{ID: "p-demo-maintenance", DocID: "DemoModule:latest:maintenance", ModuleKey: "DemoModule", ModuleName: "DemoModule", DocsVersion: "latest", PackageVersion: "1.2.3", EntryKey: "maintenance", EntryType: "markdown", Title: "模块维护说明", Description: "面向维护开发人员的架构、设计、流程和维护说明。", Path: "/docs/DemoModule/latest/maintenance", SourceFile: "docs/maintenance-guide.md", DocType: "markdown", Status: "active", OwnerGroup: "cad-team", CategoryIDs: []string{"cad", "cad.demo"}, Tags: []string{"demo", "cad", "architecture"}, ContentText: "模块维护说明包含总体架构、设计原则、模块结构、核心流程、时序逻辑、前后端设计和质量可维护性要求。", ContentMD: seedDemoMaintenanceMD, UpdatedAt: now},
		{ID: "p-cbb-build", DocID: "CBB:latest:build-cache", ModuleKey: "CBB", ModuleName: "CBB 文档", DocsVersion: "latest", PackageVersion: "2.8.0", EntryKey: "build-cache", EntryType: "markdown", Title: "构建缓存清理", Description: "CBB 构建缓存清理和常见构建问题排查。", Path: "/docs/CBB/latest/build-cache", SourceFile: "docs/build-cache.md", DocType: "markdown", Status: "active", OwnerGroup: "engineering", CategoryIDs: []string{"engineering", "engineering.cbb"}, Tags: []string{"cbb", "ci", "build"}, ContentText: "构建缓存清理用于解决依赖缓存、编译缓存和 CI 工作区残留导致的构建异常。可以重新拉取依赖并清理本地缓存。", ContentMD: seedCBBBuildCacheMD, UpdatedAt: now},
		{ID: "p-vuepress-guide", DocID: "VuePressGuide:latest:guide", ModuleKey: "VuePressGuide", ModuleName: "VuePressGuide", DocsVersion: "latest", PackageVersion: "0.4.0", EntryKey: "guide", EntryType: "vuepress", Title: "VuePress 文档站接入", Description: "VuePress 文档通过 docsctl 执行构建命令并复制 dist 输出目录。", Path: "/docs/VuePressGuide/latest/guide", SourceFile: "docs/README.md", DocType: "vuepress", Status: "active", OwnerGroup: "frontend-platform", CategoryIDs: []string{"frontend", "frontend.docs"}, Tags: []string{"vuepress", "frontend", "markdown"}, ContentText: "VuePress 文档站接入说明如何声明 docs.yaml、执行 npm run docs:build、复制 docs/.vuepress/dist 并生成标准文档包。", UpdatedAt: now.Add(-2 * time.Hour)},
		{ID: "p-fumadocs-guide", DocID: "FumadocsKit:latest:guide", ModuleKey: "FumadocsKit", ModuleName: "FumadocsKit", DocsVersion: "latest", PackageVersion: "0.2.0", EntryKey: "guide", EntryType: "fumadocs", Title: "Fumadocs 文档站接入", Description: "Fumadocs 文档通过 Next.js 与 MDX 构建，适合现代前端文档站。", Path: "/docs/FumadocsKit/latest/guide", SourceFile: "content/docs/index.mdx", DocType: "fumadocs", Status: "active", OwnerGroup: "frontend-platform", CategoryIDs: []string{"frontend", "frontend.docs"}, Tags: []string{"fumadocs", "nextjs", "mdx", "frontend"}, ContentText: "Fumadocs 文档站接入说明如何维护 MDX 内容、运行 Next.js 构建、输出静态站点并交给 Modex 进行搜索和 MCP 读取。", UpdatedAt: now.Add(-90 * time.Minute)},
	}
	s.releases = []Release{{ID: "r-demo-1", ReleaseID: "rel-demo-latest-001", ModuleKey: "DemoModule", DocsVersion: "latest", CommitSHA: "d34db33f", Branch: "main", Publisher: "alice", PipelineURL: "https://gitlab.example.com/cad/demo-module/-/pipelines/1", BuildSystem: "gitlab", BuildID: "1", ArtifactVersion: "20260609.1", PackageVersion: "1.2.3", StorageURI: "minio://modex/DemoModule/latest/docs-artifact.zip", Status: "published", PublishedAt: now, CreatedAt: now}}
	return s
}

func (s *Store) CurrentUser() User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.user
}

// Users returns all users, optionally filtered by a case-insensitive keyword
// matching username, display name, email, or department.
func (s *Store) Users(keyword string) []User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q := strings.ToLower(strings.TrimSpace(keyword))
	out := make([]User, 0, len(s.users))
	for _, u := range s.users {
		if q != "" && !strings.Contains(strings.ToLower(u.Username+" "+u.DisplayName+" "+u.Email+" "+u.Department), q) {
			continue
		}
		out = append(out, u)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Username < out[j].Username })
	return out
}

func (s *Store) UserByID(id string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.userByIDLocked(id)
}

// userByIDLocked looks up a user without acquiring the lock; callers must hold
// s.mu (read or write).
func (s *Store) userByIDLocked(id string) (User, error) {
	for _, u := range s.users {
		if u.ID == id {
			return u, nil
		}
	}
	return User{}, ErrNotFound
}

func (s *Store) UserByMCPToken(token string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.users {
		if u.MCPToken == token {
			return u, nil
		}
	}
	return User{}, ErrNotFound
}

func (s *Store) SetUserMCPToken(id string, token string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.users {
		if s.users[i].ID == id {
			s.users[i].MCPToken = token
			s.users[i].UpdatedAt = time.Now().UTC()
			return s.users[i], nil
		}
	}
	return User{}, ErrNotFound
}

func (s *Store) CreateUser(u User) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(u.Username) == "" {
		return User{}, ErrInvalid
	}
	for _, existing := range s.users {
		if strings.EqualFold(existing.Username, u.Username) || (u.ID != "" && existing.ID == u.ID) {
			return User{}, ErrConflict
		}
	}
	if u.ID == "" {
		u.ID = s.nextIDLocked("u")
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
	now := time.Now().UTC()
	u.CreatedAt = now
	u.UpdatedAt = now
	// SuperAdmin can be set at creation time (usually only by another super admin via the admin UI).
	s.users = append(s.users, u)
	s.ensureGroupsLocked(u.Groups)
	return u, nil
}

func (s *Store) UpdateUser(id string, patch User) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.users {
		if s.users[i].ID != id {
			continue
		}
		u := &s.users[i]
		if patch.DisplayName != "" {
			u.DisplayName = patch.DisplayName
		}
		if patch.Email != "" {
			u.Email = patch.Email
		}
		if patch.Department != "" {
			u.Department = patch.Department
		}
		if patch.Groups != nil {
			u.Groups = patch.Groups
			s.ensureGroupsLocked(patch.Groups)
		}
		if patch.Roles != nil {
			u.Roles = patch.Roles
		}
		if patch.ManagedCategories != nil {
			u.ManagedCategories = patch.ManagedCategories
		}
		if patch.Status != "" {
			u.Status = patch.Status
		}
		// SuperAdmin is a bool; the admin UI (which is itself super-admin only) sends the desired value explicitly.
		u.SuperAdmin = patch.SuperAdmin
		u.UpdatedAt = time.Now().UTC()
		return *u, nil
	}
	return User{}, ErrNotFound
}

func (s *Store) DeleteUser(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.users {
		if s.users[i].ID == id {
			s.users = append(s.users[:i], s.users[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

// UpsertUser syncs an identity from the SSO provider into the user directory on
// login, recording groups and refreshing the last-login timestamp. Manual role
// assignments are preserved unless the provider supplies roles.
func (s *Store) UpsertUser(u User) User {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	s.ensureGroupsLocked(u.Groups)
	for i := range s.users {
		existing := &s.users[i]
		if existing.ID == u.ID || (u.Username != "" && strings.EqualFold(existing.Username, u.Username)) {
			if u.DisplayName != "" {
				existing.DisplayName = u.DisplayName
			}
			if u.Email != "" {
				existing.Email = u.Email
			}
			if u.Department != "" {
				existing.Department = u.Department
			}
			if u.Avatar != "" {
				existing.Avatar = u.Avatar
			}
			if len(u.Groups) > 0 {
				existing.Groups = u.Groups
			}
			if len(u.Roles) > 0 {
				existing.Roles = u.Roles
			}
			existing.Source = "oidc"
			existing.Status = "active"
			existing.LastLoginAt = now
			existing.UpdatedAt = now
			return *existing
		}
	}
	if u.ID == "" {
		u.ID = s.nextIDLocked("u")
	}
	if u.Status == "" {
		u.Status = "active"
	}
	u.Source = "oidc"
	u.LastLoginAt = now
	u.CreatedAt = now
	u.UpdatedAt = now
	s.users = append(s.users, u)
	return u
}

func (s *Store) Groups() []Group {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := append([]Group{}, s.groups...) // non-nil
	sort.Slice(out, func(i, j int) bool { return out[i].GroupKey < out[j].GroupKey })
	return out
}

func (s *Store) CreateGroup(g Group) (Group, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(g.GroupKey) == "" {
		return Group{}, ErrInvalid
	}
	for _, existing := range s.groups {
		if strings.EqualFold(existing.GroupKey, g.GroupKey) {
			return Group{}, ErrConflict
		}
	}
	if g.ID == "" {
		g.ID = s.nextIDLocked("g")
	}
	if g.Name == "" {
		g.Name = g.GroupKey
	}
	if g.Source == "" {
		g.Source = "manual"
	}
	now := time.Now().UTC()
	g.CreatedAt = now
	g.UpdatedAt = now
	s.groups = append(s.groups, g)
	return g, nil
}

// ensureGroupsLocked auto-registers any group keys referenced by a user so the
// group directory stays consistent. Caller must hold the write lock.
func (s *Store) ensureGroupsLocked(keys []string) {
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		found := false
		for _, g := range s.groups {
			if strings.EqualFold(g.GroupKey, key) {
				found = true
				break
			}
		}
		if !found {
			now := time.Now().UTC()
			s.groups = append(s.groups, Group{ID: s.nextIDLocked("g"), GroupKey: key, Name: key, Source: "auto", CreatedAt: now, UpdatedAt: now})
		}
	}
}

// ensureTeamGroupLocked ensures a team's key exists as a group (for owner_group compat).
func (s *Store) ensureTeamGroupLocked(key string) {
	if strings.TrimSpace(key) == "" {
		return
	}
	for _, g := range s.groups {
		if strings.EqualFold(g.GroupKey, key) {
			return
		}
	}
	now := time.Now().UTC()
	s.groups = append(s.groups, Group{ID: s.nextIDLocked("g"), GroupKey: key, Name: key, Source: "team", CreatedAt: now, UpdatedAt: now})
}

// Teams returns all teams sorted by key.
func (s *Store) Teams() []Team {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := append([]Team{}, s.teams...) // always non-nil, even if s.teams was nil
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// Team returns a team by key (preferred) or ID.
func (s *Store) Team(key string) (Team, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	k := strings.ToLower(strings.TrimSpace(key))
	for _, t := range s.teams {
		if strings.EqualFold(t.Key, k) || t.ID == key {
			return t, nil
		}
	}
	return Team{}, ErrNotFound
}

// CreateTeam creates a maintenance team. Key is required and unique. Leader is
// auto-added to members if provided and not present.
func (s *Store) CreateTeam(t Team) (Team, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(t.Key) == "" {
		if strings.TrimSpace(t.Name) == "" {
			return Team{}, ErrInvalid
		}
		t.Key = s.generateTeamKeyLocked(t.Name)
	}
	for _, existing := range s.teams {
		if strings.EqualFold(existing.Key, t.Key) {
			return Team{}, ErrConflict
		}
	}
	if t.ID == "" {
		t.ID = s.nextIDLocked("t")
	}
	if t.Name == "" {
		t.Name = t.Key
	}
	for _, l := range t.Leaders {
		if l != "" && !contains(t.Members, l) {
			t.Members = append(t.Members, l)
		}
	}
	now := time.Now().UTC()
	t.CreatedAt = now
	t.UpdatedAt = now
	s.teams = append(s.teams, t)
	s.ensureTeamGroupLocked(t.Key)
	return t, nil
}

// UpdateTeam patches name/desc/leader/members.
func (s *Store) UpdateTeam(key string, patch Team) (Team, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.teams {
		if !strings.EqualFold(s.teams[i].Key, key) && s.teams[i].ID != key {
			continue
		}
		tm := &s.teams[i]
		if patch.Name != "" {
			tm.Name = patch.Name
		}
		if patch.Description != "" {
			tm.Description = patch.Description
		}
		if patch.Leaders != nil {
			tm.Leaders = cloneStrings(patch.Leaders)
		}
		if patch.Members != nil {
			tm.Members = cloneStrings(patch.Members)
		}
		// Every leader must also be a member.
		for _, l := range tm.Leaders {
			if l != "" && !contains(tm.Members, l) {
				tm.Members = append(tm.Members, l)
			}
		}
		tm.UpdatedAt = time.Now().UTC()
		s.ensureTeamGroupLocked(tm.Key)
		return *tm, nil
	}
	return Team{}, ErrNotFound
}

// DeleteTeam removes a team (no cascade; modules/categories keep the key as string ref).
func (s *Store) DeleteTeam(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.teams {
		if strings.EqualFold(s.teams[i].Key, key) || s.teams[i].ID == key {
			s.teams = append(s.teams[:i], s.teams[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

// AddTeamMember adds a username (or id) to the team's members. Idempotent.
func (s *Store) AddTeamMember(key, member string) (Team, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	member = strings.TrimSpace(member)
	if member == "" {
		return Team{}, ErrInvalid
	}
	for i := range s.teams {
		if !strings.EqualFold(s.teams[i].Key, key) && s.teams[i].ID != key {
			continue
		}
		tm := &s.teams[i]
		if !contains(tm.Members, member) {
			tm.Members = append(tm.Members, member)
		}
		tm.UpdatedAt = time.Now().UTC()
		return *tm, nil
	}
	return Team{}, ErrNotFound
}

// RemoveTeamMember removes a member. Leader is not auto-removed (call SetTeamLeader first if needed).
func (s *Store) RemoveTeamMember(key, member string) (Team, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	member = strings.TrimSpace(member)
	for i := range s.teams {
		if !strings.EqualFold(s.teams[i].Key, key) && s.teams[i].ID != key {
			continue
		}
		tm := &s.teams[i]
		out := tm.Members[:0]
		for _, m := range tm.Members {
			if !strings.EqualFold(m, member) {
				out = append(out, m)
			}
		}
		tm.Members = out
		tm.UpdatedAt = time.Now().UTC()
		return *tm, nil
	}
	return Team{}, ErrNotFound
}

// SetTeamLeader sets leader and ensures they are in members.
func (s *Store) SetTeamLeader(key, leader string) (Team, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	leader = strings.TrimSpace(leader)
	if leader == "" {
		return Team{}, ErrInvalid
	}
	for i := range s.teams {
		if !strings.EqualFold(s.teams[i].Key, key) && s.teams[i].ID != key {
			continue
		}
		tm := &s.teams[i]
		if !contains(tm.Leaders, leader) {
			tm.Leaders = append(tm.Leaders, leader)
		}
		if !contains(tm.Members, leader) {
			tm.Members = append(tm.Members, leader)
		}
		tm.UpdatedAt = time.Now().UTC()
		return *tm, nil
	}
	return Team{}, ErrNotFound
}

// TeamMembers returns the member list for a team (for permission/display).
func (s *Store) TeamMembers(key string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, t := range s.teams {
		if strings.EqualFold(t.Key, key) || t.ID == key {
			return cloneStrings(t.Members)
		}
	}
	return nil
}

// TeamKeysForUser returns the keys of every team the user belongs to, counting
// both the designated leader and listed members.
func (s *Store) TeamKeysForUser(u User) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var keys []string
	for _, t := range s.teams {
		// Leaders are always also members, so checking Members covers both.
		for _, m := range t.Members {
			if strings.EqualFold(m, u.Username) || strings.EqualFold(m, u.ID) {
				keys = append(keys, t.Key)
				break
			}
		}
	}
	return keys
}

// AllCategories returns a flat copy of every category (with ParentID links).
func (s *Store) AllCategories() []Category {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Category, len(s.categories))
	copy(out, s.categories)
	return out
}

// CategoryName returns the display name for a category id (or the id if unknown).
func (s *Store) CategoryName(id string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.categories {
		if c.ID == id {
			return c.Name
		}
	}
	return id
}

func (s *Store) categoryPathLocked(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	var parts []string
	for _, id := range ids {
		name := id
		for _, c := range s.categories {
			if c.ID == id {
				name = c.Name
				break
			}
		}
		parts = append(parts, name)
	}
	return strings.Join(parts, " / ")
}

// EntryModuleKey returns the module key that owns an entry, for permission checks.
func (s *Store) EntryModuleKey(entryID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range s.entries {
		if e.ID == entryID {
			return e.ModuleKey, true
		}
	}
	return "", false
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
	res := attach("")
	if res == nil {
		return []Category{}
	}
	return res
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

func (s *Store) Nav(moduleKey, docsVersion string) []NavItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneNav(s.navs[routeKey(moduleKey, docsVersion, "")])
}

func (s *Store) PageHTML(moduleKey, docsVersion, entryKey string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.html[routeKey(moduleKey, docsVersion, entryKey)]
}

func (s *Store) SiteFile(moduleKey, docsVersion, entryKey, name string) (SiteFile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if name == "" {
		name = "index.html"
	}
	f, ok := s.siteFiles[siteFileKey(moduleKey, docsVersion, entryKey, name)]
	if !ok {
		return SiteFile{}, ErrNotFound
	}
	f.Content = append([]byte(nil), f.Content...)
	return f, nil
}

func (s *Store) Pages() []Page {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Page(nil), s.pages...)
}

// Embedding returns the cached embedding vector for a document, if present.
func (s *Store) Embedding(docID string) ([]float32, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.embeddings[docID]
	if !ok {
		return nil, false
	}
	return append([]float32(nil), v...), true
}

// SetEmbedding stores (or replaces) the embedding vector for a document.
func (s *Store) SetEmbedding(docID string, vec []float32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.embeddings == nil {
		s.embeddings = map[string][]float32{}
	}
	s.embeddings[docID] = append([]float32(nil), vec...)
}

// EmbeddingCount reports how many documents currently have cached embeddings.
func (s *Store) EmbeddingCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.embeddings)
}

// ClearEmbeddings drops the entire embedding cache (used before a full reindex).
func (s *Store) ClearEmbeddings() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.embeddings = map[string][]float32{}
}

func (s *Store) IngestArtifact(a DeployArtifact) (DeployResult, error) {
	if strings.TrimSpace(a.ModuleKey) == "" || strings.TrimSpace(a.DocsVersion) == "" || len(a.Entries) == 0 || len(a.Documents) == 0 {
		return DeployResult{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	moduleName := firstNonEmpty(a.ModuleName, a.ModuleKey)
	moduleIdx, err := s.moduleIndexLocked(a.ModuleKey)
	if err != nil {
		s.modules = append(s.modules, Module{
			ID:               s.nextIDLocked("m"),
			ModuleKey:        a.ModuleKey,
			Name:             moduleName,
			Description:      a.Description,
			OwnerGroup:       firstNonEmpty(firstString(a.Authors), "docs"),
			RepoType:         firstNonEmpty(a.RepoType, "git"),
			RepoURL:          a.RepoURL,
			SourceType:       "gitlab",
			GitLabBranch:     a.Branch,
			DefaultVersion:   a.DocsVersion,
			Visibility:       "internal",
			Status:           "active",
			PackageName:      a.ModuleKey,
			PackageVersion:   a.PackageVersion,
			Channel:          "docs",
			Edition:          a.Edition,
			Keywords:         cloneStrings(a.Keywords),
			Maintainers:      cloneStrings(a.Authors),
			LastSyncedCommit: a.CommitSHA,
			LastSyncedAt:     now,
			UpdatedAt:        now,
		})
		moduleIdx = len(s.modules) - 1
	} else {
		m := &s.modules[moduleIdx]
		m.Name = moduleName
		if a.Description != "" {
			m.Description = a.Description
		}
		m.DefaultVersion = a.DocsVersion
		if a.PackageVersion != "" {
			m.PackageVersion = a.PackageVersion
		}
		if a.Edition != "" {
			m.Edition = a.Edition
		}
		if len(a.Keywords) > 0 {
			m.Keywords = cloneStrings(a.Keywords)
		}
		if len(a.Authors) > 0 {
			m.Maintainers = cloneStrings(a.Authors)
			if m.OwnerGroup == "" {
				m.OwnerGroup = a.Authors[0]
			}
		}
		if m.Status == "" {
			m.Status = "active"
		}
		if m.Visibility == "" {
			m.Visibility = "internal"
		}
		// Refresh source metadata from each CI push (repo/branch/commit).
		if a.RepoURL != "" {
			m.RepoURL = a.RepoURL
		}
		if a.RepoType != "" {
			m.RepoType = a.RepoType
		}
		if a.Branch != "" {
			m.GitLabBranch = a.Branch
		}
		if a.CommitSHA != "" {
			m.LastSyncedCommit = a.CommitSHA
		}
		m.LastSyncedAt = now
		m.UpdatedAt = now
	}
	// Preserve existing category assignment and rebuild the display path from
	// current categories so the admin UI and module cards show the right labels.
	module := &s.modules[moduleIdx]
	module.CategoryPath = s.categoryPathLocked(module.CategoryIDs)
	versionFound := false
	for i := range s.versions {
		if strings.EqualFold(s.versions[i].ModuleKey, a.ModuleKey) && s.versions[i].DocsVersion == a.DocsVersion {
			v := &s.versions[i]
			v.DisplayName = firstNonEmpty(v.DisplayName, a.DocsVersion)
			v.IsDefault = true
			v.Status = "active"
			v.PackageVersion = a.PackageVersion
			v.Edition = a.Edition
			if v.VersionType == "" {
				v.VersionType = "release"
			}
			if v.SupportStatus == "" {
				v.SupportStatus = "supported"
			}
			versionFound = true
		} else if strings.EqualFold(s.versions[i].ModuleKey, a.ModuleKey) {
			s.versions[i].IsDefault = false
		}
	}
	if !versionFound {
		s.versions = append(s.versions, Version{
			ID:             s.nextIDLocked("v"),
			ModuleKey:      a.ModuleKey,
			DocsVersion:    a.DocsVersion,
			DisplayName:    a.DocsVersion,
			VersionType:    "release",
			IsDefault:      true,
			Status:         "active",
			PackageVersion: a.PackageVersion,
			Channel:        firstNonEmpty(module.Channel, "docs"),
			Edition:        a.Edition,
			SupportStatus:  "supported",
			CreatedAt:      now,
		})
	}
	s.entries = removeEntries(s.entries, a.ModuleKey, a.DocsVersion)
	for i, e := range a.Entries {
		s.entries = append(s.entries, Entry{
			ID:          s.nextIDLocked("e"),
			ModuleKey:   a.ModuleKey,
			DocsVersion: a.DocsVersion,
			EntryKey:    e.Key,
			Title:       e.Title,
			EntryType:   firstNonEmpty(e.Type, "markdown"),
			Builder:     firstNonEmpty(e.Type, "markdown"),
			Source:      e.Source,
			StorageURI:  "memory://" + routeKey(a.ModuleKey, a.DocsVersion, e.Key),
			NavURI:      "memory://" + routeKey(a.ModuleKey, a.DocsVersion, ""),
			IndexStatus: "indexed",
			IsPrimary:   i == 0,
			SortOrder:   i + 1,
			Status:      "active",
			CreatedAt:   now,
		})
	}
	s.pages = removePages(s.pages, a.ModuleKey, a.DocsVersion)
	// Drop cached embeddings for this module/version so re-published content is
	// re-embedded on the next reindex (or lazily during search).
	if s.embeddings != nil {
		embPrefix := a.ModuleKey + ":" + a.DocsVersion + ":"
		for docID := range s.embeddings {
			if strings.HasPrefix(docID, embPrefix) {
				delete(s.embeddings, docID)
			}
		}
	}
	for _, d := range a.Documents {
		entryKey := firstNonEmpty(d.EntryKey, entryKeyFromDocID(d.DocID))
		docID := firstNonEmpty(d.DocID, a.ModuleKey+":"+a.DocsVersion+":"+entryKey)
		s.pages = append(s.pages, Page{
			ID:             s.nextIDLocked("p"),
			DocID:          docID,
			ModuleKey:      a.ModuleKey,
			ModuleName:     moduleName,
			DocsVersion:    a.DocsVersion,
			PackageVersion: firstNonEmpty(d.PackageVersion, a.PackageVersion),
			EntryKey:       entryKey,
			EntryType:      firstNonEmpty(d.EntryType, entryTypeForEntry(a.Entries, entryKey)),
			Title:          firstNonEmpty(d.Title, titleForEntry(a.Entries, entryKey)),
			Description:    d.Description,
			Path:           "/docs/" + a.ModuleKey + "/" + a.DocsVersion + "/" + entryKey,
			SourceFile:     d.SourceFile,
			DocType:        firstNonEmpty(d.EntryType, entryTypeForEntry(a.Entries, entryKey)),
			Status:         firstNonEmpty(d.Status, "active"),
			OwnerGroup:     module.OwnerGroup,
			CategoryIDs:    cloneStrings(module.CategoryIDs),
			Tags:           cloneStrings(coalesceStrings(d.Keywords, a.Keywords)),
			ContentText:    d.Content,
			ContentMD:      d.ContentMD,
			UpdatedAt:      now,
		})
	}
	if s.navs == nil {
		s.navs = map[string][]NavItem{}
	}
	s.navs[routeKey(a.ModuleKey, a.DocsVersion, "")] = cloneNav(a.Nav)
	if s.html == nil {
		s.html = map[string]string{}
	}
	if s.siteFiles == nil {
		s.siteFiles = map[string]SiteFile{}
	}
	for k := range s.html {
		prefix := routeKey(a.ModuleKey, a.DocsVersion, "") + ":"
		if strings.HasPrefix(k, prefix) {
			delete(s.html, k)
		}
	}
	for k := range s.siteFiles {
		prefix := routeKey(a.ModuleKey, a.DocsVersion, "") + ":"
		if strings.HasPrefix(k, prefix) {
			delete(s.siteFiles, k)
		}
	}
	for _, e := range a.Entries {
		html := htmlForEntry(a.SiteHTML, e.Key)
		if html != "" {
			s.html[routeKey(a.ModuleKey, a.DocsVersion, e.Key)] = html
		}
	}
	for name, content := range a.SiteFiles {
		entryKey, relName, ok := splitSiteFile(name)
		if !ok {
			continue
		}
		s.siteFiles[siteFileKey(a.ModuleKey, a.DocsVersion, entryKey, relName)] = SiteFile{
			Name: relName, Content: append([]byte(nil), content...), ContentType: contentTypeForName(relName, content),
		}
	}
	rel := Release{
		ID:              s.nextIDLocked("r"),
		ReleaseID:       "rel-" + strings.ToLower(a.ModuleKey) + "-" + strings.ToLower(a.DocsVersion) + "-" + strconv.FormatInt(now.UnixNano(), 36),
		ModuleKey:       a.ModuleKey,
		DocsVersion:     a.DocsVersion,
		Publisher:       firstNonEmpty(firstString(a.Authors), "docsctl"),
		BuildSystem:     "docsctl",
		ArtifactVersion: now.Format("20060102.150405"),
		PackageVersion:  a.PackageVersion,
		StorageURI:      "memory://" + a.ModuleKey + "/" + a.DocsVersion + "/docs-artifact.zip",
		Status:          "published",
		PublishedAt:     now,
		CreatedAt:       now,
	}
	s.releases = append(s.releases, rel)
	return DeployResult{Release: rel, PagesIndexed: len(a.Documents), EntriesIndexed: len(a.Entries), HTMLFiles: len(a.SiteHTML), SiteFiles: len(a.SiteFiles), BytesReceived: a.Bytes}, nil
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

// RecordPageView appends a page view and returns the stored record.
func (s *Store) RecordPageView(pv PageView) PageView {
	s.mu.Lock()
	defer s.mu.Unlock()
	if pv.ViewedAt.IsZero() {
		pv.ViewedAt = time.Now().UTC()
	}
	if pv.ID == "" {
		pv.ID = s.nextIDLocked("pv")
	}
	for _, p := range s.pages {
		if p.DocID == pv.DocID {
			pv.PageID = p.ID
			pv.ModuleKey = p.ModuleKey
			pv.DocsVersion = p.DocsVersion
			break
		}
	}
	s.pageViews = append(s.pageViews, pv)
	return pv
}

// RecordReadProgress updates the latest matching page view with duration and
// scroll depth for the given session and doc, or records a new view if none.
func (s *Store) RecordReadProgress(docID, sessionID string, durationSeconds int, scrollDepth float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := len(s.pageViews) - 1; i >= 0; i-- {
		pv := &s.pageViews[i]
		if pv.DocID == docID && pv.SessionID == sessionID {
			if durationSeconds > pv.DurationSeconds {
				pv.DurationSeconds = durationSeconds
			}
			if scrollDepth > pv.ScrollDepth {
				pv.ScrollDepth = scrollDepth
			}
			return
		}
	}
	s.pageViews = append(s.pageViews, PageView{
		ID: s.nextIDLocked("pv"), DocID: docID, SessionID: sessionID,
		DurationSeconds: durationSeconds, ScrollDepth: scrollDepth, ViewedAt: time.Now().UTC(),
	})
}

// PageAnalytics aggregates recorded views into per-page reading statistics.
// Pages with no recorded views fall back to seeded read counts so the admin
// dashboard is populated on a fresh start.
func (s *Store) PageAnalytics() []PageStat {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now().UTC()
	week := now.AddDate(0, 0, -7)
	month := now.AddDate(0, 0, -30)
	type agg struct {
		pv, reads7, reads30, durSum, durCount int
		users                                 map[string]struct{}
		last                                  time.Time
	}
	byDoc := map[string]*agg{}
	for _, pv := range s.pageViews {
		a := byDoc[pv.DocID]
		if a == nil {
			a = &agg{users: map[string]struct{}{}}
			byDoc[pv.DocID] = a
		}
		a.pv++
		uid := pv.UserID
		if uid == "" {
			uid = pv.SessionID
		}
		if uid != "" {
			a.users[uid] = struct{}{}
		}
		if pv.ViewedAt.After(week) {
			a.reads7++
		}
		if pv.ViewedAt.After(month) {
			a.reads30++
		}
		if pv.DurationSeconds > 0 {
			a.durSum += pv.DurationSeconds
			a.durCount++
		}
		if pv.ViewedAt.After(a.last) {
			a.last = pv.ViewedAt
		}
	}
	var out []PageStat
	for _, p := range s.pages {
		stat := PageStat{DocID: p.DocID, Title: p.Title, ModuleKey: p.ModuleKey, ModuleName: p.ModuleName, DocsVersion: p.DocsVersion, Path: p.Path, LastViewedAt: p.UpdatedAt}
		if a := byDoc[p.DocID]; a != nil {
			stat.PV = a.pv
			stat.UV = len(a.users)
			stat.Reads7d = a.reads7
			stat.Reads30d = a.reads30
			stat.LastViewedAt = a.last
			if a.durCount > 0 {
				stat.AvgDurationSec = a.durSum / a.durCount
			}
		} else {
			stat.Reads7d = s.seedReadsLocked(p.ModuleKey, true)
			stat.Reads30d = s.seedReadsLocked(p.ModuleKey, false)
		}
		out = append(out, stat)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].PV != out[j].PV {
			return out[i].PV > out[j].PV
		}
		return out[i].Reads30d > out[j].Reads30d
	})
	return out
}

func (s *Store) seedReadsLocked(moduleKey string, week bool) int {
	for _, m := range s.modules {
		if strings.EqualFold(m.ModuleKey, moduleKey) {
			if week {
				return m.Reads7d
			}
			return m.Reads30d
		}
	}
	return 0
}

// PageReadStats aggregates recorded views for one document into a daily read
// trend (last `days` days, inclusive of today) plus a per-reader breakdown.
// Readers are keyed by user id, falling back to session id for anonymous views.
func (s *Store) PageReadStats(docID string, days int) PageReadStats {
	if days <= 0 {
		days = 30
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now().UTC()
	today := now.Truncate(24 * time.Hour)
	// Pre-seed every day in the window so the line chart has no gaps.
	idxByDate := map[string]int{}
	daily := make([]DailyReadPoint, days)
	for i := 0; i < days; i++ {
		d := today.AddDate(0, 0, -(days - 1 - i))
		key := d.Format("2006-01-02")
		daily[i] = DailyReadPoint{Date: key, Count: 0}
		idxByDate[key] = i
	}
	windowStart := today.AddDate(0, 0, -(days - 1))

	type ragg struct {
		userID string
		count  int
		last   time.Time
	}
	readers := map[string]*ragg{}
	total := 0
	for _, pv := range s.pageViews {
		if pv.DocID != docID {
			continue
		}
		total++
		if !pv.ViewedAt.Before(windowStart) {
			if i, ok := idxByDate[pv.ViewedAt.UTC().Format("2006-01-02")]; ok {
				daily[i].Count++
			}
		}
		key := pv.UserID
		if key == "" {
			key = "session:" + pv.SessionID
		}
		r := readers[key]
		if r == nil {
			r = &ragg{userID: pv.UserID}
			readers[key] = r
		}
		r.count++
		if pv.ViewedAt.After(r.last) {
			r.last = pv.ViewedAt
		}
	}

	out := PageReadStats{DocID: docID, Total: total, Daily: daily, Readers: []ReaderStat{}}
	for _, r := range readers {
		name := "匿名"
		if r.userID != "" {
			if u, err := s.userByIDLocked(r.userID); err == nil {
				if u.DisplayName != "" {
					name = u.DisplayName
				} else if u.Username != "" {
					name = u.Username
				}
			} else {
				name = r.userID
			}
		}
		out.Readers = append(out.Readers, ReaderStat{Reader: name, UserID: r.userID, Count: r.count, LastReadAt: r.last})
	}
	sort.Slice(out.Readers, func(i, j int) bool {
		if out.Readers[i].Count != out.Readers[j].Count {
			return out.Readers[i].Count > out.Readers[j].Count
		}
		return out.Readers[i].LastReadAt.After(out.Readers[j].LastReadAt)
	})
	return out
}

// CreateCategory adds a new category. Key is required and must be unique.
func (s *Store) CreateCategory(c Category) (Category, error) {
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

func (s *Store) generateCategoryKeyLocked(name, parentID string) string {
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

func (s *Store) generateTeamKeyLocked(name string) string {
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
func (s *Store) MoveCategory(id, parentID string, index int) (Category, error) {
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

func (s *Store) UpdateCategory(id string, c Category) (Category, error) {
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

func (s *Store) DeleteCategory(id string) error {
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

func (s *Store) moduleKeyTakenLocked(key string) bool {
	for _, m := range s.modules {
		if strings.EqualFold(m.ModuleKey, key) {
			return true
		}
	}
	return false
}

func (s *Store) CreateModule(m Module) (Module, error) {
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
	s.modules = append(s.modules, m)
	return m, nil
}

func (s *Store) UpdateModule(moduleKey string, patch Module) (Module, error) {
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
			return out, nil
		}
	}
	return Module{}, ErrNotFound
}

func (s *Store) CreateVersion(moduleKey string, v Version) (Version, error) {
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

func (s *Store) UpdateVersion(moduleKey, docsVersion string, patch Version) (Version, error) {
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

func (s *Store) CreateEntry(moduleKey, docsVersion string, e Entry) (Entry, error) {
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

func (s *Store) UpdateEntry(entryID string, patch Entry) (Entry, error) {
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

func (s *Store) DeleteEntry(entryID string) error {
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

func (s *Store) Release(releaseID string) (Release, error) {
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
func (s *Store) RollbackRelease(releaseID string) (Release, error) {
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

func (s *Store) moduleIndexLocked(moduleKey string) (int, error) {
	for i := range s.modules {
		if strings.EqualFold(s.modules[i].ModuleKey, moduleKey) {
			return i, nil
		}
	}
	return -1, ErrNotFound
}

func (s *Store) nextIDLocked(prefix string) string {
	s.seq++
	return prefix + "-" + strconv.FormatInt(s.seq, 10) + "-" + strconv.FormatInt(time.Now().UnixNano(), 36)
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

func routeKey(moduleKey, docsVersion, entryKey string) string {
	if entryKey == "" {
		return strings.ToLower(moduleKey) + ":" + docsVersion
	}
	return strings.ToLower(moduleKey) + ":" + docsVersion + ":" + entryKey
}

func siteFileKey(moduleKey, docsVersion, entryKey, name string) string {
	return routeKey(moduleKey, docsVersion, entryKey) + ":" + path.Clean(strings.TrimPrefix(name, "/"))
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
