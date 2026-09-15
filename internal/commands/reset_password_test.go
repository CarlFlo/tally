package commands

import (
	"context"
	"testing"
	"time"

	"github.com/CarlFlo/mediaManager/internal/database"
)

func TestResetPasswordAcceptsUniqueProfileName(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec("INSERT INTO profiles(id,display_name,avatar,created_at) VALUES('profile-a','Fixture','violet',1)"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("INSERT INTO sessions(id,profile_id,created_at,last_seen,expires_at,restricted,user_agent) VALUES('session','profile-a',?,?,?,?,0,'test')", time.Now().Unix(), time.Now().Unix(), time.Now().Add(time.Hour).Unix()); err != nil {
		t.Fatal(err)
	}
	if err = resetPassword(ctx, db, []string{"Fixture"}); err != nil {
		t.Fatal(err)
	}
	var mustChange, sessions int
	if err = db.QueryRow("SELECT must_change FROM local_credentials WHERE profile_id='profile-a'").Scan(&mustChange); err != nil || mustChange != 1 {
		t.Fatal("temporary credential was not created", mustChange, err)
	}
	if err = db.QueryRow("SELECT COUNT(*) FROM sessions WHERE profile_id='profile-a'").Scan(&sessions); err != nil || sessions != 0 {
		t.Fatal("existing sessions were not revoked", sessions, err)
	}
}

func TestProfileNameResolutionRejectsAmbiguity(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec("INSERT INTO profiles(id,display_name,avatar,created_at) VALUES('profile-a','Same name','violet',1),('profile-b','Same name','mint',2)"); err != nil {
		t.Fatal(err)
	}
	if err = resetPassword(ctx, db, []string{"Same name"}); err == nil {
		t.Fatal("ambiguous display name was accepted")
	}
}
