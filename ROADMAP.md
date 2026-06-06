# Sentinel Agent Roadmap

> Goal: Build a production-grade telemetry collection agent focused on identity-aware observability, security, and extensibility.

Status Legend:

* [ ] Not Started
* [~] In Progress
* [x] Completed

---

# Milestone 0 — Foundation

Goal: Create the project structure and development environment.

## Project Setup

* [ ] Repository created
* [ ] Go module initialized
* [ ] CI/CD pipeline configured
* [ ] Linting configured
* [ ] Unit testing framework configured
* [ ] Release workflow configured
* [ ] Build artifacts generation
* [ ] Versioning strategy defined

## Documentation

* [ ] README.md
* [ ] Architecture Overview
* [ ] ADR-001 OTel Collector Strategy
* [ ] Contributing Guide
* [ ] Development Guide

Exit Criteria:

* Agent builds successfully
* CI passes
* Repository ready for contributors

---

# Milestone 1 — Core Agent Runtime

Goal: Establish the Sentinel runtime.

## Runtime

* [ ] Configuration loader
* [ ] Environment variable support
* [ ] YAML configuration support
* [ ] Validation engine
* [ ] Dynamic reload support
* [ ] Graceful shutdown

## Health

* [ ] Health endpoint
* [ ] Readiness endpoint
* [ ] Liveness endpoint

## Internal Telemetry

* [ ] Agent metrics
* [ ] Agent logs
* [ ] Agent traces

Exit Criteria:

* Agent can start and run reliably

---

# Milestone 2 — OpenTelemetry Integration

Goal: Build Sentinel as an OTel distribution.

## Receivers

### Host

* [ ] hostmetricsreceiver

### Logs

* [ ] filelogreceiver

### Traces

* [ ] otlpreceiver

### Kubernetes

* [ ] kubeletstatsreceiver

### Containers

* [ ] dockerstatsreceiver

## Exporters

* [ ] OTLP gRPC exporter
* [ ] OTLP HTTP exporter

Exit Criteria:

* Agent can collect logs, metrics and traces

---

# Milestone 3 — Resource Discovery

Goal: Automatically understand the environment.

## Host Discovery

* [ ] Hostname detection
* [ ] OS detection
* [ ] Kernel detection
* [ ] CPU discovery
* [ ] Memory discovery
* [ ] Network interface discovery

## Container Discovery

* [ ] Docker discovery
* [ ] Container metadata collection
* [ ] Image metadata collection

## Kubernetes Discovery

* [ ] Node discovery
* [ ] Namespace discovery
* [ ] Pod discovery
* [ ] Deployment discovery
* [ ] Service discovery

Exit Criteria:

* Agent automatically inventories resources

---

# Milestone 4 — Identity Engine

Goal: Create stable entity identities.

## Entity Model

* [ ] Cluster ID
* [ ] Node ID
* [ ] Namespace ID
* [ ] Workload ID
* [ ] Service ID
* [ ] Container ID

## Correlation

* [ ] Resource correlation
* [ ] Ownership correlation
* [ ] Topology correlation

## Metadata

* [ ] Labels
* [ ] Tags
* [ ] Environment attributes

Exit Criteria:

* Every telemetry event contains stable identities

---

# Milestone 5 — Telemetry Enrichment

Goal: Add context before export.

## Metrics

* [ ] Resource enrichment
* [ ] Ownership enrichment
* [ ] Environment enrichment

## Logs

* [ ] Metadata injection
* [ ] Resource linking

## Traces

* [ ] Resource linking
* [ ] Entity linking

Exit Criteria:

* Telemetry contains contextual metadata

---

# Milestone 6 — Local Reliability Layer

Goal: Ensure reliable delivery.

## Queueing

* [ ] Local queue
* [ ] Retry queue
* [ ] Persistence support

## Delivery

* [ ] Backpressure handling
* [ ] Retry policies
* [ ] Circuit breaker

## Recovery

* [ ] Offline buffering
* [ ] Recovery after restart

Exit Criteria:

* No telemetry loss during temporary outages

---

# Milestone 7 — Security Foundation

Goal: Secure agent operations.

## Security

* [ ] TLS support
* [ ] mTLS support
* [ ] Certificate validation
* [ ] Secret management

## Authentication

* [ ] API token authentication
* [ ] Agent registration
* [ ] Agent identity validation

Exit Criteria:

* Secure communication established

---

# Milestone 8 — Sentinel Scrub Integration

Goal: Integrate privacy controls.

## Policy Engine

* [ ] Policy loading
* [ ] Policy validation
* [ ] Runtime enforcement

## Actions

* [ ] Redaction
* [ ] Masking
* [ ] Hashing
* [ ] Dropping

## Audit

* [ ] Policy audit logs
* [ ] Enforcement metrics

Exit Criteria:

* Sensitive data protected before export

---

# Milestone 9 — Deployment Support

Goal: Simplify deployment.

## Linux

* [ ] Systemd package
* [ ] RPM package
* [ ] DEB package

## Containers

* [ ] Docker image
* [ ] Docker Compose support

## Kubernetes

* [ ] Helm Chart
* [ ] Operator evaluation
* [ ] Auto configuration

Exit Criteria:

* Agent deployable in all supported environments

---

# Milestone 10 — Production Readiness

Goal: Enterprise-grade stability.

## Testing

* [ ] Unit tests
* [ ] Integration tests
* [ ] Performance tests
* [ ] Stress tests

## Scalability

* [ ] 1K entities
* [ ] 10K entities
* [ ] 100K entities

## Reliability

* [ ] Failure injection tests
* [ ] Network outage tests
* [ ] Resource exhaustion tests

Exit Criteria:

* Production-ready release candidate

---

# Milestone 11 — GA Release

## Release Checklist

* [ ] Documentation complete
* [ ] API stable
* [ ] Configuration stable
* [ ] Upgrade strategy defined
* [ ] Migration strategy defined
* [ ] Security review completed
* [ ] Performance benchmark completed

Release Target:

v1.0.0

Success Criteria:

* Production deployments active
* Stable telemetry collection
* Reliable identity correlation
* Secure telemetry export
* Scrub integration operational

---

# Future Milestones

## v1.1

* [ ] eBPF integration via OBI
* [ ] Advanced topology enrichment
* [ ] Entity relationship export

## v1.2

* [ ] Edge processing
* [ ] Local anomaly detection
* [ ] Advanced policy engine

## v2.0

* [ ] Autonomous remediation hooks
* [ ] Multi-cluster federation
* [ ] Intelligent telemetry optimization
