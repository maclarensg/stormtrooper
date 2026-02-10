# ADR 001: Stormtrooper v0.1.0 Architecture

## Status

Accepted

## Context

Stormtrooper is an AI coding assistant CLI in Go. It needs to:
- Provide a conversational REPL with streaming responses
- Communicate with LLMs via OpenRouter (OpenAI-compatible API)
- Execute tools (file operations, shell, search) via function calling
- Support persistent memory across sessions
- Spawn sub-agents for parallel work
- Be configurable via files and CLI flags

We need a clean, simple architecture that a small team can build incrementally.

## Decision

### Project Layout

```
stormtrooper/
├── cmd/
│   └── stormtrooper/
│       └── main.go              # Entry point, CLI flag parsing, wiring
├── internal/
│   ├── config/
│   │   └── config.go            # Config loading (global + project + CLI flags)
│   ├── llm/
│   │   ├── client.go            # OpenRouter HTTP client (streaming, function calling)
│   │   ├── types.go             # Request/response types (OpenAI-compatible)
│   │   └── stream.go            # SSE stream parser
│   ├── tool/
│   │   ├── registry.go          # Tool registration and dispatch
│   │   ├── types.go             # Tool interface and common types
│   │   ├── read_file.go         # read_file tool
│   │   ├── write_file.go        # write_file tool
│   │   ├── edit_file.go         # edit_file tool
│   │   ├── shell_exec.go        # shell_exec tool
│   │   ├── glob.go              # glob tool
│   │   ├── grep.go              # grep tool
│   │   └── memory_write.go      # memory_write tool
│   ├── permission/
│   │   └── permission.go        # Permission checking and user prompts
│   ├── memory/
│   │   └── memory.go            # Memory loading from .stormtrooper/memory/
│   ├── context/
│   │   └── project.go           # Project context loading (STORMTROOPER.md, cwd)
│   ├── agent/
│   │   ├── agent.go             # Agent struct — owns a conversation + tool loop
│   │   └── spawn.go             # spawn_agent tool + goroutine management
│   └── repl/
│       ├── repl.go              # REPL loop (read input, send to agent, display output)
│       └── input.go             # Multi-line input handling
├── go.mod
└── go.sum
```

### Core Components and Data Flow

```
User Input
    │
    ▼
┌────────┐     ┌───────────┐     ┌────────────┐
│  REPL  │────▶│   Agent   │────▶│ LLM Client │──▶ OpenRouter API
│        │◀────│           │◀────│ (streaming) │◀──
└────────┘     │           │     └────────────┘
    │          │           │
    │          │  tool     │
    │          │  calls    │
    │          ▼           │
    │     ┌──────────┐    │
    │     │ Tool     │    │
    │     │ Registry │    │
    │     └──────────┘    │
    │          │           │
    │          ▼           │
    │     ┌──────────┐    │
    │     │Permission│    │
    │     │ System   │    │
    │     └──────────┘    │
    │
    ▼
Terminal Output (streamed tokens, tool activity, sub-agent status)
```

### Key Design Decisions

#### 1. Agent as the core abstraction

An `Agent` owns:
- A conversation history (`[]Message`)
- A reference to the LLM client
- A reference to the tool registry
- A system prompt (built from project context + memory)

The main REPL creates one root agent. `spawn_agent` creates child agents that run in goroutines with their own conversation history but share the same tool registry and LLM client.

#### 2. OpenAI-compatible types

Since OpenRouter uses the OpenAI chat completions format, we define our own Go structs that match the OpenAI API shape:
- `ChatCompletionRequest` with `messages`, `tools`, `model`, `stream`
- `ChatCompletionResponse` / `ChatCompletionChunk` for non-streaming / streaming
- `ToolCall` with `id`, `function.name`, `function.arguments`

No OpenAI Go SDK dependency. We use `net/http` + `encoding/json` directly. The SSE stream is parsed line by line from the response body.

#### 3. Tool interface

