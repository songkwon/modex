package store

import "time"

type User struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	DisplayName string   `json:"display_name"`
	Email       string   `json:"email"`
	Department  string   `json:"department"`
	Avatar      string   `json:"avatar,omitempty"`
	Roles       []string `json:"roles"`
	// ManagedCategories lists the platform/category IDs this user may manage.
	// Super admins manage everything regardless of this list.
	ManagedCategories []string `json:"managed_categories,omitempty"`
	Source            string   `json:"source,omitempty"`
	Status            string   `json:"status,omitempty"`
	// SuperAdmin is a persisted flag that marks the user as a platform super administrator.
	// It is combined with SUPER_ADMIN_USERS env (see auth service) so that
	// super admin status can be granted either statically (env, for bootstrap)
	// or dynamically via the admin UI.
	SuperAdmin bool `json:"is_super_admin,omitempty"`
	// MCPToken is the user's personal bearer token for the MCP server, so MCP
	// calls can be attributed to them. Never serialized (revealed via /api/me/mcp-token).
	MCPToken    string    `json:"-"`
	LastLoginAt time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

type ConnectedApp struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Description      string    `json:"description,omitempty"`
	ClientID         string    `json:"client_id"`
	ClientSecretHash string    `json:"client_secret_hash,omitempty"`
	RedirectURIs     []string  `json:"redirect_uris"`
	Scopes           []string  `json:"scopes"`
	Trusted          bool      `json:"trusted"`
	Enabled          bool      `json:"enabled"`
	CreatedBy        string    `json:"created_by,omitempty"`
	LastUsedAt       time.Time `json:"last_used_at,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type OAuthGrant struct {
	ID               string    `json:"id"`
	AppID            string    `json:"app_id"`
	UserID           string    `json:"user_id"`
	CodeHash         string    `json:"code_hash,omitempty"`
	AccessTokenHash  string    `json:"access_token_hash,omitempty"`
	RefreshTokenHash string    `json:"refresh_token_hash,omitempty"`
	RedirectURI      string    `json:"redirect_uri,omitempty"`
	Scopes           []string  `json:"scopes,omitempty"`
	CodeExpiresAt    time.Time `json:"code_expires_at,omitempty"`
	AccessExpiresAt  time.Time `json:"access_expires_at,omitempty"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at,omitempty"`
	RevokedAt        time.Time `json:"revoked_at,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Team represents a document maintenance team (文档维护团队).
// A team has a leader (负责人) who can add/remove members. Teams can be
// assigned as responsible_party for one or more Categories (领域/分类),
// owning the doc structure and maintenance under those domains.
// Team.Key is also used as the module/page owner_group string.
type Team struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	// Leaders are the team's responsible people (at least one). Every leader is
	// also kept in Members.
	Leaders   []string  `json:"leaders"`
	Members   []string  `json:"members"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TeamSummary struct {
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Leaders     []string `json:"leaders"`
	Members     []string `json:"members,omitempty"`
}

type Category struct {
	ID                  string       `json:"id"`
	ParentID            string       `json:"parent_id,omitempty"`
	Key                 string       `json:"key"`
	Name                string       `json:"name"`
	Description         string       `json:"description"`
	Icon                string       `json:"icon"`
	SortOrder           int          `json:"sort_order"`
	Status              string       `json:"status"`
	ResponsibleTeam     string       `json:"responsible_team,omitempty"`
	ResponsibleTeamInfo *TeamSummary `json:"responsible_team_info,omitempty"`
	Children            []Category   `json:"children,omitempty"`
}

