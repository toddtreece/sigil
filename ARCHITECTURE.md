---
owner: sigil-core
status: active
last_reviewed: 2026-02-12
source_of_truth: true
audience: both
---

# Sigil Architecture

## System Boundaries

- `apps/plugin`: plugin UI and backend proxy for Sigil query APIs.
- `api`: OTEL trace ingest, generation ingest (custom API), and query APIs.
- `sdks/*`: manual post-LLM instrumentation helpers.
- Tempo: trace storage backend.
- MySQL: metadata/query backing store (incremental rollout).
- MinIO: optional object backend profile.

## Ingest Model (Generation-First)

- Trace pipeline:
  - SDK exports traces via OTLP (`grpc` or `http`) using built-in SDK trace config.
  - API exposes OTLP gRPC (`:4317`) and OTLP HTTP traces (`/v1/traces`) and forwards to Tempo.
- Generation pipeline:
  - SDK exports normalized generations to Sigil custom ingest.
  - Primary transport is gRPC with HTTP parity.
  - HTTP and gRPC transports both call the same internal generation exporter path.
  - Export is buffered/batched asynchronously in SDK; `Shutdown(ctx)` is required to flush.

## Data Flow

1. App code wraps provider calls via Sigil SDK.
2. SDK records normalized generation payload at `End()`.
3. SDK enqueues generation into bounded async queue and exports in batches.
4. API generation ingest validates each item and persists accepted payloads.
5. SDK emits traces to OTLP ingest.
6. API forwards OTLP traces to Tempo.
7. Plugin queries Sigil query APIs for conversation/completion/trace views.

## API Contracts

- OTLP gRPC traces: `:4317` (`opentelemetry.proto.collector.trace.v1.TraceService/Export`)
- OTLP HTTP traces: `POST /v1/traces`
- Generation ingest gRPC service: `sigil.v1.GenerationIngestService.ExportGenerations`
- Generation ingest HTTP parity: `POST /api/v1/generations:export`
- Query API:
  - `GET /api/v1/conversations`
  - `GET /api/v1/conversations/{conversation_id}`
  - `GET /api/v1/completions`
  - `GET /api/v1/traces/{trace_id}`

## Generation Contract

- Generation mode is explicit:
  - `SYNC`: non-stream provider flows
  - `STREAM`: streaming provider flows
- Normalized fields are always sent:
  - model/system prompt/input/output/tools/usage/metadata/timestamps/tags
- Optional identity fields are supported end-to-end:
  - `conversation_id`
  - `agent_name`
  - `agent_version`
- Raw artifacts are optional debug payloads and default OFF.

## SDK Runtime Contracts

- `rec.Err()` reports local validation/enqueue failures only.
- Background export failures are retried with backoff and emitted via logs.
- `Client.Shutdown(ctx)` flushes generation batches and trace provider.
- `Client.Flush(ctx)` is available for explicit manual flush points.

## Removed Runtime Paths

- Records REST endpoints are removed from active runtime.
- Records-first artifact externalization is no longer the primary ingest path.

## Service Responsibilities

- `apps/plugin`: UI and backend proxy handlers.
- `api/internal/ingest`: OTLP HTTP trace ingest and Tempo forwarding.
- `api/internal/generations`: generation ingest validation + persistence abstraction.
- `api/internal/query`: read/query surfaces for plugin.
