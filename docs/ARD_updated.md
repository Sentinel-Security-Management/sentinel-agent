# sentinel-agent
# Architecture Requirements Document

Version: 1.0
Status: Active
Repository: github.com/Sentinel-Security-Management/sentinel-agent
References: PRD.md (goals), SRS.md (specifications)

---

## 1. Purpose of this document

This document records every significant architectural decision made for sentinel-agent. Each decision includes the context that made it necessary, the options that were considered, the decision reached, and the reasoning behind it. Future contributors and reviewers must read this document before making a change that touches the structural shape of the module.

This document also defines the constraints that cannot be violated regardless of future requirements, the component relationships that govern the internal dependency graph, and the failure mode catalogue that specifies expected behaviour under every known failure condition.

If a proposed change conflicts with a decision recorded here, the decision stands unless a new ADR explicitly supersedes it. New ADRs are appended at the end of section 2.

---

## 2. Architectural decisions

### ADR-001: sentinel-agent is an OTel Collector distribution, not a custom metrics collector

Date: 2026-06

Context: The initial design planned to use gopsutil to collect host metrics directly in a custom Go binary. This would require maintaining custom collection logic for CPU, memory, disk, network, filesystem, and process metrics across Linux kernel versions, distributions, and cloud environments.

Options considered:

Option A. Custom Go binary with gopsutil collecting all metrics directly. Full control, single dependency, simple to understand. Requires maintaining collection logic for every metric type across environments.

Option B. Pre-configured OTel Collector distribution. The OTel Collector Contrib project already includes hostmetricsreceiver, kubeletstatsreceiver, k8sclusterreceiver, dockerstatsreceiver, and filelogreceiver. These receivers are maintained by the OpenTelemetry community, tested across environments, and actively developed. Custom code is written only for what the OTel Collector does not already provide.

Decision: Option B. sentinel-agent is a pre-configured OTel Collector distribution.

Reasoning: The OTel Collector receivers cover every metric collection requirement identified for sentinel-agent across all four deployment environments. The hostmetricsreceiver covers what Node Exporter covers for VM and bare metal deployments. The kubeletstatsreceiver covers pod and node metrics for Kubernetes. The dockerstatsreceiver covers container metrics for Docker deployments. The filelogreceiver covers log collection across all environments. Writing custom collection code for these would add thousands of lines that the community already maintains and improves. The custom code in sentinel-agent is limited to cloud identity resolution and the sentinel-scrub processor, which have no OTel Collector equivalent.

Consequences: sentinel-agent's configuration is partly expressed as OTel Collector YAML pipeline configuration and partly as Go code wrapping the collector. This means contributors need to understand both the OTel Collector component model and the Go code that builds and runs it. This complexity is accepted because the alternative is greater.

### ADR-002: OBI is not merged into sentinel-agent

Date: 2026-06

Context: OBI runs as a DaemonSet on every node alongside sentinel-agent. Both are DaemonSets. Both export OTLP. The question arose whether to compile OBI's eBPF collection code into sentinel-agent to produce a single binary and a single DaemonSet.

Options considered:

Option A. Merge OBI into sentinel-agent. One binary, one DaemonSet, simpler from the outside.

Option B. Keep OBI separate. OBI exports OTLP to sentinel-agent on localhost. sentinel-agent receives it and processes it through the same pipeline as all other signals.

Decision: Option B. OBI stays separate.

Reasoning: OBI is an upstream OpenTelemetry community project. It has its own release cycle, its own breaking changes, and its own kernel compatibility matrix. If merged, Sentinel would own the maintenance burden for OBI's eBPF kernel probes, uprobe Go tracer, and protocol parsers for every supported protocol. That is months of engineering work that the OTel community performs for free. The moment the code is forked and merged, those upstream contributions are lost.

Additionally, OBI requires CAP_BPF, CAP_PERFMON, CAP_NET_ADMIN, CAP_SYS_PTRACE, Linux kernel 5.8+, and BTF enabled. The OTel Collector inside sentinel-agent requires none of these. If merged, the combined binary would require privileged mode even on environments where eBPF is not supported. Keeping them separate means OBI fails gracefully on unsupported kernels while sentinel-agent continues operating normally.

Operationally, OBI and sentinel-agent update on independent cadences. An OBI patch should not require a full sentinel-agent release and upgrade cycle.

The two-DaemonSet complexity is solved by sentinelhq provision, not by merging binaries. sentinelhq provision generates manifests for both in a single operation. The engineer applies them together and sees one product.

