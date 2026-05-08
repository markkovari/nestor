package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/markkovari/nestor/internal/config"
	"github.com/markkovari/nestor/internal/db"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run Nestor as an MCP server (stdio transport)",
	RunE:  runMCP,
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type mcpResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id"`
	Result  any       `json:"result,omitempty"`
	Error   *mcpError `json:"error,omitempty"`
}

type mcpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type mcpTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func runMCP(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	engine, database, err := initializeEngine(ctx, cfg)
	if err != nil {
		return err
	}
	if database != nil {
		defer database.Close(ctx)
	}

	tools := []mcpTool{
		{
			Name:        "nestor_check",
			Description: "Run Nestor conflict and dependency analysis on configured task providers. Returns structured JSON with task dependencies and ADR violations.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
				"required":   []string{},
			},
		},
		{
			Name:        "nestor_dag",
			Description: "Return the task dependency graph (DAG) as JSON. Keys are task IDs, values are arrays of task IDs they depend on.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
				"required":   []string{},
			},
		},
	}

	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req mcpRequest
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}

		var resp mcpResponse
		resp.JSONRPC = "2.0"
		resp.ID = req.ID

		switch req.Method {
		case "initialize":
			resp.Result = map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": "nestor", "version": "0.1.0"},
			}

		case "tools/list":
			resp.Result = map[string]any{"tools": tools}

		case "tools/call":
			var p struct {
				Name      string         `json:"name"`
				Arguments map[string]any `json:"arguments"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				resp.Error = &mcpError{Code: -32602, Message: "invalid params"}
				break
			}

			switch p.Name {
			case "nestor_check":
				result, err := engine.RunAnalysisResult(ctx)
				if err != nil {
					resp.Error = &mcpError{Code: -32603, Message: err.Error()}
					break
				}
				out, _ := json.Marshal(result)
				resp.Result = map[string]any{
					"content": []map[string]any{
						{"type": "text", "text": string(out)},
					},
				}

			case "nestor_dag":
				if cfg.Database.URL == "" {
					resp.Error = &mcpError{Code: -32603, Message: "database.url not configured — run 'nestor check' first to populate the graph"}
					break
				}
				dagDB, err := db.NewDatabase(ctx, cfg.Database)
				if err != nil {
					resp.Error = &mcpError{Code: -32603, Message: fmt.Sprintf("failed to connect to database: %v", err)}
					break
				}
				defer dagDB.Close(ctx)
				deps, err := dagDB.FetchDependencies(ctx)
				if err != nil {
					resp.Error = &mcpError{Code: -32603, Message: fmt.Sprintf("failed to fetch dependencies: %v", err)}
					break
				}
				out, _ := json.Marshal(deps)
				resp.Result = map[string]any{
					"content": []map[string]any{
						{"type": "text", "text": string(out)},
					},
				}

			default:
				resp.Error = &mcpError{Code: -32601, Message: fmt.Sprintf("unknown tool: %s", p.Name)}
			}

		case "notifications/initialized":
			continue

		default:
			resp.Error = &mcpError{Code: -32601, Message: fmt.Sprintf("method not found: %s", req.Method)}
		}

		encoder.Encode(resp)
	}

	return scanner.Err()
}
