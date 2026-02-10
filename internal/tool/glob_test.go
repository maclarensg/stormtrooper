package tool

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlobToolInterface(t *testing.T) {
	var _ Tool = &GlobTool{}

	tool := &GlobTool{}
	if tool.Name() != "glob" {
		t.Fatalf("expected name glob, got %s", tool.Name())
	}
	if tool.Permission() != PermissionAuto {
		t.Fatalf("expected PermissionAuto, got %d", tool.Permission())
	}

	var schema interface{}
	if err := json.Unmarshal(tool.Schema(), &schema); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}
}

func TestGlobSimplePattern(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("go"), 0644)
	os.WriteFile(filepath.Join(dir, "b.go"), []byte("go"), 0644)
	os.WriteFile(filepath.Join(dir, "c.txt"), []byte("txt"), 0644)

	tool := &GlobTool{}
	params, _ := json.Marshal(globParams{Pattern: "*.go", Path: dir})
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
	if strings.Contains(result, "c.txt") {
		t.Fatalf("should not match c.txt, got %q", result)
	}
}

func TestGlobRecursivePattern(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "sub", "deep"), 0755)
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("go"), 0644)
	os.WriteFile(filepath.Join(dir, "sub", "b.go"), []byte("go"), 0644)
	os.WriteFile(filepath.Join(dir, "sub", "deep", "c.go"), []byte("go"), 0644)
	os.WriteFile(filepath.Join(dir, "sub", "d.txt"), []byte("txt"), 0644)

	tool := &GlobTool{}
	params, _ := json.Marshal(globParams{Pattern: "**/*.go", Path: dir})
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
	if !strings.Contains(result, "c.go") {
		t.Fatalf("expected c.go in results, got %q", result)
	}
	if strings.Contains(result, "d.txt") {
		t.Fatalf("should not match d.txt, got %q", result)
	}
}

func TestGlobNoMatch(t *testing.T) {
	dir := t.TempDir()

	tool := &GlobTool{}
	params, _ := json.Marshal(globParams{Pattern: "*.xyz", Path: dir})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "No files matched") {
		t.Fatalf("expected no match message, got %q", result)
	}
}

func TestGlobSkipsHiddenDirs(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".hidden"), 0755)
	os.WriteFile(filepath.Join(dir, ".hidden", "secret.go"), []byte("go"), 0644)
	os.WriteFile(filepath.Join(dir, "visible.go"), []byte("go"), 0644)

	tool := &GlobTool{}
	params, _ := json.Marshal(globParams{Pattern: "**/*.go", Path: dir})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "visible.go") {
		t.Fatalf("expected visible.go, got %q", result)
	}
	if strings.Contains(result, "secret.go") {
		t.Fatalf("should not match files in hidden dirs, got %q", result)
	}
}

func TestGlobEmptyPattern(t *testing.T) {
	tool := &GlobTool{}
	params, _ := json.Marshal(globParams{Pattern: ""})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Error") {
		t.Fatalf("expected error for empty pattern, got %q", result)
	}
}
