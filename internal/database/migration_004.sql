CREATE TABLE activity_log (
 id INTEGER PRIMARY KEY AUTOINCREMENT, action TEXT NOT NULL,
 profile_id TEXT NOT NULL DEFAULT '', show_id TEXT NOT NULL DEFAULT '',
 show_name TEXT NOT NULL DEFAULT '', message TEXT NOT NULL,
 created_at INTEGER NOT NULL
);
CREATE INDEX activity_action_time ON activity_log(action,created_at);
CREATE TABLE browser_preferences (id TEXT PRIMARY KEY, theme TEXT NOT NULL CHECK(theme IN ('light','dark','system')), updated_at INTEGER NOT NULL);
CREATE TABLE notification_state (key TEXT PRIMARY KEY, value INTEGER NOT NULL);
CREATE TABLE notification_outbox (
 id INTEGER PRIMARY KEY AUTOINCREMENT, event_key TEXT NOT NULL UNIQUE,
 event TEXT NOT NULL, message TEXT NOT NULL, show_name TEXT NOT NULL DEFAULT '',
 occurred_at INTEGER NOT NULL, available_at INTEGER NOT NULL,
 source_key TEXT NOT NULL DEFAULT '', level TEXT NOT NULL DEFAULT 'info',
 status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','sent','failed','skipped'))
);
CREATE INDEX notification_due ON notification_outbox(status,available_at);
INSERT INTO notification_state VALUES ('activity_cursor',0),('release_cursor',unixepoch());

CREATE INDEX episode_release_time ON episodes(unixepoch(airstamp));
