package store

import (
	"errors"
	"path"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotFound = errors.New("not found")
	ErrInvalid  = errors.New("invalid input")
	ErrConflict = errors.New("resource already exists")
)

// MemoryStore is a test fake. Production assembly injects PostgresRepository.
type MemoryStore struct {
	mu         sync.RWMutex
	user       User
	users      []User
	apps       []ConnectedApp
	grants     []OAuthGrant
	teams      []Team
	categories []Category
	modules    []Module
	versions   []Version
	entries    []Entry
	releases   []Release
	pages      []Page
	searchLogs []SearchLog
	mcpLogs    []MCPLog
	feedbacks  []DocFeedback
	pageViews  []PageView
	favorites  []UserFavorite
	recentDocs []UserRecentDoc
	navs       map[string][]NavItem
	html       map[string]string
	siteFiles  map[string]SiteFile
	embeddings map[string][]float32
	settings   Settings
	seq        int64
}

// Settings returns a copy of the persisted platform settings.
func (s *MemoryStore) Settings() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

// SaveAISettings updates the AI connection. An empty AskAPIKey keeps the
// previously stored key (so a masked round-trip from the UI does not wipe it).
func (s *MemoryStore) SaveAISettings(ai AISettings) Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(ai.AskAPIKey) == "" {
		ai.AskAPIKey = s.settings.AI.AskAPIKey
	}
	if strings.TrimSpace(ai.EmbeddingAPIKey) == "" {
		ai.EmbeddingAPIKey = s.settings.AI.EmbeddingAPIKey
	}
	if strings.TrimSpace(ai.RerankAPIKey) == "" {
		ai.RerankAPIKey = s.settings.AI.RerankAPIKey
	}
	ai.UpdatedAt = time.Now().UTC()
	s.settings.AI = ai
	return s.settings
}

// NewTestStore returns an empty deterministic fake for unit tests.
func NewTestStore() *MemoryStore {
	return &MemoryStore{
		users:      []User{},
		apps:       []ConnectedApp{builtinCodexOAuthApp(time.Now().UTC())},
		grants:     []OAuthGrant{},
		teams:      []Team{},
		categories: []Category{},
		modules:    []Module{},
		versions:   []Version{},
		entries:    []Entry{},
		releases:   []Release{},
		pages:      []Page{},
		searchLogs: []SearchLog{},
		mcpLogs:    []MCPLog{},
		feedbacks:  []DocFeedback{},
		pageViews:  []PageView{},
		favorites:  []UserFavorite{},
		recentDocs: []UserRecentDoc{},
		navs:       map[string][]NavItem{},
		html:       map[string]string{},
		siteFiles:  map[string]SiteFile{},
		embeddings: map[string][]float32{},
	}
}

func NewSeededTestStore() *MemoryStore {
	now := time.Now().UTC()
	s := &MemoryStore{
		user: User{ID: "u-dev", Username: "dev", DisplayName: "研发用户", Email: "dev@example.com", Department: "工程化", Roles: []string{"admin"}},
		users: []User{
			{ID: "u-dev", Username: "dev", DisplayName: "研发用户", Email: "dev@example.com", Department: "工程化", Roles: []string{"admin"}, Source: "seed", Status: "active", CreatedAt: now, UpdatedAt: now},
			{ID: "u-alice", Username: "alice", DisplayName: "Alice", Email: "alice@example.com", Department: "CAD", Roles: []string{"maintainer"}, Source: "seed", Status: "active", CreatedAt: now, UpdatedAt: now},
			{ID: "u-bob", Username: "bob", DisplayName: "Bob", Email: "bob@example.com", Department: "前端", Roles: []string{"viewer"}, Source: "seed", Status: "active", CreatedAt: now, UpdatedAt: now},
		},
		apps:   []ConnectedApp{builtinCodexOAuthApp(now)},
		grants: []OAuthGrant{},
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
		favorites:  []UserFavorite{},
		recentDocs: []UserRecentDoc{},
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
	s.releases = []Release{{ID: "r-demo-1", ReleaseID: "rel-demo-latest-001", ModuleKey: "DemoModule", DocsVersion: "latest", CommitSHA: "d34db33f", Branch: "main", Publisher: "alice", PipelineURL: "https://gitlab.example.com/cad/demo-module/-/pipelines/1", BuildSystem: "gitlab", BuildID: "1", TriggerType: "pipeline", SourceIP: "192.0.2.10", ArtifactVersion: "20260609.1", PackageVersion: "1.2.3", StorageURI: "minio://modex/DemoModule/latest/docs-artifact.zip", Status: "published", PublishedAt: now, CreatedAt: now}}
	return s
}

func (s *MemoryStore) CurrentUser() User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.user
}

