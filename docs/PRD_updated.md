# sentinel-agent
# Product Requirements Document

Version: 1.0
Status: Active
License: FSL-1.1-Apache-2.0
Repository: github.com/Sentinel-Security-Management/sentinel-agent
Mirror: gitlab.com/sentinel-security-management/sentinel-agent
Parent product: Sentinel
Umbrella platform: The Deployment Lab
Companion documents: SRS.md, ARD.md

---

## 1. Purpose

sentinel-agent is the node-level telemetry collection and identity layer of the Sentinel observability platform. It is a pre-configured OpenTelemetry Collector distribution that runs on every node. Its job is to collect host and infrastructure signals, receive network and protocol signals from OBI and application signals from sentinel-sdk, resolve the canonical cloud and Kubernetes identity of the node, stamp that identity on every signal, enforce the PII policy defined in sentinel-policy.yaml, and forward the unified authenticated stream to sentinel-processor inside the customer's own VPC.

sentinel-agent is the single point through which all telemetry on a node passes before reaching sentinel-processor. Nothing passes it without identity stamping. Nothing leaves it toward sentinel-processor without mTLS authentication and PII enforcement.

---

## 2. Problem

### 2.1 Agent sprawl

A production node in a modern engineering organisation typically runs five to seven independent agents simultaneously. Node Exporter collects host metrics. Fluent Bit collects logs. An OTel Collector forwards traces. cAdvisor monitors containers. A cloud agent ships data to the provider's own monitoring. A security agent runs EDR. Each of these has its own configuration format, its own upgrade cadence, its own failure mode, its own connection to a backend, and its own resource footprint.

The cost is not the CPU or memory of any individual agent. It is the engineering time required to configure, upgrade, debug, and correlate data from components that share no common identity model and no common pipeline. When one agent fails silently, telemetry gaps appear with no obvious cause. When one is misconfigured, data is lost or mis-attributed. On a fleet of one thousand nodes this is not a maintenance burden. It is a continuous tax on engineering capacity.

### 2.2 Identity fragmentation

Even when multiple agents run correctly, the signals they produce are not correlated. A CPU spike reported by Node Exporter is a number attached to a hostname. A slow HTTP trace reported by the OTel Collector is a span attached to a service name. A log line reported by Fluent Bit is a string attached to a file path. None share a canonical identity that says: this metric, this trace, and this log line all came from the same pod, in the same namespace, on the same node, in the same AWS account, in the same region, at the same time.

Correlating them during a 3am incident requires querying three separate systems, matching on imprecise fields, and building the connection manually. This is why engineers have seven tabs open. The data exists. The identity that would connect it does not.

### 2.3 PII in the telemetry pipeline

Observability data contains personal data. HTTP traces carry user identifiers in URL paths and headers. Database traces carry query parameters that may include email addresses and account numbers. Log lines contain whatever the developer chose to log, which is frequently more than they should have. In a traditional observability stack this data flows to a vendor's cloud storage with minimal control over what is collected and what is retained.

Existing mitigations require customers to configure masking rules in a vendor's SaaS platform after data has already been transmitted across the network. Sensitive data crosses the boundary before any protection is applied. Misconfigured rules result in unmasked PII in vendor storage with no notification.

### 2.4 Data sovereignty

Regulations including GDPR, HIPAA, and PCI-DSS impose obligations on where personal data can be stored and who can access it. Regulated industries and security-conscious engineering organisations cannot use observability platforms that transmit telemetry to vendor cloud infrastructure, regardless of the vendor's compliance claims. No existing single-agent observability solution combines agent consolidation, canonical identity, PII enforcement at the node boundary, and complete data sovereignty.

---

## 3. Solution

sentinel-agent is a pre-configured OTel Collector distribution. It is not a custom metrics collector. It does not reimplement what the Kubernetes kubelet, cAdvisor, or Node Exporter already provide. It uses the OTel Collector's existing receivers to collect from those sources and adds three things that no existing solution provides together:

First, canonical cloud and host identity resolution. sentinel-agent queries AWS IMDSv2, GCP Compute Metadata, or Azure IMDS at startup and produces a single OTel Resource containing cloud.provider, cloud.region, host.id, host.name, and Kubernetes metadata. This resource is stamped on every signal from every source on the node, creating a shared identity across metrics, logs, and traces.

Second, PII enforcement at the node boundary. The sentinel-scrub processor runs inside the sentinel-agent pipeline before the exporter. No signal leaves the node toward sentinel-processor without passing through PII enforcement. The policy is defined in sentinel-policy.yaml, generated by the sentinel-scrub scanner, and auditable by any party.

