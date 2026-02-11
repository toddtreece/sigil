# Sigil API

Bootstrap contracts exposed by the service:

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
