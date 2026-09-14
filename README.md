# Tally

A self-hosted home for the shows you follow. A real episode calendar, personal watchlists, manual torrent search, and a clear view of background work.

Tally runs as one Go process with an embedded React/TypeScript interface, one durable SQLite database, and one `/config` volume. No Node.js runtime, Redis, PostgreSQL, or separate frontend server is needed in production.

## Start with Docker

```sh
docker compose up -d --build
```

Open **http://localhost:8080**. A new installation opens the current month's calendar as `user0`. Add a show to import its known episodes and specials. The calendar and library remain available when metadata providers are offline.

Configuration is optional: copy `.env.example` to `.env` and change only what you need. Compose binds to localhost by default. Set `APP_BIND=0.0.0.0` to expose the port to your LAN. Use `APP_AUTH_MODE=local` or `oidc` when access needs authentication. Put TLS at your reverse proxy and set `APP_PUBLIC_URL` to the public HTTPS origin so session cookies are secure. Forward the original Host header; untrusted forwarding headers are intentionally ignored.

The container runs as UID/GID **10001**, drops capabilities, and stores its data in the named `tally-config` volume. A bind mount must be writable by that UID. Do not place the SQLite volume on a network filesystem.

## Features

- Month, week, and agenda calendar; same-show daily release groups, known full-season labels, day-grouped horizon, live countdowns, Sunday/Monday week starts, and remembered views.
- TVmaze ranked search and recent highly rated suggestions in a card grid, show previews, optimistic Add/Added undo, and a durable import queue with retry notifications.
- Shared cached metadata, seasons, specials, posters, and adaptive background synchronization.
- Immutable profile IDs, built-in/uploaded avatars, individual follows and favorites, direct watched/downloaded toggles, season and aired-episode bulk actions.
- System/light/dark themes, responsive layouts, keyboard-accessible menus and dialogs, per-profile preferences, and server-persisted login appearance for each browser.
- Show action menus on cards and detail pages, clear watch history, Shift-remove, and visible watched/available/upcoming episode stripes.
- A header profile menu and personal notification inbox, with persistent read/dismiss/clear state. System tabs contain administrator-only Jobs, Statistics, and deployment Logs; members can search their own activity at `/logs`.
- Local Argon2id passwords/PINs, first-use password setup and log-based recovery, forced password changes, revocable sessions, and profile isolation.
- OIDC discovery, state/nonce/PKCE, validated tokens, and identity mapping by issuer and subject.
- Manual Torznab search with local seed/size/keyword filters, magnet copy, and replay-protected qBittorrent submission.
- Editable SQLite-backed Torznab, torrent client, webhook, and scheduling settings in separate routed categories.
- Scheduled/manual jobs with seven-day summaries, history filters, failure pause/resume, and safe debug previews; persisted statistics row limits; provider backoff/circuits and in-app alerts.
- Verified snapshot backups, separate cache storage, automatic retention, offline operator restore, safe schema-version checks.

There is no automatic torrent selection or downloading, playback, media scanning, file organization, or torrent lifecycle management.

## Authentication and profiles

`APP_AUTH_MODE=disabled` is the default for trusted local deployments. Profile selection is a convenience boundary; anyone with access can select a profile. With multiple profiles, the browser remembers its choice for a year. Unknown/deleted profile cookies open the picker.

The header profile menu opens **Profile** (`/profile`), **System** (`/system`), **Settings** (`/settings`), and **Sign out**. System and Settings are shown only to the administrator. System has Jobs, Statistics, and Logs tabs; older `/jobs` and `/statistics` links redirect there. Profile settings contain appearance, calendar preferences, Security, and Danger zone. These are regular pages with normal Back, Forward, and refresh behavior.

Sign out from the header menu before choosing another profile. The sign-in screen at `/login` includes **Add profile**, limited by `APP_MAX_PROFILES`. Explicit sign-out stays remembered even when only one profile exists. Signing out revokes the current session, clears cached account data, and updates other open Tally tabs. With OIDC, this signs out of Tally; the identity provider manages its own session and profile provisioning.

