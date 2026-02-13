---
owner: sigil-core
status: active
last_reviewed: 2026-02-13
source_of_truth: true
audience: both
---

# SDK Metrics and Telemetry Pipeline Delivery

## Goal

Move high-level AI observability metrics to Prometheus via SDK-emitted OTel metrics. Remove OTLP trace ingest from Sigil. Bundle Alloy as the standard telemetry pipeline. Update architecture docs and README to reflect the new data flow.

## Scope

- OTel metric instruments in all 5 SDKs (Go, Python, TypeScript/JavaScript, Java, .NET)
- Alloy bundling in Helm chart and docker-compose
- Trace ingest removal from Sigil (code, config, ports, metrics)
- Architecture and README updates with new data flow diagrams
- Migration guidance for existing deployments

## Source design doc

- `docs/design-docs/2026-02-13-sdk-metrics-and-telemetry-pipeline.md`

## Completion policy

- A checkbox moves to `[x]` only when implementation code and automated tests for that item are merged to `main`.
- Design docs, architecture text, or branch-local changes are not sufficient to close checklist items.

## Implementation phases

### Phase A: SDK Metrics (Go baseline)

#### Metric instruments

- [ ] Add MeterProvider setup to Go SDK core (`sdks/go/`), sharing OTLP exporter and Resource with TracerProvider.
- [ ] Define `gen_ai.client.operation.duration` histogram with buckets `[0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120]`.
- [ ] Define `gen_ai.client.token.usage` histogram with buckets `[1, 10, 50, 100, 250, 500, 1000, 2500, 5000, 10000, 50000, 100000]`.
- [ ] Define `gen_ai.client.time_to_first_token` histogram with buckets `[0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]`.
- [ ] Define `gen_ai.client.tool_calls_per_operation` histogram with buckets `[0, 1, 2, 3, 5, 10, 20, 50]`.

#### Recording at generation completion

- [ ] Record duration observation on `gen_ai.client.operation.duration` with attributes: `gen_ai.operation.name`, `gen_ai.provider.name`, `gen_ai.request.model`, `gen_ai.agent.name`, `error.type`.
- [ ] Record token observations on `gen_ai.client.token.usage` for each non-zero token type (`input`, `output`, `cache_read`, `cache_write`, `cache_creation`, `reasoning`) at generation completion.
- [ ] Record tool call count observation on `gen_ai.client.tool_calls_per_operation`: count output message parts with `Kind == PartKindToolCall` and record as histogram observation.
- [ ] Record duration observation at tool call completion with same duration attributes (`gen_ai.operation.name` = `execute_tool`).

#### Time-to-first-token (streaming)

- [ ] Capture `time.Now()` on first `stream.Next()` that returns data in provider streaming helpers.
- [ ] Compute TTFT = first_chunk_time - span_start_time and record on `gen_ai.client.time_to_first_token`.
- [ ] Only record TTFT for streaming operations (`gen_ai.operation.name` = `streamText`).
- [ ] Propagate first-chunk timestamp through `StreamSummary` or equivalent in each provider helper.

#### Missing span attributes

- [ ] Add `gen_ai.usage.reasoning_tokens` (int64) to `generationSpanAttributes()` -- currently in `TokenUsage` but not set on spans.
- [ ] Add `gen_ai.usage.cache_creation_input_tokens` (int64) to `generationSpanAttributes()` -- currently in `TokenUsage` but not set on spans.

#### Error categorization

- [ ] Add `error.category` span attribute to provider helpers: extract HTTP status code from provider error responses and map to category (`rate_limit`, `server_error`, `auth_error`, `timeout`, `client_error`, `sdk_error`).
- [ ] Set `error.category` on the `gen_ai.client.operation.duration` histogram observation alongside `error.type`.
- [ ] Implement HTTP status extraction in OpenAI, Anthropic, and Gemini Go provider helpers.

#### Lifecycle and tests

- [ ] Add MeterProvider shutdown to client shutdown flow (flush metrics alongside traces and generations).
- [ ] Unit tests: metric observation correctness, attribute values, zero-value token skipping.
- [ ] Unit tests: TTFT recording for streaming operations, no TTFT for sync operations.
- [ ] Unit tests: tool call count recording correctness (0 calls, 1 call, multiple calls).
- [ ] Unit tests: error categorization mapping (429 -> rate_limit, 500 -> server_error, etc.).
- [ ] Integration test: verify all 4 metric instruments arrive at a test OTLP receiver with correct names and attributes.

### Phase B: SDK Metrics (Python, TypeScript/JavaScript, Java, .NET)

