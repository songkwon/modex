package docs

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/url"
	"os"
	"os/exec"
	pathpkg "path"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

func Build(root, outDir string) error {
	cfg, err := LoadConfig(root)
	if err != nil {
		return err
	}
	md := LoadMetadata(root, cfg)
	if err := os.RemoveAll(outDir); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(outDir, "site"), 0o755); err != nil {
		return err
	}
	var nav []NavItem
	var records []DocumentRecord
	var manifestEntries []Entry
	var full strings.Builder
	// Entry keys become doc IDs (module:version:key), which are globally UNIQUE
	// in the store. Markdown directories generate keys dynamically, so dedupe
	// them across every entry (not just within one mount) — otherwise two
	// split mounts can emit the same key and the deploy fails on a duplicate
	// docs_page.doc_id.
	usedKeys := map[string]int{}
	for _, entry := range cfg.Entries {
		switch entry.Type {
		case "markdown":
			recs, navItems, entries, text, err := buildMarkdown(root, outDir, md, entry, usedKeys)
			if err != nil {
				return err
			}
			records = append(records, recs...)
			nav = append(nav, navItems...)
			manifestEntries = append(manifestEntries, entries...)
			full.WriteString("# " + entry.Title + "\n\n" + text + "\n\n")
		case "static":
			rec, navItem, text, err := buildStatic(root, outDir, md, entry)
			if err != nil {
				return err
			}
			records = append(records, rec)
			nav = append(nav, navItem)
			manifestEntries = append(manifestEntries, entry)
			usedKeys[entry.Key]++
			full.WriteString("# " + entry.Title + "\n\n" + text + "\n\n")
		case "vitepress", "vuepress", "fumadocs", "docusaurus", "mkdocs", "honkit", "gitbook":
			if _, err := buildCommandEntry(root, outDir, md, entry); err != nil {
				return err
			}
			label := entry.Type
			switch entry.Type {
			case "fumadocs":
				label = "Fumadocs"
			case "vitepress":
				label = "VitePress"
			case "vuepress":
				label = "VuePress"
			case "docusaurus":
				label = "Docusaurus"
			case "mkdocs":
				label = "MkDocs"
			case "honkit":
				label = "HonKit"
			case "gitbook":
				label = "GitBook"
			}
			// Index from the source markdown (one record per page/route) rather
			// than the stripped site HTML, which is a single nav-polluted blob.
			// Each record's Path is the in-site route so search hits deep-link
			// into the embedded site.
			pageRecs, fullText := buildSiteMarkdownRecords(root, md, entry)
			if len(pageRecs) == 0 {
				pageRecs = []DocumentRecord{recordFor(md, entry, label+" 文档已构建，详情见静态站点。")}
				fullText = entry.Title + " 文档已构建，详情见静态站点。"
			}
			records = append(records, pageRecs...)
			nav = append(nav, NavItem{Title: entry.Title, Path: "/" + entry.Key})
			manifestEntries = append(manifestEntries, entry)
			usedKeys[entry.Key]++
			full.WriteString("# " + entry.Title + "\n\n" + fullText + "\n\n")
		default:
			return fmt.Errorf("unsupported entry type %q", entry.Type)
		}
	}
	if err := writeJSON(filepath.Join(outDir, "manifest.json"), Manifest{SchemaVersion: "modex.docs/v1", GeneratedBy: "docsctl", Entries: manifestEntries}); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outDir, "metadata.json"), md); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outDir, "nav.json"), nav); err != nil {
		return err
	}
	if err := writeJSONL(filepath.Join(outDir, "documents.jsonl"), records); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "llms.txt"), []byte(llms(md, cfg.Entries)), 0o644); err != nil {
		return err
	}
	if full.Len() > 0 {
		if err := os.WriteFile(filepath.Join(outDir, "llms-full.txt"), []byte(full.String()), 0o644); err != nil {
			return err
		}
	}
	return copyAssets(root, outDir)
}

func buildCommandEntry(root, outDir string, md Metadata, entry Entry) (string, error) {
	if err := ensureNodeModules(root); err != nil {
		return "", fmt.Errorf("entry %q dependency setup failed: %w", entry.Key, err)
	}
	base := fmt.Sprintf("/api/docs/%s/%s/%s/site/", md.ModuleKey, md.DocsVersion, entry.Key)
	if entry.Build != "" {
		cmd := exec.Command("sh", "-c", entry.Build)
		cmd.Dir = root
		// DOCS_BASE must match the URL path Modex serves the static site from,
		// so VitePress emits asset/router paths that resolve under Modex
		// (GET /api/docs/{module}/{version}/{entry}/site/...). The doc's
		// config reads process.env.DOCS_BASE (falling back to its own default
		// for standalone deploys).
		cmd.Env = append(os.Environ(),
			"DOCS_BASE="+base,
			"MODEX_DOCS_BASE="+base,
			"VITEPRESS_BASE="+base,
			"BASE_URL="+base,
		)
		logs := newTailBuffer(64 * 1024)
		cmd.Stdout = io.MultiWriter(os.Stdout, logs)
		cmd.Stderr = io.MultiWriter(os.Stderr, logs)
		fmt.Printf("docsctl build entry=%s type=%s cwd=%s base=%s command=%q\n", entry.Key, entry.Type, root, base, redactBuildOutput(entry.Build))
		if err := cmd.Run(); err != nil {
			return "", buildCommandError(entry, root, base, err, logs.String())
		}
	}
	siteDir := filepath.Join(outDir, "site", entry.Key)
	outputPath, detected, err := resolveBuildOutputPath(root, entry)
	if err != nil {
		return "", err
	}
	if detected {
		fmt.Printf("docsctl build entry=%s detected build output: %s\n", entry.Key, outputPath)
	}
	if err := copyDir(outputPath, siteDir); err != nil {
		return "", fmt.Errorf("entry %q copy build output from %s to %s: %w", entry.Key, outputPath, siteDir, err)
	}
	if baseRewriteEnabled() {
		rewritten, err := rewriteStaticSiteBase(siteDir, base)
		if err != nil {
			return "", fmt.Errorf("entry %q normalize static-site base %s: %w", entry.Key, base, err)
		}
		relative, err := rewriteStaticSiteBaseToRelative(siteDir, base)
		if err != nil {
			return "", fmt.Errorf("entry %q relativize static-site base %s: %w", entry.Key, base, err)
		}
		rewritten += relative
		if rewritten > 0 {
			fmt.Printf("docsctl build entry=%s normalized %d root-relative asset references to %s\n", entry.Key, rewritten, base)
		}
	}
	return extractSiteText(siteDir), nil
}

