package etalon

import (
	"math"
	"strings"
)

// ScoreDAG compares the LLM-generated DAG against expected_dag in manifest
func ScoreDAG(manifest *Manifest, gotDAG map[string][]string) DAGScore {
	var totalExpected, truePos, falsePos int

	for taskID, expectedDeps := range manifest.ExpectedDAG {
		gotDeps := gotDAG[taskID]
		for _, dep := range expectedDeps {
			totalExpected++
			if containsStr(gotDeps, dep) {
				truePos++
			}
		}
		for _, dep := range gotDeps {
			if !containsStr(expectedDeps, dep) {
				falsePos++
			}
		}
	}

	precision := safeDiv(float64(truePos), float64(truePos+falsePos))
	recall := safeDiv(float64(truePos), float64(totalExpected))
	f1 := safeF1(precision, recall)

	return DAGScore{
		TotalPairs:   totalExpected,
		CorrectPairs: truePos,
		Precision:    Round2(precision),
		Recall:       Round2(recall),
		F1:           Round2(f1),
	}
}

// ScoreConflicts checks whether the conflict report covers expected conflict tasks.
// Matching strategy: task ID OR any 3+ consecutive words from the task title appear in the report.
// This handles both fixture (ID-based) and real LLM output (content-based).
func ScoreConflicts(manifest *Manifest, conflictReport string) ConflictScore {
	var tp, fp, fn int
	lower := strings.ToLower(conflictReport)

	for _, t := range manifest.Tasks {
		mentioned := taskMentioned(lower, t)
		if t.ExpectConflicts && mentioned {
			tp++
		} else if !t.ExpectConflicts && mentioned {
			fp++
		} else if t.ExpectConflicts && !mentioned {
			fn++
		}
	}

	precision := safeDiv(float64(tp), float64(tp+fp))
	recall := safeDiv(float64(tp), float64(tp+fn))
	f1 := safeF1(precision, recall)

	return ConflictScore{
		TruePositives:  tp,
		FalsePositives: fp,
		FalseNegatives: fn,
		Precision:      Round2(precision),
		Recall:         Round2(recall),
		F1:             Round2(f1),
	}
}

// BuildDetails produces per-task breakdown
func BuildDetails(manifest *Manifest, gotDAG map[string][]string, conflictReport string) []TaskEvalDetail {
	lower := strings.ToLower(conflictReport)
	var details []TaskEvalDetail
	for _, t := range manifest.Tasks {
		gotDeps := gotDAG[t.ID]
		expectedDeps := manifest.ExpectedDAG[t.ID]
		mentioned := taskMentioned(lower, t)
		details = append(details, TaskEvalDetail{
			TaskID:           t.ID,
			ExpectedConflict: t.ExpectConflicts,
			GotConflict:      mentioned,
			ExpectedDeps:     expectedDeps,
			GotDeps:          gotDeps,
			ConflictCorrect:  t.ExpectConflicts == mentioned,
			DepsCorrect:      depsEqual(expectedDeps, gotDeps),
		})
	}
	return details
}

// taskMentioned returns true if the report text references the task by ID or by title keywords.
// Title match: any 3+ consecutive lowercase words from the title appear as a substring.
func taskMentioned(reportLower string, t EtalonTask) bool {
	if strings.Contains(reportLower, strings.ToLower(t.ID)) {
		return true
	}
	words := strings.Fields(strings.ToLower(t.Title))
	for i := 0; i+2 < len(words); i++ {
		phrase := strings.Join(words[i:i+3], " ")
		if strings.Contains(reportLower, phrase) {
			return true
		}
	}
	return false
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if strings.EqualFold(v, s) {
			return true
		}
	}
	return false
}

func depsEqual(expected, got []string) bool {
	if len(expected) != len(got) {
		return false
	}
	for _, e := range expected {
		if !containsStr(got, e) {
			return false
		}
	}
	return true
}

func safeDiv(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

func safeF1(p, r float64) float64 {
	if p+r == 0 {
		return 0
	}
	return 2 * p * r / (p + r)
}

// Round2 rounds a float to 2 decimal places
func Round2(f float64) float64 {
	return math.Round(f*100) / 100
}
