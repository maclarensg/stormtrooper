# Stormtrooper 🦾

An AI-powered command-line coding assistant that helps you navigate, understand, and modify codebases through natural language conversations.

![Stormtrooper Screenshot](https://user-images.githubusercontent.com/screenshot-placeholder.jpg)

## Features

### 🤖 AI-Powered Development
- **Intelligent Code Analysis**: Understands your codebase structure and provides contextual assistance
- **Natural Language Interface**: Ask questions and give instructions in plain English
- **Multi-Model Support**: Works with various LLM providers (OpenRouter, custom endpoints)

### 🛠️ Comprehensive Tool Suite
- **File Operations**: Read, write, and edit files with content-aware assistance
- **Code Search**: Advanced search using glob patterns and regex
- **Shell Integration**: Safely execute commands with permission verification
- **Memory System**: Persistent storage for context across sessions
- **Agent Spawning**: Create specialized sub-agents for complex tasks

### 🖥️ Dual Interface Modes
- **TUI Mode**: Beautiful terminal interface with syntax highlighting and markdown support (default)
- **REPL Mode**: Simple command-line interface for quick queries (`-no-tui` flag)

### 🛡️ Safety First
- **Permission System**: Get explicit approval for potentially destructive operations
- **Sandboxes Environment**: Safely test and execute code changes
- **Undo Support**: Rollback unwanted changes

## Quick Start

### Installation

#### Via Go Install
```bash
go install github.com/gavinyap/stormtrooper@latest
```

#### Binary Releases
Download the latest binary from [Releases](https://github.com/gavinyap/stormtrooper/releases):
- Linux: `stormtrooper-linux-amd64`
- macOS: `stormtrooper-darwin-amd64` (Apple Silicon and Intel)
- Windows: `stormtrooper-windows-amd64.exe`

### Initial Setup

1. **Get an API Key**: Sign up at [OpenRouter](https://openrouter.ai/) or your preferred provider
2. **Configure Stormtrooper**:
   ```bash
   mkdir -p ~/.stormtrooper
   cat > ~/.stormtrooper/config.yaml << EOF
   api_key: "your-openrouter-api-key"
   model: "moonshotai/kimi-k2"
   base_url: "https://openrouter.ai/api/v1"
   EOF
   ```

3. **Run Stormtrooper**:
   ```bash
   cd /your/project/directory
   stormtrooper
   ```

## Usage

### Basic Commands
```bash
# Start in TUI mode (default)
stormtrooper

# Start in REPL mode
stormtrooper -no-tui

# Use a specific model
stormtrooper -model "openai/gpt-4o"
```

### Example Conversations

#### **Code Understanding**
```
> What does this project do?
Stormtrooper: This appears to be a web API built with FastAPI. 
The main functionality is in `main.py` which sets up a REST API 
for managing user accounts...
```

#### **Code Generation**
```
> Create a new endpoint to handle user login
Stormtrooper: I'll create a new `/auth/login` endpoint for you.
📁 Creating: src/auth/login.py
Would you like me to:
✅ Create the new file
❌ Cancel operation
```

#### **Debugging**
```
> Why is my test failing?
Stormtrooper: Looking at the error message and test files...
- test_account.py:42 - AssertionError: expected 200 but got 401
- The issue is in your auth middleware: `middleware/auth.py:15`
- Missing header validation for the Authorization token
```

### Memory System
Stormtrooper creates a memory directory (`.stormtrooper/memory/`) to store:
- Conversation history
- Project conventions
- Custom snippets and templates
- Persistent notes and documentation

Access memory commands:
```
> remember that we use snake_case for Python files
✓ Stored in memory: coding_styles.python_conventions

> recall our testing strategy
From memory/testing_strategy.md:
"We use pytest with fixtures in tests/conftest.py..."
```

## Configuration

### File Locations
Stormtrooper uses a layered configuration system:
1. **Defaults**: Built-in fallbacks
2. **Global Config**: `~/.stormtrooper/config.yaml` 
3. **Project Config**: `./.stormtrooper/config.yaml`
4. **Environment Variables**: `OPENROUTER_API_KEY`
5. **CLI Flags**: `-model`, `-no-tui`

### Configuration Options
```yaml
# ~/.stormtrooper/config.yaml
api_key: "your-api-key"          # Required: LLM provider API key
model: "moonshotai/kimi-k2"     # Default model (can be overridden)
base_url: "https://openrouter.ai/api/v1"  # Custom endpoint (optional)
```

### Environment Variables
```bash
export OPENROUTER_API_KEY="your-api-key"
stormtrooper
```

## Advanced Features

### Custom Models
Use any OpenAI-compatible API endpoint:
```yaml
# ~/.stormtrooper/config.yaml
base_url: "https://api.anotherprovider.com/v1"
model: "custom-model-name"
api_key: "your-custom-api-key"
```

### Context-Aware Assistance
Stormtrooper automatically builds context about your project:
- Analyzes directory structure
- Reads configuration files (`.gitignore`, package.json, etc.)
- Scans dependencies and tech stack
- Maintains project-specific prompts

### Agent Spawning
For complex tasks, Stormtrooper can spawn specialized agents:
```
> Create a sub-agent to refactor the authentication system
✓ Spawned agent "auth-refactor" with focus on authentication
auth-refactor> Analyzing current auth patterns...
```

## Safety & Permissions

Stormtrooper implements a comprehensive safety system:

### Permission Categories
- **File Write**: Creating/modifying files
- **System Commands**: Executing shell commands
- **Dangerous Operations**: Potentially destructive commands
- **Network Access**: External API calls

### Permission Flow
1. **Intent Detection**: Identifies risky operations
2. **User Verification**: Asks for explicit approval
3. **Dry Run**: Shows what will be executed
4. **Execution**: Only proceeds upon confirmation

## Development

### Building from Source
```bash
# Clone the repository
git clone https://github.com/gavinyap/stormtrooper.git
cd stormtrooper

# Install dependencies
go mod download

# Build
make build  # or: go build -o stormtrooper cmd/stormtrooper/main.go

# Run development version
./stormtrooper
```

### Architecture Overview
```
cmd/stormtrooper/main.go          # CLI entry point
internal/
├── agent/                       # AI agent implementation
├── tool/                       # Tool registry and implementations
├── tui/                         # Terminal UI (Bubble Tea)
├── repl/                       # Read-Eval-Print Loop
├── memory/                     # Persistent storage system
├── permission/                 # Safety and permission checking
└── context/                    # Project context management
```

## Troubleshooting

### Common Issues

**"API key not found"**
```bash
# Set environment variable
export OPENROUTER_API_KEY="your-key"

# Or check configuration file
ls -la ~/.stormtrooper/config.yaml
```

**"Terminal display issues"**
```bash
# Force ANSI colors
export TERM=xterm-256color

# Try REPL mode instead
stormtrooper -no-tui
```

**"Memory directory issues"**
```bash
# Check permissions
ls -la .stormtrooper/
chmod 755 .stormtrooper/
```

### Debug Mode
Enable verbose logging:
```bash
stormtrooper -debug  # (if implemented)
```

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development Setup
```bash
# Install development dependencies
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run tests
go test ./...

# Lint code
golangci-lint run
```

## Changelog

See [CHANGELOG.md](docs/CHANGELOG.md) for detailed release notes.

## License

This project is licensed under the [MIT License](LICENSE).

---

**Made with ❤️ by [gavinyap](https://github.com/gavinyap) and contributors**

**Stars are appreciated!** ⭐ If you find Stormtrooper helpful, consider starring the repository.