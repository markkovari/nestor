package core

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

type Engine struct {
	TaskProviders   []TaskProvider
	SourceProviders []SourceProvider
	LLM             LLMProvider
	DB              DataStore
	CacheTTL        int  // in minutes
	FetchOnly       bool // bypass cache if true
	ADRDir          string
	seenHashes      map[string]bool
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
		seenHashes:    make(map[string]bool),
	}
}

func (e *Engine) WithSourceProviders(providers ...SourceProvider) *Engine {
	e.SourceProviders = providers
	return e
}

func conflictHash(taskID, adrRef, reason string) string {
	h := sha256.Sum256([]byte(taskID + "|" + adrRef + "|" + reason))
	return fmt.Sprintf("%x", h)[:16]
}

// obsoleteADRStatuses are ADR status values that mean the rule is no longer active.
var obsoleteADRStatuses = map[string]bool{
	"superseded": true,
	"deprecated": true,
	"obsolete":   true,
	"rejected":   true,
}

func adrIsActive(content string) bool {
	inStatus := false
	for _, line := range strings.SplitN(content, "\n", 20) {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## Status") {
			inStatus = true
			continue
		}
		if !inStatus || trimmed == "" {
			continue
		}
		// First non-empty line after "## Status" is the status value
		return !obsoleteADRStatuses[strings.ToLower(trimmed)]
	}
	return true
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
		if adrIsActive(string(data)) {
			adrs = append(adrs, string(data))
		}
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

func (e *Engine) fetchAndSaveCodeComponents(ctx context.Context, allTasks []Task) error {
	for _, sp := range e.SourceProviders {
		components, err := sp.FetchMetadata(ctx)
		if err != nil {
			return fmt.Errorf("failed to fetch source metadata: %w", err)
		}
		for _, component := range components {
			if err := e.DB.SaveCodeComponent(ctx, component); err != nil {
				return fmt.Errorf("failed to save code component %s: %w", component.Name, err)
			}
			for _, t := range allTasks {
				if repo, ok := t.Metadata["repo"]; ok && repo == component.Path {
					if err := e.DB.CreateModification(ctx, t.ID, component.Name); err != nil {
						// Non-fatal: log but continue
						fmt.Printf("Warning: failed to create modification edge for task %s -> component %s: %v\n", t.ID, component.Name, err)
					}
				}
			}
		}
	}
	return nil
}

func (e *Engine) RunAnalysis(ctx context.Context) error {
	allTasks, _, err := e.fetchAllTasks(ctx)
	if err != nil {
		return err
	}

	if e.DB != nil && !isNil(e.DB) && len(e.SourceProviders) > 0 {
		if err := e.fetchAndSaveCodeComponents(ctx, allTasks); err != nil {
			return err
		}
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
	report, err := e.LLM.AnalyzeConflictStructured(ctx, allTasks, adrs)
	if err != nil {
		return fmt.Errorf("failed to analyze conflicts: %w", err)
	}

	var newFindings []ConflictFinding
	for _, f := range report.Findings {
		h := conflictHash(f.TaskID, f.ADRRef, f.Reason)
		seen := e.seenHashes[h]
		if !seen && e.DB != nil && !isNil(e.DB) {
			dbSeen, _ := e.DB.HasConflictHash(ctx, h)
			seen = dbSeen
		}
		if !seen {
			newFindings = append(newFindings, f)
			e.seenHashes[h] = true
			if e.DB != nil && !isNil(e.DB) {
				e.DB.SaveConflictHash(ctx, h)
				e.DB.SaveConflictFinding(ctx, f)
			}
		}
	}
	report.Findings = newFindings
	if len(newFindings) == 0 {
		fmt.Println("No new conflicts since last run.")
		return nil
	}

	fmt.Printf("Conflict Analysis: %s\n", report.Summary)
	for _, f := range report.Findings {
		fmt.Printf("  [%s] %s — %s: %q → %s\n", f.Severity, f.TaskID, f.ADRRef, f.Clause, f.Reason)
	}

	return nil
}

// AnalysisResult holds the structured output of RunAnalysisResult.
type AnalysisResult struct {
	TaskCount        int                 `json:"task_count"`
	Dependencies     map[string][]string `json:"dependencies"`
	ConflictReport   string              `json:"conflict_report"`
	ConflictFindings []ConflictFinding   `json:"conflict_findings"`
}

func (e *Engine) RunAnalysisResult(ctx context.Context) (*AnalysisResult, error) {
	allTasks, _, err := e.fetchAllTasks(ctx)
	if err != nil {
		return nil, err
	}

	if e.DB != nil && !isNil(e.DB) && len(e.SourceProviders) > 0 {
		if err := e.fetchAndSaveCodeComponents(ctx, allTasks); err != nil {
			return nil, err
		}
	}

	dag, err := e.LLM.GenerateDAG(ctx, allTasks)
	if err != nil {
		return nil, fmt.Errorf("failed to generate DAG: %w", err)
	}

	if e.DB != nil && !isNil(e.DB) {
		for taskID, blockers := range dag {
			for _, blocker := range blockers {
				e.DB.CreateDependency(ctx, blocker, taskID)
			}
		}
	}

	adrs, _ := e.loadADRs(e.ADRDir)
	if len(adrs) == 0 {
		adrs = []string{"no ADRs configured"}
	}
	report, err := e.LLM.AnalyzeConflictStructured(ctx, allTasks, adrs)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze conflicts: %w", err)
	}

	var newFindings []ConflictFinding
	for _, f := range report.Findings {
		h := conflictHash(f.TaskID, f.ADRRef, f.Reason)
		seen := e.seenHashes[h]
		if !seen && e.DB != nil && !isNil(e.DB) {
			dbSeen, _ := e.DB.HasConflictHash(ctx, h)
			seen = dbSeen
		}
		if !seen {
			newFindings = append(newFindings, f)
			e.seenHashes[h] = true
			if e.DB != nil && !isNil(e.DB) {
				e.DB.SaveConflictHash(ctx, h)
				e.DB.SaveConflictFinding(ctx, f)
			}
		}
	}
	report.Findings = newFindings

	return &AnalysisResult{
		TaskCount:        len(allTasks),
		Dependencies:     dag,
		ConflictReport:   report.Summary + formatFindings(report.Findings),
		ConflictFindings: report.Findings,
	}, nil
}

func formatFindings(findings []ConflictFinding) string {
	if len(findings) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, f := range findings {
		fmt.Fprintf(&sb, "\n[%s] %s violates %s: %q — %s", f.Severity, f.TaskID, f.ADRRef, f.Clause, f.Reason)
	}
	return sb.String()
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
	pushReport, err := e.LLM.AnalyzeConflictStructured(ctx, allTasks, adrs)
	if err != nil {
		return fmt.Errorf("failed to analyze conflicts: %w", err)
	}
	conflictReport := pushReport.Summary + formatFindings(pushReport.Findings)

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
