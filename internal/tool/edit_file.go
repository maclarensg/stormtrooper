package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// EditFileTool performs exact string replacement in a file.
type EditFileTool struct{}

type editFileParams struct {
	FilePath  string `json:"file_path"`
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
}

func (t *EditFileTool) Name() string        { return "edit_file" }
func (t *EditFileTool) Description() string { return "Replace an exact string in a file with new content" }
func (t *EditFileTool) Permission() PermissionLevel { return PermissionPrompt }

func (t *EditFileTool) Schema() json.RawMessage {
	return json.RawMessage(`{
	"type": "object",
	"properties": {
		"file_path": {
			"type": "string",
			"description": "Path to the file to edit"
		},
		"old_string": {
			"type": "string",
			"description": "The exact string to find and replace"
		},
		"new_string": {
			"type": "string",
			"description": "The replacement string"
		}
	},
	"required": ["file_path", "old_string", "new_string"]
}`)
}

// Preview returns a description for the permission prompt.
func (t *EditFileTool) Preview(params json.RawMessage) string {
	var p editFileParams
	if err := json.Unmarshal(params, &p); err != nil {
		return "Edit file (invalid params)"
	}
	return fmt.Sprintf("Edit %s\n--- old\n%s\n+++ new\n%s", p.FilePath, p.OldString, p.NewString)
}

func (t *EditFileTool) Execute(_ context.Context, params json.RawMessage) (string, error) {
	var p editFileParams
	if err := json.Unmarshal(params, &p); err != nil {
		return fmt.Sprintf("Error: invalid parameters: %v", err), nil
	}
	if p.FilePath == "" {
		return "Error: file_path is required", nil
	}
	if p.OldString == "" {
		return "Error: old_string is required", nil
	}

	data, err := os.ReadFile(p.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("Error: file not found: %s", p.FilePath), nil
		}
		return fmt.Sprintf("Error: %v", err), nil
	}

	content := string(data)
	count := strings.Count(content, p.OldString)

	switch count {
	case 0:
		return fmt.Sprintf("Error: old_string not found in %s", p.FilePath), nil
	case 1:
		// Exactly one match — proceed with replacement
	default:
		return fmt.Sprintf("Error: old_string found %d times in %s — provide more context to make it unique", count, p.FilePath), nil
	}

	newContent := strings.Replace(content, p.OldString, p.NewString, 1)
	if err := os.WriteFile(p.FilePath, []byte(newContent), 0644); err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	return fmt.Sprintf("File edited: %s", p.FilePath), nil
}
