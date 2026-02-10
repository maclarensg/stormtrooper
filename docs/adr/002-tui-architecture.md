# ADR 002: Dashboard TUI with Bubble Tea

## Status

Proposed

## Context

Stormtrooper v0.1.0 uses a plain REPL (`internal/repl/`) that writes streamed tokens and tool activity directly to stdout/stderr. This works but provides a poor user experience:

- Streaming text, tool output, and permission prompts all interleave in a single stream
- No visual separation between agent output, tool activity, and status information
- No way to see project context, memory, or agent status at a glance
- Permission prompts appear inline and can be missed during fast streaming

We want a dashboard-style TUI — similar to Lazygit or k9s — that provides:
- A scrollable chat area with markdown-rendered messages
- A sidebar showing tool activity, agent status, and project info
- A dedicated input area with multi-line support
- A status bar showing model, project, and working directory
- Clear permission prompts that can't be missed

## Decision

### Framework: Bubble Tea + Lipgloss + Bubbles

Use the Charm ecosystem:
- `github.com/charmbracelet/bubbletea` — Elm-architecture TUI framework
- `github.com/charmbracelet/lipgloss` — terminal styling (already a dependency)
- `github.com/charmbracelet/bubbles` — standard components (viewport, textarea, spinner)
- `github.com/charmbracelet/glamour` — markdown rendering (already a dependency)

### Layout

```
┌─────────────────────────────────────────────────────────────────────┐
│ [status bar] model: kimi-k2 | cwd: ~/myproject | stormtrooper v0.2│
├───────────────────────────────────────────────┬─────────────────────┤
│                                               │  Tool Activity      │
│              Chat Panel                       │  ─────────────────  │
│                                               │  > read_file        │
│  You:                                         │  > edit_file        │
│  Fix the login bug                            │  > shell_exec       │
│                                               │                     │
│  Assistant:                                   ├─────────────────────┤
│  I'll look at the authentication code...      │  Agent Status       │
│  [streaming tokens appear here]               │  ─────────────────  │
│                                               │  ● Thinking...      │
│                                               │  Tokens: 1,234      │
│                                               │                     │
│                                               ├─────────────────────┤
│                                               │  Project Info       │
│                                               │  ─────────────────  │
│                                               │  Dir: ~/myproject   │
│                                               │  Memory: loaded     │
│                                               │  Tools: 8 active    │
├───────────────────────────────────────────────┴─────────────────────┤
│ > type your message here...                                    [⏎] │
└─────────────────────────────────────────────────────────────────────┘
```

- **Status bar** (1 row, top): Model name, CWD, version. Always visible.
- **Chat panel** (left, fills remaining height): Scrollable viewport showing conversation history with markdown-rendered messages. User messages and assistant messages are visually distinct.
- **Sidebar** (right, fixed width ~30 cols): Three sections — Tool Activity (recent tool calls with status), Agent Status (thinking/idle + token count), Project Info (dir, memory, tool count).
- **Input bar** (bottom, 3 rows minimum): Multi-line text input with textarea component. Enter sends, Shift+Enter or backslash for newlines. Shows hint text when empty.

### Bubble Tea Architecture

#### Main App Model

```go
// internal/tui/app.go
type App struct {
    // Sub-models (each implements tea.Model)
    chat      ChatModel      // Chat viewport with message history
    input     InputModel     // Text input area
    sidebar   SidebarModel   // Right sidebar with panels
    statusbar StatusBarModel // Top status bar

    // State
    width     int            // Terminal width
    height    int            // Terminal height
    focus     FocusArea      // Which panel has focus (input or chat for scrolling)

    // Agent bridge
    events    chan AgentEvent // Channel receiving events from the agent
    agent     *agent.Agent
    agentBusy bool           // Whether agent is currently processing

    // Permission handling
    permReq   *PermissionRequest // Non-nil when waiting for permission
}
```

#### Focus Areas

Only two focusable areas:
- `FocusInput` (default) — keypresses go to the textarea
- `FocusChat` — keypresses scroll the chat viewport (Esc to enter, Esc or `i` to return to input)

Tab switches focus. This is deliberately simple.

#### Message Types

Custom `tea.Msg` types for communication between the agent goroutine and the TUI:

