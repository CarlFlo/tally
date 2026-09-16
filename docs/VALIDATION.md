# Validation record

## Profile roles and generated identity migration

Verified 2026-09-15 on `refactor/profile-permissions`.

- Schema 6 migrates sequential profile IDs to opaque generated IDs, rewrites every profile-owned relation, keeps legacy aliases only for compatibility, removes the obsolete profile counter, and validates role integrity.
- Administrator authorization is role-based across API routes, capabilities, logs/inbox visibility, settings, backups, jobs, and browser navigation. No runtime authorization check depends on a special profile ID.
- The database prevents demoting or deleting the last administrator while other profiles remain. Deleting the sole remaining profile is allowed, and a newly created first profile becomes administrator automatically.
- Local-auth administrator demotion/deletion re-authenticates the acting administrator with their own password. Tests reject using the target administrator's password for another actor's request.
- Profile creation/deletion and administrator grant/revoke activity is logged and maps to the subscribable `profile_access_changed` notification event.
- Profile colors accept validated six-digit HTML colors and automatic initials use the first two Unicode characters.
- The frontend production build, `go vet ./...`, `go test -race ./...`, the full Playwright suite, and the Docker build pass. Legacy schema/backup upgrade tests cover version-one through schema-six restore behavior.


Updated 2026-09-12. Current checks use Go 1.27.1, Node.js 22.14.0, Chromium 153, and Docker Engine 29.4.0.

## Header, calendar, onboarding, and backups

Verified 2026-09-12. All 255 Go application/test files remain below 150 lines (largest: 143).

- The full Go test suite passes on Windows and in the Linux image build. `go test -race ./...`, `go vet ./...`, the TypeScript/Vite build, the native executable build, and the Docker build pass.
- All 12 Playwright workflows pass against temporary local fixture servers. Existing discovery, episode progress, torrent setup/test/send, profiles, browser history, notification services, and administration regressions remain covered.
- New browser checks cover the header profile menu, Escape/focus and route closure, sign-out failure/retry and cross-tab sign-out, hidden internal profile IDs, System tabs, passwordless-profile setup, exact password preservation, Add profile from login, and member-only activity visibility.
- Scheduling checks verify immediate toggle persistence while preserving unsaved cron edits, separate Save schedule, reload persistence, and activity records. Backups checks cover saved retention, completed job notification, native browser archive downloads, and the replacement Backups tab.
- Calendar checks cover one card per show/day, known full-season labels, comma-separated episode numbers, access to individual episodes inside a release group, day-grouped horizon, and expanding all 1,005 test shows without a cap. Existing controlled-clock release countdown checks still pass.
- Backend regressions cover setup refusing existing credentials and signed-in account switching; bounded/non-reused profile creation; password/display-name validation; inbox visibility, counts, seen/dismiss/clear markers and future arrivals; filtered personal logs; download authorization, verified records, traversal and escaping symlinks; runtime retention updates; declared season sizes; stale schedule revisions and logging; removed ENV values; and sequential schema upgrades with validated snapshots.
- Backup restore verifies inbox state/dismissals and backup settings alongside credentials, activity, browser appearance, notification outbox, profiles, follows, favorites, queued actions, and schedules. No personal integration was contacted or modified by verification.

The local container is healthy at `http://localhost:8080` on the existing `mediamanager_tally-config` volume. Schema 5 retains both profiles and all three schedules, including the existing custom hourly metadata schedule. Backup retention is stored in SQLite. The pre-upgrade snapshot `/config/pre-upgrade-v4-1789239607541044796.db` is present; `/config/app.db` remains mode 0600. Thirteen page routes and the inbox are readable. Existing member requests to settings/jobs/statistics/backups return 403; member logs and inbox contain only that profile's activity. Current assets: `index-B9Z0Vp-I.js` and `index-B5wzQXd8.css`.

Screenshots in `docs/screenshots` include compact schedules, grouped calendar releases, backup settings, notification inbox, and Add profile, plus the existing desktop/mobile workflow captures.

## Library actions, logs, notifications, and administration

Verified 2026-09-12 against isolated fixtures. All Go application/test files remain below 150 lines (the current guideline permits about 150-200).

