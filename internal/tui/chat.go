package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// MessageRole identifies who authored a chat message.
type MessageRole int

const (
	RoleUser      MessageRole = iota
	RoleAssistant             // assistant (markdown-rendered)
	RoleTool                  // inline tool activity
	RoleSystem                // permission prompts, errors
)

// ChatMessage represents a single entry in the conversation.
type ChatMessage struct {
	Role    MessageRole
	Content string
	Time    time.Time
}

// ChatModel is the Bubble Tea model for the scrollable chat viewport.
type ChatModel struct {
	viewport   viewport.Model
	messages   []ChatMessage
	streaming  *strings.Builder // accumulates current assistant response tokens
	theme      *Theme
	width      int
	height     int
	autoScroll bool
	renderer   *glamour.TermRenderer
}

// NewChatModel creates a ChatModel with the given theme.
func NewChatModel(theme *Theme) ChatModel {
	vp := viewport.New(0, 0)
	vp.SetContent("")

	r, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)

	return ChatModel{
		viewport:   vp,
		theme:      theme,
		streaming:  &strings.Builder{},
		autoScroll: true,
		renderer:   r,
	}
}

// AddUserMessage appends a user message and re-renders the viewport.
func (m *ChatModel) AddUserMessage(content string) {
	m.messages = append(m.messages, ChatMessage{
		Role:    RoleUser,
		Content: content,
		Time:    time.Now(),
	})
	m.renderAll()
}

// SetSize updates the viewport dimensions and recreates the glamour renderer
// with the new width.
func (m *ChatModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w
	m.viewport.Height = h
	if w > 0 {
		r, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(w-4), // leave a small margin
		)
		if err == nil {
			m.renderer = r
		}
	}
	m.renderAll()
}

// Init returns nil; no initial commands are needed.
func (m ChatModel) Init() tea.Cmd {
	return nil
}

// Update handles incoming messages.
func (m ChatModel) Update(msg tea.Msg) (ChatModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)

	case TokenMsg:
		m.streaming.WriteString(msg.Content)
		m.renderAll()
		if m.autoScroll {
			m.viewport.GotoBottom()
		}

	case AgentDoneMsg:
		// Finalize the streaming message.
		if m.streaming.Len() > 0 {
			m.messages = append(m.messages, ChatMessage{
				Role:    RoleAssistant,
				Content: m.streaming.String(),
				Time:    time.Now(),
			})
			m.streaming.Reset()
		}
		m.renderAll()
		if m.autoScroll {
			m.viewport.GotoBottom()
		}

	case ToolStartMsg:
		m.messages = append(m.messages, ChatMessage{
			Role:    RoleTool,
			Content: fmt.Sprintf("> %s %s", msg.Name, msg.Args),
			Time:    time.Now(),
		})
		m.renderAll()
		if m.autoScroll {
			m.viewport.GotoBottom()
		}

	case ToolResultMsg:
		// Update the most recent tool message with the same name.
		for i := len(m.messages) - 1; i >= 0; i-- {
			if m.messages[i].Role == RoleTool && strings.HasPrefix(m.messages[i].Content, "> "+msg.Name) {
				if msg.Error != "" {
					m.messages[i].Content = fmt.Sprintf("> %s \u2717", msg.Name)
				} else {
					m.messages[i].Content = fmt.Sprintf("> %s \u2713", msg.Name)
				}
				break
			}
		}
		m.renderAll()
		if m.autoScroll {
			m.viewport.GotoBottom()
		}

	case PermissionRequestMsg:
		prompt := fmt.Sprintf("[PERMISSION] %s\n%s\n[y] allow  [n] deny", msg.ToolName, msg.Preview)
		m.messages = append(m.messages, ChatMessage{
			Role:    RoleSystem,
			Content: prompt,
			Time:    time.Now(),
		})
		m.renderAll()
		if m.autoScroll {
			m.viewport.GotoBottom()
		}

	case PermissionResponseMsg:
		// Update the last permission prompt to show the result.
		for i := len(m.messages) - 1; i >= 0; i-- {
			if m.messages[i].Role == RoleSystem && strings.HasPrefix(m.messages[i].Content, "[PERMISSION]") {
				if msg.Allowed {
					m.messages[i].Content += "\n-> Allowed"
				} else {
					m.messages[i].Content += "\n-> Denied"
				}
				break
			}
		}
		m.renderAll()
		if m.autoScroll {
			m.viewport.GotoBottom()
		}

	case tea.KeyMsg:
		// When chat has focus, forward keys to viewport for scrolling.
		prevOffset := m.viewport.YOffset
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)

		// If the user scrolled up, disable auto-scroll.
		// If they scroll back to the bottom, re-enable it.
		if m.viewport.YOffset != prevOffset {
			m.autoScroll = m.viewport.AtBottom()
		}
		return m, tea.Batch(cmds...)
	}

	// Always forward to viewport for internal handling (mouse, etc.)
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View returns the rendered viewport.
func (m ChatModel) View() string {
	return m.viewport.View()
}

// renderAll rebuilds the entire viewport content from messages and the
// current streaming buffer.
func (m *ChatModel) renderAll() {
	var sections []string

	for _, msg := range m.messages {
		sections = append(sections, m.renderMessage(msg))
	}

	// If we're currently streaming, render the partial assistant response.
	if m.streaming.Len() > 0 {
		prefix := m.theme.AssistantPrefix.Render("Assistant:")
		content := m.renderMarkdown(m.streaming.String())
		sections = append(sections, prefix+"\n"+content)
	}

	full := strings.Join(sections, "\n\n")
	m.viewport.SetContent(full)
}

// renderMessage renders a single ChatMessage according to its role.
func (m *ChatModel) renderMessage(msg ChatMessage) string {
	switch msg.Role {
	case RoleUser:
		prefix := m.theme.UserPrefix.Render("You:")
		content := m.theme.UserMessage.Render(msg.Content)
		return prefix + "\n" + content

	case RoleAssistant:
		prefix := m.theme.AssistantPrefix.Render("Assistant:")
		content := m.renderMarkdown(msg.Content)
		return prefix + "\n" + content

	case RoleTool:
		return m.theme.ToolInline.Render("  " + msg.Content)

	case RoleSystem:
		// Permission prompts get the amber/yellow bordered box.
		box := m.theme.PermissionBorder.
			Width(m.width - 4).
			Render(m.theme.PermissionText.Render(msg.Content))
		return box

	default:
		return msg.Content
	}
}

// renderMarkdown renders markdown text through glamour. Falls back to raw text
// if rendering fails.
func (m *ChatModel) renderMarkdown(text string) string {
	if m.renderer == nil {
		return text
	}
	rendered, err := m.renderer.Render(text)
	if err != nil {
		return text
	}
	// glamour adds trailing newlines; trim them to keep spacing consistent.
	return strings.TrimRight(rendered, "\n")
}

// borderStyle returns a styled border for the chat panel.
func (m ChatModel) borderStyle() lipgloss.Style {
	return m.theme.ChatBorder.
		Width(m.width).
		Height(m.height)
}