func resolveBuildOutputPath(root string, entry Entry) (string, bool, error) {
	configured := filepath.Join(root, entry.Output)
	if info, err := os.Stat(configured); err == nil {
		if !info.IsDir() {
			return "", false, fmt.Errorf("entry %q build output is not a directory: %s", entry.Key, configured)
		}
		return configured, false, nil
	}
	for _, candidate := range outputCandidates(root, entry) {
		if candidate == configured {
			continue
		}
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			if site := findSiteOutputDir(candidate); site != "" {
				return site, true, nil
			}
			return candidate, true, nil
		}
	}
	return "", false, fmt.Errorf("entry %q build output not found: %s; command completed but DOCS_OUTPUT/output points to a missing path (configured output: %q). If your docs config sets a custom outDir, set DOCS_OUTPUT to that directory, for example DOCS_OUTPUT=docs/dist/rd/standards", entry.Key, configured, entry.Output)
}

func outputCandidates(root string, entry Entry) []string {
	source := strings.Trim(filepath.ToSlash(entry.Source), "/")
	var rels []string
	add := func(v string) {
		v = strings.Trim(filepath.ToSlash(v), "/")
		if v != "" {
			rels = append(rels, v)
		}
	}
	add(entry.Output)
	switch entry.Type {
	case "vitepress":
		add(filepath.Join(source, ".vitepress", "dist"))
		add(filepath.Join(source, "dist"))
		add(".vitepress/dist")
		add("dist")
	case "vuepress":
		add(filepath.Join(source, ".vuepress", "dist"))
		add(filepath.Join(source, "dist"))
		add(".vuepress/dist")
		add("dist")
	case "docusaurus":
		add("build")
		add(filepath.Join(source, "build"))
		add("dist")
	case "mkdocs":
		add("site")
		add(filepath.Join(source, "site"))
	default:
		add("dist")
		add(filepath.Join(source, "dist"))
		add("build")
		add(filepath.Join(source, "build"))
		add("out")
	}
	out := make([]string, 0, len(rels))
	seen := map[string]bool{}
	for _, rel := range rels {
		path := filepath.Join(root, rel)
		if !seen[path] {
			seen[path] = true
			out = append(out, path)
		}
	}
	return out
}

func findSiteOutputDir(dir string) string {
	if exists(filepath.Join(dir, "index.html")) {
		return dir
	}
	found := ""
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || found != "" {
			return nil
		}
		if info.IsDir() {
			if path != dir && shouldSkipDocsDir(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(info.Name(), "index.html") {
			found = filepath.Dir(path)
			return filepath.SkipDir
		}
		return nil
	})
	return found
}

type tailBuffer struct {
	limit int
	data  []byte
}

func newTailBuffer(limit int) *tailBuffer {
	return &tailBuffer{limit: limit}
}

func (b *tailBuffer) Write(p []byte) (int, error) {
	written := len(p)
	if len(p) >= b.limit {
		b.data = append(b.data[:0], p[len(p)-b.limit:]...)
		return written, nil
	}
	if overflow := len(b.data) + len(p) - b.limit; overflow > 0 {
		copy(b.data, b.data[overflow:])
		b.data = b.data[:len(b.data)-overflow]
	}
	b.data = append(b.data, p...)
	return written, nil
}

func (b *tailBuffer) String() string {
	return strings.TrimSpace(string(b.data))
}

func buildCommandError(entry Entry, root, base string, err error, output string) error {
	exit := "unknown"
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		exit = fmt.Sprintf("%d", exitErr.ExitCode())
	}
	output = redactBuildOutput(output)
	if output == "" {
		output = "(the build command produced no output)"
	}
	return fmt.Errorf("documentation build failed\n  entry: %s (%s)\n  command: %s\n  working directory: %s\n  exit code: %s\n  base path: DOCS_BASE=%s\n  last output:\n%s\n  hint: run the command above in the working directory; verify dependencies, the configured output directory, and framework base-path settings", entry.Key, entry.Type, redactBuildOutput(entry.Build), root, exit, base, indentLines(output, "    "))
}

func redactBuildOutput(output string) string {
	keyValue := regexp.MustCompile(`(?i)(token|api[_-]?key|secret|password)(\s*[:=]\s*)([^\s]+)`)
	bearer := regexp.MustCompile(`(?i)(authorization\s*:\s*bearer\s+)([^\s]+)`)
	output = keyValue.ReplaceAllString(output, "$1$2[REDACTED]")
	output = bearer.ReplaceAllString(output, "$1[REDACTED]")
	return output
}

func indentLines(value, prefix string) string {
	return prefix + strings.ReplaceAll(value, "\n", "\n"+prefix)
}

func baseRewriteEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("DOCS_REWRITE_BASE")))
	return v != "0" && v != "false" && v != "off"
}

var (
	htmlRootRef = regexp.MustCompile(`(?i)(\b(?:href|src|poster|action)\s*=\s*["'])(/[^"'<>]*)(["'])`)
	htmlSrcset  = regexp.MustCompile(`(?i)(\bsrcset\s*=\s*["'])([^"'<>]*)(["'])`)
	cssRootRef  = regexp.MustCompile(`(?i)(url\(\s*["']?)(/[^)'"\s]+)(["']?\s*\))`)
)

