---
owner: sigil-core
status: completed
last_reviewed: 2026-02-19
source_of_truth: true
audience: both
---

# Phase 2 Workstream Delivery: Query Proxy and Envelope

## Goal

Deliver plugin-proxy-only query routing with Grafana-compatible query envelopes and frame contracts.

## Scope

- Endpoint-specific Sigil API query contracts (conversations/generations/search and downstream proxy prefixes).
- Plugin backend resource proxy contracts under `/api/plugins/grafana-sigil-app/resources/query/...`.
- Query response envelope and frame compatibility requirements for metrics and traces.

## Source Design Doc

- `docs/design-docs/2026-02-12-phase-2-query-proxy.md`

## Tasks

- [x] Add Sigil pass-through query proxy routes for Prometheus/Mimir and Tempo under `/api/v1/proxy/{backend}/...`.
- [x] Enforce allowlisted read/query downstream paths and reject non-allowlisted paths/methods at Sigil boundary.
- [x] Forward downstream `X-Scope-OrgID` from Sigil tenant context (auth tenant or fake-tenant mode).
- [x] Add local tests for proxy routing, allowlist/method enforcement, tenant propagation, and upstream failure handling.
- [x] Wire plugin backend resource routes to consume Sigil proxy endpoints via `/query/proxy/prometheus/...` and `/query/proxy/tempo/...`; legacy plugin `query/traces/{trace_id}` route removed.
- [x] Define endpoint-specific Sigil query contracts and proxy prefixes.
- [x] Define endpoint-specific plugin resource proxy contracts under `/api/plugins/grafana-sigil-app/resources/query/...`.
- [x] Require frontend query calls to use plugin proxy only.
- [x] Require `QueryDataResponse` envelope with `results.<refId>.frames`.
- [x] Define metrics frame compatibility requirements (time/value + metadata for graph/table).
- [x] Define trace detail frame compatibility requirements (`preferredVisualisationType: trace` + trace fields/meta shape).
- [x] Define trace search frame compatibility requirements (Tempo/Grafana table conventions).
- [x] Anchor compatibility requirements to `docs/references/grafana-query-response-shapes.md`.
- [x] Document required local test scenarios:
  - response serialization as `QueryDataResponse`
  - frame compatibility for metrics and traces
  - plugin frontend path enforcement through resource proxy only
- [x] Decision (2026-02-19): de-scope generic `POST /api/v1/query` and `POST /api/plugins/grafana-sigil-app/resources/query` from Phase 2; keep endpoint-specific query/resource routes as the canonical contract.
- [x] Record this decision in design/reference docs and execution plan indexes.

## Risks

- Frontend bypass of plugin proxy breaks tenant and deployment boundaries.
- Envelope/frame mismatch causes Grafana panel incompatibility.
- Contract drift between plugin backend and API query handlers.

## Exit Criteria

- Query access path is plugin-proxy-only by contract.
- Query responses are explicitly Grafana-compatible for metrics and trace workflows.
- Local tests assert proxy-path and envelope/frame compatibility requirements.

## Out of Scope

- Non-REST query transport in this phase.
- Generic catch-all query endpoints (`POST /api/v1/query` and `POST /api/plugins/grafana-sigil-app/resources/query`) in this phase.
