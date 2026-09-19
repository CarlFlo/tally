-- Maps Tally-submitted torrents to their authoritative shared episode marker.
-- Existing automated submissions are retained so active torrents can be
-- reconciled immediately after upgrade.
CREATE TABLE torrent_episode_downloads (
    infohash TEXT PRIMARY KEY,
    episode_id TEXT NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
    created_at INTEGER NOT NULL
);

CREATE INDEX torrent_episode_downloads_episode ON torrent_episode_downloads(episode_id);

INSERT OR IGNORE INTO torrent_episode_downloads(infohash,episode_id,created_at)
SELECT lower(selected_infohash),episode_id,COALESCE(ended_at,started_at)
FROM torrent_automation_runs
WHERE status='downloaded' AND length(trim(selected_infohash)) IN (40,64);
