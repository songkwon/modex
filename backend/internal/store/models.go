package store

import "time"

type User struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	DisplayName string   `json:"display_name"`
	Email       string   `json:"email"`
	Department  string   `json:"department"`
	Avatar      string   `json:"avatar,omitempty"`
	Groups      []string `json:"groups"`
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
	SuperAdmin  bool      `json:"is_super_admin,omitempty"`
	// MCPToken is the user's personal bearer token for the MCP server, so MCP
	// calls can be attributed to them. Never serialized (revealed via /api/me/mcp-token).
	MCPToken    string    `json:"-"`
	LastLoginAt time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

type Group struct {
	ID        string    `json:"id"`
	GroupKey  string    `json:"group_key"`
	Name      string    `json:"name"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Team represents a document maintenance team (文档维护团队).
// A team has a leader (负责人) who can add/remove members. Teams can be
// assigned as responsible_party for one or more Categories (领域/分类),
// owning the doc structure and maintenance under those domains.
// Team.Key can be used as owner_group / group reference for compatibility.
type Team struct {
	ID          string    `json:"id"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	// Leaders are the team's responsible people (at least one). Every leader is
	// also kept in Members.
	Leaders     []string  `json:"leaders"`
	Members     []string  `json:"members"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Category struct {
	ID              string     `json:"id"`
	ParentID        string     `json:"parent_id,omitempty"`
	Key             string     `json:"key"`
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	Icon            string     `json:"icon"`
	SortOrder       int        `json:"sort_order"`
	Status          string     `json:"status"`
	ResponsibleTeam string     `json:"responsible_team,omitempty"`
	Children        []Category `json:"children,omitempty"`
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

	// GitLab / source integration (similar to Mintlify deploy integration)
	SourceType       string    `json:"source_type,omitempty"` // "gitlab", "manual"
	DocType          string    `json:"doc_type,omitempty"`    // framework: vitepress|vuepress|fumadocs|markdown
	Mount            string    `json:"mount,omitempty"`       // "single" | "split" (split only for markdown)
	GitLabBranch     string    `json:"gitlab_branch,omitempty"`
	GitLabPath       string    `json:"gitlab_path,omitempty"` // subdir containing docs (e.g. "docs/standard")
	DeployToken      string    `json:"-"`                     // secret for CI deploys; never serialized (reveal via admin endpoint)
	LastSyncedCommit string    `json:"last_synced_commit,omitempty"`
	LastSyncedAt     time.Time `json:"last_synced_at,omitempty"`

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

// AISettings holds the admin-configured large-model connection used to power
// AI answers (RAG). It targets any OpenAI-compatible /chat/completions endpoint
// (OpenAI, DeepSeek, Qwen/DashScope-compat, local vLLM/Ollama, …).
type AISettings struct {
	// AskProtocol selects the API format of the chat endpoint:
	// "openai-chat" (default), "openai-responses", "anthropic", or "gemini".
	AskProtocol     string    `json:"ask_protocol"`
	AskBaseURL      string    `json:"ask_base_url"`      // e.g. https://api.openai.com/v1
	AskModel        string    `json:"ask_model"`         // fetched from the endpoint
	AskAPIKey       string    `json:"ask_api_key"`       // secret; masked when read back
	AskSystemPrompt string    `json:"ask_system_prompt"` // optional override
	// AskMaxTokens caps the answer length. 0 means "use the engine default".
	// (Required by Anthropic; optional for the others.)
	AskMaxTokens int `json:"ask_max_tokens,omitempty"`
	// AskTemperature controls sampling. nil means "use the engine default"
	// (so an explicit 0 for deterministic output is still distinguishable).
	AskTemperature *float64  `json:"ask_temperature,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Settings is the persisted, admin-editable platform configuration.
type Settings struct {
	AI AISettings `json:"ai"`
	// Plugins holds per-plugin enable/config overrides keyed by plugin key.
	// Absent keys fall back to the built-in catalog defaults (see plugins.go).
	Plugins map[string]PluginSetting `json:"plugins,omitempty"`
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

type PageView struct {
	ID              string    `json:"id"`
	PageID          string    `json:"page_id"`
	DocID           string    `json:"doc_id"`
	ModuleKey       string    `json:"module_key"`
	DocsVersion     string    `json:"docs_version"`
	UserID          string    `json:"user_id"`
	SessionID       string    `json:"session_id"`
	DurationSeconds int       `json:"duration_seconds"`
	ScrollDepth     float64   `json:"scroll_depth"`
	ViewedAt        time.Time `json:"viewed_at"`
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
	Reader     string    `json:"reader"` // display name, username, or "匿名"
	UserID     string    `json:"user_id"`
	Count      int       `json:"count"`
	LastReadAt time.Time `json:"last_read_at"`
}

// PageReadStats is the per-page reading detail surfaced behind the doc-page
// "eye" popover: a daily read trend plus a per-reader breakdown.
type PageReadStats struct {
	DocID   string           `json:"doc_id"`
	Total   int              `json:"total"`
	Daily   []DailyReadPoint `json:"daily"`
	Readers []ReaderStat     `json:"readers"`
}
