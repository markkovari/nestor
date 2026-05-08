# Nestor Architecture

## Objective
Design and build "Nestor", a Go-based CLI tool and unified server that analyzes GitHub repositories alongside external task trackers (Jira, Linear) and documentation (Figma, Notion). Nestor will identify potential code breakages, resolve semantic contradictions between current and upcoming tasks, and generate a Directed Acyclic Graph (DAG) for task parallelization.

## Background & Motivation
Modern software development involves scattered context across source control, issue trackers, and documentation platforms. Developers often start tasks without realizing they contradict upcoming work, or violate existing Architectural Decision Records (ADRs). Nestor acts as an intelligent overseer, ensuring alignment between intent (tasks/docs) and execution (code).

## Scope & Impact
- **Inputs:** GitHub, Jira, Linear, Figma, Notion.
- **Outputs:** Dependency DAGs, conflict alerts, CLI reports.
- **Storage:** Multi-model database (SurrealDB or Postgres with Graph/Vector extensions) to store a knowledge graph of tasks, code metadata, and vector embeddings for semantic search.
- **Analysis:** LLM-based reasoning engine to infer dependencies and conflicts.
- **Triggers:** Unified binary capable of running one-off CLI commands (`nestor check`) or a persistent server (`nestor serve`) for webhooks, cron, and MCP integrations.

## Proposed Solution (Architecture)

### 1. Core Components
- **CLI / Server Gateway (Cobra/Viper):** Handles command-line invocation, launches the HTTP server for webhooks and MCP.
- **Central Configuration Engine:** A robust, declarative configuration system (via Viper/YAML) allowing users to easily enable/disable adapters, configure sync intervals, and swap LLM providers without code changes.
- **Integration Adapters (Interfaces):**
  - `SourceAdapter`: GitHub, GitLab.
  - `TaskAdapter`: Jira, Linear.
  - `DocAdapter`: Notion, Figma.
- **Knowledge Base (Multi-Model DB):**
  - Using **SurrealDB** (or Postgres) to store entities as a Graph (e.g., `Task -> Blocks -> Task`, `Task -> Modifies -> CodeComponent`).
  - Storing vector embeddings of task descriptions and ADRs for semantic comparison.
- **Reasoning Engine (Pluggable LLM):**
  - Fetches context from the Knowledge Base.
  - An `LLMAdapter` interface allowing the backend to be swapped seamlessly (e.g., Gemini, OpenAI, Anthropic, Local Models).
  - Analyzes diffs, task descriptions, and ADRs to output structured data (Conflicts, Dependencies).

### 2. Data Flow
1. **Ingestion:** Webhook triggers or periodic cron jobs pull updates from Jira/GitHub based on configuration.
2. **Graph Construction:** Nestor updates the DB, generating embeddings for new text and mapping explicit relationships (e.g., Epic -> Ticket).
3. **Inference:** The configured LLM analyzes the graph to discover *implicit* relationships (e.g., Ticket A contradicts Ticket B, or Ticket C breaks ADR-001).
4. **DAG Generation:** Nestor queries the graph to generate an execution DAG, highlighting tasks that can be parallelized.
