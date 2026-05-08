package cli

import (
	"fmt"

	"github.com/markkovari/nestor/internal/config"
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

		fetchOnly, _ := cmd.Flags().GetBool("fetch")
		ctx := cmd.Context()

		engine, database, err := initializeEngine(ctx, cfg)
		if err != nil {
			return err
		}
		if database != nil {
			defer database.Close(ctx)
		}

		engine.FetchOnly = fetchOnly

		fmt.Printf("Configuration loaded (LLM Provider: %s, Cache TTL: %d min)\n", cfg.LLM.Provider, engine.CacheTTL)
		fmt.Println("Nestor is starting project analysis...")
		
		return engine.RunAnalysis(ctx)
	},
}

func init() {
	checkCmd.Flags().Bool("fetch", false, "Bypass cache and fetch fresh tasks from providers")
	rootCmd.AddCommand(checkCmd)
}
