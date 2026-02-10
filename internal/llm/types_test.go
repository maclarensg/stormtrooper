package llm

import (
	"encoding/json"
	"testing"
)

func TestMessage_MarshalJSON(t *testing.T) {
	msg := Message{
		Role:    "user",
		Content: "Hello",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var m map[string]interface{}
	json.Unmarshal(data, &m)

	if m["role"] != "user" {
		t.Errorf("expected role 'user'")
	}
	if m["content"] != "Hello" {
		t.Errorf("expected content 'Hello'")
	}
	// Omit empty fields
	if _, ok := m["tool_calls"]; ok {
		t.Error("expected tool_calls to be omitted when empty")
	}
	if _, ok := m["tool_call_id"]; ok {
		t.Error("expected tool_call_id to be omitted when empty")
	}
}

func TestMessage_ToolResultMarshal(t *testing.T) {
	msg := Message{
		Role:       "tool",
		Content:    "file contents here",
		ToolCallID: "call_123",
		Name:       "read_file",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var m map[string]interface{}
	json.Unmarshal(data, &m)

	if m["role"] != "tool" {
		t.Errorf("expected role 'tool'")
	}
	if m["tool_call_id"] != "call_123" {
		t.Errorf("expected tool_call_id 'call_123'")
	}
	if m["name"] != "read_file" {
		t.Errorf("expected name 'read_file'")
	}
}

func TestMessage_UnmarshalWithToolCalls(t *testing.T) {
	raw := `{
		"role": "assistant",
		"content": null,
		"tool_calls": [{
			"id": "call_abc",
			"type": "function",
			"function": {
				"name": "read_file",
				"arguments": "{\"file_path\":\"main.go\"}"
			}
		}]
	}`

	var msg Message
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if msg.Role != "assistant" {
		t.Errorf("expected role 'assistant', got %q", msg.Role)
	}
	if len(msg.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(msg.ToolCalls))
	}
	if msg.ToolCalls[0].Function.Name != "read_file" {
		t.Errorf("expected function name 'read_file', got %q", msg.ToolCalls[0].Function.Name)
	}
}

func TestChatCompletionRequest_MarshalWithTools(t *testing.T) {
	req := ChatCompletionRequest{
		Model: "test-model",
		Messages: []Message{{
			Role:    "user",
			Content: "Hello",
		}},
		Tools: []ToolDef{{
			Type: "function",
			Function: FunctionDef{
				Name:        "test_tool",
				Description: "A test tool",
				Parameters:  json.RawMessage(`{"type":"object"}`),
			},
		}},
		Stream: true,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var m map[string]interface{}
	json.Unmarshal(data, &m)

	if m["model"] != "test-model" {
		t.Error("expected model")
	}
	if m["stream"] != true {
		t.Error("expected stream true")
	}

	tools, ok := m["tools"].([]interface{})
	if !ok || len(tools) != 1 {
		t.Fatal("expected 1 tool definition")
	}
}
