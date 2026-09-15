# Go code map

## Frontend navigation lifecycle

The authenticated route subtree is keyed by pathname. A new page/tab owns fresh local form, loading and dialog state, while the header, query cache and single live-event subscription stay mounted. Dialogs close during layout cleanup before DOM removal. Navigation has no cooldown, pointer lock or blocking overlay.

React Query owns read deduplication and AbortSignals; imperative connection tests use `useLatestRequest` and torrent search owns an abortable latest request. `requestPool.ts` bounds API transport to six active requests and 64 queued requests, removes aborted queue entries and releases slots on failures. API calls have a 90-second deadline, including queue and response-body time. Durable manual writes may complete after a page leaves, but stale show deletion callbacks cannot navigate the new page. Live events coalesce refreshes without replacing in-flight fetches and exclude input-driven discovery/cron previews.

Provider shared requests retain work while at least one caller needs it. The last caller leaving cancels downstream HTTP/slot waits; at most 64 distinct shared operations may remain outstanding, including operations still unwinding cancellation. Browser stress tests use freshly built embedded assets, retain the same document and verify input, listener/interval/stream counts and transport concurrency after repeated switching.

Tally uses the repository-root `main.go` as the small process entrypoint. `internal/commands` owns process setup and operator commands, while the other `internal` packages own application domains. Files target one responsibility and about 150-200 lines. Domain types and repositories live beside the services that use them, so a feature can be understood without traversing generic model/helper layers.

| Package | Responsibility | Useful entry points |
| --- | --- | --- |
| `main.go` | Process entrypoint and top-level error logging | `main.go` |
| `internal/commands` | Configuration, deployment lock, startup/shutdown, CLI commands | `main.go`, `serve.go`, individual command files |
| `internal/api` | HTTP routes, authorization, request/response handling | `routes.go`, `authenticated_handler.go`, `security_middleware.go`, named `*_handler.go` files |
| `internal/auth` | Passwords, sessions, recovery, OIDC | `auth.go`, `login.go`, `session.go`, `oidc_callback.go`, `identity_repository.go` |
| `internal/metadata` | TVmaze protocol, shared metadata, queued follows | `tvmaze.go`, `service.go`, `repository.go`, `queue_worker.go` |
| `internal/jobs` | Bounded job execution, schedules, alert creation, notification-worker lifecycle | `jobs.go`, `dependencies.go`, `trigger.go`, individual `*_job.go` files |
| `internal/providers` | All outbound request coordination | `request_dispatch.go`, `request_execute.go`, `admission.go`, `rate_limit.go`, `circuit_breaker.go` |
| `internal/torrent` | Manual Jackett search, download-client adapters and stored connections | `jackett.go`, `jackett_results.go`, `jackett_fetch.go`, `client_registry.go`, `clients.go`, `client_prepare.go`, `qbittorrent.go` |
| `internal/profiles` | Transactional profile lifecycle, role changes, avatar colors and display-name validation | `repository.go`, `roles.go`, `avatar.go`, `name.go` |
| `internal/inbox` | Profile-scoped activity feeds and durable seen/clear/dismiss state | `store.go`, `state.go` |
| `internal/activity` | Durable, credential-free event records | `record.go` |
| `internal/library` | Transactional follow changes and their activity | `follow.go` |
| `internal/notifications` | Event subscriptions, durable outbox, release scheduling and Webhook/Discord delivery | `service.go`, `activity_queue.go`, `release_queue.go`, `delivery.go`, `message.go`, `send.go`, `schedule.go` |
| `internal/settings` | Durable editable settings and their validation | `settings.go`, `search.go`, `webhook.go`, `defaults.go` |
| `internal/database` | SQLite opening, migration, validation and snapshots | `database.go`, `migrations.go`, `schema_validation.go`, `snapshot.go` |
| `internal/backup` | Snapshot archives, validation, restore and retention | `create.go`, `archive_writer.go`, `extract.go`, `snapshot_validation.go`, `restore.go` |
| `internal/config` | Validated infrastructure environment values | `config.go`, `load.go`, `url.go` |
| `web` | Embedded production frontend | `embed.go` |

