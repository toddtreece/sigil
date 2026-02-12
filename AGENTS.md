# Repository Guidelines

## Architecture

- `apps/plugin`: Grafana plugin UI + backend proxy for Sigil query APIs.
- `api`: OTLP trace ingest + generation ingest + query APIs.
- `sdks/*`: manual post-LLM instrumentation helpers.

## Runtime Workflow (Compose-First)

- Keep `docker compose` running while developing; rely on hot reload.
- Use `docker compose logs -f <service>` for debugging.
- Check bind mounts and service commands in `docker-compose.yaml` before restarting containers.

## Coding Defaults

- Prefer simple, explicit code over clever abstractions.
- Add comments only where logic is non-obvious.
- Every bug fix includes a regression test.
- No hacks.

## Go Rules

- Prefer idiomatic Go and standard library first.
- Use `context.Context` only for cancellation, deadlines, tracing, and request-scoped metadata.
- Never pass runtime dependencies through `context.Context`.
- Define interfaces where they are consumed, not where they are implemented.
- Keep packages loosely coupled; avoid heavy cross-package implementation ties.
- Add tests where they improve confidence.
- Prefer table-driven tests when many similar cases exist.

## TypeScript / React Rules

- Keep KISS and DRY.
- Build small focused components.
- Always use explicit types for props, state, and API contracts.
- Keep plugin code under `apps/plugin/src`.
- Use `getBackendSrv().fetch()` for plugin-to-backend calls.

## Generation Ingest Contract

- Custom ingest is generation-first.
- Use `GenerationIngestService.ExportGenerations` (gRPC) or `POST /api/v1/generations:export` (HTTP parity).
- Provider wrappers must set generation mode explicitly:
  - non-stream: `SYNC`
  - stream: `STREAM`
- Raw provider artifacts are default OFF; enable only via explicit debug opt-in.

## Docs Map

- Canonical architecture: `ARCHITECTURE.md`
- Canonical docs index: `docs/index.md`
- Design docs index: `docs/design-docs/index.md`
- Product specs index: `docs/product-specs/index.md`
- Execution plans:
  - active: `docs/exec-plans/active/`
  - completed: `docs/exec-plans/completed/`
  - tech debt: `docs/exec-plans/tech-debt-tracker.md`
- Generation ingest reference: `docs/references/generation-ingest-contract.md`
- Benchmark + OTEL references: `docs/references/competitive-benchmark.md`

## When To Update Docs

| Repo Area | Update Triggers | Required Docs |
| --- | --- | --- |
| `apps/plugin` | query/proxy contract changes, UI behavior or data model changes | `ARCHITECTURE.md`, `docs/FRONTEND.md`, relevant plan/spec doc |
| `api` | ingest/query endpoint changes, proto changes, validation/storage behavior changes | `ARCHITECTURE.md`, `docs/references/generation-ingest-contract.md`, relevant exec plan |
| `sdks/*` | config/lifecycle changes, generation schema changes, provider wrapper behavior changes | `ARCHITECTURE.md`, `sdks/go/README.md`, relevant plan/reference docs |

## Quality Commands

- `mise run format`
- `mise run lint`
- `mise run check`

## Commit Standard

- Use Conventional Commits.
- Explain both what changed and why in commit body.
