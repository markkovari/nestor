package etalon

// PromptVariant defines a named variant of LLM prompting strategy
type PromptVariant struct {
	Name           string
	DAGPrefix      string // prepended to DAG generation prompt
	ConflictPrefix string // prepended to conflict analysis prompt
}

// DefaultVariants are the built-in prompt strategies to compare
var DefaultVariants = []PromptVariant{
	{
		Name:           "baseline",
		DAGPrefix:      "",
		ConflictPrefix: "",
	},
	{
		Name:           "explicit-chain-of-thought",
		DAGPrefix:      "Think step by step. First list all tasks, then for each task identify which other tasks it logically requires to be completed first.\n\n",
		ConflictPrefix: "Think step by step. For each task, check it against each ADR individually before writing your report.\n\n",
	},
	{
		Name:           "strict-json-first",
		DAGPrefix:      "Output ONLY valid JSON. No explanation. No markdown. No commentary.\n\n",
		ConflictPrefix: "Be strict and exhaustive. Flag every task that violates any ADR, even partially.\n\n",
	},
	{
		Name:           "conservative",
		DAGPrefix:      "Only include a dependency if you are highly confident it is required. When in doubt, omit it.\n\n",
		ConflictPrefix: "Only flag a conflict if it clearly and directly contradicts an ADR. Do not flag ambiguous cases.\n\n",
	},
}
