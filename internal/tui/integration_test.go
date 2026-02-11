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

// newStreamingIntegrationApp creates an App backed by a mock SSE server that
// streams the given tokens as individual SSE data lines. This exercises the
// full path: agent -> LLM client -> SSE parser -> EventWriter -> bridge ->
// TUI Update loop.
func newStreamingIntegrationApp(t *testing.T, tokens []string) *App {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Send role delta first.
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"role\":\"assistant\"}}]}\n\n")

		// Stream each token as a separate SSE event.
		for _, tok := range tokens {
			// Escape the token for JSON.
			escaped := strings.ReplaceAll(tok, `\`, `\\`)
			escaped = strings.ReplaceAll(escaped, `"`, `\"`)
			fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"%s\"}}]}\n\n", escaped)
		}

		fmt.Fprintf(w, "data: [DONE]\n\n")
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
		Version: "v0.2.4-test",
	})
}

// TestIntegration_StreamingTokenOrder verifies that tokens streamed from the
// LLM arrive in the correct order in the TUI output. This is the core
// regression test for the duplicate WaitForEvent bug.
//
// Under race detection, timing differences can cause AgentDoneMsg to arrive
// before all tokens are consumed from the bridge channel. This is a known
// design characteristic: the test verifies that tokens are never reordered
// (the original bug), not that they all land in a single render frame.
func TestIntegration_StreamingTokenOrder(t *testing.T) {
	tokens := []string{"Alpha", " Beta", " Gamma", " Delta", " Epsilon"}
	app := newStreamingIntegrationApp(t, tokens)
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(120, 40))

	// Wait for initial render.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "Tool Activity")
	}, teatest.WithDuration(3*time.Second))

	// Send a user message to trigger the agent loop.
	tm.Send(SendMsg{Text: "test streaming"})

	// Accumulate the full output stream and verify tokens appear in order.
	// Each render frame is appended, so we see the progression of all
	// rendered text. We wait until the last token "Epsilon" appears,
	// then verify ordering across the accumulated output.
	var accumulated strings.Builder
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		accumulated.Write(bts)
		return strings.Contains(accumulated.String(), "Epsilon")
	}, teatest.WithDuration(10*time.Second))

	full := accumulated.String()
	// Verify all tokens appeared and in the correct relative order.
	prev := 0
	for _, tok := range []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon"} {
		idx := strings.Index(full[prev:], tok)
		if idx < 0 {
			t.Fatalf("token %q not found after position %d in accumulated output", tok, prev)
		}
		prev = prev + idx + len(tok)
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

// TestIntegration_StreamingMidStreamError verifies that when the SSE server
// returns malformed JSON mid-stream, the agent surfaces an error in the TUI.
func TestIntegration_StreamingMidStreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"role\":\"assistant\"}}]}\n\n")
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Partial\"}}]}\n\n")
		// Send malformed JSON to trigger a parse error.
		fmt.Fprintf(w, "data: {INVALID JSON}\n\n")
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

	app := New(Options{
		Agent: ag,
		Config: &config.Config{
			Model: "test-model",
		},
		ProjectCtx: &projectctx.ProjectContext{
			WorkingDir: "/home/user/myproject",
			Memory:     "some memory",
		},
		Version: "v0.2.4-test",
	})

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(120, 40))

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "Tool Activity")
	}, teatest.WithDuration(3*time.Second))

	tm.Send(SendMsg{Text: "test error"})

	// The agent should finish with an error due to malformed SSE.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		s := string(bts)
		return strings.Contains(s, "Error")
	}, teatest.WithDuration(5*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
