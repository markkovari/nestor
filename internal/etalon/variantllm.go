package etalon

import (
	"context"

	"github.com/markkovari/nestor/internal/core"
)

type variantLLM struct {
	inner   core.LLMProvider
	variant PromptVariant
}

// NewVariantLLM wraps any LLMProvider with a prompt variant prefix strategy
func NewVariantLLM(inner core.LLMProvider, v PromptVariant) core.LLMProvider {
	return &variantLLM{inner: inner, variant: v}
}

func (v *variantLLM) GenerateDAG(ctx context.Context, tasks []core.Task) (map[string][]string, error) {
	// The variant prefix is embedded in the task list by adding a synthetic first task
	// that acts as a system instruction — simpler than forking the LLM interface
	if v.variant.DAGPrefix != "" {
		tasks = append([]core.Task{{
			ID:          "_INSTRUCTION",
			Title:       "System instruction",
			Description: v.variant.DAGPrefix,
			Provider:    "system",
		}}, tasks...)
	}
	dag, err := v.inner.GenerateDAG(ctx, tasks)
	if err != nil {
		return nil, err
	}
	delete(dag, "_INSTRUCTION")
	return dag, nil
}

func (v *variantLLM) AnalyzeConflict(ctx context.Context, tasks []core.Task, adrs []string) (string, error) {
	if v.variant.ConflictPrefix != "" {
		adrs = append([]string{v.variant.ConflictPrefix}, adrs...)
	}
	return v.inner.AnalyzeConflict(ctx, tasks, adrs)
}

func (v *variantLLM) AnalyzeConflictStructured(ctx context.Context, tasks []core.Task, adrs []string) (*core.ConflictReport, error) {
	if v.variant.ConflictPrefix != "" {
		adrs = append([]string{v.variant.ConflictPrefix}, adrs...)
	}
	return v.inner.AnalyzeConflictStructured(ctx, tasks, adrs)
}

func (v *variantLLM) SuggestTaskUpdate(ctx context.Context, t core.Task, conflicts string) (string, error) {
	return v.inner.SuggestTaskUpdate(ctx, t, conflicts)
}
