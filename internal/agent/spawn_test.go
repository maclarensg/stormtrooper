package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gavinyap/stormtrooper/internal/permission"
	"github.com/gavinyap/stormtrooper/internal/tool"
)

func TestSpawnAgentToolInterface(t *testing.T) {
	var _ tool.Tool = &SpawnAgentTool{}

	st := &SpawnAgentTool{Model: "test-model"}
	if st.Name() != "spawn_agent" {
		t.Fatalf("expected name spawn_agent, got %s", st.Name())
	}
	if st.Permission() != tool.PermissionPrompt {
		t.Fatalf("expected PermissionPrompt, got %d", st.Permission())
	}

	var schema interface{}
	if err := json.Unmarshal(st.Schema(), &schema); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}
}

func TestSpawnAgentPreview(t *testing.T) {
	st := &SpawnAgentTool{Model: "test-model"}

	// Short task
	params, _ := json.Marshal(spawnAgentParams{Task: "Fix the bug"})
	preview := st.Preview(params)
	if !strings.Contains(preview, "Fix the bug") {
		t.Fatalf("preview should contain task, got %q", preview)
	}

	// Long task - should be truncated
	longTask := strings.Repeat("a", 100)
	params, _ = json.Marshal(spawnAgentParams{Task: longTask})
	preview = st.Preview(params)
	if !strings.Contains(preview, "...") {
		t.Fatalf("long task preview should be truncated, got %q", preview)
	}
}

func TestSpawnAgentEmptyTask(t *testing.T) {
	st := &SpawnAgentTool{Model: "test-model"}
	params, _ := json.Marshal(spawnAgentParams{Task: ""})
	result, err := st.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Error") {
		t.Fatalf("expected error for empty task, got %q", result)
	}
}

func TestSpawnAgentInvalidParams(t *testing.T) {
	st := &SpawnAgentTool{Model: "test-model"}
	result, err := st.Execute(context.Background(), json.RawMessage(`{invalid`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Error: invalid parameters") {
		t.Fatalf("expected invalid parameters error, got %q", result)
	}
}

func TestNewSpawnAgentTool(t *testing.T) {
	reg := tool.NewRegistry()
	in := strings.NewReader("n\n")
	perm := permission.NewCheckerWithIO(in, &strings.Builder{})

	st := NewSpawnAgentTool(nil, reg, perm, "test-model")
	if st == nil {
		t.Fatal("NewSpawnAgentTool returned nil")
	}
	if st.Model != "test-model" {
		t.Fatalf("expected model test-model, got %s", st.Model)
	}
	if st.Registry != reg {
		t.Fatal("registry not set correctly")
	}
	if st.Perm != perm {
		t.Fatal("permission checker not set correctly")
	}
}

func TestSpawnAgentContextCancellation(t *testing.T) {
	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// The spawn tool needs a client, but with cancelled context it should
	// fail quickly. We test that it handles cancellation gracefully.
	st := &SpawnAgentTool{Model: "test-model"}
	params, _ := json.Marshal(spawnAgentParams{Task: "do something"})
	result, err := st.Execute(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should either return a cancel message or an LLM error
	if !strings.Contains(result, "cancel") && !strings.Contains(result, "error") && !strings.Contains(result, "Error") {
		t.Fatalf("expected cancellation or error message, got %q", result)
	}
}
