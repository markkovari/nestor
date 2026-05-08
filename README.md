# Nestor

Nestor is an intelligent project checker and task dependency analyzer. It analyzes your codebase, architectural decisions (ADRs), and task trackers (GitHub Issues, Linear, Jira) to detect conflicts and build a Directed Acyclic Graph (DAG) of your project's tasks.

## 🚀 Quick Start

1. **Clone the repository**
2. **Install dependencies**: `go mod tidy`
3. **Configure Nestor**:
   ```bash
   cp nestor.yaml.example nestor.yaml
   # Edit nestor.yaml with your API keys and repository paths
   ```
4. **Run Analysis**:
   ```bash
   go run cmd/nestor/main.go check
   ```

## 🛠 Configuration

Nestor uses a `nestor.yaml` file for configuration. It looks for this file in the current directory or in `$HOME/.nestor/`.

### Environment Variables

You can override any configuration value using environment variables with the `NESTOR_` prefix. Nested keys are separated by underscores.

- `NESTOR_DATABASE_URL` maps to `database.url`
- `NESTOR_LLM_API_KEY` maps to `llm.api_key`
- `NESTOR_ADAPTERS_GITHUB_TOKEN` maps to `adapters.github.token`

Example:
```bash
NESTOR_LLM_API_KEY=your_key_here ./nestor check
```

## 🏗 Architecture & ADRs

Architectural decisions are stored in `docs/adrs/`. These records are used by the LLM to verify that new tasks do not contradict established patterns.

To add a new ADR:
```bash
go run cmd/nestor/main.go adr add "Use Postgres for storage" -r "Scalability"
```

## 🧪 Testing

Run the verification suites to see Nestor in action with mock data:
- **Simple**: `go run cmd/verify/main.go`
- **Multi-Repo**: `go run cmd/verify_complex/main.go`

## 📖 Documentation
- [Architecture Overview](docs/architecture.md)
- [Architecture Decision Records](docs/adrs/)
