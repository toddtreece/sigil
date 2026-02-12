---
owner: sigil-core
status: active
last_reviewed: 2026-02-12
source_of_truth: true
audience: agents
---

# Bootstrap Phase 1

## Goal

Replace placeholder behavior with end-to-end functional ingest, query, and UI flows.

## Scope

- Sigil API data access and storage abstractions.
- Payload externalization flow for large content.
- Plugin pages backed by real APIs.
- Auth and tenant boundary scaffolding.

## Tasks

- [ ] Replace query placeholders with real data access to Tempo/MySQL.
- [x] Close Records-first externalization path in favor of Generation-first ingest (`docs/exec-plans/active/2026-02-12-generation-first-ingest.md`).
- [ ] Add plugin pages with real tables/details for conversations/completions/traces.
- [ ] Add auth and tenant boundaries for API and plugin proxy paths.
- [ ] Add schema migrations for generations and conversation metadata.

## Risks

- Scope expansion across plugin/API/SDK areas can slow delivery.
- Auth and tenancy decisions can block implementation details.
- Ingest/query contract drift can create incompatibilities between SDK, API, and plugin.

## Exit Criteria

- Query endpoints return persisted data instead of placeholders.
- Generation ingest APIs persist normalized generation payloads.
- Plugin pages show real conversation/completion/trace data.
- Auth boundaries are defined and enforced for exposed paths.
