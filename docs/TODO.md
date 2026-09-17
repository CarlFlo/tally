# Roadmap

This file tracks current and future work. It is intentionally not a changelog; completed implementation history belongs in Git and durable design decisions belong in `ARCHITECTURE.md` or `LESSONS.md`.

## Current branch

`feature/torrent-search-downloads`

Status: per-profile authentication rework in progress.

- [x] Torrent search submissions use clear download actions and correct success handling.
- [x] Recent torrent submissions refresh through the normal live-update path instead of requiring a page reload.
- [x] Add a Downloads page backed by the configured torrent client.
- [x] Keep torrent search and torrent downloading as independent capabilities with separate settings toggles.
- [x] Keep feature toggles outside the provider/client configuration they gate.
- [x] Hide Torrent search navigation when search is disabled and Downloads navigation/actions when downloading is disabled.
- [x] Enforce disabled capabilities on the backend as well as hiding unavailable UI.
- [x] Browser Back closes a Calendar show overlay before navigating away from the page.
- [x] Saving a profile language applies only on Save profile, and the resulting success toast renders in the newly active locale.
- [x] Reconcile bundled localization files on startup while leaving uniquely named custom locales untouched.
- [x] Add backend and browser regression coverage for changed behavior.
- [ ] Replace deployment-wide local/no-auth selection with per-profile Password or No authentication.
- [ ] Let profile creation and profile management explicitly choose and change the authentication method.
- [ ] Show each profile's authentication method on `/settings/profiles` and clearly warn when No authentication is selected.
- [ ] Use normal server sessions for no-auth profiles instead of a global disabled-auth path.
- [ ] Keep existing OIDC code/data dormant and unlinked from the active login flow for now.
- [ ] Update authentication regression coverage and run the full required validation suite.

## Current product foundations

The following are established capabilities rather than active TODO items:

- Shared TV metadata with profile-owned follows, preferences, favorites, and episode state.
- Opaque profile IDs, transferable administrator roles, sessions, and actor re-authentication for sensitive administrator changes.
- Calendar, library, discovery, show details, episode state, favorites, responsive themes, and profile-specific localization.
- SQLite-backed application settings, schedules, jobs, statistics, logs, bell notifications, Webhook/Discord delivery, and live updates.
- TVmaze metadata coordination with bounded requests, caching, retries, cancellation, rate limiting, and circuit protection.
- Validated SQLite migrations through schema 7, pre-upgrade snapshots, archive backups, retention, staged validation, and in-process restore.
- Manual Jackett search and qBittorrent submission/download monitoring with operator-managed credentials.
- Version-managed English and Ukrainian bundled locales plus validated hot-loaded custom locale files.

## Deferred scope

These remain intentionally outside the current product unless a future task explicitly changes the scope:

- OIDC/SSO enrollment and provider configuration. Existing OIDC implementation/data may remain dormant for future work.
- Automatic torrent selection or automatic downloading.
- Torrent scanning, renaming, importing, or media-library lifecycle management.
- Additional downloader adapters beyond the currently supported client.
- Movies or non-TV media tracking.
- Playback or media streaming.
- HTML scraping of torrent/indexer sites.

## Maintenance rules

Keep this file short and forward-looking. Remove completed task detail once it is represented by durable documentation and regression coverage. Do not append dated implementation diaries, CI transcripts, asset hashes, local machine state, or superseded approaches here.
