ALTER TABLE profiles ADD COLUMN auth_method TEXT NOT NULL DEFAULT 'none' CHECK(auth_method IN ('password','none','oidc_unlinked'));

UPDATE profiles
SET auth_method='password'
WHERE EXISTS (
  SELECT 1 FROM local_credentials c WHERE c.profile_id=profiles.id
);

UPDATE profiles
SET auth_method='oidc_unlinked'
WHERE auth_method='none'
  AND EXISTS (
    SELECT 1 FROM profile_identities i WHERE i.profile_id=profiles.id
  );
