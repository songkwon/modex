package docs

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type ignoreRule struct {
	raw     string
	re      *regexp.Regexp
	negate  bool
	dirOnly bool
}

type IgnoreMatcher struct {
	rules []ignoreRule
}

func LoadModexIgnore(root string) IgnoreMatcher {
	b, err := os.ReadFile(filepath.Join(root, ".modexignore"))
	if err != nil {
		return IgnoreMatcher{}
	}
	var rules []ignoreRule
	for _, line := range strings.Split(string(b), "\n") {
		rule, ok := parseIgnoreRule(line)
		if ok {
			rules = append(rules, rule)
		}
	}
	return IgnoreMatcher{rules: rules}
}

func shouldSkipDocsDir(name string) bool {
	switch strings.ToLower(name) {
	case ".git", ".modex", "node_modules", "vendor", "dist", "out", ".next", "target", "build",
		".vitepress", ".vuepress", ".docusaurus", ".gitbook", ".cursor", ".claude", ".github", ".vscode":
		return true
	default:
		return strings.HasPrefix(name, ".")
	}
}

func (m IgnoreMatcher) Ignored(rel string, isDir bool) bool {
	if len(m.rules) == 0 {
		return false
	}
	rel = strings.TrimPrefix(filepath.ToSlash(rel), "./")
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" || rel == "." {
		return false
	}
	ignored := false
	for _, rule := range m.rules {
		if rule.dirOnly && !isDir && !pathWithinDirPattern(rel, rule.raw) {
			continue
		}
		if rule.re.MatchString(rel) {
			ignored = !rule.negate
		}
	}
	return ignored
}

func parseIgnoreRule(line string) (ignoreRule, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return ignoreRule{}, false
	}
	negate := strings.HasPrefix(line, "!")
	if negate {
		line = strings.TrimSpace(strings.TrimPrefix(line, "!"))
	}
	if line == "" {
		return ignoreRule{}, false
	}
	line = filepath.ToSlash(line)
	dirOnly := strings.HasSuffix(line, "/")
	line = strings.Trim(line, "/")
	if line == "" {
		return ignoreRule{}, false
	}
	return ignoreRule{raw: line, re: regexp.MustCompile(ignorePatternRegexp(line, dirOnly)), negate: negate, dirOnly: dirOnly}, true
}

func ignorePatternRegexp(pattern string, dirOnly bool) string {
	pattern = strings.TrimPrefix(filepath.ToSlash(pattern), "/")
	var target string
	if strings.Contains(pattern, "/") {
		target = globRegexp(pattern)
	} else {
		target = `(?:^|.*/)` + globRegexp(pattern)
	}
	if dirOnly {
		return `^` + target + `(?:/.*)?$`
	}
	return `^` + target + `$`
}

func globRegexp(pattern string) string {
	var b strings.Builder
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				b.WriteString(".*")
				i++
			} else {
				b.WriteString(`[^/]*`)
			}
		case '?':
			b.WriteString(`[^/]`)
		default:
			b.WriteString(regexp.QuoteMeta(string(pattern[i])))
		}
	}
	return b.String()
}

func pathWithinDirPattern(rel, pattern string) bool {
	pattern = strings.Trim(pattern, "/")
	if strings.Contains(pattern, "/") {
		return strings.HasPrefix(rel, pattern+"/")
	}
	for _, part := range strings.Split(rel, "/") {
		if part == pattern {
			return true
		}
	}
	return false
}

func isIndexableMarkdownFile(rel string) bool {
	rel = filepath.ToSlash(rel)
	base := filepath.Base(rel)
	lowerBase := strings.ToLower(base)
	if strings.HasPrefix(base, ".") {
		return false
	}
	if !strings.HasSuffix(lowerBase, ".md") && !strings.HasSuffix(lowerBase, ".mdx") {
		return false
	}
	switch lowerBase {
	case "claude.md", "agents.md", "gemini.md", "qwen.md", "cursor.md", "copilot.md":
		return false
	default:
		return true
	}
}
