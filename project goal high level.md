# Tally - high-level project goal

Tally is a polished, self-hosted home for following television shows. It combines a personal release calendar, shared TV metadata, profile-specific watch state, manual torrent search, operational visibility, and safe administration in one small service.

This document describes the current product direction and the architectural decisions that must remain true. `TODO.md` is the implementation roadmap and verification record; the code and tests are authoritative when this document is less specific.

## Product outcome

A user should be able to:

1. Open Tally and see the current month, upcoming releases, and the shows they follow.
2. Search TVmaze metadata, preview a show, and add it without blocking the rest of the interface.
3. Browse seasons, episodes, specials, release times, and countdowns.
4. Mark episodes watched or downloaded and manage favorites independently per profile.
5. Search configured Jackett indexers for a show or episode, filter results, copy a magnet, or manually send one selected result to the single configured torrent client.
6. Understand what background jobs, providers, notifications, and the application itself are doing.
7. Back up and validate the durable deployment state, then upgrade without risking the database.

Nothing is downloaded, selected, renamed, played, or managed automatically without an explicit user action.

## Product priorities

Tally should remain:

- Fast and responsive on desktop and mobile.
- Attractive, accessible, and consistent across light, dark, and system themes.
- Simple to run as one low-resource self-hosted service.
- Secure by default, with clear authorization boundaries and no accidental secret exposure.
- Predictable under provider failures, slow networks, repeated navigation, and background load.
- Observable through logs, job history, statistics, activity notifications, and actionable errors.
- Easy to upgrade, back up, restore, test, and extend without weakening the core boundaries.

## Current implementation

Tally is implemented as:

- A Go server with a repository-root `main.go` entrypoint.
- An embedded React/TypeScript frontend served by the same process.
- Permanent SQLite storage using transactional migrations, WAL, foreign keys, and validated snapshots.
- A Docker image and Compose deployment with one `tally` service and one durable `/config` volume.
- Schema version 5 at the current milestone.

The production frontend is built into `web/dist`; there is no separate frontend server, Redis, PostgreSQL, or required microservice.

## Core features

### Profiles and personal state

- Multiple immutable profiles with non-reused IDs.
- Profile-specific follows, favorites, appearance, calendar preferences, filters, statistics limits, inbox state, and episode watched/downloaded state.
- Profile selection remembered in the browser, with explicit sign-out before switching accounts.
- User-uploaded or generated avatars with validation and safe storage.
- Protected administrator profile `user0`, which cannot be deleted.

Shared shows, seasons, episodes, specials, metadata, provider identifiers, and cached images remain deployment-wide. Personal state must never be moved into shared metadata rows.

### Calendar, library, and discovery

- Month, week, and agenda calendar views with profile timezone and week-start preferences.
- Upcoming-release horizon grouped by day and show, including combined releases and known full-season labels.
- Local library of followed shows with favorites, episode progress, completion indicators, and compact actions.
- Ranked TVmaze search, suggestions, show previews, details overlays, poster/backdrop presentation, and keyboard-accessible Add actions.
- Durable queued show imports with optimistic add/remove, undo, retry, and shared metadata reuse across profiles.
- Direct watched/downloaded toggles, season actions, clear-history controls, and meaningful confirmation only for destructive actions.

### Manual torrent workflow

- One UI-managed Jackett search connection, configured by an administrator and stored in SQLite.
- Jackett searches its configured indexers through the all-indexers Torznab endpoint.
- Tally owns local query editing, result normalization, quality/format filters, sorting, history, and profile-scoped opaque selections.
- Results can be copied as magnets or manually sent as a magnet/torrent file to one configured torrent client.
- The current downloader adapter is qBittorrent using API-key/Bearer authentication.
- Torrent searches and sends are always explicit. A send never changes episode state.
- Connection secrets are concealed by default in the UI, are available only through operator settings, and are excluded from general APIs and logs.

Additional downloader adapters, multiple simultaneous torrent clients, automatic selection, automatic downloading, and torrent lifecycle management are intentionally deferred.

### Jobs, notifications, and operations

- Bounded metadata, maintenance, and backup jobs with editable five-field UTC schedules.
- Immediate enable/disable controls, job-specific presets, seven-day summaries, history filters, pause/resume after repeated failures, cancellation, and interrupted-run recovery.
- Live Jobs, Statistics, Logs, Inbox, Library, and Settings refresh without replacing active edits or overlays.
- Profile-scoped activity inbox categories for failures, releases, and successful activity; deployment logs retain the complete authorized history.
- Webhook and Discord notification services with validation, subscriptions, daily release scheduling, test delivery, safe JSON templates, and persistent delivery state.
- Statistics for provider requests, job outcomes, durations, failures, and bounded recent request history.
- Debug previews that explain behavior without running real jobs, changing real results, or sending external messages.

### Backups and upgrades

- UI-managed automatic backup enablement, retention, manual creation, archive download, and visible failure state.
- Archives contain durable SQLite/application state, avatars, credentials needed by the deployment, notification state, schedules, and activity data.
- Disposable caches, provider response caches, and environment secrets do not belong in backups.
- Archive creation verifies checksums, SQLite integrity, foreign keys, schema, avatars, and protected profile invariants.
- Schema upgrades take and validate a pre-upgrade snapshot, run transactionally, reject unsupported downgrades, and leave a recoverable failure state.
- Offline operator commands support backup, verification, restore, deletion, password reset, and explicit OIDC identity linking.