Third, authenticated forwarding. The single OTLP/gRPC connection leaving the node uses mTLS. sentinel-processor validates the certificate fingerprint before accepting any signals. The agent is the only component that talks to sentinel-processor from the node. OBI and sentinel-sdk both export to sentinel-agent on localhost, not directly to sentinel-processor.

---

## 4. Users

### Platform engineer

Deploys and operates sentinel-agent across the fleet. Evaluates it against the existing agent stack. Measures its resource footprint against production workloads. Needs a single command to deploy and a single manifest to review.

Success: sentinel-agent deployed across a twenty-node EKS cluster in under fifteen minutes using sentinelhq provision. No manual receiver configuration. Metrics, logs, and traces visible in ClickHouse within sixty seconds of deployment.

### Security and compliance engineer

Reviews sentinel-agent before approving deployment. Reads the source code. Audits what network connections it makes, what files it reads, what capabilities it requires, and what it does with PII. Needs a definitive and unambiguous answer to: does this agent transmit data outside our infrastructure.

Success: source code audit completed in under two hours. sentinel-policy.yaml explains exactly what gets hashed and what gets dropped. The exporter configuration shows a single connection to an internal endpoint. No external calls exist in the codebase.

### On-call SRE

Consumes the output of sentinel-agent indirectly through Slack alerts and the ATLAS incident block. Needs cloud.region, k8s.namespace.name, and k8s.pod.name to be present and correct on every signal so that a 3am alert immediately identifies the affected infrastructure.

Success: Slack alert says "payment-service, pod payment-7f8b-x2k, namespace production, node ip-10-0-1-42, us-east-1" without the engineer having to look anything up.

### Open source contributor

Evaluates sentinel-agent as a contribution target. Needs clear package boundaries, documented decisions, and a path to contribute a change to a single receiver or processor without understanding the entire Sentinel platform.

Success: the ARD answers every architectural question a contributor would have before opening a PR.

---

## 5. Goals

### MVP goals

G-1. Agent consolidation. Replace Node Exporter, Fluent Bit, and a standalone OTel Collector with a single sentinel-agent binary covering all three signal types across all four deployment environments: Kubernetes, Docker, VM, and bare metal.

G-2. Canonical identity. Every signal emitted by sentinel-agent carries cloud.provider, cloud.region, host.id, and host.name. Signals from Kubernetes nodes also carry k8s.node.name, k8s.pod.name, k8s.namespace.name, k8s.container.name, and k8s.pod.uid. These are resolved once at startup and stamped consistently on every metric, log record, and trace.

G-3. PII enforcement at node boundary. sentinel-scrub processor runs as step four in the pipeline, after identity stamping and before export. No signal leaves the node with a PII field that violates sentinel-policy.yaml.

G-4. OBI and SDK aggregation. sentinel-agent receives OTLP from OBI and from sentinel-sdk on localhost:4317. Both pass through the same identity stamping and PII enforcement pipeline before forwarding.

G-5. Authenticated forwarding. All signals are exported to sentinel-processor via OTLP/gRPC with mTLS. Nodes without valid certificates cannot deliver telemetry.

G-6. Environment detection. sentinel-agent detects its deployment environment at startup and activates the correct receiver set without manual configuration.

G-7. Resilience. sentinel-agent continues collecting and buffering signals when sentinel-processor is unreachable and resumes export without data loss when the connection is restored. It never crashes due to a downstream outage.

G-8. Low resource footprint. sentinel-agent consumes less than 100MB RAM and less than 1% CPU per core under normal operation on a production node.

### Post-MVP goals

G-9. SIGHUP hot reload of sentinel-policy.yaml and mTLS certificates without restart.

G-10. Local HTTP health endpoint for liveness and readiness probes.

G-11. Structured health data exposed for sentinelhq status.

---

## 6. Non-goals

The following are explicitly outside the scope of sentinel-agent. Any contribution that introduces these belongs in a different module.

- HTTP, gRPC, or database protocol tracing. OBI handles this.
- PII field discovery or detection logic. sentinel-scrub scanner handles this. sentinel-agent only enforces the policy sentinel-scrub generates.
- Anomaly detection or alerting. sentinel-orchestrator handles this.
- Any outbound network call to sentinel-cloud or any endpoint outside the customer's VPC.
- Application code instrumentation. sentinel-sdk handles this.
- Log parsing, structured log extraction, or log transformation. sentinel-agent ships raw log lines. Processing belongs in sentinel-processor.
- Business logic context such as tenant_id, order_id, or custom spans. sentinel-sdk handles this.
- Custom eBPF code for TCP tracking or protocol parsing. OBI handles this.

---

## 7. Feature requirements

### 7.1 Environment detection and receiver activation

sentinel-agent detects its environment at startup and activates the appropriate receiver set.

