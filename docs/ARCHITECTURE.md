# Architecture

Tally is one Go service serving an embedded React application and a permanent SQLite database. The architecture favors explicit domain ownership, bounded asynchronous work, backend-enforced security, and profile-specific state over generic frameworks or duplicated client state.

## Core ownership rules

- Shared TV metadata belongs to the deployment, never to a profile.
- Profiles own follows, favorites, preferences, localization, episode state, and notification preferences.
- Torrent search, downloader configuration, release assessment, automation policy, run history, and bad-infohash feedback are deployment-global because one shared downloader is authoritative.
- Authentication maps to opaque immutable profile IDs; authorization is role-based.
- SQLite is the source of truth for durable application state.
- Infrastructure configuration comes from environment variables; operator-editable application settings live in SQLite.
- Outbound provider work goes through the provider coordinator.
- Manual torrent search remains user-directed. Automatic torrent submission is allowed only through the guarded automation decision pipeline described below.

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
| `internal/torrent` | Jackett discovery, normalized release metadata, confidence evaluation, local `.torrent` inspection, automation decisions/history/feedback, client adapters, qBittorrent protocol |
| `internal/flows` | Saved automation chain validation, show assignment, dry-run execution, and bounded run snapshots |
| `internal/database` | SQLite opening, migrations, validation, snapshots |
| `internal/backup` | Backup creation, verification, inventory, retention, restore |
| `internal/localization` | Bundled locale catalogs, validation, registry and filesystem watching |
| `internal/config` | Validated deployment/environment configuration |
| `web` | Embedded production frontend assets |

Keep new code in the owning domain. Prefer narrow interfaces at boundaries and concrete service types inside a domain. Avoid generic `utils`, global mutable runtime state, or a repository abstraction that hides important SQL transaction boundaries.

## Database and identity

SQLite runs with foreign keys and WAL enabled. Schema changes are explicit sequential migrations; schema 16 is current. Existing databases receive a validated pre-upgrade snapshot before migration. Migration work is transactional and validated before commit; downgrades from a newer unsupported schema are refused.

Profiles use generated opaque IDs. Each profile explicitly chooses Password or No authentication. Administrator privileges live in explicit role data rather than a special account ID. Database constraints protect invariants such as retaining an administrator while profiles remain, with API and UI checks providing additional defense in depth.

Sensitive administrator demotion/deletion re-authenticates the acting administrator when that administrator uses password authentication. Authorization is always enforced by the backend even when matching controls are hidden in the frontend.

## Frontend state and navigation

React Router owns page navigation. The authenticated application shell, query cache, header, and live-event connection remain mounted while route content changes normally. Do not force full application remounts to solve local state problems.

Navigation separates personal account state from deployment administration: `/account/*` contains profile preferences, bell notifications, security, and account removal; administrator-only `/admin/operations/*`, `/admin/configuration/*`, `/admin/access/*`, and `/admin/advanced/*` contain deployment-wide work. Legacy `/profile`, `/settings`, and `/system` deep links redirect to their canonical destinations.

The server places the active profile's validated theme in the HTML before first paint; login and profile-picker pages use the browser appearance preference. The pre-paint bootstrap must preserve this server choice because local storage can be stale across profile changes. React continues to apply the bootstrap preference after loading.

React Query owns server-read caching, deduplication, invalidation, and AbortSignals. Imperative actions that can be superseded use explicit latest-request ownership. Shared API transport is bounded so rapid navigation cannot create unbounded concurrent work or queues.

Components clean up dialogs, listeners, observers, timers, subscriptions, and asynchronous work on unmount. Durable writes may complete after navigation, but callbacks from an abandoned page must not mutate or navigate the newly active page.

Overlays that are meaningful navigation state should participate in browser history. For example, opening Calendar show details pushes overlay state so browser Back closes the overlay before leaving the Calendar page.

Live events invalidate only relevant resources. They should refresh visible server state without overwriting dirty form drafts or causing input-driven requests to restart unnecessarily.

## Draft state versus applied state

