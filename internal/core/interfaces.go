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

// ConflictFinding is a structured conflict finding with ADR citation.
type ConflictFinding struct {
	TaskID    string `json:"task_id"`
	TaskTitle string `json:"task_title"`
	ADRRef    string `json:"adr_ref"`    // e.g. "ADR-003" or filename
	Clause    string `json:"clause"`     // exact quoted ADR text that is violated
	Reason    string `json:"reason"`     // explanation of the violation
	Severity  string `json:"severity"`   // "high", "medium", "low"
}

// ConflictReport is the structured output of conflict analysis.
type ConflictReport struct {
	Summary  string            `json:"summary"`
	Findings []ConflictFinding `json:"findings"`
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
	AnalyzeConflictStructured(ctx context.Context, tasks []Task, adrs []string) (*ConflictReport, error)
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
	SaveConflictHash(ctx context.Context, hash string) error
	HasConflictHash(ctx context.Context, hash string) (bool, error)
	SaveConflictFinding(ctx context.Context, f ConflictFinding) error
	FetchConflictFindings(ctx context.Context) ([]ConflictFinding, error)
	FetchConflictFindingsByTask(ctx context.Context, taskID string) ([]ConflictFinding, error)
	FetchDependencies(ctx context.Context) (map[string][]string, error)
}
