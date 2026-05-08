package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// inline mocks — no imports from adapters/* to avoid import cycle

type mockLLM struct{}

func (m *mockLLM) AnalyzeConflict(_ context.Context, tasks []Task, adrs []string) (string, error) {
	if len(tasks) < 2 {
		return "no conflict", nil
	}
	return fmt.Sprintf("mock conflict: %s vs %s, adrs: %v", tasks[0].ID, tasks[1].ID, adrs), nil
}

func (m *mockLLM) GenerateDAG(_ context.Context, tasks []Task) (map[string][]string, error) {
	dag := make(map[string][]string)
	for i, t := range tasks {
		if i > 0 {
			dag[t.ID] = []string{tasks[i-1].ID}
		} else {
			dag[t.ID] = []string{}
		}
	}
	return dag, nil
}

func (m *mockLLM) SuggestTaskUpdate(_ context.Context, t Task, conflicts string) (string, error) {
	return fmt.Sprintf("%s\n\nNestor Analysis: %s", t.Description, conflicts), nil
}

type mockTaskProvider struct {
	tasks []Task
}

func (m *mockTaskProvider) Name() string { return "mock" }
func (m *mockTaskProvider) FetchTasks(_ context.Context) ([]Task, error) {
	if m.tasks != nil {
		return m.tasks, nil
	}
	return []Task{
		{ID: "TASK-1", Title: "Auth", Description: "JWT auth", Status: "Todo", Provider: "mock"},
		{ID: "TASK-2", Title: "Social Login", Description: "Depends on JWT", Status: "Todo", Provider: "mock"},
	}, nil
}
func (m *mockTaskProvider) UpdateTask(_ context.Context, _ string, _ string) error { return nil }

func TestRunAnalysis_MockProviders(t *testing.T) {
	e := NewEngine(nil, &mockLLM{}, &mockTaskProvider{})
	if err := e.RunAnalysis(context.Background()); err != nil {
		t.Fatalf("RunAnalysis returned unexpected error: %v", err)
	}
}

func TestRunAnalysis_NoTasks(t *testing.T) {
	e := NewEngine(nil, &mockLLM{}, &mockTaskProvider{tasks: []Task{}})
	if err := e.RunAnalysis(context.Background()); err != nil {
		t.Fatalf("RunAnalysis with empty tasks returned unexpected error: %v", err)
	}
}

func TestPushUpdates_ConfirmAll(t *testing.T) {
	e := NewEngine(nil, &mockLLM{}, &mockTaskProvider{})
	if err := e.PushUpdates(context.Background(), func(_ Task, _ string) bool { return true }); err != nil {
		t.Fatalf("PushUpdates (confirm all) returned unexpected error: %v", err)
	}
}

func TestPushUpdates_ConfirmNone(t *testing.T) {
	e := NewEngine(nil, &mockLLM{}, &mockTaskProvider{})
	if err := e.PushUpdates(context.Background(), func(_ Task, _ string) bool { return false }); err != nil {
		t.Fatalf("PushUpdates (confirm none) returned unexpected error: %v", err)
	}
}

func TestLoadADRs_NonexistentDir(t *testing.T) {
	e := NewEngine(nil, &mockLLM{})
	adrs, err := e.loadADRs("nonexistent/path")
	if err != nil {
		t.Fatalf("loadADRs on nonexistent dir returned error: %v", err)
	}
	if len(adrs) != 0 {
		t.Errorf("expected empty slice, got %d entries", len(adrs))
	}
}

func TestLoadADRs_EmptyDir(t *testing.T) {
	e := NewEngine(nil, &mockLLM{})
	adrs, err := e.loadADRs(t.TempDir())
	if err != nil {
		t.Fatalf("loadADRs on empty dir returned error: %v", err)
	}
	if len(adrs) != 0 {
		t.Errorf("expected empty slice, got %d entries", len(adrs))
	}
}

func TestLoadADRs_WithFiles(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"001-first.md":  "# First ADR\nContent of first.",
		"002-second.md": "# Second ADR\nContent of second.",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatalf("failed to create fixture %s: %v", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "ignore.txt"), []byte("skip me"), 0644); err != nil {
		t.Fatalf("failed to create decoy fixture: %v", err)
	}

	e := NewEngine(nil, &mockLLM{})
	adrs, err := e.loadADRs(dir)
	if err != nil {
		t.Fatalf("loadADRs returned error: %v", err)
	}
	if len(adrs) != 2 {
		t.Fatalf("expected 2 ADR entries, got %d", len(adrs))
	}
	loaded := make(map[string]bool, len(adrs))
	for _, a := range adrs {
		loaded[a] = true
	}
	for _, content := range files {
		if !loaded[content] {
			t.Errorf("content %q not found in loaded ADRs", content)
		}
	}
}
