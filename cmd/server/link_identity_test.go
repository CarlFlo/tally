package main

import (
	"context"
	"testing"

	"github.com/CarlFlo/mediaManager/internal/database"
)

func TestLinkIdentityCommandArguments(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	args := []string{"user0", "https://identity.example", "immutable-subject"}
	if err = linkIdentity(ctx, db, args); err != nil {
		t.Fatal(err)
	}
	var profile string
	if err = db.QueryRow("SELECT profile_id FROM profile_identities WHERE issuer=? AND subject=?", args[1], args[2]).Scan(&profile); err != nil || profile != args[0] {
		t.Fatal("command mapped arguments to the wrong identity", err)
	}
	for _, invalid := range [][]string{nil, {"user0"}, {"user0", "file:///bad", "subject"}, {"user0", args[1], ""}} {
		if err = linkIdentity(ctx, db, invalid); err == nil {
			t.Fatal("invalid command accepted")
		}
	}
	if err = linkIdentity(ctx, db, args); err == nil {
		t.Fatal("existing mapping overwritten")
	}
}
