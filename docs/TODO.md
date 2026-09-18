# Roadmap

This file tracks current and future work. It is intentionally not a changelog; completed implementation history belongs in Git and durable design decisions belong in `ARCHITECTURE.md` or `LESSONS.md`.

## Current branch

`feature/torrent-automation-confidence`

Status: runtime-aware size profiling and automatic magnet fallback are in progress; final validation is pending.

### Evaluation and search metadata

- [x] Add one backend-owned torrent evaluation model shared by manual search and automation.
- [x] Keep confidence (`high`, `medium`, `low`, `rejected`) separate from user preference/ranking.
- [x] Enrich normalized Jackett/Torznab results with useful metadata when supplied: category, infohash, external IDs, grabs, and ratio/download factors.
- [x] Add conservative release parsing/matching for show identity, aliases/year, season/episode forms, quality/source/codec clues, seeders, and size sanity.
- [x] Treat specials, multi-episode releases, season packs, ambiguous identities, and malformed titles conservatively.
- [x] Surface compact localized confidence on manual search without exposing internal numeric scoring. Only show confidence when an actual episode target is known; do not infer authoritative confidence from arbitrary free-text search.

### Torrent verification

- [x] Deep-inspect only shortlisted/selected candidates rather than every Jackett result.
- [x] Fetch retrievable `.torrent` files through the provider coordination boundary without exposing provider URLs to the browser.
- [x] Parse bencoded torrent metadata locally, derive/validate infohash as appropriate, and inspect the complete file tree before qBittorrent submission.
- [x] Reject clearly unsuitable payloads such as missing meaningful video, executable/script content, suspicious archive-only payloads, sample-only payloads, episode mismatch, or previously blocked infohashes.
- [x] Record verification as `verified` or `unverified`; automatic download requires a verified candidate.
- [x] Magnet-only results remain available for manual download but are never eligible for automatic download because their payload cannot be inspected first.
- [x] If the best candidate fails verification, continue through the next bounded shortlist candidate rather than immediately failing the run.

### Global automation

- [x] Add an `Automation` tab inside Torrent Search; automation/download policy is deployment-global, not profile-owned.
- [x] Add minimal global controls: enable automatic downloads, preferred quality, minimum seeders, release delay, and bounded retry behavior. High + Verified remains an invariant rather than a tunable lower threshold.
- [x] Add a global per-show download override UI using download-only choices: `Default`, `Auto-download`, or `Never auto-download`. Notifications remain profile-owned and are not part of this policy.
- [x] Keep existing search/download capability toggles backend-authoritative; automation stops before submission if downloading becomes disabled mid-run.
- [x] Prevent duplicate grabs and serialize decisions per episode while keeping overall work bounded/cancellable.
- [x] Treat `no verified candidate` as a normal outcome, not an operational error, and retry later within a bounded window rather than accepting a weak match.
- [x] Avoid duplicate qBittorrent submissions after ambiguous/time-out responses by reconciling against the selected infohash where possible.
- [x] Expose Torrent automation as its own independent scheduler job with a default 15-minute cadence and normal scheduler controls/history.

### Follow-up automation controls

- [x] Expand `/search/automation` into a practical release-selection dashboard rather than relying only on quality and minimum seeders.
- [x] Add automation include-keyword and exclude-keyword filters, with useful default suggestions and an explicit reset-to-default/reset-filters action.
- [x] Add release-group rules: whitelist/allow selected groups and separately prioritize preferred groups when otherwise valid candidates are ranked.
- [x] Normalize Jackett uploader/author metadata when an indexer exposes it; uploader rules must degrade gracefully when the metadata is unavailable because Jackett/indexers do not guarantee it for every result.
- [x] Add uploader rules that can allow/filter and prioritize trusted uploaders without treating uploader trust as proof that the show/episode identity is correct.
- [x] Add an explicit preferred provider/indexer list using the existing Jackett provider/indexer name so known high-quality sources can materially influence ordering even when uploader metadata is unavailable.
- [x] Keep trust/preference signals separate from confidence: preferred group/uploader/provider should materially influence ordering among valid candidates but must not override wrong-show, wrong-episode, blocked-payload, or verification failures.
- [x] Persist these automation rules in the global settings snapshot so Previous Runs can explain exactly which filters/trust rules affected each historical decision.
- [x] Add deterministic tests for keyword filters, reset/default behavior, group whitelist/priorities, uploader metadata present/missing, preferred-provider ranking, trusted-source ranking, and the separation of trust/preference from confidence.

### Runtime-aware size profiles and magnet automation

- [ ] Add configurable MB/minute ranges for live-action and animated shows, surfaced as compact dual range sliders in Torrent Automation.
- [ ] Use episode runtime when available and show runtime as fallback; treat missing runtime/size as neutral rather than guessing.
- [ ] Persist TV metadata needed to classify animation where available, auto-detect animated vs live-action from show type/genres, and add a per-show admin override on the show page.
- [ ] Apply the active media profile as both a hard size-per-minute filter and a bounded ranking/debug signal without changing show/episode confidence semantics.
- [ ] Record both live-action and animated size-fit scores in Previous Runs; visually de-emphasize the inactive profile while keeping it available for debugging.
- [ ] Prefer a retrievable Jackett `.torrent` for local payload inspection, but allow High-confidence automatic magnet submission when no usable `.torrent` is available and all metadata-level rules pass.
- [ ] Keep magnet fallback explicit in run history as metadata-only/unverified, preserve bad-infohash checks when an infohash can be derived, and never claim payload verification for a magnet.
- [ ] Add migration, backend/frontend tests, English/Ukrainian localization, and durable documentation for the new media-profile and magnet-selection rules.

