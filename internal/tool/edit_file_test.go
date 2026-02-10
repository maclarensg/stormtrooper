package tool

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEditFileToolInterface(t *testing.T) {
	var _ Tool = &EditFileTool{}

	tool := &EditFileTool{}
	if tool.Name() != "edit_file" {
		t.Fatalf("expected name edit_file, got %s", tool.Name())
	}
	if tool.Permission() != PermissionPrompt {
		t.Fatalf("expected PermissionPrompt, got %d", tool.Permission())
	}

	var schema interface{}
	if err := json.Unmarshal(tool.Schema(), &schema); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}
}

func TestEditFileSuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	os.WriteFile(path, []byte("hello world"), 0644)

	tool := &EditFileTool{}
	params, _ := json.Marshal(editFileParams{
		FilePath:  path,
		OldString: "world",
		NewString: "go",
	})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "File edited") {
		t.Fatalf("expected success message, got %q", result)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "hello go" {
		t.Fatalf("expected 'hello go', got %q", string(data))
	}
}

func TestEditFileNotFound(t *testing.T) {
	tool := &EditFileTool{}
	params, _ := json.Marshal(editFileParams{
		FilePath:  "/nonexistent/file.txt",
		OldString: "old",
		NewString: "new",
	})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Error: file not found") {
		t.Fatalf("expected file not found error, got %q", result)
	}
}

func TestEditFileStringNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	os.WriteFile(path, []byte("hello world"), 0644)

	tool := &EditFileTool{}
	params, _ := json.Marshal(editFileParams{
		FilePath:  path,
		OldString: "xyz",
		NewString: "abc",
	})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "not found") {
		t.Fatalf("expected not found error, got %q", result)
	}
}

func TestEditFileMultipleMatches(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	os.WriteFile(path, []byte("aaa bbb aaa"), 0644)

	tool := &EditFileTool{}
	params, _ := json.Marshal(editFileParams{
		FilePath:  path,
		OldString: "aaa",
		NewString: "ccc",
	})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "2 times") {
		t.Fatalf("expected multiple match error, got %q", result)
	}

	// File should not have been modified
	data, _ := os.ReadFile(path)
	if string(data) != "aaa bbb aaa" {
		t.Fatalf("file should not have been modified, got %q", string(data))
	}
}

func TestEditFileEmptyOldString(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	os.WriteFile(path, []byte("hello"), 0644)

	tool := &EditFileTool{}
	params, _ := json.Marshal(editFileParams{
		FilePath:  path,
		OldString: "",
		NewString: "new",
	})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Error") {
		t.Fatalf("expected error for empty old_string, got %q", result)
	}
}

func TestEditFilePreview(t *testing.T) {
	tool := &EditFileTool{}
	params, _ := json.Marshal(editFileParams{
		FilePath:  "/some/file.go",
		OldString: "func old()",
		NewString: "func new()",
	})
	preview := tool.Preview(params)
	if !strings.Contains(preview, "func old()") {
		t.Fatalf("preview should show old string, got %q", preview)
	}
	if !strings.Contains(preview, "func new()") {
		t.Fatalf("preview should show new string, got %q", preview)
	}
}
