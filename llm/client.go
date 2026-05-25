package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/yoheizuho/translate/config"
)

type Client struct {
	BaseURL    string
	APIKey     string
	Model      string
	HTTPClient *http.Client
	params     config.LLMParams
}

func NewClient(baseURL, apiKey, model string, timeoutSec int, params config.LLMParams) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		Model:   model,
		params:  params,
		HTTPClient: &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
		},
	}
}

type chatRequest struct {
	Model             string        `json:"model"`
	Messages          []chatMessage `json:"messages"`
	Stream            bool          `json:"stream"`
	MaxTokens         int           `json:"max_tokens,omitempty"`
	Temperature       *float64      `json:"temperature,omitempty"`
	TopP              *float64      `json:"top_p,omitempty"`
	TopK              *int          `json:"top_k,omitempty"`
	RepetitionPenalty *float64      `json:"repetition_penalty,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Chat sends messages to the OpenAI-compatible API.
// If system is empty, only a user message is sent (required for Hy-MT2).
func (c *Client) Chat(ctx context.Context, system, user string) (string, error) {
	var messages []chatMessage
	if system != "" {
		messages = append(messages, chatMessage{Role: "system", Content: system})
	}
	messages = append(messages, chatMessage{Role: "user", Content: user})

	payload := chatRequest{
		Model:             c.Model,
		Messages:          messages,
		Stream:            false,
		MaxTokens:         c.params.MaxTokens,
		Temperature:       c.params.Temperature,
		TopP:              c.params.TopP,
		TopK:              c.params.TopK,
		RepetitionPenalty: c.params.RepetitionPenalty,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upstream returned %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	if chatResp.Error != nil {
		return "", fmt.Errorf("upstream error: %s", chatResp.Error.Message)
	}
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
}

// Translate uses the Hy-MT2 prompt format (instruction in user message, no system prompt).
// This format also works well with general-purpose LLMs.
func (c *Client) Translate(ctx context.Context, text, sourceLang, targetLang string) (string, error) {
	var prompt string
	if sourceLang == "auto" || sourceLang == "" {
		prompt = fmt.Sprintf(
			"Translate the following text into %s. Note that you should only output the translated result without any additional explanation:\n\n%s",
			targetLang, text,
		)
	} else {
		prompt = fmt.Sprintf(
			"Translate the following text from %s into %s. Note that you should only output the translated result without any additional explanation:\n\n%s",
			sourceLang, targetLang, text,
		)
	}
	// No system prompt — compatible with Hy-MT2 and general LLMs
	return c.Chat(ctx, "", prompt)
}

func (c *Client) DetectLanguage(ctx context.Context, text string) (string, float64, error) {
	sample := text
	if len([]rune(sample)) > 300 {
		runes := []rune(text)
		sample = string(runes[:300])
	}

	prompt := fmt.Sprintf(
		"Detect the language of the following text. Respond with ONLY the ISO 639-1 two-letter language code (e.g. 'en', 'ja', 'fr'). Output nothing else.\n\n%s",
		sample,
	)
	result, err := c.Chat(ctx, "", prompt)
	if err != nil {
		return "", 0, err
	}

	code := strings.ToLower(strings.TrimSpace(result))
	code = strings.Trim(code, ".,!?\"'`")
	if idx := strings.IndexAny(code, " \t\n"); idx > 0 {
		code = code[:idx]
	}

	return code, 0.9, nil
}
