package tool

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestShellExecToolInterface(t *testing.T) {
	var _ Tool = &ShellExecTool{}

	tool := &ShellExecTool{}
	if tool.Name() != "shell_exec" {
		t.Fatalf("expected name shell_exec, got %s", tool.Name())
	}
	if tool.Permission() != PermissionPrompt {
		t.Fatalf("expected PermissionPrompt, got %d", tool.Permission())
	}

	var schema interface{}
	if err := json.Unmarshal(tool.Schema(), &schema); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}
}

func TestShellExecSuccess(t *testing.T) {
	tool := &ShellExecTool{}
	params, _ := json.Marshal(shellExecParams{Command: "echo hello"})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(result) != "hello" {
		t.Fatalf("expected 'hello', got %q", result)
	}
}

func TestShellExecNonZeroExit(t *testing.T) {
	tool := &ShellExecTool{}
	params, _ := json.Marshal(shellExecParams{Command: "exit 42"})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Exit code: 42") {
		t.Fatalf("expected exit code 42, got %q", result)
	}
}

func TestShellExecTimeout(t *testing.T) {
	tool := &ShellExecTool{}
	params, _ := json.Marshal(shellExecParams{Command: "sleep 10", Timeout: 1})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "timed out") {
		t.Fatalf("expected timeout message, got %q", result)
	}
}

func TestShellExecEmptyCommand(t *testing.T) {
	tool := &ShellExecTool{}
	params, _ := json.Marshal(shellExecParams{Command: ""})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Error") {
		t.Fatalf("expected error for empty command, got %q", result)
	}
}

func TestShellExecPreview(t *testing.T) {
	tool := &ShellExecTool{}
	params, _ := json.Marshal(shellExecParams{Command: "ls -la"})
	preview := tool.Preview(params)
	if !strings.Contains(preview, "ls -la") {
		t.Fatalf("preview should contain command, got %q", preview)
	}
}

func TestShellExecCapturesStderr(t *testing.T) {
	tool := &ShellExecTool{}
	params, _ := json.Marshal(shellExecParams{Command: "echo err >&2"})
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "err") {
		t.Fatalf("expected stderr output, got %q", result)
	}
}
