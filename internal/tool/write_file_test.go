package tool

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFileToolInterface(t *testing.T) {
	var _ Tool = &WriteFileTool{}

	tool := &WriteFileTool{}
	if tool.Name() != "write_file" {
		t.Fatalf("expected name write_file, got %s", tool.Name())
	}
	if tool.Permission() != PermissionPrompt {
		t.Fatalf("expected PermissionPrompt, got %d", tool.Permission())
	}

	var schema interface{}
	if err := json.Unmarshal(tool.Schema(), &schema); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}
}

func TestWriteFileSuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.txt")

	tool := &WriteFileTool{}
	params, _ := json.Marshal(writeFileParams{FilePath: path, Content: "hello world"})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "File written") {
		t.Fatalf("expected success message, got %q", result)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "hello world" {
		t.Fatalf("file contents don't match: got %q", string(data))
	}
}

func TestWriteFileCreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "c", "file.txt")

	tool := &WriteFileTool{}
	params, _ := json.Marshal(writeFileParams{FilePath: path, Content: "nested"})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "File written") {
		t.Fatalf("expected success message, got %q", result)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "nested" {
		t.Fatalf("file contents don't match: got %q", string(data))
	}
}

func TestWriteFileOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	os.WriteFile(path, []byte("old content"), 0644)

	tool := &WriteFileTool{}
	params, _ := json.Marshal(writeFileParams{FilePath: path, Content: "new content"})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "File written") {
		t.Fatalf("expected success message, got %q", result)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "new content" {
		t.Fatalf("file should be overwritten, got %q", string(data))
	}
}

func TestWriteFileEmptyPath(t *testing.T) {
	tool := &WriteFileTool{}
	params, _ := json.Marshal(writeFileParams{FilePath: "", Content: "test"})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Error") {
		t.Fatalf("expected error for empty path, got %q", result)
	}
}

func TestWriteFilePreview(t *testing.T) {
	tool := &WriteFileTool{}
	params, _ := json.Marshal(writeFileParams{FilePath: "/some/file.txt", Content: "hello"})
	preview := tool.Preview(params)
	if !strings.Contains(preview, "5 bytes") {
		t.Fatalf("preview should mention byte count, got %q", preview)
	}
	if !strings.Contains(preview, "/some/file.txt") {
		t.Fatalf("preview should mention file path, got %q", preview)
	}
}

func TestWriteFilePreviewOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "existing.txt")
	os.WriteFile(path, []byte("old"), 0644)

	tool := &WriteFileTool{}
	params, _ := json.Marshal(writeFileParams{FilePath: path, Content: "new"})
	preview := tool.Preview(params)
	if !strings.Contains(preview, "overwrite") {
		t.Fatalf("preview should mention overwrite, got %q", preview)
	}
}
