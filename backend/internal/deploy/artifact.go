package deploy

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"
)

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
	RepoURL        string   `json:"repo_url"`
	RepoType       string   `json:"repo_type"`
	Branch         string   `json:"branch"`
	CommitSHA      string   `json:"commit_sha"`
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

type Artifact struct {
	Metadata  Metadata
	Manifest  Manifest
	Nav       []NavItem
	Documents []DocumentRecord
	SiteHTML  map[string]string
	SiteFiles map[string][]byte
	LLMS      string
	Bytes     int64
}

func ParseZip(r io.Reader, maxBytes int64) (Artifact, error) {
	var buf bytes.Buffer
	n, err := io.Copy(&buf, io.LimitReader(r, maxBytes+1))
	if err != nil {
		return Artifact{}, err
	}
	if n > maxBytes {
		return Artifact{}, fmt.Errorf("artifact exceeds %d bytes", maxBytes)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		return Artifact{}, err
	}
	a := Artifact{SiteHTML: map[string]string{}, SiteFiles: map[string][]byte{}, Bytes: int64(buf.Len())}
	var docsJSONL []byte
	for _, f := range zr.File {
		name := path.Clean(strings.TrimPrefix(f.Name, "/"))
		if name == "." || strings.HasPrefix(name, "../") {
			continue
		}
		b, err := readZipFile(f)
		if err != nil {
			return Artifact{}, err
		}
		switch name {
		case "metadata.json":
			if err := json.Unmarshal(b, &a.Metadata); err != nil {
				return Artifact{}, fmt.Errorf("metadata.json: %w", err)
			}
		case "manifest.json":
			if err := json.Unmarshal(b, &a.Manifest); err != nil {
				return Artifact{}, fmt.Errorf("manifest.json: %w", err)
			}
		case "nav.json":
			if err := json.Unmarshal(b, &a.Nav); err != nil {
				return Artifact{}, fmt.Errorf("nav.json: %w", err)
			}
		case "documents.jsonl":
			docsJSONL = b
		case "llms.txt":
			a.LLMS = string(b)
		default:
			if strings.HasPrefix(name, "site/") {
				a.SiteFiles[name] = append([]byte(nil), b...)
				if strings.HasSuffix(name, ".html") {
					a.SiteHTML[name] = string(b)
				}
			}
		}
	}
	if len(docsJSONL) > 0 {
		docs, err := parseDocumentsJSONL(docsJSONL)
		if err != nil {
			return Artifact{}, err
		}
		a.Documents = docs
	}
	if strings.TrimSpace(a.Metadata.ModuleKey) == "" {
		return Artifact{}, fmt.Errorf("metadata.module_key is required")
	}
	if strings.TrimSpace(a.Metadata.DocsVersion) == "" {
		a.Metadata.DocsVersion = "latest"
	}
	if len(a.Manifest.Entries) == 0 {
		return Artifact{}, fmt.Errorf("manifest.entries is required")
	}
	if len(a.Documents) == 0 {
		return Artifact{}, fmt.Errorf("documents.jsonl must contain at least one document")
	}
	return a, nil
}

func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func parseDocumentsJSONL(b []byte) ([]DocumentRecord, error) {
	var out []DocumentRecord
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var rec DocumentRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			return nil, fmt.Errorf("documents.jsonl line %d: %w", lineNo, err)
		}
		out = append(out, rec)
	}
	return out, sc.Err()
}
