package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/markkovari/nestor/internal/config"
	"github.com/markkovari/nestor/internal/db"
	"github.com/spf13/cobra"
)

var dagJSON bool

var dagCmd = &cobra.Command{
	Use:   "dag",
	Short: "Show the task dependency graph stored in the database",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		if cfg.Database.URL == "" {
			return fmt.Errorf("database.url not configured — run 'nestor check' first to populate the graph")
		}

		ctx := cmd.Context()
		database, err := db.NewDatabase(ctx, cfg.Database)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer database.Close(ctx)

		deps, err := database.FetchDependencies(ctx)
		if err != nil {
			return fmt.Errorf("failed to fetch dependencies: %w", err)
		}

		if dagJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(deps)
		}

		if len(deps) == 0 {
			fmt.Println("No dependencies stored. Run 'nestor check' first.")
			return nil
		}

		tasks := make([]string, 0, len(deps))
		for t := range deps {
			tasks = append(tasks, t)
		}
		sort.Strings(tasks)

		for _, t := range tasks {
			blockers := deps[t]
			if len(blockers) == 0 {
				fmt.Printf("  %s (no blockers)\n", t)
			} else {
				fmt.Printf("  %s blocked by: %v\n", t, blockers)
			}
		}
		return nil
	},
}

func init() {
	dagCmd.Flags().BoolVar(&dagJSON, "json", false, "output dependency graph as JSON")
	rootCmd.AddCommand(dagCmd)
}