Forms should distinguish draft values from authoritative saved state. Profile language selection is a draft until Save profile succeeds; only then does the saved locale become authoritative. Similar forms should not visually imply persistence before the backend accepts the change.

Settings forms with more than one editable value use staged saving. Edits remain in a local draft and must not reach the backend until the shared **Unsaved changes** bar's primary **Save changes** action is pressed. The bar is fixed and centered at the bottom of the viewport, provides a secondary **Revert changes** action that restores the last authoritative values, and appears only while the draft differs from saved state. Dirty settings register both browser-exit protection and an in-app navigation warning. A test or preview action may run against the draft, but it must not persist the settings. A control is immediate-save only when the product explicitly defines that single action as its own save operation; one-shot actions such as adding or removing a show are not converted into staged settings.

Every editable control, including its checked state, visual styling, and on/off label, must render from the draft. The saved snapshot is used only for dirty comparison, revert, and the eventual persistence request; it must not hide a draft change from the user.

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

Background jobs are bounded and cancellable. Schedules are persisted, wake the scheduler when relevant state changes, and use deployment timezone for execution. Profile timezone affects presentation, not server execution semantics. Torrent automation is a normal scheduler job with a 15-minute cadence and is disabled by default while the feature is experimental. The persisted scheduler-job enabled state is the single server-wide automation master switch; there is no second automation-settings toggle.

Prefer event-driven refresh and filesystem watchers to frequent polling. Watchers are intentionally not masked by a polling fallback on unusual network filesystems unless a future requirement explicitly adds one.

## Advanced Automation Flows

Automation chains are administrator-owned deployment state. A chain has five fixed stages in top-to-bottom order: build query, search Jackett, filter, choose a candidate, and submit through the existing guarded downloader. A reusable preset may be assigned to any show; a show-specific chain may only be assigned to its own show. Show assignment and enrollment are separate. An unassigned show uses the editable live-action or animated default chain according to its effective media profile; explicitly assigned chains take priority. Defaults are seeded once from the existing Automation settings so upgrades retain their effective filtering behavior, and can later be reset to factory rules. The selected chain's revision, query, and effective filter settings are captured in the run snapshot. Query title/prefix/suffix, MB/min range, seeder minimum, include/exclude terms, release-group and uploader allowlists, release age, preferred quality, size preference, preferred groups/uploaders/providers, and inspection count can be customized per chain. A title override also supplies the show identity used by release and payload checks. Missing chain controls inherit Automation settings. A chain cannot enroll a show or override disabled search, download, or scheduler capabilities; confidence, payload verification, retry, and submission guards remain backend-owned.

The chain editor presents the incoming episode event above stacked stage cards. New chains are created as reusable presets, then can be assigned to a show from the editor or the Automation page; the editor names the selected show in the incoming event. Previously saved show-specific chains remain supported. Saving validates stage identity, order, configuration, and scope on the server, retaining one current definition with an optimistic revision token. Older saved graph definitions are converted to the fixed stage representation when read and rewritten before assignment; supported settings are retained. A deleted custom chain loses its show assignments through foreign-key cascades; default chains cannot be deleted and have an administrator-only reset action. Tests use a validated custom episode event or a historical automation run opened from Previous Runs and perform fresh Jackett discovery without submitting downloads. Preview filtering uses the same rule values and preliminary ranking as scheduled automation; payload verification and submission remain scheduler-only. Test runs retain their event, definition, and bounded credential-free stage outputs; editing a draft records revision zero. The Automation page stages both show enrollment and chain assignment in its shared Save changes flow.
Jackett may expose a magnet, a retrievable `.torrent`, or both. Tally retains both transport alternatives and prefers the inspectable `.torrent`. A magnet cannot expose its file list before metadata exchange, so automatic magnet fallback begins as metadata-only and `Unverified`; it is permitted only after High-confidence identity and all configured metadata-level filters pass. Every automatic magnet submission also creates durable post-verification state keyed to the selected infohash. Before new discovery work, later Torrent automation runs ask qBittorrent for that Tally-owned torrent's resolved file list. Empty/unresolved metadata stays pending without busy-polling. Once files resolve, Tally applies the same video, executable/script, sample-only, show/episode identity, and submission-time MB/min rules used for inspectable torrents. A safe payload is appended as verified follow-up state. A rejected payload is stopped and removed with its partial files, its exact hash is globally blocked, and the reason is appended without rewriting the original decision. If the Tally-owned torrent or metadata never becomes available within the bounded follow-up window, the post-check becomes unavailable rather than inventing verification. If a fetched `.torrent` reveals a meaningful payload or identity failure before submission, that stronger evidence rejects the release and Tally does not bypass it by falling back to the magnet.

