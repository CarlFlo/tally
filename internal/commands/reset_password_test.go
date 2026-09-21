package commands

import (
	"context"
	"testing"
	"time"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/config"
	"github.com/CarlFlo/tally/internal/database"
)

func resetConfig() config.Config {
	return config.Config{PasswordMin: 4, PasswordMax: 128}
}

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
	if _, err = db.Exec("INSERT INTO sessions(id,profile_id,created_at,last_seen,expires_at,restricted,user_agent) VALUES('session','profile-a',?,?,?,0,'test')", time.Now().Unix(), time.Now().Unix(), time.Now().Add(time.Hour).Unix()); err != nil {
		t.Fatal(err)
	}
	if err = resetPasswordValue(ctx, db, resetConfig(), "Fixture", "temporary-pass"); err != nil {
		t.Fatal(err)
	}
	var hash string
	var mustChange, sessions int
	if err = db.QueryRow("SELECT hash,must_change FROM local_credentials WHERE profile_id='profile-a'").Scan(&hash, &mustChange); err != nil || mustChange != 1 {
		t.Fatal("temporary credential was not created", mustChange, err)
	}
	if !auth.Verify(hash, "temporary-pass") {
		t.Fatal("operator-supplied temporary password was not stored")
	}
	if err = db.QueryRow("SELECT COUNT(*) FROM sessions WHERE profile_id='profile-a'").Scan(&sessions); err != nil || sessions != 0 {
		t.Fatal("existing sessions were not revoked", sessions, err)
	}
}

func TestResetPasswordEnforcesPolicyAndRejectsAmbiguity(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec("INSERT INTO profiles(id,display_name,avatar,created_at) VALUES('profile-a','Same name','violet',1),('profile-b','Same name','mint',2)"); err != nil {
		t.Fatal(err)
	}
	if err = resetPasswordValue(ctx, db, resetConfig(), "Same name", "temporary-pass"); err == nil {
		t.Fatal("ambiguous display name was accepted")
	}
	if err = resetPasswordValue(ctx, db, resetConfig(), "profile-a", "x"); err == nil {
		t.Fatal("password policy was bypassed")
	}
}
