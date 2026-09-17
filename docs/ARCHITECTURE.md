# Architecture

Tally is one Go service serving an embedded React application and a permanent SQLite database. The architecture favors explicit domain ownership, bounded asynchronous work, backend-enforced security, and profile-specific state over generic frameworks or duplicated client state.

## Core ownership rules

- Shared TV metadata belongs to the deployment, never to a profile.
- Profiles own follows, favorites, preferences, localization, and episode state.
- Authentication maps to opaque immutable profile IDs; authorization is role-based.
- SQLite is the source of truth for durable application state.
- Infrastructure configuration comes from environment variables; operator-editable application settings live in SQLite.
- Outbound provider work goes through the provider coordinator.
- Torrent search and download submission remain user-initiated/manual.

## Backend map

| Area | Responsibility |
| --- | --- |
| `main.go` | Small process entrypoint and top-level error handling |
| `internal/commands` | Startup/shutdown, configuration wiring, deployment lock, operator commands |
| `internal/api` | HTTP routes, authorization, validation, response handling, live events, SPA serving |
| `internal/auth` | Per-profile password/no-auth state, sessions, and recovery |
| `internal/profiles` | Profile lifecycle, roles, names, avatars, sensitive role changes |
| `internal/metadata` | TV metadata, persistence, follows/import queue, TVmaze integration |
| `internal/providers` | Bounded outbound request admission, retries, rate limits, circuits, telemetry, cancellation |
| `internal/jobs` | Scheduled/manual jobs, failure state, job history and alert creation |
| `internal/settings` | Durable editable settings and validation |
| `internal/notifications` | Bell-event subscriptions, notification outbox, release delivery, Webhook/Discord |
| `internal/activity` | Durable credential-free activity records |
| `internal/inbox` | Profile-scoped inbox/read/dismissal state |
| `internal/library` | Transactional library/follow changes |
| `internal/torrent` | Jackett search, opaque selections, client adapters, qBittorrent protocol |
| `internal/database` | SQLite opening, migrations, validation, snapshots |
| `internal/backup` | Backup creation, verification, inventory, retention, restore |
| `internal/localization` | Bundled locale catalogs, validation, registry and filesystem watching |
| `internal/config` | Validated deployment/environment configuration |
| `web` | Embedded production frontend assets |

Keep new code in the owning domain. Prefer narrow interfaces at boundaries and concrete service types inside a domain. Avoid generic `utils`, global mutable runtime state, or a repository abstraction that hides important SQL transaction boundaries.

## Database and identity

SQLite runs with foreign keys and WAL enabled. Schema changes are explicit sequential migrations; schema 8 is current. Existing databases receive a validated pre-upgrade snapshot before migration. Migration work is transactional and validated before commit; downgrades from a newer unsupported schema are refused.

Profiles use generated opaque IDs. Each profile explicitly chooses Password or No authentication. Administrator privileges live in explicit role data rather than a special account ID. Database constraints protect invariants such as retaining an administrator while profiles remain, with API and UI checks providing additional defense in depth.

Sensitive administrator demotion/deletion re-authenticates the acting administrator when that administrator uses password authentication. Authorization is always enforced by the backend even when matching controls are hidden in the frontend.

## Frontend state and navigation

React Router owns page navigation. The authenticated application shell, query cache, header, and live-event connection remain mounted while route content changes normally. Do not force full application remounts to solve local state problems.

React Query owns server-read caching, deduplication, invalidation, and AbortSignals. Imperative actions that can be superseded use explicit latest-request ownership. Shared API transport is bounded so rapid navigation cannot create unbounded concurrent work or queues.

Components clean up dialogs, listeners, observers, timers, subscriptions, and asynchronous work on unmount. Durable writes may complete after navigation, but callbacks from an abandoned page must not mutate or navigate the newly active page.

Overlays that are meaningful navigation state should participate in browser history. For example, opening Calendar show details pushes overlay state so browser Back closes the overlay before leaving the Calendar page.

Live events invalidate only relevant resources. They should refresh visible server state without overwriting dirty form drafts or causing input-driven requests to restart unnecessarily.

## Draft state versus applied state

Forms should distinguish draft values from authoritative saved state. Profile language selection is a draft until Save profile succeeds; only then does the saved locale become authoritative. Similar forms should not visually imply persistence before the backend accepts the change.

Immediate toggles are appropriate only when the product intentionally defines the toggle itself as the save action. Keep those semantics explicit instead of mixing auto-save and staged-save behavior in one control group.

