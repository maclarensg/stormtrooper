package tui

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/gavinyap/stormtrooper/internal/agent"
	"github.com/gavinyap/stormtrooper/internal/config"
	projectctx "github.com/gavinyap/stormtrooper/internal/context"
	"github.com/gavinyap/stormtrooper/internal/llm"
	"github.com/gavinyap/stormtrooper/internal/permission"
	"github.com/gavinyap/stormtrooper/internal/tool"
	"github.com/muesli/termenv"
)

func init() {
	// Force Ascii color profile in all integration tests to strip ANSI
	// escape codes and prevent terminal queries.
	lipgloss.SetColorProfile(termenv.Ascii)
	lipgloss.SetHasDarkBackground(true)
}

// newIntegrationApp creates an App backed by an LLM client that points at
// a local test server. This avoids nil-pointer panics when the agent loop
// runs.
func newIntegrationApp(t *testing.T) *App {
	t.Helper()

	// Set up a local HTTP server that returns 401 for any request.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":{"message":"unauthorized"}}`)
	}))
	t.Cleanup(srv.Close)

	client := llm.NewClient("fake-key")
	client.SetBaseURL(srv.URL)

	perm := permission.NewChecker()
	reg := tool.NewRegistry()
	ag := agent.New(agent.Options{
		Client:     client,
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
		Version: "v0.2.2-test",
	})
}

// TestIntegration_NoGarbledText verifies the TUI renders without escape
// sequence leaks such as 3030, 0a0a, rgb:, or 2424.
func TestIntegration_NoGarbledText(t *testing.T) {
	app := newIntegrationApp(t)
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(120, 40))

	// Wait for the sidebar content to appear, indicating the initial render is done.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "Tool Activity")
	}, teatest.WithDuration(3*time.Second))

	// Now grab the final output and check for garbled patterns.
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	out := tm.FinalOutput(t, teatest.WithFinalTimeout(3*time.Second))
	buf := make([]byte, 64*1024)
	n, _ := out.Read(buf)
	output := string(buf[:n])

	garbledPatterns := []string{"3030", "0a0a", "2424", "rgb:"}
	for _, pattern := range garbledPatterns {
		if strings.Contains(output, pattern) {
			t.Errorf("found garbled pattern %q in TUI output:\n%s", pattern, output)
		}
	}
}

// TestIntegration_ErrorDisplay verifies that when an agent error occurs, the
// error message is visible in the TUI output.
func TestIntegration_ErrorDisplay(t *testing.T) {
	app := newIntegrationApp(t)
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(120, 40))

	// Wait for initial render.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "Tool Activity")
	}, teatest.WithDuration(3*time.Second))

	// Simulate an agent error.
	tm.Send(AgentDoneMsg{Error: errors.New("API error (status 401): unauthorized")})

	// Wait for the error to appear in the rendered output.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		s := string(bts)
		return strings.Contains(s, "401") || strings.Contains(s, "unauthorized")
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

// TestIntegration_CtrlC_Exits verifies that pressing Ctrl+C causes the
// program to exit cleanly without hanging.
func TestIntegration_CtrlC_Exits(t *testing.T) {
	app := newIntegrationApp(t)
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(120, 40))

	// Give the program a moment to start the event loop.
	time.Sleep(100 * time.Millisecond)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
	// If we reach here without a timeout, the program exited cleanly.
}

// TestIntegration_DashboardLayout verifies that all major dashboard sections
// are rendered: Tool Activity, Agent Status, and Project Info.
func TestIntegration_DashboardLayout(t *testing.T) {
	app := newIntegrationApp(t)
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(120, 40))

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		s := string(bts)
		return strings.Contains(s, "Tool Activity") &&
			strings.Contains(s, "Agent Status") &&
			strings.Contains(s, "Project Info")
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

// TestIntegration_UserMessageAppears verifies that after sending a user
// message via SendMsg, the message text appears in the rendered output.
// The agent will fail with a 401 from the test server, but the user message
// should still be visible in the chat.
func TestIntegration_UserMessageAppears(t *testing.T) {
	app := newIntegrationApp(t)
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(120, 40))

	// Wait for initial render.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "Tool Activity")
	}, teatest.WithDuration(3*time.Second))

	// Send a user message. The agent loop will run but fail with a 401
	// from the test server, returning AgentDoneMsg with an error.
	tm.Send(SendMsg{Text: "hello world"})

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "hello world")
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
