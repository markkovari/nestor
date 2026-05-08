package core

import (
	"context"
	"fmt"
	"reflect"
	"time"
)

type Engine struct {
	TaskProviders []TaskProvider
	LLM           LLMProvider
	DB            DataStore
	CacheTTL      int  // in minutes
	FetchOnly     bool // bypass cache if true
}

func isNil(i interface{}) bool {
	if i == nil {
		return true
	}
	v := reflect.ValueOf(i)
	return v.Kind() == reflect.Ptr && v.IsNil()
}

func NewEngine(database DataStore, llm LLMProvider, tasks ...TaskProvider) *Engine {
	return &Engine{
		DB:            database,
		TaskProviders: tasks,
		LLM:           llm,
	}
}

func (e *Engine) RunAnalysis(ctx context.Context) error {
	allTasks := []Task{}

	for _, p := range e.TaskProviders {
		var providerTasks []Task
		useCache := false

		// Try cache first if not explicitly bypassing
		if !e.FetchOnly && e.DB != nil && !isNil(e.DB) {
			cached, err := e.DB.FetchTasksByProvider(ctx, p.Name())
			if err == nil && len(cached) > 0 {
				// Check TTL
				latest := cached[0].CachedAt
				for _, t := range cached {
					if t.CachedAt > latest {
						latest = t.CachedAt
					}
				}

				cachedTime, parseErr := time.Parse(time.RFC3339, latest)
				if parseErr == nil {
					if time.Since(cachedTime) < time.Duration(e.CacheTTL)*time.Minute {
						fmt.Printf("Using cached tasks for provider: %s\n", p.Name())
						providerTasks = cached
						useCache = true
					}
				}
			}
		}

		if !useCache {
			fmt.Printf("Fetching fresh tasks for provider: %s\n", p.Name())
			var err error
			providerTasks, err = p.FetchTasks(ctx)
			if err != nil {
				return fmt.Errorf("failed to fetch tasks from provider %s: %w", p.Name(), err)
			}

			// Update cache
			if e.DB != nil && !isNil(e.DB) {
				for _, t := range providerTasks {
					if err := e.DB.SaveTask(ctx, t); err != nil {
						fmt.Printf("Warning: failed to persist task %s: %v\n", t.ID, err)
					}
				}
			}
		}

		allTasks = append(allTasks, providerTasks...)
	}

	fmt.Printf("Analyzing total of %d tasks...\n", len(allTasks))

	// 1. Generate DAG
	dag, err := e.LLM.GenerateDAG(ctx, allTasks)
	if err != nil {
		return fmt.Errorf("failed to generate DAG: %w", err)
	}

	fmt.Println("Calculated Dependencies:")
	for taskID, blockers := range dag {
		fmt.Printf("  %s depends on %v\n", taskID, blockers)
		// Persist Relationships
		if e.DB != nil && !isNil(e.DB) {
			for _, blocker := range blockers {
				if err := e.DB.CreateDependency(ctx, blocker, taskID); err != nil {
					fmt.Printf("Warning: failed to persist dependency %s -> %s: %v\n", blocker, taskID, err)
				}
			}
		}
	}

	// 2. Analyze Conflicts
	conflictReport, err := e.LLM.AnalyzeConflict(ctx, allTasks, []string{"ADR-001", "ADR-002"})
	if err != nil {
		return fmt.Errorf("failed to analyze conflicts: %w", err)
	}
	fmt.Printf("Conflict Analysis: %s\n", conflictReport)

	return nil
}
