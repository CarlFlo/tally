# Roadmap

This file tracks current and future work. It is intentionally not a changelog; completed implementation history belongs in Git and durable design decisions belong in `ARCHITECTURE.md` or `LESSONS.md`.

## Current branch

`feature/torrent-automation-confidence`

Status: final torrent automation usability/performance follow-up is implemented; final validation is pending.

### Evaluation and search metadata

- [x] Add one backend-owned torrent evaluation model shared by manual search and automation.
- [x] Keep confidence (`high`, `medium`, `low`, `rejected`) separate from user preference/ranking.
- [x] Enrich normalized Jackett/Torznab results with useful metadata when supplied: category, infohash, external IDs, grabs, and ratio/download factors.
- [x] Add conservative release parsing/matching for show identity, aliases/year, season/episode forms, quality/source/codec clues, seeders, and size sanity.
- [x] Treat specials, multi-episode releases, season packs, ambiguous identities, and malformed titles conservatively.
- [x] Keep manual discovery fast: return Jackett results first, then resolve authoritative episode context/confidence only when a result is expanded or submitted. Do not infer authoritative confidence from arbitrary free-text search.

### Torrent verification

- [x] Deep-inspect only shortlisted/selected candidates rather than every Jackett result.
- [x] Fetch retrievable `.torrent` files through the provider coordination boundary without exposing provider URLs to the browser.
- [x] Parse bencoded torrent metadata locally, derive/validate infohash as appropriate, and inspect the complete file tree before qBittorrent submission.
- [x] Reject clearly unsuitable payloads such as missing meaningful video, executable/script content, suspicious archive-only payloads, sample-only payloads, episode mismatch, or previously blocked infohashes.
- [x] Record verification as `verified` or `unverified`; inspectable `.torrent` candidates require verification, while guarded magnet fallback remains explicitly unverified.
- [x] Prefer an inspectable `.torrent` when available; allow High-confidence magnet fallback only when no usable `.torrent` metadata is available and all metadata-level rules pass.
- [x] If the best candidate fails verification, continue through the next bounded shortlist candidate rather than immediately failing the run.

### Global automation

- [x] Add an `Automation` tab inside Torrent Search; automation/download policy is deployment-global, not profile-owned.
- [x] Add minimal global controls: enable automatic downloads, preferred quality, minimum seeders, release delay, and bounded retry behavior. High confidence remains an invariant rather than a tunable lower threshold; verification state depends on the available transport.
- [x] Keep per-show automation deployment-global and explicit opt-in. Only enrolled shows are searched; notifications remain profile-owned and are not part of download enrollment.
- [x] Keep existing search/download capability toggles backend-authoritative; automation stops before submission if downloading becomes disabled mid-run.
- [x] Prevent duplicate grabs and serialize decisions per episode while keeping overall work bounded/cancellable.
- [x] Treat `no suitable result` as a normal outcome, not an operational error, and retry later within a bounded window rather than accepting a weak match.
- [x] Avoid duplicate qBittorrent submissions after ambiguous/time-out responses by reconciling against the selected infohash where possible.
- [x] Expose Torrent automation as its own independent scheduler job with a default 15-minute cadence and normal scheduler controls/history.

### Global episode download state

- [x] Move authoritative downloaded state to the shared episode record while keeping watched state profile-owned.
- [x] Migrate existing data conservatively: if any profile previously marked an episode downloaded, preserve it as globally downloaded.
- [x] Make single-episode and bulk downloaded toggles immediately visible to every profile that follows the show.
- [x] Keep watched toggles and watch-history clearing isolated to the acting profile.
- [x] Exclude globally downloaded episodes from torrent automation discovery.
- [x] Add schema-v12 upgrade/backup coverage and durable documentation for the global-download/profile-watch invariant.

### Per-show automation enrollment

- [x] Change show automation to explicit opt-in: the global automation switch remains the master control, but only shows explicitly enrolled for automatic downloads are searched by the scheduler.
- [x] Add one backend list endpoint for the current profile's My Shows with global automation enrollment and upcoming-episode state; avoid one request per show.
- [x] On `/search/automation`, add an enrolled-show section with a search field: empty search shows currently active/calendar shows with a known upcoming episode, while search spans all My Shows.
- [x] Allow enrollment to be toggled directly from the automation list with clear enabled/disabled state and responsive feedback.
- [x] Add the same enrollment toggle to the individual show page near the primary show actions; both surfaces must update the same global policy.
- [x] Remove the old three-choice Default / Auto-download / Never UI so the product presents one unambiguous on/off enrollment model; keep backend compatibility for legacy stored policies where practical.
- [x] Add backend/browser coverage for opt-in scheduling, inactive-show search, list/detail synchronization, authorization, and global master-disable behavior.
- [x] Localize the new English/Ukrainian UI and update durable automation documentation.

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

### Sonarr-informed selection refinements

- [x] Treat the existing release delay as a hold window from the first qualifying Jackett `pubDate`, while retaining the episode-air-time guard and keeping missing timestamps neutral.
- [x] Once the delay window matures, evaluate all currently acceptable candidates so a newer better release can win without needing its own full delay.
- [x] Record releases held by the delay window in Previous Runs and version the changed decision engine.

### Post-magnet payload verification

