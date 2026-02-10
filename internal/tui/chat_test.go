package tui

import (
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func newTestChatModel() ChatModel {
	theme := DefaultTheme()
	m := NewChatModel(&theme)
	m.SetSize(80, 24)
	return m
}

func TestChatModel_AddUserMessage(t *testing.T) {
	m := newTestChatModel()
	m.AddUserMessage("Hello, world!")

	view := m.View()
	if !strings.Contains(view, "You:") {
		t.Error("expected view to contain user prefix 'You:'")
	}
	if !strings.Contains(view, "Hello, world!") {
		t.Error("expected view to contain user message content")
	}
	if len(m.messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(m.messages))
	}
	if m.messages[0].Role != RoleUser {
		t.Errorf("expected RoleUser, got %d", m.messages[0].Role)
	}
}

func TestChatModel_StreamTokens(t *testing.T) {
	m := newTestChatModel()

	tokens := []string{"Hello", " ", "from", " ", "the", " ", "assistant"}
	for _, tok := range tokens {
		var cmd tea.Cmd
		m, cmd = m.Update(TokenMsg{Content: tok})
		_ = cmd
	}

	view := stripANSI(m.View())
	if !strings.Contains(view, "Hello from the assistant") {
		t.Errorf("expected streamed content in view, got:\n%s", view)
	}
	// The streaming buffer should contain all tokens.
	if m.streaming.String() != "Hello from the assistant" {
		t.Errorf("expected streaming buffer to accumulate, got %q", m.streaming.String())
	}
	// No finalized messages yet.
	if len(m.messages) != 0 {
		t.Errorf("expected 0 finalized messages while streaming, got %d", len(m.messages))
	}
}

func TestChatModel_AgentDone(t *testing.T) {
	m := newTestChatModel()

	// Stream some tokens first.
	m, _ = m.Update(TokenMsg{Content: "Hello"})
	m, _ = m.Update(TokenMsg{Content: " world"})

	// Finalize.
	m, _ = m.Update(AgentDoneMsg{})

	if len(m.messages) != 1 {
		t.Fatalf("expected 1 finalized message, got %d", len(m.messages))
	}
	if m.messages[0].Role != RoleAssistant {
		t.Errorf("expected RoleAssistant, got %d", m.messages[0].Role)
	}
	if m.messages[0].Content != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", m.messages[0].Content)
	}
	// Streaming buffer should be cleared.
	if m.streaming.Len() != 0 {
		t.Errorf("expected streaming buffer to be cleared, got %q", m.streaming.String())
	}
}

func TestChatModel_AgentDone_EmptyStream(t *testing.T) {
	m := newTestChatModel()

	// AgentDoneMsg with no tokens should not create a message.
	m, _ = m.Update(AgentDoneMsg{})

	if len(m.messages) != 0 {
		t.Errorf("expected 0 messages for empty stream, got %d", len(m.messages))
	}
}

func TestChatModel_ToolStartAndResult(t *testing.T) {
	m := newTestChatModel()

	m, _ = m.Update(ToolStartMsg{ID: "1", Name: "read_file", Args: "src/main.go"})

	if len(m.messages) != 1 {
		t.Fatalf("expected 1 message after ToolStartMsg, got %d", len(m.messages))
	}
	if m.messages[0].Role != RoleTool {
		t.Errorf("expected RoleTool, got %d", m.messages[0].Role)
	}
	if !strings.Contains(m.messages[0].Content, "read_file") {
		t.Error("expected tool message to contain tool name")
	}

	// Complete the tool successfully.
	m, _ = m.Update(ToolResultMsg{ID: "1", Name: "read_file", Result: "ok"})

	if !strings.Contains(m.messages[0].Content, "\u2713") {
		t.Errorf("expected checkmark in completed tool message, got %q", m.messages[0].Content)
	}
}

func TestChatModel_ToolError(t *testing.T) {
	m := newTestChatModel()

	m, _ = m.Update(ToolStartMsg{ID: "1", Name: "shell_exec", Args: "ls"})
	m, _ = m.Update(ToolResultMsg{ID: "1", Name: "shell_exec", Error: "exit 1"})

	if !strings.Contains(m.messages[0].Content, "\u2717") {
		t.Errorf("expected cross mark for errored tool, got %q", m.messages[0].Content)
	}
}

func TestChatModel_PermissionRequest(t *testing.T) {
	m := newTestChatModel()

	resp := make(chan bool, 1)
	m, _ = m.Update(PermissionRequestMsg{
		ID:       "p1",
		ToolName: "shell_exec",
		Preview:  "Command: rm -rf node_modules",
		Response: resp,
	})

	if len(m.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(m.messages))
	}
	if m.messages[0].Role != RoleSystem {
		t.Errorf("expected RoleSystem, got %d", m.messages[0].Role)
	}
	view := m.View()
	if !strings.Contains(view, "shell_exec") {
		t.Error("expected permission prompt to contain tool name")
	}
	if !strings.Contains(view, "[y] allow") {
		t.Error("expected permission prompt to show allow/deny hint")
	}
}

func TestChatModel_PermissionResponse(t *testing.T) {
	m := newTestChatModel()

	resp := make(chan bool, 1)
	m, _ = m.Update(PermissionRequestMsg{
		ID:       "p1",
		ToolName: "shell_exec",
		Preview:  "Command: rm something",
		Response: resp,
	})

	m, _ = m.Update(PermissionResponseMsg{Allowed: true})

	if !strings.Contains(m.messages[0].Content, "Allowed") {
		t.Errorf("expected 'Allowed' in permission message, got %q", m.messages[0].Content)
	}
}

func TestChatModel_Resize(t *testing.T) {
	m := newTestChatModel()

	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	if m.width != 120 {
		t.Errorf("expected width 120, got %d", m.width)
	}
	if m.height != 40 {
		t.Errorf("expected height 40, got %d", m.height)
	}
	if m.viewport.Width != 120 {
		t.Errorf("expected viewport width 120, got %d", m.viewport.Width)
	}
	if m.viewport.Height != 40 {
		t.Errorf("expected viewport height 40, got %d", m.viewport.Height)
	}
}

func TestChatModel_AddSystemMessage(t *testing.T) {
	m := newTestChatModel()
	m.AddSystemMessage("Error: something went wrong")

	if len(m.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(m.messages))
	}
	if m.messages[0].Role != RoleSystem {
		t.Errorf("expected RoleSystem, got %d", m.messages[0].Role)
	}
	if m.messages[0].Content != "Error: something went wrong" {
		t.Errorf("expected error content, got %q", m.messages[0].Content)
	}
}