Consequences: sentinel-agent must always be deployed alongside OBI for full network and protocol observability. The PRD and all customer documentation must be clear that the pair together constitute the collection layer, not sentinel-agent alone. The otlpreceiver on localhost:4317 must always be active to receive OBI's output.

### ADR-003: sentinel-scrub runs inside sentinel-agent's pipeline, not in sentinel-processor

Date: 2026-06

Context: PII enforcement could be placed either in sentinel-agent on the node, or in sentinel-processor in the customer's VPC. Both are within the customer's infrastructure.

Options considered:

Option A. PII enforcement in sentinel-processor. Centralised, single place to update the policy, simpler agent.

Option B. PII enforcement in sentinel-agent. PII is hashed or dropped before any signal leaves the node. sentinel-processor receives already-clean signals.

Decision: Option B. sentinel-scrub processor runs as step four in the sentinel-agent pipeline.

Reasoning: The data sovereignty guarantee is that customer data stays within the customer's infrastructure. Even within the customer's infrastructure, there is a meaningful difference between PII existing on a single node versus PII flowing across the internal network to sentinel-processor and being stored in ClickHouse. Enforcing at the node boundary means PII never travels across any network connection in raw form, even an internal one. This is a stronger guarantee and simpler to audit. A security engineer auditing sentinel-agent can verify that the scrub processor runs before the exporter by reading the pipeline configuration. They do not need to audit sentinel-processor's internal pipeline as well.

Consequences: sentinel-agent has a direct dependency on the scrub package. The scrub package must be compiled into sentinel-agent. The sentinel-policy.yaml file must be present on every node, typically delivered via a ConfigMap or a volume mount. The agent must handle the case where the policy file is missing at startup (fail with a clear error) and where it is updated (reload on SIGHUP without restart).

### ADR-004: cloud identity resolution uses direct IMDS probes, not the OTel resourcedetectionprocessor

Date: 2026-06

Context: The OTel Collector Contrib project includes a resourcedetectionprocessor that can detect cloud identity from AWS, GCP, and Azure metadata endpoints. This could replace the custom identity package.

Options considered:

Option A. Use resourcedetectionprocessor from OTel Collector Contrib.

Option B. Write a custom cloud identity resolver in the identity package.

Decision: Option B for AWS IMDSv2 specifically. Option A is used as a fallback for attributes the custom resolver does not cover.

Reasoning: The resourcedetectionprocessor does not implement the IMDSv2 token-based flow correctly as of the OTel Collector version pinned in this project. IMDSv1 is being deprecated on new AWS instances. A custom resolver that correctly implements the PUT-then-GET IMDSv2 flow is required to reliably detect cloud identity on modern AWS infrastructure. For GCP and Azure the resourcedetectionprocessor is sufficient. The custom identity package handles AWS IMDSv2 and falls back to the resourcedetectionprocessor for GCP and Azure where its implementation is correct.

Consequences: The identity package must be maintained across AWS IMDS API changes. Integration tests must run against real AWS IMDS endpoints on a scheduled basis in CI to catch any API changes before they affect customers.

### ADR-005: OBI and sentinel-sdk export to sentinel-agent, not to sentinel-processor directly

Date: 2026-06

Context: OBI and sentinel-sdk both export OTLP. They could export directly to sentinel-processor, bypassing sentinel-agent.

Options considered:

Option A. OBI and sentinel-sdk export to sentinel-processor directly. sentinel-agent only collects host metrics and logs.

Option B. OBI and sentinel-sdk export to sentinel-agent on localhost:4317. sentinel-agent receives all signals from all sources and forwards a unified stream to sentinel-processor.

Decision: Option B. All signals on a node pass through sentinel-agent.

Reasoning: If OBI and sentinel-sdk export directly to sentinel-processor, they bypass sentinel-agent's pipeline. This means their signals do not receive cloud identity stamping, do not receive Kubernetes metadata enrichment, and do not pass through the sentinel-scrub processor. An OBI trace would reach sentinel-processor without cloud.provider, without k8s.pod.name, and with potential PII in protocol-level attributes like IP addresses and query parameters.

Centralising all signal ingress through sentinel-agent means there is exactly one place where resource stamping happens, exactly one place where PII enforcement happens, and exactly one authenticated connection leaving the node. This is simpler to audit, simpler to debug, and provides a stronger security boundary.

Consequences: sentinel-agent must listen on localhost:4317 in all deployment modes. OBI's export configuration must point to localhost:4317. sentinel-sdk documentation must instruct developers to set OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317. The otlpreceiver must be active in all pipeline configurations.

