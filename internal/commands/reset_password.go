package commands

import (
	"context"
	"crypto/subtle"
	"fmt"
	"os"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/config"
	"github.com/CarlFlo/tally/internal/database"
	"golang.org/x/term"
)

func resetPassword(ctx context.Context, db *database.Store, c config.Config, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: tally reset-password <profile-id-or-name>")
	}
	password, err := readResetPassword()
	if err != nil {
		return err
	}
	return resetPasswordValue(ctx, db, c, args[0], password)
}

func readResetPassword() (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return "", fmt.Errorf("reset-password requires an interactive terminal")
	}
	fmt.Fprint(os.Stderr, "Temporary password: ")
	first, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	defer clear(first)
	fmt.Fprint(os.Stderr, "Confirm temporary password: ")
	second, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	defer clear(second)
	if subtle.ConstantTimeCompare(first, second) != 1 {
		return "", fmt.Errorf("passwords do not match")
	}
	if len(first) == 0 {
		return "", fmt.Errorf("password cannot be empty")
	}
	return string(first), nil
}

func resetPasswordValue(ctx context.Context, db *database.Store, c config.Config, profileRef, password string) error {
	profileID, err := resolveProfile(ctx, db, profileRef)
	if err != nil {
		return err
	}
	authService := auth.New(db, c)
	if err = authService.Policy(password); err != nil {
		return err
	}
	hash, err := authService.HashPassword(ctx, password)
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "UPDATE profiles SET auth_method='password' WHERE id=?", profileID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO local_credentials VALUES(?,?,1) ON CONFLICT(profile_id) DO UPDATE SET hash=excluded.hash,must_change=1", profileID, hash); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM sessions WHERE profile_id=?", profileID); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "Password reset. Existing sessions were revoked; the profile must replace this password after signing in.")
	return nil
}
