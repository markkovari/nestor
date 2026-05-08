# ADR-002: Multi-Model Database for Knowledge Graph

## Status
Accepted

## Context
Nestor needs to store complex relationships between tasks (e.g., "blocks", "depends on"), code components, and architectural decisions. We also need to perform semantic searches using vector embeddings.

## Decision
We will use a multi-model database that supports both Graph and Vector operations. Our primary choice is **SurrealDB**.

## Rationale
- **Graph Support:** SurrealDB natively supports graph relations, which is essential for building and querying the task DAG.
- **Vector Search:** It provides integrated vector indexing for semantic similarity checks.
- **Flexibility:** Being multi-model, it can handle document storage for task metadata alongside the graph.
- **Alternative:** If SurrealDB proves unsuitable, we will use Postgres with `pgvector` and `Apache AGE` (or recursive CTEs for graph traversal).

## Consequences
- Requires running a database instance or using an embedded version if available/stable for Go.
- Ingestion logic must maintain both the graph structure and vector embeddings.
