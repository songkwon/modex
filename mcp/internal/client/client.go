package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

type TokenInfo struct {
	UserID    string    `json:"user_id"`
	Scopes    []string  `json:"scopes"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (c Client) WithToken(token string) Client {
	c.Token = token
	return c
}

func FromEnv() Client {
	return Client{
		BaseURL: env("MODEX_API_BASE_URL", env("DOCS_API_BASE_URL", "http://localhost:8671")),
		Token:   os.Getenv("MODEX_MCP_TOKEN"),
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

func (c Client) VerifyToken(token string) (TokenInfo, error) {
	result, err := c.WithToken(token).get(c.BaseURL + "/api/mcp/token-info")
	if err != nil {
		return TokenInfo{}, err
	}
	data, err := json.Marshal(result)
	if err != nil {
		return TokenInfo{}, err
	}
	var info TokenInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return TokenInfo{}, err
	}
	if info.UserID == "" || info.ExpiresAt.IsZero() {
		return TokenInfo{}, fmt.Errorf("token info response is incomplete")
	}
	return info, nil
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
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var decoded any
	if len(bytes.TrimSpace(body)) > 0 {
		if err := json.Unmarshal(body, &decoded); err != nil {
			if resp.StatusCode >= 300 {
				return nil, fmt.Errorf("api status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
			}
			return nil, err
		}
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("api status %d: %s", resp.StatusCode, apiErrorMessage(decoded))
	}
	return decoded, nil
}

func apiErrorMessage(decoded any) string {
	if m, ok := decoded.(map[string]any); ok {
		if errObj, ok := m["error"].(map[string]any); ok {
			code, _ := errObj["code"].(string)
			message, _ := errObj["message"].(string)
			if code != "" || message != "" {
				return strings.TrimSpace(code + ": " + message)
			}
		}
	}
	return fmt.Sprint(decoded)
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
