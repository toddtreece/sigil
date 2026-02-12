---
owner: sigil-core
status: active
last_reviewed: 2026-02-11
source_of_truth: true
audience: both
---

# Core Beliefs

## Context

Sigil is an AI observability system where users should quickly understand what happened after an LLM action. The product and architecture must keep chronology, causality, and cross-linking between outputs and traces visible.

## Decision

- Keep language short, technical, and concrete.
- Prioritize observability after LLM actions: tokens, outputs, tool actions, references, and traces.
- Treat prompts as execution inputs and traces as the source of runtime truth.
- Keep docs and architecture legible for both humans and agents.

### Anti-Principles

- Do not hide behavior behind vague abstractions.
- Do not expand UI or API surface before traceability and reliability are clear.
- Do not add bootstrap hacks that create long-term maintenance debt.

## Alternatives

- Marketing-first narrative docs with low implementation detail.
- Feature-by-feature docs without cross-system causal flow.
- Auto-generated docs only, without curated engineering decision records.

## Consequences

- Documentation must stay close to runtime contracts and boundaries.
- Product and UX work should preserve trace-linking and timeline coherence.
- New features should include explicit observability and failure-path thinking.
