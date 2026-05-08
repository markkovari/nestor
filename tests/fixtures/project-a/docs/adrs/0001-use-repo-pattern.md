---
date: 2026-05-01
reason: Standardizing database access
tldr: All database access must go through the repository pattern
version: 1.0
action: creation
---

# ADR-0001: Use Repository Pattern

## Status
Accepted

## Decision
Direct database calls are forbidden in the business logic layer.
