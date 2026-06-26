package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"modex/backend/internal/store"
)

// LLM API formats supported for the AI-ask feature. The frontend exposes these
// as selectable "API 格式" options so any mainstream provider can be wired in.
const (
	protoOpenAIChat      = "openai-chat"      // POST /chat/completions  (OpenAI & all compatible vendors)
	protoOpenAIResponses = "openai-responses" // POST /responses         (OpenAI Responses API)
	protoAnthropic       = "anthropic"        // POST /v1/messages       (Anthropic Messages)
	protoGemini          = "gemini"           // POST /v1beta/models/{m}:generateContent (Google Gemini)
)

func normalizeProtocol(p string) string {
	switch strings.TrimSpace(strings.ToLower(p)) {
	case protoOpenAIResponses:
		return protoOpenAIResponses
	case protoAnthropic:
		return protoAnthropic
	case protoGemini:
		return protoGemini
	default:
		return protoOpenAIChat
	}
}

// Engine defaults applied when the admin leaves the fields unset. max_tokens is
// generous so long answers are not truncated (Anthropic *requires* the field);
// temperature is low for grounded, deterministic doc answers.
const (
	defaultAskMaxTokens   = 4096
	defaultAskTemperature = 0.2
)

func maxTokensOf(ai store.AISettings) int {
	if ai.AskMaxTokens > 0 {
		return ai.AskMaxTokens
	}
	return defaultAskMaxTokens
}

func temperatureOf(ai store.AISettings) float64 {
	if ai.AskTemperature != nil {
		return *ai.AskTemperature
	}
	return defaultAskTemperature
}

// chatComplete sends a single-turn (system + user) completion request using the
// protocol configured in settings and returns the assistant's text.
func chatComplete(ctx context.Context, ai store.AISettings, system, user string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(ai.AskBaseURL), "/")
	temp := temperatureOf(ai)
	switch normalizeProtocol(ai.AskProtocol) {
	case protoAnthropic:
		// max_tokens is required by Anthropic; the others default to the model max.
		return chatAnthropic(ctx, base, ai.AskAPIKey, ai.AskModel, maxTokensOf(ai), temp, system, user)
	case protoGemini:
		return chatGemini(ctx, base, ai.AskAPIKey, ai.AskModel, temp, system, user)
	case protoOpenAIResponses:
		return chatOpenAIResponses(ctx, base, ai.AskAPIKey, ai.AskModel, temp, system, user)
	default:
		return chatOpenAIChat(ctx, base, ai.AskAPIKey, ai.AskModel, temp, system, user)
	}
}

func chatCompleteStream(ctx context.Context, ai store.AISettings, system, user string, onDelta func(string) bool) error {
	if normalizeProtocol(ai.AskProtocol) != protoOpenAIChat {
		return fmt.Errorf("streaming is only supported for OpenAI Chat compatible protocol")
	}
	base := strings.TrimRight(strings.TrimSpace(ai.AskBaseURL), "/")
	return chatOpenAIChatStream(ctx, base, ai.AskAPIKey, ai.AskModel, temperatureOf(ai), system, user, onDelta)
}

// listModels fetches available model ids from the provider for the given
// protocol so the admin never has to type a model name by hand.
func listModels(ctx context.Context, protocol, base, key string) ([]string, error) {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	switch normalizeProtocol(protocol) {
	case protoAnthropic:
		return modelsAnthropic(ctx, base, key)
	case protoGemini:
		return modelsGemini(ctx, base, key)
	default: // openai-chat & openai-responses share the /models listing
		return modelsOpenAI(ctx, base, key)
	}
}

func httpJSON(ctx context.Context, method, url string, headers map[string]string, body any) ([]byte, int, error) {
	var reader io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = strings.NewReader(string(raw))
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, 0, err
	}
	for k, v := range headers {
		if v != "" {
			req.Header.Set(k, v)
		}
	}
	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return raw, resp.StatusCode, nil
}

