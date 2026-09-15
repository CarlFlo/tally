# Implementation roadmap

Current feature: header navigation, grouped calendar, account onboarding, and backup management.
Current milestone: COMPLETE (2026-09-12).
Current work: the `/shows` Favorites heading uses a filled star. Scheduling keeps only the three job cards, and enabled notification alerts use stronger theme-aware text. Primary purple button hover states preserve readable contrast in dark, light, and system themes. Scheduling cards explain the purpose of Metadata, Maintenance, and Backup beneath each heading. The calendar filter uses an app-themed menu, notification services validate fields inline, respect each profile's clock format, require saved valid configuration before alerts can be enabled, and preview webhook payloads. Shared cron descriptions combine numeric day-of-month, weekday, and month constraints; the header remains visible while scrolling; torrent connection secrets default to concealed.
Follow-up UI polish: header notification order, discovery Add placement, version label, episode status alignment, compact logs, and notification copy are implemented and verified by the frontend build and browser suite. Bell notifications now use per-profile event categories with failure/release defaults while Logs retain the complete activity history; the Show card uses compact transparent favorite and kebab actions whose open menu layers above the sidebar.
No-op saves for deployment settings and schedules preserve their revision and do not create activity log entries.
Bell notification settings are grouped into attention, media update, and successful activity categories.
Visible logs refresh in place every three seconds. Settings poll for saved changes every five seconds and only replace untouched form values, preserving active edits, dialogs, and overlays.
Discovery Add actions sit on the poster footer and appear on hover or keyboard focus; show titles and metadata remain below the poster.
Layout/security follow-up complete: System shares the Profile/Settings header and page container; stable scrollbar space prevents horizontal shifts and the header stays visible while scrolling. Server webhook validation checks decoded placeholders and rejects templated keys. Notification fields retain stable accessible names during validation. Go tests, vet, frontend build and browser layout/security regressions pass.
Blockers: none.

## Current milestone acceptance criteria

- [x] Header profile menu and notification inbox; tabbed System area; hide internal profile IDs.
- [x] Day-grouped horizon, combined same-show releases/full-season labels, unrestricted Show more.
- [x] Subtle favorite/star animation and centered detail star.
- [x] Compact schedules, immediate toggles, logged changes, hourly metadata default.
- [x] Animated three-mode login appearance, Add profile, first-use password setup, simplified password policy.
- [x] UI backup controls, fixed internal backup path, archive downloads, in-app restore/delete, version visibility, and backup failures routed to Notifications/Logs.
- [x] Schema/backup/access/onboarding tests, full verification, documentation, and local deployment.

## Refactor acceptance criteria

- [x] Production Go files target about 150 lines, with coherent responsibilities and domain-specific names.
- [x] HTTP routing/middleware/handlers, service orchestration, persistence, and provider policies are separated.
- [x] Dependencies use narrow interfaces where useful; constructors return concrete types; errors remain explicit.
- [x] Existing behavior, data format, migrations, authorization, and outbound coordinator boundaries remain intact.
- [x] Backend tests, race detector, vet, frontend build, browser regressions, and native/container builds pass.
- [x] Document package responsibilities and remaining justified size exceptions.

## Current request checklist

- [x] Add Show focus/backdrop fixes; suggestion/search card grid and details overlay.
- [x] Optimistic queued add/remove; persistent server queue; retry notifications at the top.
- [x] Direct watched/downloaded toggles and downloaded reset.
- [x] Favorites section and compact calendar stars; stable month controls; expanded horizon.
- [x] Visible connection secrets with hide/show controls and password-manager hints.
- [x] Routed settings categories; database-managed webhook, Torznab, and schedules.
- [x] Job toggles, seven-day metrics, failure pause/resume, filters, friendly cron labels.
- [x] Persistent debug mode with safe UI failure previews.
- [x] Statistics limits (20/50/100, default 20), persisted per profile.
- [x] Live statistics refresh; unified five-field scheduler parser, backend cron description/next-run previews, and combined Scheduling & backups settings.
- [x] Event-driven active-page refresh for jobs, statistics, logs, inbox, library, and settings; stable Scheduling & backups layout; UTC cron review with three-run previews and `*/60` compatibility.
- [x] Scheduling UI polish: job-specific common presets, local-time preview rendering, compact aligned fields, slim grouped sections, concealed secret fields by default, and edge-aligned input focus states.
- [x] Migration/backup/queue/settings/protocol tests, browser checks, build, local deployment.

