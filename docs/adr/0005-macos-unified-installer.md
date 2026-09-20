# ADR 0005: One macOS installer per architecture

- Status: Accepted
- Date: 2026-09-20
- Supersedes: the separate independent/GUI-only package entry points in ADR 0004.

## Decision

Distribute one PKG per CPU. Include the GUI and inactive source-verified daemon
components, but do not put an active LaunchDaemon in the package payload.
Privileged scripts detect services before and after copying files. Only when
no external service exists do they register/start the bundled service. Existing
product-owned services are updated with state preserved. Existing external
services are neither stopped, updated, reconfigured, nor removed.

The Installer welcome and Installation Type pages explain service reuse,
managed upgrade, and automatic detection. Filesystem-based UI hints are advisory;
scripts additionally check processes, user installations and sockets. The actual
decision is logged and recorded in a root-owned `installation-mode` file.
The GUI exposes the service source and unavailable/permission/compatibility
states. Installation success does not imply a stopped external service is usable.

An external service does not get elevated by the installer: no untrusted external
binary or plist is executed. LocalAPI discovery and authorization remain
upstream's responsibility. External service reuse shares its profiles and
preferences; it is not a separate VPN/account session.

Detected simultaneous managed and external services cause a preinstall failure,
without stopping either. Migration requires explicit removal of the unwanted
service, then rerunning the same PKG. No generic overwrite option is offered for
official network extensions. Standard-user rights, signing and real-hardware
acceptance limits from ADR 0004 still apply.

## Verification

Native CI checks fresh installation, managed upgrades, stopped-app detection,
running/stopped external userspace daemon reuse, conflict rejection, external
process/payload/preferences preservation, uninstall safety and explicit migration.
The official signed GUI/Network Extension variants still need real-Mac tests;
the stopped-app fixture is only a detection test, not certification of those apps.
