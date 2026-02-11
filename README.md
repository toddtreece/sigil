# Grafana Sigil

<img src="./logo.png" alt="Grafana Sigil logo" width="280" />

Sigil is an OSS AI Observability project from Grafana.

Prompt-as-spell metaphor: you cast a sigil, then observe what happened.

## Bootstrap Architecture

- Grafana app plugin (`/apps/plugin`) for conversations, completions, traces, and settings.
- Go service (`/api`) for ingest and query:
  - OTLP gRPC `:4317`
  - OTLP HTTP `:4318/v1/traces`
  - Records API and query API on `:8080`
- Tempo (docker compose) as trace storage.
- MySQL as default metadata and record-reference storage.
- Optional MinIO profile for object storage-backed payloads.
- SDKs (`/sdks`) with Go started first, Python/JS scaffolds present.

## Repository Layout

- `/apps/plugin`: Grafana app plugin and backend proxy layer.
- `/api`: Sigil ingest/query service skeleton.
- `/sdks/go`: Go SDK bootstrap.
- `/sdks/python`: Python SDK skeleton.
- `/sdks/js`: JS SDK skeleton.
- `/docs`: architecture, vibe, and backlog docs.

## Quickstart

```bash
mise trust
mise install
mise run deps
mise run up
```

Grafana is available at [http://localhost:3000](http://localhost:3000).

To run with optional MinIO profile:

```bash
mise run up:object
```

## Current Scope

This repository is in bootstrap phase: contracts, folder structure, and dev stack are set up, while implementation logic remains intentionally minimal.
