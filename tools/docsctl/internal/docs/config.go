package docs

import (
	"bufio"
	"errors"
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

func Validate(root string) error {
	cfgPath := filepath.Join(root, "docs.yaml")
	cfg, err := LoadDocsYAML(cfgPath)
	if err != nil {
		return err
	}
	if len(cfg.Entries) == 0 {
		return errors.New("docs.yaml must contain entries")
	}
	for _, e := range cfg.Entries {
		if e.Key == "" || e.Title == "" || e.Type == "" || e.Source == "" {
			return errors.New("each entry requires key/title/type/source")
		}
		if _, err := os.Stat(filepath.Join(root, e.Source)); err != nil {
			return err
		}
		if e.Type == "vuepress" && (e.Build == "" || e.Output == "") {
			return errors.New("vuepress entry requires build and output")
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
		Channel:        firstEnv("DOCS_CHANNEL", cbb.Channel, ""),
		Description:    firstEnv("DOCS_DESCRIPTION", cbb.Description, ""),
		Authors:        cbb.Authors,
		Edition:        firstEnv("DOCS_EDITION", cbb.Edition, ""),
		Keywords:       cbb.Keywords,
	}
	if cbbOK == nil {
		md.Source.MetadataFile = "cbb.toml"
	}
	return md
}

type CBBPackage struct {
	Name        string
	Version     string
	Channel     string
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
		case "channel":
			pkg.Channel = tomlString(val)
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
