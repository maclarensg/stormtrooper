// Package agent implements the conversation loop that sends messages
// to the LLM, handles tool calls, and manages conversation history.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/gavinyap/stormtrooper/internal/llm"
	"github.com/gavinyap/stormtrooper/internal/permission"
	"github.com/gavinyap/stormtrooper/internal/tool"
)

// Agent orchestrates a conversation with an LLM, dispatching tool calls
// and maintaining history.
type Agent struct {
	client     *llm.Client
	registry   *tool.Registry
	permission *permission.Checker
	model      string
	history    []llm.Message
	stdout     io.Writer
	stderr     io.Writer
}

// Options configures a new Agent.
type Options struct {
	Client       *llm.Client
	Registry     *tool.Registry
	Permission   *permission.Checker
	Model        string
	SystemPrompt string
}

// New creates an Agent with the given options.
// If SystemPrompt is non-empty, it is prepended to the conversation history.
func New(opts Options) *Agent {
	a := &Agent{
		client:     opts.Client,
		registry:   opts.Registry,
		permission: opts.Permission,
		model:      opts.Model,
		stdout:     os.Stdout,
		stderr:     os.Stderr,
	}

	if opts.SystemPrompt != "" {
		a.history = append(a.history, llm.Message{
			Role:    "system",
			Content: opts.SystemPrompt,
		})
	}

	return a
}

// SetOutput overrides stdout and stderr writers (for testing).
func (a *Agent) SetOutput(stdout, stderr io.Writer) {
	a.stdout = stdout
	a.stderr = stderr
}

// Send processes a user message through the conversation loop.
// It streams the response, handles tool calls, and loops until
// the model produces a text-only response.
func (a *Agent) Send(ctx context.Context, userMessage string) error {
	a.history = append(a.history, llm.Message{
		Role:    "user",
		Content: userMessage,
	})

	return a.loop(ctx)
}

// loop runs the core agent loop: send to LLM, handle tool calls, repeat.
func (a *Agent) loop(ctx context.Context) error {
	for {
		// Check for context cancellation before each iteration.
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("agent cancelled: %w", err)
		}

		// Build tool definitions from registry.
		toolDefs := a.convertToolDefs()

		req := llm.ChatCompletionRequest{
			Model:    a.model,
			Messages: a.history,
			Tools:    toolDefs,
		}

		// Stream the response.
		msg, err := a.client.ChatCompletionStream(ctx, req, func(chunk llm.ChatCompletionChunk) {
			for _, choice := range chunk.Choices {
				if choice.Delta.Content != "" {
					fmt.Fprint(a.stdout, choice.Delta.Content)
				}
			}
		})
		if err != nil {
			return fmt.Errorf("LLM request failed: %w", err)
		}

		// Append assistant message to history.
		a.history = append(a.history, *msg)

		// If no tool calls, we're done.
		if len(msg.ToolCalls) == 0 {
			fmt.Fprintln(a.stdout)
			return nil
		}

		// Process each tool call.
		for _, tc := range msg.ToolCalls {
			result := a.executeTool(ctx, tc)
			a.history = append(a.history, llm.Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
				Content:    result,
			})
		}

		// Loop back to send tool results to the model.
	}
}

// executeTool handles a single tool call: lookup, permission check, execution.
func (a *Agent) executeTool(ctx context.Context, tc llm.ToolCall) string {
	t := a.registry.Get(tc.Function.Name)
	if t == nil {
		fmt.Fprintf(a.stderr, "[tool] Unknown tool: %s\n", tc.Function.Name)
		return fmt.Sprintf("Unknown tool: %s", tc.Function.Name)
	}

	// Permission check.
	if t.Permission() == tool.PermissionPrompt {
		var preview string
		if p, ok := t.(tool.Previewer); ok {
			preview = p.Preview(json.RawMessage(tc.Function.Arguments))
		} else {
			preview = fmt.Sprintf("%s(%s)", tc.Function.Name, truncateArgs(tc.Function.Arguments, 200))
		}
		if !a.permission.Check(tc.Function.Name, preview) {
			fmt.Fprintf(a.stderr, "[tool] %s: permission denied\n", tc.Function.Name)
			return "Permission denied by user"
		}
	}

	fmt.Fprintf(a.stderr, "[tool] %s\n", tc.Function.Name)

	result, err := t.Execute(ctx, json.RawMessage(tc.Function.Arguments))
	if err != nil {
		return fmt.Sprintf("Tool error: %v", err)
	}

	return result
}

// convertToolDefs converts tool.ToolDef to llm.ToolDef.
func (a *Agent) convertToolDefs() []llm.ToolDef {
	defs := a.registry.Definitions()
	llmDefs := make([]llm.ToolDef, len(defs))
	for i, d := range defs {
		llmDefs[i] = llm.ToolDef{
			Type: d.Type,
			Function: llm.FunctionDef{
				Name:        d.Function.Name,
				Description: d.Function.Description,
				Parameters:  d.Function.Parameters,
			},
		}
	}
	return llmDefs
}

// truncateArgs shortens a JSON arguments string for display.
func truncateArgs(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
