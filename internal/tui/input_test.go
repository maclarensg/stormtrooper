package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func newTestInputModel() InputModel {
	theme := DefaultTheme()
	keymap := DefaultKeyMap()
	return NewInputModel(&theme, &keymap)
}

func TestInputModel_Send(t *testing.T) {
	m := newTestInputModel()

	// Type some text into the textarea.
	m.textarea.SetValue("Hello agent")

	// Press Enter.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated

	if cmd == nil {
		t.Fatal("expected a command from Enter press")
	}

	// Execute the command to get the message.
	msg := cmd()
	sendMsg, ok := msg.(SendMsg)
	if !ok {
		t.Fatalf("expected SendMsg, got %T", msg)
	}
	if sendMsg.Text != "Hello agent" {
		t.Errorf("expected 'Hello agent', got %q", sendMsg.Text)
	}

	// Textarea should be cleared.
	if m.textarea.Value() != "" {
		t.Errorf("expected textarea to be cleared, got %q", m.textarea.Value())
	}
}

func TestInputModel_EmptyNoSend(t *testing.T) {
	m := newTestInputModel()

	// Press Enter with empty textarea.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd != nil {
		t.Error("expected nil command when sending empty input")
	}
}

func TestInputModel_Disabled(t *testing.T) {
	m := newTestInputModel()
	m.SetDisabled(true)

	// Try to type while disabled.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	if cmd != nil {
		t.Error("expected nil command when input is disabled")
	}

	// Verify the view shows spinner text.
	view := m.View()
	if !strings.Contains(view, "Thinking...") {
		t.Errorf("expected 'Thinking...' in disabled view, got %q", view)
	}
}

func TestInputModel_NewLine(t *testing.T) {
	m := newTestInputModel()

	// Type some text first.
	m.textarea.SetValue("line1")
	// Move cursor to end.
	m.textarea.CursorEnd()

	// Press Ctrl+J for newline.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlJ})

	val := m.textarea.Value()
	if !strings.Contains(val, "\n") {
		t.Errorf("expected newline in textarea value, got %q", val)
	}
}

func TestInputModel_Resize(t *testing.T) {
	m := newTestInputModel()

	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	if m.width != 100 {
		t.Errorf("expected width 100, got %d", m.width)
	}
}

func TestInputModel_FocusBlur(t *testing.T) {
	m := newTestInputModel()

	m.Blur()
	if m.textarea.Focused() {
		t.Error("expected textarea to be blurred")
	}

	m.Focus()
	if !m.textarea.Focused() {
		t.Error("expected textarea to be focused")
	}
}

func TestInputModel_SetDisabledToggle(t *testing.T) {
	m := newTestInputModel()

	m.SetDisabled(true)
	if !m.disabled {
		t.Error("expected disabled to be true")
	}

	m.SetDisabled(false)
	if m.disabled {
		t.Error("expected disabled to be false")
	}
}
