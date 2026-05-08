# ADR-001: Use Go as the Primary Language

## Status
Accepted

## Context
We need a language that is well-suited for building both CLI tools and long-lived server processes. The project requires high performance, excellent concurrency support (for parallel ingestion), and easy cross-platform distribution.

## Decision
We will use Go as the primary programming language for Nestor.

## Rationale
- **Single Binary:** Go compiles to a single static binary, making it easy to distribute for CLI usage.
- **Concurrency:** Goroutines and channels provide a robust model for handling multiple API integrations and background sync tasks.
- **Standard Library:** Go's standard library has excellent support for HTTP servers and networking.
- **Ecosystem:** Strong libraries exist for CLI (Cobra/Viper) and various API integrations.

## Consequences
- The team must be proficient in Go.
- We benefit from fast build times and high runtime performance.
