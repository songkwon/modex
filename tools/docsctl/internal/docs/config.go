package docs

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func LoadDocsYAML(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()
	var cfg Config
	var current *Entry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") || trim == "entries:" {
			continue
		}
		if strings.HasPrefix(trim, "- ") {
			cfg.Entries = append(cfg.Entries, Entry{})
			current = &cfg.Entries[len(cfg.Entries)-1]
			trim = strings.TrimSpace(strings.TrimPrefix(trim, "- "))
			if trim == "" {
				continue
			}
		}
		if current == nil {
			continue
		}
		key, val, ok := strings.Cut(trim, ":")
		if !ok {
			continue
		}
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		switch strings.TrimSpace(key) {
		case "key":
			current.Key = val
		case "title":
			current.Title = val
		case "type":
			current.Type = val
		case "source":
			current.Source = val
		case "build":
			current.Build = val
		case "output":
			current.Output = val
		}
	}
	return cfg, scanner.Err()
}

// LoadConfig resolves the docs config for a repo. It prefers a committed
// docs.yaml; when absent it synthesizes a single-entry config from environment
// variables (DOCS_BUILDER/DOCS_BUILD/DOCS_OUTPUT/...), so CI pipelines can drive
// docsctl purely via variables without committing a config file into the repo.
func LoadConfig(root string) (Config, error) {
	cfgPath := filepath.Join(root, "docs.yaml")
	if _, err := os.Stat(cfgPath); err == nil {
		return LoadDocsYAML(cfgPath)
	}
	return SynthesizeConfig(root), nil
}

// SynthesizeConfig builds a single-entry config from env vars, falling back to
// auto-detected defaults. Env overrides: DOCS_BUILDER (type), DOCS_ENTRY_KEY,
// DOCS_ENTRY_TITLE, DOCS_ENTRY_SOURCE, DOCS_BUILD, DOCS_OUTPUT.
func SynthesizeConfig(root string) Config {
	kind := firstEnv("DOCS_BUILDER", DetectProjectKind(root))
	e := DefaultEntry(root, kind)
	e.Type = kind
	if v := os.Getenv("DOCS_ENTRY_KEY"); v != "" {
		e.Key = v
	}
	if v := os.Getenv("DOCS_ENTRY_TITLE"); v != "" {
		e.Title = v
	}
	if v := os.Getenv("DOCS_ENTRY_SOURCE"); v != "" {
		e.Source = v
	}
	if v := os.Getenv("DOCS_BUILD"); v != "" {
		e.Build = v
	}
	if v := os.Getenv("DOCS_OUTPUT"); v != "" {
		e.Output = v
	}
	return Config{Entries: []Entry{e}}
}

func requiresBuild(t string) bool {
	switch t {
	case "vitepress", "vuepress", "fumadocs", "docusaurus", "mkdocs", "honkit", "gitbook":
		return true
	default:
		return false
	}
}

func Validate(root string) error {
	cfg, err := LoadConfig(root)
	if err != nil {
		return fmt.Errorf("load documentation config from %s: %w", filepath.Join(root, "docs.yaml"), err)
	}
	if len(cfg.Entries) == 0 {
		return errors.New("documentation config contains no entries; add at least one entry to docs.yaml or set DOCS_BUILDER")
	}
	for _, e := range cfg.Entries {
		var missing []string
		if e.Key == "" {
			missing = append(missing, "key")
		}
		if e.Title == "" {
			missing = append(missing, "title")
		}
		if e.Type == "" {
			missing = append(missing, "type")
		}
		if e.Source == "" {
			missing = append(missing, "source")
		}
		if len(missing) > 0 {
			return fmt.Errorf("entry %q is missing required fields: %s", e.Key, strings.Join(missing, ", "))
		}
		sourcePath := filepath.Join(root, e.Source)
		if _, err := os.Stat(sourcePath); err != nil {
			return fmt.Errorf("entry %q source not found: %s: %w", e.Key, sourcePath, err)
		}
		if requiresBuild(e.Type) {
			if e.Build == "" {
				return fmt.Errorf("entry %q (%s) has no build command; set build in docs.yaml or DOCS_BUILD", e.Key, e.Type)
			}
			if e.Output == "" {
				return fmt.Errorf("entry %q (%s) has no output directory; set output in docs.yaml or DOCS_OUTPUT", e.Key, e.Type)
			}
		}
	}
	if _, err := os.Stat(filepath.Join(root, "cbb.toml")); err == nil {
		_, err = LoadCBBToml(filepath.Join(root, "cbb.toml"))
		return err
	}
	return nil
}

func LoadMetadata(root string, cfg Config) Metadata {
	cbb, cbbOK := LoadCBBToml(filepath.Join(root, "cbb.toml"))
	module := firstEnv("DOCS_MODULE", cbb.Name, "UnknownModule")
	md := Metadata{
		ModuleKey:      module,
		ModuleName:     module,
		DocsVersion:    firstEnv("DOCS_VERSION", "", "latest"),
		PackageVersion: firstEnv("DOCS_PACKAGE_VERSION", cbb.Version, ""),
		Description:    firstEnv("DOCS_DESCRIPTION", cbb.Description, ""),
		Authors:        cbb.Authors,
		Edition:        firstEnv("DOCS_EDITION", cbb.Edition, ""),
		Keywords:       cbb.Keywords,
		RepoURL:        firstEnv("DOCS_REPO_URL", ""),
		RepoType:       firstEnv("DOCS_REPO_TYPE", "git"),
		Branch:         firstEnv("DOCS_BRANCH", ""),
		CommitSHA:      firstEnv("DOCS_COMMIT_SHA", ""),
	}
	if cbbOK == nil {
		md.Source.MetadataFile = "cbb.toml"
	}
	return md
}

type CBBPackage struct {
	Name        string
	Version     string
	Description string
	Authors     []string
	Edition     string
	Keywords    []string
}

func LoadCBBToml(path string) (CBBPackage, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return CBBPackage{}, err
	}
	var pkg CBBPackage
	inPackage := false
	for _, line := range strings.Split(string(b), "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		if strings.HasPrefix(trim, "[") {
			inPackage = trim == "[package]"
			continue
		}
		if !inPackage {
			continue
		}
		key, val, ok := strings.Cut(trim, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		switch key {
		case "name":
			pkg.Name = tomlString(val)
		case "version":
			pkg.Version = tomlString(val)
		case "description":
			pkg.Description = tomlString(val)
		case "authors":
			pkg.Authors = tomlArray(val)
		case "edition":
			pkg.Edition = tomlString(val)
		case "keywords":
			pkg.Keywords = tomlArray(val)
		}
	}
	return pkg, nil
}

func firstEnv(key string, values ...string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func tomlString(v string) string {
	return strings.Trim(strings.TrimSpace(v), `"'`)
}

func tomlArray(v string) []string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(strings.TrimSuffix(v, "]"), "[")
	if strings.TrimSpace(v) == "" {
		return nil
	}
	var out []string
	re := regexp.MustCompile(`"([^"]*)"|'([^']*)'|([^,\s]+)`)
	for _, m := range re.FindAllStringSubmatch(v, -1) {
		for i := 1; i < len(m); i++ {
			if m[i] != "" {
				out = append(out, strings.TrimSpace(m[i]))
				break
			}
		}
	}
	return out
}