## Boundaries

- HTTP handler filenames identify individual operations. Routing, JSON handling, authorization, security middleware, health checks and SPA assets are separate. Existing JSON response shapes and routes remain stable.
- Metadata models (`show.go`, `episode.go`, `season.go`) describe provider data. `sync.go` fetches that data; `Repository.Save` persists the complete response transactionally. Shared metadata never becomes profile-owned.
- Jobs accept small `MetadataSource`, `ProviderControl` and `BackupCreator` interfaces. Constructors return concrete services. Each job implementation returns a named `runResult` with an explicit error.
- TVmaze, Jackett, torrent-client adapters and OIDC transport accept `providers.Requester`. Production wiring always supplies the coordinator. Request admission, rate limiting, one HTTP attempt, cache handling, retry policy, circuit state and telemetry have distinct files.
- SQLite remains concrete infrastructure. Repositories and transaction boundaries remain in their owning domains; no general-purpose repository framework is introduced.
- Torrent selections belong to a `Server` instance. Jackett only searches configured indexers; Tally owns filtering and opaque user selections, and the configured torrent client performs downloads. Adapter definitions are constructed afresh, including their field slices. Package-level values are limited to embedded resources, sentinel errors and precompiled read-only validation patterns; mutable runtime state belongs to service instances.
- Backup creation is a pipeline: snapshot, enumerate durable files, write archive, verify, publish, record, retain. Extraction separates path/size checks, manifest/checksum validation and database validation. Manifests retain format 1 and now optionally record the source Tally version; schema compatibility remains authoritative for restore. In-app restore validates and migrates a staged database first, then replaces durable SQLite table contents in one live transaction. Current backup inventory is intentionally preserved because it describes archives on the active filesystem.

## Working on a feature

Frontend pages use one outer `.page` container for responsive padding, width and centering. Use `PageHeader` for a text title, eyebrow and description (System, Settings and Profile share it); use `.settings-tabs` for section navigation. Nested System content removes its own outer padding. The root reserves scrollbar space to prevent route/loading height changes from moving content. Do not introduce route-specific heading margins or raw HTML rendering: header props are strings escaped by React.

Treat provider metadata and form values as untrusted text. Keep URL and template validation on the server, outbound calls behind the coordinator, SQL values parameterized, and operator/profile authorization at existing API boundaries. Webhook templates operate on decoded JSON string values, never executable code or HTML. Add failure-path tests when changing these boundaries.

Start with the route or domain entry point above, then read its focused implementation and matching tests. Add new files when responsibilities diverge. Keep related transactional work together even when a file needs a little more space; the size target is a readability guide.

The refactor's largest production Go file is 143 lines, down from 466. Test fixtures and regression tests are also split by responsibility, with all application Go files below 150 lines at this checkpoint. Tests assert behavior and failure boundaries rather than enforcing file counts.

Run `go test ./...`, `go vet ./...` and the frontend build after relevant changes. Concurrency changes also need `go test -race ./...`; HTTP/application changes should run the existing Playwright suite against its isolated fixtures.

## Library and notification milestone

Schema 4 adds `activity_log`, `browser_preferences`, `notification_state`, and `notification_outbox`. Existing schema 1-3 upgrades remain sequential and take a validated snapshot before migration. Activity records intentionally retain actor/show labels without foreign keys, so deleting an account or removing a follow does not erase the deployment's history. Application APIs never put connection bodies or credentials in activity descriptions.

`admin_paths.go` identifies deployment-only API paths and authorization checks the authenticated profile's administrator role even when authentication is disabled. `capabilities_handler.go` exposes only connection availability and safe provider names for personal search. Browser appearance has a public CSRF-protected update handler and a random HttpOnly browser cookie; profile appearance remains independent.

