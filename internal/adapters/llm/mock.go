package llm

import (
	"context"
	"fmt"

	"github.com/markkovari/nestor/internal/core"
)

type MockLLM struct{}

func (m *MockLLM) AnalyzeConflict(ctx context.Context, tasks []core.Task, adrs []string) (string, error) {
	if len(tasks) < 2 {
		return "No conflict detected (insufficient tasks for analysis).", nil
	}
	return fmt.Sprintf("MOCK ANALYSIS: Task %s might conflict with Task %s regarding ADRs: %v", tasks[0].ID, tasks[1].ID, adrs), nil
}

func (m *MockLLM) GenerateDAG(ctx context.Context, tasks []core.Task) (map[string][]string, error) {
	dag := make(map[string][]string)
	for i, task := range tasks {
		if i > 0 {
			// Mock a simple sequential dependency
			dag[task.ID] = []string{tasks[i-1].ID}
		} else {
			dag[task.ID] = []string{}
		}
	}
	return dag, nil
}
