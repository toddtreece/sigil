---
owner: sigil-core
status: draft
last_reviewed: 2026-02-15
source_of_truth: false
audience: both
---

# Execution Plan: Offline Evaluation (Datasets + Experiments + CI Gates)

## Goal

Ship offline evaluation primitives for Sigil:

- publishable dataset-backed runs
- run comparisons and CI gating
- deep drilldowns from regressions to generations and traces

## Source Design Doc

- `docs/design-docs/drafts/2026-02-15-offline-evaluation.md`

## Dependencies

- Online score ingest + storage primitives from:
  - `docs/exec-plans/active/drafts/2026-02-15-online-evaluation.md`
  - specifically `POST /api/v1/scores:export`, `generation_scores`, and query surfaces.

Offline eval builds on the same score model; it primarily adds **run grouping** and **comparison UX**.

## Scope

Phase 1 (MVP):

- Run metadata storage (`eval_runs`) and APIs (create/list/get).
- Simple comparison API (baseline vs candidate) with aggregate deltas and regression budgets.
- Plugin UI for run list + compare + failure drilldown.
- SDK helper methods to create runs and attach `run_id` to scores.
- Documentation + CI templates (GitHub Actions example).

Phase 2 (optional, if we choose to build a canonical runner):

- `sigil eval` CLI or a minimal “Sigil reporter” that integrates with an existing runner.

## Out of Scope (This Plan)

- Full dataset management UI and storage (unless we explicitly choose to include it).
- Hosted replay execution in Sigil.
- Agent environment sandboxing.

## Decision Gates (Resolve Before Coding)

1. Dataset management:
   - Option A: datasets as files (git/blob storage); Sigil stores references.
   - Option B: datasets as first-class Sigil objects (MySQL/object store + UI).

2. Runner strategy:
   - Option A: integrate existing runners via “Sigil reporter” (recommended MVP).
   - Option B: build a Sigil CLI runner.

Plan assumes Option A for both.

## Implementation Phases

### Phase 1: Run metadata schema + migrations

- [ ] Add MySQL model `EvalRunModel` in `sigil/internal/storage/mysql/models.go`.
- [ ] Register migration in `sigil/internal/storage/mysql/migrate.go`.
- [ ] Add store methods in `sigil/internal/storage/mysql`:
  - `UpsertEvalRun(...)` (idempotent by `tenant_id + run_id`)
  - `ListEvalRuns(...)` (by dataset, time, status)
  - `GetEvalRun(...)`

### Phase 2: Eval run APIs

- [ ] Define HTTP contracts (reference doc):
  - `docs/references/offline-eval-api.md` (new)
- [ ] Add HTTP handlers in `sigil/internal/server/http.go`:
  - `POST /api/v1/eval/runs` (create/update run metadata)
  - `GET /api/v1/eval/runs?dataset_id=&limit=&cursor=`
  - `GET /api/v1/eval/runs/{run_id}`
- [ ] Add validation semantics:
  - required: `run_id`, `name`, `kind`, `dataset_id`, `dataset_version`, `started_at`
  - candidate metadata stored as JSON with a size cap
- [ ] Add unit + HTTP tests:
  - idempotency replay behavior
  - conflict behavior on same `run_id` with different payload
  - tenant isolation

### Phase 3: Run comparison (aggregates + regression budgets)

- [ ] Implement comparison logic in `sigil/internal/eval/compare`:
  - aggregate score metrics from `generation_scores` grouped by `run_id` and `score_key`
  - compute:
    - mean/median for numeric scores
    - pass-rate for boolean scores (and numeric with `passed`)
    - count of new failures compared to baseline (by dataset `item_id` in metadata, if provided)
- [ ] Add API:
  - `GET /api/v1/eval/runs/{run_id}/compare?baseline_run_id=...`
  - response includes:
    - aggregates for baseline and candidate
    - deltas
    - top regressed slices (if slice metadata exists)
    - list of “new failing items” with generation ids for drilldown
