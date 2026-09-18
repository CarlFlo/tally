CREATE TABLE torrent_automation_episode_state (
    episode_id TEXT PRIMARY KEY REFERENCES episodes(id) ON DELETE CASCADE,
    attempts INTEGER NOT NULL DEFAULT 0 CHECK(attempts >= 0),
    next_search_at INTEGER NOT NULL DEFAULT 0,
    last_search_at INTEGER NOT NULL DEFAULT 0,
    updated_at INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX torrent_automation_episode_due
    ON torrent_automation_episode_state(next_search_at);

-- Preserve an approximate retry gate for unfinished historical attempts when
-- upgrading. New attempts use the live configurable policy after migration.
INSERT INTO torrent_automation_episode_state(episode_id,attempts,next_search_at,last_search_at,updated_at)
SELECT
    r.episode_id,
    COUNT(*),
    MAX(CASE
        WHEN r.status='downloaded' AND m.status='rejected' THEN COALESCE(m.completed_at,r.started_at)
        ELSE r.started_at
    END) + CASE
        WHEN COUNT(*)=1 THEN 30 * 60
        WHEN COUNT(*)=2 THEN 2 * 60 * 60
        ELSE 6 * 60 * 60
    END,
    MAX(CASE
        WHEN r.status='downloaded' AND m.status='rejected' THEN COALESCE(m.completed_at,r.started_at)
        ELSE r.started_at
    END),
    unixepoch()
FROM torrent_automation_runs r
LEFT JOIN torrent_magnet_verifications m ON m.run_id=r.id
WHERE r.status NOT IN ('running','downloaded')
   OR (r.status='downloaded' AND m.status='rejected')
GROUP BY r.episode_id;
