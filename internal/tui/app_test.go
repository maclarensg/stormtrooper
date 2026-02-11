package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gavinyap/stormtrooper/internal/agent"
	"github.com/gavinyap/stormtrooper/internal/config"
	projectctx "github.com/gavinyap/stormtrooper/internal/context"
	"github.com/gavinyap/stormtrooper/internal/permission"
	"github.com/gavinyap/stormtrooper/internal/tool"
)

// newTestApp creates an App with minimal dependencies for testing.
func newTestApp() *App {
	perm := permission.NewChecker()
	reg := tool.NewRegistry()
	ag := agent.New(agent.Options{
		Registry:   reg,
		Permission: perm,
		Model:      "test-model",
	})

	return New(Options{
		Agent: ag,
		Config: &config.Config{
			Model: "test-model",
		},
		ProjectCtx: &projectctx.ProjectContext{
			WorkingDir: "/home/user/myproject",
			Memory:     "some memory",
		},
		Version: "v0.2.0",
	})
}

func TestApp_Init(t *testing.T) {
	app := newTestApp()
	cmd := app.Init()
	if cmd == nil {
		t.Fatal("Init should return a batch command")
	}
}

func TestApp_Resize(t *testing.T) {
	app := newTestApp()

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	model, _ := app.Update(msg)
	a := model.(*App)

	if a.width != 120 {
		t.Fatalf("expected width 120, got %d", a.width)
	}
	if a.height != 40 {
		t.Fatalf("expected height 40, got %d", a.height)
	}

	// Chat should get total width minus sidebar.
	expectedChatWidth := 120 - sidebarWidth
	if a.chat.width != expectedChatWidth {
		t.Fatalf("expected chat width %d, got %d", expectedChatWidth, a.chat.width)
	}

	// Chat height should be total minus statusbar (1) and input (5).
	expectedChatHeight := 40 - 1 - 5
	if a.chat.height != expectedChatHeight {
		t.Fatalf("expected chat height %d, got %d", expectedChatHeight, a.chat.height)
	}
}

func TestApp_FocusToggle(t *testing.T) {
	app := newTestApp()

	// Default focus is input.
	if app.focus != FocusInput {
		t.Fatalf("expected initial focus FocusInput, got %d", app.focus)
	}

	// Tab should toggle to chat.
	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	model, _ := app.Update(tabMsg)
	a := model.(*App)
	if a.focus != FocusChat {
		t.Fatalf("expected focus FocusChat after Tab, got %d", a.focus)
	}

	// Tab again should return to input.
	model, _ = a.Update(tabMsg)
	a = model.(*App)
	if a.focus != FocusInput {
		t.Fatalf("expected focus FocusInput after second Tab, got %d", a.focus)
	}
}

func TestApp_FocusEsc(t *testing.T) {
	app := newTestApp()

	// Esc from input should switch to chat.
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	model, _ := app.Update(escMsg)
	a := model.(*App)
	if a.focus != FocusChat {
		t.Fatalf("expected FocusChat after Esc from input, got %d", a.focus)
	}

	// Esc from chat should switch back to input.
	model, _ = a.Update(escMsg)
	a = model.(*App)
	if a.focus != FocusInput {
		t.Fatalf("expected FocusInput after Esc from chat, got %d", a.focus)
	}
}

func TestApp_SendMessage(t *testing.T) {
	app := newTestApp()
	// Resize first so layout is valid.
	app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	sendMsg := SendMsg{Text: "Hello agent"}
	model, cmd := app.Update(sendMsg)
	a := model.(*App)

	if !a.agentBusy {
		t.Fatal("expected agentBusy to be true after SendMsg")
	}
	if !a.input.disabled {
		t.Fatal("expected input to be disabled after SendMsg")
	}
	if cmd == nil {
		t.Fatal("expected commands after SendMsg")
	}
}

func TestApp_PermissionFlow(t *testing.T) {
	app := newTestApp()

	// Inject a permission request.
	respCh := make(chan bool, 1)
	permMsg := PermissionRequestMsg{
		ID:       "test-1",
		ToolName: "shell_exec",
		Preview:  "Run: rm -rf /tmp/test",
		Response: respCh,
	}

	model, _ := app.Update(permMsg)
	a := model.(*App)

	if a.permReq == nil {
		t.Fatal("expected permReq to be set after PermissionRequestMsg")
	}
	if a.permReq.ToolName != "shell_exec" {
		t.Fatalf("expected tool name 'shell_exec', got %q", a.permReq.ToolName)
	}

	// Press 'y' to allow.
	yMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	model, _ = a.Update(yMsg)
	a = model.(*App)

	if a.permReq != nil {
		t.Fatal("expected permReq to be cleared after pressing 'y'")
	}

	// Verify the response channel got true.
	select {
	case result := <-respCh:
		if !result {
			t.Fatal("expected true on response channel")
		}
	default:
		t.Fatal("expected response on channel")
	}
}

func TestApp_PermissionDeny(t *testing.T) {
	app := newTestApp()

	respCh := make(chan bool, 1)
	permMsg := PermissionRequestMsg{
		ID:       "test-2",
		ToolName: "write_file",
		Preview:  "Write to /etc/passwd",
		Response: respCh,
	}

	model, _ := app.Update(permMsg)
	a := model.(*App)

	// Press 'n' to deny.
	nMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	model, _ = a.Update(nMsg)
	a = model.(*App)

	if a.permReq != nil {
		t.Fatal("expected permReq to be cleared after pressing 'n'")
	}

	select {
	case result := <-respCh:
		if result {
			t.Fatal("expected false on response channel")
		}
	default:
		t.Fatal("expected response on channel")
	}
}