- [ ] Add “gate evaluation” endpoint or pure client-side gating:
  - Option A: `POST /api/v1/eval/gates:check` (server checks budgets and returns pass/fail)
  - Option B: runner/CI computes gate locally from compare response (recommended MVP)

### Phase 4: Query drilldowns for failures

- [ ] Add endpoints to fetch run failures:
  - `GET /api/v1/eval/runs/{run_id}/failures?score_key=&limit=&cursor=`
  - returns rows with:
    - dataset `item_id` (from score metadata)
    - generation_id, conversation_id, trace_id/span_id
    - failed keys + explanations
- [ ] Ensure drilldown links work:
  - failure -> generation detail (`GET /api/v1/generations/{id}` once available)
  - generation -> Tempo trace view (existing proxy)

### Phase 5: Plugin backend proxy routes

- [ ] Add proxy routes in `apps/plugin/pkg/plugin/resources.go`:
  - `/query/eval/runs` (GET/POST)
  - `/query/eval/runs/{id}` (GET)
  - `/query/eval/runs/{id}/compare` (GET)
  - `/query/eval/runs/{id}/failures` (GET)
- [ ] Add resource route tests in `apps/plugin/pkg/plugin/resources_test.go`.

### Phase 6: Plugin frontend UX (runs + compare + drilldown)

- [ ] Add a new page:
  - `apps/plugin/src/pages/EvaluationPage.tsx`
- [ ] Add nav route in `apps/plugin/src/constants.ts` and module wiring in `apps/plugin/src/module.tsx`.
- [ ] Add types + API client:
  - `apps/plugin/src/eval/types.ts`
  - `apps/plugin/src/eval/api.ts`
- [ ] Implement UX:
  - runs list (filter by dataset, status)
  - select candidate run + baseline run to compare
  - show aggregate deltas (helpfulness mean, pass-rate)
  - “new failures” table with links to generation/conversation pages
- [ ] Add Storybook:
  - run list story
  - compare view story

### Phase 7: SDK helper methods for offline runs

- [ ] Add `CreateEvalRun` helpers across SDKs (pattern mirrors ratings):
  - Go: `CreateEvalRun(ctx, input)`
  - JS/TS: `createEvalRun(input)`
  - Python: `create_eval_run(input)`
  - Java: `createEvalRun(request)`
  - .NET: `CreateEvalRunAsync(request)`
- [ ] Ensure score submission helpers support `run_id` (online plan already adds score export helpers).
- [ ] Update SDK README snippets for “prompt CI” pattern.

### Phase 8: CI templates + runner integrations (MVP)

- [ ] Publish a minimal GitHub Actions example under `docs/`:
  - “run Promptfoo, export results to Sigil, compare to baseline, fail on budget”
- [ ] Provide a “Sigil reporter” script/tool:
  - input: JSON results from a runner
  - output: calls `POST /api/v1/eval/runs` and `POST /api/v1/scores:export`
  - keep it simple; Go or Node script in `scripts/` (decision)

## Risks

- **Runner fragmentation:** without a canonical runner, adoption can be uneven; mitigate by shipping strong templates and at least one integration path.
- **Weak debuggability if results aren’t linked to generations:** mitigate by making “generation_id is required” for first-class drilldown, and clearly labeling degraded modes.
- **Evaluation drift:** judge models/prompts drift; mitigate with strict versioning and hash capture.
- **Privacy:** datasets mined from prod can leak PII; mitigate by redaction at dataset creation and strict docs/guardrails.

## Exit Criteria

- Runs can be created/listed/retrieved via API with idempotency semantics.
- Scores can be grouped by `run_id` and compared to a baseline with clear deltas.
- CI can fetch compare output and fail on regression budgets.
- Plugin UI supports:
  - run list
  - run compare
  - drilldown to failures with generation/trace links
- SDKs provide helpers to publish run metadata and scores.

