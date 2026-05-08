package task

import (
	"context"
	"github.com/markkovari/nestor/internal/core"
)

type FixtureTaskProvider struct {
	Tasks []core.Task
}

func (f *FixtureTaskProvider) Name() string {
	return "fixture"
}

func (f *FixtureTaskProvider) FetchTasks(ctx context.Context) ([]core.Task, error) {
	return f.Tasks, nil
}

func (f *FixtureTaskProvider) UpdateTask(ctx context.Context, taskID string, description string) error {
	return nil
}
