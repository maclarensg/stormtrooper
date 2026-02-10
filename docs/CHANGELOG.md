# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

## [0.2.0] - 2026-02-10

### Added
- Dashboard-style terminal UI powered by Bubble Tea.
- Chat panel with scrollable history and syntax-highlighted code blocks.
- Sidebar showing tool activity, agent status, and project info in real-time.
- Styled input bar with multi-line support.
- Status bar displaying current model, project name, and working directory.
- Spinner indicator while AI is thinking.
- Inline permission prompts within the TUI.
- `--no-tui` flag to fall back to the simple conversational CLI.

## [0.1.1] - 2026-02-10

### Fixed
- Ctrl+C now exits the application cleanly instead of getting stuck in an error loop.
- Double Ctrl+C force-exits immediately.

## [0.1.0] - 2026-02-10

### Added
- Interactive conversational CLI for AI-assisted coding.
- OpenRouter integration with support for Kimi-K2, MiniMax-M2.1, and GLM-4.7.
- Tool system: file reading/editing, shell execution, code search.
- Permission system for approving AI actions.
- Persistent memory across sessions.
- Project context awareness.
- Agent teams for parallel work.
- Configuration system (global and per-project).
