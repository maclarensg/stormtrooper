package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// SendMsg is emitted when the user presses Enter with non-empty input.
type SendMsg struct {
	Text string
}

// InputModel wraps a textarea with send/newline behavior and a busy spinner.
type InputModel struct {
	textarea textarea.Model
	theme    *Theme
	keymap   *KeyMap
	width    int
	height   int // typically 3 rows
	disabled bool
	spinner  spinner.Model
}

// NewInputModel creates an InputModel with configured textarea defaults.
func NewInputModel(theme *Theme, keymap *KeyMap) InputModel {
	ta := textarea.New()
	ta.Placeholder = "Type a message... (Enter to send, Ctrl+J for newline)"
	ta.ShowLineNumbers = false
	ta.CharLimit = 10000
	ta.SetHeight(3)
	ta.MaxHeight = 5
	ta.Focus()

	s := spinner.New()
	s.Spinner = spinner.Dot

	return InputModel{
		textarea: ta,
		theme:    theme,
		keymap:   keymap,
		height:   3,
		spinner:  s,
	}
}

// Init returns the textarea blink command for cursor animation.
func (m InputModel) Init() tea.Cmd {
	return textarea.Blink
}

// Update handles key presses, resizes, and spinner ticks.
func (m InputModel) Update(msg tea.Msg) (InputModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.disabled {
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keymap.Send):
			text := strings.TrimSpace(m.textarea.Value())
			if text == "" {
				return m, nil
			}
			m.textarea.Reset()
			return m, func() tea.Msg { return SendMsg{Text: text} }

		case key.Matches(msg, m.keymap.NewLine):
			m.textarea.InsertString("\n")
			return m, nil

		default:
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}

	case tea.WindowSizeMsg:
		m.SetWidth(msg.Width)
		return m, nil

	case spinner.TickMsg:
		if m.disabled {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	// Forward other messages to the textarea.
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

// View renders the input area. When disabled, shows a spinner.
func (m InputModel) View() string {
	if m.disabled {
		return m.theme.InputBorder.
			Width(m.width).
			Render(m.spinner.View() + " Thinking...")
	}
	return m.textarea.View()
}

// SetDisabled enables or disables input. When disabled, the spinner is shown.
func (m *InputModel) SetDisabled(disabled bool) {
	m.disabled = disabled
}

// Focus gives keyboard focus to the textarea.
func (m *InputModel) Focus() tea.Cmd {
	return m.textarea.Focus()
}

// Blur removes keyboard focus from the textarea.
func (m *InputModel) Blur() {
	m.textarea.Blur()
}

// SetWidth updates the input width.
func (m *InputModel) SetWidth(w int) {
	m.width = w
	m.textarea.SetWidth(w)
}
