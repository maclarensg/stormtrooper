// Entry point for the stormtrooper CLI.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gavinyap/stormtrooper/internal/agent"
	"github.com/gavinyap/stormtrooper/internal/config"
	projectctx "github.com/gavinyap/stormtrooper/internal/context"
	"github.com/gavinyap/stormtrooper/internal/llm"
	"github.com/gavinyap/stormtrooper/internal/memory"
	"github.com/gavinyap/stormtrooper/internal/permission"
	"github.com/gavinyap/stormtrooper/internal/repl"
	"github.com/gavinyap/stormtrooper/internal/tool"
	"github.com/gavinyap/stormtrooper/internal/tui"

	gocontext "context"
)

func main() {
	model := flag.String("model", "", "LLM model to use (overrides config)")
	noTUI := flag.Bool("no-tui", false, "Use plain REPL instead of TUI")
	flag.Parse()

	// Load config.
	cfg, err := config.Load(*model)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Create LLM client.
	client := llm.NewClient(cfg.APIKey)
	if cfg.BaseURL != "" {
		client.SetBaseURL(cfg.BaseURL)
	}

	// Create tool registry and register all tools.
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not determine working directory: %v\n", err)
		os.Exit(1)
	}

	registry := tool.NewRegistry()
	registry.Register(&tool.ReadFileTool{})
	registry.Register(&tool.WriteFileTool{})
	registry.Register(&tool.EditFileTool{})
	registry.Register(&tool.ShellExecTool{})
	registry.Register(&tool.GlobTool{})
	registry.Register(&tool.GrepTool{})
	registry.Register(&tool.MemoryWriteTool{MemoryDir: memory.Dir(cwd)})

	// Load project context and build system prompt.
	projCtx, err := projectctx.Load(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load project context: %v\n", err)
		projCtx = &projectctx.ProjectContext{WorkingDir: cwd}
	}
	systemPrompt := projCtx.BuildSystemPrompt()

	// Create permission checker.
	perm := permission.NewChecker()

	// Register spawn_agent tool (needs client, registry, and permission checker).
	registry.Register(agent.NewSpawnAgentTool(client, registry, perm, cfg.Model))

	// Create root agent.
	rootAgent := agent.New(agent.Options{
		Client:       client,
		Registry:     registry,
		Permission:   perm,
		Model:        cfg.Model,
		SystemPrompt: systemPrompt,
	})

	if *noTUI {
		// REPL mode — existing behavior unchanged.
		ctx, cancel := gocontext.WithCancel(gocontext.Background())
		defer cancel()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh // First signal: graceful shutdown
			cancel()
			<-sigCh // Second signal: force exit
			os.Exit(1)
		}()

		r := repl.New(rootAgent)
		if err := r.Run(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		// TUI mode — Bubble Tea handles signals via tea.KeyMsg.
		app := tui.New(tui.Options{
			Agent:      rootAgent,
			Config:     cfg,
			ProjectCtx: projCtx,
			Version:    "0.2.0",
		})
		p := tea.NewProgram(app, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}
