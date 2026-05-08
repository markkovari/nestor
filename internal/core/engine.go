package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

type Engine struct {
	TaskProviders []TaskProvider
	LLM           LLMProvider
	DB            DataStore
	CacheTTL      int  // in minutes
	FetchOnly     bool // bypass cache if true
	ADRDir        string
}

func isNil(i any) bool {
	if i == nil {
		return true
	}
	v := reflect.ValueOf(i)
	return v.Kind() == reflect.Pointer && v.IsNil()
}

func NewEngine(database DataStore, llm LLMProvider, tasks ...TaskProvider) *Engine {
	return &Engine{
		DB:            database,
		TaskProviders: tasks,
		LLM:           llm,
		ADRDir:        "docs/adrs",
	}
}

func (e *Engine) loadADRs(dir string) ([]string, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return []string{}, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var adrs []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		adrs = append(adrs, string(data))
	}
	return adrs, nil
}

func (e *Engine) fetchAllTasks(ctx context.Context) ([]Task, map[string]TaskProvider, error) {
	allTasks := []Task{}
	providerMap := make(map[string]TaskProvider)

	for _, p := range e.TaskProviders {
		providerMap[p.Name()] = p
		var providerTasks []Task
		useCache := false

		if !e.FetchOnly && e.DB != nil && !isNil(e.DB) {
			cached, err := e.DB.FetchTasksByProvider(ctx, p.Name())
			if err == nil && len(cached) > 0 {
				latest := cached[0].CachedAt
				for _, t := range cached {
					if t.CachedAt > latest {
						latest = t.CachedAt
					}
				}
				cachedTime, parseErr := time.Parse(time.RFC3339, latest)
				if parseErr == nil && time.Since(cachedTime) < time.Duration(e.CacheTTL)*time.Minute {
					fmt.Printf("Using cached tasks for provider: %s\n", p.Name())
					providerTasks = cached
					useCache = true
				}
			}
		}

		if !useCache {
			fmt.Printf("Fetching fresh tasks for provider: %s\n", p.Name())
			var err error
			providerTasks, err = p.FetchTasks(ctx)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to fetch tasks from %s: %w", p.Name(), err)
			}
			if e.DB != nil && !isNil(e.DB) {
				for _, t := range providerTasks {
					e.DB.SaveTask(ctx, t)
				}
			}
		}
		allTasks = append(allTasks, providerTasks...)
	}
	return allTasks, providerMap, nil
}

func (e *Engine) RunAnalysis(ctx context.Context) error {
	allTasks, _, err := e.fetchAllTasks(ctx)
	if err != nil {
		return err
	}

	fmt.Printf("Analyzing total of %d tasks...\n", len(allTasks))

	dag, err := e.LLM.GenerateDAG(ctx, allTasks)
	if err != nil {
		return fmt.Errorf("failed to generate DAG: %w", err)
	}

	fmt.Println("Calculated Dependencies:")
	for taskID, blockers := range dag {
		fmt.Printf("  %s depends on %v\n", taskID, blockers)
		if e.DB != nil && !isNil(e.DB) {
			for _, blocker := range blockers {
				e.DB.CreateDependency(ctx, blocker, taskID)
			}
		}
	}

	adrs, _ := e.loadADRs(e.ADRDir)
	if len(adrs) == 0 {
		adrs = []string{"no ADRs configured"}
	}
	conflictReport, err := e.LLM.AnalyzeConflict(ctx, allTasks, adrs)
	if err != nil {
		return fmt.Errorf("failed to analyze conflicts: %w", err)
	}
	fmt.Printf("Conflict Analysis: %s\n", conflictReport)

	return nil
}

func (e *Engine) PushUpdates(ctx context.Context, confirm func(Task, string) bool) error {
	allTasks, providers, err := e.fetchAllTasks(ctx)
	if err != nil {
		return err
	}

	adrs, _ := e.loadADRs(e.ADRDir)
	if len(adrs) == 0 {
		adrs = []string{"no ADRs configured"}
	}
	conflictReport, err := e.LLM.AnalyzeConflict(ctx, allTasks, adrs)
	if err != nil {
		return fmt.Errorf("failed to analyze conflicts: %w", err)
	}

	for _, t := range allTasks {
		suggestion, err := e.LLM.SuggestTaskUpdate(ctx, t, conflictReport)
		if err != nil {
			continue
		}

		if confirm(t, suggestion) {
			if p, ok := providers[t.Provider]; ok {
				if err := p.UpdateTask(ctx, t.ID, suggestion); err != nil {
					fmt.Printf("Error updating task %s: %v\n", t.ID, err)
				} else {
					fmt.Printf("Successfully updated task %s\n", t.ID)
				}
			}
		}
	}

	return nil
}