func chatOpenAIChatStream(ctx context.Context, base, key, model string, temperature float64, system, user string, onDelta func(string) bool) error {
	raw, _ := json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"temperature": temperature,
		"stream":      true,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/chat/completions", strings.NewReader(string(raw)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Authorization", bearer(key))
	}
	resp, err := (&http.Client{Timeout: 0}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("chat endpoint %d: %s", resp.StatusCode, string(body))
	}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		line = strings.TrimPrefix(line, "data:")
		line = strings.TrimSpace(line)
		if line == "[DONE]" {
			return nil
		}
		var event struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		for _, choice := range event.Choices {
			if choice.Delta.Content != "" && !onDelta(choice.Delta.Content) {
				return nil
			}
		}
	}
	return scanner.Err()
}

// ---- OpenAI Chat Completions ------------------------------------------------

func chatOpenAIChat(ctx context.Context, base, key, model string, temperature float64, system, user string) (string, error) {
	raw, code, err := httpJSON(ctx, http.MethodPost, base+"/chat/completions",
		map[string]string{"Content-Type": "application/json", "Authorization": bearer(key)},
		map[string]any{
			"model": model,
			"messages": []map[string]string{
				{"role": "system", "content": system},
				{"role": "user", "content": user},
			},
			"temperature": temperature,
			"stream":      false,
		})
	if err != nil {
		return "", err
	}
	if code >= 300 {
		return "", fmt.Errorf("chat endpoint %d: %s", code, string(raw))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("chat endpoint returned no choices")
	}
	return out.Choices[0].Message.Content, nil
}

// ---- OpenAI Responses API ---------------------------------------------------

