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
// When tool calls are present, content that looks like leaked tool call
// arguments (JSON blobs, special tokens) is stripped from the message.
func (a *DeltaAccumulator) Message() Message {
	content := a.content.String()

	msg := Message{
		Role: a.role,
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

		// When tool calls are present, open-source models sometimes leak
		// tool call arguments as regular content. Strip that out.
		content = cleanToolCallContent(content)
	}

	msg.Content = content
	return msg
}

// cleanToolCallContent removes leaked tool call artifacts from content
// when the response includes tool calls. It strips known special tokens
// and detects JSON blobs that look like tool call arguments.
func cleanToolCallContent(s string) string {
	// Strip known special tokens.
	for _, tok := range knownSpecialTokens {
		s = strings.ReplaceAll(s, tok, "")
	}

	s = strings.TrimSpace(s)

	// If the remaining content looks like tool call JSON, discard it entirely.
	if looksLikeToolCallJSON(s) {
		return ""
	}

	return s
}

// knownSpecialTokens are model-specific tokens that should never appear in
// user-visible content.
var knownSpecialTokens = []string{
	"<|tool_call_end|>",
	"<|tool_call_start|>",
	"<|function|>",
	"<|tool_sep|>",
	"<|im_end|>",
}

// looksLikeToolCallJSON returns true if the string appears to be a tool call
// argument leak. This covers:
//   - Pure JSON objects: {"path":"main.go"}
//   - Function name + JSON: read_file{"path":"main.go"}
func looksLikeToolCallJSON(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return false
	}
	// Pure JSON object.
	if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
		return true
	}
	// Function name followed by JSON arguments (e.g., read_file{"path":"..."}).
	if idx := strings.Index(s, "{"); idx > 0 && strings.HasSuffix(s, "}") {
		return true
	}
	return false
}
