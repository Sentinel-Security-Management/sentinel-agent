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
