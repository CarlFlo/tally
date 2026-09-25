ALTER TABLE automation_flows ADD COLUMN show_id TEXT REFERENCES shows(id) ON DELETE CASCADE;
CREATE INDEX automation_flows_show ON automation_flows(show_id);
CREATE TABLE torrent_show_chain (
 show_id TEXT PRIMARY KEY REFERENCES shows(id) ON DELETE CASCADE,
 flow_id TEXT NOT NULL REFERENCES automation_flows(id) ON DELETE CASCADE
);
