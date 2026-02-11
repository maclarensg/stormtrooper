package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gavinyap/stormtrooper/internal/llm"
	"github.com/gavinyap/stormtrooper/internal/permission"
	"github.com/gavinyap/stormtrooper/internal/tool"
)

// mockTool implements tool.Tool for testing.
type mockTool struct {
	name       string
	perm       tool.PermissionLevel
	result     string
	err        error
	lastParams string
}

func (m *mockTool) Name() string                { return m.name }
func (m *mockTool) Description() string          { return "Mock tool" }
func (m *mockTool) Schema() json.RawMessage      { return json.RawMessage(`{"type":"object"}`) }
func (m *mockTool) Permission() tool.PermissionLevel { return m.perm }
func (m *mockTool) Execute(_ context.Context, params json.RawMessage) (string, error) {
	m.lastParams = string(params)
	return m.result, m.err
}

// sseResponse builds a complete SSE stream from chunks for test servers.
func sseTextResponse(content string) string {
	// Split content into a role chunk and a content chunk.
	var b strings.Builder
	b.WriteString(fmt.Sprintf("data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"\"},\"finish_reason\":null}]}\n\n"))
	b.WriteString(fmt.Sprintf("data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{\"content\":%s},\"finish_reason\":null}]}\n\n", jsonStr(content)))
	b.WriteString("data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
	b.WriteString("data: [DONE]\n")
	return b.String()
}

func sseToolCallResponse(callID, toolName, args string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"tool_calls\":[{\"index\":0,\"id\":%s,\"type\":\"function\",\"function\":{\"name\":%s,\"arguments\":%s}}]},\"finish_reason\":null}]}\n\n",
		jsonStr(callID), jsonStr(toolName), jsonStr(args)))
	b.WriteString("data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n")
	b.WriteString("data: [DONE]\n")
	return b.String()
}

func jsonStr(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func TestAgent_SimpleTextResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte(sseTextResponse("Hello there!")))
	}))
	defer server.Close()

	client := llm.NewClient("test-key")
	client.SetBaseURL(server.URL)

	reg := tool.NewRegistry()
	perm := permission.NewCheckerWithIO(strings.NewReader(""), &bytes.Buffer{})

	ag := New(Options{
		Client:       client,
		Registry:     reg,
		Permission:   perm,
		Model:        "test-model",
		SystemPrompt: "You are helpful.",
	})

	var stdout, stderr bytes.Buffer
	ag.SetOutput(&stdout, &stderr)

	err := ag.Send(context.Background(), "Hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout.String(), "Hello there!") {
		t.Errorf("expected 'Hello there!' in stdout, got %q", stdout.String())
	}
}

func TestAgent_ToolCallAndResponse(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "text/event-stream")

		if callCount == 1 {
			// First call: respond with a tool call
			w.Write([]byte(sseToolCallResponse("call_1", "test_tool", `{"input":"hello"}`)))
		} else {
			// Second call: respond with text (after processing tool result)
			w.Write([]byte(sseTextResponse("Tool result was: mock-result")))
		}
	}))
	defer server.Close()

	client := llm.NewClient("test-key")
	client.SetBaseURL(server.URL)

	reg := tool.NewRegistry()
	mt := &mockTool{name: "test_tool", perm: tool.PermissionAuto, result: "mock-result"}
	reg.Register(mt)

	perm := permission.NewCheckerWithIO(strings.NewReader(""), &bytes.Buffer{})

	ag := New(Options{
		Client:       client,
		Registry:     reg,
		Permission:   perm,
		Model:        "test-model",
		SystemPrompt: "You are helpful.",
	})

	var stdout, stderr bytes.Buffer
	ag.SetOutput(&stdout, &stderr)

	err := ag.Send(context.Background(), "Use the tool")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify tool was called
	if mt.lastParams != `{"input":"hello"}` {
		t.Errorf("expected tool params, got %q", mt.lastParams)
	}

	// Verify tool activity logged to stderr
	if !strings.Contains(stderr.String(), "[tool] test_tool") {
		t.Errorf("expected tool activity in stderr, got %q", stderr.String())
	}

	// Verify final text response
	if !strings.Contains(stdout.String(), "Tool result was: mock-result") {
		t.Errorf("expected final text in stdout, got %q", stdout.String())
	}

	// Verify it made 2 API calls
	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}
}

