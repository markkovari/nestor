package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/markkovari/nestor/internal/core"
)

// parseConflictReport parses a raw LLM JSON string into a ConflictReport.
// Strips markdown fences and falls back gracefully on bad JSON.
func parseConflictReport(raw string) *core.ConflictReport {
	raw = stripMarkdown(raw)
	var report core.ConflictReport
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		return &core.ConflictReport{Summary: raw, Findings: []core.ConflictFinding{}}
	}
	if report.Findings == nil {
		report.Findings = []core.ConflictFinding{}
	}
	return &report
}

// parseDAG parses a raw LLM JSON string into a task dependency map.
func parseDAG(raw string) (map[string][]string, error) {
	raw = stripMarkdown(raw)
	var dag map[string][]string
	if err := json.Unmarshal([]byte(raw), &dag); err != nil {
		return nil, fmt.Errorf("failed to parse DAG JSON: %w (raw: %.200s)", err, raw)
	}
	return dag, nil
}

func stripMarkdown(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