### ADR-006: mTLS is required for the sentinel-agent to sentinel-processor connection

Date: 2026-06

Context: The connection between sentinel-agent and sentinel-processor is gRPC. Authentication options include no authentication, API keys in headers, JWT tokens, and mutual TLS.

Options considered:

Option A. No authentication. Any process that can reach sentinel-processor's port can inject telemetry.

Option B. API key in a gRPC header. Simple but the key must be stored on every node and rotated manually.

Option C. mTLS with per-node client certificates. Each node has a unique certificate. sentinel-processor validates the certificate fingerprint against an allowlist. Compromising one node's certificate does not compromise other nodes.

Decision: Option C. mTLS with per-node client certificates.

Reasoning: sentinel-processor accepts telemetry that will be stored and queried as ground truth about the customer's infrastructure. If any process can inject data into sentinel-processor, the integrity of the entire observability platform is compromised. mTLS provides per-node authentication, mutual verification that both sides are legitimate Sentinel components, and certificate revocation capability without agent restart (sentinel-processor updates its allowlist without restarting).

Consequences: sentinelhq provision must generate a certificate for each node as part of the deployment process. Certificate rotation must be possible without agent restart (SIGHUP reloads cert paths). The deployment documentation must explain certificate management clearly. The mTLS connection adds negligible latency because it is a persistent gRPC connection, not per-request TLS handshakes.

### ADR-007: transport layer is direct OTLP to sentinel-processor, not through NATS

Date: 2026-06

Context: The full Sentinel architecture uses NATS JetStream as a message bus inside sentinel-processor. The question is whether sentinel-agent should export to NATS directly or to the OTel Collector inside sentinel-processor which then publishes to NATS.

Options considered:

Option A. sentinel-agent exports OTLP to NATS directly. The OTLP-to-NATS bridge runs on every node.

Option B. sentinel-agent exports OTLP to the OTel Collector inside sentinel-processor. The OTel Collector publishes to NATS after receiving and processing.

Decision: Option B. sentinel-agent exports OTLP to sentinel-processor's OTel Collector.

Reasoning: sentinel-agent's job is collection and identity. It should not know or care about the transport architecture inside sentinel-processor. If NATS sits between sentinel-processor and its consumers (ClickHouse writer, entity service, ghost-agent), sentinel-agent never needs to change when the transport layer evolves. This is a cleaner separation of concerns. The otlpexporter is battle-tested and supported by all OTel tooling. A NATS exporter for the OTel Collector exists but is less mature.

Consequences: sentinel-agent's exporter configuration is the standard otlpexporter. When the transport layer inside sentinel-processor changes, sentinel-agent is unaffected. The OTel Collector inside sentinel-processor is responsible for publishing to NATS after the scrub processor has already run on the agent side.

### ADR-008: storage layer uses ClickHouse and MinIO plus Iceberg, not the Victoria stack

Date: 2026-06

Context: The initial Sentinel architecture planned to use VictoriaMetrics, VictoriaLogs, and VictoriaTraces as the storage layer inside sentinel-processor. This was evaluated against a lakehouse architecture using ClickHouse for hot storage and MinIO plus Apache Iceberg for cold storage.

Options considered:

Option A. Victoria stack. VictoriaMetrics for metrics, VictoriaLogs for logs, VictoriaTraces for traces. Three separate storage engines, three separate query languages, no cross-signal joins.

Option B. ClickHouse plus MinIO plus Iceberg. ClickHouse as the hot storage engine for all three signal types. MinIO plus Iceberg as the cold archive and source of truth. Single SQL query engine. Cross-signal joins possible.

Decision: Option B. ClickHouse plus MinIO plus Iceberg.

Reasoning: The Victoria stack would require the AI reasoning engine to query three separate systems and join results in application code to answer questions like "which traces correlate with this metric spike and what did the logs say during that window." This is the seven-tabs-open problem in code form. ClickHouse stores metrics, logs, and traces in the otel.metrics, otel.logs, and otel.traces tables with a shared schema and supports SQL joins across all three. The official OTel Collector clickhouseexporter exists and is production-ready. ClickHouse is battle-tested at Uber, Cloudflare, and ByteDance at scales far beyond Sentinel's initial requirements. MinIO plus Iceberg as the source of truth means the query engine is replaceable without moving data.

This decision does not affect sentinel-agent. The agent exports OTLP to sentinel-processor's OTel Collector. What sentinel-processor does with those signals is outside the agent's scope.

