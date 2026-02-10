package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ToolCallEntry represents a tool call displayed in the sidebar.
type ToolCallEntry struct {
	Name    string
	Running bool
	Error   bool
}

// SidebarOptions holds static project info for the sidebar.
type SidebarOptions struct {
	ProjectDir   string
	MemoryLoaded bool
	ToolCount    int
	ModelName    string
}

// SidebarModel is the Bubble Tea model for the right sidebar.
type SidebarModel struct {
	theme *Theme
	width int
	height int

	// Tool Activity
	toolCalls []ToolCallEntry
	maxTools  int

	// Agent Status
	agentBusy bool
	spinner   spinner.Model

	// Project Info
	projectDir   string
	memoryLoaded bool
	toolCount    int
	modelName    string
}

// NewSidebarModel creates a SidebarModel with the given options.
func NewSidebarModel(theme *Theme, opts SidebarOptions) SidebarModel {
	s := spinner.New()
	s.Spinner = spinner.Dot

	return SidebarModel{
		theme:        theme,
		width:        30,
		maxTools:     10,
		spinner:      s,
		projectDir:   opts.ProjectDir,
		memoryLoaded: opts.MemoryLoaded,
		toolCount:    opts.ToolCount,
		modelName:    opts.ModelName,
	}
}

// Init returns the spinner tick command.
func (m SidebarModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles tool events, agent status changes, and spinner ticks.
func (m SidebarModel) Update(msg tea.Msg) (SidebarModel, tea.Cmd) {
	switch msg := msg.(type) {
	case ToolStartMsg:
		entry := ToolCallEntry{Name: msg.Name, Running: true}
		// Prepend (most recent at top).
		m.toolCalls = append([]ToolCallEntry{entry}, m.toolCalls...)
		if len(m.toolCalls) > m.maxTools {
			m.toolCalls = m.toolCalls[:m.maxTools]
		}
		return m, nil

	case ToolResultMsg:
		// Find the most recent running entry with this name.
		for i := range m.toolCalls {
			if m.toolCalls[i].Name == msg.Name && m.toolCalls[i].Running {
				m.toolCalls[i].Running = false
				m.toolCalls[i].Error = msg.Error != ""
				break
			}
		}
		return m, nil

	case AgentDoneMsg:
		m.agentBusy = false
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.height = msg.Height
		return m, nil
	}

	return m, nil
}

// View renders the three sidebar sections stacked vertically.
func (m SidebarModel) View() string {
	// Inner width accounts for border padding.
	innerWidth := m.width - 4
	if innerWidth < 10 {
		innerWidth = 10
	}

	sections := []string{
		m.renderToolActivity(innerWidth),
		m.renderAgentStatus(innerWidth),
		m.renderProjectInfo(innerWidth),
	}

	content := strings.Join(sections, "\n\n")

	return m.theme.SidebarBorder.
		Width(m.width).
		Height(m.height).
		Render(content)
}

// SetAgentBusy updates the agent busy state.
func (m *SidebarModel) SetAgentBusy(busy bool) {
	m.agentBusy = busy
}

// SetHeight updates the sidebar height.
func (m *SidebarModel) SetHeight(h int) {
	m.height = h
}

func (m SidebarModel) renderToolActivity(width int) string {
	heading := m.theme.SidebarHeading.Render("Tool Activity")
	separator := m.theme.SidebarItem.Render(strings.Repeat("\u2500", min(width, 15)))

	var lines []string
	lines = append(lines, heading, separator)

	if len(m.toolCalls) == 0 {
		lines = append(lines, m.theme.SidebarItem.Render("No activity"))
	} else {
		for _, tc := range m.toolCalls {
			lines = append(lines, m.renderToolEntry(tc))
		}
	}

	return strings.Join(lines, "\n")
}

func (m SidebarModel) renderToolEntry(tc ToolCallEntry) string {
	if tc.Running {
		return m.theme.ToolRunning.Render(m.spinner.View() + " " + tc.Name)
	}
	if tc.Error {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("\u2717 " + tc.Name)
	}
	return m.theme.ToolDone.Render("\u2713 " + tc.Name)
}

func (m SidebarModel) renderAgentStatus(width int) string {
	heading := m.theme.SidebarHeading.Render("Agent Status")
	separator := m.theme.SidebarItem.Render(strings.Repeat("\u2500", min(width, 15)))

	var status string
	if m.agentBusy {
		status = m.theme.ToolRunning.Render(m.spinner.View() + " Thinking...")
	} else {
		status = m.theme.SidebarItem.Render("Idle")
	}

	return fmt.Sprintf("%s\n%s\n%s", heading, separator, status)
}

func (m SidebarModel) renderProjectInfo(width int) string {
	heading := m.theme.SidebarHeading.Render("Project Info")
	separator := m.theme.SidebarItem.Render(strings.Repeat("\u2500", min(width, 15)))

	memStatus := "not loaded"
	if m.memoryLoaded {
		memStatus = "loaded"
	}

	lines := []string{
		heading,
		separator,
		m.theme.SidebarItem.Render(fmt.Sprintf("Dir: %s", m.projectDir)),
		m.theme.SidebarItem.Render(fmt.Sprintf("Memory: %s", memStatus)),
		m.theme.SidebarItem.Render(fmt.Sprintf("Tools: %d", m.toolCount)),
		m.theme.SidebarItem.Render(fmt.Sprintf("Model: %s", m.modelName)),
	}

	return strings.Join(lines, "\n")
}
