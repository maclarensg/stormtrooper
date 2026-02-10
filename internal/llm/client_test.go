package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChatCompletion_Success(t *testing.T) {
	resp := ChatCompletionResponse{
		ID: "resp-1",
		Choices: []Choice{{
			Index: 0,
			Message: Message{
				Role:    "assistant",
				Content: "Hello!",
			},
			FinishReason: "stop",
		}},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected auth header, got %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected json content type")
		}
		if r.Header.Get("HTTP-Referer") == "" {
			t.Errorf("expected HTTP-Referer header")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key")
	client.SetBaseURL(server.URL)

	result, err := client.ChatCompletion(context.Background(), ChatCompletionRequest{
		Model: "test-model",
		Messages: []Message{{
			Role:    "user",
			Content: "Hi",
		}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Choices[0].Message.Content != "Hello!" {
		t.Errorf("expected 'Hello!', got %q", result.Choices[0].Message.Content)
	}
}

func TestChatCompletion_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid api key"}`))
	}))
	defer server.Close()

	client := NewClient("bad-key")
	client.SetBaseURL(server.URL)

	_, err := client.ChatCompletion(context.Background(), ChatCompletionRequest{
		Model: "test-model",
		Messages: []Message{{
			Role:    "user",
			Content: "Hi",
		}},
	})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("expected status 401, got %d", apiErr.StatusCode)
	}
}

func TestChatCompletionStream_TextResponse(t *testing.T) {
	sseData := `data: {"id":"1","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}

data: {"id":"1","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}

data: {"id":"1","choices":[{"index":0,"delta":{"content":" there"},"finish_reason":null}]}

data: {"id":"1","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

data: [DONE]
`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte(sseData))
	}))
	defer server.Close()

	client := NewClient("test-key")
	client.SetBaseURL(server.URL)

	var callbackCount int
	msg, err := client.ChatCompletionStream(context.Background(), ChatCompletionRequest{
		Model: "test-model",
		Messages: []Message{{
			Role:    "user",
			Content: "Hi",
		}},
	}, func(chunk ChatCompletionChunk) {
		callbackCount++
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg.Role != "assistant" {
		t.Errorf("expected role 'assistant', got %q", msg.Role)
	}
	if msg.Content != "Hello there" {
		t.Errorf("expected 'Hello there', got %q", msg.Content)
	}
	if callbackCount != 4 {
		t.Errorf("expected 4 callbacks, got %d", callbackCount)
	}
}

func TestChatCompletionStream_ToolCall(t *testing.T) {
	sseData := `data: {"id":"1","choices":[{"index":0,"delta":{"role":"assistant","tool_calls":[{"index":0,"id":"call_abc","type":"function","function":{"name":"read_file","arguments":"{\"file"}}]},"finish_reason":null}]}

data: {"id":"1","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"_path\":\"test.go\"}"}}]},"finish_reason":null}]}

data: {"id":"1","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}

data: [DONE]
`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte(sseData))
	}))
	defer server.Close()

	client := NewClient("test-key")
	client.SetBaseURL(server.URL)

	msg, err := client.ChatCompletionStream(context.Background(), ChatCompletionRequest{
		Model: "test-model",
		Messages: []Message{{
			Role:    "user",
			Content: "Read test.go",
		}},
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(msg.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(msg.ToolCalls))
	}

	tc := msg.ToolCalls[0]
	if tc.ID != "call_abc" {
		t.Errorf("expected ID 'call_abc', got %q", tc.ID)
	}
	if tc.Function.Name != "read_file" {
		t.Errorf("expected name 'read_file', got %q", tc.Function.Name)
	}
	expected := `{"file_path":"test.go"}`
	if tc.Function.Arguments != expected {
		t.Errorf("expected arguments %q, got %q", expected, tc.Function.Arguments)
	}
}

func TestChatCompletionStream_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer server.Close()

	client := NewClient("test-key")
	client.SetBaseURL(server.URL)

	_, err := client.ChatCompletionStream(context.Background(), ChatCompletionRequest{
		Model: "test-model",
		Messages: []Message{{
			Role:    "user",
			Content: "Hi",
		}},
	}, nil)
	if err == nil {
		t.Fatal("expected error for 429 response")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 429 {
		t.Errorf("expected status 429, got %d", apiErr.StatusCode)
	}
}
