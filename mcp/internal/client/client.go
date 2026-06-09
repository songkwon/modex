package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

func FromEnv() Client {
	return Client{
		BaseURL: env("DOCS_API_BASE_URL", "http://localhost:8080"),
		Token:   os.Getenv("MCP_TOKEN"),
		HTTP:    &http.Client{Timeout: 20 * time.Second},
	}
}

func (c Client) ListModules(categoryID, keyword string) (any, error) {
	u, _ := url.Parse(c.BaseURL + "/api/modules")
	q := u.Query()
	if categoryID != "" {
		q.Set("category_id", categoryID)
	}
	if keyword != "" {
		q.Set("keyword", keyword)
	}
	u.RawQuery = q.Encode()
	return c.get(u.String())
}

func (c Client) ListVersions(moduleKey string) (any, error) {
	return c.get(c.BaseURL + "/api/modules/" + url.PathEscape(moduleKey) + "/versions")
}

func (c Client) SearchDocs(req map[string]any) (any, error) {
	return c.post(c.BaseURL+"/api/search", req)
}

func (c Client) GetDocPage(docID string) (any, error) {
	return c.get(c.BaseURL + "/api/docs/page/" + url.PathEscape(docID))
}

func (c Client) LogMCP(toolName, query string, input any, resultCount int) {
	b, _ := json.Marshal(input)
	_, _ = c.post(c.BaseURL+"/api/mcp/log", map[string]any{
		"tool_name": toolName, "query": query, "input_json": string(b), "result_count": resultCount,
	})
}

func (c Client) get(endpoint string) (any, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

func (c Client) post(endpoint string, body any) (any, error) {
	b, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

func (c Client) do(req *http.Request) (any, error) {
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var decoded any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("api status %d: %v", resp.StatusCode, decoded)
	}
	return decoded, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
