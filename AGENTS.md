# Repository Guidelines

## Architecture Overview

- **Grafana Plugin** (`apps/plugin`): app plugin UI and backend proxy endpoints for Sigil query APIs.
- **Sigil API** (`api`): OTLP ingest, Records API, and query endpoints.
- **SDKs** (`sdks/*`): manual post-LLM instrumentation helpers.

## Development

This repository uses `mise` as task runner and `pnpm` for workspace dependencies.

- Start stack: `mise run up`
- Start stack with object storage profile: `mise run up:object`
- Stop stack: `mise run down`
- Plugin dev only: `mise run dev:plugin`
- API dev only: `mise run dev:api`

## Coding Conventions

### TypeScript / Plugin

- Keep plugin code under `apps/plugin/src`.
- Use `getBackendSrv().fetch()` for plugin-to-backend calls.
- Avoid template boilerplate pages once replaced by Sigil pages.

### Go / Services

- Keep HTTP/ingest/query/storage concerns split in `api/internal`.
- Favor explicit interfaces for storage and transport abstractions.
- Add tests for new handlers and contract edge cases.

## Scope Guardrails

- Tempo runs in compose; do not add custom trace DB logic in bootstrap phase.
- Use OTEL/OTLP conventions for GenAI attributes as baseline.
- SDKs are manual instrumentation helpers, not provider monkeypatch wrappers.

## NO HACKS

Prioritize clear, maintainable scaffolding over quick hacks.
