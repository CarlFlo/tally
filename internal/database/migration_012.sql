ALTER TABLE episodes ADD COLUMN downloaded INTEGER NOT NULL DEFAULT 0 CHECK(downloaded IN (0,1));

UPDATE episodes
SET downloaded=1
WHERE EXISTS (
    SELECT 1
    FROM profile_episode_state p
    WHERE p.episode_id=episodes.id AND p.downloaded=1
);
