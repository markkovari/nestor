---
date: 2026-05-01
reason: Ensuring backward compatibility
tldr: All Backend API changes must include a versioned path (e.g., /v2/...)
version: 2.0
action: creation
---

# ADR-0002: Versioned API endpoints

## Status
Accepted

## Decision
New endpoints in `backend-api` must be prefixed with a version number.
No breaking changes to `/v1/` endpoints are allowed.
