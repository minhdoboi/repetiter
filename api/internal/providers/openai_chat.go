package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OpenAIChat struct {
	Name            string
	APIKey          string
	BaseURL         string
	Client          *http.Client
	ReasoningEffort string
}

func NewOpenAIChat(apiKey, baseURL string) *OpenAIChat {
	return &OpenAIChat{
		APIKey:  apiKey,
		BaseURL: strings.TrimRight(baseURL, "/"),
		Client:  &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *OpenAIChat) Complete(ctx context.Context, req TextRequest) (Suggestion, error) {
	if c.APIKey == "" {
		name := c.Name
		if name == "" {
			name = "text"
		}
		return Suggestion{}, fmt.Errorf("missing %s API key", name)
	}
	model := CanonicalTextModel(c.Name, req.Model)
	if model == "" {
		model = "gpt-4o-mini"
	}
	content, err := c.completeRaw(ctx, model, req, true)
	if err != nil {
		content, err = c.completeRaw(ctx, model, req, false)
		if err != nil {
			return Suggestion{}, err
		}
	}
	return ParseSuggestion(content)
}

func (c *OpenAIChat) completeRaw(ctx context.Context, model string, req TextRequest, jsonMode bool) (string, error) {
	body := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": SystemPrompt(req.SourceLang, req.TargetLang)},
			{"role": "user", "content": req.SourceText},
		},
		"temperature": 0.4,
	}
	if jsonMode {
		body["response_format"] = map[string]string{"type": "json_object"}
	}
	if c.ReasoningEffort != "" {
		body["reasoning_effort"] = c.ReasoningEffort
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("chat %s: %s", resp.Status, truncate(b, 400))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content json.RawMessage `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(b, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("empty chat response")
	}
	return extractChatContent(parsed.Choices[0].Message.Content)
}

func extractChatContent(raw json.RawMessage) (string, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return "", fmt.Errorf("empty chat content")
	}
	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return "", err
		}
		return s, nil
	}
	var chunks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &chunks); err != nil {
		return "", fmt.Errorf("chat content: %w", err)
	}
	var b strings.Builder
	for _, ch := range chunks {
		if ch.Text != "" {
			b.WriteString(ch.Text)
		}
	}
	s := strings.TrimSpace(b.String())
	if s == "" {
		return "", fmt.Errorf("empty chat content")
	}
	return s, nil
}

func truncate(b []byte, n int) string {
	s := strings.TrimSpace(string(b))
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
