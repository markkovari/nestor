package task

import (
	"context"

	"github.com/markkovari/nestor/internal/core"
)

type MockTaskProvider struct{}

func (m *MockTaskProvider) Name() string {
	return "mock"
}

func (m *MockTaskProvider) FetchTasks(ctx context.Context) ([]core.Task, error) {
	return []core.Task{
		{
			ID:          "TASK-1",
			Title:       "Implement User Auth",
			Description: "Need to add JWT based authentication",
			Status:      "Todo",
			Provider:    "mock",
		},
		{
			ID:          "TASK-2",
			Title:       "Add Social Login",
			Description: "Support Google and GitHub login. Depends on JWT auth.",
			Status:      "Todo",
			Provider:    "mock",
		},
	}, nil
}