Deep inspection is intentionally lazy. Tally does not fetch every search result. The runner makes a cheap preliminary assessment, applies global release filters and preference ranking, creates a bounded shortlist, then fetches and verifies candidates sequentially until one becomes eligible or the shortlist is exhausted. A failed candidate does not prevent the next shortlisted candidate from being considered.

Background discovery is intentionally conservative. Each Torrent automation run has a deployment-global episode-search budget (default 5, bounded 1Ã¢â‚¬â€œ25), and each episode has durable retry state with its next eligible Jackett search time. The default retry pacing is 30 minutes, then 2 hours, then 6 hours; operators can adjust the non-decreasing delays within bounded limits. The retry gate is persisted before an outbound automated search begins, so a timeout, provider error, process restart, or later scheduler tick cannot immediately repeat the same query. Automated Jackett discovery disables immediate provider retries and instead leaves recovery to the durable scheduler cadence, while still using the shared provider coordinator for rate limiting, Retry-After handling, request coalescing, circuit breaking, deadlines, and telemetry.

When more episodes are due than the per-run budget allows, the remainder stay unclaimed and are considered by later runs. By default, due episodes are ordered by most recent airstamp so newly released episodes are serviced before older backlog; operators may disable this recency priority to process the oldest due backlog first. Deep .torrent inspection has its own separately configurable shortlist cap (default 5, bounded 1Ã¢â‚¬â€œ20), and individual torrent metadata fetches remain no-retry.
Torrent automation is explicit opt-in per show. The Torrent automation scheduler job enabled state is the server-wide master control, and the scheduler only considers shows whose deployment-global enrollment policy is `auto`. Unset/default shows are not automated; legacy `never` values are also treated as not enrolled for compatibility. Administrators can change the same enrollment state from the Automation page or the individual show page. The Automation page initially shows the current profile's My Shows whose provider status is `Running`; entering a search reveals any matching show from that profile's full My Shows list regardless of status or whether a future episode is currently known. A separate media-profile override is `auto`, `live`, or `animated`; `auto` classifies animation conservatively from persisted provider show type/genres, while an administrator can override the effective profile without rewriting provider metadata. Global search and download capability switches remain authoritative kill switches; the Torrent automation scheduler enabled state controls automatic execution, and notification choices remain profile-owned.

Runs are serialized/deduplicated per episode with database constraints and eligibility checks. Automatic discovery is for current releases, not historical season backfill: only episodes aired inside the configured retry window are eligible, the default window is 24 hours, and validation hard-caps it at 168 hours (7 days). `no suitable result` is a normal terminal result and is retried with bounded backoff during that window rather than lowering acceptance standards. Episodes marked downloaded in Tally's shared episode state are excluded from automation. Successful automation submissions are tracked separately in immutable run history so they are also not submitted twice unless a later magnet payload rejection explicitly re-opens the episode after backoff.

Immediately before a client side effect, the runner reloads capability settings. If qBittorrent returns an ambiguous error after submission, Tally reconciles against the selected infohash in the Tally category before deciding that the action failed, preventing blind duplicate retries.