func rewriteStaticSiteBase(siteDir, base string) (int, error) {
	base = "/" + strings.Trim(strings.TrimSpace(base), "/") + "/"
	total := 0
	err := filepath.Walk(siteDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || strings.HasSuffix(strings.ToLower(info.Name()), ".map") {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(filePath))
		if ext != ".html" && ext != ".htm" && ext != ".css" && ext != ".json" && ext != ".webmanifest" {
			return nil
		}
		b, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		original := string(b)
		updated := original
		if ext == ".html" || ext == ".htm" {
			updated, total = rewriteMatches(updated, htmlRootRef, siteDir, base, total)
			updated, total = rewriteSrcsetMatches(updated, siteDir, base, total)
			updated, total = rewriteMatches(updated, cssRootRef, siteDir, base, total)
		} else if ext == ".css" {
			updated, total = rewriteMatches(updated, cssRootRef, siteDir, base, total)
		} else {
			updated, total = rewriteJSONRootRefs(updated, siteDir, base, total)
		}
		if updated != original {
			return os.WriteFile(filePath, []byte(updated), info.Mode().Perm())
		}
		return nil
	})
	return total, err
}

func rewriteStaticSiteBaseToRelative(siteDir, base string) (int, error) {
	base = "/" + strings.Trim(strings.TrimSpace(base), "/") + "/"
	total := 0
	err := filepath.Walk(siteDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || strings.HasSuffix(strings.ToLower(info.Name()), ".map") {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(filePath))
		if ext != ".html" && ext != ".htm" && ext != ".css" && ext != ".json" && ext != ".webmanifest" {
			return nil
		}
		b, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		original := string(b)
		updated := original
		if ext == ".html" || ext == ".htm" {
			updated, total = rewriteBaseMatchesToRelative(updated, htmlRootRef, siteDir, filePath, base, total)
			updated, total = rewriteBaseSrcsetToRelative(updated, siteDir, filePath, base, total)
			updated, total = rewriteBaseMatchesToRelative(updated, cssRootRef, siteDir, filePath, base, total)
		} else if ext == ".css" {
			updated, total = rewriteBaseMatchesToRelative(updated, cssRootRef, siteDir, filePath, base, total)
		} else {
			updated, total = rewriteJSONBaseRefsToRelative(updated, siteDir, filePath, base, total)
		}
		if updated != original {
			return os.WriteFile(filePath, []byte(updated), info.Mode().Perm())
		}
		return nil
	})
	return total, err
}

func rewriteBaseMatchesToRelative(input string, pattern *regexp.Regexp, siteDir, filePath, base string, total int) (string, int) {
	updated := pattern.ReplaceAllStringFunc(input, func(match string) string {
		parts := pattern.FindStringSubmatch(match)
		if len(parts) != 4 {
			return match
		}
		rewritten, ok := rewriteBaseRefToRelative(parts[2], siteDir, filePath, base)
		if !ok {
			return match
		}
		total++
		return parts[1] + rewritten + parts[3]
	})
	return updated, total
}

func rewriteBaseSrcsetToRelative(input, siteDir, filePath, base string, total int) (string, int) {
	updated := htmlSrcset.ReplaceAllStringFunc(input, func(match string) string {
		parts := htmlSrcset.FindStringSubmatch(match)
		if len(parts) != 4 {
			return match
		}
		candidates := strings.Split(parts[2], ",")
		for i, candidate := range candidates {
			fields := strings.Fields(strings.TrimSpace(candidate))
			if len(fields) == 0 {
				continue
			}
			if rewritten, ok := rewriteBaseRefToRelative(fields[0], siteDir, filePath, base); ok {
				fields[0] = rewritten
				candidates[i] = strings.Join(fields, " ")
				total++
			}
		}
		return parts[1] + strings.Join(candidates, ", ") + parts[3]
	})
	return updated, total
}

func rewriteJSONBaseRefsToRelative(input, siteDir, filePath, base string, total int) (string, int) {
	startTotal := total
	var value any
	if json.Unmarshal([]byte(input), &value) != nil {
		return input, total
	}
	value, total = walkJSONStringsToRelative(value, siteDir, filePath, base, total)
	if total == startTotal {
		return input, total
	}
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return input, total
	}
	return string(b) + "\n", total
}

func walkJSONStringsToRelative(value any, siteDir, filePath, base string, total int) (any, int) {
	switch current := value.(type) {
	case string:
		if rewritten, ok := rewriteBaseRefToRelative(current, siteDir, filePath, base); ok {
			return rewritten, total + 1
		}
	case []any:
		for i, item := range current {
			current[i], total = walkJSONStringsToRelative(item, siteDir, filePath, base, total)
		}
	case map[string]any:
		for key, item := range current {
			current[key], total = walkJSONStringsToRelative(item, siteDir, filePath, base, total)
		}
	}
	return value, total
}

func rewriteBaseRefToRelative(raw, siteDir, filePath, base string) (string, bool) {
	if !strings.HasPrefix(raw, base) {
		return raw, false
	}
	suffix := strings.TrimPrefix(raw, base)
	targetPath, trailer := splitURLPathAndTrailer(suffix)
	clean, ok := resolveSiteTargetRel(targetPath, siteDir, false)
	if !ok {
		return raw, false
	}
	target := filepath.Join(siteDir, filepath.FromSlash(clean))
	fromDir := filepath.Dir(filePath)
	rel, err := filepath.Rel(fromDir, target)
	if err != nil {
		return raw, false
	}
	rel = filepath.ToSlash(rel)
	if !strings.HasPrefix(rel, ".") {
		rel = "./" + rel
	}
	return rel + trailer, true
}

