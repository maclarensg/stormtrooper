package tool

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMemoryWriteToolInterface(t *testing.T) {
	var _ Tool = &MemoryWriteTool{}

	tool := &MemoryWriteTool{MemoryDir: "/tmp/mem"}
	if tool.Name() != "memory_write" {
		t.Fatalf("expected name memory_write, got %s", tool.Name())
	}
	if tool.Permission() != PermissionPrompt {
		t.Fatalf("expected PermissionPrompt, got %d", tool.Permission())
	}

	var schema interface{}
	if err := json.Unmarshal(tool.Schema(), &schema); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}
}

func TestMemoryWriteSuccess(t *testing.T) {
	dir := t.TempDir()
	memDir := filepath.Join(dir, "memory")
	os.MkdirAll(memDir, 0755)

	tool := &MemoryWriteTool{MemoryDir: memDir}
	params, _ := json.Marshal(memoryWriteParams{FilePath: "MEMORY.md", Content: "hello memory"})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Memory written") {
		t.Fatalf("expected success message, got %q", result)
	}

	data, _ := os.ReadFile(filepath.Join(memDir, "MEMORY.md"))
	if string(data) != "hello memory" {
		t.Fatalf("expected 'hello memory', got %q", string(data))
	}
}

func TestMemoryWriteCreatesSubdirectories(t *testing.T) {
	dir := t.TempDir()
	memDir := filepath.Join(dir, "memory")
	os.MkdirAll(memDir, 0755)

	tool := &MemoryWriteTool{MemoryDir: memDir}
	params, _ := json.Marshal(memoryWriteParams{FilePath: "notes/debug.md", Content: "debug notes"})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Memory written") {
		t.Fatalf("expected success message, got %q", result)
	}

	data, _ := os.ReadFile(filepath.Join(memDir, "notes", "debug.md"))
	if string(data) != "debug notes" {
		t.Fatalf("expected 'debug notes', got %q", string(data))
	}
}

func TestMemoryWritePathTraversal(t *testing.T) {
	dir := t.TempDir()
	memDir := filepath.Join(dir, "memory")
	os.MkdirAll(memDir, 0755)

	tool := &MemoryWriteTool{MemoryDir: memDir}
	params, _ := json.Marshal(memoryWriteParams{FilePath: "../../etc/passwd", Content: "hacked"})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Error") {
		t.Fatalf("expected error for path traversal, got %q", result)
	}
	if !strings.Contains(result, "escape") {
		t.Fatalf("expected escape error, got %q", result)
	}
}

func TestMemoryWriteEmptyPath(t *testing.T) {
	tool := &MemoryWriteTool{MemoryDir: "/tmp/mem"}
	params, _ := json.Marshal(memoryWriteParams{FilePath: "", Content: "test"})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Error") {
		t.Fatalf("expected error for empty path, got %q", result)
	}
}

func TestMemoryWritePreview(t *testing.T) {
	tool := &MemoryWriteTool{MemoryDir: "/project/.stormtrooper/memory"}
	params, _ := json.Marshal(memoryWriteParams{FilePath: "MEMORY.md", Content: "hello"})
	preview := tool.Preview(params)
	if !strings.Contains(preview, "5 bytes") {
		t.Fatalf("preview should mention byte count, got %q", preview)
	}
	if !strings.Contains(preview, "memory") {
		t.Fatalf("preview should mention memory, got %q", preview)
	}
}
