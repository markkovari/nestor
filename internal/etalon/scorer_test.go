package etalon

import (
	"context"
	"testing"
)

func testCtx() context.Context { return context.Background() }

var testManifest = &Manifest{
	Tasks: []EtalonTask{
		{ID: "T-1", Title: "Drop legacy password hash column", ExpectConflicts: true, ExpectConflictReason: "violates ADR-002"},
		{ID: "T-2", Title: "Add versioned login endpoint", ExpectConflicts: false},
		{ID: "T-3", Title: "Implement session cookies for admin", ExpectConflicts: true, ExpectConflictReason: "violates ADR-003"},
		{ID: "T-4", Title: "Add health check endpoint", ExpectConflicts: false},
	},
	ExpectedDAG: map[string][]string{
		"T-1": {},
		"T-2": {"T-1"},
		"T-3": {},
		"T-4": {"T-1", "T-2"},
	},
}

func TestScoreDAG_Perfect(t *testing.T) {
	got := map[string][]string{
		"T-1": {},
		"T-2": {"T-1"},
		"T-3": {},
		"T-4": {"T-1", "T-2"},
	}
	s := ScoreDAG(testManifest, got)
	if s.F1 != 1.0 {
		t.Errorf("expected F1=1.0, got %.2f", s.F1)
	}
	if s.CorrectPairs != s.TotalPairs {
		t.Errorf("expected all pairs correct: total=%d correct=%d", s.TotalPairs, s.CorrectPairs)
	}
}

func TestScoreDAG_Empty(t *testing.T) {
	got := map[string][]string{}
	s := ScoreDAG(testManifest, got)
	if s.Recall != 0 {
		t.Errorf("empty DAG should have recall=0, got %.2f", s.Recall)
	}
	if s.F1 != 0 {
		t.Errorf("empty DAG should have F1=0, got %.2f", s.F1)
	}
}

func TestScoreDAG_FalsePositives(t *testing.T) {
	got := map[string][]string{
		"T-1": {},
		"T-2": {"T-1", "T-3"}, // T-3 is extra (false positive)
		"T-3": {"T-4"},         // T-4 dep is extra
		"T-4": {"T-1", "T-2"},
	}
	s := ScoreDAG(testManifest, got)
	if s.Precision >= 1.0 {
		t.Errorf("expected precision < 1.0 due to false positives, got %.2f", s.Precision)
	}
}

func TestScoreConflicts_ByID(t *testing.T) {
	// Report mentions conflict tasks by ID
	report := "Conflicts detected: T-1 violates migration policy. T-3 violates auth policy."
	s := ScoreConflicts(testManifest, report)
	if s.TruePositives != 2 {
		t.Errorf("expected 2 TPs (T-1, T-3), got %d", s.TruePositives)
	}
	if s.FalseNegatives != 0 {
		t.Errorf("expected 0 FNs, got %d", s.FalseNegatives)
	}
}

func TestScoreConflicts_ByKeyword(t *testing.T) {
	// Report describes conflicts by content, not by ID
	report := "The task to drop legacy password hash column violates ADR-002. " +
		"Implement session cookies for admin also violates ADR-003."
	s := ScoreConflicts(testManifest, report)
	if s.TruePositives != 2 {
		t.Errorf("expected 2 TPs via keyword match, got %d (FN=%d)", s.TruePositives, s.FalseNegatives)
	}
}

func TestScoreConflicts_AllMissed(t *testing.T) {
	report := "No conflicts found."
	s := ScoreConflicts(testManifest, report)
	if s.TruePositives != 0 {
		t.Errorf("expected 0 TPs, got %d", s.TruePositives)
	}
	if s.FalseNegatives != 2 {
		t.Errorf("expected 2 FNs (T-1, T-3), got %d", s.FalseNegatives)
	}
	if s.F1 != 0 {
		t.Errorf("expected F1=0, got %.2f", s.F1)
	}
}

func TestScoreConflicts_FalsePositive(t *testing.T) {
	// T-2 (no conflict expected) mentioned as conflicting
	report := "T-2 and T-1 both conflict."
	s := ScoreConflicts(testManifest, report)
	if s.FalsePositives == 0 {
		t.Errorf("expected at least 1 FP (T-2 mentioned but clean), got 0")
	}
}

func TestBuildDetails_Counts(t *testing.T) {
	dag := map[string][]string{"T-1": {}, "T-2": {"T-1"}, "T-3": {}, "T-4": {"T-1", "T-2"}}
	report := "T-1 violates ADR. T-3 also violates ADR."
	details := BuildDetails(testManifest, dag, report)
	if len(details) != len(testManifest.Tasks) {
		t.Fatalf("expected %d detail entries, got %d", len(testManifest.Tasks), len(details))
	}
}

func TestRound2(t *testing.T) {
	cases := []struct{ in, want float64 }{
		{0.12345, 0.12},
		{0.999, 1.0},
		{0.0, 0.0},
		{0.005, 0.01},
	}
	for _, c := range cases {
		got := Round2(c.in)
		if got != c.want {
			t.Errorf("Round2(%.5f) = %.5f, want %.5f", c.in, got, c.want)
		}
	}
}

func TestEvalMockLLM_PerfectScores(t *testing.T) {
	m := &EvalMockLLM{Manifest: testManifest}
	ctx := testCtx()

	tasks := TasksToCoreTasks(testManifest.Tasks)

	dag, err := m.GenerateDAG(ctx, tasks)
	if err != nil {
		t.Fatalf("GenerateDAG error: %v", err)
	}
	dagScore := ScoreDAG(testManifest, dag)
	if dagScore.F1 != 1.0 {
		t.Errorf("EvalMockLLM DAG F1 should be 1.0, got %.2f", dagScore.F1)
	}

	report, err := m.AnalyzeConflict(ctx, tasks, nil)
	if err != nil {
		t.Fatalf("AnalyzeConflict error: %v", err)
	}
	conflictScore := ScoreConflicts(testManifest, report)
	if conflictScore.FalseNegatives != 0 {
		t.Errorf("EvalMockLLM should have 0 FNs, got %d", conflictScore.FalseNegatives)
	}
	if conflictScore.TruePositives != 2 {
		t.Errorf("EvalMockLLM should have 2 TPs, got %d", conflictScore.TruePositives)
	}
}