func rewriteSrcsetMatches(input, siteDir, base string, total int) (string, int) {
	updated := htmlSrcset.ReplaceAllStringFunc(input, func(match string) string {
		parts := htmlSrcset.FindStringSubmatch(match)
		if len(parts) != 4 {
			return match
		}
		candidates := strings.Split(parts[2], ",")
		for i, candidate := range candidates {
			fields := strings.Fields(strings.TrimSpace(candidate))
			if len(fields) == 0 {
				continue
			}
			if rewritten, ok := rewriteRootRef(fields[0], siteDir, base); ok {
				fields[0] = rewritten
				candidates[i] = strings.Join(fields, " ")
				total++
			}
		}
		return parts[1] + strings.Join(candidates, ", ") + parts[3]
	})
	return updated, total
}

func rewriteMatches(input string, pattern *regexp.Regexp, siteDir, base string, total int) (string, int) {
	updated := pattern.ReplaceAllStringFunc(input, func(match string) string {
		parts := pattern.FindStringSubmatch(match)
		if len(parts) != 4 {
			return match
		}
		rewritten, ok := rewriteRootRef(parts[2], siteDir, base)
		if !ok {
			return match
		}
		total++
		return parts[1] + rewritten + parts[3]
	})
	return updated, total
}

func rewriteJSONRootRefs(input, siteDir, base string, total int) (string, int) {
	startTotal := total
	var value any
	if json.Unmarshal([]byte(input), &value) != nil {
		return input, total
	}
	value, total = walkJSONStrings(value, siteDir, base, total)
	if total == startTotal {
		return input, total
	}
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return input, total
	}
	return string(b) + "\n", total
}

func walkJSONStrings(value any, siteDir, base string, total int) (any, int) {
	switch current := value.(type) {
	case string:
		if rewritten, ok := rewriteRootRef(current, siteDir, base); ok {
			return rewritten, total + 1
		}
	case []any:
		for i, item := range current {
			current[i], total = walkJSONStrings(item, siteDir, base, total)
		}
	case map[string]any:
		for key, item := range current {
			current[key], total = walkJSONStrings(item, siteDir, base, total)
		}
	}
	return value, total
}

func rewriteRootRef(raw, siteDir, base string) (string, bool) {
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.HasPrefix(raw, base) {
		return raw, false
	}
	pathPart, trailer := splitURLPathAndTrailer(raw)
	clean, ok := resolveSiteTargetRel(pathPart, siteDir, true)
	if !ok {
		return raw, false
	}
	if clean == "index.html" && (pathPart == "" || pathPart == "/") {
		clean = ""
	}
	return base + clean + trailer, true
}

func splitURLPathAndTrailer(raw string) (string, string) {
	if i := strings.IndexAny(raw, "?#"); i >= 0 {
		return raw[:i], raw[i:]
	}
	return raw, ""
}

func resolveSiteTargetRel(rawPath, siteDir string, allowStripPrefix bool) (string, bool) {
	if rawPath == "" {
		rawPath = "index.html"
	}
	decoded, err := url.PathUnescape(rawPath)
	if err != nil {
		return "", false
	}
	for _, segment := range strings.Split(decoded, "/") {
		if segment == ".." {
			return "", false
		}
	}
	clean := strings.TrimPrefix(pathpkg.Clean(decoded), "/")
	if clean == "." || clean == "" {
		clean = "index.html"
	}
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", false
	}
	if rel, ok := existingSiteTargetRel(siteDir, clean); ok {
		return rel, true
	}
	if !allowStripPrefix {
		return "", false
	}
	parts := strings.Split(clean, "/")
	for i := 1; i < len(parts); i++ {
		candidate := pathpkg.Join(parts[i:]...)
		if rel, ok := existingSiteTargetRel(siteDir, candidate); ok {
			return rel, true
		}
	}
	return "", false
}

func existingSiteTargetRel(siteDir, rel string) (string, bool) {
	target := filepath.Join(siteDir, filepath.FromSlash(rel))
	info, err := os.Stat(target)
	if err != nil {
		return "", false
	}
	if info.IsDir() {
		if _, err := os.Stat(filepath.Join(target, "index.html")); err != nil {
			return "", false
		}
		return pathpkg.Join(rel, "index.html"), true
	}
	return rel, true
}

// extractSiteText walks the generated static site and returns plain text from
// HTML files, stripping scripts/styles so the content can be indexed by Modex.
func extractSiteText(dir string) string {
	var out strings.Builder
	reTags := regexp.MustCompile(`<[^>]+>`)
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(info.Name(), ".html") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		text := stripHTMLForIndex(string(b), reTags)
		if text != "" {
			out.WriteString(text)
			out.WriteString("\n\n")
		}
		return nil
	})
	return strings.TrimSpace(out.String())
}

func stripHTMLForIndex(html string, reTags *regexp.Regexp) string {
	without := strings.ReplaceAll(html, "\\n", "\n")
	without = regexp.MustCompile(`<script\b[^>]*>[\s\S]*?<\/script>`).ReplaceAllString(without, " ")
	without = regexp.MustCompile(`<style\b[^>]*>[\s\S]*?<\/style>`).ReplaceAllString(without, " ")
	without = reTags.ReplaceAllString(without, " ")
	return strings.Join(strings.Fields(without), " ")
}

// ensureNodeModules installs dependencies for Node-based doc builders when a
// package.json exists and node_modules is missing or stale. Set
// DOCS_NPM_INSTALL=0 to skip.
func ensureNodeModules(root string) error {
	if os.Getenv("DOCS_NPM_INSTALL") == "0" || os.Getenv("DOCS_NPM_INSTALL") == "false" {
		return nil
	}
	pkg := filepath.Join(root, "package.json")
	if _, err := os.Stat(pkg); err != nil {
		return nil
	}
	mods := filepath.Join(root, "node_modules")
	info, err := os.Stat(mods)
	if err == nil && info.IsDir() {
		// Reinstall if package.json is newer than node_modules.
		pkgInfo, err := os.Stat(pkg)
		if err == nil && pkgInfo.ModTime().After(info.ModTime()) {
			return runNPMInstall(root)
		}
		return nil
	}
	return runNPMInstall(root)
}

