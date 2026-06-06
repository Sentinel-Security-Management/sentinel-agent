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
