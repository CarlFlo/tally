# Validation

This file describes the current verification baseline and what must be checked for future changes. It is not a chronological test log; GitHub Actions and Git history retain that detail.

## Current baseline

The last completed full baseline was verified on 2026-09-17 for `feature/torrent-search-downloads`. The current `feature/torrent-automation-confidence` branch must complete the same full check before it is considered ready.

A complete branch check includes:

- `npm ci`
- `npm audit --audit-level=high`
- frontend TypeScript/Vite production build
- `go vet ./...`
- `go test -race ./...`
- `govulncheck ./...`
- full Playwright browser suite
- Docker image build
- Trivy container scan

Treat isolated reruns as diagnostic evidence, not as permission to ignore reproducible failures. The latest branch-head run, not an earlier green commit, is the final readiness signal.

## What the automated suite covers

Backend coverage includes authentication and sessions, profile isolation and roles, migrations and schema validation, backup/restore, settings revisions, schedules/jobs, provider coordination, cancellation, rate limiting, torrent protocols and automation, localization, notifications, and failure paths.

Torrent coverage includes normalized Jackett metadata, release parsing/evaluation, confidence versus preference, `.torrent` parsing and file-tree inspection, hard rejections, bad-infohash handling, bounded candidate fallback, retry gating, duplicate prevention, capability no-op/re-check behavior, scheduler integration, and ambiguous qBittorrent submission reconciliation by infohash.

Browser coverage includes primary navigation, same-document lifecycle behavior, profile/login flows, localization, Calendar and show overlays, library state, settings, schedules/backups, notifications/logs, Jackett search, episode-aware torrent confidence, qBittorrent submission, Torrent Automation, Previous Runs/feedback, feature toggles and disabled routes, Downloads, responsive behavior, and persistence across reloads where relevant.

Schema 9 is the current database version. Migration tests must continue to cover supported older schemas and backup restore/upgrade paths, including durable torrent automation runs, show policies, feedback, and bad-infohash state.

External-service protocol tests use local fixtures or in-memory requesters. Validation must not contact or modify an operator's personal TVmaze alternatives, Jackett, qBittorrent, Discord, webhook, or OIDC services unless a task explicitly requires and authorizes an integration test.

## Validation by change type

### Frontend behavior

Run the production frontend build and the focused browser tests. Run the full Playwright suite before finalizing a branch that changes shared navigation, dialogs, state management, localization, live updates, settings, or reusable components.

When the bug depends on navigation or lifecycle, keep the same browser document alive in the regression test. A reload can hide leaked listeners, stale state, aborted-request reuse, or cleanup bugs.

### Backend behavior

Run `go test ./...` and `go vet ./...`. Use `go test -race ./...` for concurrency, cancellation, shared state, jobs, provider coordination, live events, or shutdown behavior.

### Database, migrations, and backups

Verify forward migration from supported earlier schemas, schema validation, pre-upgrade snapshot behavior, and backup round-trips for newly durable state. Restore tests should prove relational state survives cascades and that failed restores leave the current database usable.

For torrent automation migrations, verify the run-history, feedback, bad-infohash, and show-policy tables survive backup/restore and retain their constraints. Detailed-run retention may prune old runs, but bad-infohash state needed to prevent repeat loops must remain durable.

### Authentication and authorization

Test backend enforcement directly; UI visibility is not an authorization boundary. Include negative cases for other profiles, non-administrators, stale or revoked sessions, and sensitive actions that require re-authentication.

Global torrent automation configuration and per-show download policy are administrative deployment state. Read-only history may be visible according to the product route, but mutation endpoints must enforce their intended authorization independently of hidden controls.

### External providers and torrent features

Use deterministic local fixtures. Verify exact request semantics, authentication headers, redirect/error behavior, bounded responses, cancellation, and secret redaction. Feature toggles must be tested at both API and UI boundaries.

For manual search, verify confidence appears only with authoritative episode context and that arbitrary free-text search is not assigned an authoritative confidence band. Confidence must remain separate from quality/size ranking.

For automatic downloads, prove all of the following as applicable:

- only High + Verified + non-rejected candidates are eligible;
- actual `.torrent` metainfo is inspected before qBittorrent submission;
- magnet-only candidates remain manual;
- executable/script, sample-only, wrong-show/episode, unsupported pack, malformed metadata, and known-bad hashes are rejected;
- a failed shortlisted candidate can fall through to the next bounded candidate;
- disabled search/download/automation capabilities cause a safe no-op and are re-checked before submission;
- retry backoff does not silently lower the confidence standard;
- an already downloaded episode is not submitted twice;
- ambiguous client errors are reconciled by the known infohash before retrying;
- run history never persists credentials or authenticated provider URLs.

### Localization

Test both the saved locale and fallback behavior. When an action changes locale, test any feedback produced by that same action so it is rendered using the newly active locale rather than a previously captured translation.

When bundled user-facing torrent text changes, update English and Ukrainian together and increment their catalog versions. Keep tests synchronized with the new bundled versions.

### Deployment-facing changes

Build the container and run the security scan. Verify readiness, graceful shutdown, file permissions, persistent state, and backup behavior when those areas change.

## Test design rules

Tests should reproduce the behavior being protected rather than merely assert implementation details. Prefer deterministic fixtures, controlled clocks, stable accessibility queries, and explicit failure-path assertions.

Do not weaken, skip, or rewrite a test solely to make a change pass. When a failure appears unrelated, inspect its trace/logs and changed surface first; rerun only to determine whether it is transient. Repeated failures are a bug until explained.

Keep tests synchronized with behavior changes. A feature is not complete until obsolete expectations are updated and meaningful regression coverage exists for the new behavior.

Browser fixtures for external integrations should provide local deterministic Jackett/qBittorrent behavior. Mock only the specific history/result state that would otherwise require nondeterministic timing; keep route, rendering, feedback, and responsive behavior real.

## Deployment-specific limits

HTTPS termination, reverse proxies, bind-mount permissions, network filesystems, and off-host backup storage depend on the operator's deployment. Filesystem watchers are designed for normal local filesystems; unusual NFS/SMB behavior may require a manual page revisit/refresh rather than hidden polling.
