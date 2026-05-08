package etalon

import (
	"context"
	"fmt"
	"strings"

	"github.com/markkovari/nestor/internal/core"
)

// EvalMockLLM is a deterministic mock that produces realistic output for eval scoring.
// Unlike the generic MockLLM it understands the etalon task set:
//   - GenerateDAG: returns the exact expected_dag from the manifest
//   - AnalyzeConflict: mentions every task ID that should conflict, by ID and by title keyword
//
// This lets fixture-mode evals produce meaningful (perfect-score) baselines.
type EvalMockLLM struct {
	Manifest *Manifest
}

func (e *EvalMockLLM) GenerateDAG(_ context.Context, tasks []core.Task) (map[string][]string, error) {
	dag := make(map[string][]string, len(tasks))
	for _, t := range tasks {
		if t.ID == "_INSTRUCTION" {
			continue
		}
		if deps, ok := e.Manifest.ExpectedDAG[t.ID]; ok {
			dag[t.ID] = deps
		} else {
			dag[t.ID] = []string{}
		}
	}
	return dag, nil
}

func (e *EvalMockLLM) AnalyzeConflict(_ context.Context, tasks []core.Task, _ []string) (string, error) {
	var sb strings.Builder
	sb.WriteString("Conflict analysis report:\n\n")

	for _, t := range e.Manifest.Tasks {
		if t.ExpectConflicts {
			fmt.Fprintf(&sb, "- %s (%s): %s\n", t.ID, t.Title, t.ExpectConflictReason)
		}
	}

	sb.WriteString("\nAll other tasks are compliant with architectural guidelines.\n")

	return sb.String(), nil
}

func (e *EvalMockLLM) SuggestTaskUpdate(_ context.Context, t core.Task, conflicts string) (string, error) {
	return fmt.Sprintf("%s\n\nNestor Analysis: %s", t.Description, conflicts), nil
}
