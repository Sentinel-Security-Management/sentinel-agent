### ADR-007: transport layer is direct OTLP to sentinel-processor, not through NATS

Date: 2026-06

Context: The full Sentinel architecture uses NATS JetStream as a message bus inside sentinel-processor. The question is whether sentinel-agent should export to NATS directly or to the OTel Collector inside sentinel-processor which then publishes to NATS.

Options considered:

Option A. sentinel-agent exports OTLP to NATS directly. The OTLP-to-NATS bridge runs on every node.

Option B. sentinel-agent exports OTLP to the OTel Collector inside sentinel-processor. The OTel Collector publishes to NATS after receiving and processing.

Decision: Option B. sentinel-agent exports OTLP to sentinel-processor's OTel Collector.

Reasoning: sentinel-agent's job is collection and identity. It should not know or care about the transport architecture inside sentinel-processor. If NATS sits between sentinel-processor and its consumers (ClickHouse writer, entity service, ghost-agent), sentinel-agent never needs to change when the transport layer evolves. This is a cleaner separation of concerns. The otlpexporter is battle-tested and supported by all OTel tooling. A NATS exporter for the OTel Collector exists but is less mature.

Consequences: sentinel-agent's exporter configuration is the standard otlpexporter. When the transport layer inside sentinel-processor changes, sentinel-agent is unaffected. The OTel Collector inside sentinel-processor is responsible for publishing to NATS after the scrub processor has already run on the agent side.
