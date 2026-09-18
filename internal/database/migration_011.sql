CREATE TABLE torrent_magnet_verifications (
    run_id TEXT PRIMARY KEY REFERENCES torrent_automation_runs(id) ON DELETE CASCADE,
    infohash TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','verified','rejected','unavailable')),
    attempts INTEGER NOT NULL DEFAULT 0,
    last_checked_at INTEGER NOT NULL DEFAULT 0,
    completed_at INTEGER,
    assessment TEXT NOT NULL DEFAULT '{}',
    size_profile TEXT NOT NULL DEFAULT '{}',
    error TEXT NOT NULL DEFAULT ''
);

CREATE INDEX torrent_magnet_verifications_pending
    ON torrent_magnet_verifications(status,last_checked_at)
    WHERE status='pending';

-- A magnet decision remains immutable as "downloaded" even when later payload
-- evidence rejects and removes it. More than one historical downloaded attempt
-- must therefore be possible for the same episode; the runner's accepted-run
-- eligibility check remains authoritative while the one-running constraint
-- continues to serialize decisions.
DROP INDEX IF EXISTS torrent_automation_one_downloaded_episode;
CREATE INDEX torrent_automation_downloaded_episode
    ON torrent_automation_runs(episode_id,started_at DESC)
    WHERE status='downloaded';
