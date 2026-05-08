package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/jxnl/instructor-go/pkg/instructor"
	"github.com/markkovari/nestor/internal/core"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/googleai"
	"github.com/tmc/langchaingo/llms/ollama"
	lcOpenAI "github.com/tmc/langchaingo/llms/openai"
	newgenai "google.golang.org/genai"
)

// Provider wraps any langchaingo LLM backend with instructor-go structured extraction.
// Supports: gemini, openai, ollama, anthropic — selected via cfg.LLM.Provider.
type Provider struct {
	lc        llms.Model
	ins       *instructor.InstructorGoogle // non-nil only for gemini
	modelName string
}

// NewProvider constructs an LLMProvider from config values.
// provider: "gemini" | "openai" | "ollama" | "anthropic"
// baseURL is optional: Ollama server URL or OpenAI-compatible endpoint.
func NewProvider(ctx context.Context, providerName, apiKey, model, baseURL string) (*Provider, error) {
	var lc llms.Model
	var ins *instructor.InstructorGoogle
	var err error

	switch strings.ToLower(providerName) {
	case "gemini", "":
		opts := []googleai.Option{googleai.WithAPIKey(apiKey)}
		if model != "" {
			opts = append(opts, googleai.WithDefaultModel(model))
		}
		lc, err = googleai.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("googleai init: %w", err)
		}
		genaiClient, gerr := newgenai.NewClient(ctx, &newgenai.ClientConfig{
			APIKey:  apiKey,
			Backend: newgenai.BackendGeminiAPI,
		})
		if gerr != nil {
			return nil, fmt.Errorf("genai client init: %w", gerr)
		}
		ins = instructor.FromGoogle(genaiClient,
			instructor.WithMode(instructor.ModeJSON),
			instructor.WithMaxRetries(3),
		)

	case "openai":
		opts := []lcOpenAI.Option{lcOpenAI.WithToken(apiKey)}
		if model != "" {
			opts = append(opts, lcOpenAI.WithModel(model))
		}
		if baseURL != "" {
			opts = append(opts, lcOpenAI.WithBaseURL(baseURL))
		}
		lc, err = lcOpenAI.New(opts...)
		if err != nil {
			return nil, fmt.Errorf("openai init: %w", err)
		}

	case "ollama":
		opts := []ollama.Option{}
		if model != "" {
			opts = append(opts, ollama.WithModel(model))
		}
		if baseURL != "" {
			opts = append(opts, ollama.WithServerURL(baseURL))
		}
		lc, err = ollama.New(opts...)
		if err != nil {
			return nil, fmt.Errorf("ollama init: %w", err)
		}

	case "anthropic":
		opts := []anthropic.Option{anthropic.WithToken(apiKey)}
		if model != "" {
			opts = append(opts, anthropic.WithModel(model))
		}
		lc, err = anthropic.New(opts...)
		if err != nil {
			return nil, fmt.Errorf("anthropic init: %w", err)
		}

	default:
		return nil, fmt.Errorf("unknown llm provider %q — supported: gemini, openai, ollama, anthropic", providerName)
	}

	return &Provider{lc: lc, ins: ins, modelName: model}, nil
}

func (p *Provider) AnalyzeConflict(ctx context.Context, tasks []core.Task, adrs []string) (string, error) {
	prompt := buildConflictPrompt(tasks, adrs) +
		"\nIdentify any architectural violations or logical contradictions. Return a concise report."
	return llms.GenerateFromSinglePrompt(ctx, p.lc, prompt)
}

func (p *Provider) AnalyzeConflictStructured(ctx context.Context, tasks []core.Task, adrs []string) (*core.ConflictReport, error) {
	if p.ins != nil {
		prompt := "You are an architectural compliance checker. Analyze tasks against provided ADRs.\n" +
			"Return structured findings. Empty findings list if no violations.\n\n" +
			buildConflictPrompt(tasks, adrs)
		var report core.ConflictReport
		_, err := instructor.ChatHandler(p.ins, ctx, instructor.GoogleRequest{
			Model: p.modelName,
			Contents: []*newgenai.Content{
				{Role: "user", Parts: []*newgenai.Part{{Text: prompt}}},
			},
		}, &report)
		if err != nil {
			return nil, fmt.Errorf("structured conflict analysis: %w", err)
		}
		if report.Findings == nil {
			report.Findings = []core.ConflictFinding{}
		}
		return &report, nil
	}

	// Fallback for non-gemini: request JSON, parse manually
	prompt := buildConflictPrompt(tasks, adrs) + `

Return ONLY valid JSON, no markdown:
{"summary":"...","findings":[{"task_id":"","task_title":"","adr_ref":"","clause":"","reason":"","severity":"high|medium|low"}]}`
	raw, err := llms.GenerateFromSinglePrompt(ctx, p.lc, prompt)
	if err != nil {
		return nil, err
	}
	return parseConflictReport(raw), nil
}

func (p *Provider) GenerateDAG(ctx context.Context, tasks []core.Task) (map[string][]string, error) {
	prompt := buildTaskListPrompt(tasks)

	if p.ins != nil {
		type dagResult struct {
			Dependencies map[string][]string `json:"dependencies" jsonschema:"description=map of task ID to list of blocker task IDs"`
		}
		var result dagResult
		_, err := instructor.ChatHandler(p.ins, ctx, instructor.GoogleRequest{
			Model: p.modelName,
			Contents: []*newgenai.Content{
				{Role: "user", Parts: []*newgenai.Part{{Text: "Analyze tasks and determine dependencies.\n\n" + prompt}}},
			},
		}, &result)
		if err != nil {
			return nil, fmt.Errorf("DAG generation: %w", err)
		}
		if result.Dependencies == nil {
			return make(map[string][]string), nil
		}
		return result.Dependencies, nil
	}

	raw, err := llms.GenerateFromSinglePrompt(ctx, p.lc,
		"Analyze these tasks and determine dependencies. Return ONLY a JSON object: keys=Task IDs, values=arrays of blocker Task IDs. No markdown.\n\n"+prompt)
	if err != nil {
		return nil, err
	}
	return parseDAG(raw)
}

func (p *Provider) SuggestTaskUpdate(ctx context.Context, task core.Task, conflicts string) (string, error) {
	prompt := fmt.Sprintf(
		"Based on this conflict report, suggest an updated description for the task including a 'Nestor Analysis' section.\n\nConflict Report:\n%s\n\nTask: %s\nDescription: %s\n\nReturn ONLY the new description text.",
		conflicts, task.Title, task.Description,
	)
	return llms.GenerateFromSinglePrompt(ctx, p.lc, prompt)
}

func buildConflictPrompt(tasks []core.Task, adrs []string) string {
	var sb strings.Builder
	sb.WriteString("ADRs:\n")
	for _, adr := range adrs {
		sb.WriteString(adr)
		sb.WriteString("\n---\n")
	}
	sb.WriteString("\nTasks:\n")
	sb.WriteString(buildTaskListPrompt(tasks))
	return sb.String()
}

func buildTaskListPrompt(tasks []core.Task) string {
	var sb strings.Builder
	for _, t := range tasks {
		if prs := t.Metadata["linked_prs"]; prs != "" {
			fmt.Fprintf(&sb, "[%s] %s: %s (linked PRs: %s)\n", t.ID, t.Title, t.Description, prs)
		} else {
			fmt.Fprintf(&sb, "[%s] %s: %s\n", t.ID, t.Title, t.Description)
		}
	}
	return sb.String()
}
