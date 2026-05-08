package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/markkovari/nestor/internal/adapters/llm"
	"github.com/markkovari/nestor/internal/adapters/source"
	"github.com/markkovari/nestor/internal/adapters/task"
	"github.com/markkovari/nestor/internal/config"
	"github.com/markkovari/nestor/internal/core"
	"github.com/markkovari/nestor/internal/db"
)

func initializeEngine(ctx context.Context, cfg *config.Config) (*core.Engine, *db.Database, error) {
	var database *db.Database
	var err error
	if cfg.Database.URL != "" {
		database, err = db.NewDatabase(ctx, cfg.Database)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: database unavailable (%v). Task caching and dependency persistence disabled.\n", err)
		}
	}

	var llmProvider core.LLMProvider
	switch cfg.LLM.Provider {
	case "mock", "":
		llmProvider = &llm.MockLLM{}
	default:
		llmProvider, err = llm.NewProvider(ctx, cfg.LLM.Provider, cfg.LLM.APIKey, cfg.LLM.Model, cfg.LLM.BaseURL)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to init llm provider %q: %w", cfg.LLM.Provider, err)
		}
	}

	var taskProviders []core.TaskProvider
	if cfg.Adapters.GitHub.Token != "" {
		taskProviders = append(taskProviders, task.NewGitHubIssueProvider(cfg.Adapters.GitHub.Token, cfg.Adapters.GitHub.Repos))
	}
	if cfg.Adapters.Linear.APIKey != "" {
		taskProviders = append(taskProviders, task.NewLinearProvider(cfg.Adapters.Linear.APIKey))
	}
	if cfg.Adapters.Jira.Domain != "" && cfg.Adapters.Jira.Token != "" {
		taskProviders = append(taskProviders, task.NewJiraProvider(cfg.Adapters.Jira.Domain, cfg.Adapters.Jira.User, cfg.Adapters.Jira.Token))
	}

	// Fallback to mock if no providers configured
	if len(taskProviders) == 0 {
		taskProviders = append(taskProviders, &task.MockTaskProvider{})
	}

	engine := core.NewEngine(database, llmProvider, taskProviders...)
	engine.CacheTTL = cfg.CacheTTL
	if engine.CacheTTL == 0 {
		engine.CacheTTL = 60
	}

	if cfg.Adapters.GitHub.Token != "" {
		sourceProvider := source.NewGitHubProvider(ctx, cfg.Adapters.GitHub.Token, cfg.Adapters.GitHub.Repos)
		engine.WithSourceProviders(sourceProvider)
	}

	return engine, database, nil
}
