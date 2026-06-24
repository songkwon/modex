package store

import "time"

// DataStore is the business data boundary used by the application. Production
// injects PostgresRepository; MemoryStore exists only as a deterministic test
// fake. Methods intentionally preserve the existing service API while storage
// is migrated away from process-local state.
type DataStore interface {
	Settings() Settings
	SaveAISettings(AISettings) Settings
	CurrentUser() User
	Users(string) []User
	UserByID(string) (User, error)
	UserByMCPToken(string) (User, error)
	SetUserMCPToken(string, string) (User, error)
	CreateUser(User) (User, error)
	UpdateUser(string, User) (User, error)
	DeleteUser(string) error
	UpsertUser(User) User
	Teams() []Team
	Team(string) (Team, error)
	CreateTeam(Team) (Team, error)
	UpdateTeam(string, Team) (Team, error)
	DeleteTeam(string) error
	AddTeamMember(string, string) (Team, error)
	RemoveTeamMember(string, string) (Team, error)
	SetTeamLeader(string, string) (Team, error)
	TeamMembers(string) []string
	TeamKeysForUser(User) []string
	AllCategories() []Category
	CategoryName(string) string
	CategoryTree() []Category
	CreateCategory(Category) (Category, error)
	MoveCategory(string, string, int) (Category, error)
	UpdateCategory(string, Category) (Category, error)
	DeleteCategory(string) error
	Modules(string, string) []Module
	Module(string) (Module, error)
	ModuleByDeployToken(string) (Module, error)
	CreateModule(Module) (Module, error)
	UpdateModule(string, Module) (Module, error)
	Versions(string) []Version
	CreateVersion(string, Version) (Version, error)
	UpdateVersion(string, string, Version) (Version, error)
	Entries(string, string) []Entry
	EntryModuleKey(string) (string, bool)
	CreateEntry(string, string, Entry) (Entry, error)
	UpdateEntry(string, Entry) (Entry, error)
	DeleteEntry(string) error
	Releases() []Release
	Release(string) (Release, error)
	RollbackRelease(string) (Release, error)
	Page(string) (Page, error)
	PageByRoute(string, string, string) (Page, error)
	Pages() []Page
	Nav(string, string) []NavItem
	PageHTML(string, string, string) string
	SiteFile(string, string, string, string) (SiteFile, error)
	SiteObjects() map[string]SiteFile
	ClearSiteAssets()
	Embedding(string) ([]float32, bool)
	SetEmbedding(string, []float32)
	EmbeddingCount() int
	ClearEmbeddings()
	IngestArtifact(DeployArtifact) (DeployResult, error)
	AddSearchLog(SearchLog)
	SearchLogs() []SearchLog
	AddMCPLog(MCPLog)
	MCPLogs() []MCPLog
	AddDocFeedback(DocFeedback) DocFeedback
	DocFeedbacks() []DocFeedback
	RecordPageView(PageView) PageView
	RecordReadProgress(string, string, string, int, float64) PageView
	PageAnalytics() []PageStat
	PageReadStats(string, int) PageReadStats
	UserFavorites(string) []UserFavorite
	SetUserFavorite(string, string, bool) ([]UserFavorite, error)
	UserRecentDocs(string, int) []UserRecentDoc
	RecordUserRecentDoc(string, UserRecentDoc) (UserRecentDoc, error)
	ConnectedApps() []ConnectedApp
	ConnectedAppByClientID(string) (ConnectedApp, error)
	CreateConnectedApp(ConnectedApp, string) (ConnectedApp, error)
	UpdateConnectedApp(string, ConnectedApp) (ConnectedApp, error)
	DeleteConnectedApp(string) error
	VerifyConnectedAppSecret(string, string) (ConnectedApp, error)
	CreateOAuthCode(string, string, string, []string, string, time.Duration) (OAuthGrant, error)
	RedeemOAuthCode(string, string, string, string, string, time.Duration, time.Duration) (OAuthGrant, ConnectedApp, User, error)
	RefreshOAuthToken(string, string, string, string, time.Duration, time.Duration) (OAuthGrant, ConnectedApp, User, error)
	UserByOAuthAccessToken(string) (User, ConnectedApp, OAuthGrant, error)
	RevokeOAuthToken(string, string) bool
	PluginStates() []PluginState
	SavePluginSettings(map[string]PluginSetting) []PluginState
	PluginEffective() map[string]PluginSetting
	UploadedPlugins() []UploadedPlugin
	EnabledUploadedPlugins() []UploadedPlugin
	SaveUploadedPlugin(UploadedPlugin) (UploadedPlugin, error)
	DeleteUploadedPlugin(string) bool
	SnippetData() ([]Snippet, map[string]string)
	SaveSnippetData([]Snippet, map[string]string) ([]Snippet, map[string]string)
}

var _ DataStore = (*MemoryStore)(nil)
var _ DataStore = (*PostgresRepository)(nil)