`Previous Runs` is the explainability surface. Each run stores immutable episode/show identity, query, settings snapshot, decision-engine version, decision steps, verification outcome, selected release/infohash, and timing. Magnet post-verification is stored as separate durable follow-up state so a run that was originally selected as `High Ã‚Â· Unverified` remains historically accurate while later qBittorrent evidence is shown as Pending, Verified, Rejected, or Unavailable. Filtering records include the counts removed by seeders, keyword rules, release-group/uploader allowlists, confidence, MB/minute ranges, unusable transport, and bad-infohash history; shortlisted rows retain provider/uploader, preference signals, and both live-action and animated size-fit scores. The active media score is used in the decision while the inactive score remains debug-only. Credentials, authenticated provider URLs, cookies, and secrets are never stored there. Later `Mark as bad` feedback is appended instead of rewriting the original decision. A bad exact infohash is blocked globally from future automatic selection but remains visible in manual search with a warning. Detailed run history is pruned after the bounded retention period while the small bad-infohash set remains available to prevent repeat loops.

Episode downloaded state is deployment-global because there is one shared media/download environment; watched state remains profile-owned. A manual downloaded toggle therefore changes the episode for every profile that follows the show, while watched/unwatched and watch-history clearing affect only the acting profile. Unreleased episodes cannot be marked watched or downloaded: individual writes reject setting either state to true before release, and bulk season/show updates always operate on the released subset only; clearing stale state remains allowed. Tally keeps a durable episode-to-infohash link for episode-targeted manual and automated submissions. On each enabled Torrent automation run, it marks a released linked episode downloaded once the Tally-category torrent reaches the configured completion percentage (100% by default); submission alone never marks it. This supports external cleanup services that remove a completed torrent before Tally observes 100%. Schema migration promotes an episode to globally downloaded when any legacy profile had previously marked it downloaded. The Downloads page still reflects client-reported torrent state; media-import/renaming lifecycle management remains out of scope.

## Activity, logs, and notifications

Logs are the complete operational history appropriate to the current profile/administrator scope. Bell notifications are intentionally more selective and configurable so routine user-initiated saves do not become noise.

Activity records must not contain credentials. Notification delivery uses durable outbox/state where required and reloads authoritative settings before sending. Ambiguous or failed external deliveries must be observable without silently duplicating sends.

Torrent automation is global, but notification delivery remains profile-scoped. A shared torrent decision must not be duplicated merely because multiple profiles follow the same show.

## Backups and restore

Backup creation follows a pipeline: snapshot durable state, collect durable files, write the archive, verify it, publish it, and apply retention. The backup directory is the archive inventory source of truth; valid manually copied archives can be discovered without separate registration metadata.

Caches and environment-provided secrets are excluded. UI-managed credentials, automation settings, run state, episode-to-infohash download tracking, per-show automation enrollment, global episode downloaded state, per-show media-profile overrides, and bad-infohash feedback are durable application state and belong in protected backups.

Restore extracts into staging, validates archive paths/sizes/checksums, validates and upgrades the staged database when compatible, prepares durable files, and only then applies database state transactionally. A failed restore must leave the running state usable. Relational restore tests must account for foreign-key cascades and insertion/deletion ordering.

## Secrets and settings

Infrastructure configuration remains environment-owned. User/operator-managed integrations and schedules are stored in SQLite and applied without process restart where supported.

Secrets are revealed only in explicitly authorized settings views. General APIs, logs, errors, notifications, telemetry, activity records, automation history, and decision snapshots must not echo them. When editing a connection, an intentionally blank secret field means retain the saved secret unless the operation explicitly requests removal/replacement.

## Working on a change

Start from the domain entry point and matching tests. Preserve API contracts, transaction boundaries, authorization, provider-coordinator usage, localization, and existing UX unless the task explicitly changes them.

Use one outer `.page` container for normal pages and the shared page-header/navigation patterns rather than route-specific layout workarounds. Keep accessibility labels stable enough for both users and automated browser tests.

When a bug reveals a reusable engineering principle, capture the generalized lesson in `LESSONS.md`. Keep incident-specific history in Git rather than expanding architecture or roadmap files with dated narratives.

Validation expectations are defined in `VALIDATION.md`.
