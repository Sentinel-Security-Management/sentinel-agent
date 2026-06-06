### ADR-004: cloud identity resolution uses direct IMDS probes, not the OTel resourcedetectionprocessor

Date: 2026-06

Context: The OTel Collector Contrib project includes a resourcedetectionprocessor that can detect cloud identity from AWS, GCP, and Azure metadata endpoints. This could replace the custom identity package.

Options considered:

Option A. Use resourcedetectionprocessor from OTel Collector Contrib.

Option B. Write a custom cloud identity resolver in the identity package.

Decision: Option B for AWS IMDSv2 specifically. Option A is used as a fallback for attributes the custom resolver does not cover.

Reasoning: The resourcedetectionprocessor does not implement the IMDSv2 token-based flow correctly as of the OTel Collector version pinned in this project. IMDSv1 is being deprecated on new AWS instances. A custom resolver that correctly implements the PUT-then-GET IMDSv2 flow is required to reliably detect cloud identity on modern AWS infrastructure. For GCP and Azure the resourcedetectionprocessor is sufficient. The custom identity package handles AWS IMDSv2 and falls back to the resourcedetectionprocessor for GCP and Azure where its implementation is correct.

Consequences: The identity package must be maintained across AWS IMDS API changes. Integration tests must run against real AWS IMDS endpoints on a scheduled basis in CI to catch any API changes before they affect customers.
