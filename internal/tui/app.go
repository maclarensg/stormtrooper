package tui

import (
	gocontext "context"
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gavinyap/stormtrooper/internal/agent"
	"github.com/gavinyap/stormtrooper/internal/config"
	projectctx "github.com/gavinyap/stormtrooper/internal/context"
)

// FocusArea identifies which panel has keyboard focus.
type FocusArea int

const (
	FocusInput FocusArea = iota
	FocusChat
)

const sidebarWidth = 30

// App is the top-level Bubble Tea model that composes all sub-models.
type App struct {
	// Sub-models
	chat      ChatModel
	input     InputModel
	sidebar   SidebarModel
	statusbar StatusBarModel

	// Layout
	width  int
	height int
	focus  FocusArea

	// Agent integration
	bridge    *Bridge
	agent     *agent.Agent
	agentBusy bool

	// Permission state
	permReq *PermissionRequestMsg

	// Theme and keymap
	theme  Theme
	keymap KeyMap
}

// Options configures a new App.
type Options struct {
	Agent      *agent.Agent
	Config     *config.Config
	ProjectCtx *projectctx.ProjectContext
	Version    string
}

// New creates a new App, wiring the agent to the bridge and constructing
// all sub-models.
func New(opts Options) *App {
	theme := DefaultTheme()
	keymap := DefaultKeyMap()
	bridge := NewBridge()

	// Wire the agent's output and permission handler through the bridge.
	opts.Agent.SetOutput(bridge.Stdout(), bridge.Stderr())
	opts.Agent.SetPermission(bridge.Permission())

	// Derive sidebar options from project context and config.
	projectDir := ""
	memoryLoaded := false
	if opts.ProjectCtx != nil {
		projectDir = filepath.Base(opts.ProjectCtx.WorkingDir)
		memoryLoaded = opts.ProjectCtx.Memory != ""
	}

	modelName := ""
	if opts.Config != nil {
		modelName = opts.Config.Model
	}

	cwd := ""
	if opts.ProjectCtx != nil {
		cwd = opts.ProjectCtx.WorkingDir
	}

	return &App{
		chat:  NewChatModel(&theme),
		input: NewInputModel(&theme, &keymap),
		sidebar: NewSidebarModel(&theme, SidebarOptions{
			ProjectDir:   projectDir,
			MemoryLoaded: memoryLoaded,
			ToolCount:    0,
			ModelName:    modelName,
		}),
		statusbar: NewStatusBarModel(&theme, opts.Version, modelName, cwd),
		focus:     FocusInput,
		bridge:    bridge,
		agent:     opts.Agent,
		theme:     theme,
		keymap:    keymap,
	}
}

// Init starts the input cursor blink, sidebar spinner, and bridge event listener.
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.input.Init(),
		a.sidebar.Init(),
		WaitForEvent(a.bridge.Events()),
	)
}

// Update routes messages to the appropriate sub-models.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.recalcLayout()
		return a, nil

	case tea.KeyMsg:
		// Permission prompt takes priority over all other key handling.
		if a.permReq != nil {
			return a.handlePermissionKey(msg)
		}

		// Global keys.
		switch {
		case key.Matches(msg, a.keymap.Quit):
			return a, tea.Quit

		case key.Matches(msg, a.keymap.Tab):
			a.toggleFocus()
			return a, nil

		case key.Matches(msg, a.keymap.FocusChat):
			if a.focus == FocusInput {
				a.setFocus(FocusChat)
			} else {
				a.setFocus(FocusInput)
			}
			return a, nil
		}

		// Forward to focused sub-model.
		if a.focus == FocusInput {
			var cmd tea.Cmd
			a.input, cmd = a.input.Update(msg)
			cmds = append(cmds, cmd)
		} else {
			var cmd tea.Cmd
			a.chat, cmd = a.chat.Update(msg)
			cmds = append(cmds, cmd)
		}
		return a, tea.Batch(cmds...)

	case SendMsg:
		a.chat.AddUserMessage(msg.Text)
		a.agentBusy = true
		a.input.SetDisabled(true)
		a.sidebar.SetAgentBusy(true)
		return a, tea.Batch(
			a.runAgent(msg.Text),
			WaitForEvent(a.bridge.Events()),
			a.input.Init(), // restart spinner
			a.sidebar.Init(),
		)

	case TokenMsg:
		var cmd tea.Cmd
		a.chat, cmd = a.chat.Update(msg)
		cmds = append(cmds, cmd, WaitForEvent(a.bridge.Events()))
		return a, tea.Batch(cmds...)

	case ToolStartMsg:
		var chatCmd, sidebarCmd tea.Cmd
		a.chat, chatCmd = a.chat.Update(msg)
		a.sidebar, sidebarCmd = a.sidebar.Update(msg)
		cmds = append(cmds, chatCmd, sidebarCmd, WaitForEvent(a.bridge.Events()))
		return a, tea.Batch(cmds...)

	case ToolResultMsg:
		var chatCmd, sidebarCmd tea.Cmd
		a.chat, chatCmd = a.chat.Update(msg)
		a.sidebar, sidebarCmd = a.sidebar.Update(msg)
		cmds = append(cmds, chatCmd, sidebarCmd, WaitForEvent(a.bridge.Events()))
		return a, tea.Batch(cmds...)

	case PermissionRequestMsg:
		a.permReq = &msg
		var cmd tea.Cmd
		a.chat, cmd = a.chat.Update(msg)
		cmds = append(cmds, cmd, WaitForEvent(a.bridge.Events()))
		return a, tea.Batch(cmds...)

	case AgentDoneMsg:
		a.agentBusy = false
		a.input.SetDisabled(false)
		a.sidebar.SetAgentBusy(false)
		a.setFocus(FocusInput)

		if msg.Error != nil {
			a.chat.AddSystemMessage(fmt.Sprintf("Error: %v", msg.Error))
		}

		var chatCmd, sidebarCmd tea.Cmd
		a.chat, chatCmd = a.chat.Update(msg)
		a.sidebar, sidebarCmd = a.sidebar.Update(msg)
		cmds = append(cmds, chatCmd, sidebarCmd)
		return a, tea.Batch(cmds...)

	case SubAgentSpawnMsg:
		var cmd tea.Cmd
		a.chat, cmd = a.chat.Update(msg)
		cmds = append(cmds, cmd, WaitForEvent(a.bridge.Events()))
		return a, tea.Batch(cmds...)

	case SubAgentDoneMsg:
		cmds = append(cmds, WaitForEvent(a.bridge.Events()))
		return a, tea.Batch(cmds...)
	}

	// Forward spinner ticks and other messages to sub-models that need them.
	var inputCmd, sidebarCmd tea.Cmd
	a.input, inputCmd = a.input.Update(msg)
	a.sidebar, sidebarCmd = a.sidebar.Update(msg)
	cmds = append(cmds, inputCmd, sidebarCmd)
	return a, tea.Batch(cmds...)
}

