---
owner: sigil-core
status: active
last_reviewed: 2026-02-12
source_of_truth: true
audience: both
---

# Tech Debt Tracker

## Goal

Track cross-cutting debt, deferred architecture choices, and future-phase work.

## Scope

- Deferred Phase 2 implementation areas.
- Open architectural and product decisions that block scaling.

## Tasks

- [ ] Expand Go SDK with end-to-end examples.
- [ ] Implement Python and JS SDK behavior (manual instrumentation helpers).
- [ ] Add CI workflows (lint, typecheck, tests, e2e).
- [ ] Add integration tests for OTLP ingest and Tempo forwarding.
- [ ] Add benchmark and payload-size guardrail tests.
- [x] Records-first externalization replaced by Generation-first ingest.
- [ ] Define retention policies for generation payload and optional raw artifacts.
- [ ] Define multi-tenant auth model for OSS versus Cloud mode.

## Risks

- Deferred decisions may force rework in API, storage, and SDK interfaces.
- Missing guardrail tests can create reliability regressions under large payloads.
- Lack of CI automation increases drift risk.

## Exit Criteria

- Deferred items are either completed or converted into scoped execution plans.
- Open decisions have written ADR-level outcomes linked from design docs.
- Reliability and CI guardrails exist for critical ingest/query paths.
