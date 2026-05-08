package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run Nestor as a background server for webhooks and MCP",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Nestor server starting...")
		// Server logic will be called here
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
