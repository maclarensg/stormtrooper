package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

const (
	defaultTimeout = 30 * time.Second
	maxTimeout     = 300 * time.Second
	maxOutputSize  = 50 * 1024 // 50KB
)

// ShellExecTool runs shell commands.
type ShellExecTool struct{}

type shellExecParams struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
}

func (t *ShellExecTool) Name() string        { return "shell_exec" }
func (t *ShellExecTool) Description() string { return "Execute a shell command and return its output" }
func (t *ShellExecTool) Permission() PermissionLevel { return PermissionPrompt }

func (t *ShellExecTool) Schema() json.RawMessage {
	return json.RawMessage(`{
	"type": "object",
	"properties": {
		"command": {
			"type": "string",
			"description": "The shell command to execute"
		},
		"timeout": {
			"type": "integer",
			"description": "Timeout in seconds (default 30)"
		}
	},
	"required": ["command"]
}`)
}

// Preview returns the command string for the permission prompt.
func (t *ShellExecTool) Preview(params json.RawMessage) string {
	var p shellExecParams
	if err := json.Unmarshal(params, &p); err != nil {
		return "Run command (invalid params)"
	}
	return fmt.Sprintf("Run command: %s", p.Command)
}

func (t *ShellExecTool) Execute(ctx context.Context, params json.RawMessage) (string, error) {
	var p shellExecParams
	if err := json.Unmarshal(params, &p); err != nil {
		return fmt.Sprintf("Error: invalid parameters: %v", err), nil
	}
	if p.Command == "" {
		return "Error: command is required", nil
	}

	timeout := defaultTimeout
	if p.Timeout > 0 {
		timeout = time.Duration(p.Timeout) * time.Second
		if timeout > maxTimeout {
			timeout = maxTimeout
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", p.Command)
	output, err := cmd.CombinedOutput()

	// Truncate if too large
	truncated := false
	if len(output) > maxOutputSize {
		output = output[:maxOutputSize]
		truncated = true
	}

	result := string(output)
	if truncated {
		result += "\n\n[truncated — output exceeds 50KB]"
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Sprintf("Command timed out after %ds\n%s", int(timeout.Seconds()), result), nil
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Sprintf("Exit code: %d\n%s", exitErr.ExitCode(), result), nil
		}
		return fmt.Sprintf("Error: %v\n%s", err, result), nil
	}

	return result, nil
}
