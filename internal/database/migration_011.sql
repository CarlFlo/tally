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
