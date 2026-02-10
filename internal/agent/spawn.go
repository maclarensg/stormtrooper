package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/gavinyap/stormtrooper/internal/llm"
	"github.com/gavinyap/stormtrooper/internal/permission"
	"github.com/gavinyap/stormtrooper/internal/tool"
)

// SpawnAgentTool creates and runs a sub-agent with a focused task.
type SpawnAgentTool struct {
	Client  *llm.Client
	Registry *tool.Registry
	Perm    *permission.Checker
	Model   string // parent's model as default
}

// NewSpawnAgentTool creates a spawn_agent tool with the given shared resources.
func NewSpawnAgentTool(client *llm.Client, registry *tool.Registry, perm *permission.Checker, defaultModel string) *SpawnAgentTool {
	return &SpawnAgentTool{
		Client:   client,
		Registry: registry,
		Perm:     perm,
		Model:    defaultModel,
	}
}

type spawnAgentParams struct {
	Task  string `json:"task"`
	Model string `json:"model"`
}

func (t *SpawnAgentTool) Name() string        { return "spawn_agent" }
func (t *SpawnAgentTool) Description() string { return "Spawn a sub-agent to work on a focused task" }
func (t *SpawnAgentTool) Permission() tool.PermissionLevel { return tool.PermissionPrompt }

func (t *SpawnAgentTool) Schema() json.RawMessage {
	return json.RawMessage(`{
	"type": "object",
	"properties": {
		"task": {
			"type": "string",
			"description": "The task description for the sub-agent"
		},
		"model": {
			"type": "string",
			"description": "Model to use for the sub-agent (optional, defaults to parent's model)"
		}
	},
	"required": ["task"]
}`)
}

// Preview returns a description for the permission prompt.
func (t *SpawnAgentTool) Preview(params json.RawMessage) string {
	var p spawnAgentParams
	if err := json.Unmarshal(params, &p); err != nil {
		return "Spawn sub-agent (invalid params)"
	}
	task := p.Task
	if len(task) > 80 {
		task = task[:80] + "..."
	}
	return fmt.Sprintf("Spawn sub-agent: %s", task)
}

func (t *SpawnAgentTool) Execute(ctx context.Context, params json.RawMessage) (string, error) {
	var p spawnAgentParams
	if err := json.Unmarshal(params, &p); err != nil {
		return fmt.Sprintf("Error: invalid parameters: %v", err), nil
	}
	if p.Task == "" {
		return "Error: task is required", nil
	}

	model := t.Model
	if p.Model != "" {
		model = p.Model
	}

	taskPreview := p.Task
	if len(taskPreview) > 80 {
		taskPreview = taskPreview[:80] + "..."
	}
	fmt.Fprintf(os.Stderr, "[agent] Spawning sub-agent: %s\n", taskPreview)

	systemPrompt := "You are a sub-agent. Complete the following task:\n\n" + p.Task + "\n\nWhen done, provide a concise summary of what you did and the results."

	// Create child agent
	child := New(Options{
		Client:       t.Client,
		Registry:     t.Registry,
		Permission:   t.Perm,
		Model:        model,
		SystemPrompt: systemPrompt,
	})

	// Capture child output
	var outputBuf bytes.Buffer
	child.SetOutput(&outputBuf, os.Stderr)

	// Run sub-agent in a goroutine and block on the result
	type result struct {
		output string
		err    error
	}
	ch := make(chan result, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				ch <- result{err: fmt.Errorf("sub-agent panic: %v", r)}
			}
		}()
		err := child.Send(ctx, p.Task)
		ch <- result{output: outputBuf.String(), err: err}
	}()

	// Block until sub-agent completes or context is cancelled
	select {
	case r := <-ch:
		fmt.Fprintf(os.Stderr, "[agent] Sub-agent completed\n")
		if r.err != nil {
			return fmt.Sprintf("Sub-agent error: %v", r.err), nil
		}
		if r.output == "" {
			return "Sub-agent completed with no output", nil
		}
		return r.output, nil
	case <-ctx.Done():
		return fmt.Sprintf("Sub-agent cancelled: %v", ctx.Err()), nil
	}
}

// prefixWriter wraps a writer and prefixes each write with a given string.
// Kept for future use if we want prefixed sub-agent output.
type prefixWriter struct {
	prefix string
	w      io.Writer
}

func (pw *prefixWriter) Write(p []byte) (int, error) {
	prefixed := append([]byte(pw.prefix), p...)
	_, err := pw.w.Write(prefixed)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}