- [ ] Port all 4 metric instruments from Go to Python SDK (`sdks/python/sigil_sdk/`).
- [ ] Port all 4 metric instruments from Go to TypeScript/JavaScript SDK (`sdks/js/src/`).
- [ ] Port all 4 metric instruments from Go to Java SDK (`sdks/java/core/`).
- [ ] Port all 4 metric instruments from Go to .NET SDK (`sdks/dotnet/src/Grafana.Sigil.Core/`).
- [ ] Port TTFT capture for streaming helpers in each SDK's provider packages.
- [ ] Port error categorization (HTTP status extraction) to each SDK's provider helpers.
- [ ] Port missing span attributes (`reasoning_tokens`, `cache_creation_input_tokens`) to each SDK.
- [ ] Each SDK: unit tests for metric observation correctness (all 4 instruments, TTFT, error categories).
- [ ] Each SDK: verify OTLP metric export works alongside trace export.

### Phase C: Bundle Alloy and Prometheus

#### docker-compose

- [ ] Add Alloy service to `docker-compose.yaml` with OTLP receiver (`:4317` gRPC, `:4318` HTTP).
- [ ] Create Alloy config (`config/alloy/config.river`) routing traces to Tempo and metrics to Prometheus.
- [ ] Configure Alloy with Docker metadata enrichment (`discovery.docker` and `discovery.relabel`) so metrics and traces are labeled with container name, compose service, and image.
- [ ] Add Prometheus service to `docker-compose.yaml` (for local dev metrics storage) with OTLP receiver enabled and appropriate scrape/retention config.
- [ ] Create `Dockerfile.sdk-traffic` in `.config/` -- a traffic generator container that uses the latest Go SDK to send realistic generation + trace + metric data to Alloy. This validates the full pipeline end-to-end in local dev.
- [ ] Add `sdk-traffic` service to `docker-compose.yaml` that builds from `.config/Dockerfile.sdk-traffic`, points `OTEL_EXPORTER_OTLP_ENDPOINT` at Alloy, and `SIGIL_ENDPOINT` at Sigil.

#### Auth delegation

- [ ] SDK auth for traces and metrics is no longer needed in the SDK config -- Alloy handles auth injection (tenant headers, bearer tokens) for upstream backends. Remove trace auth configuration from SDK examples and simplify the SDK config surface.
- [ ] Alloy config includes tenant header injection (`X-Scope-OrgID`) for Tempo and Prometheus when auth is enabled.

#### Helm chart

- [ ] Add Alloy deployment templates to Helm chart (`charts/sigil/templates/alloy-deployment.yaml`, `alloy-service.yaml`, `alloy-configmap.yaml`).
- [ ] Add Alloy configuration values to `charts/sigil/values.yaml` (`alloy.enabled`, `alloy.image`, `alloy.config`).
- [ ] Add Prometheus deployment templates to Helm chart (or document using an external Prometheus).
- [ ] Configure Alloy in Helm with k8s metadata enrichment (`discovery.kubernetes`).
- [ ] Configure Alloy in Helm with auth/tenant header injection for multi-tenant deployments.

#### Verification

- [ ] Update SDK example configs to point `OTEL_EXPORTER_OTLP_ENDPOINT` at Alloy instead of Sigil.
- [ ] Verify end-to-end locally: `sdk-traffic` container -> Alloy -> traces appear in Tempo + metrics appear in Prometheus.
- [ ] Verify Docker metadata labels appear on metrics in Prometheus (e.g., `container`, `compose_service`).

### Phase D: Remove Trace Ingest from Sigil

- [ ] Remove `sigil/internal/tempo/` package (client.go, tests).
- [ ] Remove `sigil/internal/ingest/trace/` package (service.go, http.go, grpc.go, tests).
- [ ] Remove OTLP HTTP server (`:4318`) from `sigil/internal/server_module.go`.
- [ ] Remove OTLP gRPC trace service registration (`collecttracev1.RegisterTraceServiceServer`) from `server_module.go`.
- [ ] Remove Tempo client lifecycle (init in `start`, close in `stop`) from `server_module.go`.
- [ ] Keep generation gRPC service on `:4317` (or move to main API server on `:8080`; decide and document).
- [ ] Remove `SIGIL_TEMPO_OTLP_GRPC_ENDPOINT` and `SIGIL_TEMPO_OTLP_HTTP_ENDPOINT` from `sigil/internal/config/config.go`.
- [ ] Remove `TempoOTLPGRPCEndpoint` and `TempoOTLPHTTPEndpoint` from `Config` struct.
- [ ] Remove `sigil_tempo_forward_*` Prometheus metrics from `sigil/internal/tempo/client.go` (already deleted with the package).
- [ ] Update Helm chart: remove `sigil.tempo.grpcEndpoint` and `sigil.tempo.httpEndpoint` values.
- [ ] Update Helm chart: remove OTLP port `:4318` from Sigil service definition.
- [ ] Update `docker-compose.yaml`: SDK/Alloy sends traces to Tempo directly, not through Sigil.
- [ ] Remove Tempo OTLP endpoint environment variables from `docker-compose.yaml` Sigil service.
- [ ] Run `mise run ci` to verify no broken imports, lint errors, or test failures.

