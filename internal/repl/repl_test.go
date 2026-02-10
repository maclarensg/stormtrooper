package repl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gavinyap/stormtrooper/internal/agent"
	"github.com/gavinyap/stormtrooper/internal/llm"
	"github.com/gavinyap/stormtrooper/internal/permission"
	"github.com/gavinyap/stormtrooper/internal/tool"
)

// sseTextResponse builds a complete SSE stream for a text-only response.
func sseTextResponse(content string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"\"},\"finish_reason\":null}]}\n\n"))
	contentJSON, _ := json.Marshal(content)
	b.WriteString(fmt.Sprintf("data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{\"content\":%s},\"finish_reason\":null}]}\n\n", string(contentJSON)))
	b.WriteString("data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
	b.WriteString("data: [DONE]\n")
	return b.String()
}

// newTestAgent creates an agent backed by a test HTTP server.
func newTestAgent(t *testing.T, server *httptest.Server) *agent.Agent {
	t.Helper()
	client := llm.NewClient("test-key")
	client.SetBaseURL(server.URL)
	reg := tool.NewRegistry()
	perm := permission.NewCheckerWithIO(strings.NewReader(""), &bytes.Buffer{})
	ag := agent.New(agent.Options{
		Client:     client,
		Registry:   reg,
		Permission: perm,
		Model:      "test-model",
	})
	ag.SetOutput(&bytes.Buffer{}, &bytes.Buffer{})
	return ag
}

func TestRun_ExitCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("LLM should not be called for /exit")
	}))
	defer server.Close()

	ag := newTestAgent(t, server)
	in := strings.NewReader("/exit\n")
	out := &bytes.Buffer{}
	inputReader := NewInputReaderWithIO(in, out)
	r := NewWithIO(ag, inputReader, out)

	err := r.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "Goodbye!") {
		t.Errorf("expected 'Goodbye!' in output, got %q", out.String())
	}
}

func TestRun_EOF(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("LLM should not be called for EOF")
	}))
	defer server.Close()

	ag := newTestAgent(t, server)
	in := strings.NewReader("") // immediate EOF
	out := &bytes.Buffer{}
	inputReader := NewInputReaderWithIO(in, out)
	r := NewWithIO(ag, inputReader, out)

	err := r.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "Goodbye!") {
		t.Errorf("expected 'Goodbye!' in output, got %q", out.String())
	}
}

func TestRun_CancelledContextExitsCleanly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("LLM should not be called when context is already cancelled")
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	ag := newTestAgent(t, server)
	// Provide input that would keep the loop going if context check is missing.
	in := strings.NewReader("hello\nhello\nhello\n")
	out := &bytes.Buffer{}
	inputReader := NewInputReaderWithIO(in, out)
	r := NewWithIO(ag, inputReader, out)

	err := r.Run(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "Goodbye!") {
		t.Errorf("expected 'Goodbye!' in output, got %q", out.String())
	}
}

func TestRun_ContextCancelledDuringSendExitsCleanly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Cancel context while the agent is processing, simulating Ctrl+C.
		cancel()
		// Return an error response by closing without valid SSE.
		w.Header().Set("Content-Type", "text/event-stream")
		// Close the connection without sending data — agent.Send will fail.
	}))
	defer server.Close()

	ag := newTestAgent(t, server)
	// Provide a message followed by more input — if context check is missing,
	// the loop would continue and try to send more.
	in := strings.NewReader("hello\nmore input\n")
	out := &bytes.Buffer{}
	inputReader := NewInputReaderWithIO(in, out)
	r := NewWithIO(ag, inputReader, out)

	err := r.Run(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "Goodbye!") {
		t.Errorf("expected 'Goodbye!' in output, got %q", out.String())
	}
	// Should NOT contain repeated error messages — it should exit after the first.
	errCount := strings.Count(out.String(), "Error:")
	if errCount > 0 {
		t.Errorf("expected no error messages (should exit cleanly), but found %d", errCount)
	}
}

func TestRun_NormalSendThenExit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte(sseTextResponse("Hello!")))
	}))
	defer server.Close()

	ag := newTestAgent(t, server)
	in := strings.NewReader("hi\n/exit\n")
	out := &bytes.Buffer{}
	inputReader := NewInputReaderWithIO(in, out)
	r := NewWithIO(ag, inputReader, out)

	err := r.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "Goodbye!") {
		t.Errorf("expected 'Goodbye!' in output, got %q", out.String())
	}
}
