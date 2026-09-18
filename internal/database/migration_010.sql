ALTER TABLE shows ADD COLUMN show_type TEXT NOT NULL DEFAULT '';

CREATE TABLE torrent_show_media_profile (
    show_id TEXT PRIMARY KEY REFERENCES shows(id) ON DELETE CASCADE,
    profile TEXT NOT NULL CHECK(profile IN ('live','animated')),
    updated_at INTEGER NOT NULL
);
