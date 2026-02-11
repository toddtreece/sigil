# Sigil Architecture (Bootstrap)

## Data Flow

1. Application instruments post-LLM calls via Sigil SDK helpers.
2. OTLP traces are sent to Sigil ingest (`:4317`, `:4318/v1/traces`).
3. Large payloads can be externalized into Records API storage and represented by `sigil://record/{id}` references.
4. Trace payloads are forwarded to Tempo.
5. Grafana plugin backend proxies query requests to Sigil service APIs.
6. Plugin UI renders conversations, completions, and trace-linked records.

## Contracts

- OTLP gRPC: `:4317`
- OTLP HTTP traces: `:4318/v1/traces`
- Records API:
  - `POST /api/v1/records`
  - `GET /api/v1/records/{record_id}`
- Query API:
  - `GET /api/v1/conversations`
  - `GET /api/v1/conversations/{conversation_id}`
  - `GET /api/v1/completions`
  - `GET /api/v1/traces/{trace_id}`

## Span Externalization Attributes

When payloads are externalized, the ingest path uses these attributes:

- `sigil.payload_externalized=true`
- `sigil.record_ids=["<record-id>"]`

and payload references use:

- `sigil://record/{id}`

## Services

- `apps/plugin`: Grafana app plugin and backend resource handlers.
- `api`: ingest/query service with storage abstractions.
- `tempo`: trace backend.
- `mysql`: metadata store.
- `minio` (optional profile): object payload backing store.