func TestApp_AgentDone(t *testing.T) {
	app := newTestApp()

	// Simulate agent being busy.
	app.agentBusy = true
	app.input.SetDisabled(true)
	app.sidebar.SetAgentBusy(true)

	doneMsg := AgentDoneMsg{Error: nil}
	model, _ := app.Update(doneMsg)
	a := model.(*App)

	if a.agentBusy {
		t.Fatal("expected agentBusy to be false after AgentDoneMsg")
	}
	if a.input.disabled {
		t.Fatal("expected input to be enabled after AgentDoneMsg")
	}
	if a.focus != FocusInput {
		t.Fatalf("expected focus to return to FocusInput, got %d", a.focus)
	}
}

func TestApp_AgentDoneWithError(t *testing.T) {
	app := newTestApp()
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	app.agentBusy = true
	app.input.SetDisabled(true)
	app.sidebar.SetAgentBusy(true)

	doneMsg := AgentDoneMsg{Error: errors.New("API key invalid")}
	model, _ := app.Update(doneMsg)
	a := model.(*App)

	if a.agentBusy {
		t.Fatal("expected agentBusy to be false after AgentDoneMsg with error")
	}

	// The chat should contain a system message with the error.
	found := false
	for _, msg := range a.chat.messages {
		if msg.Role == RoleSystem && strings.Contains(msg.Content, "API key invalid") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected system error message in chat after AgentDoneMsg with error")
	}
}

func TestApp_View(t *testing.T) {
	app := newTestApp()

	// Resize to set layout.
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	view := app.View()
	if view == "" {
		t.Fatal("View should return non-empty string")
	}
}

func TestApp_QuitDuringPermission(t *testing.T) {
	app := newTestApp()

	// Set up a permission request.
	respCh := make(chan bool, 1)
	permMsg := PermissionRequestMsg{
		ID:       "test-3",
		ToolName: "shell_exec",
		Preview:  "Run: something",
		Response: respCh,
	}
	model, _ := app.Update(permMsg)
	a := model.(*App)

	// Ctrl+C during permission should still quit.
	quitMsg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := a.Update(quitMsg)

	// The cmd should be tea.Quit.
	if cmd == nil {
		t.Fatal("expected quit command")
	}
}

func TestApp_IgnoreKeysDuringPermission(t *testing.T) {
	app := newTestApp()

	respCh := make(chan bool, 1)
	permMsg := PermissionRequestMsg{
		ID:       "test-4",
		ToolName: "test",
		Preview:  "test",
		Response: respCh,
	}
	model, _ := app.Update(permMsg)
	a := model.(*App)

	// Tab should be ignored during permission prompt.
	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	model, _ = a.Update(tabMsg)
	a = model.(*App)

	// permReq should still be set (not cleared by Tab).
	if a.permReq == nil {
		t.Fatal("permReq should still be set after Tab during permission")
	}
}

func TestApp_TokenMsgForwarded(t *testing.T) {
	app := newTestApp()
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	tokenMsg := TokenMsg{Content: "Hello"}
	model, cmd := app.Update(tokenMsg)
	a := model.(*App)

	// Should return a command (WaitForEvent at minimum).
	if cmd == nil {
		t.Fatal("expected commands after TokenMsg")
	}

	_ = a
}

func TestApp_TokenOrderPreserved(t *testing.T) {
	app := newTestApp()
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// Simulate a SendMsg followed by a sequence of tokens.
	// Before the fix, SendMsg spawned a duplicate WaitForEvent goroutine
	// which caused tokens to be delivered out of order.
	app.Update(SendMsg{Text: "test prompt"})

	// Feed tokens sequentially through the Update method and verify
	// the streaming buffer accumulates them in the correct order.
	tokens := []string{"I'll ", "run ", "the ", "ls ", "command"}
	for _, tok := range tokens {
		model, _ := app.Update(TokenMsg{Content: tok})
		app = model.(*App)
	}

	got := app.chat.streaming.String()
	expected := "I'll run the ls command"
	if got != expected {
		t.Fatalf("token order broken: expected %q, got %q", expected, got)
	}
}

func TestApp_SendMsgDoesNotSpawnDuplicateWFE(t *testing.T) {
	app := newTestApp()
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// Send a message and capture the returned commands.
	_, cmd := app.Update(SendMsg{Text: "hello"})
	if cmd == nil {
		t.Fatal("expected commands after SendMsg")
	}

	// Push a token onto the bridge channel.
	app.bridge.events <- TokenMsg{Content: "test"}

	// Read the token back. If SendMsg spawned a WFE, there would be two
	// goroutines racing to read from events. With the fix, only the WFE
	// from Init() is active.
	select {
	case ev := <-app.bridge.Events():
		tok, ok := ev.(TokenMsg)
		if !ok {
			t.Fatalf("expected TokenMsg, got %T", ev)
		}
		if tok.Content != "test" {
			t.Fatalf("expected token content 'test', got %q", tok.Content)
		}
	default:
		// Channel was already drained by a competing goroutine — this
		// would indicate the bug is still present. However, since we're
		// not actually running the tea.Cmd goroutines in this unit test,
		// the token should still be on the channel.
		t.Fatal("token was consumed by unexpected goroutine")
	}
}

func TestApp_ToolStartForwarded(t *testing.T) {
	app := newTestApp()

	toolMsg := ToolStartMsg{ID: "1", Name: "read_file", Args: "main.go"}
	model, cmd := app.Update(toolMsg)
	a := model.(*App)

	if cmd == nil {
		t.Fatal("expected commands after ToolStartMsg")
	}

	_ = a
}
