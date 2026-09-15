PRAGMA defer_foreign_keys=ON;

CREATE TEMP TABLE profile_id_migration (
 old_id TEXT PRIMARY KEY,
 new_id TEXT NOT NULL UNIQUE
);
INSERT INTO profile_id_migration(old_id,new_id)
 SELECT id,lower(hex(randomblob(16))) FROM profiles;

CREATE TABLE profile_id_aliases (
 alias TEXT PRIMARY KEY,
 profile_id TEXT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE
);
INSERT INTO profile_id_aliases(alias,profile_id)
 SELECT old_id,new_id FROM profile_id_migration;

UPDATE profile_preferences SET profile_id=(SELECT new_id FROM profile_id_migration WHERE old_id=profile_id);
UPDATE profile_identities SET profile_id=(SELECT new_id FROM profile_id_migration WHERE old_id=profile_id);
UPDATE local_credentials SET profile_id=(SELECT new_id FROM profile_id_migration WHERE old_id=profile_id);
UPDATE sessions SET profile_id=(SELECT new_id FROM profile_id_migration WHERE old_id=profile_id);
UPDATE profile_shows SET profile_id=(SELECT new_id FROM profile_id_migration WHERE old_id=profile_id);
UPDATE profile_episode_state SET profile_id=(SELECT new_id FROM profile_id_migration WHERE old_id=profile_id);
UPDATE torrent_search_history SET profile_id=(SELECT new_id FROM profile_id_migration WHERE old_id=profile_id);
UPDATE torrent_send_history SET profile_id=(SELECT new_id FROM profile_id_migration WHERE old_id=profile_id);
UPDATE show_actions SET profile_id=(SELECT new_id FROM profile_id_migration WHERE old_id=profile_id);
UPDATE inbox_state SET profile_id=(SELECT new_id FROM profile_id_migration WHERE old_id=profile_id);
UPDATE inbox_dismissals SET profile_id=(SELECT new_id FROM profile_id_migration WHERE old_id=profile_id);
UPDATE activity_log SET profile_id=COALESCE((SELECT new_id FROM profile_id_migration WHERE old_id=profile_id),profile_id);
UPDATE profiles SET id=(SELECT new_id FROM profile_id_migration WHERE old_id=id);

CREATE TABLE profile_roles (
 profile_id TEXT PRIMARY KEY REFERENCES profiles(id) ON DELETE CASCADE,
 is_admin INTEGER NOT NULL DEFAULT 0 CHECK(is_admin IN (0,1))
);
INSERT INTO profile_roles(profile_id,is_admin)
 SELECT id,CASE WHEN id=(SELECT profile_id FROM profile_id_aliases WHERE alias='user0') THEN 1 ELSE 0 END
 FROM profiles;
UPDATE profile_roles SET is_admin=1
 WHERE profile_id=(SELECT id FROM profiles ORDER BY created_at,id LIMIT 1)
 AND EXISTS(SELECT 1 FROM profiles)
 AND NOT EXISTS(SELECT 1 FROM profile_roles WHERE is_admin=1);

CREATE TRIGGER profile_role_after_insert
AFTER INSERT ON profiles
BEGIN
 INSERT INTO profile_roles(profile_id,is_admin)
 VALUES(NEW.id,CASE WHEN (SELECT COUNT(*) FROM profiles)=1 THEN 1 ELSE 0 END);
END;

CREATE TRIGGER profile_role_last_admin_update
BEFORE UPDATE OF is_admin ON profile_roles
WHEN OLD.is_admin=1 AND NEW.is_admin=0
 AND EXISTS(SELECT 1 FROM profiles WHERE id<>OLD.profile_id)
 AND NOT EXISTS(SELECT 1 FROM profile_roles WHERE profile_id<>OLD.profile_id AND is_admin=1)
BEGIN
 SELECT RAISE(ABORT,'at least one administrator is required while profiles remain');
END;

CREATE TRIGGER profile_role_last_admin_delete
BEFORE DELETE ON profiles
WHEN EXISTS(SELECT 1 FROM profile_roles WHERE profile_id=OLD.id AND is_admin=1)
 AND EXISTS(SELECT 1 FROM profiles WHERE id<>OLD.id)
 AND NOT EXISTS(SELECT 1 FROM profile_roles WHERE profile_id<>OLD.id AND is_admin=1)
BEGIN
 SELECT RAISE(ABORT,'at least one administrator is required while profiles remain');
END;

CREATE TRIGGER profile_role_direct_delete
BEFORE DELETE ON profile_roles
WHEN EXISTS(SELECT 1 FROM profiles WHERE id=OLD.profile_id)
BEGIN
 SELECT RAISE(ABORT,'profile roles are removed with their profile');
END;

DROP TABLE profile_id_migration;
