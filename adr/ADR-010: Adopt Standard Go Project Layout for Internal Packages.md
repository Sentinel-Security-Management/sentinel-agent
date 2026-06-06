### ADR-010: Adopt Standard Go Project Layout for Internal Packages

**Date:** 2026-06

**Context:** The initial SRS (Section 8) specified a flat top-level directory structure for `config`, `identity`, `pipeline`, and `scrub`. However, adhering to the standard Go project layout (`pkg/`, `internal/`, `cmd/`) provides better developer familiarity.

**Decision:** We will adopt the standard Go layout, moving the core agent packages (`config`, `identity`, `scrub`, `pipeline`) into `internal/agent/`.

**Reasoning:** Placing the core packages inside `internal/` enforces the Go compiler's strict visibility rules, guaranteeing that no external project can import our internal pipeline or scrub logic. The internal component relationships (Layer 0 through Layer 3) defined in ARD Section 4 remain fully intact within the `internal/` boundary.

**Consequences:** Section 8 of the SRS is hereby superseded to reflect the `internal/agent/` structure.