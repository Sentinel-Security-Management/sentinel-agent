### ADR-009: the agent does not perform anomaly detection

Date: 2026-06

Context: Some observability agents perform local anomaly detection and fire alerts without a central coordinator.

Options considered:

Option A. sentinel-agent includes threshold-based anomaly detection and can fire Slack alerts directly.

Option B. sentinel-agent collects and forwards. Anomaly detection is the job of sentinel-orchestrator running in sentinel-processor.

Decision: Option B. No anomaly detection in sentinel-agent.

Reasoning: Keeping the agent thin is what makes it trustworthy to enterprise security teams and easy to audit. An agent that does more than collect and forward is harder to explain to a security reviewer. Anomaly detection requires state, requires tuning, and requires coordination across multiple nodes to avoid false positives. sentinel-orchestrator can query ClickHouse across the entire cluster and compute a multi-node causal anomaly vector. An individual agent sees only one node and cannot perform meaningful multi-node correlation.

Consequences: sentinel-agent has no alerting configuration, no threshold configuration, and no external connections other than to sentinel-processor. This must be maintained permanently. Any contribution that adds alerting capability to sentinel-agent is rejected.
