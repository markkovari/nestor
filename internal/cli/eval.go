package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/markkovari/nestor/internal/adapters/llm"
	"github.com/markkovari/nestor/internal/config"
	"github.com/markkovari/nestor/internal/core"
	"github.com/markkovari/nestor/internal/etalon"
	"github.com/spf13/cobra"
)

var evalCmd = &cobra.Command{
	Use:   "eval",
	Short: "Evaluate Nestor accuracy against etalon fixtures",
	RunE:  runEval,
}

var (
	evalEtalonDir  string
	evalLive       bool
	evalLoopN      int
	evalOutputFile string
	evalHistory    string
)

func init() {
	evalCmd.Flags().StringVar(&evalEtalonDir, "etalon-dir", ".", "path to etalon directory containing etalon.json")
	evalCmd.Flags().BoolVar(&evalLive, "live", false, "use real LLM (requires config) instead of eval mock")
	evalCmd.Flags().IntVar(&evalLoopN, "loop", len(etalon.DefaultVariants), "number of prompt variants to run (1–4)")
	evalCmd.Flags().StringVar(&evalOutputFile, "output", "eval-report.json", "path to write JSON report")
	evalCmd.Flags().StringVar(&evalHistory, "history", "eval-history.jsonl", "path to append run results for trend tracking")
	rootCmd.AddCommand(evalCmd)
}

func runEval(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	manifest, err := etalon.LoadManifest(evalEtalonDir)
	if err != nil {
		return fmt.Errorf("failed to load etalon manifest: %w", err)
	}

	adrs, err := etalon.LoadADRContents(evalEtalonDir, manifest)
	if err != nil {
		return fmt.Errorf("failed to load ADR contents: %w", err)
	}

	tasks := etalon.TasksToCoreTasks(manifest.Tasks)

	var baseLLM core.LLMProvider
	if evalLive {
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config for live mode: %w", err)
		}
		baseLLM, err = llm.NewGeminiProvider(ctx, cfg.LLM.APIKey, cfg.LLM.Model)
		if err != nil {
			return fmt.Errorf("failed to init LLM: %w", err)
		}
	} else {
		baseLLM = &etalon.EvalMockLLM{Manifest: manifest}
	}

	n := max(1, min(evalLoopN, len(etalon.DefaultVariants)))
	variants := etalon.DefaultVariants[:n]

	fmt.Printf("Nestor eval — mode: %s, etalon: %s, variants: %d\n\n", modeStr(evalLive), evalEtalonDir, len(variants))
	fmt.Printf("%-30s  %6s  %6s  %6s  %6s  %6s  %6s  %7s\n",
		"variant", "dag-P", "dag-R", "dag-F1", "con-P", "con-R", "con-F1", "overall")
	fmt.Printf("%s\n", repeatStr("-", 85))

	var results []etalon.EvalResult
	bestScore := -1.0
	bestVariant := ""

	for _, variant := range variants {
		wrappedLLM := etalon.NewVariantLLM(baseLLM, variant)

		dag, err := wrappedLLM.GenerateDAG(ctx, tasks)
		if err != nil {
			fmt.Fprintf(os.Stderr, "DAG failed for variant %s: %v\n", variant.Name, err)
			continue
		}

		conflictReport, err := wrappedLLM.AnalyzeConflict(ctx, tasks, adrs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Conflict failed for variant %s: %v\n", variant.Name, err)
			continue
		}

		dagScore := etalon.ScoreDAG(manifest, dag)
		conflictScore := etalon.ScoreConflicts(manifest, conflictReport)
		details := etalon.BuildDetails(manifest, dag, conflictReport)
		overall := etalon.Round2((dagScore.F1 + conflictScore.F1) / 2)

		fmt.Printf("%-30s  %6.2f  %6.2f  %6.2f  %6.2f  %6.2f  %6.2f  %7.2f\n",
			variant.Name,
			dagScore.Precision, dagScore.Recall, dagScore.F1,
			conflictScore.Precision, conflictScore.Recall, conflictScore.F1,
			overall)

		result := etalon.EvalResult{
			RunAt:         time.Now().UTC().Format(time.RFC3339),
			Mode:          modeStr(evalLive),
			PromptVariant: variant.Name,
			DAGScore:      dagScore,
			ConflictScore: conflictScore,
			OverallScore:  overall,
			Details:       details,
		}
		results = append(results, result)

		if overall > bestScore {
			bestScore = overall
			bestVariant = variant.Name
		}
	}

	fmt.Printf("%s\n", repeatStr("-", 85))
	fmt.Printf("Best: %s  (overall F1: %.2f)\n\n", bestVariant, bestScore)

	// Conflict detail breakdown
	if len(results) > 0 {
		best := results[0]
		for _, r := range results {
			if r.PromptVariant == bestVariant {
				best = r
			}
		}
		fmt.Printf("Per-task conflict accuracy (%s):\n", bestVariant)
		fmt.Printf("  %-12s  %-8s  %-8s  %s\n", "task", "expect", "got", "correct")
		for _, d := range best.Details {
			mark := "✓"
			if !d.ConflictCorrect {
				mark = "✗"
			}
			fmt.Printf("  %-12s  %-8v  %-8v  %s\n", d.TaskID, d.ExpectedConflict, d.GotConflict, mark)
		}
		fmt.Println()
	}

	// Write full report
	report := map[string]any{
		"best_variant": bestVariant,
		"best_score":   bestScore,
		"runs":         results,
	}
	out, _ := json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile(evalOutputFile, out, 0644); err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}
	fmt.Printf("Report written to %s\n", evalOutputFile)

	// Append to history for trend tracking
	if err := appendHistory(evalHistory, results); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write history: %v\n", err)
	} else {
		fmt.Printf("History appended to %s\n", evalHistory)
	}

	return nil
}

func appendHistory(path string, results []etalon.EvalResult) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, r := range results {
		if err := enc.Encode(r); err != nil {
			return err
		}
	}
	return nil
}

func modeStr(live bool) string {
	if live {
		return "live"
	}
	return "fixture"
}

func repeatStr(s string, n int) string {
	var sb strings.Builder
	for range n {
		sb.WriteString(s)
	}
	return sb.String()
}
