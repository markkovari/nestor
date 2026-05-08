package etalon

// EtalonTask is one task entry from etalon.json
type EtalonTask struct {
	ID                   string   `json:"id"`
	Title                string   `json:"title"`
	Description          string   `json:"description"`
	Status               string   `json:"status"`
	ExpectConflicts      bool     `json:"expect_conflicts"`
	ExpectConflictReason string   `json:"expect_conflict_reason"`
	ExpectDependsOn      []string `json:"expect_depends_on"`
}

// Manifest is the full etalon.json structure
type Manifest struct {
	Version     string              `json:"version"`
	Description string              `json:"description"`
	ADRs        []string            `json:"adrs"`
	Tasks       []EtalonTask        `json:"tasks"`
	ExpectedDAG map[string][]string `json:"expected_dag"`
}

// EvalResult holds scores for one eval run
type EvalResult struct {
	RunAt         string           `json:"run_at"`
	Mode          string           `json:"mode"` // "fixture" or "live"
	PromptVariant string           `json:"prompt_variant"`
	DAGScore      DAGScore         `json:"dag_score"`
	ConflictScore ConflictScore    `json:"conflict_score"`
	OverallScore  float64          `json:"overall_score"`
	Details       []TaskEvalDetail `json:"details"`
}

type DAGScore struct {
	TotalPairs   int     `json:"total_pairs"`
	CorrectPairs int     `json:"correct_pairs"`
	Precision    float64 `json:"precision"`
	Recall       float64 `json:"recall"`
	F1           float64 `json:"f1"`
}

type ConflictScore struct {
	TruePositives  int     `json:"true_positives"`
	FalsePositives int     `json:"false_positives"`
	FalseNegatives int     `json:"false_negatives"`
	Precision      float64 `json:"precision"`
	Recall         float64 `json:"recall"`
	F1             float64 `json:"f1"`
}

type TaskEvalDetail struct {
	TaskID           string   `json:"task_id"`
	ExpectedConflict bool     `json:"expected_conflict"`
	GotConflict      bool     `json:"got_conflict"`
	ExpectedDeps     []string `json:"expected_deps"`
	GotDeps          []string `json:"got_deps"`
	ConflictCorrect  bool     `json:"conflict_correct"`
	DepsCorrect      bool     `json:"deps_correct"`
}
