package tool

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFileToolInterface(t *testing.T) {
	var _ Tool = &ReadFileTool{}

	tool := &ReadFileTool{}
	if tool.Name() != "read_file" {
		t.Fatalf("expected name read_file, got %s", tool.Name())
	}
	if tool.Permission() != PermissionAuto {
		t.Fatalf("expected PermissionAuto, got %d", tool.Permission())
	}

	var schema interface{}
	if err := json.Unmarshal(tool.Schema(), &schema); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}
}

func TestReadFileSuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello world"), 0644)

	tool := &ReadFileTool{}
	params, _ := json.Marshal(readFileParams{FilePath: path})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello world" {
		t.Fatalf("expected 'hello world', got %q", result)
	}
}

func TestReadFileNotFound(t *testing.T) {
	tool := &ReadFileTool{}
	params, _ := json.Marshal(readFileParams{FilePath: "/nonexistent/file.txt"})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Error: file not found") {
		t.Fatalf("expected file not found error, got %q", result)
	}
}

func TestReadFileDirectory(t *testing.T) {
	dir := t.TempDir()
	tool := &ReadFileTool{}
	params, _ := json.Marshal(readFileParams{FilePath: dir})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "is a directory") {
		t.Fatalf("expected directory error, got %q", result)
	}
}

func TestReadFileTruncation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.txt")
	data := make([]byte, maxReadSize+1000)
	for i := range data {
		data[i] = 'x'
	}
	os.WriteFile(path, data, 0644)

	tool := &ReadFileTool{}
	params, _ := json.Marshal(readFileParams{FilePath: path})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "[truncated") {
		t.Fatalf("expected truncation notice, got last 50 chars: %q", result[len(result)-50:])
	}
}

func TestReadFileEmptyPath(t *testing.T) {
	tool := &ReadFileTool{}
	params, _ := json.Marshal(readFileParams{FilePath: ""})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Error") {
		t.Fatalf("expected error for empty path, got %q", result)
	}
}

func TestReadFileInvalidParams(t *testing.T) {
	tool := &ReadFileTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{invalid`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Error: invalid parameters") {
		t.Fatalf("expected invalid parameters error, got %q", result)
	}
}
