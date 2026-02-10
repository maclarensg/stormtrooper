// stream.go implements SSE (Server-Sent Events) stream parsing
// for streamed chat completion responses.
package llm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// StreamCallback is called for each parsed chunk from the SSE stream.
type StreamCallback func(chunk ChatCompletionChunk)

// ParseSSEStream reads an SSE stream from reader and calls callback for each
// data chunk. It returns when the stream ends (data: [DONE]) or an error occurs.
func ParseSSEStream(reader io.Reader, callback StreamCallback) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		if data == "[DONE]" {
			return nil
		}

		var chunk ChatCompletionChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return fmt.Errorf("failed to parse SSE chunk: %w", err)
		}

		callback(chunk)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("SSE stream read error: %w", err)
	}

	return nil
}

// DeltaAccumulator collects streaming deltas into a final Message.
type DeltaAccumulator struct {
	role      string
	content   strings.Builder
	toolCalls map[int]*ToolCall // keyed by tool call index from deltas
}

// NewDeltaAccumulator creates a new accumulator.
func NewDeltaAccumulator() *DeltaAccumulator {
	return &DeltaAccumulator{
		toolCalls: make(map[int]*ToolCall),
	}
}

// Add processes a single streaming chunk delta.
func (a *DeltaAccumulator) Add(chunk ChatCompletionChunk) {
	for _, choice := range chunk.Choices {
		d := choice.Delta

		if d.Role != "" {
			a.role = d.Role
		}
		if d.Content != "" {
			a.content.WriteString(d.Content)
		}

		for _, tcd := range d.ToolCalls {
			existing, ok := a.toolCalls[tcd.Index]
			if !ok {
				tc := ToolCall{
					ID:   tcd.ID,
					Type: tcd.Type,
					Function: FunctionCall{
						Name:      tcd.Function.Name,
						Arguments: tcd.Function.Arguments,
					},
				}
				a.toolCalls[tcd.Index] = &tc
			} else {
				if tcd.ID != "" {
					existing.ID = tcd.ID
				}
				if tcd.Type != "" {
					existing.Type = tcd.Type
				}
				if tcd.Function.Name != "" {
					existing.Function.Name += tcd.Function.Name
				}
				existing.Function.Arguments += tcd.Function.Arguments
			}
		}
	}
}

// Message returns the accumulated complete Message.
func (a *DeltaAccumulator) Message() Message {
	msg := Message{
		Role:    a.role,
		Content: a.content.String(),
	}

	if len(a.toolCalls) > 0 {
		maxIdx := 0
		for idx := range a.toolCalls {
			if idx > maxIdx {
				maxIdx = idx
			}
		}
		for i := 0; i <= maxIdx; i++ {
			if tc, ok := a.toolCalls[i]; ok {
				msg.ToolCalls = append(msg.ToolCalls, *tc)
			}
		}
	}

	return msg
}
