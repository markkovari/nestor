package cli

import (
	"fmt"
	"strings"

	"github.com/markkovari/nestor/internal/config"
	"github.com/markkovari/nestor/internal/core"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Interactively update external tasks with analysis suggestions",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		ctx := cmd.Context()
		engine, database, err := initializeEngine(ctx, cfg)
		if err != nil {
			return err
		}
		if database != nil {
			defer database.Close(ctx)
		}

		confirm := func(t core.Task, suggestion string) bool {
			fmt.Printf("\n--- Suggestion for Task %s: %s ---\n", t.ID, t.Title)
			fmt.Printf("Proposed Description:\n%s\n\n", suggestion)
			fmt.Print("Do you want to push this update to the provider? (y/N): ")
			
			var response string
			fmt.Scanln(&response)
			return strings.ToLower(response) == "y"
		}

		fmt.Println("Nestor is analyzing tasks and preparing suggestions...")
		return engine.PushUpdates(ctx, confirm)
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
}
