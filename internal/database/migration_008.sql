ALTER TABLE profiles ADD COLUMN auth_method TEXT NOT NULL DEFAULT 'none' CHECK(auth_method IN ('password','none'));

UPDATE profiles
SET auth_method='password'
WHERE EXISTS (
  SELECT 1 FROM local_credentials c WHERE c.profile_id=profiles.id
);

DROP TABLE IF EXISTS profile_identities;
