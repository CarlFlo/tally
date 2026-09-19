# Roadmap

This file tracks current and future work. It is intentionally not a changelog; completed implementation history belongs in Git and durable design decisions belong in `ARCHITECTURE.md` or `LESSONS.md`.

## Current work

- [ ] Complete the latest-head full validation baseline before deployment, including the Playwright suite and container scan.

Completed torrent automation work is recorded in Git; durable behavior is described in the architecture and validation guides.

## Current product foundations

The following are established capabilities rather than active TODO items:

- Shared TV metadata with profile-owned follows, preferences, favorites, and watched state; episode downloaded state is deployment-global.
- Per-profile Password or No authentication, opaque profile IDs, transferable administrator roles, sessions, and actor re-authentication for sensitive administrator changes.
- Calendar, library, discovery, show details, episode state, favorites, responsive themes, and profile-specific localization.
- SQLite-backed application settings, schedules, jobs, statistics, logs, bell notifications, Webhook/Discord delivery, and live updates.
- TVmaze metadata coordination with bounded requests, caching, retries, cancellation, rate limiting, and circuit protection.
- Validated SQLite migrations, pre-upgrade snapshots, archive backups, retention, staged validation, and in-process restore.
- Jackett discovery plus manual and guarded automated qBittorrent submission/download monitoring with operator-managed credentials and configurable episode-download completion marking.
- Version-managed English and Ukrainian bundled locales plus validated hot-loaded custom locale files.

## Deferred scope

These remain intentionally outside the current product unless a future task explicitly changes the scope:

- Torrent renaming/importing or media-library lifecycle management.
- Additional downloader adapters beyond the currently supported client.
- Movies or non-TV media tracking.
- Playback or media streaming.
- HTML scraping of torrent/indexer sites.

## Maintenance rules

Keep this file short and forward-looking. Remove completed task detail once it is represented by durable documentation and regression coverage. Do not append dated implementation diaries, CI transcripts, asset hashes, local machine state, or superseded approaches here.
