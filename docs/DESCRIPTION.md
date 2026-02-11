# Stormtrooper: AI Coding Assistant CLI Tool

Stormtrooper is a sophisticated command-line AI coding assistant that helps developers interact with codebases through natural language conversations. It provides a terminal-based user interface (TUI) and a Read-Eval-Print Loop (REPL) for AI-powered code analysis, modification, and development tasks.

## Core Features

### 1. **AI Agent System**
- Uses LLM models (configurable) to understand and execute developer requests
- Supports both graphical TUI mode and traditional REPL mode (`-no-tui` flag)
- Maintains project context for intelligent code suggestions and modifications

### 2. **Powerful Tool Integration**
Stormtrooper includes built-in tools for:
- **File Operations**: Read, write, and edit files
- **Code Search**: Search using glob patterns and grep/regex
- **Shell Execution**: Run shell commands safely with permission checking
- **Memory System**: Persistent storage across sessions using `.stormtrooper/memory/`
- **Agent Spawning**: Create sub-agents for focused tasks

### 3. **Smart Project Context**
Automatically analyzes project structure and context to provide relevant assistance, including:
- Current working directory awareness
- Project-specific system prompts
- Integrated development environment understanding

### 4. **Safety & Permission System**
Implements a permission checker that ensures potentially destructive operations (like file writes and shell commands) are verified before execution.

## Usage Modes

### TUI Mode (Default)
Interactive terminal interface with:
- Real-time conversation view
- Context-aware suggestions
- Visual file browsing and code preview
- Markdown rendering and syntax highlighting

### REPL Mode
Simple command-line interface for:  
- Quick queries without graphics
- Headless environments
- Scripting and automation

## Technical Architecture

Built with Go, utilizing:
- **Bubble Tea** for terminal user interface
- **Chroma** for syntax highlighting
- **Glamour** for markdown rendering
- **lipgloss** for styling and theming
- Modular tool system with pluggable LLM backends

Version: 0.2.3
Author: gavinyap
Repository: github.com/gavinyap/stormtrooper