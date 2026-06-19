package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"modex/tools/docsctl/internal/docs"
)

// docsctlVersion is overridable at build time via -ldflags "-X main.docsctlVersion=...".
var docsctlVersion = "dev"

// options collects every setting docsctl understands. Each field is populated
// from a --flag, falling back to the matching DOCS_* environment variable so
// existing env-driven CI pipelines keep working unchanged.
type options struct {
	source   string
	buildDir string
	artifact string

	deployURL   string
	deployToken string

	module         string
	version        string
	packageVersion string
	description    string
	edition        string
	repoURL        string
	repoType       string
	branch         string
	commitSHA      string

	// Entry overrides used when there is no docs.yaml (config is synthesized).
	builder     string
	entryKey    string
	entryTitle  string
	entrySource string
	build       string
	output      string

	force bool
	write bool
	depth int
}

func main() {
	if len(os.Args) < 2 {
		fatal(usage)
	}
	cmd := os.Args[1]
	switch cmd {
	case "version", "--version", "-v":
		fmt.Println("docsctl", docsctlVersion)
		return
	case "help", "--help", "-h":
		fmt.Println(usage)
		return
	}

	opt := parseFlags(cmd, os.Args[2:])
	// Propagate flag values to the DOCS_* env vars consumed deep inside the docs
	// package (metadata resolution), so flags and env share one code path.
	opt.applyEnv()

	root := opt.source
	buildDir := opt.buildDir
	if buildDir == "" {
		buildDir = filepath.Join(root, ".modex", "build")
	}
	artifact := opt.artifact
	if artifact == "" {
		artifact = filepath.Join(root, ".modex", "docs-artifact.zip")
	}

	switch cmd {
	case "init":
		must(docs.Init(root, opt.force))
		fmt.Println("docsctl init ok:", filepath.Join(root, "docs.yaml"))
	case "discover":
		found, err := docs.Discover(root, opt.depth, opt.write, opt.force)
		must(err)
		for _, project := range found {
			status := "missing"
			if project.HasConfig {
				status = "ready"
			} else if project.Written {
				status = "created"
			}
			fmt.Printf("%s\t%s\t%s\t%s\n", status, project.Kind, project.RelPath, project.ConfigPath)
		}
		fmt.Printf("docsctl discover ok: %d projects\n", len(found))
	case "validate":
		must(docs.Validate(root))
		fmt.Println("docsctl validate ok")
	case "build":
		must(docs.Validate(root))
		must(docs.Build(root, buildDir))
		fmt.Println("docsctl build ok:", buildDir)
	case "package":
		if _, err := os.Stat(buildDir); err != nil {
			must(docs.Build(root, buildDir))
		}
		must(docs.Package(buildDir, artifact))
		fmt.Println("docsctl package ok:", artifact)
	case "deploy":
		must(deploy(root, buildDir, artifact, opt.deployURL, opt.deployToken))
		fmt.Println("docsctl deploy ok")
	default:
		fatal("unknown command: " + cmd + "\n\n" + usage)
	}
}

