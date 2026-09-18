ALTER TABLE episodes ADD COLUMN downloaded INTEGER NOT NULL DEFAULT 0 CHECK(downloaded IN (0,1));

UPDATE episodes
SET downloaded=1
WHERE EXISTS (
    SELECT 1
    FROM profile_episode_state p
    WHERE p.episode_id=episodes.id AND p.downloaded=1
);

-- Downloaded is deployment-global from schema 12 onward. Keep the legacy
-- column only for schema compatibility, but remove stale per-profile truth.
UPDATE profile_episode_state SET downloaded=0;