func runNPMInstall(root string) error {
	fmt.Println("docsctl: installing npm dependencies in", root)
	cmd := exec.Command("npm", "install")
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("npm install failed in %s (set DOCS_NPM_INSTALL=0 to skip): %w", root, err)
	}
	return nil
}

func Package(buildDir, artifact string) error {
	if _, err := os.Stat(filepath.Join(buildDir, "llms.txt")); err != nil {
		return fmt.Errorf("llms.txt is required: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(artifact), 0o755); err != nil {
		return err
	}
	file, err := os.Create(artifact)
	if err != nil {
		return err
	}
	defer file.Close()
	zw := zip.NewWriter(file)
	defer zw.Close()
	return filepath.Walk(buildDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(buildDir, path)
		w, err := zw.Create(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(w, f)
		return err
	})
}

func buildMarkdown(root, outDir string, md Metadata, entry Entry, usedKeys map[string]int) ([]DocumentRecord, []NavItem, []Entry, string, error) {
	src := filepath.Join(root, entry.Source)
	if fi, err := os.Stat(src); err == nil && fi.IsDir() {
		return buildMarkdownDirectory(root, outDir, md, entry, src, usedKeys)
	}
	usedKeys[entry.Key]++
	b, err := os.ReadFile(src)
	if err != nil {
		return nil, nil, nil, "", err
	}
	text := string(b)
	entryDir := filepath.Join(outDir, "site", entry.Key)
	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		return nil, nil, nil, "", err
	}
	// Bundle local resources referenced by the Markdown: copy them next to the
	// entry's site files and rewrite the (often Windows-style) relative paths to
	// the URL Modex serves them from, so images and attachments render instead
	// of 404ing.
	base := fmt.Sprintf("/api/docs/%s/%s/%s/site/", md.ModuleKey, md.DocsVersion, entry.Key)
	text = bundleMarkdownResources(text, src, entryDir, base)
	body := markdownToHTML(text)
	page := "<!doctype html><html><head><meta charset=\"utf-8\"><title>" + html.EscapeString(entry.Title) + "</title></head><body><main>" + body + "</main></body></html>"
	if err := os.WriteFile(filepath.Join(entryDir, "index.html"), []byte(page), 0o644); err != nil {
		return nil, nil, nil, "", err
	}
	rec := recordFor(md, entry, stripMarkdown(text))
	rec.ContentMD = strings.TrimSpace(stripFrontmatter(text))
	return []DocumentRecord{rec}, []NavItem{{Title: entry.Title, Path: "/" + entry.Key, Children: headings(text)}}, []Entry{entry}, stripMarkdown(text), nil
}

type markdownPage struct {
	path    string // absolute path on disk
	srcRel  string // path relative to the mount root (slash form)
	rootRel string // path relative to the repo root (slash form)
	key     string // generated entry key / route segment
}

func buildMarkdownDirectory(root, outDir string, md Metadata, baseEntry Entry, src string, usedKeys map[string]int) ([]DocumentRecord, []NavItem, []Entry, string, error) {
	var records []DocumentRecord
	var nav []NavItem
	var entries []Entry
	var full strings.Builder
	ignore := LoadModexIgnore(root)

	// First pass: enumerate every indexable file and assign its key. Links
	// between docs can only be rewritten once the whole tree is known, since a
	// page may reference a sibling that the walk hasn't reached yet. Keys are
	// assigned here (once) so the dedupe counter in usedKeys stays correct.
	var pages []markdownPage
	keyBySource := map[string]string{} // mount-relative source path -> route key
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		if info.IsDir() {
			if path != src && (shouldSkipDocsDir(info.Name()) || ignore.Ignored(rel, true)) {
				return filepath.SkipDir
			}
			return nil
		}
		srcRel, srcRelErr := filepath.Rel(src, path)
		if srcRelErr != nil {
			return srcRelErr
		}
		if ignore.Ignored(rel, false) || !isIndexableMarkdownFile(srcRel) {
			return nil
		}
		srcRelSlash := filepath.ToSlash(srcRel)
		key := markdownEntryKey(baseEntry.Key, srcRelSlash, usedKeys)
		keyBySource[srcRelSlash] = key
		pages = append(pages, markdownPage{path: path, srcRel: srcRelSlash, rootRel: filepath.ToSlash(rel), key: key})
		return nil
	})
	if err != nil {
		return nil, nil, nil, "", err
	}

	// Second pass: render each page, rewriting intra-repo Markdown links to the
	// target doc's route so cross-references resolve instead of 404ing.
	for _, p := range pages {
		raw, readErr := os.ReadFile(p.path)
		if readErr != nil {
			return nil, nil, nil, "", readErr
		}
		text := string(raw)
		title := markdownFileTitle(p.srcRel)
		pageEntry := Entry{Key: p.key, Title: title, Type: "markdown", Source: p.rootRel}
		entryDir := filepath.Join(outDir, "site", p.key)
		if err := os.MkdirAll(entryDir, 0o755); err != nil {
			return nil, nil, nil, "", err
		}
		base := fmt.Sprintf("/api/docs/%s/%s/%s/site/", md.ModuleKey, md.DocsVersion, p.key)
		renderText := bundleMarkdownResources(text, p.path, entryDir, base)
		renderText = rewriteInternalDocLinks(renderText, p.srcRel, keyBySource, md)
		body := markdownToHTML(renderText)
		page := "<!doctype html><html><head><meta charset=\"utf-8\"><title>" + html.EscapeString(title) + "</title></head><body><main>" + body + "</main></body></html>"
		if err := os.WriteFile(filepath.Join(entryDir, "index.html"), []byte(page), 0o644); err != nil {
			return nil, nil, nil, "", err
		}
		rec := recordFor(md, pageEntry, stripMarkdown(renderText))
		rec.ContentMD = strings.TrimSpace(stripFrontmatter(renderText))
		records = append(records, rec)
		insertMarkdownNav(&nav, p.srcRel, title, "/"+p.key, headings(renderText))
		entries = append(entries, pageEntry)
		full.WriteString("# " + title + "\n\n" + stripMarkdown(renderText) + "\n\n")
	}
	if len(records) == 0 {
		return nil, nil, nil, "", fmt.Errorf("no markdown files found under %s", src)
	}
	return records, nav, entries, strings.TrimSpace(full.String()), nil
}

