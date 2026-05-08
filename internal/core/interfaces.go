package core

import "context"

// Task represents a generic task from Jira or Linear
type Task struct {
	ID          string
	Title       string
	Description string
	Status      string
	Provider    string // "jira", "linear", etc.
	Metadata    map[string]string
	CachedAt    string // ISO8601 timestamp
}

// CodeComponent represents a part of the codebase (file, package, etc.)
type CodeComponent struct {
	Path        string
	Name        string
	Description string
	Metadata    map[string]string
}

// LLMProvider defines the interface for pluggable LLMs
type LLMProvider interface {
	AnalyzeConflict(ctx context.Context, tasks []Task, adrs []string) (string, error)
	GenerateDAG(ctx context.Context, tasks []Task) (map[string][]string, error) // Returns task ID to list of blocker IDs
	SuggestTaskUpdate(ctx context.Context, task Task, conflicts string) (string, error)
}

// TaskProvider defines the interface for external task trackers
type TaskProvider interface {
	Name() string
	FetchTasks(ctx context.Context) ([]Task, error)
	UpdateTask(ctx context.Context, taskID string, description string) error
}

// SourceProvider defines the interface for source control (GitHub, GitLab)
type SourceProvider interface {
	FetchMetadata(ctx context.Context) ([]CodeComponent, error)
}

// DataStore defines the interface for the knowledge graph persistence
type DataStore interface {
	SaveTask(ctx context.Context, t Task) error
	SaveCodeComponent(ctx context.Context, c CodeComponent) error
	CreateDependency(ctx context.Context, blockerID, blockedID string) error
	CreateModification(ctx context.Context, taskID, componentID string) error
	FetchTasksByProvider(ctx context.Context, provider string) ([]Task, error)
}
