# Branch TODO — feature/torrent-search-downloads

## Requested changes

- [ ] Calendar show overlay: browser Back closes the overlay before leaving /calendar.
- [ ] Add a standalone **Toggle torrent downloads** control above/outside the qBittorrent configuration section on `/settings/torrent`.
- [ ] Make **Toggle torrent search** a standalone control above/outside the Jackett configuration section on `/settings/search`, so the feature toggle is conceptually separate from Jackett itself.
- [ ] Keep Jackett/qBittorrent connection configuration intact when their corresponding feature toggle is off; the toggle only enables/disables the Tally feature.
- [ ] Hide Torrent search navigation when torrent search is disabled.
- [ ] Hide Downloads navigation when torrent downloads are disabled.
- [ ] When torrent search is disabled, hide/disable the related search page functionality as specified while preserving saved Jackett configuration.
- [ ] When torrent downloads are disabled, hide/disable the related download page/actions as specified while preserving saved qBittorrent configuration.
- [ ] Hide torrent-search Download actions when torrent downloads are disabled; keep magnet copy available.
- [ ] Downloads pause/resume updates immediately after a successful command without requiring refresh.
- [ ] Downloads removal prompts for torrent-only vs torrent + files on disk.
- [ ] Remove the tally-category explanatory string from Downloads in English and Ukrainian.
- [ ] Calendar overlay "Search torrents" navigates to /search, fills the show title, and automatically starts the search.
- [ ] Review/update backend/frontend tests for changed behavior.
- [ ] Run relevant verification and review final branch diff.
- [ ] Remove this temporary file before finalizing the branch.

## Notes / discoveries

- Work in progress.