const usage = `usage: docsctl <command> [flags]

commands:
  version            print docsctl version
  init               create docs.yaml in the source directory
  discover           scan sub-projects and report/create docs.yaml
  validate           validate docs.yaml and entries
  build              validate + build entries into the build dir
  package            build (if needed) + zip into an artifact
  deploy             build + package + upload to a modex server

common flags:
  --source <dir>        doc source dir            (env DOCS_SOURCE_DIR, default ".")
  --build-dir <dir>     build output dir          (env DOCS_BUILD_DIR, default <source>/.modex/build)
  --artifact <file>     packaged zip path         (env DOCS_ARTIFACT, default <source>/.modex/docs-artifact.zip)

deploy flags:
  --deploy-url <url>    modex deploy endpoint      (env DOCS_DEPLOY_URL, default http://localhost:8671/api/deploy)
  --token <token>       deploy token              (env DOCS_DEPLOY_TOKEN)

metadata flags (override cbb.toml / env):
  --module <key>        module key & name         (env DOCS_MODULE)
  --version <ver>       docs version              (env DOCS_VERSION, default "latest")
  --package-version <v> package version          (env DOCS_PACKAGE_VERSION)
  --description <text>  module description        (env DOCS_DESCRIPTION)
  --edition <text>      edition                   (env DOCS_EDITION)
  --repo-url <url>      source repo url           (env DOCS_REPO_URL)
  --repo-type <type>    "git" | "svn"             (env DOCS_REPO_TYPE, default "git")
  --branch <name>       source branch             (env DOCS_BRANCH)
  --commit <sha>        source commit sha         (env DOCS_COMMIT_SHA)

entry flags (used when there is no docs.yaml):
  --builder <type>      markdown|vitepress|vuepress|fumadocs|docusaurus|mkdocs|honkit|gitbook|static (env DOCS_BUILDER)
  --entry-key <key>     entry key                 (env DOCS_ENTRY_KEY)
  --entry-title <text>  entry title               (env DOCS_ENTRY_TITLE)
  --entry-source <dir>  entry source path         (env DOCS_ENTRY_SOURCE)
  --build <cmd>         build command             (env DOCS_BUILD, e.g. "npm run docs:build")
  --output <dir>        built site dir (rel source) (env DOCS_OUTPUT, e.g. "dist")

init/discover flags:
  --force               overwrite existing docs.yaml (env DOCS_INIT_FORCE)
  --write               write docs.yaml during discover (env DOCS_DISCOVER_WRITE)
  --depth <n>           discover scan depth          (env DOCS_DISCOVER_DEPTH, default 4)`

func parseFlags(cmd string, args []string) options {
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	fs.Usage = func() { fmt.Fprintln(os.Stderr, usage) }
	var o options

	fs.StringVar(&o.source, "source", env("DOCS_SOURCE_DIR", "."), "doc source directory")
	fs.StringVar(&o.buildDir, "build-dir", env("DOCS_BUILD_DIR", ""), "build output directory")
	fs.StringVar(&o.artifact, "artifact", env("DOCS_ARTIFACT", ""), "packaged artifact path")

	fs.StringVar(&o.deployURL, "deploy-url", env("DOCS_DEPLOY_URL", "http://localhost:8671/api/deploy"), "modex deploy endpoint")
	fs.StringVar(&o.deployToken, "token", env("DOCS_DEPLOY_TOKEN", ""), "deploy token")

	fs.StringVar(&o.module, "module", env("DOCS_MODULE", ""), "module key & name")
	fs.StringVar(&o.version, "version", env("DOCS_VERSION", ""), "docs version")
	fs.StringVar(&o.packageVersion, "package-version", env("DOCS_PACKAGE_VERSION", ""), "package version")
	fs.StringVar(&o.description, "description", env("DOCS_DESCRIPTION", ""), "module description")
	fs.StringVar(&o.edition, "edition", env("DOCS_EDITION", ""), "edition")
	fs.StringVar(&o.repoURL, "repo-url", env("DOCS_REPO_URL", ""), "source repo url")
	fs.StringVar(&o.repoType, "repo-type", env("DOCS_REPO_TYPE", ""), "source repo type (git|svn)")
	fs.StringVar(&o.branch, "branch", env("DOCS_BRANCH", ""), "source branch")
	fs.StringVar(&o.commitSHA, "commit", env("DOCS_COMMIT_SHA", ""), "source commit sha")

	fs.StringVar(&o.builder, "builder", env("DOCS_BUILDER", ""), "doc builder type (markdown|vitepress|vuepress|fumadocs|docusaurus|mkdocs|honkit|gitbook|static)")
	fs.StringVar(&o.entryKey, "entry-key", env("DOCS_ENTRY_KEY", ""), "entry key (no docs.yaml)")
	fs.StringVar(&o.entryTitle, "entry-title", env("DOCS_ENTRY_TITLE", ""), "entry title (no docs.yaml)")
	fs.StringVar(&o.entrySource, "entry-source", env("DOCS_ENTRY_SOURCE", ""), "entry source path (no docs.yaml)")
	fs.StringVar(&o.build, "build", env("DOCS_BUILD", ""), "build command, e.g. 'npm run docs:build'")
	fs.StringVar(&o.output, "output", env("DOCS_OUTPUT", ""), "built site output dir, relative to source")

	fs.BoolVar(&o.force, "force", env("DOCS_INIT_FORCE", "false") == "true", "overwrite existing docs.yaml")
	fs.BoolVar(&o.write, "write", env("DOCS_DISCOVER_WRITE", "false") == "true", "write docs.yaml during discover")
	fs.IntVar(&o.depth, "depth", envInt("DOCS_DISCOVER_DEPTH", 4), "discover scan depth")

	_ = fs.Parse(args)
	return o
}

