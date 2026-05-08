package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/markkovari/nestor/internal/config"
	"github.com/markkovari/nestor/internal/core"
	"github.com/markkovari/nestor/internal/db"
	"github.com/spf13/cobra"
)

var (
	conflictsJSON     bool
	conflictsSeverity string
	conflictsTaskID   string
)

var conflictsCmd = &cobra.Command{
	Use:   "conflicts",
	Short: "Show stored conflict findings from the database",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		if cfg.Database.URL == "" {
			return fmt.Errorf("database.url not configured — run 'nestor check' first")
		}

		ctx := cmd.Context()
		database, err := db.NewDatabase(ctx, cfg.Database)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer database.Close(ctx)

		var findings []core.ConflictFinding
		if conflictsTaskID != "" {
			findings, err = database.FetchConflictFindingsByTask(ctx, conflictsTaskID)
		} else {
			findings, err = database.FetchConflictFindings(ctx)
		}
		if err != nil {
			return fmt.Errorf("failed to fetch conflicts: %w", err)
		}

		// filter by severity if set
		if conflictsSeverity != "" {
			var filtered []core.ConflictFinding
			for _, f := range findings {
				if f.Severity == conflictsSeverity {
					filtered = append(filtered, f)
				}
			}
			findings = filtered
		}

		if len(findings) == 0 {
			fmt.Println("No conflict findings stored. Run 'nestor check' first.")
			return nil
		}

		if conflictsJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(findings)
		}

		for _, f := range findings {
			fmt.Printf("[%s] %s — %s: %q → %s\n", f.Severity, f.TaskID, f.ADRRef, f.Clause, f.Reason)
		}
		return nil
	},
}

func init() {
	conflictsCmd.Flags().BoolVar(&conflictsJSON, "json", false, "output as JSON")
	conflictsCmd.Flags().StringVar(&conflictsSeverity, "severity", "", "filter by severity (high/medium/low)")
	conflictsCmd.Flags().StringVar(&conflictsTaskID, "task", "", "filter by task ID")
	rootCmd.AddCommand(conflictsCmd)
}
