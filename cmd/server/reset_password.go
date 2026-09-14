package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/CarlFlo/mediaManager/internal/auth"
	"github.com/CarlFlo/mediaManager/internal/database"
)

func resetPassword(ctx context.Context, db *database.Store, args []string) error {
	var e error
	if len(args) != 1 {
		return fmt.Errorf("usage: tally reset-password <profile>")
	}
	var exists int
	if e = db.QueryRowContext(ctx, "SELECT 1 FROM profiles WHERE id=?", args[0]).Scan(&exists); e != nil {
		return fmt.Errorf("profile does not exist")
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
	if _, err = tx.ExecContext(ctx, "INSERT INTO local_credentials VALUES(?,?,1) ON CONFLICT(profile_id) DO UPDATE SET hash=excluded.hash,must_change=1", args[0], hash); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM sessions WHERE profile_id=?", args[0]); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	slog.Warn("OPERATOR PASSWORD RESET — replace immediately", "profile_id", args[0], "temporary_password", temporary)
	return nil
}