## Authentication and authorization

Supported modes are:

- Disabled authentication for trusted local deployments.
- Local password authentication with first-use setup, Argon2id hashes, recovery, forced password changes, progressive login delay, revocable sessions, CSRF protection, and bounded password policy.
- OIDC authorization-code login with discovery, state, nonce, PKCE, signature/token validation, and immutable issuer/subject identity mapping.

Authentication maps to immutable profile IDs; it is not embedded into show ownership or calendar data. Administrators alone manage deployment settings, profiles, integrations, jobs, backups, and system logs. Members can access only their own profile data, activity, inbox, library, and normal media views.

## Provider and background-work model

Every outbound request, including metadata, Jackett, qBittorrent, OIDC, webhook, and Discord traffic, passes through the provider coordinator. The coordinator provides:

- Admission and concurrency bounds.
- Per-provider rate limits and minimum intervals.
- Deadlines, bounded response sizes, and cautious redirect behavior.
- Request coalescing, caching, retries, `Retry-After` handling, and circuit breakers.
- Cancellation when the final caller leaves and persisted request/job telemetry.

Jobs are bounded, deduplicated, cancellable, restart-aware, and provider-safe. External provider failure must not make local readiness fail or crash the process. Failed entities are deferred instead of replaying an entire batch.

## Security and data rules

The following are permanent boundaries:

1. SQLite is the permanent database, not a temporary cache.
2. Shared metadata and profile-owned state remain separate.
3. Profiles are independent from authentication identities.
4. External provider IDs are mappings, never universal application IDs.
5. Provider-specific code stays behind narrow interfaces and the coordinator.
6. SQL values are parameterized and untrusted provider content is rendered as text.
7. There is no generic arbitrary-URL fetch endpoint.
8. Configured URLs, images, redirects, response sizes, and schemes are validated.
9. Passwords, hashes, API keys, tokens, authorization headers, and connection credentials never appear in ordinary logs or general APIs.
10. Operator-only connection settings may reveal saved secrets only as explicitly requested by the operator.
11. Backups and restore paths are validated against traversal, escaping symlinks, checksums, schema, and integrity failures.
12. Durable writes use transactions and revision checks where concurrent edits are possible.

## Deployment and configuration

Infrastructure configuration comes from environment variables and is validated at startup. This includes authentication mode, paths, address/port, timezone, concurrency, retry limits, retention defaults, and request/job limits.

The UI manages durable application settings in SQLite, including the Jackett connection, qBittorrent client, notification services, schedules, backup settings, and profile preferences. Secrets managed by the UI are durable state and are included in protected backups.

The standard deployment is:

```sh
docker compose up -d --build
```

The service runs as an unprivileged container user, drops capabilities, uses a named `/config` volume, exposes port 8080 by default on localhost, and provides liveness/readiness endpoints. Graceful shutdown stops new work, drains or cancels active work within a deadline, persists state, closes SQLite, and exits cleanly.

## Frontend experience

The frontend is local-data driven and should retain existing content while refreshing. TanStack Query owns read deduplication and cancellation; the transport pool bounds concurrent API work. Avoid request-per-keystroke behavior, blind focus refetches, effect loops, excessive retries, and sequential calls that do not improve correctness.

Navigation uses normal browser routes and Back/Forward behavior. The header remains visible, settings and profile pages share consistent containers, dialogs close during route cleanup, and active form edits are not overwritten by live refresh. Form fields have stable accessible names, visible focus states, concise feedback, and helper text positioned immediately beside the field it explains.

Errors should be specific and actionable. Use Undo for reversible actions and confirmation dialogs only for meaningful destructive operations.

## Non-goals

Do not add these unless the product requirements change:

- Movies or non-TV media management.
- Playback or media streaming.
- Media-library scanning, importing, moving, renaming, hardlinking, or duplicate detection.
- Automatic torrent search, selection, downloading, seeding, or lifecycle management.
- Multiple simultaneous downloader clients.
- Runtime plugin loading.
- Required Redis, PostgreSQL, or microservices.
- Generic outbound URL fetching.
- Unbounded provider requests, unbounded jobs, or background work that bypasses the coordinator.

## Engineering expectations

- Keep Go files focused on one responsibility, generally around 150-200 lines.
- Prefer domain packages under `internal/`, narrow dependency interfaces, concrete constructors, and instance-owned runtime state.
- Preserve API contracts, SQL transaction boundaries, migration/backup guarantees, authorization, and coordinator policies when refactoring.
- Update `TODO.md` whenever implementation state changes.
- Add failure-path tests with changes to authentication, authorization, providers, migrations, backups, queues, concurrency, or external protocols.
- Run `go test ./...`, `go vet ./...`, the frontend production build, and the relevant browser suite after changes. Concurrency changes also require the race detector.

## Current status

The planned core product is implemented and deployed locally. The current milestone is complete, with the latest verification covering the Go test suite, vet, frontend build, Docker build, browser regressions, lifecycle stress cases, and a healthy local container. Remaining work should be treated as focused maintenance, regression prevention, and explicitly approved product extensions rather than a redesign of the core model.
