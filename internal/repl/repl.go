// Package repl provides the terminal REPL loop that reads user input,
// sends it to the agent, and displays streamed output.
package repl

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/gavinyap/stormtrooper/internal/agent"
)

// REPL manages the read-eval-print loop.
type REPL struct {
	agent   *agent.Agent
	input   *InputReader
	out     io.Writer
	version string
}

// New creates a new REPL with the given agent and version string.
func New(ag *agent.Agent, version string) *REPL {
	return &REPL{
		agent:   ag,
		input:   NewInputReader(),
		out:     os.Stderr,
		version: version,
	}
}

// NewWithIO creates a REPL with custom I/O for testing.
func NewWithIO(ag *agent.Agent, version string, input *InputReader, out io.Writer) *REPL {
	return &REPL{
		agent:   ag,
		input:   input,
		out:     out,
		version: version,
	}
}

// Run starts the REPL loop. Blocks until the user exits or input is closed.
func (r *REPL) Run(ctx context.Context) error {
	fmt.Fprintf(r.out, "Stormtrooper v%s — AI coding assistant\n", r.version)
	fmt.Fprintln(r.out, "Type /exit or Ctrl+C to quit.")
	fmt.Fprintln(r.out)

	for {
		// Check if context is cancelled (e.g., Ctrl+C).
		if ctx.Err() != nil {
			break
		}

		input, err := r.input.ReadInput()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(r.out, "Input error: %v\n", err)
			continue
		}

		if input == "" {
			continue
		}

		if input == "/exit" {
			break
		}

		if err := r.agent.Send(ctx, input); err != nil {
			if ctx.Err() != nil {
				break // Context cancelled (Ctrl+C), exit REPL
			}
			fmt.Fprintf(r.out, "Error: %v\n", err)
			continue
		}

		fmt.Fprintln(r.out)
	}

	fmt.Fprintln(r.out, "Goodbye!")
	return nil
}