Consequences: sentinel-processor uses the clickhouseexporter. The Victoria stack is not deployed. This ARD records the decision so that contributors do not reintroduce Victoria stack references into the codebase.

### ADR-009: the agent does not perform anomaly detection

Date: 2026-06

Context: Some observability agents perform local anomaly detection and fire alerts without a central coordinator.

Options considered:

Option A. sentinel-agent includes threshold-based anomaly detection and can fire Slack alerts directly.

Option B. sentinel-agent collects and forwards. Anomaly detection is the job of sentinel-orchestrator running in sentinel-processor.

Decision: Option B. No anomaly detection in sentinel-agent.

Reasoning: Keeping the agent thin is what makes it trustworthy to enterprise security teams and easy to audit. An agent that does more than collect and forward is harder to explain to a security reviewer. Anomaly detection requires state, requires tuning, and requires coordination across multiple nodes to avoid false positives. sentinel-orchestrator can query ClickHouse across the entire cluster and compute a multi-node causal anomaly vector. An individual agent sees only one node and cannot perform meaningful multi-node correlation.

Consequences: sentinel-agent has no alerting configuration, no threshold configuration, and no external connections other than to sentinel-processor. This must be maintained permanently. Any contribution that adds alerting capability to sentinel-agent is rejected.

---

## 3. System context

```
                 customer infrastructure boundary
.................................................................
.                                                               .
.  node (one sentinel-agent daemonset pod per node)            .
.  ..........................................................   .
.  .                                                        .   .
.  .  OBI daemonset                                         .   .
.  .    eBPF kernel probes                                  .   .
.  .    HTTP, gRPC, DB, Kafka, GenAI traces                 .   .
.  .    TCP RTT, retransmits, network flows                 .   .
.  .    OTLP --> localhost:4317                              .   .
.  .                          |                             .   .
.  .  app pods (optional)     |                             .   .
.  .    sentinel-sdk          |                             .   .
.  .    internal spans        |                             .   .
.  .    OTLP --> localhost:4317                             .   .
.  .                          |                             .   .
.  .  sentinel-agent          v                             .   .
.  .    hostmetricsreceiver --+                             .   .
.  .    kubeletstatsreceiver --+                            .   .
.  .    filelogreceiver ------+                             .   .
.  .    otlpreceiver <--------+                             .   .
.  .         |                                              .   .
.  .    memorylimiter                                       .   .
.  .    resourcedetection (stamps cloud identity)           .   .
.  .    k8sattributes (stamps k8s metadata)                 .   .
.  .    sentinel-scrub (enforces PII policy)                .   .
.  .    batchprocessor                                      .   .
.  .         |                                              .   .
.  .    otlpexporter (mTLS)                                 .   .
.  .         |                                              .   .
.  ...........|..............................................   .
.             |                                                 .
.  customer VPC                                                 .
.  ...........|..............................................   .
.  .          v                                             .   .
.  .  sentinel-processor                                    .   .
.  .    OTel Collector (receives, validates cert)           .   .
.  .    NATS JetStream (fan-out to consumers)               .   .
.  .    ClickHouse (hot storage, 30 days)                   .   .
.  .    MinIO + Iceberg (cold archive, 90 days+)            .   .
.  .    context graph (Apache AGE)                          .   .
.  .    entity service                                      .   .
.  .    scrub-worker (day 5 PII elimination)                .   .
.  .    sync-worker (daily metrics + topology to cloud)     .   .
.  ..........................................................   .
.                                                               .
.................................................................
                              |
                    HTTPS, daily sync
                    metrics + topology only
                    no raw logs, no traces, no PII
                              |
                              v
                    sentinel-cloud (Sentinel SaaS)
                    reasoning engine
                    LLM layer
                    Slack alerts
                    ATLAS integration
```

sentinel-agent is the only component in the node boundary that has a connection to sentinel-processor. OBI and sentinel-sdk both connect to sentinel-agent on localhost. No component in the node boundary has a connection to sentinel-cloud or to any external endpoint.

---

## 4. Internal component relationships

The dependency graph within sentinel-agent is strictly layered. Higher-level packages may import lower-level packages. Lower-level packages must not import higher-level packages. Circular imports are forbidden.

```
Layer 0 (no internal dependencies):
  config
  identity

Layer 1 (imports layer 0):
  scrub         imports config

Layer 2 (imports layers 0 and 1):
  pipeline      imports config, identity, scrub

Layer 3 (imports layers 0, 1, and 2):
  cmd/agent     imports config, identity, scrub, pipeline
```

