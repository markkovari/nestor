# ADR-003: Pluggable LLM Architecture

## Status
Accepted

## Context
Determining if natural language task descriptions contradict each other or break architectural rules requires deep semantic understanding. LLM providers vary in cost, performance, and privacy.

## Decision
We will implement an `LLMAdapter` interface to allow for pluggable LLM backends.

## Rationale
- **No Vendor Lock-in:** Users can switch between Gemini, OpenAI, Anthropic, or local models (via Ollama/Llama.cpp).
- **Privacy:** Organizations can use local or VPC-hosted models for sensitive code/task data.
- **Future Proofing:** New and better models can be integrated by simply implementing a new adapter.

## Consequences
- We need to define a clean interface for prompt execution and structured output parsing.
- Integration tests will need to support mock LLM responses.
