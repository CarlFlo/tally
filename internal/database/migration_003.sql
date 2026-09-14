CREATE TABLE application_settings (key TEXT PRIMARY KEY, data TEXT NOT NULL, revision INTEGER NOT NULL DEFAULT 1);
ALTER TABLE profile_shows ADD COLUMN favorite INTEGER NOT NULL DEFAULT 0 CHECK(favorite IN (0,1));
ALTER TABLE jobs ADD COLUMN enabled INTEGER NOT NULL DEFAULT 1 CHECK(enabled IN (0,1));
ALTER TABLE jobs ADD COLUMN paused INTEGER NOT NULL DEFAULT 0 CHECK(paused IN (0,1));
ALTER TABLE jobs ADD COLUMN failures INTEGER NOT NULL DEFAULT 0;
ALTER TABLE jobs ADD COLUMN revision INTEGER NOT NULL DEFAULT 1;
CREATE TABLE show_actions (
 profile_id TEXT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
 external_id INTEGER NOT NULL CHECK(external_id>0),
 name TEXT NOT NULL DEFAULT '', desired INTEGER NOT NULL CHECK(desired IN (0,1)),
 status TEXT NOT NULL CHECK(status IN ('queued','running','done','failed')),
 revision INTEGER NOT NULL DEFAULT 1, error TEXT NOT NULL DEFAULT '',
 show_id TEXT NOT NULL DEFAULT '', updated_at INTEGER NOT NULL,
 PRIMARY KEY(profile_id,external_id)
);
CREATE INDEX show_actions_queue ON show_actions(status,updated_at);
