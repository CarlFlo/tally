# Go code map

Tally uses `cmd/server` for process setup and operator commands, and `internal` for application packages. Files target one responsibility and about 150-200 lines. Domain types and repositories live beside the services that use them, so a feature can be understood without traversing generic model/helper layers.

| Package | Responsibility | Useful entry points |
| --- | --- | --- |
| `cmd/server` | Configuration, deployment lock, startup/shutdown, CLI commands | `main.go`, `serve.go`, individual command files |
| `internal/api` | HTTP routes, authorization, request/response handling | `routes.go`, `authenticated_handler.go`, `security_middleware.go`, named `*_handler.go` files |
| `internal/auth` | Passwords, sessions, recovery, OIDC | `auth.go`, `login.go`, `session.go`, `oidc_callback.go`, `identity_repository.go` |
| `internal/metadata` | TVmaze protocol, shared metadata, queued follows | `tvmaze.go`, `service.go`, `repository.go`, `queue_worker.go` |
| `internal/jobs` | Bounded job execution, schedules, alert creation, notification-worker lifecycle | `jobs.go`, `dependencies.go`, `trigger.go`, individual `*_job.go` files |
| `internal/providers` | All outbound request coordination | `request_dispatch.go`, `request_execute.go`, `admission.go`, `rate_limit.go`, `circuit_breaker.go` |
| `internal/torrent` | Manual Jackett search, download-client adapters and stored connections | `jackett.go`, `jackett_results.go`, `jackett_fetch.go`, `client_registry.go`, `clients.go`, `client_prepare.go`, `qbittorrent.go` |
| `internal/profiles` | Transactional profile creation and display-name validation | `repository.go`, `name.go` |
| `internal/inbox` | Profile-scoped activity feeds and durable seen/clear/dismiss state | `store.go`, `state.go` |
| `internal/activity` | Durable, credential-free event records | `record.go` |
| `internal/library` | Transactional follow changes and their activity | `follow.go` |
| `internal/notifications` | Event subscriptions, durable outbox, release scheduling and Webhook/Discord delivery | `service.go`, `activity_queue.go`, `release_queue.go`, `delivery.go`, `message.go`, `send.go`, `schedule.go` |
| `internal/settings` | Durable editable settings and their validation | `settings.go`, `search.go`, `webhook.go`, `legacy.go` |
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
- Backup creation is a pipeline: snapshot, enumerate durable files, write archive, verify, publish, record, retain. Extraction separates path/size checks, manifest/checksum validation and database validation. The schema and archive format remain unchanged.

## Working on a feature

Frontend pages use one outer `.page` container for responsive padding, width and centering. Use `PageHeader` for a text title, eyebrow and description (System, Settings and Profile share it); use `.settings-tabs` for section navigation. Nested System content removes its own outer padding. The root reserves scrollbar space to prevent route/loading height changes from moving content. Do not introduce route-specific heading margins or raw HTML rendering: header props are strings escaped by React.

Treat provider metadata and form values as untrusted text. Keep URL and template validation on the server, outbound calls behind the coordinator, SQL values parameterized, and operator/profile authorization at existing API boundaries. Webhook templates operate on decoded JSON string values, never executable code or HTML. Add failure-path tests when changing these boundaries.

Start with the route or domain entry point above, then read its focused implementation and matching tests. Add new files when responsibilities diverge. Keep related transactional work together even when a file needs a little more space; the size target is a readability guide.

The refactor's largest production Go file is 143 lines, down from 466. Test fixtures and regression tests are also split by responsibility, with all application Go files below 150 lines at this checkpoint. Tests assert behavior and failure boundaries rather than enforcing file counts.

Run `go test ./...`, `go vet ./...` and the frontend build after relevant changes. Concurrency changes also need `go test -race ./...`; HTTP/application changes should run the existing Playwright suite against its isolated fixtures.

## Library and notification milestone

Schema 4 adds `activity_log`, `browser_preferences`, `notification_state`, and `notification_outbox`. Existing schema 1-3 upgrades remain sequential and take a validated snapshot before migration. Activity records intentionally retain actor/show labels without foreign keys, so deleting an account or removing a follow does not erase the deployment's history. Application APIs never put connection bodies or credentials in activity descriptions.

`admin_paths.go` guards deployment APIs for user0 even when authentication is disabled. `capabilities_handler.go` exposes only connection availability and safe provider names for personal search. Browser appearance has a public CSRF-protected update handler and a random HttpOnly browser cookie; profile appearance remains independent.

The notification worker runs as one cancellable goroutine. A tick reads at most 100 activity entries and 100 releases, then attempts at most 10 due deliveries within a 30-second context. System errors and job failures take priority. Each send reloads settings and claims its pending row against the settings revision; master-toggle changes cannot revive a previously selected, skipped notification. Delivery uses `providers.Requester`, with production always injecting the coordinator. Failure is recorded before the attempt to prevent automatic duplicate sends after ambiguous responses or crashes. Pending releases survive restarts and use unique episode event keys. Date-only metadata is not assigned an invented release time.

Frontend components keep show actions, login appearance, danger-zone deletion, notification fields/settings, logs, and release-time formatting separate from the existing pages.

## Header, onboarding, and backup milestone

Schema 5 adds `inbox_state` and `inbox_dismissals`, both owned by immutable profile IDs. The activity feed has independent visibility and read/dismissal state: the administrator sees deployment activity, other profiles see only their own. `/api/logs` applies the same scope to rows, counts, search, and action options. Operator System pages remain guarded; `/logs` is a personal activity page for members and redirects to System Logs for the administrator. The inbox excludes noisy per-episode progress and archive-download events.

Password setup inserts credentials exactly once inside a transaction; an existing hash cannot be overwritten, including by competing setup requests. Public registration uses the same transactional profile repository as administrator creation, preserves non-reused IDs, checks the profile limit, and requires a password in local mode. Existing signed-in sessions must sign out before registration/setup. Password strings preserve spaces, case, and symbols; display names reject controls and trim outer whitespace.

Backups use the fixed data-directory subfolder. Retention is loaded from SQLite for every retention pass. Archive downloads require the administrator, use opaque record IDs and verified records, and open files beneath an `os.Root` to reject traversal and escaping symlinks. The archive format is unchanged; new settings and inbox state are durable database content included in protected backups. Schema migration still requires a validated pre-upgrade snapshot.

Schedule updates combine optimistic revision checks, the job update, and activity logging in one transaction. A checkbox save sends the stored cron, preserving an unsaved editor draft. The frontend has focused profile/menu/inbox, login credentials/registration, schedule/retention/backup, System, calendar grid/horizon, and release grouping components. Known full-season labels use declared season metadata; an incomplete imported episode list does not establish a complete season.
