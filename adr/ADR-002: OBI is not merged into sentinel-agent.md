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
