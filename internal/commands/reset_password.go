package commands

import (
	"context"
	"crypto/subtle"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/config"
	"github.com/CarlFlo/tally/internal/database"
	"golang.org/x/term"
)

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

func printProfileList(profiles []profileSummary) {
	if len(profiles) == 0 {
		fmt.Fprintln(os.Stdout, "No profiles found.")
		return
	}
	fmt.Fprintln(os.Stdout, "Profiles:")
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tROLE\tAUTHENTICATION\tPROFILE ID")
	for _, p := range profiles {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Name, p.Role, p.Authentication, p.ID)
	}
	_ = w.Flush()
	fmt.Fprintln(os.Stdout, "\nReset with: tally reset-password <profile-id-or-unique-name>")
}

func resetPasswordValue(ctx context.Context, db *database.Store, c config.Config, profileRef, password string) error {
	profileID, err := resolveProfile(ctx, db, profileRef)
	if err != nil {
		return err
	}
	return auth.New(db, c).ResetPassword(ctx, profileID, password)
}

func printPasswordResetSuccess() {
	fmt.Fprintln(os.Stderr, "Password reset. Existing sessions were revoked; the profile must replace this password after signing in.")
}
