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
)

type Client struct {
	baseURL string
	apiKey  string
	model   string
	http    *http.Client
}

func NewClient(baseURL, apiKey, model string) *Client {
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		http:    &http.Client{},
	}
}

const maxRetries = 3

// retryable returns true for errors worth retrying (timeouts, 429, 5xx).
func retryable(err error, statusCode int) bool {
	if err != nil {
		return true // network errors, context deadline, etc.
	}
	return statusCode == 429 || statusCode >= 500
}

func (c *Client) Complete(ctx context.Context, messages []ChatMessage, maxTokens int) (string, error) {
	reqBody := ChatRequest{
		Model:       c.model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: 0.2,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	url := c.baseURL + "chat/completions"
	backoff := 5 * time.Second
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			fmt.Printf("Retry %d/%d after %s...\n", attempt, maxRetries, backoff)
			select {
			case <-ctx.Done():
				return "", fmt.Errorf("LLM request failed after %d attempts: %w", attempt-1, lastErr)
			case <-time.After(backoff):
			}
			backoff *= 2
		}

		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
		if err != nil {
			return "", fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if c.apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+c.apiKey)
		}

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("LLM request failed: %w", err)
			if retryable(err, 0) && attempt < maxRetries {
				continue
			}
			return "", lastErr
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return "", fmt.Errorf("reading response: %w", err)
		}

		if retryable(nil, resp.StatusCode) && attempt < maxRetries {
			lastErr = fmt.Errorf("LLM API error (%d): %s", resp.StatusCode, string(body))
			fmt.Printf("LLM returned %d, will retry\n", resp.StatusCode)
			continue
		}

		if resp.StatusCode >= 300 {
			return "", fmt.Errorf("LLM API error (%d): %s", resp.StatusCode, string(body))
		}

		var chatResp ChatResponse
		if err := json.Unmarshal(body, &chatResp); err != nil {
			return "", fmt.Errorf("parsing LLM response: %w", err)
		}

		if len(chatResp.Choices) == 0 {
			return "", fmt.Errorf("LLM returned no choices")
		}

		fmt.Printf("LLM usage: %d prompt tokens, %d completion tokens\n",
			chatResp.Usage.PromptTokens, chatResp.Usage.CompletionTokens)

		return chatResp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("LLM request failed after %d attempts: %w", maxRetries, lastErr)
}