In local mode, selecting a profile without credentials opens password setup. The first successful setup secures that profile and starts its session; existing credentials cannot be replaced through setup. This also applies when enabling local authentication on an existing passwordless installation. Complete setup for unprotected profiles before exposing the installation beyond its intended users. Administrators can create a profile without assigning its password, allowing its owner to choose one on first sign-in.

Passwords accept letters, numbers, symbols, and spaces. They are stored as Argon2id hashes and are never trimmed or normalized; control characters and invalid UTF-8 are rejected. The default length is 4-128 characters, configurable through the existing minimum/maximum infrastructure settings. `LOCAL_PASSWORD_ALLOW_NUMERIC_ONLY` is obsolete and ignored. Login attempts have progressive per-profile delay and Argon2 computations have a concurrency bound. Recovery writes a temporary credential to operator logs, expires after 30 minutes or restart, and leaves the old password valid until a replacement is committed.

`user0` cannot be deleted. Authenticated users can only edit their own profile and episode state. The initial `user0` account (shown as admin in the interface) can manage profiles and run deployment maintenance/backup actions. Only the administrator can create and download backup archives; restore remains an offline operator responsibility. OIDC profiles are provisioned on sign-in; the first identity claims `user0` only on a pristine installation. Existing identity mappings are never inferred from names or email addresses.

For OIDC, configure the issuer, client ID, optional secret, and an exact redirect URL ending in `/auth/oidc/callback`. The server uses an authorization-code flow and does not store tokens. Availability of the identity provider is not part of readiness. Switching an existing deployment from local to OIDC may need deliberate operator identity mapping; Tally will not silently claim a profile containing someone else's data.

## Torrent integrations

Configure the shared torrent client in **Settings → Torrent client**. Choose qBittorrent from the supported-client dropdown, enter its Web UI URL and API key, then use **Test connection** and **Save torrent client**. The test checks authentication and API access without saving changes or sending a torrent. Saving applies immediately and survives restarts. Choosing **No torrent client** and saving disables sending and removes the stored connection details; search and magnet copying remain available.

Use an address reachable from the Tally server/container. For example, another container on the same network may be `http://qbittorrent:8080`; `localhost` inside the Tally container refers to Tally itself. Reverse-proxy paths are supported. Keep credentials in the separate fields, not the URL. Paste only the `qbt_...` key; Tally adds `Authorization: Bearer` to each request. Saved keys are visible by default in operator settings, with a Hide/Show control. Leaving an existing key unchanged preserves it. Enter a new key to rotate it, or disable the client to remove its stored details. When changing the address, re-enter the key to prevent forwarding an existing secret to a different connection.

The connection is shared by all profiles. Only the `user0` deployment owner can view/edit its details or test draft connections, in every authentication mode. Other profiles can use the configured client for manual sends. Connection fields use ordinary text/URL inputs with password-manager ignore hints. Saved secrets are available only through explicit operator connection settings, with non-cacheable responses; general APIs and logs do not expose them. They must be recoverable to authenticate to the client, so they are stored in the private SQLite database and included in backups. Protect the configuration volume and archives accordingly; they are not encrypted archives.

