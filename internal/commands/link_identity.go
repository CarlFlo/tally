package commands

import (
	"context"
	"fmt"

	"github.com/CarlFlo/tally/internal/config"
	"github.com/CarlFlo/tally/internal/database"
)

func linkIdentity(ctx context.Context, db *database.Store, args []string) error {
	var e error
	if len(args) != 3 {
		return fmt.Errorf("usage: tally link-identity <profile-id-or-name> <issuer> <subject>")
	}
	if e = config.ValidateURL(args[1]); e != nil {
		return e
	}
	if args[2] == "" {
		return fmt.Errorf("subject cannot be empty")
	}
	profileID, err := resolveProfile(ctx, db, args[0])
	if err != nil {
		return err
	}
	_, e = db.ExecContext(ctx, "INSERT INTO profile_identities(issuer,subject,profile_id) VALUES(?,?,?)", args[1], args[2], profileID)
	if e == nil {
		fmt.Println("Identity linked to profile")
	}
	return e
}
