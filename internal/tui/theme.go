package tui

import "github.com/charmbracelet/lipgloss"

// Theme holds all lipgloss styles used across the TUI.
type Theme struct {
	// Panel borders
	ChatBorder    lipgloss.Style
	SidebarBorder lipgloss.Style
	InputBorder   lipgloss.Style
	StatusBar     lipgloss.Style

	// Message styles
	UserPrefix      lipgloss.Style // "You:" label
	AssistantPrefix lipgloss.Style // "Assistant:" label
	UserMessage     lipgloss.Style
	ToolInline      lipgloss.Style // Inline tool activity in chat

	// Sidebar section styles
	SidebarHeading lipgloss.Style
	SidebarItem    lipgloss.Style
	ToolRunning    lipgloss.Style // spinner + name while tool runs
	ToolDone       lipgloss.Style // checkmark + name when tool completes

	// Permission prompt
	PermissionBorder lipgloss.Style
	PermissionText   lipgloss.Style

	// Input
	InputPlaceholder lipgloss.Style
}

// DefaultTheme returns a Theme with sensible defaults for light and dark terminals.
func DefaultTheme() Theme {
	cyan := lipgloss.Color("6")
	purple := lipgloss.Color("63")
	gray := lipgloss.Color("245")
	amber := lipgloss.Color("214")
	green := lipgloss.Color("2")
	statusBg := lipgloss.Color("236")
	statusFg := lipgloss.Color("252")

	border := lipgloss.RoundedBorder()

	return Theme{
		ChatBorder: lipgloss.NewStyle().
			Border(border).
			BorderForeground(gray),
		SidebarBorder: lipgloss.NewStyle().
			Border(border).
			BorderForeground(gray),
		InputBorder: lipgloss.NewStyle().
			Border(border).
			BorderForeground(cyan),
		StatusBar: lipgloss.NewStyle().
			Background(statusBg).
			Foreground(statusFg).
			Padding(0, 1),

		UserPrefix: lipgloss.NewStyle().
			Foreground(cyan).
			Bold(true),
		AssistantPrefix: lipgloss.NewStyle().
			Bold(true),
		UserMessage: lipgloss.NewStyle().
			Foreground(cyan),
		ToolInline: lipgloss.NewStyle().
			Foreground(gray).
			Italic(true),

		SidebarHeading: lipgloss.NewStyle().
			Foreground(purple).
			Bold(true),
		SidebarItem: lipgloss.NewStyle().
			Foreground(gray),
		ToolRunning: lipgloss.NewStyle().
			Foreground(amber),
		ToolDone: lipgloss.NewStyle().
			Foreground(green),

		PermissionBorder: lipgloss.NewStyle().
			Border(border).
			BorderForeground(amber),
		PermissionText: lipgloss.NewStyle().
			Foreground(amber),

		InputPlaceholder: lipgloss.NewStyle().
			Foreground(gray).
			Italic(true),
	}
}
