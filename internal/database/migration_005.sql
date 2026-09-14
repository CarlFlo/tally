CREATE TABLE inbox_state (
 profile_id TEXT PRIMARY KEY REFERENCES profiles(id) ON DELETE CASCADE,
 seen_id INTEGER NOT NULL DEFAULT 0, cleared_id INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE inbox_dismissals (
 profile_id TEXT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
 activity_id INTEGER NOT NULL REFERENCES activity_log(id) ON DELETE CASCADE,
 PRIMARY KEY(profile_id,activity_id)
);
INSERT INTO activity_log(action,profile_id,message,created_at)
 SELECT 'schedule_updated','user0','Changed the default metadata schedule to hourly',unixepoch()
 FROM jobs WHERE key='metadata' AND schedule='*/15 * * * *' AND revision=1;
UPDATE jobs SET schedule='0 * * * *',revision=revision+1,
 next_run=CASE WHEN enabled=1 AND paused=0 THEN (unixepoch()/3600+1)*3600 ELSE 0 END
 WHERE key='metadata' AND schedule='*/15 * * * *' AND revision=1;