## Architecture

One Go process serves an embedded React application and a permanent SQLite database.
Shared shows, seasons, episodes, and external IDs are deployment-wide. Profile follows and episode states are separate.
Authentication maps onto immutable profiles. Infrastructure settings remain ENV-driven; torrent clients, Torznab providers, webhooks, and job schedules are UI-managed in SQLite.
Remote calls use one coordinator, including integrations. No automatic torrent selection or downloading.

## Roadmap and acceptance criteria

- DONE: Header/calendar/onboarding/backups milestone - profile dropdown and persistent personal notification inbox; admin System tabs and scoped member logs; hidden profile IDs; grouped calendar releases and days, full-season labels, unlimited Show more, star feedback; compact immediate schedule controls and hourly metadata default; animated login appearance, Add profile and first-use passwords; UI backup settings and downloads. All Go/race/vet, 12 browser workflows, frontend/native/Linux builds pass. Local schema 5 is healthy with the existing two profiles, three schedules, private database permissions, and validated pre-upgrade snapshot. See `docs/VALIDATION.md`.

- DONE: Library actions and administration milestone - show/card menus, watch-history clear, Shift-remove, coloured episode status and completion badges; searchable action logs; show names in jobs; Webhook/Discord services with templates, test delivery, master toggle, subscriptions, and persistent daily release delivery; live countdowns; user0-only administration; self-deletion Danger zone; server-persisted browser login appearance. All backend/race/vet, nine browser workflows, frontend/native/Docker builds, and local deployment checks pass. Schema 4 upgraded with a validated snapshot and retained both profiles and all three schedules. See `docs/VALIDATION.md`.

- DONE: Modular Go refactor - all 213 application Go files are below 150 lines (largest production file: 143, previously 466). Split HTTP handlers, middleware, commands, authentication, migrations, backups, jobs, metadata persistence and provider policies; introduce narrow outbound/job interfaces, structured job results, instance-owned torrent selections and fresh adapter definitions. Existing behavior is covered by full backend/race/vet checks and six browser workflows. Native/Docker builds and local readiness, schema, profiles, routes and redaction are verified. See `docs/ARCHITECTURE.md`.

- DONE: Discovery/library/settings milestone - focused card discovery, suggestions and details, durable optimistic add/undo/retry, direct state toggles, favorites and calendar refinements, categorized SQLite-managed connections/schedules, job summaries/filtering/pause/resume, debug previews, and persisted statistics limits.
- DONE: Schema 3 upgrade - verified pre-upgrade snapshot, restored settings/favorites/queue/schedules in backup tests, and a healthy local deployment with both profiles retained.

