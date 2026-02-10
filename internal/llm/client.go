// Package llm provides the OpenRouter HTTP client with streaming
// and function calling support.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const defaultBaseURL = "https://openrouter.ai/api/v1"

// Client is an HTTP client for the OpenRouter chat completions API.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// NewClient creates a new LLM client with the given API key.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		http:    &http.Client{},
	}
}

// SetBaseURL overrides the default API base URL. Useful for testing.
func (c *Client) SetBaseURL(url string) {
	c.baseURL = url
}

// ChatCompletion sends a non-streaming chat completion request.
func (c *Client) ChatCompletion(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error) {
	req.Stream = false

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.setHeaders(httpReq)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, readAPIError(resp)
	}

	var result ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ChatCompletionStream sends a streaming chat completion request.
// The callback is called for each chunk as it arrives (for real-time display).
// Returns the fully accumulated assistant message after the stream ends.
func (c *Client) ChatCompletionStream(ctx context.Context, req ChatCompletionRequest, callback StreamCallback) (*Message, error) {
	req.Stream = true

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.setHeaders(httpReq)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, readAPIError(resp)
	}

	acc := NewDeltaAccumulator()

	err = ParseSSEStream(resp.Body, func(chunk ChatCompletionChunk) {
		acc.Add(chunk)
		if callback != nil {
			callback(chunk)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("stream error: %w", err)
	}

	msg := acc.Message()
	return &msg, nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/gavinyap/stormtrooper")
}

// APIError represents an error response from the API.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Body)
}

func readAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	return &APIError{
		StatusCode: resp.StatusCode,
		Body:       string(body),
	}
}
