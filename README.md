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

### Example Configuration (`nestor.yaml.example`)
```yaml
database:
  url: "ws://localhost:8000"
  ns: "nestor"
  db: "nestor"
  user: "root"
  password: "root"

llm:
  provider: "gemini" # options: gemini, openai, ollama, mock
  api_key: "YOUR_API_KEY"
  model: "gemini-1.5-pro"

adapters:
  github:
    token: "YOUR_GITHUB_TOKEN"
    repos: ["owner/repo"]
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
