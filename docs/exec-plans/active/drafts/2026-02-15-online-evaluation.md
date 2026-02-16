---
owner: sigil-core
status: draft
last_reviewed: 2026-02-16
source_of_truth: false
audience: both
---

# Execution Plan: Online Evaluation (Live Scoring)

## Goal

Ship online evaluation for Sigil: configurable async scoring on production generations with score storage + query + Grafana plugin UX integration, designed to be cost-controlled and debuggable.

## Source Design Doc

- `docs/design-docs/drafts/2026-02-15-online-evaluation.md`

## Scope

- MySQL storage for generation scores (append-only) and a simple work queue.
- Score ingest API (`scores:export`) to support bring-your-own evaluator.
- Online rule execution loop (workers) with sampling/budgets and basic evaluators.
- Query surfaces to fetch scores for generations/conversations.
- Plugin UX to surface scores in conversation/generation workflows.
- SDK helper methods to export scores (Go/TS/Python/Java/.NET).

## Out of Scope (This Plan)

- Offline datasets + run comparison (separate plan).
- Hosted execution for evaluation jobs.
- Guardrails / request-path enforcement.
- Step/trajectory evaluators.

## Decision Gates (Resolve Before Coding)

1. Control plane storage:
   - Option A: config-first only (recommended MVP).
   - Option B: CRUD APIs + MySQL persistence for rules/evaluators.

2. Deployment topology:
   - Option A: eval worker runs inside `server` target.
   - Option B: new `eval-worker` target (recommended for scale and isolation), run as a second Sigil process with the same image.

Plan assumes Option A + Option B, but can be adjusted with minimal churn.

## Implementation Phases

### Phase 1: Data model and MySQL migrations

- [ ] Add MySQL models in `sigil/internal/storage/mysql/models.go`:
  - `GenerationScoreModel`
  - `EvalWorkItemModel`
  - optional `EvalOnlineRuleModel`/`EvalEvaluatorModel` only if we choose API-managed control plane
- [ ] Add `AutoMigrate` registrations in `sigil/internal/storage/mysql/migrate.go`.
- [ ] Add store methods in `sigil/internal/storage/mysql`:
  - `InsertGenerationScores(...)` with idempotency (`tenant_id + score_id`)
  - `EnqueueWorkItems(...)` with idempotency (`tenant_id + work_id`)
  - `ClaimWorkBatch(...)` with row locking and status transitions
- [ ] Update generated schema doc `docs/generated/db-schema.md` once models are stable.

### Phase 2: Score ingest API (`scores:export`)

- [ ] Define API contract in a reference doc:
  - `docs/references/evaluation-score-ingest.md` (new)
- [ ] Add proto + HTTP parity:
  - `sigil/proto/sigil/v1/evaluation_ingest.proto`
  - generate stubs in `sigil/internal/gen/sigil/v1/`
- [ ] Implement service in `sigil/internal/eval/ingest`:
  - validation (typed value oneof, required fields, size limits)
  - per-item acceptance response
  - idempotency conflict handling (409-like semantics)
- [ ] Register routes:
  - HTTP: `POST /api/v1/scores:export` in `sigil/internal/server/http.go`
  - gRPC: `EvaluationIngestService.ExportScores` in `sigil/internal/server_module.go`
- [ ] Add unit tests for:
  - validation errors
  - idempotency conflict
  - mixed success responses

### Phase 3: Query APIs for scores

- [ ] Add handlers:
  - `GET /api/v1/generations/{id}/scores?limit&cursor` (pagination contract mirrors feedback endpoints)
  - optional `GET /api/v1/conversations/{id}/scores/summary`
- [ ] Extend query service interfaces in `sigil/internal/query/service.go` to fetch score summaries for conversation/generation detail.
- [ ] Add MySQL read methods in `sigil/internal/storage/mysql`:
  - list scores by generation id (tenant-scoped, newest-first)
  - list latest score per key (optional summary)
- [ ] Add HTTP tests in `sigil/internal/server/http_test.go` for routing + auth + pagination edge cases.

### Phase 4: Online rule config (config-first)

- [ ] Add config struct fields in `sigil/internal/config/config.go`:
  - `OnlineEvalEnabled`
  - `OnlineEvalConfigPath` (or embed in env vars)
  - worker budgets (max concurrency, max executions/min)
  - optional OTLP export config for evaluation spans
- [ ] Define YAML schema and loader in `sigil/internal/eval/config`:
  - evaluators (id, version, kind, input policy)
  - rules (id, match, sample rate, evaluator refs)
- [ ] Add rule `select` support with a small set of built-in selectors:
  - `user_visible_turn` (default; assistant output with text and no tool calls)
  - `all_assistant_generations`, `tool_call_steps`, `errored_generations`
  - document how tags can override (`sigil.visibility=internal|user`, `sigil.run_id=<id>`)
- [ ] Add validation and stable hashing for:
  - evaluator definitions
  - rule definitions

### Phase 5: Work queue + worker loop

- [ ] Add worker package `sigil/internal/eval/worker`:
  - load rules/evaluators once (and support reload later)
  - claim queued work items from MySQL
  - fetch generation payload via WAL store (hot; cold fallback optional later)
  - execute evaluator and export scores (write to MySQL)
  - update work item status and record last error
