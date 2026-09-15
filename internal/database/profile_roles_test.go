package database

import (
	"context"
	"testing"
)

func TestProfileRoleTriggersProtectLastAdministrator(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err = db.Exec("INSERT INTO profiles(id,display_name,avatar,created_at) VALUES('profile-admin','Admin','violet',1)"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("UPDATE profile_roles SET is_admin=0 WHERE profile_id='profile-admin'"); err == nil {
		t.Fatal("sole administrator was demoted directly")
	}

	if _, err = db.Exec("INSERT INTO profiles(id,display_name,avatar,created_at) VALUES('profile-member','Member','mint',2)"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("DELETE FROM profiles WHERE id='profile-admin'"); err == nil {
		t.Fatal("last administrator was deleted while another profile remained")
	}

	if _, err = db.Exec("UPDATE profile_roles SET is_admin=1 WHERE profile_id='profile-member'"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("UPDATE profile_roles SET is_admin=0 WHERE profile_id='profile-admin'"); err != nil {
		t.Fatal("administrator transfer was blocked", err)
	}
}

func TestOpenRejectsInconsistentProfileRoles(t *testing.T) {
	ctx := context.Background()
	for _, test := range []struct {
		name   string
		damage string
	}{
		{"missing role", "DELETE FROM profile_roles WHERE profile_id='profile-admin'"},
		{"no administrator", "DROP TRIGGER profile_role_last_admin_update; UPDATE profile_roles SET is_admin=0"},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			db, err := Open(ctx, dir)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = db.Exec("INSERT INTO profiles(id,display_name,avatar,created_at) VALUES('profile-admin','Admin','violet',1)"); err != nil {
				t.Fatal(err)
			}
			if _, err = db.Exec(test.damage); err != nil {
				t.Fatal(err)
			}
			if err = db.Close(); err != nil {
				t.Fatal(err)
			}
			if reopened, err := Open(ctx, dir); err == nil {
				reopened.Close()
				t.Fatal("inconsistent role state reopened successfully")
			}
		})
	}
}
