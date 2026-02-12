---
owner: sigil-core
status: active
last_reviewed: 2026-02-12
source_of_truth: true
audience: both
---

# Docs Index

This is the canonical navigation map for repository documentation.

## Core Sources

- `ARCHITECTURE.md`: canonical system architecture, data flow, and contracts.
- `AGENTS.md`: repository workflow guardrails and doc update triggers.

## Domain Guides

- [`DESIGN.md`](DESIGN.md)
- [`FRONTEND.md`](FRONTEND.md)
- [`PLANS.md`](PLANS.md)
- [`PRODUCT_SENSE.md`](PRODUCT_SENSE.md)
- [`QUALITY_SCORE.md`](QUALITY_SCORE.md)
- [`RELIABILITY.md`](RELIABILITY.md)
- [`SECURITY.md`](SECURITY.md)

## Structured Collections

- Design docs: [`design-docs/index.md`](design-docs/index.md)
- Product specs: [`product-specs/index.md`](product-specs/index.md)
- Execution plans:
  - Active: `exec-plans/active/`
  - Completed: `exec-plans/completed/`
  - Tech debt tracker: [`exec-plans/tech-debt-tracker.md`](exec-plans/tech-debt-tracker.md)
- Generated docs: [`generated/db-schema.md`](generated/db-schema.md)
- External references: [`references/index.md`](references/index.md)
  - Generation ingest contract: [`references/generation-ingest-contract.md`](references/generation-ingest-contract.md)

## Redundancy Rule

- Keep path catalogs centralized here.
- Other docs should reference this file instead of repeating full navigation lists.
