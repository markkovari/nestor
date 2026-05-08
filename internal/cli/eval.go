package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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
)

func init() {
	evalCmd.Flags().StringVar(&evalEtalonDir, "etalon-dir", ".", "path to etalon directory containing etalon.json")
	evalCmd.Flags().BoolVar(&evalLive, "live", false, "use real LLM (requires config) instead of mock")
	evalCmd.Flags().IntVar(&evalLoopN, "loop", 1, "number of prompt variant iterations to run")
	evalCmd.Flags().StringVar(&evalOutputFile, "output", "eval-report.json", "path to write JSON report")
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

	// Determine LLM provider
	var llmProvider core.LLMProvider
	if evalLive {
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config for live mode: %w", err)
		}
		llmProvider, err = llm.NewGeminiProvider(ctx, cfg.LLM.APIKey, cfg.LLM.Model)
		if err != nil {
			return fmt.Errorf("failed to init LLM: %w", err)
		}
	} else {
		llmProvider = &llm.MockLLM{}
	}

	variants := etalon.DefaultVariants
	if evalLoopN > len(variants) {
		evalLoopN = len(variants)
	}
	variants = variants[:evalLoopN]

	var results []etalon.EvalResult
	bestScore := -1.0
	bestVariant := ""

	for _, variant := range variants {
		fmt.Printf("\n--- Variant: %s ---\n", variant.Name)

		wrappedLLM := etalon.NewVariantLLM(llmProvider, variant)

		dag, err := wrappedLLM.GenerateDAG(ctx, tasks)
		if err != nil {
			fmt.Fprintf(os.Stderr, "DAG generation failed for variant %s: %v\n", variant.Name, err)
			continue
		}

		conflictReport, err := wrappedLLM.AnalyzeConflict(ctx, tasks, adrs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Conflict analysis failed for variant %s: %v\n", variant.Name, err)
			continue
		}

		dagScore := etalon.ScoreDAG(manifest, dag)
		conflictScore := etalon.ScoreConflicts(manifest, conflictReport)
		details := etalon.BuildDetails(manifest, dag, conflictReport)

		overall := (dagScore.F1 + conflictScore.F1) / 2

		result := etalon.EvalResult{
			RunAt:         time.Now().UTC().Format(time.RFC3339),
			Mode:          modeStr(evalLive),
			PromptVariant: variant.Name,
			DAGScore:      dagScore,
			ConflictScore: conflictScore,
			OverallScore:  etalon.Round2(overall),
			Details:       details,
		}
		results = append(results, result)

		fmt.Printf("  DAG      — precision: %.2f  recall: %.2f  F1: %.2f\n", dagScore.Precision, dagScore.Recall, dagScore.F1)
		fmt.Printf("  Conflict — precision: %.2f  recall: %.2f  F1: %.2f  (TP:%d FP:%d FN:%d)\n",
			conflictScore.Precision, conflictScore.Recall, conflictScore.F1,
			conflictScore.TruePositives, conflictScore.FalsePositives, conflictScore.FalseNegatives)
		fmt.Printf("  Overall  — %.2f\n", overall)

		if overall > bestScore {
			bestScore = overall
			bestVariant = variant.Name
		}
	}

	fmt.Printf("\nBest variant: %s (score: %.2f)\n", bestVariant, bestScore)

	// Write report
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
	return nil
}

func modeStr(live bool) string {
	if live {
		return "live"
	}
	return "fixture"
}
