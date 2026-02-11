package tui

import (
	"strings"
	"testing"
)

func newTestSidebarModel() SidebarModel {
	theme := DefaultTheme()
	return NewSidebarModel(&theme, SidebarOptions{
		ProjectDir:   "myproject",
		MemoryLoaded: true,
		ToolCount:    8,
		ModelName:    "kimi-k2",
	})
}

func TestSidebar_ToolStart(t *testing.T) {
	m := newTestSidebarModel()

	m, _ = m.Update(ToolStartMsg{ID: "1", Name: "read_file", Args: "src/main.go"})

	if len(m.toolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(m.toolCalls))
	}
	if !m.toolCalls[0].Running {
		t.Error("expected tool to be running")
	}
	if m.toolCalls[0].Name != "read_file" {
		t.Errorf("expected 'read_file', got %q", m.toolCalls[0].Name)
	}
}

func TestSidebar_ToolComplete(t *testing.T) {
	m := newTestSidebarModel()

	m, _ = m.Update(ToolStartMsg{ID: "1", Name: "read_file", Args: "src/main.go"})
	m, _ = m.Update(ToolResultMsg{ID: "1", Name: "read_file", Result: "ok"})

	if len(m.toolCalls) != 0 {
		t.Fatalf("expected completed tool to be removed, got %d entries", len(m.toolCalls))
	}

	view := m.View()
	if strings.Contains(view, "read_file") {
		t.Error("expected completed tool to not appear in sidebar view")
	}
}

func TestSidebar_ToolError(t *testing.T) {
	m := newTestSidebarModel()

	m, _ = m.Update(ToolStartMsg{ID: "1", Name: "shell_exec", Args: "ls"})
	m, _ = m.Update(ToolResultMsg{ID: "1", Name: "shell_exec", Error: "exit 1"})

	if len(m.toolCalls) != 0 {
		t.Fatalf("expected errored tool to be removed, got %d entries", len(m.toolCalls))
	}
}

func TestSidebar_MaxTools(t *testing.T) {
	m := newTestSidebarModel()
	m.maxTools = 3

	for i := 0; i < 5; i++ {
		m, _ = m.Update(ToolStartMsg{ID: "t", Name: "tool", Args: ""})
	}

	if len(m.toolCalls) != 3 {
		t.Errorf("expected 3 tool calls (maxTools), got %d", len(m.toolCalls))
	}
}

func TestSidebar_AgentStatus(t *testing.T) {
	m := newTestSidebarModel()

	// Initially idle.
	view := m.View()
	if !strings.Contains(view, "Idle") {
		t.Error("expected 'Idle' in initial view")
	}

	// Set busy.
	m.SetAgentBusy(true)
	view = m.View()
	if !strings.Contains(view, "Thinking...") {
		t.Error("expected 'Thinking...' when agent is busy")
	}

	// AgentDoneMsg sets it back to idle.
	m, _ = m.Update(AgentDoneMsg{})
	view = m.View()
	if !strings.Contains(view, "Idle") {
		t.Error("expected 'Idle' after AgentDoneMsg")
	}
}

func TestSidebar_ProjectInfo(t *testing.T) {
	m := newTestSidebarModel()
	view := m.View()

	checks := []string{
		"myproject",
		"loaded",
		"Tools: 8",
		"kimi-k2",
	}
	for _, check := range checks {
		if !strings.Contains(view, check) {
			t.Errorf("expected view to contain %q", check)
		}
	}
}

func TestSidebar_ProjectInfoMemoryNotLoaded(t *testing.T) {
	theme := DefaultTheme()
	m := NewSidebarModel(&theme, SidebarOptions{
		ProjectDir:   "test",
		MemoryLoaded: false,
		ToolCount:    5,
		ModelName:    "gpt-4",
	})

	view := m.View()
	if !strings.Contains(view, "not loaded") {
		t.Error("expected 'not loaded' for memory when MemoryLoaded is false")
	}
}

func TestSidebar_MostRecentAtTop(t *testing.T) {
	m := newTestSidebarModel()

	m, _ = m.Update(ToolStartMsg{ID: "1", Name: "first_tool", Args: ""})
	m, _ = m.Update(ToolStartMsg{ID: "2", Name: "second_tool", Args: ""})

	if m.toolCalls[0].Name != "second_tool" {
		t.Errorf("expected most recent tool at top, got %q", m.toolCalls[0].Name)
	}
	if m.toolCalls[1].Name != "first_tool" {
		t.Errorf("expected older tool second, got %q", m.toolCalls[1].Name)
	}
}