// Users returns all users, optionally filtered by a case-insensitive keyword
// matching username, display name, email, or department.
func (s *MemoryStore) Users(keyword string) []User {
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

func (s *MemoryStore) UserByID(id string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.userByIDLocked(id)
}

// userByIDLocked looks up a user without acquiring the lock; callers must hold
// s.mu (read or write).
func (s *MemoryStore) userByIDLocked(id string) (User, error) {
	for _, u := range s.users {
		if u.ID == id {
			return u, nil
		}
	}
	return User{}, ErrNotFound
}

func (s *MemoryStore) UserByMCPToken(token string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.users {
		if u.MCPToken == token {
			return u, nil
		}
	}
	return User{}, ErrNotFound
}

func (s *MemoryStore) SetUserMCPToken(id string, token string) (User, error) {
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

func (s *MemoryStore) CreateUser(u User) (User, error) {
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
	return u, nil
}

func (s *MemoryStore) UpdateUser(id string, patch User) (User, error) {
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

func (s *MemoryStore) DeleteUser(id string) error {
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
// login, refreshing the last-login timestamp. Manual role assignments are
// preserved unless the provider supplies roles.
func (s *MemoryStore) UpsertUser(u User) User {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
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

// Teams returns all teams sorted by key.
func (s *MemoryStore) Teams() []Team {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := append([]Team{}, s.teams...) // always non-nil, even if s.teams was nil
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// Team returns a team by key (preferred) or ID.
func (s *MemoryStore) Team(key string) (Team, error) {
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
func (s *MemoryStore) CreateTeam(t Team) (Team, error) {
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
	return t, nil
}

// UpdateTeam patches name/desc/leader/members.
func (s *MemoryStore) UpdateTeam(key string, patch Team) (Team, error) {
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
		return *tm, nil
	}
	return Team{}, ErrNotFound
}

// DeleteTeam removes a team (no cascade; modules/categories keep the key as string ref).
func (s *MemoryStore) DeleteTeam(key string) error {
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
func (s *MemoryStore) AddTeamMember(key, member string) (Team, error) {
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
func (s *MemoryStore) RemoveTeamMember(key, member string) (Team, error) {
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
func (s *MemoryStore) SetTeamLeader(key, leader string) (Team, error) {
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
func (s *MemoryStore) TeamMembers(key string) []string {
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
func (s *MemoryStore) TeamKeysForUser(u User) []string {
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
func (s *MemoryStore) AllCategories() []Category {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Category, len(s.categories))
	copy(out, s.categories)
	s.attachResponsibleTeamInfoLocked(out)
	return out
}

// CategoryName returns the display name for a category id (or the id if unknown).
func (s *MemoryStore) CategoryName(id string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.categories {
		if c.ID == id {
			return c.Name
		}
	}
	return id
}

func (s *MemoryStore) categoryPathLocked(ids []string) string {
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
func (s *MemoryStore) EntryModuleKey(entryID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range s.entries {
		if e.ID == entryID {
			return e.ModuleKey, true
		}
	}
	return "", false
}

func (s *MemoryStore) CategoryTree() []Category {
	s.mu.RLock()
	defer s.mu.RUnlock()
	byParent := map[string][]Category{}
	for _, c := range s.categories {
		cp := c
		cp.Children = nil
		cp.ResponsibleTeamInfo = s.responsibleTeamInfoLocked(cp.ResponsibleTeam)
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

func (s *MemoryStore) attachResponsibleTeamInfoLocked(categories []Category) {
	for i := range categories {
		categories[i].ResponsibleTeamInfo = s.responsibleTeamInfoLocked(categories[i].ResponsibleTeam)
	}
}

func (s *MemoryStore) responsibleTeamInfoLocked(teamKey string) *TeamSummary {
	if strings.TrimSpace(teamKey) == "" {
		return nil
	}
	for _, t := range s.teams {
		if strings.EqualFold(t.Key, teamKey) {
			return &TeamSummary{
				Key:         t.Key,
				Name:        t.Name,
				Description: t.Description,
				Leaders:     cloneStrings(t.Leaders),
				Members:     cloneStrings(t.Members),
			}
		}
	}
	return nil
}

func (s *MemoryStore) Modules(categoryID, keyword string) []Module {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Module, 0)
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
		cp.DeployTokenSet = cp.DeployToken != ""
		out = append(out, cp)
	}
	return out
}

func (s *MemoryStore) Module(moduleKey string) (Module, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, m := range s.modules {
		if strings.EqualFold(m.ModuleKey, moduleKey) {
			m.AvailableVers = s.versionsForLocked(m.ModuleKey)
			m.DeployTokenSet = m.DeployToken != ""
			return m, nil
		}
	}
	return Module{}, ErrNotFound
}

func (s *MemoryStore) ModuleByDeployToken(token string) (Module, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return Module{}, ErrNotFound
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, m := range s.modules {
		if m.DeployToken == token {
			cp := m
			cp.DeployTokenSet = cp.DeployToken != ""
			cp.AvailableVers = s.versionsForLocked(m.ModuleKey)
			return cp, nil
		}
	}
	return Module{}, ErrNotFound
}

func (s *MemoryStore) Versions(moduleKey string) []Version {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.versionsForLocked(moduleKey)
}

func (s *MemoryStore) Entries(moduleKey, docsVersion string) []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Entry, 0)
	for _, e := range s.entries {
		if strings.EqualFold(e.ModuleKey, moduleKey) && e.DocsVersion == docsVersion {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SortOrder < out[j].SortOrder })
	return out
}

func (s *MemoryStore) Releases() []Release {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := append([]Release(nil), s.releases...)
	sort.Slice(out, func(i, j int) bool { return out[i].PublishedAt.After(out[j].PublishedAt) })
	return out
}

func (s *MemoryStore) Page(docID string) (Page, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.pages {
		if p.DocID == docID {
			return p, nil
		}
	}
	return Page{}, ErrNotFound
}

func (s *MemoryStore) PageByRoute(moduleKey, docsVersion, entryKey string) (Page, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.pages {
		if strings.EqualFold(p.ModuleKey, moduleKey) && p.DocsVersion == docsVersion && p.EntryKey == entryKey {
			return p, nil
		}
	}
	return Page{}, ErrNotFound
}

func (s *MemoryStore) Nav(moduleKey, docsVersion string) []NavItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneNav(s.navs[routeKey(moduleKey, docsVersion, "")])
}

func (s *MemoryStore) PageHTML(moduleKey, docsVersion, entryKey string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.html[routeKey(moduleKey, docsVersion, entryKey)]
}

func (s *MemoryStore) SiteFile(moduleKey, docsVersion, entryKey, name string) (SiteFile, error) {
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

func (s *MemoryStore) Pages() []Page {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Page(nil), s.pages...)
}

// Embedding returns the cached embedding vector for a document, if present.
func (s *MemoryStore) Embedding(docID string) ([]float32, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.embeddings[docID]
	if !ok {
		return nil, false
	}
	return append([]float32(nil), v...), true
}

// SetEmbedding stores (or replaces) the embedding vector for a document.
func (s *MemoryStore) SetEmbedding(docID string, vec []float32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.embeddings == nil {
		s.embeddings = map[string][]float32{}
	}
	s.embeddings[docID] = append([]float32(nil), vec...)
}

// EmbeddingCount reports how many documents currently have cached embeddings.
func (s *MemoryStore) EmbeddingCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.embeddings)
}

// ClearEmbeddings drops the entire embedding cache (used before a full reindex).
func (s *MemoryStore) ClearEmbeddings() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.embeddings = map[string][]float32{}
}

// ClearSiteAssets drops cached rendered HTML and static site files from memory.
// Deployed environments can serve these assets from MinIO instead, which keeps
// the backend RSS from growing with image-heavy or large documentation sites.
func (s *MemoryStore) ClearSiteAssets() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.html = map[string]string{}
	s.siteFiles = map[string]SiteFile{}
}

// SiteObjects returns the currently cached site assets keyed by their MinIO
// object path. It is used once at startup to migrate legacy snapshot data.
func (s *MemoryStore) SiteObjects() map[string]SiteFile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]SiteFile, len(s.html)+len(s.siteFiles))
	moduleNames := make(map[string]string, len(s.modules))
	for _, module := range s.modules {
		moduleNames[strings.ToLower(module.ModuleKey)] = module.ModuleKey
	}
	for key, html := range s.html {
		parts := strings.SplitN(key, ":", 3)
		if len(parts) != 3 {
			continue
		}
		moduleKey := firstNonEmpty(moduleNames[parts[0]], parts[0])
		name := path.Join("modules", moduleKey, parts[1], "site", parts[2], "index.html")
		out[name] = SiteFile{Name: name, Content: []byte(html), ContentType: "text/html; charset=utf-8"}
	}
	for key, file := range s.siteFiles {
		parts := strings.SplitN(key, ":", 4)
		if len(parts) != 4 {
			continue
		}
		moduleKey := firstNonEmpty(moduleNames[parts[0]], parts[0])
		name := path.Join("modules", moduleKey, parts[1], "site", parts[2], parts[3])
		file.Name = name
		out[name] = file
	}
	return out
}
