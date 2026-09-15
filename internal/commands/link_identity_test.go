package commands

import (
	"context"
	"testing"

	"github.com/CarlFlo/tally/internal/database"
)

func TestLinkIdentityAcceptsUniqueProfileNameOrID(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec("INSERT INTO profiles(id,display_name,avatar,created_at) VALUES('profile-a','Fixture','violet',1),('profile-b','Second','mint',2)"); err != nil {
		t.Fatal(err)
	}

	byName := []string{"Fixture", "https://identity.example", "name-subject"}
	if err = linkIdentity(ctx, db, byName); err != nil {
		t.Fatal(err)
	}
	var profile string
	if err = db.QueryRow("SELECT profile_id FROM profile_identities WHERE issuer=? AND subject=?", byName[1], byName[2]).Scan(&profile); err != nil || profile != "profile-a" {
		t.Fatal("display name resolved to the wrong profile", profile, err)
	}

	byID := []string{"profile-b", "https://identity.example", "id-subject"}
	if err = linkIdentity(ctx, db, byID); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow("SELECT profile_id FROM profile_identities WHERE issuer=? AND subject=?", byID[1], byID[2]).Scan(&profile); err != nil || profile != "profile-b" {
		t.Fatal("profile ID resolved incorrectly", profile, err)
	}

	if _, err = db.Exec("INSERT INTO profiles(id,display_name,avatar,created_at) VALUES('profile-c','Fixture','blue',3)"); err != nil {
		t.Fatal(err)
	}
	if err = linkIdentity(ctx, db, []string{"Fixture", "https://identity.example", "ambiguous"}); err == nil {
		t.Fatal("ambiguous display name was accepted")
	}

	for _, invalid := range [][]string{
		nil,
		{"Fixture"},
		{"Missing", "https://identity.example", "subject"},
		{"profile-a", "file:///bad", "subject"},
		{"profile-a", "https://identity.example", ""},
	} {
		if err = linkIdentity(ctx, db, invalid); err == nil {
			t.Fatal("invalid command accepted", invalid)
		}
	}
	if err = linkIdentity(ctx, db, byID); err == nil {
		t.Fatal("existing identity mapping overwritten")
	}
}