func chatOpenAIResponses(ctx context.Context, base, key, model string, temperature float64, system, user string) (string, error) {
	raw, code, err := httpJSON(ctx, http.MethodPost, base+"/responses",
		map[string]string{"Content-Type": "application/json", "Authorization": bearer(key)},
		map[string]any{
			"model":        model,
			"instructions": system,
			"input":        user,
			"temperature":  temperature,
		})
	if err != nil {
		return "", err
	}
	if code >= 300 {
		return "", fmt.Errorf("responses endpoint %d: %s", code, string(raw))
	}
	// Prefer the flattened convenience field; fall back to walking output[].
	var out struct {
		OutputText string `json:"output_text"`
		Output     []struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if strings.TrimSpace(out.OutputText) != "" {
		return out.OutputText, nil
	}
	for _, o := range out.Output {
		for _, c := range o.Content {
			if c.Text != "" {
				return c.Text, nil
			}
		}
	}
	return "", fmt.Errorf("responses endpoint returned no text")
}

// ---- Anthropic Messages -----------------------------------------------------

// anthropicBase normalizes a base URL to the host root (Messages lives at
// /v1/messages), tolerating a base entered with or without a trailing /v1.
func anthropicBase(base string) string {
	return strings.TrimSuffix(strings.TrimRight(base, "/"), "/v1")
}

func chatAnthropic(ctx context.Context, base, key, model string, maxTokens int, temperature float64, system, user string) (string, error) {
	headers := map[string]string{
		"Content-Type":      "application/json",
		"x-api-key":         key,
		"anthropic-version": "2023-06-01",
	}
	raw, code, err := httpJSON(ctx, http.MethodPost, anthropicBase(base)+"/v1/messages", headers,
		map[string]any{
			"model":       model,
			"max_tokens":  maxTokens,
			"temperature": temperature,
			"system":      system,
			"messages": []map[string]string{
				{"role": "user", "content": user},
			},
		})
	if err != nil {
		return "", err
	}
	if code >= 300 {
		return "", fmt.Errorf("anthropic endpoint %d: %s", code, string(raw))
	}
	var out struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	for _, c := range out.Content {
		if c.Type == "text" && c.Text != "" {
			return c.Text, nil
		}
	}
	return "", fmt.Errorf("anthropic endpoint returned no text")
}

// ---- Google Gemini ----------------------------------------------------------

// geminiBase normalizes to the API root; generateContent lives under
// /v1beta/models/{model}:generateContent.
func geminiBase(base string) string {
	b := strings.TrimRight(base, "/")
	b = strings.TrimSuffix(b, "/v1beta")
	b = strings.TrimSuffix(b, "/v1")
	return b
}

func chatGemini(ctx context.Context, base, key, model string, temperature float64, system, user string) (string, error) {
	// Pass the key via the x-goog-api-key header rather than a ?key= query param
	// so it does not end up in proxy/gateway access logs.
	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent", geminiBase(base), model)
	raw, code, err := httpJSON(ctx, http.MethodPost, url,
		map[string]string{"Content-Type": "application/json", "x-goog-api-key": key},
		map[string]any{
			"systemInstruction": map[string]any{"parts": []map[string]string{{"text": system}}},
			"contents": []map[string]any{
				{"role": "user", "parts": []map[string]string{{"text": user}}},
			},
			"generationConfig": map[string]any{"temperature": temperature},
		})
	if err != nil {
		return "", err
	}
	if code >= 300 {
		return "", fmt.Errorf("gemini endpoint %d: %s", code, string(raw))
	}
	var out struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	for _, c := range out.Candidates {
		for _, p := range c.Content.Parts {
			if p.Text != "" {
				return p.Text, nil
			}
		}
	}
	return "", fmt.Errorf("gemini endpoint returned no text")
}

// ---- Model listing per protocol ---------------------------------------------

func modelsOpenAI(ctx context.Context, base, key string) ([]string, error) {
	raw, code, err := httpJSON(ctx, http.MethodGet, base+"/models",
		map[string]string{"Authorization": bearer(key)}, nil)
	if err != nil {
		return nil, err
	}
	if code >= 300 {
		return nil, fmt.Errorf("%d: %s", code, string(raw))
	}
	var parsed struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(raw, &parsed)
	ids := make([]string, 0, len(parsed.Data))
	for _, m := range parsed.Data {
		if m.ID != "" {
			ids = append(ids, m.ID)
		}
	}
	return ids, nil
}

func modelsAnthropic(ctx context.Context, base, key string) ([]string, error) {
	headers := map[string]string{"x-api-key": key, "anthropic-version": "2023-06-01"}
	root := anthropicBase(base)
	ids := make([]string, 0, 16)
	afterID := ""
	// The models list is paginated; follow has_more / last_id so every model is
	// returned, not just the first page. Cap the loop defensively.
	for page := 0; page < 20; page++ {
		url := root + "/v1/models?limit=1000"
		if afterID != "" {
			url += "&after_id=" + afterID
		}
		raw, code, err := httpJSON(ctx, http.MethodGet, url, headers, nil)
		if err != nil {
			return nil, err
		}
		if code >= 300 {
			return nil, fmt.Errorf("%d: %s", code, string(raw))
		}
		var parsed struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
			HasMore bool   `json:"has_more"`
			LastID  string `json:"last_id"`
		}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return nil, err
		}
		for _, m := range parsed.Data {
			if m.ID != "" {
				ids = append(ids, m.ID)
			}
		}
		if !parsed.HasMore || parsed.LastID == "" {
			break
		}
		afterID = parsed.LastID
	}
	return ids, nil
}

func modelsGemini(ctx context.Context, base, key string) ([]string, error) {
	raw, code, err := httpJSON(ctx, http.MethodGet,
		geminiBase(base)+"/v1beta/models?pageSize=1000",
		map[string]string{"x-goog-api-key": key}, nil)
	if err != nil {
		return nil, err
	}
	if code >= 300 {
		return nil, fmt.Errorf("%d: %s", code, string(raw))
	}
	var parsed struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	_ = json.Unmarshal(raw, &parsed)
	ids := make([]string, 0, len(parsed.Models))
	for _, m := range parsed.Models {
		ids = append(ids, strings.TrimPrefix(m.Name, "models/"))
	}
	return ids, nil
}

func bearer(key string) string {
	if key == "" {
		return ""
	}
	return "Bearer " + key
}
