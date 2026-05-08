package etalon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"github.com/markkovari/nestor/internal/core"
)

// LoadManifest reads etalon.json from the given directory
func LoadManifest(dir string) (*Manifest, error) {
	data, err := os.ReadFile(filepath.Join(dir, "etalon.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to read etalon.json: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to parse etalon.json: %w", err)
	}
	return &m, nil
}

// LoadADRContents reads the ADR files listed in the manifest and returns their text
func LoadADRContents(dir string, manifest *Manifest) ([]string, error) {
	var adrs []string
	for _, rel := range manifest.ADRs {
		path := filepath.Join(dir, rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read ADR %s: %w", rel, err)
		}
		adrs = append(adrs, string(data))
	}
	return adrs, nil
}

// TasksToCoreTasks converts etalon tasks to []core.Task for use with engine/LLM
func TasksToCoreTasks(tasks []EtalonTask) []core.Task {
	result := make([]core.Task, len(tasks))
	for i, t := range tasks {
		result[i] = core.Task{
			ID:          t.ID,
			Title:       t.Title,
			Description: t.Description,
			Status:      t.Status,
			Provider:    "etalon",
		}
	}
	return result
}

// LoadGitHubEtalon fetches issues from a GitHub repo and returns them as EtalonTasks.
// Used for --live mode. Expects repo in "owner/repo" format.
func LoadGitHubEtalon(token, repo string) ([]EtalonTask, error) {
	// Implemented via gh CLI to avoid circular deps; returns tasks only (no ground truth)
	// For now return an error — live mode requires manual etalon.json in the repo
	return nil, fmt.Errorf("live GitHub loading not yet implemented: clone the etalon repo and use --etalon-dir")
}

