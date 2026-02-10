package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

const maxReadSize = 100 * 1024 // 100KB

// ReadFileTool reads the contents of a file.
type ReadFileTool struct{}

type readFileParams struct {
	FilePath string `json:"file_path"`
}

func (t *ReadFileTool) Name() string        { return "read_file" }
func (t *ReadFileTool) Description() string { return "Read the contents of a file" }
func (t *ReadFileTool) Permission() PermissionLevel { return PermissionAuto }

func (t *ReadFileTool) Schema() json.RawMessage {
	return json.RawMessage(`{
	"type": "object",
	"properties": {
		"file_path": {
			"type": "string",
			"description": "Absolute or relative path to the file to read"
		}
	},
	"required": ["file_path"]
}`)
}

func (t *ReadFileTool) Execute(_ context.Context, params json.RawMessage) (string, error) {
	var p readFileParams
	if err := json.Unmarshal(params, &p); err != nil {
		return fmt.Sprintf("Error: invalid parameters: %v", err), nil
	}
	if p.FilePath == "" {
		return "Error: file_path is required", nil
	}

	info, err := os.Stat(p.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("Error: file not found: %s", p.FilePath), nil
		}
		return fmt.Sprintf("Error: %v", err), nil
	}
	if info.IsDir() {
		return fmt.Sprintf("Error: %s is a directory, not a file", p.FilePath), nil
	}

	data, err := os.ReadFile(p.FilePath)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}

	if len(data) > maxReadSize {
		return string(data[:maxReadSize]) + "\n\n[truncated — file exceeds 100KB]", nil
	}
	return string(data), nil
}