- `go test ./...`, `go test -race ./...`, `go vet ./...`, the TypeScript/Vite build, and the native Windows build pass.
- All nine Playwright workflows pass. New workflows cover show/card menus, Escape/Back closure, clear watch history with downloaded markers retained, Shift-remove without confirmation, action-filtered log search, and named show jobs. Existing personal-profile, authentication, discovery, torrent, settings, scheduling, and statistics workflows continue to pass.
- Browser tests also cover server-stored light/dark/system login appearance, administrator-role navigation and APIs, self-deletion/sign-out, last-admin protection, Sunday week starts, persisted Webhook/Discord settings, custom JSON, local Test Notification requests, the master switch, and a controlled clock crossing a release time into Available.
- Backend tests cover cross-profile history isolation, idempotent follow logging, filtered named job history, both disabled/local administration boundaries, deletion/session revocation, browser-cookie isolation, JSON escaping, Discord confirmation/payload/mention suppression, immediate error priority, subscription filtering, restart-safe release deduplication, local-time/DST scheduling, failure logging without automatic retries, and a disable/re-enable race against an already selected notification batch.
- Schema 3 to 4 upgrades preserve settings revision and favorites and leave a validated version-3 snapshot. Earlier schemas still upgrade sequentially. Backup round-trips now include activity, browser appearance, Discord connection settings, and the pending notification outbox, alongside existing profile/client/library state.
- Desktop and 390px mobile layouts were inspected: [show actions](screenshots/show-actions-mobile.png), [activity logs](screenshots/activity-logs-desktop.png), [notification services](screenshots/notification-services-desktop.png), [mobile notifications](screenshots/notification-services-mobile.png), and [login appearance](screenshots/login-appearance.png).

Notification protocol tests use local HTTP fixtures or in-memory requesters exclusively. No personal qBittorrent, Discord, or webhook connection was tested. Scheduled release notifications require a confirmed episode timestamp. Failed/ambiguous notification attempts are visible in Logs and are not retried automatically; queued releases keep the daily time selected when they were queued. Activity starts with this upgrade; earlier history is not fabricated.

The updated Docker image passed its Linux test/build stage and is healthy at `http://localhost:8080` on the existing `mediamanager_tally-config` volume. The schema is 4; both profiles and all three schedules remain. The validated pre-upgrade version-3 snapshot is present, and `/config/app.db` retains mode 0600. Readiness, 11 page routes, current assets, filtered named job history, and Logs pass. Existing non-admin requests to settings/jobs/statistics/logs return 403 while personal capabilities remain accessible. Current assets: `index-DF_1yUtW.js` and `index-Bb2kMAtH.css`.

## Modular Go refactor

Verified 2026-09-12. All 213 Go application/test files under `cmd`, `internal`, and `web` are below 150 lines; the largest production file is 143 lines, previously 466. Responsibilities are documented in [the code map](ARCHITECTURE.md).

- The complete Go suite, race detector, and `go vet ./...` pass. Existing migration, backup/restore, provider retry/circuit/cache, queue, job, account and torrent protocol coverage remains intact after moving the code.
- Added regressions verify server-local torrent selections, expiration, fresh adapter definitions that callers cannot mutate globally, CLI identity argument mapping, and invalid request construction avoiding retries/failure accounting.
- The frontend TypeScript/Vite build and all six Playwright workflows pass with no page errors. The frontend source and asset hashes are unchanged by this backend refactor.
- Native Windows and Docker builds pass; the Docker build includes Linux Go tests.
- The updated local container is healthy with its existing volume, two profiles and three schedules. Readiness, schema 3, 12 page routes, current assets, general API secret redaction and private database permissions pass.

No schema or archive-format change was made. No personal integration was tested during verification. The code now separates command dispatch, HTTP handlers/middleware, metadata persistence, backup stages, job implementations, and provider policies. Adapters accept the outbound request interface; jobs accept narrow metadata/provider/backup interfaces; runtime torrent selections belong to individual server instances.

## Discovery, library, settings, and jobs

The current update implements focused discovery search, suggestion/search grids and detail dialogs, durable background add/undo, retry notifications, direct episode-state toggles, favorites, stable calendar controls, categorized settings, SQLite-managed integrations and schedules, job visibility, debug previews, and saved statistics limits.

Completed checks:

- `go test ./...`, `go test -race ./...`, and `go vet ./...` pass.
- The TypeScript/Vite production build and native Windows executable build pass.
- All six Playwright workflows pass against isolated local fixture servers, with no browser page errors. Existing profile, authentication, torrent-client, and manual-search/send flows pass alongside the new workflows.
- Discovery regressions cover initial focus, interior padding versus backdrop clicks, suggestions replaced by search, detail previews, concurrent optimistic adds, undo while an earlier request is pending, and usable Retry notifications over an open dialog. A dialog-closing notification regression found in the full suite was fixed and rechecked.
- Episode controls toggle watched and downloaded both ways directly in the show list; bulk downloaded state can be cleared. Favorites persist after reload and appear as calendar stars. Month navigation coordinates remain stable.
- Settings regressions cover routed categories, visible saved connection keys and Hide/Show, the single Jackett connection, webhook enable/disable, schedule changes and disabling, debug previews, history filters, 12-hour schedule labels, and separate saved statistics limits.
- Queue tests cover no provider work during enqueue, undo invalidating an in-flight add, failed imports and retries, database transaction failure recovery, persistence after reopening, and shared metadata with profile-isolated follows.
- Job tests cover saved disabled schedules, stale-revision rejection, invalid cron rejection, pause after three consecutive failures, blocked automatic runs while paused, Resume, cancellation, and timeouts counting as failures. A local HTTP fixture verifies that saved webhook settings take effect immediately and disabling delivery retains in-app alerts.
- Suggestion tests verify six bounded cached schedule requests, rating order, deduplication, the 24-show cap, and plain-text summaries. Suggestions sample recent US broadcasts and worldwide streaming; they are not a global popularity chart.
- Migration tests cover version 2 to 3 with a verified pre-upgrade snapshot and preserved job/client data. Version 1 archives still restore and upgrade sequentially. Backup round-trips preserve favorites, queued actions, paused/disabled schedules, application settings and connection credentials, while excluding caches.
- Configuration tests verify that infrastructure environment values remain validated while UI-managed settings are seeded with SQLite defaults and remain independent of the process environment.
- General API responses remain redacted. Explicit operator connection settings are non-cacheable and may reveal credentials as requested. Other authenticated profiles cannot access those settings.

