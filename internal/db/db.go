package db

import (
	"context"
	"fmt"
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
		d.Conn.Close(ctx)
	}
}

// SaveTask upserts a task into the database
func (d *Database) SaveTask(ctx context.Context, t core.Task) error {
	data := map[string]interface{}{
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
	vars := map[string]interface{}{
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
	data := map[string]interface{}{
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

		"DEFINE TABLE blocks TYPE RELATION FROM task TO task;",
		"DEFINE TABLE modifies TYPE RELATION FROM task TO code_component;",
	}

	for _, q := range queries {
		if _, err := surrealdb.Query[any](ctx, d.Conn, q, nil); err != nil {
			return fmt.Errorf("failed to execute schema query [%s]: %w", q, err)
		}
	}

	return nil
}
