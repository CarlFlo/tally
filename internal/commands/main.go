package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	_ "time/tzdata"

	"github.com/gofrs/flock"

	"github.com/CarlFlo/tally/internal/backup"
	"github.com/CarlFlo/tally/internal/config"
	"github.com/CarlFlo/tally/internal/database"
)

func Run(args []string) error {
	c, err := config.Load()
	if err != nil {
		return err
	}
	command := "serve"
	if len(args) > 0 {
		command, args = args[0], args[1:]
	}
	switch command {
	case "healthcheck":
		return healthcheck(c)
	case "serve", "backup", "restore", "verify-backup", "link-identity", "reset-password", "delete-backup":
	default:
		return fmt.Errorf("usage: tally [serve|healthcheck|backup|restore <archive>|verify-backup <archive>|link-identity <profile-id-or-name> <issuer> <subject>|reset-password <profile-id-or-name>|delete-backup <filename>]")
	}
	if err = os.MkdirAll(c.DataDir, 0700); err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if command == "verify-backup" {
		return verifyBackup(ctx, c, args)
	}
	lock := flock.New(filepath.Join(c.DataDir, "app.lock"))
	locked, err := lock.TryLock()
	if err != nil {
		return err
	}
	if !locked {
		return fmt.Errorf("another Tally process is using this volume; stop it before running operator commands")
	}
	defer lock.Unlock()
	if command == "restore" {
		return restoreBackup(ctx, c, args)
	}
	if command == "delete-backup" {
		b := &backup.Service{DataDir: c.DataDir, Path: filepath.Join(c.DataDir, "backups"), Timezone: c.Timezone}
		return deleteBackup(ctx, b, args)
	}
	db, err := database.Open(ctx, c.DataDir)
	if err != nil {
		return err
	}
	defer db.Close()
	b := &backup.Service{DB: db, DataDir: c.DataDir, Path: filepath.Join(c.DataDir, "backups"), Timezone: c.Timezone}
	switch command {
	case "link-identity":
		return linkIdentity(ctx, db, args)
	case "reset-password":
		return resetPassword(ctx, db, args)
	case "backup":
		return createBackup(ctx, b)
	default:
		return serve(ctx, c, db, b)
	}
}
