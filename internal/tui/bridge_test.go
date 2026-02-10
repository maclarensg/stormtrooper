package tui

import (
	"testing"
	"time"

	"github.com/gavinyap/stormtrooper/internal/permission"
)

func TestEventWriter(t *testing.T) {
	ch := make(chan AgentEvent, 10)
	w := &EventWriter{events: ch}

	n, err := w.Write([]byte("hello world"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 11 {
		t.Fatalf("expected 11 bytes written, got %d", n)
	}

	select {
	case ev := <-ch:
		tok, ok := ev.(TokenMsg)
		if !ok {
			t.Fatalf("expected TokenMsg, got %T", ev)
		}
		if tok.Content != "hello world" {
			t.Fatalf("expected 'hello world', got %q", tok.Content)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestToolEventWriter_ToolStart(t *testing.T) {
	ch := make(chan AgentEvent, 10)
	w := &ToolEventWriter{events: ch}

	w.Write([]byte("[tool] read_file\n"))

	select {
	case ev := <-ch:
		msg, ok := ev.(ToolStartMsg)
		if !ok {
			t.Fatalf("expected ToolStartMsg, got %T", ev)
		}
		if msg.Name != "read_file" {
			t.Fatalf("expected 'read_file', got %q", msg.Name)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestToolEventWriter_PartialLines(t *testing.T) {
	ch := make(chan AgentEvent, 10)
	w := &ToolEventWriter{events: ch}

	// Write partial line
	w.Write([]byte("[tool] edi"))
	// No event should be emitted yet
	select {
	case ev := <-ch:
		t.Fatalf("unexpected event from partial line: %T", ev)
	default:
	}

	// Complete the line
	w.Write([]byte("t_file\n"))

	select {
	case ev := <-ch:
		msg, ok := ev.(ToolStartMsg)
		if !ok {
			t.Fatalf("expected ToolStartMsg, got %T", ev)
		}
		if msg.Name != "edit_file" {
			t.Fatalf("expected 'edit_file', got %q", msg.Name)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestToolEventWriter_PermissionDeniedIgnored(t *testing.T) {
	ch := make(chan AgentEvent, 10)
	w := &ToolEventWriter{events: ch}

	w.Write([]byte("[tool] shell_exec: permission denied\n"))

	select {
	case ev := <-ch:
		t.Fatalf("expected no event for permission denied, got %T", ev)
	default:
		// expected: no event
	}
}

func TestToolEventWriter_SubAgentSpawn(t *testing.T) {
	ch := make(chan AgentEvent, 10)
	w := &ToolEventWriter{events: ch}

	w.Write([]byte("[agent] Spawning sub-agent: Fix the login bug\n"))

	select {
	case ev := <-ch:
		msg, ok := ev.(SubAgentSpawnMsg)
		if !ok {
			t.Fatalf("expected SubAgentSpawnMsg, got %T", ev)
		}
		if msg.Task != "Fix the login bug" {
			t.Fatalf("expected 'Fix the login bug', got %q", msg.Task)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestToolEventWriter_SubAgentDone(t *testing.T) {
	ch := make(chan AgentEvent, 10)
	w := &ToolEventWriter{events: ch}

	w.Write([]byte("[agent] Sub-agent completed\n"))

	select {
	case ev := <-ch:
		_, ok := ev.(SubAgentDoneMsg)
		if !ok {
			t.Fatalf("expected SubAgentDoneMsg, got %T", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestToolEventWriter_MultipleLines(t *testing.T) {
	ch := make(chan AgentEvent, 10)
	w := &ToolEventWriter{events: ch}

	w.Write([]byte("[tool] read_file\n[tool] edit_file\n"))

	events := make([]AgentEvent, 0, 2)
	for i := 0; i < 2; i++ {
		select {
		case ev := <-ch:
			events = append(events, ev)
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for event %d", i)
		}
	}

	msg1, ok := events[0].(ToolStartMsg)
	if !ok {
		t.Fatalf("expected ToolStartMsg, got %T", events[0])
	}
	if msg1.Name != "read_file" {
		t.Fatalf("expected 'read_file', got %q", msg1.Name)
	}

	msg2, ok := events[1].(ToolStartMsg)
	if !ok {
		t.Fatalf("expected ToolStartMsg, got %T", events[1])
	}
	if msg2.Name != "edit_file" {
		t.Fatalf("expected 'edit_file', got %q", msg2.Name)
	}
}

func TestPermissionInterceptor_Allow(t *testing.T) {
	ch := make(chan AgentEvent, 10)
	interceptor := NewPermissionInterceptor(ch)

	// Verify it implements permission.Handler.
	var _ permission.Handler = interceptor

	done := make(chan bool, 1)
	go func() {
		result := interceptor.Check("shell_exec", "Run: ls -la")
		done <- result
	}()

	// Receive the permission request from the channel.
	select {
	case ev := <-ch:
		msg, ok := ev.(PermissionRequestMsg)
		if !ok {
			t.Fatalf("expected PermissionRequestMsg, got %T", ev)
		}
		if msg.ToolName != "shell_exec" {
			t.Fatalf("expected 'shell_exec', got %q", msg.ToolName)
		}
		if msg.Preview != "Run: ls -la" {
			t.Fatalf("expected 'Run: ls -la', got %q", msg.Preview)
		}
		if msg.ID == "" {
			t.Fatal("expected non-empty ID")
		}
		// Allow
		msg.Response <- true
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for permission request")
	}

	select {
	case result := <-done:
		if !result {
			t.Fatal("expected Check to return true")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Check result")
	}
}

func TestPermissionInterceptor_Deny(t *testing.T) {
	ch := make(chan AgentEvent, 10)
	interceptor := NewPermissionInterceptor(ch)

	done := make(chan bool, 1)
	go func() {
		result := interceptor.Check("write_file", "Write to /etc/passwd")
		done <- result
	}()

	select {
	case ev := <-ch:
		msg := ev.(PermissionRequestMsg)
		msg.Response <- false
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for permission request")
	}

	select {
	case result := <-done:
		if result {
			t.Fatal("expected Check to return false")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Check result")
	}
}

func TestBridge(t *testing.T) {
	b := NewBridge()

	if b.Stdout() == nil {
		t.Fatal("Stdout() should not be nil")
	}
	if b.Stderr() == nil {
		t.Fatal("Stderr() should not be nil")
	}
	if b.Permission() == nil {
		t.Fatal("Permission() should not be nil")
	}
	if b.Events() == nil {
		t.Fatal("Events() should not be nil")
	}

	// Verify all components write to the same channel.
	b.Stdout().Write([]byte("token"))
	select {
	case ev := <-b.Events():
		if _, ok := ev.(TokenMsg); !ok {
			t.Fatalf("expected TokenMsg from Stdout, got %T", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}

	b.Stderr().Write([]byte("[tool] grep\n"))
	select {
	case ev := <-b.Events():
		if _, ok := ev.(ToolStartMsg); !ok {
			t.Fatalf("expected ToolStartMsg from Stderr, got %T", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
}

func TestWaitForEvent(t *testing.T) {
	ch := make(chan AgentEvent, 1)
	ch <- TokenMsg{Content: "test"}

	cmd := WaitForEvent(ch)
	msg := cmd()

	tok, ok := msg.(TokenMsg)
	if !ok {
		t.Fatalf("expected TokenMsg, got %T", msg)
	}
	if tok.Content != "test" {
		t.Fatalf("expected 'test', got %q", tok.Content)
	}
}

func TestWaitForEvent_ClosedChannel(t *testing.T) {
	ch := make(chan AgentEvent)
	close(ch)

	cmd := WaitForEvent(ch)
	msg := cmd()
	if msg != nil {
		t.Fatalf("expected nil from closed channel, got %T", msg)
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()
	if id1 == id2 {
		t.Fatalf("expected unique IDs, got %q and %q", id1, id2)
	}
}