This means:

config imports nothing from sentinel-agent internals.
identity imports nothing from sentinel-agent internals.
scrub imports config for the policy file path.
pipeline imports config, identity to build the resource, and scrub to register the processor.
cmd/agent is the only package that imports everything.

A new package may only be added at the layer where its dependencies place it. A new dependency from a lower layer package to a higher layer package is not permitted.

---

## 5. Constraints

The following constraints cannot be changed by any PR. A change to a constraint requires a new ADR that explicitly supersedes the relevant existing ADR and a discussion in the PR that explains the new context that makes the change necessary.

C-1. sentinel-agent must never make a network call to any endpoint other than SENTINEL_OTLP_ENDPOINT and the cloud IMDS endpoints probed during startup. The cloud IMDS probes happen once at startup with strict timeouts and are not repeated. No other outbound calls are permitted at any time during the agent's lifecycle.

C-2. The sentinel-scrub processor must always run after the resourcedetectionprocessor and the k8sattributesprocessor in the pipeline and before the batchprocessor and the exporter. This order is not configurable. It ensures that identity attributes are stamped before scrubbing (preventing identity attributes from being accidentally scrubbed) and that PII is removed before export (ensuring no PII leaves the node).

C-3. OBI must not be merged into sentinel-agent. See ADR-002.

C-4. sentinel-agent must start successfully and collect host metrics even when sentinel-processor is unreachable. An unreachable downstream is not a startup failure. The agent queues and retries. It never exits due to downstream unavailability.

C-5. sentinel-agent must start successfully and collect host metrics even when OBI is not running. The otlpreceiver on localhost:4317 will simply receive no signals. The absence of OBI signals is not an error condition for sentinel-agent.

C-6. sentinel-agent must start successfully even when cloud identity detection fails for all probes. The bare_metal fallback always succeeds. A failure to detect cloud identity is a WARN log, not a startup error.

C-7. sentinel-agent must never log environment variable values that could contain secrets. The configuration package must log only the names and types of variables, never their values.

C-8. The binary must compile with CGO_ENABLED=0 for all code. The binary must be statically linked. No shared library dependencies at runtime.

C-9. No dependency with a GPL, LGPL, or AGPL license may be added. All dependencies must be Apache 2.0, MIT, BSD-2, BSD-3, or ISC licensed.

C-10. The agent must not perform anomaly detection, threshold evaluation, or alerting of any kind. See ADR-009.

---

## 6. Failure mode catalogue

This catalogue defines the expected behaviour for every known failure condition. Engineers debugging the agent in production and contributors writing new code must consult this catalogue. A failure mode not listed here must be added as a new entry before the code handling it is merged.

### 6.1 sentinel-processor unreachable at startup

Expected behaviour: the agent starts all receivers, begins collecting signals, and buffers them in the persistent queue. The exporter retries with exponential backoff starting at 1 second, capped at 30 seconds. The agent logs a WARN on the first failure and then logs the retry count at INFO on each subsequent attempt. The agent does not exit. Collection continues during the outage.

### 6.2 sentinel-processor unreachable during operation

Expected behaviour: same as 6.1 for the export path. Collection and pipeline processing continue uninterrupted. When the persistent queue reaches SENTINEL_QUEUE_MAX_SIZE, oldest batches are dropped and a WARN log is emitted with the count of dropped batches.

### 6.3 sentinel-processor connection restored after outage

Expected behaviour: the exporter detects the restored connection on the next retry attempt. The persistent queue drains in order. All signals queued during the outage are exported. No restart required.

### 6.4 AWS IMDSv2 probe timeout

Expected behaviour: the probe fails after 80ms. The agent logs DEBUG: "aws imds probe timed out, trying gcp". The GCP probe runs. If all probes time out, the bare_metal fallback activates. Total startup delay due to probe timeouts is under 500ms. The agent starts normally with cloud.provider=bare_metal.

### 6.5 Cloud IMDS endpoint returns unexpected response body

Expected behaviour: the agent logs WARN with the status code and URL. The probe is considered failed. The next probe runs. If all probes return unexpected responses, the bare_metal fallback activates.

### 6.6 sentinel-policy.yaml missing at startup

Expected behaviour: the agent logs ERROR: "sentinel-policy.yaml not found at {path}" and exits with code 1. The DaemonSet restartPolicy causes Kubernetes to restart the pod. The operator must ensure the ConfigMap delivering sentinel-policy.yaml is applied before sentinel-agent starts. sentinelhq provision generates both the agent manifest and the policy ConfigMap together to prevent this condition.

