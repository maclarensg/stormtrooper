package tui

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gavinyap/stormtrooper/internal/permission"
)

// Ensure interfaces are satisfied at compile time.
var (
	_ io.Writer        = (*EventWriter)(nil)
	_ io.Writer        = (*ToolEventWriter)(nil)
	_ permission.Handler = (*PermissionInterceptor)(nil)
)

// idCounter is used to generate unique IDs for permission requests.
var idCounter atomic.Uint64

func generateID() string {
	return fmt.Sprintf("perm-%d", idCounter.Add(1))
}

// EventWriter implements io.Writer. Each Write sends a TokenMsg
// on the events channel. Used as the agent's stdout.
type EventWriter struct {
	events chan<- AgentEvent
}

func (w *EventWriter) Write(p []byte) (int, error) {
	w.events <- TokenMsg{Content: string(p)}
	return len(p), nil
}

// ToolEventWriter implements io.Writer. It parses stderr output from the
// agent and converts recognized patterns into structured events.
type ToolEventWriter struct {
	events chan<- AgentEvent
	mu     sync.Mutex
	buf    []byte
}

func (w *ToolEventWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buf = append(w.buf, p...)

	for {
		idx := bytes.IndexByte(w.buf, '\n')
		if idx < 0 {
			break
		}
		line := string(w.buf[:idx])
		w.buf = w.buf[idx+1:]
		w.parseLine(line)
	}

	return len(p), nil
}

func (w *ToolEventWriter) parseLine(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}

	switch {
	case strings.HasPrefix(line, "[tool] "):
		rest := strings.TrimPrefix(line, "[tool] ")
		// Skip "permission denied" lines — handled by the permission flow.
		if strings.Contains(rest, ": permission denied") {
			return
		}
		// Skip "Unknown tool:" lines.
		if strings.HasPrefix(rest, "Unknown tool:") {
			return
		}
		w.events <- ToolStartMsg{Name: rest}

	case strings.HasPrefix(line, "[agent] Spawning sub-agent: "):
		task := strings.TrimPrefix(line, "[agent] Spawning sub-agent: ")
		w.events <- SubAgentSpawnMsg{Task: task}

	case line == "[agent] Sub-agent completed":
		w.events <- SubAgentDoneMsg{}
	}
}

// PermissionInterceptor implements permission.Handler for TUI mode.
// It sends permission requests to the Bubble Tea event loop and blocks
// until the user responds via the TUI.
type PermissionInterceptor struct {
	events chan<- AgentEvent
}

// NewPermissionInterceptor creates a new PermissionInterceptor.
func NewPermissionInterceptor(events chan<- AgentEvent) *PermissionInterceptor {
	return &PermissionInterceptor{events: events}
}

// Check sends a permission request to the TUI and blocks until the user responds.
func (p *PermissionInterceptor) Check(toolName string, preview string) bool {
	respCh := make(chan bool, 1)
	p.events <- PermissionRequestMsg{
		ID:       generateID(),
		ToolName: toolName,
		Preview:  preview,
		Response: respCh,
	}
	return <-respCh
}

// Bridge connects an agent.Agent to the Bubble Tea event loop.
type Bridge struct {
	events chan AgentEvent
	stdout *EventWriter
	stderr *ToolEventWriter
	perm   *PermissionInterceptor
}

// NewBridge creates a new Bridge with a buffered event channel.
func NewBridge() *Bridge {
	events := make(chan AgentEvent, 256)
	return &Bridge{
		events: events,
		stdout: &EventWriter{events: events},
		stderr: &ToolEventWriter{events: events},
		perm:   NewPermissionInterceptor(events),
	}
}

// Events returns the receive-only events channel for the TUI to listen on.
func (b *Bridge) Events() <-chan AgentEvent {
	return b.events
}

// Stdout returns the io.Writer to set as the agent's stdout.
func (b *Bridge) Stdout() io.Writer { return b.stdout }

// Stderr returns the io.Writer to set as the agent's stderr.
func (b *Bridge) Stderr() io.Writer { return b.stderr }

// Permission returns the permission handler for TUI mode.
func (b *Bridge) Permission() permission.Handler { return b.perm }

// WaitForEvent returns a tea.Cmd that blocks until an AgentEvent is received
// from the bridge, then returns it as a tea.Msg.
func WaitForEvent(events <-chan AgentEvent) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-events
		if !ok {
			return nil
		}
		return event
	}
}
