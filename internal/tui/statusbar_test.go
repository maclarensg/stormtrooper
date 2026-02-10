package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func newTestStatusBarModel() StatusBarModel {
	theme := DefaultTheme()
	m := NewStatusBarModel(&theme, "v0.2.0", "kimi-k2", "~/myproject")
	m.SetWidth(80)
	return m
}

func TestStatusBar_View(t *testing.T) {
	m := newTestStatusBarModel()
	view := m.View()

	if !strings.Contains(view, "v0.2.0") {
		t.Error("expected view to contain version")
	}
	if !strings.Contains(view, "kimi-k2") {
		t.Error("expected view to contain model name")
	}
	if !strings.Contains(view, "myproject") {
		t.Error("expected view to contain CWD")
	}
	if !strings.Contains(view, "stormtrooper") {
		t.Error("expected view to contain 'stormtrooper'")
	}
}

func TestStatusBar_Resize(t *testing.T) {
	m := newTestStatusBarModel()

	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	if m.width != 120 {
		t.Errorf("expected width 120, got %d", m.width)
	}

	// View should still render correctly at new width.
	view := m.View()
	if !strings.Contains(view, "v0.2.0") {
		t.Error("expected view to contain version after resize")
	}
}

func TestStatusBar_LongCWD(t *testing.T) {
	theme := DefaultTheme()
	longCWD := "/home/user/very/deeply/nested/project/directory/structure/src/main"
	m := NewStatusBarModel(&theme, "v0.2.0", "kimi-k2", longCWD)
	m.SetWidth(60) // narrow width forces truncation

	view := m.View()

	// The CWD should be truncated with "..." prefix.
	if !strings.Contains(view, "...") {
		t.Error("expected truncated CWD to contain '...'")
	}

	// The full original path should not appear.
	if strings.Contains(view, "/home/user/very/deeply") {
		t.Error("expected long CWD to be truncated, but found full prefix")
	}
}

func TestStatusBar_ShortCWD(t *testing.T) {
	theme := DefaultTheme()
	m := NewStatusBarModel(&theme, "v0.2.0", "kimi-k2", "~/app")
	m.SetWidth(80)

	view := m.View()

	// Short CWD should appear without truncation.
	if !strings.Contains(view, "~/app") {
		t.Error("expected short CWD to appear in full")
	}
	if strings.Contains(view, "...") {
		t.Error("short CWD should not be truncated")
	}
}

func TestStatusBar_ZeroWidth(t *testing.T) {
	m := newTestStatusBarModel()
	m.SetWidth(0)

	view := m.View()
	if view != "" {
		t.Errorf("expected empty view at zero width, got %q", view)
	}
}
