# ADR-004: Unified CLI and Server Architecture

## Status
Accepted

## Context
Nestor needs to support various trigger mechanisms: manual checks (CLI), periodic runs (Cron), and real-time updates (Webhooks/PubSub).

## Decision
Nestor will be a single binary that can operate in two modes: `check` (one-off CLI execution) and `serve` (long-lived server).

## Rationale
- **Deployment Simplicity:** A single binary simplifies CI/CD pipelines and local installation.
- **Shared Logic:** Both modes share the same configuration, database connections, and reasoning engine.
- **Extensibility:** The server mode can easily expose an MCP (Model Context Protocol) server or a REST API.

## Consequences
- The CLI must handle subcommands gracefully (using Cobra).
- The server mode requires proper lifecycle management and logging.