// rewriteInternalDocLinks rewrites Markdown links that point to another Markdown
// file in the same mount to that file's Modex doc route, so cross-references
// resolve to a real page instead of 404ing on the raw source path. srcRel is the
// current file's path relative to the mount root (slash form).
func rewriteInternalDocLinks(text, srcRel string, keyBySource map[string]string, md Metadata) string {
	dir := pathpkg.Dir(srcRel)
	return mdResourceRe.ReplaceAllStringFunc(text, func(m string) string {
		sub := mdResourceRe.FindStringSubmatch(m)
		if len(sub) < 4 || strings.HasPrefix(sub[1], "!") {
			// Image links (![...]) are handled by the resource bundler.
			return m
		}
		route, ok := internalDocRoute(strings.TrimSpace(sub[2]), dir, keyBySource, md)
		if !ok {
			return m
		}
		return sub[1] + route + sub[3]
	})
}

// internalDocRoute maps a Markdown link target (resolved relative to dir) to the
// in-app doc route for the referenced page, or reports false when the target is
// not a local Markdown file tracked in keyBySource (remote, anchor-only, asset,
// or a page that doesn't exist yet).
func internalDocRoute(target, dir string, keyBySource map[string]string, md Metadata) (string, bool) {
	if target == "" || strings.HasPrefix(target, "#") || strings.HasPrefix(target, "/") || strings.HasPrefix(target, "//") {
		return "", false
	}
	low := strings.ToLower(target)
	if strings.HasPrefix(low, "http://") || strings.HasPrefix(low, "https://") || strings.HasPrefix(low, "mailto:") ||
		strings.HasPrefix(low, "tel:") || strings.HasPrefix(low, "data:") {
		return "", false
	}
	pathPart, suffix := splitResourceSuffix(target)
	pathPart = strings.ReplaceAll(pathPart, "\\", "/")
	if !strings.HasSuffix(strings.ToLower(pathPart), ".md") {
		return "", false
	}
	resolved := strings.TrimPrefix(pathpkg.Join(dir, pathPart), "./")
	key, ok := keyBySource[resolved]
	if !ok {
		return "", false
	}
	return "/docs/" + md.ModuleKey + "/" + md.DocsVersion + "/" + key + suffix, true
}

func markdownEntryKey(prefix, rel string, used map[string]int) string {
	rel = strings.TrimSuffix(filepath.ToSlash(rel), filepath.Ext(rel))
	rel = strings.TrimSuffix(rel, "/index")
	rel = strings.TrimSuffix(rel, "/README")
	rel = strings.TrimSuffix(rel, "/readme")
	if rel == "" || strings.EqualFold(rel, "README") || strings.EqualFold(rel, "index") {
		rel = prefix
	} else {
		rel = prefix + "-" + rel
	}
	key := slugKey(rel)
	used[key]++
	if used[key] > 1 {
		key = fmt.Sprintf("%s-%d", key, used[key])
	}
	return key
}

func slugKey(s string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "guide"
	}
	return out
}

func markdownFileTitle(rel string) string {
	base := filepath.Base(filepath.ToSlash(rel))
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func insertMarkdownNav(nav *[]NavItem, rel, title, pagePath string, children []NavItem) {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) == 0 {
		return
	}
	insertNavParts(nav, parts, title, pagePath, children)
}

func insertNavParts(items *[]NavItem, parts []string, title, pagePath string, children []NavItem) {
	if len(parts) == 1 {
		*items = append(*items, NavItem{Title: title, Path: pagePath, Children: children})
		return
	}
	folder := parts[0]
	for i := range *items {
		if (*items)[i].Path == "" && (*items)[i].Title == folder {
			insertNavParts(&(*items)[i].Children, parts[1:], title, pagePath, children)
			return
		}
	}
	*items = append(*items, NavItem{Title: folder})
	insertNavParts(&(*items)[len(*items)-1].Children, parts[1:], title, pagePath, children)
}

func buildStatic(root, outDir string, md Metadata, entry Entry) (DocumentRecord, NavItem, string, error) {
	src := filepath.Join(root, entry.Source)
	dst := filepath.Join(outDir, "site", entry.Key)
	if err := copyDir(src, dst); err != nil {
		return DocumentRecord{}, NavItem{}, "", fmt.Errorf("entry %q copy static source from %s to %s: %w", entry.Key, src, dst, err)
	}
	base := fmt.Sprintf("/api/docs/%s/%s/%s/site/", md.ModuleKey, md.DocsVersion, entry.Key)
	if baseRewriteEnabled() {
		rewritten, err := rewriteStaticSiteBase(dst, base)
		if err != nil {
			return DocumentRecord{}, NavItem{}, "", fmt.Errorf("entry %q normalize static-site base %s: %w", entry.Key, base, err)
		}
		if rewritten > 0 {
			fmt.Printf("docsctl build entry=%s normalized %d root-relative asset references to %s\n", entry.Key, rewritten, base)
		}
	}
	text := extractHTMLText(dst)
	if text == "" {
		text = entry.Title + " static html documentation"
	}
	return recordFor(md, entry, text), NavItem{Title: entry.Title, Path: "/" + entry.Key}, text, nil
}

