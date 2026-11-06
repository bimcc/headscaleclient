# Branch and release workflow

`main` is the shared integration branch for desktop and Android development.
Android remains a technical preview until its documented acceptance gates pass.
Merging source into `main` does not publish a stable release or change the
latest release tag.

## Changes and integration

- Start short-lived feature or fix branches from the latest `main`; avoid
  permanent branches for each platform.
- Keep application changes, contracts, tests, and relevant documentation together.
- Run the checks appropriate to the change before integration. CI verifies
  Windows, macOS, and Linux; the Android preview and macOS package workflows
  also run on relevant changes to `main`.
- Integrate reviewed, tested work through a pull request or an explicitly
  authorized direct merge. Preserve existing history and do not force-push
  shared branches.
- Delete a feature branch only after all its commits are reachable from
  the remote `main`. Preserve historical tags and unrelated worktrees.

## Releases

Prepare release version metadata, acceptance evidence, signing, and release
notes together. Create release tags and publish installers only as an explicit
release operation. Build artifacts from CI are verification artifacts; they
are not automatically stable releases.

## Consolidation on 2026-11-06

The work from `codex/android-preview` is integrated into `main`, including the
Android preview, single-install upgrades, identity reauthentication/removal,
and macOS service diagnostics. The Android workflow now follows `main`.

Known follow-ups remain: login-time snapshot refresh can time out while waiting
for the mutation lock, and inactive saved identities can misleadingly display
as signed in. This integration does not claim to resolve those issues.