Desktop and 390-pixel mobile screenshots were inspected:

- [Discovery desktop](screenshots/discovery-desktop.png) and [mobile](screenshots/discovery-mobile.png)
- [Job previews](screenshots/jobs-previews-desktop.png)
- [Mobile scheduling](screenshots/scheduling-mobile.png)
- [Client settings desktop](screenshots/torrent-client-settings-desktop.png) and [mobile](screenshots/torrent-client-settings-mobile.png)
- [Calendar desktop](screenshots/calendar-desktop.png) and [mobile](screenshots/calendar-mobile.png)
- [Personal settings](screenshots/profile-settings-desktop.png)

The Docker image built successfully, including Linux Go tests, and the local container is healthy at `http://localhost:8080` using its existing persistent volume. Readiness, schema 3, both retained profiles, all three schedules, 12 page routes, current JavaScript/CSS assets, default statistics limits, and general API redaction pass. The pre-upgrade version 2 snapshot exists and `/config/app.db` retains mode 0600. These deployment checks made no manual external integration requests.

## Authentication and profile navigation

The profile control opens `/profile`; sign-out remains in personal settings, including mobile. Switching profiles requires signing out first. Routed personal settings, deployment categories, and local sign-in prompts support Back, Forward, and refresh.

Browser tests cover explicit sign-out with one profile, failed sign-out retaining the account for retry, profile selection after sign-out, cross-tab logout, cache clearing, and local password sign-in. Backend tests cover immutable profile IDs, deleted profile cookies, profile isolation, rejection of in-session account changes, session revocation, OIDC signature/token/state/nonce validation and replay rejection, bootstrap/recovery restrictions, and avatar validation.

## Torrent client and provider protocols

The qBittorrent adapter sends `Authorization: Bearer qbt_...` for tests, magnet submissions, and torrent-file uploads. It uses no username/password login or session cookies. Supported adapters and connection fields come from the Go registry. qBittorrent and Jackett have separate focused files under `internal/torrent`.

Local fixture tests cover rejected API keys, missing/malformed fields, version response validation, testing without saving or sending, saved-key retention and replacement, changed-address protection, disabling, revision conflicts, and manual sends. Protocol checks verify exact magnet/file payloads, one request per operation, no secret echo, and redirect/authentication failures without retries. Jackett tests cover connection validation, normalized indexer results, magnet handling, server-side torrent-file fetching, filtering, opaque profile-bound selections, URL secrecy, and replay prevention.

No personal qBittorrent address or API key was provided. No test or torrent submission was made to a personal client. Webhook delivery tests also use local fixtures only. Debug previews do not send messages, run jobs, or alter real job results.

## Earlier production smoke and dependency checks

The earlier disposable-container smoke test imported the real TVmaze show Severance: 19 episodes and three seasons, using four metadata requests. Calendar browsing and another profile's follow reused local data without additional metadata requests. Poster fetch, a verified backup, non-root execution, readiness, and restart persistence passed.


The 2026-09-11 dependency audit reported zero npm vulnerabilities and no reachable or imported-package Go vulnerabilities. GO-2026-5932 applied to the unused `golang.org/x/crypto/openpgp` package in a dependency module; Tally uses Argon2id, not OpenPGP. These are point-in-time results; dependencies were not changed in this update.

## Deployment-specific limits

Personal OIDC, Jackett, qBittorrent, and webhook services were not tested. Their protocols have local fixture coverage. HTTPS termination, remote access, bind-mount permissions, and off-host backup storage depend on the deployment.

The current schema is version 4. UI-managed credentials and settings are durable state included in protected backups; archives are not encrypted. Caches and environment secrets are excluded. Future schema changes require explicit migrations and upgrade tests. Additional downloader adapters remain deferred as specified.