func TestAgent_UnknownTool(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "text/event-stream")

		if callCount == 1 {
			w.Write([]byte(sseToolCallResponse("call_1", "nonexistent_tool", `{}`)))
		} else {
			w.Write([]byte(sseTextResponse("I see the tool was not found.")))
		}
	}))
	defer server.Close()

	client := llm.NewClient("test-key")
	client.SetBaseURL(server.URL)

	reg := tool.NewRegistry()
	perm := permission.NewCheckerWithIO(strings.NewReader(""), &bytes.Buffer{})

	ag := New(Options{
		Client:     client,
		Registry:   reg,
		Permission: perm,
		Model:      "test-model",
	})

	var stdout, stderr bytes.Buffer
	ag.SetOutput(&stdout, &stderr)

	err := ag.Send(context.Background(), "Use a tool")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stderr.String(), "Unknown tool: nonexistent_tool") {
		t.Errorf("expected unknown tool warning in stderr, got %q", stderr.String())
	}
}

func TestAgent_PermissionDenied(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "text/event-stream")

		if callCount == 1 {
			w.Write([]byte(sseToolCallResponse("call_1", "dangerous_tool", `{"cmd":"rm -rf /"}`)))
		} else {
			w.Write([]byte(sseTextResponse("Permission was denied.")))
		}
	}))
	defer server.Close()

	client := llm.NewClient("test-key")
	client.SetBaseURL(server.URL)

	reg := tool.NewRegistry()
	mt := &mockTool{name: "dangerous_tool", perm: tool.PermissionPrompt, result: "should not see this"}
	reg.Register(mt)

	// Simulate user denying permission by providing "n".
	perm := permission.NewCheckerWithIO(strings.NewReader("n\n"), &bytes.Buffer{})

	ag := New(Options{
		Client:     client,
		Registry:   reg,
		Permission: perm,
		Model:      "test-model",
	})

	var stdout, stderr bytes.Buffer
	ag.SetOutput(&stdout, &stderr)

	err := ag.Send(context.Background(), "Do something dangerous")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify tool was NOT executed
	if mt.lastParams != "" {
		t.Errorf("tool should not have been executed, but got params: %q", mt.lastParams)
	}

	if !strings.Contains(stderr.String(), "permission denied") {
		t.Errorf("expected permission denied in stderr, got %q", stderr.String())
	}
}

func TestAgent_PermissionApproved(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "text/event-stream")

		if callCount == 1 {
			w.Write([]byte(sseToolCallResponse("call_1", "write_tool", `{"path":"test.txt"}`)))
		} else {
			w.Write([]byte(sseTextResponse("File written.")))
		}
	}))
	defer server.Close()

	client := llm.NewClient("test-key")
	client.SetBaseURL(server.URL)

	reg := tool.NewRegistry()
	mt := &mockTool{name: "write_tool", perm: tool.PermissionPrompt, result: "ok"}
	reg.Register(mt)

	// Simulate user approving permission.
	perm := permission.NewCheckerWithIO(strings.NewReader("y\n"), &bytes.Buffer{})

	ag := New(Options{
		Client:     client,
		Registry:   reg,
		Permission: perm,
		Model:      "test-model",
	})

	var stdout, stderr bytes.Buffer
	ag.SetOutput(&stdout, &stderr)

	err := ag.Send(context.Background(), "Write a file")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mt.lastParams != `{"path":"test.txt"}` {
		t.Errorf("expected tool to be executed with params, got %q", mt.lastParams)
	}
}

func TestAgent_SystemPromptInHistory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request includes system prompt.
		var req llm.ChatCompletionRequest
		json.NewDecoder(r.Body).Decode(&req)

		if len(req.Messages) < 2 {
			t.Errorf("expected at least 2 messages (system + user), got %d", len(req.Messages))
		}
		if req.Messages[0].Role != "system" {
			t.Errorf("expected first message to be system, got %q", req.Messages[0].Role)
		}
		if req.Messages[0].Content != "You are a test agent." {
			t.Errorf("expected system prompt content, got %q", req.Messages[0].Content)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte(sseTextResponse("ok")))
	}))
	defer server.Close()

	client := llm.NewClient("test-key")
	client.SetBaseURL(server.URL)

	reg := tool.NewRegistry()
	perm := permission.NewCheckerWithIO(strings.NewReader(""), &bytes.Buffer{})

	ag := New(Options{
		Client:       client,
		Registry:     reg,
		Permission:   perm,
		Model:        "test-model",
		SystemPrompt: "You are a test agent.",
	})

	var stdout bytes.Buffer
	ag.SetOutput(&stdout, &bytes.Buffer{})

	ag.Send(context.Background(), "Hi")
}

func TestAgent_CancelledContextReturnsError(t *testing.T) {
	// Create an already-cancelled context.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("LLM should not be called when context is already cancelled")
	}))
	defer server.Close()

	client := llm.NewClient("test-key")
	client.SetBaseURL(server.URL)

	reg := tool.NewRegistry()
	perm := permission.NewCheckerWithIO(strings.NewReader(""), &bytes.Buffer{})

	ag := New(Options{
		Client:     client,
		Registry:   reg,
		Permission: perm,
		Model:      "test-model",
	})

	var stdout, stderr bytes.Buffer
	ag.SetOutput(&stdout, &stderr)

	err := ag.Send(ctx, "Hi")
	if err == nil {
		t.Fatal("expected error when context is cancelled")
	}
	if !strings.Contains(err.Error(), "cancelled") {
		t.Errorf("expected 'cancelled' in error, got: %v", err)
	}
}

