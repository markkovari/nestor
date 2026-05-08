package task

import (
	"context"
	"github.com/markkovari/nestor/internal/core"
)

type FixtureTaskProvider struct {
	Tasks []core.Task
}

func (f *FixtureTaskProvider) FetchTasks(ctx context.Context) ([]core.Task, error) {
	return f.Tasks, nil
}
