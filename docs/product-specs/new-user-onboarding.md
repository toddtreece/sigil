---
owner: sigil-core
status: draft
last_reviewed: 2026-02-11
source_of_truth: true
audience: both
---

# New User Onboarding

## Problem

New users can start the stack and open Grafana, but they need a guided first-run path to understand Sigil's ingest, query, and trace-linking model.

## User Flow

1. User starts the stack with `mise run up`.
2. User opens Grafana and the Sigil app plugin.
3. User sees a guided checklist for ingest endpoint setup and sample data emission.
4. User confirms trace, conversation, and completion views render linked data.

## Requirements

- Provide a first-run checklist in the plugin settings/onboarding surface.
- Include copy-paste OTLP endpoint examples for local development.
- Include validation cues that data ingestion and query APIs are working.
- Link to architecture and reference docs for deeper troubleshooting.

## Acceptance Criteria

- A first-time user can reach a successful ingest and view data in under 10 minutes.
- Onboarding steps clearly indicate success/failure states.
- Onboarding content references canonical docs paths and contracts.
