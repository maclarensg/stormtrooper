// Package tool provides the tool registration, dispatch,
// and the standard set of built-in tools.
package tool

import (
	"context"
	"encoding/json"
)

// PermissionLevel controls whether a tool requires user approval.
type PermissionLevel int

const (
	PermissionAuto   PermissionLevel = iota // Runs without asking
	PermissionPrompt                        // Asks user before running
)

// Tool is the interface all tools must implement.
type Tool interface {
	Name() string
	Description() string
	Schema() json.RawMessage
	Permission() PermissionLevel
	Execute(ctx context.Context, params json.RawMessage) (string, error)
}

// Previewer is an optional interface that tools can implement to provide
// human-readable previews for permission prompts. Tools that require
// PermissionPrompt should implement this to show meaningful context
// (e.g., the command for shell_exec, a diff for edit_file) instead of
// raw JSON arguments.
type Previewer interface {
	Preview(params json.RawMessage) string
}

// ToolDef represents a tool definition in OpenAI function calling format.
type ToolDef struct {
	Type     string      `json:"type"`
	Function FunctionDef `json:"function"`
}

// FunctionDef describes a function for the LLM's tool use.
type FunctionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}