- [ ] Implement sampling:
  - deterministic hash-based sampling per `(tenant_id, generation_id, rule_id)`
- [ ] Implement global budgets:
  - token bucket or leaky bucket in-memory per worker process
  - backpressure when queue backlog exceeds limit
- [ ] Add worker metrics in Prometheus:
  - executions total by status
  - durations
  - queue depth (from DB) and claim lag
- [ ] Add tests:
  - matching + sampling determinism
  - retry/backoff behavior
  - idempotency on score writes

### Phase 6: Ingest-time enqueue (trigger)

- [ ] Decide trigger mechanism:
  - Option A: enqueue from generation ingest service
  - Option B: enqueue via a DB trigger/poll loop
- [ ] Implement Option A (preferred):
  - add an enqueue hook after `SaveBatch` succeeds in the MySQL WAL store path (not in memory store)
  - compute eligible rules cheaply from normalized generation fields
  - apply selector eligibility (for example `user_visible_turn`) before enqueueing work
  - insert work items into `eval_work_items`
- [ ] Add integration tests:
  - ingest generation -> work item created
  - sampling respects rate

### Phase 7: Built-in evaluators (minimal starter set)

- [ ] Implement evaluator interface in `sigil/internal/eval/evaluators` returning a list of typed score outputs.
- [ ] Implement:
  - `regex` evaluator (response text)
  - `json_schema` evaluator (response text parsed to JSON)
  - `heuristic` evaluator (length bounds, empty response)
- [ ] Implement `llm_judge` behind an explicit enable flag:
  - define provider interface and support at least one provider initially
  - strict timeouts and token limits
  - stable prompt hashing and metadata capture
- [ ] Add unit tests for evaluator input mapping and output typing.

### Phase 8: Optional “evaluation spans” export

- [ ] Add OTLP exporter config for Sigil worker process (distinct from app SDK exporters).
- [ ] Implement span creation with remote parent using stored `trace_id`/`span_id` on generation payload.
- [ ] Ensure spans can be disabled completely by config.
- [ ] Add a small integration test that validates spans are emitted (mock OTLP collector).

### Phase 9: Plugin backend proxy routes

- [ ] Add plugin backend resource routes in `apps/plugin/pkg/plugin/resources.go`:
  - `POST /query/scores:export` (if needed from UI)
  - `GET /query/generations/{id}/scores`
  - `GET /query/conversations/{id}/scores/summary` (if implemented)
- [ ] Add resource tests in `apps/plugin/pkg/plugin/resources_test.go`.

### Phase 10: Plugin frontend UX (minimal but useful)

- [ ] Extend conversation/generation types:
  - add `GenerationScore` and `GenerationScoresResponse` in `apps/plugin/src/shared/types.ts` or `apps/plugin/src/conversation/types.ts`
- [ ] Add API calls in `apps/plugin/src/conversation/api.ts` (or a new `apps/plugin/src/eval/api.ts`):
  - list generation scores
- [ ] Update `apps/plugin/src/pages/ConversationsPage.tsx`:
  - show score badges in detail panel (for selected conversation’s last generation, or per-generation once generation detail exists)
  - show “failing eval” count in merged timeline (as a new event kind) once available
- [ ] Add Storybook stories:
  - score badge component
  - conversations page with mocked scores

### Phase 11: SDK helper methods (bring-your-own score)

- [ ] Add SDK helper methods mirroring the ratings pattern:
  - Go: `sdks/go/sigil/score.go` + tests
  - JS: `sdks/js/src/client.ts` + tests
  - Python: `sdks/python/sigil_sdk/client.py` + tests
  - Java: `sdks/java/core/.../SigilClient.java` + tests
  - .NET: `sdks/dotnet/.../SigilClient.cs` + tests
- [ ] Update SDK READMEs with minimal examples.

### Phase 12: Docs and examples

- [ ] Add operator docs:
  - “how to enable online eval” (config + budgets + judge credentials)
  - “how to publish scores from your app”
- [ ] Add an example config under `config/`:
  - `config/eval/online.example.yaml`
- [ ] Update `docs/design-docs/index.md` and `docs/index.md` to include draft docs under a “Drafts” section.

## Risks

- **Cost blowups (LLM judge):** mitigated by sampling defaults, required budgets, strict token/time limits.
- **Ingest coupling:** ingest-time enqueue must be bounded; failures must not break ingest.
- **Privacy leakage:** evaluator inputs must be policy-driven and truncated; judge prompts/explanations must be handled carefully.
- **Query complexity:** score filters integrated into conversation search can explode; keep MVP focused on display and dedicated eval pages.

## Exit Criteria

- Scores can be exported via `POST /api/v1/scores:export` with idempotency semantics.
- Generations can be queried for scores with stable pagination.
- Online rules can be configured and run asynchronously with sampling + budgets.
- Plugin UI shows score badges linked to generations/conversations.
- SDKs can submit scores easily (bring-your-own evaluator).
- Metrics exist for worker health and evaluation execution outcomes.