### 6.7 sentinel-policy.yaml invalid YAML

Expected behaviour: same as 6.6. The agent exits with code 1 and a descriptive parse error including the line number.

### 6.8 sentinel-policy.yaml updated at runtime (SIGHUP)

Expected behaviour: the scrub processor reloads the policy file. If the new policy is valid, it replaces the previous policy. Signals entering the processor after the reload use the new policy. If the new policy is invalid, the previous policy remains active and a WARN log is emitted with the parse error. The agent does not exit.

### 6.9 mTLS certificate rejected by sentinel-processor

Expected behaviour: the exporter logs WARN: "certificate rejected by sentinel-processor: {reason}" and retries. The agent does not exit. If the certificate is permanently invalid, signals queue until the queue is exhausted. The operator must rotate the certificate and trigger SIGHUP to reload it.

### 6.10 OBI not running or not exporting

Expected behaviour: the otlpreceiver receives no signals from OBI. This is not an error. The agent collects and exports host metrics and logs normally. When OBI starts exporting, its signals are received and processed. No restart required.

### 6.11 sentinel-sdk not installed in application pods

Expected behaviour: same as 6.10. The otlpreceiver receives no signals from sentinel-sdk. The agent operates normally. Network and protocol observability from OBI is still available. Application-layer business context is not available.

### 6.12 kubeletstatsreceiver cannot reach kubelet

Expected behaviour: the receiver logs WARN and retries at the collection interval. The agent continues collecting all other signals. If the kubelet is unreachable for more than five consecutive collection intervals, the receiver logs ERROR. The agent does not exit. When the kubelet becomes reachable, collection resumes automatically.

### 6.13 filelogreceiver log file deleted

Expected behaviour: the receiver detects the file deletion via inotify or polling. It logs DEBUG: "log file deleted: {path}, waiting for recreation". When the file is recreated, the receiver opens the new file and reads from offset 0 of the new file. It does not attempt to read the deleted inode.

### 6.14 filelogreceiver log rotation

Expected behaviour: the receiver detects the inode change or file truncation. It closes the current file handle and opens the new file. Reading continues from offset 0 of the new file.

### 6.15 Memory limit exceeded (memorylimiterprocessor)

Expected behaviour: the memorylimiterprocessor activates when resident memory exceeds SENTINEL_MEMORY_LIMIT_MIB. It begins refusing new signals from receivers, which causes the receivers to buffer signals at the source. When memory drops below the limit, the processor resumes accepting signals. The agent logs WARN: "memory limit exceeded, refusing new signals" at the start of the backpressure period and INFO: "memory recovered, resuming signal acceptance" when it ends.

### 6.16 SIGTERM received

Expected behaviour: the root context is cancelled. All receivers stop accepting new signals. The processor pipeline drains. The exporter Shutdown is called with a 30-second deadline. All successfully queued signals are exported. The agent logs INFO: "shutdown complete" and exits 0. If the drain deadline expires, the agent logs WARN: "shutdown deadline exceeded, force exiting" and exits with code 1.

### 6.17 Panic in any goroutine

Expected behaviour: the panic is caught by the errgroup recover wrapper. The panic message and stack trace are logged at ERROR level. The root context is cancelled, triggering graceful shutdown of all other goroutines. The agent exits with code 1 after the drain completes or the 30-second deadline passes.

---

## 7. Forward-looking notes

These are architectural directions that are expected but not yet decided. They are recorded here so that current implementation decisions do not inadvertently foreclose them.

FLN-1: SIGHUP reload of mTLS certificates. The current implementation requires a restart to rotate certificates. A future version should reload certificate files on SIGHUP without interrupting the OTLP connection.

FLN-2: sentinelhq status health endpoint. A future version should expose a local HTTP endpoint on a configurable port returning a structured JSON health response consumed by sentinelhq status. The endpoint must listen only on localhost.

FLN-3: Windows support. The current implementation targets Linux only. The OTel Collector hostmetricsreceiver supports Windows. A future version may add Windows support for VM deployments. eBPF via OBI remains Linux-only indefinitely. The identity package would require Azure IMDS support for Windows VMs which is already partially implemented.

FLN-4: ARM64 in CI. The current CI pipeline builds for linux/amd64 and linux/arm64 but runs tests only on linux/amd64. A future CI update should run the full test suite on linux/arm64 using an ARM runner.