// View composes the full TUI layout.
func (a *App) View() string {
	statusBar := a.statusbar.View()
	chatView := a.chat.View()
	sidebarView := a.sidebar.View()
	mainArea := lipgloss.JoinHorizontal(lipgloss.Top, chatView, sidebarView)
	inputView := a.input.View()

	return lipgloss.JoinVertical(lipgloss.Left, statusBar, mainArea, inputView)
}

// Bridge returns the agent bridge for external access (e.g., setting permission handler).
func (a *App) Bridge() *Bridge {
	return a.bridge
}

// handlePermissionKey processes y/n keys during a permission prompt.
func (a *App) handlePermissionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keymap.PermAllow):
		a.permReq.Response <- true
		a.permReq = nil
		var cmd tea.Cmd
		a.chat, cmd = a.chat.Update(PermissionResponseMsg{Allowed: true})
		return a, cmd

	case key.Matches(msg, a.keymap.PermDeny):
		a.permReq.Response <- false
		a.permReq = nil
		var cmd tea.Cmd
		a.chat, cmd = a.chat.Update(PermissionResponseMsg{Allowed: false})
		return a, cmd

	case key.Matches(msg, a.keymap.Quit):
		return a, tea.Quit
	}

	// Ignore all other keys during permission prompt.
	return a, nil
}

// toggleFocus switches between FocusInput and FocusChat.
func (a *App) toggleFocus() {
	if a.focus == FocusInput {
		a.setFocus(FocusChat)
	} else {
		a.setFocus(FocusInput)
	}
}

// setFocus changes focus and updates input focus/blur state.
func (a *App) setFocus(f FocusArea) {
	a.focus = f
	if f == FocusInput {
		a.input.Focus()
	} else {
		a.input.Blur()
	}
}

// recalcLayout distributes available space among sub-models.
func (a *App) recalcLayout() {
	// Status bar: 1 row.
	statusBarHeight := 1
	// Input: 3 rows + 2 for borders.
	inputHeight := 5
	// Sidebar width is fixed.
	sbWidth := sidebarWidth

	// Chat gets the remaining space.
	chatWidth := a.width - sbWidth
	if chatWidth < 10 {
		chatWidth = 10
	}
	chatHeight := a.height - statusBarHeight - inputHeight
	if chatHeight < 3 {
		chatHeight = 3
	}

	a.statusbar.SetWidth(a.width)
	a.chat.SetSize(chatWidth, chatHeight)
	a.sidebar.SetHeight(chatHeight)
	a.input.SetWidth(a.width)
}

// runAgent starts the agent in a goroutine and returns AgentDoneMsg when complete.
func (a *App) runAgent(userMessage string) tea.Cmd {
	ag := a.agent
	return func() tea.Msg {
		err := ag.Send(gocontext.Background(), userMessage)
		return AgentDoneMsg{Error: err}
	}
}