type Module struct {
	ID             string   `json:"id"`
	ModuleKey      string   `json:"module_key"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	OwnerGroup     string   `json:"owner_group"`
	RepoType       string   `json:"repo_type"` // "gitlab", "github", "manual"
	RepoURL        string   `json:"repo_url"`
	DefaultVersion string   `json:"default_version"`
	Visibility     string   `json:"visibility"`
	Status         string   `json:"status"`
	PackageName    string   `json:"package_name"`
	PackageVersion string   `json:"package_version"`
	Channel        string   `json:"channel"`
	Edition        string   `json:"edition"`
	Keywords       []string `json:"keywords"`
	Maintainers    []string `json:"maintainers"`
	CategoryIDs    []string `json:"category_ids"`
	CategoryPath   string   `json:"category_path"`
	CreatedBy      string   `json:"created_by,omitempty"`
	CreatedByName  string   `json:"created_by_name,omitempty"`

	// GitLab / source integration (similar to Mintlify deploy integration)
	SourceType       string    `json:"source_type,omitempty"` // "gitlab", "manual"
	DocType          string    `json:"doc_type,omitempty"`    // framework: vitepress|vuepress|fumadocs|markdown
	Mount            string    `json:"mount,omitempty"`       // "single" | "split" (split only for markdown)
	GitLabBranch     string    `json:"gitlab_branch,omitempty"`
	GitLabPath       string    `json:"gitlab_path,omitempty"` // subdir containing docs (e.g. "docs/standard")
	DeployToken      string    `json:"-"`                     // secret for CI deploys; never serialized (reveal via admin endpoint)
	LastSyncedCommit string    `json:"last_synced_commit,omitempty"`
	LastSyncedAt     time.Time `json:"last_synced_at,omitempty"`

	// Read-only flag for admins (the actual token is secret and exposed only via /deploy-token).
	DeployTokenSet bool `json:"deploy_token_set,omitempty"`

	AvailableVers []Version `json:"available_versions,omitempty"`
	UpdatedAt     time.Time `json:"updated_at"`
	Reads7d       int       `json:"reads_7d"`
	Reads30d      int       `json:"reads_30d"`
}

type Version struct {
	ID             string    `json:"id"`
	ModuleKey      string    `json:"module_key"`
	DocsVersion    string    `json:"docs_version"`
	DisplayName    string    `json:"display_name"`
	VersionType    string    `json:"version_type"`
	IsDefault      bool      `json:"is_default"`
	Status         string    `json:"status"`
	SourceBranch   string    `json:"source_branch"`
	PackageVersion string    `json:"package_version"`
	Channel        string    `json:"channel"`
	Edition        string    `json:"edition"`
	SupportStatus  string    `json:"support_status"`
	CreatedAt      time.Time `json:"created_at"`
}

type Entry struct {
	ID          string    `json:"id"`
	ModuleKey   string    `json:"module_key"`
	DocsVersion string    `json:"docs_version"`
	EntryKey    string    `json:"entry_key"`
	Title       string    `json:"title"`
	EntryType   string    `json:"entry_type"`
	Builder     string    `json:"builder"`
	Source      string    `json:"source"`
	StorageURI  string    `json:"storage_uri"`
	NavURI      string    `json:"nav_uri"`
	IndexStatus string    `json:"index_status"`
	IsPrimary   bool      `json:"is_primary"`
	SortOrder   int       `json:"sort_order"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type Release struct {
	ID              string    `json:"id"`
	ReleaseID       string    `json:"release_id"`
	ModuleKey       string    `json:"module_key"`
	DocsVersion     string    `json:"docs_version"`
	CommitSHA       string    `json:"commit_sha"`
	Branch          string    `json:"branch"`
	Publisher       string    `json:"publisher"`
	PipelineURL     string    `json:"pipeline_url"`
	BuildSystem     string    `json:"build_system"`
	BuildID         string    `json:"build_id"`
	TriggerType     string    `json:"trigger_type"`
	SourceIP        string    `json:"source_ip"`
	ArtifactVersion string    `json:"artifact_version"`
	PackageVersion  string    `json:"package_version"`
	StorageURI      string    `json:"storage_uri"`
	Status          string    `json:"status"`
	PublishedAt     time.Time `json:"published_at"`
	CreatedAt       time.Time `json:"created_at"`
}

type Page struct {
	ID             string    `json:"id"`
	DocID          string    `json:"doc_id"`
	ModuleKey      string    `json:"module_key"`
	ModuleName     string    `json:"module_name"`
	DocsVersion    string    `json:"docs_version"`
	PackageVersion string    `json:"package_version"`
	EntryKey       string    `json:"entry_key"`
	EntryType      string    `json:"entry_type"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Path           string    `json:"path"`
	SourceFile     string    `json:"source_file"`
	DocType        string    `json:"doc_type"`
	Status         string    `json:"status"`
	OwnerGroup     string    `json:"owner_group"`
	CategoryIDs    []string  `json:"category_ids"`
	Tags           []string  `json:"tags"`
	ContentText    string    `json:"content_text"`
	ContentHTML    string    `json:"content_html,omitempty"`
	ContentMD      string    `json:"content_md,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// DefaultAskSystemPrompt is the built-in RAG system prompt used when an admin
// has not configured a custom one. It is exposed via the settings API so the
// admin UI can pre-fill the editor and offer a "reset to default" action.
const DefaultAskSystemPrompt = "你是企业研发文档助手。只依据提供的【文档片段】回答用户问题，使用简洁中文；若片段中没有答案，明确说明未在文档中找到，不要编造。回答末尾不要重复罗列来源。"

// AISettings holds the admin-configured model and retrieval settings used to
// power AI answers (RAG), semantic search, reranking, and recall evaluation.
type AISettings struct {
	// AskProtocol selects the API format of the chat endpoint:
	// "openai-chat" (default), "openai-responses", "anthropic", or "gemini".
	AskProtocol     string `json:"ask_protocol"`
	AskBaseURL      string `json:"ask_base_url"`      // e.g. https://api.openai.com/v1
	AskModel        string `json:"ask_model"`         // fetched from the endpoint
	AskAPIKey       string `json:"ask_api_key"`       // secret; masked when read back
	AskSystemPrompt string `json:"ask_system_prompt"` // optional override
	// AskMaxTokens caps the answer length. 0 means "use the engine default".
	// (Required by Anthropic; optional for the others.)
	AskMaxTokens int `json:"ask_max_tokens,omitempty"`
	// AskTemperature controls sampling. nil means "use the engine default"
	// (so an explicit 0 for deterministic output is still distinguishable).
	AskTemperature *float64 `json:"ask_temperature,omitempty"`

	EmbeddingBaseURL string `json:"embedding_base_url,omitempty"`
	EmbeddingModel   string `json:"embedding_model,omitempty"`
	EmbeddingAPIKey  string `json:"embedding_api_key,omitempty"` // secret; masked when read back
	EmbeddingDim     int    `json:"embedding_dim,omitempty"`     // fixed at embedding.Dim (1024); the configured model must output this many dims

	RerankBaseURL string `json:"rerank_base_url,omitempty"`
	RerankModel   string `json:"rerank_model,omitempty"`
	RerankAPIKey  string `json:"rerank_api_key,omitempty"` // secret; masked when read back
	RerankTopK    int    `json:"rerank_top_k,omitempty"`

	ChunkStrategy string `json:"chunk_strategy,omitempty"` // fixed, heading, markdown, semantic
	ChunkSize     int    `json:"chunk_size,omitempty"`
	ChunkOverlap  int    `json:"chunk_overlap,omitempty"`

	RecallTestQuery  string `json:"recall_test_query,omitempty"`
	RecallTestTopK   int    `json:"recall_test_top_k,omitempty"`
	RecallTestDocIDs string `json:"recall_test_doc_ids,omitempty"` // newline/comma separated expected doc ids

	UpdatedAt time.Time `json:"updated_at"`
}

// Settings is the persisted, admin-editable platform configuration.
type Settings struct {
	AI AISettings `json:"ai"`
	// Plugins holds per-plugin enable/config overrides keyed by plugin key.
	// Absent keys fall back to the built-in catalog defaults (see plugins.go).
	Plugins map[string]PluginSetting `json:"plugins,omitempty"`
	// Snippets and Variables power reusable doc content (see snippets.go).
	Snippets  []Snippet         `json:"snippets,omitempty"`
	Variables map[string]string `json:"variables,omitempty"`
	// UploadedPlugins are admin-imported, sandbox-rendered JSX plugins
	// (see uploaded_plugins.go). Enable/disable reuses Plugins overrides.
	UploadedPlugins []UploadedPlugin `json:"uploaded_plugins,omitempty"`
}

type NavItem struct {
	Title    string    `json:"title"`
	Path     string    `json:"path"`
	Children []NavItem `json:"children,omitempty"`
}

type DeployEntry struct {
	Key    string `json:"key"`
	Title  string `json:"title"`
	Type   string `json:"type"`
	Source string `json:"source"`
	Output string `json:"output,omitempty"`
}

type DeployDocument struct {
	DocID          string   `json:"doc_id"`
	ModuleKey      string   `json:"module_key"`
	ModuleName     string   `json:"module_name"`
	DocsVersion    string   `json:"docs_version"`
	PackageVersion string   `json:"package_version"`
	EntryKey       string   `json:"entry_key"`
	EntryType      string   `json:"entry_type"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	Content        string   `json:"content"`
	ContentMD      string   `json:"content_md,omitempty"`
	Path           string   `json:"path"`
	SourceFile     string   `json:"source_file"`
	Keywords       []string `json:"keywords"`
	Status         string   `json:"status"`
}

type DeployArtifact struct {
	ModuleKey      string
	ModuleName     string
	DocsVersion    string
	PackageVersion string
	Description    string
	Authors        []string
	Edition        string
	Keywords       []string
	RepoURL        string
	RepoType       string
	Branch         string
	CommitSHA      string
	TriggerType    string
	SourceIP       string
	Entries        []DeployEntry
	Documents      []DeployDocument
	Nav            []NavItem
	SiteHTML       map[string]string
	SiteFiles      map[string][]byte
	Bytes          int64
}

type DeployResult struct {
	Release        Release `json:"release"`
	PagesIndexed   int     `json:"pages_indexed"`
	EntriesIndexed int     `json:"entries_indexed"`
	HTMLFiles      int     `json:"html_files"`
	SiteFiles      int     `json:"site_files"`
	BytesReceived  int64   `json:"bytes_received"`
}

type SiteFile struct {
	Name        string
	Content     []byte
	ContentType string
}

type SearchLog struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	IPAddress    string    `json:"ip_address,omitempty"` // recorded for anonymous searches
	Query        string    `json:"query"`
	Mode         string    `json:"mode"`
	FiltersJSON  string    `json:"filters_json"`
	ResultCount  int       `json:"result_count"`
	ClickedDocID string    `json:"clicked_doc_id"`
	SearchedAt   time.Time `json:"searched_at"`
}

type MCPLog struct {
	ID          string    `json:"id"`
	ToolName    string    `json:"tool_name"`
	UserID      string    `json:"user_id"`
	Query       string    `json:"query"`
	InputJSON   string    `json:"input_json"`
	ResultCount int       `json:"result_count"`
	CreatedAt   time.Time `json:"created_at"`
}

type DocFeedback struct {
	ID        string    `json:"id"`
	DocID     string    `json:"doc_id"`
	PageID    string    `json:"page_id"`
	ModuleKey string    `json:"module_key"`
	Title     string    `json:"title"`
	Rating    string    `json:"rating"`
	Comment   string    `json:"comment"`
	UserID    string    `json:"user_id"`
	SessionID string    `json:"session_id"`
	CreatedAt time.Time `json:"created_at"`
}

type PageView struct {
	ID              string    `json:"id"`
	PageID          string    `json:"page_id"`
	DocID           string    `json:"doc_id"`
	ModuleKey       string    `json:"module_key"`
	ModuleName      string    `json:"module_name,omitempty"`
	DocsVersion     string    `json:"docs_version"`
	EntryKey        string    `json:"entry_key,omitempty"`
	Title           string    `json:"title,omitempty"`
	Path            string    `json:"path,omitempty"`
	UserID          string    `json:"user_id"`
	SessionID       string    `json:"session_id"`
	ReadID          string    `json:"read_id,omitempty"`
	DurationSeconds int       `json:"duration_seconds"`
	ScrollDepth     float64   `json:"scroll_depth"`
	ViewedAt        time.Time `json:"viewed_at"`
}

type UserFavorite struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	ModuleKey string    `json:"module_key"`
	CreatedAt time.Time `json:"created_at"`
}

type UserRecentDoc struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	DocID       string    `json:"doc_id"`
	Title       string    `json:"title"`
	ModuleKey   string    `json:"module_key"`
	ModuleName  string    `json:"module_name"`
	DocsVersion string    `json:"docs_version"`
	EntryKey    string    `json:"entry_key"`
	Href        string    `json:"href"`
	ViewedAt    time.Time `json:"viewed_at"`
}

// PageStat is an aggregated reading-statistics row for one document page.
type PageStat struct {
	DocID          string    `json:"doc_id"`
	Title          string    `json:"title"`
	ModuleKey      string    `json:"module_key"`
	ModuleName     string    `json:"module_name"`
	DocsVersion    string    `json:"docs_version"`
	Path           string    `json:"path"`
	PV             int       `json:"pv"`
	UV             int       `json:"uv"`
	Reads7d        int       `json:"reads_7d"`
	Reads30d       int       `json:"reads_30d"`
	AvgDurationSec int       `json:"avg_duration_seconds"`
	LastViewedAt   time.Time `json:"last_viewed_at"`
}

// DailyReadPoint is one day's read count for a single page (line-chart point).
type DailyReadPoint struct {
	Date  string `json:"date"` // YYYY-MM-DD (UTC)
	Count int    `json:"count"`
}

// ReaderStat is one reader's aggregated read activity for a single page.
type ReaderStat struct {
	Reader         string    `json:"reader"` // display name, username, or "匿名"
	UserID         string    `json:"user_id"`
	Count          int       `json:"count"`
	AvgDurationSec int       `json:"avg_duration_seconds"`
	LastReadAt     time.Time `json:"last_read_at"`
}

// PageReadStats is the per-page reading detail surfaced behind the doc-page
// "eye" popover: a daily read trend plus a per-reader breakdown.
type PageReadStats struct {
	DocID          string           `json:"doc_id"`
	Total          int              `json:"total"`
	AvgDurationSec int              `json:"avg_duration_seconds"`
	Daily          []DailyReadPoint `json:"daily"`
	Readers        []ReaderStat     `json:"readers"`
}