func recordFor(md Metadata, entry Entry, content string) DocumentRecord {
	desc := content
	if len(desc) > 140 {
		desc = desc[:140]
	}
	return DocumentRecord{
		DocID: md.ModuleKey + ":" + md.DocsVersion + ":" + entry.Key, ModuleKey: md.ModuleKey, ModuleName: md.ModuleName,
		DocsVersion: md.DocsVersion, PackageVersion: md.PackageVersion, EntryKey: entry.Key,
		EntryType: entry.Type, Title: entry.Title, Description: desc, Content: content, Path: "/" + entry.Key,
		SourceFile: entry.Source, Keywords: md.Keywords, Status: "active",
	}
}

// buildSiteMarkdownRecords walks the entry's source markdown tree and returns
// one DocumentRecord per page plus the concatenated clean text (for llms-full).
// It is used for site-builder entries (VitePress/VuePress/Fumadocs) so search
// has per-page granularity and clean content instead of stripped site HTML.
func buildSiteMarkdownRecords(root string, md Metadata, entry Entry) ([]DocumentRecord, string) {
	srcDir := entry.Source
	if srcDir == "" {
		srcDir = "."
	}
	base := filepath.Join(root, srcDir)
	var recs []DocumentRecord
	var full strings.Builder
	ignore := LoadModexIgnore(root)
	_ = filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rootRel, rootRelErr := filepath.Rel(root, path)
		if rootRelErr != nil {
			return nil
		}
		if info.IsDir() {
			if path != base && (shouldSkipDocsDir(info.Name()) || ignore.Ignored(rootRel, true)) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, relErr := filepath.Rel(base, path)
		if relErr != nil {
			return nil
		}
		relSlash := filepath.ToSlash(rel)
		if ignore.Ignored(rootRel, false) || !isIndexableMarkdownFile(relSlash) {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		body := stripFrontmatter(string(raw))
		clean := strings.TrimSpace(body)
		if clean == "" {
			return nil
		}
		route := routeForMarkdown(relSlash)
		title := markdownTitle(body)
		if title == "" {
			title = entry.Title
		}
		text := stripMarkdown(body)
		desc := text
		if len(desc) > 140 {
			desc = desc[:140]
		}
		recs = append(recs, DocumentRecord{
			DocID:          md.ModuleKey + ":" + md.DocsVersion + ":" + entry.Key + route,
			ModuleKey:      md.ModuleKey,
			ModuleName:     md.ModuleName,
			DocsVersion:    md.DocsVersion,
			PackageVersion: md.PackageVersion,
			EntryKey:       entry.Key,
			EntryType:      entry.Type,
			Title:          title,
			Description:    desc,
			Content:        clean,
			ContentMD:      clean,
			Path:           route,
			SourceFile:     filepath.ToSlash(filepath.Join(srcDir, rel)),
			Keywords:       md.Keywords,
			Status:         "active",
		})
		full.WriteString("## " + title + "\n\n" + clean + "\n\n")
		return nil
	})
	return recs, strings.TrimSpace(full.String())
}

// routeForMarkdown maps a source-relative markdown path to its VitePress route.
//
//	index.md            -> /
//	maintenance/index.md -> /maintenance/
//	tools/git.md        -> /tools/git
func routeForMarkdown(rel string) string {
	rel = strings.TrimSuffix(filepath.ToSlash(rel), ".md")
	switch {
	case rel == "index":
		return "/"
	case strings.HasSuffix(rel, "/index"):
		return "/" + strings.TrimSuffix(rel, "index") // keep trailing slash
	default:
		return "/" + rel
	}
}

// markdownTitle returns the YAML frontmatter `title:` or the first ATX H1.
func markdownTitle(text string) string {
	for _, line := range strings.Split(text, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(trim, "# "))
		}
	}
	return ""
}

// stripFrontmatter removes a leading YAML frontmatter block (--- ... ---).
func stripFrontmatter(text string) string {
	t := strings.TrimLeft(text, "\ufeff \t\r\n")
	if !strings.HasPrefix(t, "---") {
		return text
	}
	rest := strings.TrimPrefix(t, "---")
	if idx := strings.Index(rest, "\n---"); idx >= 0 {
		after := rest[idx+len("\n---"):]
		if nl := strings.IndexByte(after, '\n'); nl >= 0 {
			return after[nl+1:]
		}
		return ""
	}
	return text
}

func markdownToHTML(text string) string {
	var out strings.Builder
	for _, line := range strings.Split(text, "\n") {
		trim := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trim, "### "):
			out.WriteString("<h3>" + html.EscapeString(strings.TrimPrefix(trim, "### ")) + "</h3>\n")
		case strings.HasPrefix(trim, "## "):
			out.WriteString("<h2>" + html.EscapeString(strings.TrimPrefix(trim, "## ")) + "</h2>\n")
		case strings.HasPrefix(trim, "# "):
			out.WriteString("<h1>" + html.EscapeString(strings.TrimPrefix(trim, "# ")) + "</h1>\n")
		case strings.HasPrefix(trim, "- "):
			out.WriteString("<p>• " + html.EscapeString(strings.TrimPrefix(trim, "- ")) + "</p>\n")
		case trim == "":
			out.WriteString("\n")
		default:
			out.WriteString("<p>" + html.EscapeString(trim) + "</p>\n")
		}
	}
	return out.String()
}

func stripMarkdown(text string) string {
	re := regexp.MustCompile(`(?m)^#{1,6}\s*|[*_` + "`" + `>#-]`)
	return strings.Join(strings.Fields(re.ReplaceAllString(text, " ")), " ")
}

