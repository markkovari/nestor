package core

import (
	"context"
	"fmt"
	"reflect"
)

type Engine struct {
	TaskProviders []TaskProvider
	LLM           LLMProvider
	DB            DataStore
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
		tasks, err := p.FetchTasks(ctx)
		if err != nil {
			return fmt.Errorf("failed to fetch tasks from provider: %w", err)
		}
		allTasks = append(allTasks, tasks...)
	}

	fmt.Printf("Analyzing and Ingesting %d tasks...\n", len(allTasks))

	// Ingest Tasks into DB
	if e.DB != nil && !isNil(e.DB) {
		for _, t := range allTasks {
			if err := e.DB.SaveTask(ctx, t); err != nil {
				fmt.Printf("Warning: failed to persist task %s: %v\n", t.ID, err)
			}
		}
	}

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
