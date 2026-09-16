# Torrent search / downloads TODO

Branch: `feature/torrent-search-downloads`

## Done
- [x] Created feature branch.
- [x] Renamed torrent action from **Send** to **Download** and changed the icon.
- [x] Fixed qBittorrent submission validation so HTTP success is authoritative instead of requiring an exact `Ok.` body.
- [x] Added/ensured qBittorrent category `tally` and assign Tally submissions to it.
- [x] Updated recent submission refresh to use the app's live invalidation flow, including failed qBittorrent attempts.
- [x] Changed quality filtering so selections within one category are OR matches while different categories combine.
- [x] Added clear buttons and API endpoints for recent searches and recent submissions.
- [x] Bounded the torrent result list height and made the list scroll independently.
- [x] Added a Downloads page with Tally-only qBittorrent status, speed/progress stats, pause/resume, and remove controls.
- [x] Kept remove non-destructive to downloaded files.
- [x] Restricted download controls to torrents in the `tally` category.
- [x] Added automatic download-status polling while the Downloads page is active.
- [x] Hide Torrent search and Downloads when Jackett is disabled.
- [x] Redirect direct visits to hidden torrent routes back to `/calendar`.
- [x] Made saved Jackett state update navigation without a one-frame stale menu state.
- [x] Added English and Ukrainian localization for new UI.
- [x] Added backend tests for qBittorrent category/submission/download controls.
- [x] Added browser regression coverage for navigation gating, Download label, OR quality filtering, and bounded scrolling.
- [x] Fixed existing Jackett and browser qBittorrent fixtures for the new category request.
- [x] Updated existing Playwright expectations for Download/Added labels and the Downloads sidebar entry.

## In progress
- [ ] Complete the full GitHub Checks workflow on the latest code.
- [ ] Investigate and fix any remaining CI/test failures.

## Review findings resolved
- [x] Failed downloader attempts now invalidate torrent history even though failed HTTP mutations do not emit live-update events.

## Final validation
- [ ] Re-review the final diff for missed requirements, stale code, or unsafe edge cases.
- [ ] Confirm frontend build, Go vet/race tests, vulnerability scan, Playwright tests, Docker build, and Trivy all pass.
- [ ] Remove this temporary `TODO.md` before the branch is considered finished.
