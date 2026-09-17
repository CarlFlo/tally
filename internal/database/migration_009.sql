CREATE TABLE torrent_automation_runs (
    id TEXT PRIMARY KEY,
    show_id TEXT NOT NULL,
    episode_id TEXT NOT NULL,
    show_name TEXT NOT NULL,
    season INTEGER NOT NULL,
    episode INTEGER NOT NULL,
    query TEXT NOT NULL,
    status TEXT NOT NULL CHECK(status IN ('running','downloaded','no_verified_candidate','rejected','failed','skipped','cancelled')),
    confidence TEXT NOT NULL DEFAULT '' CHECK(confidence IN ('','high','medium','low','rejected')),
    verification TEXT NOT NULL DEFAULT '' CHECK(verification IN ('','verified','unverified')),
    selected_name TEXT NOT NULL DEFAULT '',
    selected_infohash TEXT NOT NULL DEFAULT '',
    settings_snapshot TEXT NOT NULL DEFAULT '{}',
    decision_log TEXT NOT NULL DEFAULT '[]',
    engine_version TEXT NOT NULL,
    started_at INTEGER NOT NULL,
    ended_at INTEGER,
    duration_ms INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX torrent_automation_runs_created ON torrent_automation_runs(started_at DESC);
CREATE INDEX torrent_automation_runs_episode ON torrent_automation_runs(episode_id,started_at DESC);
CREATE UNIQUE INDEX torrent_automation_one_running_episode ON torrent_automation_runs(episode_id) WHERE status='running';
CREATE UNIQUE INDEX torrent_automation_one_downloaded_episode ON torrent_automation_runs(episode_id) WHERE status='downloaded';

CREATE TABLE torrent_automation_feedback (
    run_id TEXT PRIMARY KEY REFERENCES torrent_automation_runs(id) ON DELETE CASCADE,
    profile_id TEXT REFERENCES profiles(id) ON DELETE SET NULL,
    reason TEXT NOT NULL CHECK(reason IN ('wrong_show','wrong_episode','wrong_language','poor_quality','corrupt','suspicious_files','other')),
    note TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL
);

CREATE TABLE torrent_bad_hashes (
    infohash TEXT PRIMARY KEY,
    reason TEXT NOT NULL,
    source_run_id TEXT REFERENCES torrent_automation_runs(id) ON DELETE SET NULL,
    marked_by TEXT REFERENCES profiles(id) ON DELETE SET NULL,
    created_at INTEGER NOT NULL
);

CREATE TABLE torrent_show_policy (
    show_id TEXT PRIMARY KEY REFERENCES shows(id) ON DELETE CASCADE,
    policy TEXT NOT NULL CHECK(policy IN ('default','auto','never')),
    updated_at INTEGER NOT NULL
);