When authoritative state changes invalidate derived UI, render feedback from the new state rather than capturing stale derived values. Localization-sensitive toast messages, for example, should resolve their translation after the new locale is active.

## Localization

Locale catalogs are server-owned and loaded from the persistent data directory. English is the canonical key contract and final fallback; Ukrainian is bundled as an additional locale. Operators may add validated locale files without rebuilding Tally.

Profile locale is authoritative. The browser does not auto-select locale from browser preferences. Locale changes update document language/direction metadata and all frontend translation consumers.

User-facing frontend text belongs in the localization catalogs. Provider metadata, raw server logs, protocol identifiers, and unstable provider errors remain untranslated unless a stable application error code exists.

See `LOCALIZATION.md` for catalog format and authoring rules.

## Provider and asynchronous work

All production external-provider requests use the coordinator. It centralizes admission limits, response-size bounds, retries, Retry-After handling, rate limiting, circuit state, cancellation, caching policy, and telemetry.

Shared requests continue only while at least one caller still needs the result. When the last waiter leaves, downstream HTTP work and admission waits are cancelled. Queued work must be removable on cancellation, and every acquired concurrency slot must be released on all success/error paths.

Background jobs are bounded and cancellable. Schedules are persisted, wake the scheduler when relevant state changes, and use deployment timezone for execution. Profile timezone affects presentation, not server execution semantics.

Prefer event-driven refresh and filesystem watchers to frequent polling. Watchers are intentionally not masked by a polling fallback on unusual network filesystems unless a future requirement explicitly adds one.

## Torrent capabilities

Torrent search and torrent downloading are independent capabilities.

- Search availability is controlled by the saved search setting and configured Jackett connection.
- Download availability is controlled by its own saved torrent-download setting and configured client.
- When a capability is disabled, the frontend hides navigation/actions that cannot work.
- The backend independently rejects disabled operations; hiding UI is not enforcement.
- Capability toggles sit outside the provider/client configuration section they govern so disabling a feature does not make its own switch unreachable.

Jackett search returns normalized results and opaque profile-bound selections. Tally does not expose provider download URLs to the browser. qBittorrent submissions and status queries use the adapter protocol; protocol success is determined by the actual client contract rather than assumptions such as requiring a JSON response.

The Downloads page reflects client-reported torrent state. Tally does not automatically choose torrents, scan media, rename files, or mark episodes downloaded merely because a torrent was submitted.

## Activity, logs, and notifications

Logs are the complete operational history appropriate to the current profile/administrator scope. Bell notifications are intentionally more selective and configurable so routine user-initiated saves do not become noise.

Activity records must not contain credentials. Notification delivery uses durable outbox/state where required and reloads authoritative settings before sending. Ambiguous or failed external deliveries must be observable without silently duplicating sends.

## Backups and restore

Backup creation follows a pipeline: snapshot durable state, collect durable files, write the archive, verify it, publish it, and apply retention. The backup directory is the archive inventory source of truth; valid manually copied archives can be discovered without separate registration metadata.

Caches and environment-provided secrets are excluded. UI-managed credentials are durable application state and belong in protected backups.

Restore extracts into staging, validates archive paths/sizes/checksums, validates and upgrades the staged database when compatible, prepares durable files, and only then applies database state transactionally. A failed restore must leave the running state usable. Relational restore tests must account for foreign-key cascades and insertion/deletion ordering.

## Secrets and settings

Infrastructure configuration remains environment-owned. User/operator-managed integrations and schedules are stored in SQLite and applied without process restart where supported.

Secrets are revealed only in explicitly authorized settings views. General APIs, logs, errors, notifications, telemetry, and activity records must not echo them. When editing a connection, an intentionally blank secret field means retain the saved secret unless the operation explicitly requests removal/replacement.

## Working on a change

Start from the domain entry point and matching tests. Preserve API contracts, transaction boundaries, authorization, provider-coordinator usage, localization, and existing UX unless the task explicitly changes them.

Use one outer `.page` container for normal pages and the shared page-header/navigation patterns rather than route-specific layout workarounds. Keep accessibility labels stable enough for both users and automated browser tests.

When a bug reveals a reusable engineering principle, capture the generalized lesson in `LESSONS.md`. Keep incident-specific history in Git rather than expanding architecture or roadmap files with dated narratives.

Validation expectations are defined in `VALIDATION.md`.
