package cli

import (
	"fmt"

	"github.com/markkovari/nestor/internal/adapters/llm"
	"github.com/markkovari/nestor/internal/adapters/task"
	"github.com/markkovari/nestor/internal/config"
	"github.com/markkovari/nestor/internal/core"
	"github.com/markkovari/nestor/internal/db"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Perform a one-off analysis of tasks and code",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		fmt.Printf("Configuration loaded (LLM Provider: %s)\n", cfg.LLM.Provider)

		// Initialize Database
		var database *db.Database
		if cfg.Database.URL != "" {
			database, err = db.NewDatabase(cmd.Context(), cfg.Database)
			if err != nil {
				fmt.Printf("Warning: failed to connect to database: %v. Continuing without persistence.\n", err)
			} else {
				defer database.Close(cmd.Context())
				fmt.Println("Connected to SurrealDB.")
			}
		}

		// Initializing mock adapters for Phase 2 demonstration
		mockLLM := &llm.MockLLM{}
		mockTasks := &task.MockTaskProvider{}

		engine := core.NewEngine(database, mockLLM, mockTasks)
		
		fmt.Println("Nestor is starting project analysis (Mock Mode)...")
		return engine.RunAnalysis(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
