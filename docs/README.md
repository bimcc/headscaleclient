# Documentation

This directory is the engineering source of truth for HeadscaleClient.
Architecture and task documents must be updated in the same change that alters
their corresponding behavior.

## Reading order

1. [Product scope](product/PRODUCT.md): goals, non-goals, users, and acceptance criteria.
2. [UI principles](product/UI.md): information architecture and interaction rules.
3. [Architecture](architecture/ARCHITECTURE.md): system boundaries and component ownership.
4. [Technical design](architecture/TECHNICAL-DESIGN.md): versions, APIs, data flow, and testing.
5. [Security model](architecture/SECURITY.md): trust boundaries and secret handling.
6. [Architecture decisions](adr/): decisions that are expensive to reverse.
7. [Roadmap](ROADMAP.md): delivery milestones.
8. [Task list](TASKS.md): executable work and current status.
9. [Verification](verification/): dated, sanitized platform and integration evidence.

## Document rules

- ADRs are immutable after acceptance. Supersede them with a new ADR.
- `TASKS.md` is updated continuously, not only at release time.
- Public frontend contracts are documented before adding new backend methods.
- Tailscale types never become public frontend contracts.
- Platform-specific behavior must document its fallback and failure state.
