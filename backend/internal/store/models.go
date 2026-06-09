package store

import "time"

type User struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email"`
	Department  string    `json:"department"`
	Groups      []string  `json:"groups"`
	Roles       []string  `json:"roles"`
	Source      string    `json:"source,omitempty"`
	Status      string    `json:"status,omitempty"`
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

type Category struct {
	ID          string     `json:"id"`
	ParentID    string     `json:"parent_id,omitempty"`
	Key         string     `json:"key"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Icon        string     `json:"icon"`
	SortOrder   int        `json:"sort_order"`
	Status      string     `json:"status"`
	Children    []Category `json:"children,omitempty"`
}

type Module struct {
	ID             string    `json:"id"`
	ModuleKey      string    `json:"module_key"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	OwnerGroup     string    `json:"owner_group"`
	RepoType       string    `json:"repo_type"`
	RepoURL        string    `json:"repo_url"`
	DefaultVersion string    `json:"default_version"`
	Visibility     string    `json:"visibility"`
	Status         string    `json:"status"`
	PackageName    string    `json:"package_name"`
	PackageVersion string    `json:"package_version"`
	Channel        string    `json:"channel"`
	Edition        string    `json:"edition"`
	Keywords       []string  `json:"keywords"`
	Maintainers    []string  `json:"maintainers"`
	CategoryIDs    []string  `json:"category_ids"`
	CategoryPath   string    `json:"category_path"`
	AvailableVers  []Version `json:"available_versions,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
	Reads7d        int       `json:"reads_7d"`
	Reads30d       int       `json:"reads_30d"`
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
	UpdatedAt      time.Time `json:"updated_at"`
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
