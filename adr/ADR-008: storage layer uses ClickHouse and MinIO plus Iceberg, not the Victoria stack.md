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
