package commands

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/gofrs/flock"

	"github.com/CarlFlo/tally/internal/backup"
	"github.com/CarlFlo/tally/internal/config"
	"github.com/CarlFlo/tally/internal/database"
)

const usage = "usage: tally [serve|healthcheck|backup|restore <archive>|verify-backup <archive>|reset-password <profile-id-or-name>|delete-backup <filename>]"

func Run(args []string) error {
	command := "serve"
	if len(args) > 0 {
		command, args = args[0], args[1:]
	}
	switch command {
	case "help", "-h", "--help":
		fmt.Println(usage)
		return nil
	case "healthcheck", "serve", "backup", "restore", "verify-backup", "reset-password", "delete-backup":
	default:
		return fmt.Errorf("%s", usage)
	}
	if err := validateCommandArgs(command, args); err != nil {
		return err
	}

	c, err := config.Load()
	if err != nil {
		return err
	}
	if command == "healthcheck" {
		return healthcheck(c)
	}
	if err = os.MkdirAll(c.DataDir, 0700); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer func() {
		signal.Stop(signals)
		cancel()
	}()
	go func() {
		select {
		case sig := <-signals:
			if command == "serve" {
				slog.Info("Shutdown requested", "signal", sig.String())
			}
			cancel()
		case <-ctx.Done():
		}
	}()

	if command == "verify-backup" {
		return verifyBackup(ctx, c, args)
	}

	var resetValue string
	if command == "reset-password" {
		resetValue, err = readResetPassword()
		if err != nil {
			return err
		}
		handled, operatorErr := callRunningOperator(ctx, c, operatorRequest{
			Action:     "reset-password",
			ProfileRef: args[0],
			Password:   resetValue,
		})
		if handled {
			if operatorErr == nil {
				printPasswordResetSuccess()
			}
			return operatorErr
		}
	}
	if command == "restore" {
		archive, pathErr := filepath.Abs(args[0])
		if pathErr != nil {
			return pathErr
		}
		handled, operatorErr := callRunningOperator(ctx, c, operatorRequest{Action: "restore", Archive: archive})
		if handled {
			if operatorErr == nil {
				fmt.Println("Backup restored live")
			}
			return operatorErr
		}
	}

	lock := flock.New(filepath.Join(c.DataDir, "app.lock"))
	locked, err := lock.TryLock()
	if err != nil {
		return err
	}
	if !locked {
		return fmt.Errorf("another Tally process is using this volume; use the running server for supported live operator commands or stop it before running this command")
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
	b := &backup.Service{DB: db, DataDir: c.DataDir, Path: filepath.Join(c.DataDir, "backups"), Timezone: c.Timezone}

	if command == "serve" {
		shutdownStarted, serveErr := serve(ctx, c, db, b)
		if shutdownStarted.IsZero() {
			shutdownStarted = time.Now()
		}
		slog.Info("Shutdown: finalizing database")
		checkpointCtx, checkpointDone := context.WithTimeout(context.Background(), 3*time.Second)
		if checkpointErr := db.Checkpoint(checkpointCtx, true); checkpointErr != nil {
			slog.Warn("Shutdown: database checkpoint did not finish", "error", checkpointErr)
		}
		checkpointDone()
		closeErr := db.Close()
		if closeErr != nil {
			slog.Error("Shutdown: database close failed", "error", closeErr)
		}
		slog.Info("Tally stopped", "duration_ms", time.Since(shutdownStarted).Milliseconds())
		if serveErr != nil {
			return serveErr
		}
		return closeErr
	}
	defer db.Close()

	switch command {
	case "reset-password":
		if err = resetPasswordValue(ctx, db, c, args[0], resetValue); err != nil {
			return err
		}
		printPasswordResetSuccess()
		return nil
	case "backup":
		return createBackup(ctx, b)
	default:
		return nil
	}
}

func validateCommandArgs(command string, args []string) error {
	switch command {
	case "restore":
		if len(args) != 1 {
			return fmt.Errorf("usage: tally restore <archive>")
		}
	case "verify-backup":
		if len(args) != 1 {
			return fmt.Errorf("usage: tally verify-backup <archive>")
		}
	case "reset-password":
		if len(args) != 1 {
			return fmt.Errorf("usage: tally reset-password <profile-id-or-name>")
		}
	case "delete-backup":
		if len(args) != 1 {
			return fmt.Errorf("usage: tally delete-backup <filename>")
		}
	default:
		if len(args) != 0 {
			return fmt.Errorf("%s does not accept arguments", command)
		}
	}
	return nil
}
