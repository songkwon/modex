package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"modex/tools/docsctl/internal/docs"
)

func main() {
	if len(os.Args) < 2 {
		fatal("usage: docsctl validate|build|package|deploy")
	}
	root := env("DOCS_SOURCE_DIR", ".")
	buildDir := env("DOCS_BUILD_DIR", filepath.Join(root, ".modex", "build"))
	artifact := env("DOCS_ARTIFACT", filepath.Join(root, ".modex", "docs-artifact.zip"))
	switch os.Args[1] {
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
		must(deploy(artifact))
		fmt.Println("docsctl deploy ok")
	default:
		fatal("unknown command: " + os.Args[1])
	}
}

func deploy(artifact string) error {
	b, err := os.ReadFile(artifact)
	if err != nil {
		return err
	}
	url := env("DOCS_DEPLOY_URL", "http://localhost:8671/api/deploy")
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/zip")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("deploy failed: %s %s", resp.Status, string(body))
	}
	return nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
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
