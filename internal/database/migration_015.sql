CREATE TABLE automation_flows (
 id TEXT PRIMARY KEY,
 name TEXT NOT NULL,
 revision INTEGER NOT NULL CHECK(revision > 0),
 created_at INTEGER NOT NULL,
 updated_at INTEGER NOT NULL
);
CREATE TABLE automation_flow_revisions (
 flow_id TEXT NOT NULL REFERENCES automation_flows(id) ON DELETE CASCADE,
 revision INTEGER NOT NULL CHECK(revision > 0),
 definition TEXT NOT NULL,
 created_at INTEGER NOT NULL,
 PRIMARY KEY(flow_id, revision)
);
CREATE TABLE automation_flow_runs (
 id TEXT PRIMARY KEY,
 flow_id TEXT NOT NULL REFERENCES automation_flows(id) ON DELETE CASCADE,
 flow_revision INTEGER NOT NULL,
 source_run_id TEXT REFERENCES torrent_automation_runs(id) ON DELETE SET NULL,
 trigger_event TEXT NOT NULL,
 definition TEXT NOT NULL,
 trace TEXT NOT NULL,
 status TEXT NOT NULL CHECK(status IN ('completed','no_candidate','stopped','failed')),
 started_at INTEGER NOT NULL,
 duration_ms INTEGER NOT NULL CHECK(duration_ms >= 0)
);
CREATE INDEX automation_flow_runs_started ON automation_flow_runs(started_at DESC);
