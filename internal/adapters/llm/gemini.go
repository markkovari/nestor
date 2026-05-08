package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/markkovari/nestor/internal/core"
	"google.golang.org/api/option"
)

type GeminiProvider struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

func NewGeminiProvider(ctx context.Context, apiKey, modelName string) (*GeminiProvider, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}
	model := client.GenerativeModel(modelName)
	return &GeminiProvider{
		client: client,
		model:  model,
	}, nil
}

func (g *GeminiProvider) AnalyzeConflict(ctx context.Context, tasks []core.Task, adrs []string) (string, error) {
	var sb strings.Builder
	sb.WriteString("Analyze the following tasks for potential conflicts or contradictions with the provided ADRs:\n\n")
	
	sb.WriteString("ADRs:\n")
	for _, adr := range adrs {
		sb.WriteString(fmt.Sprintf("- %s\n", adr))
	}

	sb.WriteString("\nTasks:\n")
	for _, t := range tasks {
		sb.WriteString(fmt.Sprintf("[%s]: %s - %s\n", t.ID, t.Title, t.Description))
	}

	sb.WriteString("\nIdentify any architectural violations or logical contradictions between tasks. Return a concise report.")

	resp, err := g.model.GenerateContent(ctx, genai.Text(sb.String()))
	if err != nil {
		return "", err
	}

	return formatResponse(resp), nil
}

func (g *GeminiProvider) GenerateDAG(ctx context.Context, tasks []core.Task) (map[string][]string, error) {
	var sb strings.Builder
	sb.WriteString("Analyze these tasks and determine their dependencies. Return ONLY a JSON map where keys are Task IDs and values are arrays of Task IDs they depend on.\n\n")
	
	for _, t := range tasks {
		sb.WriteString(fmt.Sprintf("[%s]: %s - %s\n", t.ID, t.Title, t.Description))
	}

	resp, err := g.model.GenerateContent(ctx, genai.Text(sb.String()))
	if err != nil {
		return nil, err
	}

	raw := formatResponse(resp)
	// Strip markdown blocks if present
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var dag map[string][]string
	if err := json.Unmarshal([]byte(raw), &dag); err != nil {
		return nil, fmt.Errorf("failed to parse DAG JSON: %w (raw output: %s)", err, raw)
	}

	return dag, nil
}

func formatResponse(resp *genai.GenerateContentResponse) string {
	var parts []string
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				parts = append(parts, fmt.Sprint(part))
			}
		}
	}
	return strings.Join(parts, "\n")
}
