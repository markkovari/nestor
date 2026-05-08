package db

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/markkovari/nestor/internal/config"
	"github.com/markkovari/nestor/internal/core"
	"github.com/surrealdb/surrealdb.go"
)

type Database struct {
	Conn *surrealdb.DB
}

func NewDatabase(ctx context.Context, cfg config.DatabaseConfig) (*Database, error) {
	// Use FromEndpointURLString for v1.4.0
	db, err := surrealdb.FromEndpointURLString(ctx, cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to surrealdb: %w", err)
	}

	if _, err = db.SignIn(ctx, surrealdb.Auth{
		Namespace: cfg.NS,
		Database:  cfg.DB,
		Username:  cfg.User,
		Password:  cfg.Password,
	}); err != nil {
		return nil, fmt.Errorf("failed to sign in to surrealdb: %w", err)
	}

	if err = db.Use(ctx, cfg.NS, cfg.DB); err != nil {
		return nil, fmt.Errorf("failed to select namespace/database: %w", err)
	}

	return &Database{Conn: db}, nil
}

func (d *Database) Close(ctx context.Context) {
	if d.Conn != nil {
		_ = d.Conn.Close(ctx)
	}
}

// SaveTask upserts a task into the database
func (d *Database) SaveTask(ctx context.Context, t core.Task) error {
	data := map[string]any{
		"title":       t.Title,
		"description": t.Description,
		"status":      t.Status,
		"provider":    t.Provider,
		"metadata":    t.Metadata,
		"cached_at":   time.Now().Format(time.RFC3339),
	}

	id := fmt.Sprintf("task:%s", t.ID)
	// v1.4.0 uses package-level Update with DB instance as argument
	_, err := surrealdb.Update[any](ctx, d.Conn, id, data)
	if err != nil {
		return fmt.Errorf("failed to save task %s: %w", t.ID, err)
	}

	return nil
}

// FetchTasksByProvider retrieves all tasks for a specific provider
func (d *Database) FetchTasksByProvider(ctx context.Context, provider string) ([]core.Task, error) {
	q := "SELECT * FROM task WHERE provider = $provider"
	vars := map[string]any{
		"provider": provider,
	}

	res, err := surrealdb.Query[[]core.Task](ctx, d.Conn, q, vars)
	if err != nil {
		return nil, err
	}

	if len(*res) == 0 {
		return []core.Task{}, nil
	}

	return (*res)[0].Result, nil
}

// SaveCodeComponent upserts a code component into the database
func (d *Database) SaveCodeComponent(ctx context.Context, c core.CodeComponent) error {
	data := map[string]any{
		"path":        c.Path,
		"name":        c.Name,
		"description": c.Description,
		"metadata":    c.Metadata,
	}

	id := fmt.Sprintf("code_component:%s", c.Name)
	_, err := surrealdb.Update[any](ctx, d.Conn, id, data)
	if err != nil {
		return fmt.Errorf("failed to save code component %s: %w", c.Name, err)
	}

	return nil
}

// CreateDependency creates a 'blocks' relationship between two tasks
func (d *Database) CreateDependency(ctx context.Context, blockerID, blockedID string) error {
	q := fmt.Sprintf("RELATE task:%s->blocks->task:%s;", blockerID, blockedID)
	_, err := surrealdb.Query[any](ctx, d.Conn, q, nil)
	if err != nil {
		return fmt.Errorf("failed to relate task %s to %s: %w", blockerID, blockedID, err)
	}
	return nil
}

// CreateModification creates a 'modifies' relationship between a task and a code component
func (d *Database) CreateModification(ctx context.Context, taskID, componentID string) error {
	q := fmt.Sprintf("RELATE task:%s->modifies->code_component:%s;", taskID, componentID)
	_, err := surrealdb.Query[any](ctx, d.Conn, q, nil)
	if err != nil {
		return fmt.Errorf("failed to relate task %s to component %s: %w", taskID, componentID, err)
	}
	return nil
}

// SaveConflictHash records a conflict hash to prevent duplicate reporting.
func (d *Database) SaveConflictHash(ctx context.Context, hash string) error {
	data := map[string]any{
		"hash":       hash,
		"created_at": time.Now().Format(time.RFC3339),
	}
	_, err := surrealdb.Update[any](ctx, d.Conn, fmt.Sprintf("conflict_hash:%s", hash), data)
	return err
}

// HasConflictHash returns true if the hash was previously saved.
func (d *Database) HasConflictHash(ctx context.Context, hash string) (bool, error) {
	res, err := surrealdb.Query[[]map[string]any](ctx, d.Conn,
		"SELECT hash FROM conflict_hash WHERE hash = $hash LIMIT 1",
		map[string]any{"hash": hash})
	if err != nil {
		return false, err
	}
	if res == nil || len(*res) == 0 {
		return false, nil
	}
	return len((*res)[0].Result) > 0, nil
}