// applyEnv writes metadata flag values back into the DOCS_* env vars so the
// docs package's firstEnv() resolution picks them up. Only non-empty values are
// set, preserving any cbb.toml / pre-set env fallbacks for omitted flags.
func (o options) applyEnv() {
	setIf := func(key, val string) {
		if val != "" {
			_ = os.Setenv(key, val)
		}
	}
	setIf("DOCS_MODULE", o.module)
	setIf("DOCS_VERSION", o.version)
	setIf("DOCS_PACKAGE_VERSION", o.packageVersion)
	setIf("DOCS_DESCRIPTION", o.description)
	setIf("DOCS_EDITION", o.edition)
	setIf("DOCS_REPO_URL", o.repoURL)
	setIf("DOCS_REPO_TYPE", o.repoType)
	setIf("DOCS_BRANCH", o.branch)
	setIf("DOCS_COMMIT_SHA", o.commitSHA)

	setIf("DOCS_BUILDER", o.builder)
	setIf("DOCS_ENTRY_KEY", o.entryKey)
	setIf("DOCS_ENTRY_TITLE", o.entryTitle)
	setIf("DOCS_ENTRY_SOURCE", o.entrySource)
	setIf("DOCS_BUILD", o.build)
	setIf("DOCS_OUTPUT", o.output)
}

func deploy(root, buildDir, artifact, url, token string) error {
	if strings.TrimSpace(url) == "" {
		return fmt.Errorf("deploy url is required (set --deploy-url or DOCS_DEPLOY_URL)")
	}
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("deploy token is required (set --token or DOCS_DEPLOY_TOKEN; generate it from Modex admin)")
	}
	if _, err := os.Stat(artifact); err != nil {
		// Auto-build and package when the artifact is missing, so `docsctl deploy`
		// is the only command users need in CI/local workflows.
		if err := docs.Validate(root); err != nil {
			return err
		}
		if err := docs.Build(root, buildDir); err != nil {
			return err
		}
		if err := docs.Package(buildDir, artifact); err != nil {
			return err
		}
	}
	b, err := os.ReadFile(artifact)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), envDuration("DOCS_DEPLOY_TIMEOUT", 2*time.Minute))
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/zip")
	req.Header.Set("User-Agent", "docsctl/"+docsctlVersion)
	// Per-module / global deploy token (matches backend /api/deploy auth).
	req.Header.Set("X-Modex-Deploy-Token", token)
	fmt.Printf("docsctl deploy uploading %s (%d bytes) to %s\n", artifact, len(b), url)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("deploy request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("deploy failed: %s: %s", resp.Status, formatAPIError(body))
	}
	return nil
}

func formatAPIError(body []byte) string {
	var payload struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Deploy struct {
			Stages []struct {
				Name   string `json:"name"`
				Status string `json:"status"`
				Error  string `json:"error"`
				Note   string `json:"note"`
			} `json:"stages"`
		} `json:"deploy"`
	}
	if err := json.Unmarshal(body, &payload); err == nil && (payload.Error.Code != "" || payload.Error.Message != "") {
		msg := strings.TrimSpace(payload.Error.Code + ": " + payload.Error.Message)
		for _, st := range payload.Deploy.Stages {
			if st.Status == "failed" {
				msg += fmt.Sprintf(" (failed stage: %s", st.Name)
				if st.Error != "" {
					msg += ": " + st.Error
				}
				msg += ")"
				break
			}
		}
		return msg
	}
	return strings.TrimSpace(string(body))
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil || n <= 0 {
		return fallback
	}
	return n
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err == nil && d > 0 {
		return d
	}
	var seconds int
	if _, err := fmt.Sscanf(v, "%d", &seconds); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return fallback
}

func must(err error) {
	if err != nil {
		fatal(err.Error())
	}
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
