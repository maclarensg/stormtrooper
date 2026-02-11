package llm

import (
	"strings"
	"testing"
)

func TestParseSSEStream_TextContent(t *testing.T) {
	input := `data: {"id":"1","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}

data: {"id":"1","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}

data: {"id":"1","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}

data: {"id":"1","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

data: [DONE]
`

	var chunks []ChatCompletionChunk
	err := ParseSSEStream(strings.NewReader(input), func(chunk ChatCompletionChunk) {
		chunks = append(chunks, chunk)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) != 4 {
		t.Fatalf("expected 4 chunks, got %d", len(chunks))
	}

	if chunks[0].Choices[0].Delta.Role != "assistant" {
		t.Errorf("expected role 'assistant', got %q", chunks[0].Choices[0].Delta.Role)
	}
	if chunks[1].Choices[0].Delta.Content != "Hello" {
		t.Errorf("expected 'Hello', got %q", chunks[1].Choices[0].Delta.Content)
	}
	if chunks[2].Choices[0].Delta.Content != " world" {
		t.Errorf("expected ' world', got %q", chunks[2].Choices[0].Delta.Content)
	}

	stop := "stop"
	if chunks[3].Choices[0].FinishReason == nil || *chunks[3].Choices[0].FinishReason != stop {
		t.Errorf("expected finish_reason 'stop'")
	}
}

func TestParseSSEStream_SkipsNonDataLines(t *testing.T) {
	input := `: this is a comment
event: message
data: {"id":"1","choices":[{"index":0,"delta":{"content":"ok"},"finish_reason":null}]}

data: [DONE]
`

	var chunks []ChatCompletionChunk
	err := ParseSSEStream(strings.NewReader(input), func(chunk ChatCompletionChunk) {
		chunks = append(chunks, chunk)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Choices[0].Delta.Content != "ok" {
		t.Errorf("expected 'ok', got %q", chunks[0].Choices[0].Delta.Content)
	}
}

func TestParseSSEStream_InvalidJSON(t *testing.T) {
	input := `data: {invalid json}
`

	err := ParseSSEStream(strings.NewReader(input), func(chunk ChatCompletionChunk) {})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseSSEStream_EmptyStream(t *testing.T) {
	err := ParseSSEStream(strings.NewReader(""), func(chunk ChatCompletionChunk) {
		t.Fatal("should not be called")
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeltaAccumulator_TextOnly(t *testing.T) {
	acc := NewDeltaAccumulator()

	acc.Add(ChatCompletionChunk{
		Choices: []ChunkChoice{{
			Index: 0,
			Delta: MessageDelta{Role: "assistant"},
		}},
	})
	acc.Add(ChatCompletionChunk{
		Choices: []ChunkChoice{{
			Index: 0,
			Delta: MessageDelta{Content: "Hello"},
		}},
	})
	acc.Add(ChatCompletionChunk{
		Choices: []ChunkChoice{{
			Index: 0,
			Delta: MessageDelta{Content: " world"},
		}},
	})

	msg := acc.Message()
	if msg.Role != "assistant" {
		t.Errorf("expected role 'assistant', got %q", msg.Role)
	}
	if msg.Content != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", msg.Content)
	}
	if len(msg.ToolCalls) != 0 {
		t.Errorf("expected no tool calls, got %d", len(msg.ToolCalls))
	}
}

func TestDeltaAccumulator_SingleToolCall(t *testing.T) {
	acc := NewDeltaAccumulator()

	acc.Add(ChatCompletionChunk{
		Choices: []ChunkChoice{{
			Index: 0,
			Delta: MessageDelta{
				Role: "assistant",
				ToolCalls: []ToolCallDelta{{
					Index: 0,
					ID:    "call_123",
					Type:  "function",
					Function: FunctionCall{
						Name:      "read_file",
						Arguments: `{"pat`,
					},
				}},
			},
		}},
	})
	acc.Add(ChatCompletionChunk{
		Choices: []ChunkChoice{{
			Index: 0,
			Delta: MessageDelta{
				ToolCalls: []ToolCallDelta{{
					Index: 0,
					Function: FunctionCall{
						Arguments: `h":"foo.txt"}`,
					},
				}},
			},
		}},
	})

	msg := acc.Message()
	if len(msg.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(msg.ToolCalls))
	}

	tc := msg.ToolCalls[0]
	if tc.ID != "call_123" {
		t.Errorf("expected ID 'call_123', got %q", tc.ID)
	}
	if tc.Function.Name != "read_file" {
		t.Errorf("expected name 'read_file', got %q", tc.Function.Name)
	}
	if tc.Function.Arguments != `{"path":"foo.txt"}` {
		t.Errorf("expected full arguments, got %q", tc.Function.Arguments)
	}
}

func TestDeltaAccumulator_ToolCallWithLeakedContent(t *testing.T) {
	acc := NewDeltaAccumulator()

	// Simulate an open-source model leaking tool call args as content.
	acc.Add(ChatCompletionChunk{
		Choices: []ChunkChoice{{
			Index: 0,
			Delta: MessageDelta{
				Role:    "assistant",
				Content: `<|tool_call_start|><|function|>read_file`,
			},
		}},
	})
	acc.Add(ChatCompletionChunk{
		Choices: []ChunkChoice{{
			Index: 0,
			Delta: MessageDelta{
				Content: `{"path":"main.go"}`,
				ToolCalls: []ToolCallDelta{{
					Index: 0,
					ID:    "call_1",
					Type:  "function",
					Function: FunctionCall{
						Name:      "read_file",
						Arguments: `{"path":"main.go"}`,
					},
				}},
			},
		}},
	})
	acc.Add(ChatCompletionChunk{
		Choices: []ChunkChoice{{
			Index: 0,
			Delta: MessageDelta{
				Content: `<|tool_call_end|>`,
			},
		}},
	})

	msg := acc.Message()
	if len(msg.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(msg.ToolCalls))
	}
	// Content should be cleaned — special tokens and JSON removed.
	if strings.Contains(msg.Content, "<|tool_call") {
		t.Errorf("expected special tokens to be stripped from content, got %q", msg.Content)
	}
	if strings.Contains(msg.Content, "read_file") {
		t.Errorf("expected tool-call-like content to be removed, got %q", msg.Content)
	}
}

func TestDeltaAccumulator_ToolCallContentIsJustJSON(t *testing.T) {
	acc := NewDeltaAccumulator()

	// Model sends the tool call arguments as content (JSON blob).
	acc.Add(ChatCompletionChunk{
		Choices: []ChunkChoice{{
			Index: 0,
			Delta: MessageDelta{
				Role:    "assistant",
				Content: `{"path":"src/main.go","content":"hello world"}`,
				ToolCalls: []ToolCallDelta{{
					Index: 0,
					ID:    "call_1",
					Type:  "function",
					Function: FunctionCall{
						Name:      "write_file",
						Arguments: `{"path":"src/main.go","content":"hello world"}`,
					},
				}},
			},
		}},
	})

	msg := acc.Message()
	if msg.Content != "" {
		t.Errorf("expected empty content when tool calls present and content is JSON, got %q", msg.Content)
	}
}

func TestCleanToolCallContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "special tokens stripped",
			input:    "<|tool_call_start|><|function|>read_file<|tool_call_end|>",
			expected: "read_file",
		},
		{
			name:     "json blob removed",
			input:    `{"path":"main.go","content":"data"}`,
			expected: "",
		},
		{
			name:     "mixed special tokens and json",
			input:    `<|tool_call_start|>{"path":"main.go"}<|tool_call_end|>`,
			expected: "",
		},
		{
			name:     "empty after stripping",
			input:    "<|tool_call_end|><|im_end|>",
			expected: "",
		},
		{
			name:     "real text preserved",
			input:    "Here is the file content you requested.",
			expected: "Here is the file content you requested.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanToolCallContent(tt.input)
			if got != tt.expected {
				t.Errorf("cleanToolCallContent(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestDeltaAccumulator_MultipleToolCalls(t *testing.T) {
	acc := NewDeltaAccumulator()

	// First tool call starts.
	acc.Add(ChatCompletionChunk{
		Choices: []ChunkChoice{{
			Index: 0,
			Delta: MessageDelta{
				Role: "assistant",
				ToolCalls: []ToolCallDelta{{
					Index: 0,
					ID:    "call_1",
					Type:  "function",
					Function: FunctionCall{
						Name:      "read_file",
						Arguments: `{"path":"a.txt"}`,
					},
				}},
			},
		}},
	})

	// Second tool call starts.
	acc.Add(ChatCompletionChunk{
		Choices: []ChunkChoice{{
			Index: 0,
			Delta: MessageDelta{
				ToolCalls: []ToolCallDelta{{
					Index: 1,
					ID:    "call_2",
					Type:  "function",
					Function: FunctionCall{
						Name:      "read_file",
						Arguments: `{"path":"b.txt"}`,
					},
				}},
			},
		}},
	})

	msg := acc.Message()
	if len(msg.ToolCalls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(msg.ToolCalls))
	}

	if msg.ToolCalls[0].ID != "call_1" {
		t.Errorf("first tool call ID: expected 'call_1', got %q", msg.ToolCalls[0].ID)
	}
	if msg.ToolCalls[1].ID != "call_2" {
		t.Errorf("second tool call ID: expected 'call_2', got %q", msg.ToolCalls[1].ID)
	}
}