### Phase E: Documentation and Architecture Updates

- [ ] Update `ARCHITECTURE.md`:
  - Rewrite System Boundaries to list Alloy as a system boundary and remove "OTLP trace ingest" from Sigil's description.
  - Rewrite Ingest Model: remove trace pipeline section, add SDK metrics pipeline section, keep generation pipeline.
  - Update Deployment topology guidance: Alloy is the standard telemetry path, Sigil is generation ingest + query only.
  - Update write path diagram: remove Sigil trace ingest -> Tempo path, add SDK -> Alloy -> Tempo + Prometheus path.
  - Update API Contracts: remove OTLP trace endpoints (`:4317` TraceService, `POST /v1/traces`).
  - Update Service Responsibilities: remove `sigil/internal/ingest/trace` entry, add SDK metrics description.
  - Add new section: SDK Metrics describing the four instruments, attributes, and cardinality.
- [ ] Update `README.md`:
  - Update "What You Get" section: remove OTLP gRPC/HTTP ports from Sigil, add Alloy as telemetry pipeline.
  - Update "Architecture At A Glance" mermaid diagram to show SDK -> Alloy -> Tempo + Prometheus, SDK -> Sigil for generations.
  - Update "Why Sigil" bullets: update OpenTelemetry-native description.
  - Update SDK example: change trace endpoint from Sigil to Alloy.
  - Update local stack description to include Alloy.
- [ ] Update `docs/design-docs/index.md`: add `2026-02-13-sdk-metrics-and-telemetry-pipeline.md` entry.
- [ ] Update `docs/exec-plans/active/2026-02-12-phase-2-delivery.md`: add SDK metrics and telemetry pipeline track.
- [ ] Update SDK READMEs with metrics setup guidance (document that metrics are emitted automatically).
- [ ] Update `docs/references/helm-chart.md` with Alloy configuration reference.
- [ ] Update `.env.example`: remove Tempo endpoint vars, add Alloy guidance.
- [ ] Update Hybrid storage data flow diagram in `ARCHITECTURE.md`: remove `Sigil Trace Ingest` node, add `Alloy` node.

## Risks

- SDK changes across 5 languages is significant work; mitigated by Go baseline first, then porting the same pattern.
- Customers without a collector lose trace/metric enrichment; mitigated by bundling Alloy in Helm/docker-compose.
- OTel GenAI metric semantic conventions are not yet stable; mitigated by accepting a rename if needed.
- `gen_ai.agent.name` cardinality; mitigated by being bounded by real agent count. Collector-level attribute filtering available as escape hatch.
- Generation gRPC port (`:4317` currently shared with OTLP traces); needs clear migration guidance. Alloy takes over `:4317`/`:4318` for OTLP.

## Rollout

1. Ship SDK metrics (Phase A + B) as additive -- no breaking changes, metrics are new.
2. Ship Alloy bundling (Phase C) as additive -- Alloy runs alongside existing stack.
3. Ship trace ingest removal (Phase D) as a breaking change with migration guide.
4. Ship docs (Phase E) alongside Phase D.

## Exit criteria

- All 4 SDK-emitted metrics (duration, token usage, TTFT, tool calls per operation) flow through Alloy to Prometheus with collector-enriched labels.
- TTFT is captured for streaming operations across all SDKs.
- Error categorization (rate_limit, server_error, auth_error, timeout) is implemented in provider helpers across all SDKs.
- Missing span attributes (`reasoning_tokens`, `cache_creation_input_tokens`) are set on generation spans across all SDKs.
- Sigil no longer has OTLP trace ingest code, config, or ports.
- Alloy is bundled and pre-configured in Helm chart and docker-compose.
- `ARCHITECTURE.md` and `README.md` accurately reflect the new architecture with updated diagrams.
- All changes are covered by unit and integration tests.
- Migration guidance is documented for existing deployments.

## Out of scope

- Sigil query API proxying PromQL to Prometheus (future single-datasource experience).
- Custom metric instruments beyond the 4 defined (can be added later).
- Alloy advanced configuration (sampling, tail-based sampling, custom processors).
- Grafana dashboard provisioning for AI metrics.
- Input message count / context depth metrics (derivable from generation records; can be added as a 5th instrument later if needed).