// sseToolCallWithContentResponse simulates an open-source model that sends
// tool call arguments as both Delta.Content and Delta.ToolCalls simultaneously.
func sseToolCallWithContentResponse(callID, toolName, args, leakedContent string) string {
	var b strings.Builder
	// Chunk that has both content and tool calls.
	b.WriteString(fmt.Sprintf(
		"data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":%s,\"tool_calls\":[{\"index\":0,\"id\":%s,\"type\":\"function\",\"function\":{\"name\":%s,\"arguments\":%s}}]},\"finish_reason\":null}]}\n\n",
		jsonStr(leakedContent), jsonStr(callID), jsonStr(toolName), jsonStr(args)))
	b.WriteString("data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n")
	b.WriteString("data: [DONE]\n")
	return b.String()
}

// sseContentWithSpecialTokens simulates a model emitting special tokens in content.
func sseContentWithSpecialTokens(content string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"\"},\"finish_reason\":null}]}\n\n"))
	b.WriteString(fmt.Sprintf("data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{\"content\":%s},\"finish_reason\":null}]}\n\n", jsonStr(content)))
	b.WriteString("data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
	b.WriteString("data: [DONE]\n")
	return b.String()
}

func TestAgent_StreamingFiltersToolCallContent(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "text/event-stream")

		if callCount == 1 {
			// Model sends tool call args as content AND in tool_calls.
			w.Write([]byte(sseToolCallWithContentResponse(
				"call_1", "read_file", `{"path":"main.go"}`, `{"path":"main.go"}`)))
		} else {
			w.Write([]byte(sseTextResponse("Here is the file.")))
		}
	}))
	defer server.Close()

	client := llm.NewClient("test-key")
	client.SetBaseURL(server.URL)

	reg := tool.NewRegistry()
	mt := &mockTool{name: "read_file", perm: tool.PermissionAuto, result: "file contents"}
	reg.Register(mt)

	perm := permission.NewCheckerWithIO(strings.NewReader(""), &bytes.Buffer{})

	ag := New(Options{
		Client:     client,
		Registry:   reg,
		Permission: perm,
		Model:      "test-model",
	})

	var stdout, stderr bytes.Buffer
	ag.SetOutput(&stdout, &stderr)

	err := ag.Send(context.Background(), "Read main.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The leaked JSON content should NOT appear in stdout.
	if strings.Contains(stdout.String(), `"path"`) {
		t.Errorf("expected tool call JSON to be filtered from stdout, got %q", stdout.String())
	}
	// The real text response should still appear.
	if !strings.Contains(stdout.String(), "Here is the file.") {
		t.Errorf("expected real text response in stdout, got %q", stdout.String())
	}
}

func TestAgent_StreamingStripsSpecialTokens(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte(sseContentWithSpecialTokens("Hello<|im_end|>")))
	}))
	defer server.Close()

	client := llm.NewClient("test-key")
	client.SetBaseURL(server.URL)

	reg := tool.NewRegistry()
	perm := permission.NewCheckerWithIO(strings.NewReader(""), &bytes.Buffer{})

	ag := New(Options{
		Client:     client,
		Registry:   reg,
		Permission: perm,
		Model:      "test-model",
	})

	var stdout bytes.Buffer
	ag.SetOutput(&stdout, &bytes.Buffer{})

	err := ag.Send(context.Background(), "Hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(stdout.String(), "<|im_end|>") {
		t.Errorf("expected special tokens to be stripped, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Hello") {
		t.Errorf("expected 'Hello' in output, got %q", stdout.String())
	}
}

func TestStripSpecialTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no tokens", "Hello world", "Hello world"},
		{"tool_call_end", "text<|tool_call_end|>", "text"},
		{"tool_call_start", "<|tool_call_start|>text", "text"},
		{"im_end", "Hello<|im_end|>", "Hello"},
		{"multiple tokens", "<|tool_call_start|>fn<|tool_sep|>args<|tool_call_end|>", "fnargs"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripSpecialTokens(tt.input)
			if got != tt.expected {
				t.Errorf("stripSpecialTokens(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestTruncateArgs(t *testing.T) {
	short := "short"
	if truncateArgs(short, 10) != "short" {
		t.Errorf("short string should not be truncated")
	}

	long := strings.Repeat("a", 300)
	result := truncateArgs(long, 200)
	if len(result) != 203 { // 200 + "..."
		t.Errorf("expected truncated length 203, got %d", len(result))
	}
	if !strings.HasSuffix(result, "...") {
		t.Error("expected ... suffix")
	}
}
