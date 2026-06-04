# Sentinel Branching Strategy

## Protected Branches

### main

The production-ready branch.

Rules:

* No direct pushes.
* Pull Requests required.
* At least 1 approval required.
* CI must pass.
* Force pushes disabled.

## Feature Branches

Naming:

feature/<feature-name>

Examples:

feature/agent-scheduler
feature/container-discovery
feature/postgres-support

## Bug Fix Branches

Naming:

fix/<issue>

Examples:

fix/memory-leak
fix/race-condition
fix/api-timeout

## Security Branches

Naming:

security/<issue>

Examples:

security/cve-2026-001
security/token-validation

Security branches may remain private until disclosure.

## Documentation Branches

Naming:

docs/<topic>

Examples:

docs/api-reference
docs/getting-started

## Release Process

All releases originate from main.

Tag examples:

v0.1.0
v0.2.0
v1.0.0

No release branches are required.

## Pull Request Requirements

* CI passes
* Documentation updated
* Tests included when applicable
* Reviewer approval obtained

## Commit Convention

feat:
fix:
docs:
refactor:
test:
chore:
security:

Examples:

feat(scanner): add container image scanning

fix(agent): resolve memory leak

docs(readme): update installation guide
