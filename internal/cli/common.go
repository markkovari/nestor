package cli

import (
	"context"
	"fmt"

	"github.com/markkovari/nestor/internal/adapters/llm"
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
			fmt.Printf("Warning: failed to connect to database: %v. Continuing without persistence.\n", err)
		}
	}

	var llmProvider core.LLMProvider
	switch cfg.LLM.Provider {
	case "gemini":
		llmProvider, err = llm.NewGeminiProvider(ctx, cfg.LLM.APIKey, cfg.LLM.Model)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to init gemini: %w", err)
		}
	default:
		llmProvider = &llm.MockLLM{}
	}

	var taskProviders []core.TaskProvider
	if cfg.Adapters.GitHub.Token != "" {
		taskProviders = append(taskProviders, task.NewGitHubIssueProvider(cfg.Adapters.GitHub.Token, cfg.Adapters.GitHub.Repos))
	}
	if cfg.Adapters.Linear.APIKey != "" {
		taskProviders = append(taskProviders, task.NewLinearProvider(cfg.Adapters.Linear.APIKey))
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

	return engine, database, nil
}
