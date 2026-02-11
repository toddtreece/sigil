# Langfuse OTEL Ingestion: How It Works, Pros/Cons, and Conversation Summary

## What This Covers
This note summarizes how Langfuse instrumentation and data collection work when using OpenTelemetry (OTEL), what gets collected, where size/privacy risks appear, and the key conclusions from our conversation.

## How It Works (Practical Flow)

1. Your app emits OTEL spans
- Spans can contain standard GenAI attributes/events (`gen_ai.*`, `llm.input_messages.*`, etc.) and/or `langfuse.*` attributes.
- Typical Langfuse fields are encoded as span attributes, e.g. `langfuse.trace.input`, `langfuse.observation.input`, `langfuse.observation.output`, model, usage, cost, prompt metadata.

2. Spans are sent via OTLP
- You can send OTLP directly to Langfuse (`/api/public/otel/v1/traces`) or route through an OTEL Collector.
- Collector is optional, not mandatory.

3. Langfuse OTEL endpoint receives raw OTLP payload
- API accepts JSON/protobuf OTLP trace payloads.
- Raw `resourceSpans` are uploaded to blob storage and queued for async processing.

4. Worker processes and maps spans to Langfuse entities
- Worker reads queued OTEL payloads and optionally applies ingestion masking (EE feature).
- `OtelIngestionProcessor` maps each span into Langfuse ingestion events.
- Observation type is inferred from attributes (e.g., `generation`, `span`, `tool`, `agent`, etc.).

5. Input/output extraction and metadata deduplication
- Message/completion fields are extracted into canonical `input` / `output`.
- Those same keys are removed from `metadata.attributes` to avoid duplication.
- Important: this is normalization/deduplication, not global redaction.

## What Data Is Actually Collected

- If your spans include full message history, Langfuse can ingest/store that as `input`/`output`.
- OTEL event-based chat content (e.g. `gen_ai.user.message`) can also be mapped into `input`/`output`.
- Therefore, conversation content is collected if you emit it.

## Collector and Downstream Backends (Tempo, etc.)

- If your app sends OTLP to a Collector, conversation payload is sent to that Collector.
- If Collector fans out to Langfuse + Tempo, both exporters typically receive the same payload unless you filter/transform per-exporter.
- Langfuse internal stripping from metadata does not sanitize what other exporters already received.

## Size and Safety Implications

### Why size can become a problem
- Full chat history on every span causes repeated large payloads.
- This stresses client batching, collector memory/queues, network, and backend limits.

### Practical safe pattern
- Send current turn input/output (or summarized history), not full history on every span.
- Apply truncation/filtering in SDK or Collector before fanout.
- Treat collector/export paths as potentially sensitive data pipelines.

## Pros
- OTEL-native path is now first-class; legacy ingestion is deprecated.
- Works with standard OTLP tooling and collectors.
- Flexible mapping supports many frameworks and semantic conventions.
- Central Langfuse model (trace + observations) is built automatically from spans.

## Cons
- If you over-attach message history, payload size can become large fast.
- Privacy risk increases when fanout sends identical payloads to multiple backends.
- Dedup in Langfuse metadata is not a replacement for source-side/collector-side redaction.
- Feature completeness beyond observability still requires non-trace APIs/SDK features (prompts, datasets, scoring flows, etc.).

## Summary of Our Conversation

1. You asked whether Langfuse only sends conversation via traces and if that is dangerous due to size.
- Conclusion: conversation can be collected through OTEL span attributes/events and mapped into observation/trace `input`/`output`; size risk is real if full history is attached repeatedly.

2. You asked if OTEL collects completion/message history.
- Conclusion: yes, if instrumentation emits those fields (`langfuse.observation.input/output`, `gen_ai.input.messages`, `llm.input_messages.*`, event messages, etc.).

3. You asked whether Langfuse strips this data.
- Conclusion: Langfuse strips duplicated keys from `metadata.attributes`, but preserves data in canonical `input`/`output`.

4. You asked whether sending traces elsewhere means all messages are included there too.
- Conclusion: yes, downstream systems receive what you export unless filtered before export/fanout.

5. You asked whether Collector usage is required and whether it is too much.
- Conclusion: Collector is optional; direct OTLP to Langfuse works. But if using Collector, it will see that data, so apply filtering/truncation as needed.

6. You asked if all Langfuse features work with OTEL only.
- Conclusion: OTEL-only gives strong observability, but some product capabilities still rely on dedicated SDK/API flows beyond trace ingestion.

7. You asked whether Langfuse SDK relies on OTEL.
- Conclusion: current recommended ingestion path is OTEL-based; legacy ingestion endpoints still exist but are deprecated.
