// input.go handles multi-line input detection and reading from the terminal.
package repl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	primaryPrompt      = "> "
	continuationPrompt = ". "
)

// InputReader reads user input with multi-line support.
type InputReader struct {
	scanner *bufio.Scanner
	out     io.Writer
}

// NewInputReader creates an InputReader that reads from stdin
// and prints prompts to stderr.
func NewInputReader() *InputReader {
	return &InputReader{
		scanner: bufio.NewScanner(os.Stdin),
		out:     os.Stderr,
	}
}

// NewInputReaderWithIO creates an InputReader with custom I/O for testing.
func NewInputReaderWithIO(in io.Reader, out io.Writer) *InputReader {
	return &InputReader{
		scanner: bufio.NewScanner(in),
		out:     out,
	}
}

// ReadInput reads user input, supporting multi-line input via backslash
// continuation. Returns io.EOF if the input stream is closed.
func (r *InputReader) ReadInput() (string, error) {
	fmt.Fprint(r.out, primaryPrompt)

	var lines []string
	first := true

	for {
		if !first {
			fmt.Fprint(r.out, continuationPrompt)
		}
		first = false

		if !r.scanner.Scan() {
			if err := r.scanner.Err(); err != nil {
				return "", err
			}
			return "", io.EOF
		}

		line := r.scanner.Text()

		if strings.HasSuffix(line, "\\") {
			// Strip trailing backslash and continue reading.
			lines = append(lines, strings.TrimSuffix(line, "\\"))
			continue
		}

		lines = append(lines, line)
		break
	}

	return strings.Join(lines, "\n"), nil
}
