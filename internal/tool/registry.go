package tool

import "fmt"

// Registry holds all registered tools and dispatches calls by name.
type Registry struct {
	tools map[string]Tool
	order []string // preserves registration order
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry. Panics if a tool with the same name
// is already registered.
func (r *Registry) Register(t Tool) {
	name := t.Name()
	if _, exists := r.tools[name]; exists {
		panic(fmt.Sprintf("tool already registered: %s", name))
	}
	r.tools[name] = t
	r.order = append(r.order, name)
}

// Get returns the tool with the given name, or nil if not found.
func (r *Registry) Get(name string) Tool {
	return r.tools[name]
}

// Definitions returns all registered tools in OpenAI function calling format,
// preserving registration order.
func (r *Registry) Definitions() []ToolDef {
	defs := make([]ToolDef, 0, len(r.order))
	for _, name := range r.order {
		t := r.tools[name]
		defs = append(defs, ToolDef{
			Type: "function",
			Function: FunctionDef{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.Schema(),
			},
		})
	}
	return defs
}