### Manual search result inspection

- [x] Replace the current inline `Why` details disclosure with a click-to-expand panel attached directly beneath the selected `.torrent-result`; expansion must push later results down naturally rather than overlaying them.
- [x] Make the whole result row the primary expansion target while preserving normal Download/copy/action button behavior without accidental toggles.
- [x] Show a clear final evaluation in the expanded panel plus side-by-side strengths/pros and concerns/cons.
- [x] Collapse redundant identity reasons such as `Show matches` + `Episode matches` into a concise human-readable statement such as `Correct show and episode`.
- [x] Classify positive evidence such as healthy seeders, matching identity, useful metadata, preferred release group/uploader/provider, and verified payload under strengths.
- [x] Classify actual negative/limiting evidence such as an uninspectable magnet payload, ambiguous or missing metadata, low swarm health, previous bad history, or hard rejection reasons under concerns. The absence of a preferred group/uploader/provider is neutral rather than evidence against a release.
- [x] Show useful parsed metadata in the expanded panel (quality, source, codec, release group, uploader when available, provider/indexer, size, seeders, verification state) without exposing the internal numeric score.
- [x] Keep the compact collapsed result row readable and consistent with the existing Tally visual language.
- [x] Add browser coverage for result expansion/collapse, button interactions, strengths/concerns rendering, and layout behavior with multiple adjacent results.

### Previous Runs

- [x] Add a `Previous Runs` tab inside Torrent Search; this is the user-facing explainability/history surface rather than an "Audit" page.
- [x] Persist immutable global run records containing episode/show IDs, query, settings snapshot, candidate/filter/ranking decisions, verification result, selected infohash, submission result, timestamps/durations, and decision-engine version.
- [x] Never persist credentials, authenticated URLs, cookies, API keys, or other connection secrets in run history.
- [x] Render desktop runs/timeline/details layout and responsive tablet/mobile variants in the existing Tally visual language.
- [x] Show a chronological decision timeline: search, filtering, candidate ranking, torrent inspection, decision, qBittorrent submission, and later feedback.
- [x] Allow verified torrent inspection details to expand into the parsed file tree.
- [x] Add `Mark as bad` feedback with reasons such as wrong show/episode/language, poor quality, corrupt, suspicious files, or other.
- [x] Preserve original decisions and append later feedback instead of rewriting history.
- [x] Block the exact bad infohash globally while avoiding automatic whole-indexer/release-group blacklisting from a single report.
- [x] Previously bad hashes remain visible in manual search with a warning but are automatically excluded from automation.
- [x] Add bounded retention for detailed run history while retaining the small bad-infohash history needed to prevent repeat selection loops.

### Notifications and ownership

- [x] Keep automation and Previous Runs deployment-global because one shared downloader is authoritative.
- [x] Keep release/bell notifications profile-owned and scoped through each profile's followed shows/calendar and notification preferences; do not add notification controls to global torrent automation.
- [x] Ensure one global torrent action cannot produce duplicate downloads merely because multiple profiles follow the show.

### Navigation and responsiveness

- [x] Re-check `/search`, `/search/automation`, and `/search/runs` navigation for the intermittent unresponsive behavior observed during testing and fix the clean-settings synchronization loop that could starve a route commit.
- [x] Stress-cycle the three Torrent Search tabs in one browser document and verify navigation remains responsive, inputs/buttons remain interactive, and no stale searches/listeners/subscriptions block the tab.
- [x] Confirm active Search requests are cancelled cleanly on route changes without delaying navigation or leaving stale callbacks.
- [x] Add or refine browser regression coverage for rapid repeated Torrent Search tab navigation rather than masking any problem with cooldowns or forced reloads.

### Quality and documentation

- [x] Add database migration for automation/run history state and keep it inside the normal backup/restore database path.
- [x] Add/update English and Ukrainian localization keys and increment bundled catalog versions for all existing user-facing torrent automation text.
- [x] Complete deterministic backend coverage for the initial release parsing, confidence vs preference, `.torrent` parsing/inspection, blocked hashes, immutable run snapshots, retries/deduplication, capability enforcement, verified automation, and magnet exclusion.
- [x] Add initial browser coverage for Search confidence, Automation, Previous Runs, bad-run feedback, responsive layouts, and disabled feature states.
- [x] Update `ARCHITECTURE.md`, `DEVELOPMENT.md`, `VALIDATION.md`, and `LESSONS.md` where the durable manual-only torrent contract changes.
- [x] Update English and Ukrainian localization for all follow-up filter/trust/result-evaluation UI.
- [x] Update durable documentation for the automation filtering/trust model and its validation rules.
- [ ] Run the complete required validation suite on the final branch head before the branch is considered ready.

## Current product foundations

The following are established capabilities rather than active TODO items:

- Shared TV metadata with profile-owned follows, preferences, favorites, and episode state.
- Per-profile Password or No authentication, opaque profile IDs, transferable administrator roles, sessions, and actor re-authentication for sensitive administrator changes.
- Calendar, library, discovery, show details, episode state, favorites, responsive themes, and profile-specific localization.
- SQLite-backed application settings, schedules, jobs, statistics, logs, bell notifications, Webhook/Discord delivery, and live updates.
- TVmaze metadata coordination with bounded requests, caching, retries, cancellation, rate limiting, and circuit protection.
- Validated SQLite migrations, pre-upgrade snapshots, archive backups, retention, staged validation, and in-process restore.
- Jackett discovery plus manual and verified automated qBittorrent submission/download monitoring with operator-managed credentials.
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
