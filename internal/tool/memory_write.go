package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MemoryWriteTool writes content to the memory directory.
type MemoryWriteTool struct {
	MemoryDir string // Absolute path to .stormtrooper/memory/
}

type memoryWriteParams struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

func (t *MemoryWriteTool) Name() string        { return "memory_write" }
func (t *MemoryWriteTool) Description() string { return "Write content to a memory file for persistent storage across sessions" }
func (t *MemoryWriteTool) Permission() PermissionLevel { return PermissionPrompt }

func (t *MemoryWriteTool) Schema() json.RawMessage {
	return json.RawMessage(`{
	"type": "object",
	"properties": {
		"file_path": {
			"type": "string",
			"description": "Path relative to .stormtrooper/memory/ (e.g., 'MEMORY.md' or 'notes/debug.md')"
		},
		"content": {
			"type": "string",
			"description": "Content to write to the memory file"
		}
	},
	"required": ["file_path", "content"]
}`)
}

// Preview returns a description for the permission prompt.
func (t *MemoryWriteTool) Preview(params json.RawMessage) string {
	var p memoryWriteParams
	if err := json.Unmarshal(params, &p); err != nil {
		return "Write memory file (invalid params)"
	}
	resolved := filepath.Join(t.MemoryDir, p.FilePath)
	return fmt.Sprintf("Write %d bytes to memory: %s", len(p.Content), resolved)
}

func (t *MemoryWriteTool) Execute(_ context.Context, params json.RawMessage) (string, error) {
	var p memoryWriteParams
	if err := json.Unmarshal(params, &p); err != nil {
		return fmt.Sprintf("Error: invalid parameters: %v", err), nil
	}
	if p.FilePath == "" {
		return "Error: file_path is required", nil
	}

	// Resolve and validate path
	resolved := filepath.Join(t.MemoryDir, p.FilePath)
	resolved, err := filepath.Abs(resolved)
	if err != nil {
		return fmt.Sprintf("Error: invalid path: %v", err), nil
	}

	absMemDir, err := filepath.Abs(t.MemoryDir)
	if err != nil {
		return fmt.Sprintf("Error: invalid memory directory: %v", err), nil
	}

	// Path traversal protection
	if !strings.HasPrefix(resolved, absMemDir+string(filepath.Separator)) && resolved != absMemDir {
		return "Error: file_path must not escape the memory directory", nil
	}

	// Create parent directories
	dir := filepath.Dir(resolved)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Sprintf("Error: failed to create directory: %v", err), nil
	}

	if err := os.WriteFile(resolved, []byte(p.Content), 0644); err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	return fmt.Sprintf("Memory written: %s", resolved), nil
}
