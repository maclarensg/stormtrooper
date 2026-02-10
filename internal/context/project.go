// Package context loads project context (STORMTROOPER.md, CLAUDE.md)
// and environment info for the system prompt.
package context

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gavinyap/stormtrooper/internal/memory"
)

// ProjectContext holds information about the current project environment.
type ProjectContext struct {
	WorkingDir   string
	Instructions string // Contents of STORMTROOPER.md or CLAUDE.md
	Memory       string // Contents of MEMORY.md
	Platform     string // runtime.GOOS
	Date         string // current date YYYY-MM-DD
}

// instructionFiles lists project instruction files in priority order.
var instructionFiles = []string{
	"STORMTROOPER.md",
	"CLAUDE.md",
}

// Load reads project context from the given directory.
func Load(dir string) (*ProjectContext, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolve directory: %w", err)
	}

	pc := &ProjectContext{
		WorkingDir: absDir,
		Platform:   runtime.GOOS,
		Date:       time.Now().Format("2006-01-02"),
	}

	// Load project instructions
	for _, name := range instructionFiles {
		path := filepath.Join(absDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		pc.Instructions = string(data)
		break
	}

	// Load memory
	mem, err := memory.Load(absDir)
	if err != nil {
		return nil, fmt.Errorf("load memory: %w", err)
	}
	pc.Memory = mem

	return pc, nil
}

// BuildSystemPrompt constructs the full system prompt from the project context.
func (pc *ProjectContext) BuildSystemPrompt() string {
	var b strings.Builder

	b.WriteString("You are Stormtrooper, an AI coding assistant. You help developers by reading, editing, and searching code, running commands, and managing project context. Use the available tools to interact with the codebase.")

	if pc.Instructions != "" {
		b.WriteString("\n\n# Project Instructions\n\n")
		b.WriteString(pc.Instructions)
	}

	if pc.Memory != "" {
		b.WriteString("\n\n# Memory\n\n")
		b.WriteString(pc.Memory)
	}

	b.WriteString("\n\n# Environment\n")
	b.WriteString(fmt.Sprintf("- Working directory: %s\n", pc.WorkingDir))
	b.WriteString(fmt.Sprintf("- Platform: %s\n", pc.Platform))
	b.WriteString(fmt.Sprintf("- Date: %s\n", pc.Date))

	return b.String()
}
