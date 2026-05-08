package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "nestor",
	Short: "Nestor is an intelligent project checker and task dependency analyzer",
	Long: `Nestor helps teams stay aligned by analyzing GitHub repos, task trackers (Jira/Linear), 
and documentation to find potential code breakages and task contradictions.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Root flags can be defined here
}
