### ADR-001: sentinel-agent is an OTel Collector distribution, not a custom metrics collector

Date: 2026-06

Context: The initial design planned to use gopsutil to collect host metrics directly in a custom Go binary. This would require maintaining custom collection logic for CPU, memory, disk, network, filesystem, and process metrics across Linux kernel versions, distributions, and cloud environments.

Options considered:

Option A. Custom Go binary with gopsutil collecting all metrics directly. Full control, single dependency, simple to understand. Requires maintaining collection logic for every metric type across environments.

Option B. Pre-configured OTel Collector distribution. The OTel Collector Contrib project already includes hostmetricsreceiver, kubeletstatsreceiver, k8sclusterreceiver, dockerstatsreceiver, and filelogreceiver. These receivers are maintained by the OpenTelemetry community, tested across environments, and actively developed. Custom code is written only for what the OTel Collector does not already provide.

Decision: Option B. sentinel-agent is a pre-configured OTel Collector distribution.

Reasoning: The OTel Collector receivers cover every metric collection requirement identified for sentinel-agent across all four deployment environments. The hostmetricsreceiver covers what Node Exporter covers for VM and bare metal deployments. The kubeletstatsreceiver covers pod and node metrics for Kubernetes. The dockerstatsreceiver covers container metrics for Docker deployments. The filelogreceiver covers log collection across all environments. Writing custom collection code for these would add thousands of lines that the community already maintains and improves. The custom code in sentinel-agent is limited to cloud identity resolution and the sentinel-scrub processor, which have no OTel Collector equivalent.

Consequences: sentinel-agent's configuration is partly expressed as OTel Collector YAML pipeline configuration and partly as Go code wrapping the collector. This means contributors need to understand both the OTel Collector component model and the Go code that builds and runs it. This complexity is accepted because the alternative is greater.