The notification worker runs as one cancellable goroutine. A tick reads at most 100 activity entries and 100 releases, then attempts at most 10 due deliveries within a 30-second context. System errors and job failures take priority. Each send reloads settings and claims its pending row against the settings revision; master-toggle changes cannot revive a previously selected, skipped notification. Delivery uses `providers.Requester`, with production always injecting the coordinator. Failure is recorded before the attempt to prevent automatic duplicate sends after ambiguous responses or crashes. Pending releases survive restarts and use unique episode event keys. Date-only metadata is not assigned an invented release time.

Frontend components keep show actions, login appearance, danger-zone deletion, notification fields/settings, logs, and release-time formatting separate from the existing pages.

## Header, onboarding, and backup milestone

Schema 5 adds `inbox_state` and `inbox_dismissals`, both owned by immutable profile IDs. The activity feed has independent visibility and read/dismissal state: the administrator sees deployment activity, other profiles see only their own. `/api/logs` applies the same scope to rows, counts, search, and action options. Operator System pages remain guarded; `/logs` is a personal activity page for members and redirects to System Logs for the administrator. The inbox excludes noisy per-episode progress and archive-download events.

Password setup inserts credentials exactly once inside a transaction; an existing hash cannot be overwritten, including by competing setup requests. Public registration uses the same transactional profile repository as administrator creation, preserves non-reused IDs, checks the profile limit, and requires a password in local mode. Existing signed-in sessions must sign out before registration/setup. Password strings preserve spaces, case, and symbols; display names reject controls and trim outer whitespace.

Backups use the fixed data-directory subfolder. Retention is loaded from SQLite for every retention pass. Backup download, restore, and deletion require the administrator and use opaque record IDs. Downloads remain confined beneath an `os.Root`; management operations reject unsafe names, symlinks, and non-regular archives. Restore fully extracts and validates the archive, upgrades compatible older schemas in staging, normalizes runtime schedules, prepares avatars without overwriting conflicting files, and only then copies durable database state inside one transaction. Failed application rolls back the database and any newly prepared avatars, while successful restore takes effect in the existing process. Backup inventory is not rolled back to the snapshot because it represents the archives physically present on the current host.

Schedule updates combine optimistic revision checks, the job update, and activity logging in one transaction. Cron expressions are interpreted in the deployment `TZ` timezone (UTC when unset), while profile timezone is display-only for timestamps such as previews, calendar releases, job times, and last-sync labels. Notification delivery time is always interpreted in deployment `TZ`; the old persisted timezone field is retained only for data compatibility and is normalized to the server timezone when settings are loaded or saved. Profile timezone is presentation-only and the notification UI converts the next server-time delivery for the current user. A checkbox save sends the stored cron, preserving an unsaved editor draft. The frontend has focused profile/menu/inbox, login credentials/registration, schedule/retention/backup, System, calendar grid/horizon, and release grouping components. Known full-season labels use declared season metadata; an incomplete imported episode list does not establish a complete season.


## Profile roles and identity

Schema 6 removes the sequential `userN` identity model. Profiles use opaque generated IDs and administrator access lives in `profile_roles`; no profile is permanent. Migration 6 rewrites profile-owned foreign keys, preserves legacy IDs only in `profile_id_aliases` for migration/session compatibility, and removes the obsolete profile counter.

The first profile created in an empty deployment is promoted automatically by the database trigger. While any profiles remain, database triggers prevent the last administrator from being demoted or deleted, so API/UI checks are defense in depth rather than the security boundary. Deleting the sole remaining profile is valid; the next profile created becomes administrator.

Administrator promotion is role-based. Under local authentication, demoting or deleting an administrator requires re-authentication with the acting administrator's own password. Profile creation/deletion and administrator grant/revoke activity is always logged and can be delivered through the `profile_access_changed` notification subscription.