- DONE: qBittorrent API-key authentication - Bearer headers for tests and magnet/file sends, required key with operator Hide/Show controls, key retention/replacement, older configuration handling, and local protocol/browser regressions.
- DONE: UI-managed torrent client — compiled adapter registry drives the dropdown and fields; qBittorrent and Torznab live in separate files. Operator-only connection management, unsaved connection tests, secret redaction/retention behavior, target-change protection, revision conflict detection, immediate application of saves, and disable are implemented.
- DONE: Schema 2 migration — private database permissions, automatic pre-upgrade snapshot, transactional schema validation, version 1 backup restore/upgrade, and client credential persistence/backup/restore verified. Superseded by schema 3 in the current milestone.
- DONE: Sign-out placement — no sidebar or main-page sign-out; it remains inside `/profile` and its security section, plus the required-password-change screen.
- DONE: Profile navigation fix — personal settings at `/profile`, routed settings sections and sign-in prompts, visible desktop/mobile sign-out, remembered explicit sign-out, server checks against account switching, revoked sessions, cross-tab cache clearing, and Back/Forward/refresh regression tests. Updated container is healthy at localhost:8080 with its existing data volume.
- DONE: Go service, validated configuration, SQLite WAL/foreign keys, transactional migrations/schema validation, health, graceful shutdown.
- DONE: Immutable non-reused profile IDs, protected user0, preferences, re-encoded avatars, remembered/deleted-cookie handling.
- DONE: Local calendar, library, detail, optimistic episode drawer state, bulk state, profile picker, responsive themes.
- DONE: TVmaze ranked search, shared metadata/seasons/specials, adaptive refresh, allowlisted image cache.
- DONE: Central cache/coalescing/rate limits/retries/Retry-After/circuits/probes, bounded responses, request records/aggregates, shutdown drain.
- DONE: Local Argon2id authentication, bootstrap/recovery expiry/restart, forced change, revocable sessions, CSRF/isolation.
- DONE: Bounded deduplicated jobs, cancellation, interrupted-run recovery, editable persistent schedules, statistics, alerts and webhook.
- DONE: Torznab search, filters/history, opaque selections, qBittorrent magnet/file submission and replay protection. Sends do not change episode state.
- DONE: Snapshot backups, checksums/schema/avatars validation, retention, staged operator restore, pre-migration snapshot and downgrade refusal.
- DONE: OIDC discovery/state/nonce/PKCE and issuer+subject mapping. Protocol tests cover signature/token validation, replay and invalid state/nonce.
- DONE: Operator commands for backup/verify/restore/delete, password reset and explicit identity mapping.
- DONE: Expanded browser verification for multiple Torznab providers, manual sends, profile isolation, and responsive/date preference changes. Backend tests cover partial failure and URL secrecy.
- DONE: Final production assets, native binary, Linux container build, Compose validation, live TVmaze import/poster, verified backup, and container restart persistence.
- DONE: Boundary/failure tests, initial frontend build, initial Playwright workflow, dependency updates, operator docs and environment reference.

## Verification and explicit limits

- Backend tests, race detector, and go vet pass (2026-09-12). Tests cover profiles/auth/CSRF, OIDC protocol, avatars, metadata sharing, retries/circuits, cancellation, backups/migrations, and submission idempotency. The Docker build also runs the Go tests on Linux.
- The full 12-test Playwright suite passes, including these earlier workflows: discovery focus/grid/details, queued add/undo/retry, direct episode toggles, favorites, calendar layout, saved integrations/schedules/debug/filters/statistics limits, plus account/settings history, single-profile sign-out, local password sign-in, cross-tab logout, show import, episode persistence, season state, multiple Torznab providers, filtered results, exact manual submission, profile selection after sign-out/remembering/isolation, jobs/statistics, mobile and light theme, and the full torrent-client setup/test/save/reload/send flow. The full suite also covers notifications while opening and closing dialogs.
- Previous dependency audit (2026-09-11): npm audit: zero vulnerabilities. govulncheck: zero reachable or imported-package vulnerabilities; GO-2026-5932 applies to the unused `golang.org/x/crypto/openpgp` package in a dependency module. Recheck when updating dependencies.
- Earlier live disposable-container smoke test imported Severance (19 episodes, three seasons) with four TVmaze requests. Calendar browsing and a second profile's follow reused local data without extra metadata requests. Poster fetch, verified backup, non-root execution, health check, and restart persistence passed.
- English is the implemented language. Navigation strings are centralized; date/time display uses Intl and profile preferences.
- Torznab searches cap at 100 results per endpoint and eight endpoints. HTML scraping is excluded.
- Jobs bound provider retries; failed entities are deferred instead of replaying a whole batch.
- Current native and Docker builds pass; Linux tests run during the image build. Local readiness, schema 3, private database permissions, pre-upgrade snapshot, 12 routes, current assets, retained profiles, and general API secret redaction are verified.
- No personal qBittorrent, Torznab, or webhook connection was tested or modified during verification.
- OIDC has fixture coverage; operators must verify their real identity-provider configuration.
- Backup download, restore, and deletion require the administrator. Web restore validates and migrates in staging, applies durable SQLite state transactionally without a process restart, and preserves the current archive inventory; archives include durable connection credentials.
- DEFERRED as specified: additional downloader adapters. No movies, playback, automatic downloads, scanning/renaming, or torrent lifecycle management.

