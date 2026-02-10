package tui

// AgentEvent is the interface for all events sent from the agent bridge
// to the Bubble Tea event loop. Each event type implements this with a
// marker method.
type AgentEvent interface {
	agentEvent()
}

// TokenMsg carries a streamed text token from the assistant.
type TokenMsg struct {
	Content string
}

// ToolStartMsg signals that a tool call has begun.
type ToolStartMsg struct {
	ID   string
	Name string
	Args string // truncated for display, max ~80 chars
}

// ToolResultMsg signals that a tool call has completed.
type ToolResultMsg struct {
	ID     string
	Name   string
	Result string // truncated for display
	Error  string // non-empty if the tool errored
}

// PermissionRequestMsg asks the user to approve/deny a tool execution.
// The agent goroutine blocks until a response is sent on the Response channel.
type PermissionRequestMsg struct {
	ID       string
	ToolName string
	Preview  string
	Response chan<- bool // send true=allow, false=deny
}

// PermissionResponseMsg is sent by the TUI after the user responds to a permission prompt.
type PermissionResponseMsg struct {
	Allowed bool
}

// AgentDoneMsg signals that the agent has finished processing the user's message.
type AgentDoneMsg struct {
	Error error
}

// SubAgentSpawnMsg signals that a sub-agent has been spawned.
type SubAgentSpawnMsg struct {
	Task string
}

// SubAgentDoneMsg signals that a sub-agent has completed.
type SubAgentDoneMsg struct{}

// agentEvent marker implementations.
func (TokenMsg) agentEvent()              {}
func (ToolStartMsg) agentEvent()          {}
func (ToolResultMsg) agentEvent()         {}
func (PermissionRequestMsg) agentEvent()  {}
func (PermissionResponseMsg) agentEvent() {}
func (AgentDoneMsg) agentEvent()          {}
func (SubAgentSpawnMsg) agentEvent()      {}
func (SubAgentDoneMsg) agentEvent()       {}
