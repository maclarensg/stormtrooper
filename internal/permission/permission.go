// Package permission provides permission checking and user prompts
// for tool execution.
package permission

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Handler is the interface for permission checking.
// The REPL uses Checker (reads stdin), the TUI uses its own implementation.
type Handler interface {
	Check(toolName string, preview string) bool
}

// Checker handles permission prompts for tool execution.
// It implements the Handler interface.
type Checker struct {
	in  io.Reader
	out io.Writer
}

// NewChecker creates a Checker that reads from stdin and writes to stderr.
func NewChecker() *Checker {
	return &Checker{
		in:  os.Stdin,
		out: os.Stderr,
	}
}

// NewCheckerWithIO creates a Checker with custom I/O for testing.
func NewCheckerWithIO(in io.Reader, out io.Writer) *Checker {
	return &Checker{in: in, out: out}
}

// Check prompts the user for approval and returns true if approved.
// toolName is the name of the tool requesting permission.
// preview is a description of what the tool will do.
func (c *Checker) Check(toolName string, preview string) bool {
	fmt.Fprintf(c.out, "\n[permission] %s\n%s\n[y/n]: ", toolName, preview)

	scanner := bufio.NewScanner(c.in)
	if !scanner.Scan() {
		return false
	}
	line := strings.TrimSpace(scanner.Text())
	return len(line) > 0 && (line[0] == 'y' || line[0] == 'Y')
}