```go
type Tool interface {
    Name() string
    Description() string
    Schema() json.RawMessage  // JSON Schema for parameters
    Execute(ctx context.Context, params json.RawMessage) (string, error)
}
```

The `Registry` holds all tools, provides them as `[]ToolDefinition` for the API request, and dispatches calls by name. Each tool is a self-contained struct implementing this interface.

#### 4. Permission system

Each tool declares its permission level:
- `PermissionAuto` — runs without asking (read_file, glob, grep)
- `PermissionPrompt` — asks user before running (shell_exec, write_file, edit_file, memory_write)

The permission system is a simple function that checks the tool's level and, if needed, prints a preview and prompts `[y/n]`. It returns allow/deny. The agent loop checks permission before executing any tool.

For write_file and edit_file, the preview shows a diff. For shell_exec, it shows the command.

#### 5. Conversation loop (Agent.Run)

```
loop:
  1. Send conversation history to LLM (streaming)
  2. Stream response tokens to terminal
  3. If response contains tool_calls:
     a. For each tool call:
        - Check permission
        - Execute tool (or return "permission denied")
        - Append tool result to conversation
     b. Go to step 1 (let model process tool results)
  4. If response is just text (no tool calls):
     - Done, return to REPL for next user input
```

#### 6. Config layering

Config is loaded in order (later overrides earlier):
1. Defaults (hardcoded)
2. Global config: `~/.stormtrooper/config.yaml`
3. Project config: `.stormtrooper/config.yaml`
4. Environment variables (`OPENROUTER_API_KEY`)
5. CLI flags (`--model`)

We use `gopkg.in/yaml.v3` for YAML parsing. Config struct is flat and simple for v0.1.0.

#### 7. Memory system

- On startup, load `MEMORY.md` from `.stormtrooper/memory/` and inject into system prompt.
- The `memory_write` tool writes/appends to files in `.stormtrooper/memory/`.
- No special indexing or retrieval — just file contents in the prompt.

#### 8. Project context

- On startup, look for `STORMTROOPER.md` or `CLAUDE.md` in the working directory.
- If found, include its contents in the system prompt.
- Also include CWD, platform info, date in the system prompt.

#### 9. Sub-agents

- `spawn_agent` is a tool callable by the AI.
- It creates a new `Agent` with a focused system prompt and task description.
- Runs in a goroutine. The parent blocks on a channel waiting for the result.
- Sub-agents share the LLM client and tool registry (tools are stateless).
- Sub-agent output is prefixed with an identifier in the terminal.
- Sub-agents can use a different model via a parameter.

#### 10. Minimal dependencies

External dependencies:
- `gopkg.in/yaml.v3` — YAML config parsing
- `github.com/charmbracelet/glamour` — Markdown rendering in terminal (for response formatting)

Everything else uses Go stdlib: `net/http`, `encoding/json`, `bufio` (SSE), `os`, `os/exec`, `path/filepath`, `regexp`, `fmt`, `sync`.

### System Prompt Structure

```
[Project instructions from STORMTROOPER.md / CLAUDE.md]
[Memory from .stormtrooper/memory/MEMORY.md]
[Environment info: CWD, platform, date]
[Core instructions: you are a coding assistant, use tools to help the user, etc.]
```

## Consequences

### Positive
- Simple, idiomatic Go — easy to understand and extend.
- Minimal dependencies — fast builds, small binary, few supply chain risks.
- Agent abstraction enables sub-agents naturally.
- Tool interface is easy to add new tools to later.
- Config layering is standard and unsurprising.

### Negative
- No OpenAI SDK means we maintain our own types — but they're small and stable.
- Glamour adds a dependency — but terminal markdown rendering is hard to do well from scratch.
- Sub-agents sharing a tool registry means tools must be goroutine-safe (they are, since they're stateless file/exec operations).

### Risks
- OpenRouter's function calling support varies by model. Some open-source models may not support tool use well. Mitigation: test with the three target models early.
- SSE parsing edge cases. Mitigation: follow the spec carefully, test with real OpenRouter responses.
