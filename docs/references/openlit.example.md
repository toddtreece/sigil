# OpenLIT + OTel GenAI Summary (Conversation Recap)

## Snapshot
- Date: 2026-02-11
- Scope: OpenLIT SDK vs raw OTel / OTel contrib GenAI instrumentation, with focus on custom instrumentation and large prompt/response payloads.

## TL;DR
- Yes, you can use OpenLIT with your own instrumentation (not tied to a specific LLM vendor SDK wrapper).
- OpenLIT is OTLP/OTel-native, but its current schema/UI still relies on some deprecated GenAI keys (`gen_ai.system`, `gen_ai.prompt`, `gen_ai.completion`).
- OTel GenAI guidance says: do **not** capture full messages by default; use opt-in modes, and prefer external storage + references for production.
- For very large payloads (e.g., 1M tokens), content handling should happen primarily at the SDK/app layer, with transport/collector guardrails as defense-in-depth.

## How It Works

### OpenLIT path
- `openlit.init(...)` configures tracing/metrics export via OTel and auto-instrumentation.
- Python also exposes manual helpers:
  - `@openlit.trace`
  - `openlit.start_trace(...)`
- OpenLIT dashboards query specific span attributes/events for model, provider, cost, token usage, prompt/completion.

### OTel contrib GenAI path
- Instrumentors patch provider SDK methods (e.g., OpenAI/Google GenAI integrations in contrib).
- They emit:
  - spans (operation/model/tokens/status)
  - metrics (duration, token usage, etc. depending on package)
  - events/logs (message-level data when enabled)
- Newer contrib/util-genai supports content capture modes and completion hooks for external content handling.

## The “Don’t Capture Messages” vs “They Log Messages” Confusion
- Both are true, but for different modes:
1. Default recommendation: do **not** capture raw instructions/messages/outputs.
2. If user opts in, instrumentations can capture content (often in events/logs, sometimes on span attributes).
3. In production, OTel recommends external storage for big/sensitive content and storing references in telemetry.

So the recommendation is policy-driven:
- Default = no raw content.
- Opt-in = capture for debugging/pre-prod.
- Production scale = externalize content and keep telemetry lightweight.

## Pros / Cons

### OpenLIT SDK
**Pros**
- Fast setup and broad integration surface.
- Built-in platform UX for LLM observability (cost/tokens/trace views).
- Useful manual tracing APIs in Python.

**Cons**
- Some schema/UI expectations currently lean on deprecated GenAI keys from latest OTel perspective.
- Default content capture behavior can be too permissive for production if left unchanged.
- If you fully standardize on latest OTel GenAI fields/events, you may need mapping for OpenLIT UI compatibility.

### OTel contrib GenAI instrumentations
**Pros**
- Closer to current OTel semantic convention direction.
- Clearer model for content capture modes and external-upload hooks.
- Lower lock-in long-term if your goal is strict OTel standard alignment.

**Cons**
- Contrib repo still flags beta/development status for many packages/conventions.
- Coverage and maturity vary by integration.
- More assembly effort if you want a unified product experience comparable to OpenLIT UI.

## What We Concluded For Large Payloads (Multi-LLM, 1M-token class)
- Do not ship full transcripts by default in span attributes/events.
- Prefer this pattern:
1. At SDK/app level:
   - redact/classify/truncate content
   - upload full content to external storage
   - put references (URI/object id/hash/chunk ids) on span/events
2. At transport/collector level:
   - enforce limits, sampling, retention, and final filtering
- Keep operational telemetry small and queryable (model, provider, token counts, cost, latency, error type).

## Recommended Practical Strategy
1. Use OTel GenAI-compatible core fields (`gen_ai.operation.name`, `gen_ai.provider.name`, token usage, etc.).
2. Set content capture OFF by default in production.
3. Use a completion hook / app-side hook to externalize raw content.
4. Add reference attributes/events only.
5. If OpenLIT dashboard compatibility is required today, add a mapping layer for legacy keys expected by current OpenLIT UI.

## Conversation Recap (What We Answered)
- Can OpenLIT be used with custom instrumentation?  
  Yes.
- Is `text_completion` the message content?  
  No, it is operation type only.
- What does OTel recommend for raw messages?  
  Default no capture; opt-in capture; for production prefer external storage + references.
- How do contrib SDKs collect prompts/messages?  
  By wrapping SDK calls and reading request/response objects; they emit structured telemetry according to capture mode.
- Which is more future-proof?  
  OTel-first schema is more standard/future-aligned; OpenLIT is faster for turnkey product value now.

