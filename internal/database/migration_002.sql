CREATE TABLE download_client_settings (
    id INTEGER PRIMARY KEY CHECK(id=1),
    adapter TEXT NOT NULL DEFAULT '',
    fields TEXT NOT NULL DEFAULT '{}',
    revision INTEGER NOT NULL DEFAULT 0,
    updated_at INTEGER NOT NULL DEFAULT 0
);
INSERT INTO download_client_settings(id) VALUES(1);
