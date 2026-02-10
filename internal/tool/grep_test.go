package tool

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGrepToolInterface(t *testing.T) {
	var _ Tool = &GrepTool{}

	tool := &GrepTool{}
	if tool.Name() != "grep" {
		t.Fatalf("expected name grep, got %s", tool.Name())
	}
	if tool.Permission() != PermissionAuto {
		t.Fatalf("expected PermissionAuto, got %d", tool.Permission())
	}

	var schema interface{}
	if err := json.Unmarshal(tool.Schema(), &schema); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}
}

func TestGrepSingleFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	os.WriteFile(path, []byte("line one\nline two\nline three\n"), 0644)

	tool := &GrepTool{}
	params, _ := json.Marshal(grepParams{Pattern: "two", Path: path})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, ":2:") {
		t.Fatalf("expected line 2 match, got %q", result)
	}
	if !strings.Contains(result, "line two") {
		t.Fatalf("expected 'line two' in match, got %q", result)
	}
}

func TestGrepDirectory(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("func hello() {}\n"), 0644)
	os.WriteFile(filepath.Join(dir, "b.go"), []byte("func world() {}\n"), 0644)

	tool := &GrepTool{}
	params, _ := json.Marshal(grepParams{Pattern: "func", Path: dir})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "a.go") {
		t.Fatalf("expected a.go in results, got %q", result)
	}
	if !strings.Contains(result, "b.go") {
		t.Fatalf("expected b.go in results, got %q", result)
	}
}

func TestGrepWithIncludeFilter(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("func hello()\n"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("func world()\n"), 0644)

	tool := &GrepTool{}
	params, _ := json.Marshal(grepParams{Pattern: "func", Path: dir, Include: "*.go"})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "a.go") {
		t.Fatalf("expected a.go in results, got %q", result)
	}
	if strings.Contains(result, "b.txt") {
		t.Fatalf("should not match b.txt with include filter, got %q", result)
	}
}

func TestGrepNoMatch(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello world\n"), 0644)

	tool := &GrepTool{}
	params, _ := json.Marshal(grepParams{Pattern: "xyz123", Path: dir})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "No matches found") {
		t.Fatalf("expected no matches message, got %q", result)
	}
}

func TestGrepInvalidRegex(t *testing.T) {
	tool := &GrepTool{}
	params, _ := json.Marshal(grepParams{Pattern: "[invalid"})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Error: invalid regex") {
		t.Fatalf("expected regex error, got %q", result)
	}
}

func TestGrepSkipsHiddenDirs(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)
	os.WriteFile(filepath.Join(dir, ".git", "config"), []byte("find me\n"), 0644)
	os.WriteFile(filepath.Join(dir, "visible.txt"), []byte("find me\n"), 0644)

	tool := &GrepTool{}
	params, _ := json.Marshal(grepParams{Pattern: "find me", Path: dir})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "visible.txt") {
		t.Fatalf("expected visible.txt, got %q", result)
	}
	if strings.Contains(result, ".git") {
		t.Fatalf("should skip .git directory, got %q", result)
	}
}

func TestGrepSkipsBinaryFiles(t *testing.T) {
	dir := t.TempDir()
	// Create a binary file with null bytes
	binaryContent := []byte("find me\x00binary data")
	os.WriteFile(filepath.Join(dir, "binary.bin"), binaryContent, 0644)
	os.WriteFile(filepath.Join(dir, "text.txt"), []byte("find me\n"), 0644)

	tool := &GrepTool{}
	params, _ := json.Marshal(grepParams{Pattern: "find me", Path: dir})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "text.txt") {
		t.Fatalf("expected text.txt, got %q", result)
	}
	if strings.Contains(result, "binary.bin") {
		t.Fatalf("should skip binary files, got %q", result)
	}
}

func TestGrepEmptyPattern(t *testing.T) {
	tool := &GrepTool{}
	params, _ := json.Marshal(grepParams{Pattern: ""})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Error") {
		t.Fatalf("expected error for empty pattern, got %q", result)
	}
}

func TestIsBinary(t *testing.T) {
	dir := t.TempDir()

	textFile := filepath.Join(dir, "text.txt")
	os.WriteFile(textFile, []byte("hello world"), 0644)
	if isBinary(textFile) {
		t.Fatal("text file should not be detected as binary")
	}

	binFile := filepath.Join(dir, "binary.bin")
	os.WriteFile(binFile, []byte("hello\x00world"), 0644)
	if !isBinary(binFile) {
		t.Fatal("binary file should be detected as binary")
	}
}
