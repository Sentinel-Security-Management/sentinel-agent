### ADR-003: sentinel-scrub runs inside sentinel-agent's pipeline, not in sentinel-processor

Date: 2026-06

Context: PII enforcement could be placed either in sentinel-agent on the node, or in sentinel-processor in the customer's VPC. Both are within the customer's infrastructure.

Options considered:

Option A. PII enforcement in sentinel-processor. Centralised, single place to update the policy, simpler agent.

Option B. PII enforcement in sentinel-agent. PII is hashed or dropped before any signal leaves the node. sentinel-processor receives already-clean signals.

Decision: Option B. sentinel-scrub processor runs as step four in the sentinel-agent pipeline.

Reasoning: The data sovereignty guarantee is that customer data stays within the customer's infrastructure. Even within the customer's infrastructure, there is a meaningful difference between PII existing on a single node versus PII flowing across the internal network to sentinel-processor and being stored in ClickHouse. Enforcing at the node boundary means PII never travels across any network connection in raw form, even an internal one. This is a stronger guarantee and simpler to audit. A security engineer auditing sentinel-agent can verify that the scrub processor runs before the exporter by reading the pipeline configuration. They do not need to audit sentinel-processor's internal pipeline as well.

Consequences: sentinel-agent has a direct dependency on the scrub package. The scrub package must be compiled into sentinel-agent. The sentinel-policy.yaml file must be present on every node, typically delivered via a ConfigMap or a volume mount. The agent must handle the case where the policy file is missing at startup (fail with a clear error) and where it is updated (reload on SIGHUP without restart).