Features are DONE only after their meaningful failure paths and acceptance checks are implemented.
# Completed: Added a two-week automatic backup preset and replaced the redundant Sunday maintenance preset with a monthly maintenance preset.
# Completed: Removed manual refresh controls from live Jobs and Statistics pages and trimmed redundant page subtitles.
# Completed: Marked cron schedule fields as non-credential inputs so password managers do not autofill them.
# Completed: Replaced multi-provider Torznab search with one UI-managed Jackett connection, validation, normalized results, and opaque magnet/torrent-file handoff to the configured torrent client.
# Completed: Moved calendar overview metrics into a compact rail summary above On the horizon and removed the calendar subtitle to prioritize the calendar grid.
# Completed: Reordered calendar navigation and view filtering, moved the status legend below the calendar, and shortened the weekday header row.
# Earlier mitigation: Coalesced live refreshes and added request cancellation; the reported navigation freeze persisted. The original navigation test reloaded before checking interactivity and did not establish a root cause.
# Completed: Bounded shared date/time formatter reuse (20,200 allocations reduced to 3 in populated System navigation), isolated countdown ticks from the calendar grid, explicitly closed captured dialog elements on unmount, and removed reused aborted prefetch controllers. Regression coverage now keeps one document alive through pointer navigation, viewport changes, delayed requests and modal teardown. The user's exact browser freeze has not yet been reproduced in the isolated runner.
# Verified 2026-09-14: All 20 browser regressions, go test ./..., go vet ./..., frontend build and Docker build passed. Updated local container is healthy.
# Completed: Removed the Calendar page local-library footer and timezone labels.
# Superseded: Removed navigation cooldown state, timers, CSS and keyboard interception. Route-effect cleanup could cancel the unlock timer and leave navigation permanently blocked.
# Completed: Explicit page lifecycle boundaries, synchronous dialog teardown, cancellation of abandoned connection tests, prevention of stale show-removal navigation, six-request frontend transport with bounded queue/deadline, selective live refresh, and waiter-owned bounded provider work. Shared provider requests cancel when the last caller leaves.
# Verification: Added 200-route same-document stress coverage for listener growth, live streams, intervals, request concurrency and post-stress input; queue cancellation/recovery tests and backend shared-caller cancellation/admission tests. Browser test command now builds embedded assets first.
# Verified lifecycle changes: 23-test browser suite plus the added abandoned-connection-test regression pass (24 distinct tests); Go tests, vet, provider/live race tests and Docker build pass. Stress runs show no freeze or resource accumulation. The earlier pre-cooldown browser-specific freeze remains unreproduced; the permanent cooldown lockout path is removed.
# Completed: Torrent-client endpoint changes no longer require API-key re-entry. Saved qBittorrent keys are retained when the Web UI URL changes for connection tests and saves; Jackett already follows the same behavior. Added endpoint-change regression coverage.
# Completed: Removed the qBittorrent Web UI URL helper text and saved API-key explanation from the torrent settings form while retaining validation, API-key guidance, and conceal/reveal controls.
# Completed: Matched Jackett search helper text spacing to the torrent client settings fields.
# Completed: Moved the executable entrypoint to the repository-root main.go and grouped command/deployment operations under internal/commands; build and run documentation now use the root package.
# Completed: Removed legacy environment-variable migration and obsolete environment references; UI-managed settings now seed from SQLite defaults only.

# Completed: Backup archives now expose Manual/Automatic type and source Tally/schema versions, support confirmed in-app restore and deletion, preserve current state on failed restore, migrate compatible older backups in staging, and apply restored SQLite state without restarting Tally.

# Fixed 2026-09-15: Live restore now deletes all restorable tables before inserting snapshot rows so SQLite cascades cannot erase restored follows/preferences. Archive inventory no longer renders failed job runs; backup/restore failures are surfaced through Notifications and Logs.
