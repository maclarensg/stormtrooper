package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WriteFileTool creates or overwrites a file.
type WriteFileTool struct{}

type writeFileParams struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

func (t *WriteFileTool) Name() string        { return "write_file" }
func (t *WriteFileTool) Description() string { return "Create or overwrite a file with the given content" }
func (t *WriteFileTool) Permission() PermissionLevel { return PermissionPrompt }

func (t *WriteFileTool) Schema() json.RawMessage {
	return json.RawMessage(`{
	"type": "object",
	"properties": {
		"file_path": {
			"type": "string",
			"description": "Path to the file to write"
		},
		"content": {
			"type": "string",
			"description": "The content to write to the file"
		}
	},
	"required": ["file_path", "content"]
}`)
}

// Preview returns a description for the permission prompt.
func (t *WriteFileTool) Preview(params json.RawMessage) string {
	var p writeFileParams
	if err := json.Unmarshal(params, &p); err != nil {
		return "Write file (invalid params)"
	}
	msg := fmt.Sprintf("Write %d bytes to %s", len(p.Content), p.FilePath)
	if _, err := os.Stat(p.FilePath); err == nil {
		msg += " (overwrite existing file)"
	}
	return msg
}

func (t *WriteFileTool) Execute(_ context.Context, params json.RawMessage) (string, error) {
	var p writeFileParams
	if err := json.Unmarshal(params, &p); err != nil {
		return fmt.Sprintf("Error: invalid parameters: %v", err), nil
	}
	if p.FilePath == "" {
		return "Error: file_path is required", nil
	}

	dir := filepath.Dir(p.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Sprintf("Error: failed to create directory %s: %v", dir, err), nil
	}

	if err := os.WriteFile(p.FilePath, []byte(p.Content), 0644); err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	return fmt.Sprintf("File written: %s", p.FilePath), nil
}
