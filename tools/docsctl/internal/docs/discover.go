package docs

import (
	"os"
	"path/filepath"
)

type DiscoveredProject struct {
	Root       string
	RelPath    string
	Kind       string
	HasConfig  bool
	Written    bool
	ConfigPath string
}

func Discover(root string, maxDepth int, write, force bool) ([]DiscoveredProject, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	var out []DiscoveredProject
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if shouldSkipDir(d.Name()) && path != root {
			return filepath.SkipDir
		}
		depth := relDepth(root, path)
		if depth > maxDepth {
			return filepath.SkipDir
		}
		kind := DetectProjectKind(path)
		hasSignals := kind != "markdown" || exists(filepath.Join(path, "docs")) || exists(filepath.Join(path, "README.md")) || exists(filepath.Join(path, "cbb.toml"))
		if !hasSignals {
			return nil
		}
		configPath := filepath.Join(path, "docs.yaml")
		hasConfig := exists(configPath)
		if hasConfig {
			kind = kindFromConfig(configPath, kind)
		}
		rel, _ := filepath.Rel(root, path)
		if rel == "." {
			rel = "."
		}
		project := DiscoveredProject{Root: path, RelPath: filepath.ToSlash(rel), Kind: kind, HasConfig: hasConfig, ConfigPath: configPath}
		if write && (!hasConfig || force) {
			if err := Init(path, force); err != nil {
				return err
			}
			project.Written = true
			project.HasConfig = true
		}
		out = append(out, project)
		if kind == "vuepress" || kind == "fumadocs" || hasConfig {
			return filepath.SkipDir
		}
		return nil
	})
	return out, err
}

func kindFromConfig(configPath, fallback string) string {
	cfg, err := LoadDocsYAML(configPath)
	if err != nil || len(cfg.Entries) == 0 || cfg.Entries[0].Type == "" {
		return fallback
	}
	return cfg.Entries[0].Type
}

func relDepth(root, path string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return 0
	}
	depth := 1
	for _, r := range rel {
		if r == filepath.Separator {
			depth++
		}
	}
	return depth
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", ".modex", "node_modules", "vendor", "dist", "out", ".next", "target", "build":
		return true
	default:
		return false
	}
}