- [x] Persist automatic magnet submissions that still require payload verification so the check survives restarts.
- [x] Extend qBittorrent integration to read the resolved file list for a Tally-owned torrent by infohash.
- [x] Have the torrent-automation scheduler process pending magnet verifications before new discovery work, without long blocking polls.
- [x] Run the resolved qBittorrent file list through the same video/executable/sample/episode identity checks used for inspectable `.torrent` metadata.
- [x] Recalculate actual payload MB/min from the resolved file sizes and reject an out-of-range payload using the submission-time media profile/settings.
- [x] If post-verification fails, stop/remove the Tally-owned torrent with its partial files, globally block the exact infohash, and record the reason without rewriting the original immutable decision.
- [x] If post-verification succeeds, retain the original metadata-only decision and append durable follow-up verification state for Previous Runs.
- [x] Treat unresolved magnet metadata as pending/retryable; do not busy-poll or turn an empty file list into a false rejection.
- [x] Add migration/backup coverage, qBittorrent protocol tests, deterministic automation tests, Previous Runs UI/localization, and durable architecture/validation docs.

### Runtime-aware size profiles and magnet automation

- [x] Add configurable MB/minute ranges for live-action and animated shows, surfaced as compact dual range sliders in Torrent Automation.
- [x] Use episode runtime when available and show runtime as fallback; treat missing runtime/size as neutral rather than guessing.
- [x] Persist TV metadata needed to classify animation where available, auto-detect animated vs live-action from show type/genres, and add a per-show admin override on the show page.
- [x] Apply the active media profile as both a hard size-per-minute filter and a bounded ranking/debug signal without changing show/episode confidence semantics.
- [x] Record both live-action and animated size-fit scores in Previous Runs; visually de-emphasize the inactive profile while keeping it available for debugging.
- [x] Prefer a retrievable Jackett `.torrent` for local payload inspection, but allow High-confidence automatic magnet submission when no usable `.torrent` is available and all metadata-level rules pass.
- [x] Keep magnet fallback explicit in run history as metadata-only/unverified, preserve bad-infohash checks when an infohash can be derived, and never claim payload verification for a magnet.
- [x] Add migration, backend/frontend tests, English/Ukrainian localization, and durable documentation for the new media-profile and magnet-selection rules.

### Manual search result inspection

- [x] Replace the current inline `Why` details disclosure with a click-to-expand panel attached directly beneath the selected `.torrent-result`; expansion must push later results down naturally rather than overlaying them.
- [x] Make the whole result row the primary expansion target while preserving normal Download/copy/action button behavior without accidental toggles.
- [x] Show a clear final evaluation in the expanded panel plus side-by-side strengths/pros and concerns/cons.
- [x] Treat correct show/episode identity as an eligibility assumption rather than a displayed strength; mismatches remain rejection/concern evidence.
- [x] Classify positive evidence such as healthy seeders, runtime-normalized size inside the configured active range, preferred release group/uploader/provider, and verified payload under strengths.
- [x] Classify actual negative/limiting evidence such as an uninspectable magnet payload, ambiguous or missing metadata, low swarm health, previous bad history, or hard rejection reasons under concerns. The absence of a preferred group/uploader/provider is neutral rather than evidence against a release.
- [x] Show useful parsed metadata in the expanded panel (quality, source, codec, release group, uploader when available, provider/indexer, size, seeders, verification state) without exposing the internal numeric score.
- [x] When manual search has authoritative episode context and runtime, show the release's runtime-normalized MB/min ratio in the expanded metadata; omit it for arbitrary free-text searches without runtime context.
- [x] For manual free-text queries that clearly contain one season/episode, resolve episode context from shared local metadata first and cached TVmaze metadata second; require a unique exact title/year match, never auto-follow/persist a remotely resolved show, and leave ambiguous queries unscored.
- [x] Add deterministic coverage for local unfollowed resolution, read-only TVmaze resolution, MB/min from remotely resolved runtime, and ambiguous-TVmaze fallback to ordinary free-text behavior.
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

### Final usability/performance follow-up

- [x] Accept `torrent_automation` consistently in schedule save/manual trigger/job filters and keep manual execution operator-only.
- [x] Defer free-text episode resolution, confidence, trust/history checks, and MB/min evaluation until a result is expanded or submitted.
- [x] Keep server-side submission safety authoritative even when a result was never expanded.
- [x] Remember the latest successful manual search/results across normal SPA navigation using the profile-scoped query cache; sign-out clears it and normal cache expiry bounds retention.
- [x] Fix dual-range track/thumb alignment and use a two-column automation settings layout when viewport width permits.
- [x] Replace identity-as-strength copy with configured MB/min fit and preferred-source evidence.

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

- Shared TV metadata with profile-owned follows, preferences, favorites, and watched state; episode downloaded state is deployment-global.
- Per-profile Password or No authentication, opaque profile IDs, transferable administrator roles, sessions, and actor re-authentication for sensitive administrator changes.
- Calendar, library, discovery, show details, episode state, favorites, responsive themes, and profile-specific localization.
- SQLite-backed application settings, schedules, jobs, statistics, logs, bell notifications, Webhook/Discord delivery, and live updates.
- TVmaze metadata coordination with bounded requests, caching, retries, cancellation, rate limiting, and circuit protection.
- Validated SQLite migrations, pre-upgrade snapshots, archive backups, retention, staged validation, and in-process restore.
- Jackett discovery plus manual and guarded automated qBittorrent submission/download monitoring with operator-managed credentials.
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
