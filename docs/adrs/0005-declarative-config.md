# ADR-005: Declarative, Centralized Configuration

## Status
Accepted

## Context
Nestor integrates with many external services and requires specific settings for database, LLMs, and sync intervals. These should be easily adjustable by the user.

## Decision
All configuration will be handled via a central, declarative configuration file (YAML/JSON/TOML) managed by Viper.

## Rationale
- **Ease of Use:** Users can configure all integrations in one place.
- **Flexibility:** Viper supports environment variable overrides, making it cloud-native and CI-friendly.
- **Decoupling:** Implementation logic is decoupled from specific credentials or service endpoints.

## Consequences
- A robust configuration schema must be defined and validated at startup.
- Documentation must clearly explain all available configuration options.
