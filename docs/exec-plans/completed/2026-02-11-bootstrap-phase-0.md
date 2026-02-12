---
owner: sigil-core
status: stable
last_reviewed: 2026-02-11
source_of_truth: true
audience: agents
---

# Bootstrap Phase 0

## Goal

Establish the monorepo scaffolding, baseline services, and primary contracts.

## Scope

- Workspace tooling and compose stack.
- Plugin and API project scaffolding.
- Initial SDK skeletons.

## Tasks

- [x] Monorepo root with `mise`, `pnpm` workspace, and compose stack.
- [x] Assistant-style plugin layout in `apps/plugin`.
- [x] Plugin backend query proxy scaffold.
- [x] Sigil API skeleton with OTLP, Records, and query contracts.
- [x] SDK skeletons for Go, Python, and JS.

## Risks

- Bootstrap scaffolding may diverge from production architecture.
- Placeholder implementations can be mistaken for production-ready behavior.

## Exit Criteria

- Local stack starts with documented commands.
- Core contract endpoints and repository structure are present.
- Teams can begin implementing functional Phase 1 features.