Kubernetes mode is detected by the presence of /var/run/secrets/kubernetes.io/serviceaccount. Active receivers: kubeletstatsreceiver for node and pod metrics from the kubelet API, k8sclusterreceiver for cluster-level metrics and K8s events, filelogreceiver reading from /var/log/pods, otlpreceiver on localhost:4317 for OBI and sentinel-sdk.

Docker mode is detected by the presence of /var/run/docker.sock and the absence of the Kubernetes service account path. Active receivers: dockerstatsreceiver for container CPU, memory, and network metrics, hostmetricsreceiver for node-level CPU, memory, disk, and network, filelogreceiver reading from /var/lib/docker/containers, otlpreceiver on localhost:4317.

VM and bare metal mode activates when neither Kubernetes nor Docker is detected. Active receivers: hostmetricsreceiver for CPU, memory, disk, filesystem, network, and system load, filelogreceiver reading from configurable paths defaulting to /var/log, otlpreceiver on localhost:4317.

### 7.2 Cloud identity resolution

sentinel-agent resolves cloud identity at startup. Probes run sequentially with an 80ms timeout each. First successful probe terminates the sequence. Total resolution time must not exceed 500ms. The agent must never fail to start due to a probe timeout.

AWS. PUT to IMDSv2 token endpoint with X-aws-ec2-metadata-token-ttl-seconds: 60. On 200 response, use the token to GET placement/region and instance-id. Sets cloud.provider=aws, cloud.region, host.id, cloud.account.id, host.type.

GCP. GET to metadata.google.internal/computeMetadata/v1/instance/id with Metadata-Flavor: Google header. On 200 response, GET computeMetadata/v1/instance/zone and parse the region. Sets cloud.provider=gcp, host.id, cloud.region.

Azure. GET to 169.254.169.254/metadata/instance with api-version=2021-02-01 and Metadata: true header. Parse JSON response for compute.location, compute.vmId, compute.name. Sets cloud.provider=azure, cloud.region, host.id, host.name.

Bare metal fallback. Activates when all probes fail or time out. Sets cloud.provider=bare_metal, host.name from os.Hostname.

### 7.3 Kubernetes metadata enrichment

When running in Kubernetes, sentinel-agent reads metadata from downward API environment variables injected by the DaemonSet manifest. These are combined with the cloud identity into the single OTel Resource stamped on all signals.

The DaemonSet manifest must inject: NODE_NAME from spec.nodeName, POD_NAME from metadata.name, POD_UID from metadata.uid, POD_NAMESPACE from metadata.namespace, CONTAINER_NAME from the container name field.

These map to OTel attributes: k8s.node.name, k8s.pod.name, k8s.pod.uid, k8s.namespace.name, k8s.container.name.

### 7.4 Pipeline processor order

Every signal entering the pipeline passes through processors in this exact order:

1. memorylimiterprocessor. Safety gate. Drops signals when memory exceeds the configured limit to prevent OOM. Always first.

2. resourcedetectionprocessor. Stamps cloud.provider, cloud.region, host.id, host.name on every signal using the resolved cloud identity.

3. k8sattributesprocessor. Stamps k8s.pod.name, k8s.namespace.name, k8s.node.name, k8s.container.name on every signal when in Kubernetes mode.

4. sentinel-scrub processor. Reads sentinel-policy.yaml. Hashes attributes listed with action: hash using SHA-256. Drops attributes listed with action: drop. Detects PII by value using email regex, credit card Luhn check, JWT pattern, and phone number pattern regardless of attribute name. Must run after steps 2 and 3 so identity attributes are already stamped and can be excluded from scrubbing. Must run before step 5 so no PII reaches the exporter.

5. batchprocessor. Batches signals for efficient export. Flushes at 8192 signals or 5 seconds, whichever comes first.

### 7.5 OBI and sentinel-sdk signal aggregation

OBI exports OTLP to localhost:4317. sentinel-sdk exports OTLP to localhost:4317. Both are received by the otlpreceiver inside sentinel-agent and enter the same pipeline as signals from all other receivers. OBI and sentinel-sdk signals receive the same resource stamping and PII enforcement as host metrics and logs. Neither OBI nor sentinel-sdk connects to sentinel-processor directly.

This design means OBI traces, which may contain protocol-level data including IP addresses and query patterns, pass through sentinel-scrub before leaving the node.

### 7.6 mTLS authenticated export

sentinel-agent exports to sentinel-processor via OTLP/gRPC with mutual TLS. The agent presents a client certificate. sentinel-processor validates the certificate fingerprint against its allowlist. Certificate paths are configurable via environment variables. The agent logs a clear error and retries with exponential backoff if the certificate is rejected.

### 7.7 Resilience and buffering