func headings(text string) []NavItem {
	var out []NavItem
	for _, line := range strings.Split(text, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "## ") || strings.HasPrefix(trim, "### ") {
			title := strings.TrimSpace(strings.TrimLeft(trim, "#"))
			out = append(out, NavItem{Title: title, Path: "#" + slug(title)})
		}
	}
	return out
}

func slug(s string) string {
	return strings.ToLower(strings.ReplaceAll(strings.Join(strings.Fields(s), "-"), "/", "-"))
}

func llms(md Metadata, entries []Entry) string {
	var b strings.Builder
	b.WriteString("# " + md.ModuleName + "\n\n")
	b.WriteString("Description: " + md.Description + "\n")
	b.WriteString("Docs Version: " + md.DocsVersion + "\n")
	b.WriteString("Package Version: " + md.PackageVersion + "\n")
	b.WriteString("Keywords: " + strings.Join(md.Keywords, ", ") + "\n\n")
	b.WriteString("## Entries\n\n")
	for _, e := range entries {
		b.WriteString("- " + e.Title + ": /" + e.Key + "\n")
		b.WriteString("  Type: " + e.Type + "\n")
		b.WriteString("  Source: " + e.Source + "\n")
		b.WriteString("  Summary: " + e.Title + " 文档入口。\n\n")
	}
	b.WriteString("## Recommended Reading\n\n")
	for i, e := range entries {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, e.Title))
	}
	b.WriteString("\n## Notes for AI\n\nUse documents.jsonl for precise retrieval.\nUse this file only as a high-level map of the documentation.\n")
	return b.String()
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

func writeJSONL(path string, records []DocumentRecord) error {
	var b bytes.Buffer
	for _, r := range records {
		line, _ := json.Marshal(r)
		b.Write(line)
		b.WriteByte('\n')
	}
	return os.WriteFile(path, b.Bytes(), 0o644)
}

func copyAssets(root, outDir string) error {
	for _, dir := range []string{"docs/assets", "assets"} {
		src := filepath.Join(root, dir)
		if _, err := os.Stat(src); err == nil {
			return copyDir(src, filepath.Join(outDir, "assets"))
		}
	}
	assetsDir := filepath.Join(outDir, "assets")
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(assetsDir, ".keep"), []byte(""), 0o644)
}

var (
	mdResourceRe       = regexp.MustCompile(`(!?\[[^\]]*\]\()([^)]+)(\))`)
	htmlResourceAttrRe = regexp.MustCompile(`(?i)\b(src|href)\s*=\s*(["'])([^"']+)(["'])`)
)

// bundleMarkdownResources copies each locally-referenced resource into the
// entry's site directory and rewrites its Markdown/HTML path to Modex's served
// URL. Remote, absolute, anchor-only, and markdown-page links are left intact.
func bundleMarkdownResources(text, srcFile, entryDir, base string) string {
	rewrite := func(raw string) (string, bool) {
		return bundleLocalResource(raw, srcFile, entryDir, base)
	}
	text = mdResourceRe.ReplaceAllStringFunc(text, func(m string) string {
		sub := mdResourceRe.FindStringSubmatch(m)
		if len(sub) < 4 {
			return m
		}
		next, ok := rewrite(strings.TrimSpace(sub[2]))
		if !ok {
			return m
		}
		return sub[1] + next + sub[3]
	})
	return htmlResourceAttrRe.ReplaceAllStringFunc(text, func(m string) string {
		sub := htmlResourceAttrRe.FindStringSubmatch(m)
		if len(sub) < 5 {
			return m
		}
		next, ok := rewrite(strings.TrimSpace(sub[3]))
		if !ok {
			return m
		}
		return sub[1] + "=" + sub[2] + next + sub[4]
	})
}

func bundleLocalResource(raw, srcFile, entryDir, base string) (string, bool) {
	pathPart, suffix := splitResourceSuffix(raw)
	if !isBundleableLocalResource(pathPart) {
		return raw, false
	}
	rel := strings.TrimPrefix(strings.ReplaceAll(pathPart, "\\", "/"), "./")
	src := filepath.Clean(filepath.Join(filepath.Dir(srcFile), filepath.FromSlash(rel)))
	if fi, err := os.Stat(src); err != nil || fi.IsDir() {
		return raw, false
	}
	destRel := outputResourcePath(rel)
	if err := copyFile(src, filepath.Join(entryDir, filepath.FromSlash(destRel))); err != nil {
		return raw, false
	}
	return base + destRel + suffix, true
}

func splitResourceSuffix(raw string) (string, string) {
	cut := len(raw)
	if idx := strings.IndexAny(raw, "?#"); idx >= 0 {
		cut = idx
	}
	return raw[:cut], raw[cut:]
}

func isBundleableLocalResource(path string) bool {
	if path == "" || strings.HasPrefix(path, "#") || strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		return false
	}
	low := strings.ToLower(path)
	if strings.HasPrefix(low, "http://") || strings.HasPrefix(low, "https://") || strings.HasPrefix(low, "data:") ||
		strings.HasPrefix(low, "mailto:") || strings.HasPrefix(low, "tel:") {
		return false
	}
	ext := strings.ToLower(filepath.Ext(path))
	return ext != ".md" && ext != ".mdx"
}

func outputResourcePath(rel string) string {
	clean := filepath.ToSlash(filepath.Clean(rel))
	for strings.HasPrefix(clean, "../") {
		clean = strings.TrimPrefix(clean, "../")
	}
	clean = strings.TrimPrefix(clean, "/")
	if clean == "." || clean == "" {
		return "asset"
	}
	return clean
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func copyDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", src)
	}
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
}

func extractHTMLText(dir string) string {
	var out strings.Builder
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(info.Name(), ".html") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err == nil {
			reTags := regexp.MustCompile(`<[^>]+>`)
			out.WriteString(reTags.ReplaceAllString(string(b), " "))
			out.WriteByte('\n')
		}
		return nil
	})
	return strings.Join(strings.Fields(out.String()), " ")
}
