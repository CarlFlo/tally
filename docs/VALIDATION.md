# Validation

This file describes the current verification baseline and what must be checked for future changes. It is not a chronological test log; GitHub Actions and Git history retain that detail.

## Current baseline

The latest recorded full baseline was verified on 2026-09-19 for `refactor/test-pipeline-simplification`. Run the complete check on the latest branch head before treating a newer branch as ready.

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

CI runs the full check for every push. For pull requests from the same repository, that push run is the authoritative check and the duplicate pull-request job is skipped; fork pull requests still run the full check through the `pull_request` event.

## What the automated suite covers

Backend coverage includes authentication and sessions, profile isolation and roles, migrations and schema validation, backup/restore, settings revisions, schedules/jobs, provider coordination, cancellation, rate limiting, torrent protocols and automation, localization, notifications, and failure paths.

Torrent coverage includes normalized Jackett metadata (including optional uploader/author data and simultaneous `.torrent` + magnet alternatives), release parsing/evaluation, confidence versus preference, keyword and allowlist filtering, trusted-source preference ranking, release-upload delay-window behavior, runtime-aware MB/minute profiles, live/animated classification and override, `.torrent` parsing and file-tree inspection, guarded magnet fallback, durable post-magnet qBittorrent file-list verification, unsafe magnet removal/hash blocking, hard rejections, bad-infohash handling, bounded candidate fallback, retry gating, duplicate prevention, configurable completion-threshold episode marking, capability no-op/re-check behavior, scheduler integration, and ambiguous client submission reconciliation by infohash.

Browser coverage includes primary navigation, same-document lifecycle behavior, profile/login flows, localization, Calendar and show overlays, global downloaded/profile-watched library state, settings, schedules/backups, notifications/logs, Jackett search, episode-aware torrent confidence, expandable torrent-result evaluation, qBittorrent submission, Torrent Automation filters/trust controls and per-show enrollment, Previous Runs/feedback, feature toggles and disabled routes, Downloads, responsive behavior, and persistence across reloads where relevant. Settings coverage must verify that drafts do not persist before the shared Save changes action, Revert restores authoritative values, the fixed bottom bar appears only when dirty, and navigation warns while changes remain unsaved. One-shot show add/remove actions remain immediate.

Schema 14 is the current database version. Migration tests must continue to cover supported older schemas and backup restore/upgrade paths, including durable torrent automation runs, episode-to-infohash tracking, show policies, feedback, and bad-infohash state.

External-service protocol tests use local fixtures or in-memory requesters. Validation must not contact or modify an operator's personal TVmaze alternatives, Jackett, qBittorrent, Discord, webhook, or OIDC services unless a task explicitly requires and authorizes an integration test.

## Validation by change type

### Frontend behavior

Run the production frontend build and the focused browser tests. Run the full Playwright suite before finalizing a branch that changes shared navigation, dialogs, state management, localization, live updates, settings, or reusable components.

When the bug depends on navigation or lifecycle, keep the same browser document alive in the regression test. A reload can hide leaked listeners, stale state, aborted-request reuse, or cleanup bugs. Torrent Search route coverage should repeatedly cycle Search, Automation, and Previous Runs and must also prove that navigating away from a pending Search request stays responsive while the abandoned request is cancelled.

### Backend behavior

Run `go test ./...` and `go vet ./...`. Use `go test -race ./...` for concurrency, cancellation, shared state, jobs, provider coordination, live events, or shutdown behavior.

### Database, migrations, and backups

Verify forward migration from supported earlier schemas, schema validation, pre-upgrade snapshot behavior, and backup round-trips for newly durable state. Restore tests should prove relational state survives cascades and that failed restores leave the current database usable.

For torrent automation migrations, verify the run-history, feedback, bad-infohash, explicit show enrollment, persisted show type, per-show media-profile override, pending/completed magnet post-verification state, and global episode downloaded state survive backup/restore and retain their constraints. Detailed-run retention may prune old runs, but bad-infohash state needed to prevent repeat loops must remain durable.

### Authentication and authorization

Test backend enforcement directly; UI visibility is not an authorization boundary. Include negative cases for other profiles, non-administrators, stale or revoked sessions, and sensitive actions that require re-authentication.