On sentinel-processor unavailability, signals are queued using the OTel Collector's persistent queue exporter. The queue spills to a local file when the in-memory limit is reached. On reconnection, the queue drains in order. Signals are never silently dropped. Every dropped signal due to queue overflow is logged at WARN level with a count.

### 7.8 Graceful shutdown

On SIGTERM, sentinel-agent stops all receivers, drains the pipeline, and calls Shutdown on the exporter with a 30-second deadline. If the deadline passes, it force-exits with a non-zero code. All signals successfully queued before SIGTERM are exported before shutdown completes.

---

## 8. Deployment

sentinel-agent is deployed by sentinelhq provision. Engineers do not write deployment manifests manually.

sentinelhq provision detects the environment and cloud provider, generates the correct manifest for sentinel-agent, OBI, and sentinel-processor, and outputs a directory of ready-to-apply files. The engineer reviews the generated files and applies them with kubectl apply -f sentinel/ or docker compose up.

For Kubernetes deployments, sentinelhq provision generates a DaemonSet manifest, a ClusterRole and ClusterRoleBinding for the kubelet and K8s API permissions sentinel-agent requires, and a ConfigMap containing the resolved OTel Collector configuration.

---

## 9. Licensing

sentinel-agent is released under the Functional Source License 1.1 with Apache 2.0 as the change license, written as FSL-1.1-Apache-2.0.

Under FSL-1.1, anyone may use, modify, and distribute sentinel-agent for any purpose other than competing commercially with Sentinel. Competing observability vendors and APM platforms may not use sentinel-agent in a competing commercial product for two years from each release date. The license converts automatically to Apache 2.0 two years after each release date. The source code is fully readable and auditable by any party including enterprise security teams at any time.

---

## 10. Success criteria

The sentinel-agent MVP is complete when all of the following are true:

1. sentinelhq provision generates a working manifest for EKS, plain Docker, and a bare metal Linux host without manual editing of any generated file.

2. sentinel-agent deployed on a three-node k3d cluster shows all node and pod metrics in ClickHouse within sixty seconds of kubectl apply.

3. cloud.provider, host.id, k8s.pod.name, and k8s.namespace.name are present on 100% of metrics, logs, and traces in ClickHouse for signals originating from Kubernetes nodes.

4. OBI traces visible in ClickHouse carry the same resource attributes as host metrics from the same node, confirming that OBI signals pass through the sentinel-agent pipeline correctly.

5. RAM consumption stays under 100MB per node during steady-state collection on a node running twenty pods.

6. sentinel-agent survives a five-minute sentinel-processor outage, queues signals to disk, resumes export without restart, and delivers all queued signals after the connection is restored.

7. A security engineer can answer "does this agent transmit data outside our VPC" definitively by reading the exporter configuration and sentinel-policy.yaml without reading any Go source code.

---

## 11. Risks

Risk: OTel Collector receiver API changes between major versions require sentinel-agent to update internal configuration. Likelihood: medium. Impact: medium. Mitigation: pin the OTel Collector version in go.mod and the Dockerfile. Run integration tests on version upgrade PRs before merging.

Risk: kubeletstatsreceiver requires RBAC permissions that a customer security team rejects as too broad. Likelihood: medium. Impact: high. Mitigation: document the minimum required RBAC as a read-only ClusterRole scoped to nodes and pods only. Make kubelet metrics an optional feature flag. Document what is lost when disabled.

Risk: hostmetricsreceiver has gaps relative to Node Exporter for specific Linux subsystems on older kernels. Likelihood: low. Impact: medium. Mitigation: benchmark hostmetricsreceiver against Node Exporter on the target kernel versions in CI. Document known gaps explicitly in the README.

Risk: IMDSv2 endpoint IP address changes on AWS breaking cloud identity detection. Likelihood: low. Impact: high. Mitigation: integration tests against real AWS endpoints in CI on a weekly schedule. Fallback to IMDSv1 if IMDSv2 returns non-200. Cloud identity failure falls back to bare_metal, not a crash.

Risk: sentinel-scrub misconfiguration causes legitimate operational data to be dropped, breaking dashboards or alerts. Likelihood: medium. Impact: high. Mitigation: sentinel-scrub processor defaults to hash not drop for uncertain fields. A dry-run mode logs what would be hashed or dropped without modifying signals. The scanner generates policy with confidence levels so engineers review before applying.

Risk: memory exhaustion during a long sentinel-processor outage if the persistent queue grows without bound. Likelihood: low. Impact: high. Mitigation: memorylimiterprocessor configured as step one in the pipeline. Persistent queue has a configurable maximum size. When the maximum is reached, oldest signals are dropped first and a WARN log is emitted with a count.