```go
// internal/tui/events.go

// AgentEvent is the union type sent from the agent bridge to the TUI.
type AgentEvent interface{ agentEvent() }

// Streamed text token from the assistant
type TokenMsg struct {
    Content string
}

// Tool call started
type ToolStartMsg struct {
    ID   string
    Name string
    Args string // truncated for display
}

// Tool call completed
type ToolResultMsg struct {
    ID     string
    Name   string
    Result string // truncated for display
    Error  string // non-empty if tool errored
}

// Permission request from the agent
type PermissionRequestMsg struct {
    ID       string
    ToolName string
    Preview  string
    Response chan<- bool // TUI sends y/n back through this channel
}

// Agent finished processing (returned from Send())
type AgentDoneMsg struct {
    Error error
}

// Sub-agent spawned
type SubAgentSpawnMsg struct {
    Task string
}

// Sub-agent completed
type SubAgentDoneMsg struct{}
```

#### Agent Bridge

The critical piece: how the existing agent (which writes to `io.Writer` and reads stdin for permissions) connects to Bubble Tea's message loop.

**Approach: Event-emitting `io.Writer` + Permission interceptor**

Rather than modifying the agent package, we create adapter types that the agent writes to, which translate writes into TUI events:

```go
// internal/tui/bridge.go

// EventWriter implements io.Writer. Every Write() sends the content
// as a TokenMsg on the events channel.
type EventWriter struct {
    events chan<- AgentEvent
}

func (w *EventWriter) Write(p []byte) (int, error) {
    w.events <- TokenMsg{Content: string(p)}
    return len(p), nil
}

// ToolEventWriter implements io.Writer for stderr.
// Parses "[tool] ..." and "[permission] ..." lines into structured events.
type ToolEventWriter struct {
    events chan<- AgentEvent
}

// PermissionInterceptor implements permission.Checker behavior for TUI mode.
// Instead of reading stdin, it sends a PermissionRequestMsg and blocks
// until the TUI responds.
type PermissionInterceptor struct {
    events chan<- AgentEvent
}
```

The bridge runs `agent.Send()` in a goroutine:

```go
func (app *App) runAgent(userMessage string) tea.Cmd {
    return func() tea.Msg {
        err := app.agent.Send(context.Background(), userMessage)
        return AgentDoneMsg{Error: err}
    }
}
```

**Key insight**: The agent's `SetOutput(stdout, stderr)` method already exists. We set stdout to an `EventWriter` (captures streamed tokens) and stderr to a `ToolEventWriter` (captures tool activity). For permissions, we need to create a `permission.Checker` that blocks on a channel instead of reading stdin — the existing `NewCheckerWithIO` can be used with a pipe, or we introduce a small interface.

**Permission flow in detail:**

1. Agent calls `permission.Check()` during tool execution
2. The TUI-mode permission checker sends a `PermissionRequestMsg` on the events channel (which includes a response channel)
3. The Bubble Tea Update loop receives this message, stores it in `app.permReq`, and renders a permission prompt in the chat area
4. User presses `y` or `n`
5. TUI sends the boolean on the response channel
6. The permission checker unblocks and returns the result to the agent

This is the cleanest approach because it requires no changes to the agent package — only a different `permission.Checker` implementation and different `io.Writer` instances.

#### Component Details

**Chat Panel** (`internal/tui/chat.go`):
- Uses `bubbles/viewport` for scrolling
- Maintains a list of `ChatMessage` structs: `{Role, Content, Timestamp}`
- User messages rendered with a "You:" prefix in one color
- Assistant messages rendered with glamour (markdown) as they stream in
- Tool activity is optionally shown inline (e.g., "Used read_file: src/main.go")
- Auto-scrolls to bottom on new content; manual scroll-up disables auto-scroll

**Input Bar** (`internal/tui/input.go`):
- Uses `bubbles/textarea` component
- Placeholder text: "Type a message... (Enter to send, Esc to scroll chat)"
- Enter sends the message (if not empty)
- Ctrl+J or Shift+Enter inserts a newline (multi-line input)
- When agent is busy, shows a spinner and disables input

**Sidebar** (`internal/tui/sidebar.go`):
- Three stacked sections, each a simple lipgloss-styled box
- **Tool Activity**: Last N tool calls with name and status (spinner while running, checkmark when done). Scrolls if too many.
- **Agent Status**: "Idle", "Thinking..." (with spinner), or "Waiting for permission". Shows token count if available.
- **Project Info**: Working directory (basename), memory status (loaded/not loaded), number of registered tools, current model.

**Status Bar** (`internal/tui/statusbar.go`):
- Single row at the top
- Left-aligned: "stormtrooper v0.2.0"
- Center: model name
- Right-aligned: CWD (truncated to fit)
- Styled with lipgloss background color

**Permission Prompt**:
- Rendered inline in the chat panel as a highlighted block
- Shows tool name, preview text, and "[y] allow / [n] deny" hint
- When active, keypress `y` or `n` is captured at the App level (overrides input focus)
- Styled with a warning color (yellow/amber border)

