# Roadmap

This file tracks current and future work. It is intentionally not a changelog; completed implementation history belongs in Git and durable design decisions belong in `ARCHITECTURE.md` or `LESSONS.md`.

## Current branch

`feature/torrent-automation-confidence`

Status: torrent confidence, verified inspection, automation, and Previous Runs implementation in progress.

### Evaluation and search metadata

- [x] Add one backend-owned torrent evaluation model shared by manual search and future automation.
- [x] Keep confidence (`high`, `medium`, `low`, `rejected`) separate from user preference/ranking.
- [x] Enrich normalized Jackett/Torznab results with useful metadata when supplied: category, infohash, external IDs, grabs, and ratio/download factors.
- [x] Add conservative release parsing/matching for show identity, aliases/year, season/episode forms, quality/source/codec clues, seeders, and size sanity.
- [x] Treat specials, multi-episode releases, season packs, ambiguous identities, and malformed titles conservatively.
- [ ] Surface compact localized confidence on manual search without exposing internal numeric scoring. Only show confidence when an actual episode target is known; do not infer authoritative confidence from arbitrary free-text search.

### Torrent verification

- [x] Deep-inspect only shortlisted/selected candidates rather than every Jackett result.
- [x] Fetch retrievable `.torrent` files through the provider coordination boundary without exposing provider URLs to the browser.
- [x] Parse bencoded torrent metadata locally, derive/validate infohash as appropriate, and inspect the complete file tree before qBittorrent submission.
- [ ] Reject clearly unsuitable payloads such as missing meaningful video, executable/script content, suspicious archive-only payloads, sample-only payloads, episode mismatch, or previously blocked infohashes.
- [x] Record verification as `verified` or `unverified`; automatic download requires a verified candidate.
- [x] Magnet-only results remain available for manual download but are never eligible for automatic download because their payload cannot be inspected first.
- [ ] If the best candidate fails verification, continue through the next bounded shortlist candidate rather than immediately failing the run.

### Global automation

- [ ] Add an `Automation` tab inside Torrent Search; automation/download policy is deployment-global, not profile-owned.
- [ ] Add minimal global controls: enable automatic downloads, preferred quality, minimum seeders, high-confidence-only default, and a sensible built-in release delay/retry policy.
- [ ] Add global per-show download policy: use default/manual, notify only where applicable, auto-download, or never download without creating contradictory per-profile downloader behavior.
- [ ] Keep existing search/download capability toggles backend-authoritative; automation must stop before submission if downloading becomes disabled mid-run.
- [ ] Prevent duplicate grabs and serialize decisions per episode while keeping overall work bounded/cancellable.
- [ ] Treat `no verified candidate` as a normal outcome, not an operational error, and retry later within a bounded window rather than accepting a weak match.
- [ ] Avoid duplicate qBittorrent submissions after ambiguous/time-out responses by reconciling against the selected infohash where possible.

### Previous Runs

- [ ] Add a `Previous Runs` tab inside Torrent Search; this is the user-facing explainability/history surface rather than an "Audit" page.
- [ ] Persist immutable global run records containing episode/show IDs, query, settings snapshot, candidate/filter/ranking decisions, verification result, selected infohash, submission result, timestamps/durations, and decision-engine version.
- [ ] Never persist credentials, authenticated URLs, cookies, API keys, or other connection secrets in run history.
- [ ] Render desktop runs/timeline/details layout and responsive tablet/mobile variants in the existing Tally visual language.
- [ ] Show a chronological decision timeline: search, filtering, candidate ranking, torrent inspection, decision, qBittorrent submission, and later feedback.
- [ ] Allow verified torrent inspection details to expand into the parsed file tree.
- [ ] Add `Mark as bad` feedback with reasons such as wrong show/episode/language, poor quality, corrupt, suspicious files, or other.
- [ ] Preserve original decisions and append later feedback instead of rewriting history.
- [ ] Block the exact bad infohash globally while avoiding automatic whole-indexer/release-group blacklisting from a single report.
- [ ] Previously bad hashes remain visible in manual search with a warning but are automatically excluded from automation.
- [ ] Add bounded retention for detailed run history while retaining the small bad-infohash history needed to prevent repeat selection loops.

### Notifications and ownership

- [ ] Keep automation and Previous Runs deployment-global because one shared downloader is authoritative.
- [ ] Keep notifications profile-owned and fan automation/release outcomes out only to profiles that follow the relevant show and have enabled the corresponding notification category.
- [ ] Ensure one global torrent action cannot produce duplicate downloads merely because multiple profiles follow the show.

### Quality and documentation

- [ ] Add database migrations and backup/restore coverage for new durable state.
- [ ] Add/update English and Ukrainian localization keys and increment bundled catalog versions for all user-facing text.
- [ ] Add deterministic backend tests for release parsing, confidence vs preference, `.torrent` parsing/inspection, blocked hashes, immutable run snapshots, retries/deduplication, and capability enforcement.
- [ ] Add browser coverage for Search confidence, Automation, Previous Runs, bad-run feedback, responsive layouts, and disabled feature states.
- [ ] Update `ARCHITECTURE.md`, `DEVELOPMENT.md`, `VALIDATION.md`, and `LESSONS.md` where the durable manual-only torrent contract changes.
- [ ] Run the complete required validation suite before the branch is considered ready.

## Current product foundations

The following are established capabilities rather than active TODO items:

- Shared TV metadata with profile-owned follows, preferences, favorites, and episode state.
- Per-profile Password or No authentication, opaque profile IDs, transferable administrator roles, sessions, and actor re-authentication for sensitive administrator changes.
- Calendar, library, discovery, show details, episode state, favorites, responsive themes, and profile-specific localization.
- SQLite-backed application settings, schedules, jobs, statistics, logs, bell notifications, Webhook/Discord delivery, and live updates.
- TVmaze metadata coordination with bounded requests, caching, retries, cancellation, rate limiting, and circuit protection.
- Validated SQLite migrations, pre-upgrade snapshots, archive backups, retention, staged validation, and in-process restore.
- Manual Jackett search and qBittorrent submission/download monitoring with operator-managed credentials.
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
