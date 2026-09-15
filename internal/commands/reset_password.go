package commands

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/database"
)

func resetPassword(ctx context.Context, db *database.Store, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: tally reset-password <profile-id-or-name>")
	}
	profileID, err := resolveProfile(ctx, db, args[0])
	if err != nil {
		return err
	}
	temporary := auth.Token()
	hash, err := auth.Hash(temporary)
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "INSERT INTO local_credentials VALUES(?,?,1) ON CONFLICT(profile_id) DO UPDATE SET hash=excluded.hash,must_change=1", profileID, hash); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM sessions WHERE profile_id=?", profileID); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	slog.Warn("OPERATOR PASSWORD RESET — replace immediately", "profile_id", profileID, "temporary_password", temporary)
	return nil
}
