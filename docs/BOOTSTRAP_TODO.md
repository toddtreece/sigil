# Bootstrap TODO

## Phase 0 (Done in this bootstrap)

- [x] Monorepo root with `mise`, `pnpm` workspace, compose stack.
- [x] Assistant-style plugin layout in `apps/plugin`.
- [x] Plugin backend query proxy scaffold.
- [x] Sigil API skeleton with OTLP + Records + query contracts.
- [x] SDK skeletons for Go, Python, JS.

## Phase 1 (Next)

- [ ] Replace query placeholders with real data access to Tempo/MySQL.
- [ ] Implement Records storage abstraction with MySQL + optional object backend.
- [ ] Add OTLP payload mutation flow: externalize large payloads and inject references.
- [ ] Add plugin pages with real tables/details views for conversations/completions/traces.
- [ ] Add auth/tenant boundaries for API and plugin proxy paths.
- [ ] Add schema migrations for records and conversation metadata.

## Phase 2

- [ ] Expand Go SDK with end-to-end examples.
- [ ] Implement Python and JS SDK behavior (manual instrumentation helpers).
- [ ] Add CI workflows (lint/typecheck/tests/e2e).
- [ ] Add integration tests for OTLP ingest and Tempo forwarding.
- [ ] Add benchmark and payload-size guardrail tests.

## Open Decisions

- [ ] Final naming for non-OTLP surfaces beyond `Records API`.
- [ ] Retention policies for record payload storage.
- [ ] Multi-tenant auth model for OSS vs Cloud mode.