Global torrent automation configuration and per-show enrollment are administrative deployment state. Enrollment list reads are scoped to the current profile's My Shows, while the enrollment value itself is global because the downloader is shared. Mutation endpoints must enforce administrator authorization independently of hidden controls.

### External providers and torrent features

Use deterministic local fixtures. Verify exact request semantics, authentication headers, redirect/error behavior, bounded responses, cancellation, and secret redaction. Feature toggles must be tested at both API and UI boundaries. Torrent automation has one scheduler-owned enabled state shared by Settings and System → Jobs; it defaults off while experimental and enabling it from either surface requires the experimental-feature confirmation.

For manual search, verify confidence appears only with authoritative episode context. Free-text queries containing one conservatively parsed season/episode may resolve against shared local metadata first and the cached TVmaze provider path second; exact unique matches may receive confidence and MB/min metadata, while ambiguous/unresolved queries remain unscored. Remote lookup must not add/follow or persist the show. Confidence must remain separate from quality/size/source ranking. Clicking a result should expand its inline evaluation without activating row actions, show a concise combined identity signal such as `Correct show and episode`, and present strengths and concerns without exposing the internal numeric score. Manual confidence is advisory: an inspected result must remain manually downloadable when the transport is otherwise usable, and a preliminary rejected classification is presented as neutral review/concern evidence rather than a disabled action. Actual fetched torrent payload safety and show/episode identity checks remain authoritative before submission.

Jackett uploader/author metadata is optional. Tests must cover uploader supplied through normalized Torznab metadata and the missing-uploader case. Provider/indexer identity is a separate preference signal and should remain usable even when uploader data is absent.

For automatic downloads, prove all of the following as applicable:

- the Torrent automation scheduler job must be enabled before scheduled automation runs, with no separate automation-settings master toggle;
- every automatic candidate must be High confidence and non-rejected;
- when Jackett exposes a retrievable `.torrent`, inspect it before submission and reject meaningful payload/identity failures rather than bypassing them with a magnet;
- when no usable `.torrent` is available, High-confidence magnets may use the metadata-only fallback path and the original decision must remain explicitly Unverified;
- automatic magnet submissions must create durable post-verification state; later scheduler runs inspect qBittorrent's resolved file list without busy-polling, reuse submission-time runtime/media ranges, and stop/remove + hash-block rejected payloads;
- release delay uses Jackett's reported upload time when available, holds until the first qualifying known release matures, and then evaluates all current candidates; unknown timestamps remain neutral;
- runtime-aware MB/minute filtering uses episode runtime first, show runtime second, treats missing inputs as neutral, and uses separate live-action/animated ranges;
- executable/script, sample-only, wrong-show/episode, unsupported pack, malformed metadata, and known-bad hashes are rejected;
- minimum seeders and include/exclude keyword rules filter before deep inspection;
- an active release-group or uploader allowlist is strict, including rejection of unknown/missing identity that cannot satisfy the allowlist;
- preferred release groups, uploaders, and providers/indexers materially influence ordering among valid candidates but never alter correctness confidence or override hard rejection;
- short keyword rules match release terms rather than accidental substrings in unrelated words;
- the settings snapshot and Previous Runs explanation retain the exact filter/trust rules and counts used for the historical decision;
- a failed shortlisted candidate can fall through to the next bounded candidate;
- disabled search/download/automation capabilities cause a safe no-op and are re-checked before submission;
- retry backoff does not silently lower the confidence standard;
- a globally downloaded episode is not considered for automation;
- only explicitly enrolled shows are considered while the global automation switch is enabled;
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

Browser tests must not rely on filename ordering or state leaked from earlier tests. Tests that change durable shared/profile state should restore it when the change is not itself the subject of later coverage. Keep broad navigation/performance stress coverage consolidated by failure mode; use separate specs for distinct risks such as request cancellation, populated-view cost, authentication, or destructive backup/restore behavior.

## Deployment-specific limits

HTTPS termination, reverse proxies, bind-mount permissions, network filesystems, and off-host backup storage depend on the operator's deployment. Filesystem watchers are designed for normal local filesystems; unusual NFS/SMB behavior may require a manual page revisit/refresh rather than hidden polling.
