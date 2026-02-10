package tool

import (
	"context"
	"encoding/json"
	"testing"
)

// mockTool is a simple Tool implementation for testing.
type mockTool struct {
	name       string
	desc       string
	schema     json.RawMessage
	permission PermissionLevel
	execResult string
	execErr    error
}

func (m *mockTool) Name() string                                                   { return m.name }
func (m *mockTool) Description() string                                            { return m.desc }
func (m *mockTool) Schema() json.RawMessage                                        { return m.schema }
func (m *mockTool) Permission() PermissionLevel                                    { return m.permission }
func (m *mockTool) Execute(_ context.Context, _ json.RawMessage) (string, error) {
	return m.execResult, m.execErr
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}
	defs := r.Definitions()
	if len(defs) != 0 {
		t.Fatalf("expected 0 definitions, got %d", len(defs))
	}
}

func TestRegisterAndGet(t *testing.T) {
	r := NewRegistry()
	tool := &mockTool{
		name:   "test_tool",
		desc:   "A test tool",
		schema: json.RawMessage(`{"type":"object"}`),
	}

	r.Register(tool)

	got := r.Get("test_tool")
	if got == nil {
		t.Fatal("Get returned nil for registered tool")
	}
	if got.Name() != "test_tool" {
		t.Fatalf("expected name test_tool, got %s", got.Name())
	}
}

func TestGetUnknownTool(t *testing.T) {
	r := NewRegistry()
	got := r.Get("nonexistent")
	if got != nil {
		t.Fatal("expected nil for unknown tool, got non-nil")
	}
}

func TestDuplicateRegistrationPanics(t *testing.T) {
	r := NewRegistry()
	tool := &mockTool{name: "dup_tool"}

	r.Register(tool)

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on duplicate registration")
		}
	}()
	r.Register(tool)
}

func TestDefinitions(t *testing.T) {
	r := NewRegistry()
	schema1 := json.RawMessage(`{"type":"object","properties":{"a":{"type":"string"}}}`)
	schema2 := json.RawMessage(`{"type":"object","properties":{"b":{"type":"integer"}}}`)

	r.Register(&mockTool{name: "tool_a", desc: "Tool A", schema: schema1})
	r.Register(&mockTool{name: "tool_b", desc: "Tool B", schema: schema2})

	defs := r.Definitions()
	if len(defs) != 2 {
		t.Fatalf("expected 2 definitions, got %d", len(defs))
	}

	// Check order is preserved
	if defs[0].Function.Name != "tool_a" {
		t.Fatalf("expected first tool to be tool_a, got %s", defs[0].Function.Name)
	}
	if defs[1].Function.Name != "tool_b" {
		t.Fatalf("expected second tool to be tool_b, got %s", defs[1].Function.Name)
	}

	// Check type field
	if defs[0].Type != "function" {
		t.Fatalf("expected type 'function', got %s", defs[0].Type)
	}

	// Check descriptions
	if defs[0].Function.Description != "Tool A" {
		t.Fatalf("expected description 'Tool A', got %s", defs[0].Function.Description)
	}

	// Check schemas
	if string(defs[0].Function.Parameters) != string(schema1) {
		t.Fatalf("expected schema %s, got %s", string(schema1), string(defs[0].Function.Parameters))
	}
}

func TestDefinitionsJSON(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockTool{
		name:   "read_file",
		desc:   "Read a file",
		schema: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}`),
	})

	defs := r.Definitions()
	data, err := json.Marshal(defs)
	if err != nil {
		t.Fatalf("failed to marshal definitions: %v", err)
	}

	// Verify it's valid JSON and has expected structure
	var parsed []map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("definitions JSON is not valid: %v", err)
	}
	if len(parsed) != 1 {
		t.Fatalf("expected 1 definition in JSON, got %d", len(parsed))
	}
	if parsed[0]["type"] != "function" {
		t.Fatalf("expected type 'function', got %v", parsed[0]["type"])
	}
}
