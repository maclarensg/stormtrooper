package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// StatusBarModel renders a single-row status bar at the top of the TUI.
type StatusBarModel struct {
	theme   *Theme
	width   int
	version string // e.g. "v0.2.0"
	model   string // e.g. "kimi-k2"
	cwd     string // e.g. "~/myproject"
}

// NewStatusBarModel creates a StatusBarModel with the given static values.
func NewStatusBarModel(theme *Theme, version, model, cwd string) StatusBarModel {
	return StatusBarModel{
		theme:   theme,
		version: version,
		model:   model,
		cwd:     cwd,
	}
}

// Init returns nil; no initial commands are needed.
func (m StatusBarModel) Init() tea.Cmd {
	return nil
}

// Update handles window resize events.
func (m StatusBarModel) Update(msg tea.Msg) (StatusBarModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	}
	return m, nil
}

// View renders the status bar as a single full-width row.
func (m StatusBarModel) View() string {
	if m.width <= 0 {
		return ""
	}

	left := "stormtrooper " + m.version
	center := m.model
	right := m.truncateCWD(m.cwd)

	// Calculate available space for padding.
	usedWidth := lipgloss.Width(left) + lipgloss.Width(center) + lipgloss.Width(right)
	totalPadding := m.width - usedWidth
	if totalPadding < 2 {
		// Not enough room — just join with single spaces.
		row := left + " " + center + " " + right
		return m.theme.StatusBar.Width(m.width).Render(row)
	}

	leftPad := totalPadding / 2
	rightPad := totalPadding - leftPad

	row := left + strings.Repeat(" ", leftPad) + center + strings.Repeat(" ", rightPad) + right
	return m.theme.StatusBar.Width(m.width).Render(row)
}

// SetWidth updates the status bar width.
func (m *StatusBarModel) SetWidth(w int) {
	m.width = w
}

// truncateCWD shortens a CWD from the left if it exceeds available space.
// For example, "/home/user/projects/myapp/src" becomes "...ojects/myapp/src".
func (m StatusBarModel) truncateCWD(cwd string) string {
	// Reserve space for the other segments (version + model + padding).
	maxCWD := m.width / 3
	if maxCWD < 10 {
		maxCWD = 10
	}

	if len(cwd) <= maxCWD {
		return cwd
	}

	return "..." + cwd[len(cwd)-(maxCWD-3):]
}