// sha256sum returns the SHA-256 digest of the given string.
func sha256sum(s string) [32]byte {
	return sha256.Sum256([]byte(s))
}

// SaveConflictFinding upserts a ConflictFinding record, keyed by a hash of task_id+adr_ref.
func (d *Database) SaveConflictFinding(ctx context.Context, f core.ConflictFinding) error {
	data := map[string]any{
		"task_id":    f.TaskID,
		"task_title": f.TaskTitle,
		"adr_ref":    f.ADRRef,
		"clause":     f.Clause,
		"reason":     f.Reason,
		"severity":   f.Severity,
		"created_at": time.Now().Format(time.RFC3339),
	}
	h := fmt.Sprintf("%x", sha256sum(f.TaskID+"|"+f.ADRRef))[:16]
	_, err := surrealdb.Update[any](ctx, d.Conn, fmt.Sprintf("conflict_finding:%s", h), data)
	return err
}

// FetchConflictFindings retrieves all conflict finding records.
func (d *Database) FetchConflictFindings(ctx context.Context) ([]core.ConflictFinding, error) {
	res, err := surrealdb.Query[[]core.ConflictFinding](ctx, d.Conn, "SELECT * FROM conflict_finding", nil)
	if err != nil {
		return nil, err
	}
	if res == nil || len(*res) == 0 {
		return []core.ConflictFinding{}, nil
	}
	return (*res)[0].Result, nil
}

// FetchConflictFindingsByTask retrieves all conflict findings for the given task ID.
func (d *Database) FetchConflictFindingsByTask(ctx context.Context, taskID string) ([]core.ConflictFinding, error) {
	res, err := surrealdb.Query[[]core.ConflictFinding](ctx, d.Conn,
		"SELECT * FROM conflict_finding WHERE task_id = $task_id",
		map[string]any{"task_id": taskID})
	if err != nil {
		return nil, err
	}
	if res == nil || len(*res) == 0 {
		return []core.ConflictFinding{}, nil
	}
	return (*res)[0].Result, nil
}

// FetchDependencies returns a map of taskID -> []blockerIDs by querying the blocks relation.
func (d *Database) FetchDependencies(ctx context.Context) (map[string][]string, error) {
	type edge struct {
		In  string `json:"in"`
		Out string `json:"out"`
	}
	res, err := surrealdb.Query[[]edge](ctx, d.Conn, "SELECT in, out FROM blocks", nil)
	if err != nil {
		return nil, err
	}
	dag := make(map[string][]string)
	if res == nil || len(*res) == 0 {
		return dag, nil
	}
	for _, e := range (*res)[0].Result {
		// RELATE blocker->blocks->blocked means: out is blocked by in
		// in = blocker (task:A), out = blocked (task:B)
		blocker := strings.TrimPrefix(e.In, "task:")
		blocked := strings.TrimPrefix(e.Out, "task:")
		dag[blocked] = append(dag[blocked], blocker)
	}
	return dag, nil
}

// InitSchema sets up basic tables and indexes for the knowledge graph
func (d *Database) InitSchema(ctx context.Context) error {
	queries := []string{
		"DEFINE TABLE task SCHEMAFULL;",
		"DEFINE FIELD title ON task TYPE string;",
		"DEFINE FIELD description ON task TYPE string;",
		"DEFINE FIELD provider ON task TYPE string;",
		"DEFINE FIELD status ON task TYPE string;",
		"DEFINE FIELD cached_at ON task TYPE string;",

		"DEFINE TABLE code_component SCHEMAFULL;",
		"DEFINE FIELD path ON code_component TYPE string;",
		"DEFINE FIELD name ON code_component TYPE string;",
		"DEFINE FIELD description ON code_component TYPE string;",
		"DEFINE FIELD metadata ON code_component TYPE object;",

		"DEFINE TABLE blocks TYPE RELATION FROM task TO task;",
		"DEFINE TABLE modifies TYPE RELATION FROM task TO code_component;",

		"DEFINE TABLE conflict_hash SCHEMAFULL;",
		"DEFINE FIELD hash ON conflict_hash TYPE string;",
		"DEFINE FIELD created_at ON conflict_hash TYPE string;",

		"DEFINE TABLE conflict_finding SCHEMAFULL;",
		"DEFINE FIELD task_id ON conflict_finding TYPE string;",
		"DEFINE FIELD task_title ON conflict_finding TYPE string;",
		"DEFINE FIELD adr_ref ON conflict_finding TYPE string;",
		"DEFINE FIELD clause ON conflict_finding TYPE string;",
		"DEFINE FIELD reason ON conflict_finding TYPE string;",
		"DEFINE FIELD severity ON conflict_finding TYPE string;",
		"DEFINE FIELD created_at ON conflict_finding TYPE string;",
	}

	for _, q := range queries {
		if _, err := surrealdb.Query[any](ctx, d.Conn, q, nil); err != nil {
			return fmt.Errorf("failed to execute schema query [%s]: %w", q, err)
		}
	}

	return nil
}
