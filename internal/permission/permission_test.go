package permission

import (
	"bytes"
	"strings"
	"testing"
)

func TestCheckApproved(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"y\n", true},
		{"Y\n", true},
		{"yes\n", true},
		{"Yes\n", true},
		{"n\n", false},
		{"N\n", false},
		{"no\n", false},
		{"\n", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			in := strings.NewReader(tt.input)
			out := &bytes.Buffer{}
			c := NewCheckerWithIO(in, out)

			got := c.Check("test_tool", "preview text")
			if got != tt.want {
				t.Errorf("Check() with input %q = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestCheckOutput(t *testing.T) {
	in := strings.NewReader("n\n")
	out := &bytes.Buffer{}
	c := NewCheckerWithIO(in, out)

	c.Check("shell_exec", "Run command: ls -la")

	output := out.String()
	if !strings.Contains(output, "shell_exec") {
		t.Error("output should contain tool name")
	}
	if !strings.Contains(output, "Run command: ls -la") {
		t.Error("output should contain preview text")
	}
	if !strings.Contains(output, "[y/n]") {
		t.Error("output should contain [y/n] prompt")
	}
}

func TestNewChecker(t *testing.T) {
	c := NewChecker()
	if c == nil {
		t.Fatal("NewChecker returned nil")
	}
}
