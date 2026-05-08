package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var adrCmd = &cobra.Command{
	Use:   "adr",
	Short: "Manage Architecture Decision Records (ADRs)",
}

var adrAddCmd = &cobra.Command{
	Use:   "add [title]",
	Short: "Add a new ADR with an agent-optimized template",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		title := strings.Join(args, " ")
		date, _ := cmd.Flags().GetString("date")
		reason, _ := cmd.Flags().GetString("reason")
		action, _ := cmd.Flags().GetString("action")
		version, _ := cmd.Flags().GetString("version")

		if date == "" {
			date = time.Now().Format("2006-01-02")
		}

		// Calculate index
		files, _ := os.ReadDir("docs/adrs")
		index := len(files) + 1
		filename := fmt.Sprintf("%04d-%s.md", index, strings.ReplaceAll(strings.ToLower(title), " ", "-"))
		path := filepath.Join("docs/adrs", filename)

		template := fmt.Sprintf(`---
date: %s
reason: %s
tldr: %s
version: %s
action: %s
---

# ADR-%04d: %s

## Status
Proposed

## Context
[Describe the forces at play...]

## Decision
[The proactive decision...]

## Rationale
[Why we chose this path...]

## Consequences
[What happens next...]
`, date, reason, title, version, action, index, title)

		if err := os.MkdirAll("docs/adrs", 0755); err != nil {
			return err
		}

		if err := os.WriteFile(path, []byte(template), 0644); err != nil {
			return err
		}

		fmt.Printf("ADR created: %s\n", path)
		return nil
	},
}

func init() {
	adrAddCmd.Flags().StringP("date", "d", "", "Date of the ADR (YYYY-MM-DD), defaults to today")
	adrAddCmd.Flags().StringP("reason", "r", "New architectural decision", "The reason/context for this ADR")
	adrAddCmd.Flags().StringP("action", "a", "creation", "Action type (creation, deprecation, revision)")
	adrAddCmd.Flags().StringP("version", "v", "1.0", "Version of the decision")

	adrCmd.AddCommand(adrAddCmd)
	rootCmd.AddCommand(adrCmd)
}