### Styling

Use a consistent lipgloss theme:

- **Background**: Terminal default (transparent)
- **Borders**: `lipgloss.RoundedBorder()` for panels
- **User messages**: Cyan/blue accent
- **Assistant messages**: Default text color, glamour-rendered
- **Tool activity**: Dim/gray text
- **Permission prompts**: Yellow/amber border and text
- **Status bar**: Subtle background (dark gray), white text
- **Sidebar headings**: Bold, subtle color

### Package Structure

```
internal/tui/
├── app.go          # Main App model (Init, Update, View), focus management
├── events.go       # AgentEvent types (TokenMsg, ToolStartMsg, etc.)
├── bridge.go       # EventWriter, ToolEventWriter, PermissionInterceptor
├── chat.go         # ChatModel — viewport, message list, markdown rendering
├── input.go        # InputModel — textarea wrapper, send behavior
├── sidebar.go      # SidebarModel — tool activity, agent status, project info
├── statusbar.go    # StatusBarModel — model, CWD, version
├── theme.go        # Lipgloss styles and color constants
└── keymap.go       # Key bindings (centralized)
```

### Integration with main.go

```go
// In main.go, add a --no-tui flag
noTUI := flag.Bool("no-tui", false, "Use plain REPL instead of TUI")

if *noTUI {
    // Existing REPL path (unchanged)
    r := repl.New(rootAgent)
    r.Run(ctx)
} else {
    // New TUI path
    app := tui.New(tui.Options{
        Agent:      rootAgent,
        Config:     cfg,
        ProjectCtx: projCtx,
        Permission: perm,  // Pass so TUI can swap to interceptor
    })
    p := tea.NewProgram(app, tea.WithAltScreen())
    p.Run()
}
```

**Important**: In TUI mode, before starting the program, we swap the agent's stdout/stderr writers and the permission checker to the TUI-bridged versions. The `tui.New()` constructor handles this wiring.

### What We Do NOT Change

- `internal/agent/` — No changes. We use `SetOutput()` and inject a TUI-aware permission checker.
- `internal/llm/` — No changes.
- `internal/tool/` — No changes.
- `internal/repl/` — No changes. Stays as `--no-tui` fallback.
- `internal/config/` — No changes (the `--no-tui` flag is handled in main.go, not config).
- `internal/memory/` — No changes.
- `internal/context/` — No changes.

### Permission Checker Refactoring (Minimal)

The one small change needed: the `permission.Checker` currently reads directly from `io.Reader`. For the TUI, we need a `permission.Checker` that uses channels instead. Two options:

**Option A (preferred)**: Create a `permission.Handler` interface:
```go
type Handler interface {
    Check(toolName string, preview string) bool
}
```
The existing `Checker` struct implements this. The TUI creates its own implementation that uses channels. The agent accepts a `Handler` interface instead of `*Checker`.

**Option B**: Use the existing `NewCheckerWithIO` with an `io.Pipe()` — the TUI writes "y\n" or "n\n" to the pipe when the user responds. This avoids any interface changes but is hacky.

We go with **Option A** because it's cleaner and the change to agent.go is minimal (change the field type from `*permission.Checker` to `permission.Handler`).

## Consequences

### Positive

- Clean separation: TUI is an entirely new package, existing packages untouched (except the small permission interface extraction).
- The agent bridge pattern (event-emitting writers) is simple and testable.
- Two developers can work in parallel: one on the core TUI framework (app, bridge, events, theme) and one on individual components (chat, sidebar, input, statusbar).
- The `--no-tui` flag preserves the existing REPL for environments without full terminal support (CI, pipes, etc.).
- Bubble Tea's Elm architecture makes state management predictable and testable.

### Negative

- Adds bubbletea and bubbles as dependencies (but we already depend on lipgloss and glamour from the Charm ecosystem).
- The permission channel-based flow adds concurrency complexity, but it's contained in a single bridge component.
- The TUI cannot easily be tested end-to-end in CI (no terminal), but individual models can be unit tested via their Update functions.

### Risks

- **Terminal compatibility**: Some terminals may not support alt-screen or certain Unicode characters. Mitigation: test on common terminals (iTerm2, Windows Terminal, Linux VTE). The `--no-tui` flag provides a fallback.
- **Glamour rendering width**: Glamour needs to know the terminal width to wrap markdown correctly. We pass `app.width - sidebarWidth` to the renderer on every resize.
- **Permission deadlock**: If the agent requests permission but the TUI isn't processing events, we'd deadlock. Mitigation: the permission response channel is buffered (size 1), and we use a context-aware select in the interceptor.