qBittorrent API keys require version 5.2.0 or later (WebAPI 2.14.1). Generate a key in **Preferences / WebUI / API Key / Generate**, as described in the [official API-key documentation](https://github.com/qbittorrent/qBittorrent/wiki/API-Key-Authentication-%28%E2%89%A5v5.2.0%29). Tally uses API-key authentication for both tests and submissions.

Torrent-client settings are read exclusively from SQLite; legacy `DOWNLOAD_CLIENT*` environment variables are ignored and can be removed. Previously saved username/password configurations keep their address but require an API key in Settings before sending becomes available. Saving the API key replaces the old credential fields; passwords cannot be converted into API keys.

Configure Torznab endpoints in **Settings → Torrent search**. Add a name, stable provider ID, endpoint URL and API key, then save. Enable or disable each provider without deleting its details. Up to eight endpoints are supported. Prowlarr endpoints commonly use `http://prowlarr:9696/1/api`; Jackett may use `http://jackett:9117/api/v2.0/indexers/all/results/torznab/api`. An aggregate endpoint can search multiple indexers. Changes apply immediately.

Profiles remember their selected providers. Search runs at most four providers concurrently, merges results by seed count, and retains successful results when another provider fails. Each provider has its own coordinator limits, cache, and circuit. Search is explicit and capped at 100 results per provider. Episode searches prefill an editable query. Quality filters are title keyword matches and combine with AND. Torrent download URLs remain server-side; the browser receives short-lived profile-bound selection tokens. Torrent files are fetched only from the corresponding configured Torznab origin, and redirects are refused. Indexers that redirect torrent downloads to another host should return magnets instead.

If a send fails or times out, check qBittorrent before retrying: the client may have accepted the request. Replaying the same submission key will not send it twice. Sending does not mark an episode downloaded.

The first profile, **user0**, is the administrator in every authentication mode. Only user0 can open Settings, Jobs, Statistics, and Logs, or use their APIs. Other profiles retain their calendar, shows, manual torrent search, and personal settings. With authentication disabled, profile selection remains intended for a trusted network. Personal settings at **/profile** contain appearance, Sunday/Monday week start, security, sign-out, and a **Danger zone** for deleting your own account. Deletion revokes all sessions and removes personal follows/preferences/progress; shared metadata and other profiles remain. The permanent user0 account cannot be deleted.

Use the three-dot show menu to clear all watched progress, refresh metadata (on the detail page), or remove a follow. Clearing watched history preserves downloaded markers. Removing normally asks for confirmation; holding Shift while clicking Remove show skips that prompt. Episode stripes are green when watched, light blue when available, and grey for upcoming releases. Completed/all-caught-up badges summarize progress. Date-only releases show Time TBA until availability can be established. Logs record actions from this upgrade onward; previous activity is not reconstructed.

## Background work and configuration

See `.env.example` for infrastructure configuration: authentication, paths, retention, concurrency, and request/job limits. These values are validated before startup and read-only in the UI. Settings has separate Deployment, Profiles, Torrent client, Torrent search, Notifications, Scheduling, and Debug pages. Shared connections and schedules are saved in SQLite; appearance, filters, debug previews, and statistics row limits are saved per profile. Schedules use **UTC**, while the calendar uses each profile's IANA timezone (initialized from its browser).

TV metadata is shared across all profiles. Refresh is due-based: about six hours for episodes within a week, two days for distant scheduled episodes, daily for active shows without a next episode, three days when status is undetermined, and monthly for ended shows, plus jitter. Each scheduler pass handles at most `JOB_MAX_BATCH_SIZE` shows. Removing a follow preserves shared metadata and the profile's previous episode state.

All integration HTTP uses the coordinator: per-provider concurrency, a 500 ms minimum request interval, response deadlines/size limits, caching/coalescing, conditional requests, bounded retries, Retry-After, and persisted circuits. Readiness only checks local service health. Images are limited to TVmaze's image host. HTTP redirects are deliberately refused.

`JOB_MAX_RETRIES` bounds transient HTTP retries inside a job. A failed entity is deferred for one hour instead of repeatedly rerunning the whole job. A crashed running job is recorded as interrupted at startup; due entities are picked up on a later scheduler pass. Manual refresh bypasses freshness, never provider safety controls. The Jobs page refreshes every three seconds while open. Three consecutive failed runs pause an automatic schedule; a successful manual retry or explicit Resume clears the pause. Disabled schedules retain their cron expression and can still be run manually. Timeouts count as failures; explicit cancellation does not. Job cards show seven-day success/failure counts, readable schedules in the profile's 12/24-hour format, and the latest status. History filters persist per profile. Debug mode exposes labelled visual previews without running jobs, changing real results, or sending webhooks. Recent requests and next metadata checks default to 20 rows, with separate saved 20/50/100 choices.

Use **Settings > Notifications** to configure **Notification Services**: Webhook with a JSON payload template, or Discord with a webhook URL, bot display name, and message prefix. Test Notification sends a sample using the current form without saving it. The master switch applies immediately and preserves both service configurations. Event checkboxes select system errors, failed jobs, show additions/removals, watch-history clears, and new episode releases.

Errors have priority and are dispatched immediately through the coordinator; releases with a confirmed timestamp are queued for the next selected daily time in an IANA timezone, including daylight-saving changes. Only followed shows qualify. Pending deliveries and release deduplication survive restart and backups. Changing the daily time applies to subsequently queued releases. Disabling notifications skips pending deliveries and does not replay events from the disabled period. An already dispatched request may finish. Failed/ambiguous sends are logged and are not automatically retried; inspect Logs and use Test Notification to diagnose the connection. Webhook templates accept `{{message}}`, `{{event}}`, `{{show}}`, `{{time}}`, `{{level}}`, and `{{key}}` inside JSON string values. Values are safely encoded. Discord sends use server confirmation and suppress automatic mentions ([Discord webhook reference](https://docs.discord.com/developers/resources/webhook#execute-webhook)).

Use **Settings > Scheduling** to edit standard five-field cron expressions. **Run automatically** saves immediately, preserving any unsaved cron text. **Save schedule** applies that text separately. Changes appear in Logs. Defaults are metadata hourly, maintenance at 03:30 UTC, and backups at 03:00 UTC. The upgrade changes the untouched old 15-minute metadata default to hourly, preserving schedules previously edited by a user.

On the first upgrade, valid legacy `TORZNAB_*`, `WEBHOOK_URL`, and schedule environment values seed settings once, while existing job schedules are retained. After that, SQLite is authoritative: stale or malformed values in these old environment variables cannot overwrite UI changes or block startup. Remove those old entries from your deployment after upgrading.

Adding shows records intent immediately in a persistent queue. Other cards remain usable while metadata imports run. Clicking Added reverses the intent, including during an import. Failures restore Add and produce a top notification with Retry even after discovery closes. Shared metadata is reused across profiles. Favorites appear above the rest of each profile's library and have compact calendar stars. Discovery suggestions sample today, yesterday, and a week ago from TVmaze's US broadcast and worldwide streaming schedules, retaining rated shows (7+) and showing up to 24. This uses six bounded, cached requests, not a global popularity chart.

The durable database is `/config/app.db` (WAL, foreign keys). Disposable provider/search responses live in `/config/cache.db`; poster files live in `/config/cache/images`. Raw request history defaults to 30 days and daily aggregates to one year. Caches and histories are pruned by maintenance. Local episode and profile state is never pruned by maintenance.

## Backups, restore, and upgrades

Use **Settings > Backups** for automatic scheduling, enable/disable, retention (1-1,000 automatic archives), manual creation, and **Download**. Failed attempts are marked red. Saved retention applies at the next successful backup without restarting. Manual archives are never automatically deleted.

The backup directory is always `APP_DATA_DIR/backups` (`/config/backups` in Docker). To place archives elsewhere on the host, mount that host location at `/config/backups` and make it writable by container UID 10001. `BACKUP_PATH` is ignored; files from a previously customized path are not moved automatically. A valid legacy `BACKUP_KEEP` is imported once when the new backup settings are first created, then SQLite is authoritative. Scheduling and enablement already migrate from their legacy environment values once.

Archives contain a SQLite `VACUUM INTO` snapshot, avatars, and a manifest with SHA-256 checksums. Creation validates checksums, SQLite integrity/foreign keys, schema, and `user0`. Caches, environment variables, and temporary credentials are excluded. Credential hashes, identity mappings, and UI-managed connection credentials, notification settings/outbox, personal inbox markers/dismissals, activity logs, browser appearance preferences, backup settings, schedules, favorites, and queued library actions **are included**, so protect archives like the deployment itself.

Operator commands take a deployment lock. Stop the application before the offline backup/restore commands:

```sh
docker compose stop tally
docker compose run --rm tally backup
docker compose run --rm tally verify-backup /config/backups/tally-manual-EXAMPLE.zip
docker compose run --rm tally restore /config/backups/tally-manual-EXAMPLE.zip
docker compose up -d
```

Replace the example filename with an actual archive. Download archives in Settings > Backups, or copy a chosen archive from the volume using `docker compose cp tally:/config/backups/ACTUAL.zip ./ACTUAL.zip` while the container exists. With the app stopped, `docker compose run --rm tally delete-backup ACTUAL.zip` deliberately deletes an archive and its record.

Offline account tools also take the deployment lock: `tally reset-password user1` prints a new temporary password and revokes that profile's sessions; `tally link-identity user0 https://issuer.example.com immutable-subject` links an existing profile to an OIDC identity. Use the exact issuer and subject from the identity provider. Existing mappings cannot be overwritten by this command.

Restore validates and stages the complete archive before replacing anything. Existing database files and avatars are preserved under a `pre-restore-*` directory for rollback. Invalid archives, path traversal, unknown schema versions, and bad checksums are rejected. Keep pre-restore directories until you have checked the restored application.

Before changing an existing schema, startup takes and verifies a pre-upgrade database snapshot. A failed snapshot prevents migration. Schema changes and schema validation run transactionally; failures write `migration-failed.json`, leave the snapshot intact, and prevent the server becoming ready. Binaries refuse newer schemas. Schema 5 adds personal inbox state and upgrades only the untouched metadata default to hourly. Schema 4 adds activity logs, browser appearance, and durable notification delivery. Schema 3 adds editable settings, favorites, queued library actions, and job controls. Schema 2 introduced torrent-client settings. Older supported archives remain restorable and are upgraded sequentially when opened. Add explicit sequential migrations for future versions.

## Development

Requirements: Go matching `go.mod` (automatic toolchain download is supported), Node.js 22+, npm.

```sh
cd frontend
npm ci
npm run build
cd ..
go test ./...
go vet ./...
go build -o tally ./cmd/server
APP_DATA_DIR=./config ./tally
```

PowerShell:

```powershell
$env:APP_DATA_DIR = './config'
go run ./cmd/server
```

Frontend development: start Go, then `npm run dev` in `frontend`. Its API proxy targets port 8080. The compiled `web/dist` is embedded in the binary. Frontend production assets are checked into this initial source delivery so a Go-only build is runnable; rebuild them after UI changes.

Tests use isolated temporary databases and local fake providers/clients. `frontend/tests` contains browser checks; use `npm run test:e2e` after installing Chromium with `npx playwright install chromium`. See [the validation record](docs/VALIDATION.md) for completed checks and deployment-specific limits, and `TODO.md` for implementation status.

The Go implementation uses focused files under `cmd/server` and domain packages under `internal`. See [the code map](docs/ARCHITECTURE.md) for handler, repository, job, provider, and command entry points. Files target one responsibility and about 150 lines.

Torrent adapters live under `internal/torrent`: `qbittorrent.go` implements the download client, `torznab.go` implements search, and `torrent.go` contains shared interfaces and types. To support another client, add its implementation and field definition in its own file, then return its definition from `clientAdapters()` in `client_registry.go`. The UI dropdown and connection fields come from that registry. Implement `TestConnection`, `AddMagnet`, and `AddTorrent` through the provider coordinator. The qBittorrent adapter uses the official Web UI API with Bearer API-key authentication; it does not call the cookie-based login or logout endpoints.

## Data attribution

TV metadata and images are provided by [TVmaze](https://www.tvmaze.com/api) under [CC BY-SA](https://creativecommons.org/licenses/by-sa/4.0/). Provider summaries are converted to plain text before display. Tally is an original implementation inspired by the supplied product specification.
