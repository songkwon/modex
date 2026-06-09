package docs

type Config struct {
	Entries []Entry `json:"entries"`
}

type Entry struct {
	Key    string `json:"key"`
	Title  string `json:"title"`
	Type   string `json:"type"`
	Source string `json:"source"`
	Build  string `json:"build,omitempty"`
	Output string `json:"output,omitempty"`
}

type Metadata struct {
	ModuleKey      string   `json:"module_key"`
	ModuleName     string   `json:"module_name"`
	DocsVersion    string   `json:"docs_version"`
	PackageVersion string   `json:"package_version"`
	Description    string   `json:"description"`
	Authors        []string `json:"authors"`
	Edition        string   `json:"edition"`
	Keywords       []string `json:"keywords"`
	Source         struct {
		MetadataFile string `json:"metadata_file"`
	} `json:"source"`
}

type Manifest struct {
	SchemaVersion string  `json:"schema_version"`
	GeneratedBy   string  `json:"generated_by"`
	Entries       []Entry `json:"entries"`
}

type NavItem struct {
	Title    string    `json:"title"`
	Path     string    `json:"path"`
	Children []NavItem `json:"children,omitempty"`
}

type DocumentRecord struct {
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
