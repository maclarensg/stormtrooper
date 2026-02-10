package repl

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestReadInput_SingleLine(t *testing.T) {
	in := strings.NewReader("hello world\n")
	out := &bytes.Buffer{}
	r := NewInputReaderWithIO(in, out)

	input, err := r.ReadInput()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if input != "hello world" {
		t.Errorf("expected 'hello world', got %q", input)
	}
	if !strings.Contains(out.String(), "> ") {
		t.Error("expected primary prompt in output")
	}
}

func TestReadInput_MultiLine(t *testing.T) {
	in := strings.NewReader("first line\\\nsecond line\\\nthird line\n")
	out := &bytes.Buffer{}
	r := NewInputReaderWithIO(in, out)

	input, err := r.ReadInput()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if input != "first line\nsecond line\nthird line" {
		t.Errorf("expected multi-line input, got %q", input)
	}
	// Should have printed continuation prompts.
	if strings.Count(out.String(), ". ") != 2 {
		t.Errorf("expected 2 continuation prompts, got output: %q", out.String())
	}
}

func TestReadInput_EmptyLine(t *testing.T) {
	in := strings.NewReader("\n")
	out := &bytes.Buffer{}
	r := NewInputReaderWithIO(in, out)

	input, err := r.ReadInput()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if input != "" {
		t.Errorf("expected empty input, got %q", input)
	}
}

func TestReadInput_EOF(t *testing.T) {
	in := strings.NewReader("")
	out := &bytes.Buffer{}
	r := NewInputReaderWithIO(in, out)

	_, err := r.ReadInput()
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}

func TestReadInput_ExitCommand(t *testing.T) {
	in := strings.NewReader("/exit\n")
	out := &bytes.Buffer{}
	r := NewInputReaderWithIO(in, out)

	input, err := r.ReadInput()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if input != "/exit" {
		t.Errorf("expected '/exit', got %q", input)
	}
}

func TestReadInput_MultipleReads(t *testing.T) {
	in := strings.NewReader("first\nsecond\n")
	out := &bytes.Buffer{}
	r := NewInputReaderWithIO(in, out)

	input1, err := r.ReadInput()
	if err != nil {
		t.Fatalf("unexpected error on first read: %v", err)
	}
	if input1 != "first" {
		t.Errorf("expected 'first', got %q", input1)
	}

	input2, err := r.ReadInput()
	if err != nil {
		t.Fatalf("unexpected error on second read: %v", err)
	}
	if input2 != "second" {
		t.Errorf("expected 'second', got %q", input2)
	}
}
