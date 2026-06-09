package docs

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func Build(root, outDir string) error {
	cfg, err := LoadDocsYAML(filepath.Join(root, "docs.yaml"))
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
	var full strings.Builder
	for _, entry := range cfg.Entries {
		switch entry.Type {
		case "markdown":
			rec, navItem, text, err := buildMarkdown(root, outDir, md, entry)
			if err != nil {
				return err
			}
			records = append(records, rec)
			nav = append(nav, navItem)
			full.WriteString("# " + entry.Title + "\n\n" + text + "\n\n")
		case "static":
			rec, navItem, text, err := buildStatic(root, outDir, md, entry)
			if err != nil {
				return err
			}
			records = append(records, rec)
			nav = append(nav, navItem)
			full.WriteString("# " + entry.Title + "\n\n" + text + "\n\n")
		case "vuepress":
			if entry.Build != "" {
				cmd := exec.Command("sh", "-c", entry.Build)
				cmd.Dir = root
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					return err
				}
			}
			copyDir(filepath.Join(root, entry.Output), filepath.Join(outDir, "site", entry.Key))
			rec := recordFor(md, entry, "VuePress 文档已构建，详情见静态站点。")
			records = append(records, rec)
			nav = append(nav, NavItem{Title: entry.Title, Path: "/" + entry.Key})
		default:
			return fmt.Errorf("unsupported entry type %q", entry.Type)
		}
	}
	if err := writeJSON(filepath.Join(outDir, "manifest.json"), Manifest{SchemaVersion: "modex.docs/v1", GeneratedBy: "docsctl", Entries: cfg.Entries}); err != nil {
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

func buildMarkdown(root, outDir string, md Metadata, entry Entry) (DocumentRecord, NavItem, string, error) {
	b, err := os.ReadFile(filepath.Join(root, entry.Source))
	if err != nil {
		return DocumentRecord{}, NavItem{}, "", err
	}
	text := string(b)
	body := markdownToHTML(text)
	page := "<!doctype html><html><head><meta charset=\"utf-8\"><title>" + html.EscapeString(entry.Title) + "</title></head><body><main>" + body + "</main></body></html>"
	entryDir := filepath.Join(outDir, "site", entry.Key)
	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		return DocumentRecord{}, NavItem{}, "", err
	}
	if err := os.WriteFile(filepath.Join(entryDir, "index.html"), []byte(page), 0o644); err != nil {
		return DocumentRecord{}, NavItem{}, "", err
	}
	rec := recordFor(md, entry, stripMarkdown(text))
	return rec, NavItem{Title: entry.Title, Path: "/" + entry.Key, Children: headings(text)}, stripMarkdown(text), nil
}

func buildStatic(root, outDir string, md Metadata, entry Entry) (DocumentRecord, NavItem, string, error) {
	src := filepath.Join(root, entry.Source)
	dst := filepath.Join(outDir, "site", entry.Key)
	if err := copyDir(src, dst); err != nil {
		return DocumentRecord{}, NavItem{}, "", err
	}
	text := extractHTMLText(src)
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
		DocsVersion: md.DocsVersion, PackageVersion: md.PackageVersion, Channel: md.Channel, EntryKey: entry.Key,
		EntryType: entry.Type, Title: entry.Title, Description: desc, Content: content, Path: "/" + entry.Key,
		SourceFile: entry.Source, Keywords: md.Keywords, Status: "active",
	}
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
	b.WriteString("Channel: " + md.Channel + "\n")
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
