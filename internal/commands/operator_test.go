package commands

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/backup"
	"github.com/CarlFlo/tally/internal/config"
	"github.com/CarlFlo/tally/internal/database"
)

func TestOperatorServerResetsPasswordWhileApplicationIsRunning(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dir := t.TempDir()
	c := config.Config{DataDir: dir, Timezone: "UTC", PasswordMin: 4, PasswordMax: 128}
	db, err := database.Open(ctx, dir)
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
	backups := &backup.Service{DB: db, DataDir: dir, Path: filepath.Join(dir, "backups"), Timezone: "UTC"}
	server, err := startOperatorServer(ctx, c, db, backups, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	handled, err := callRunningOperator(ctx, c, operatorRequest{Action: "reset-password", ProfileRef: "Fixture", Password: "temporary-pass"})
	if err != nil {
		t.Fatal(err)
	}
	if !handled {
		t.Fatal("running operator server was not detected")
	}
	var hash string
	var mustChange, sessions int
	if err = db.QueryRow("SELECT hash,must_change FROM local_credentials WHERE profile_id='profile-a'").Scan(&hash, &mustChange); err != nil {
		t.Fatal(err)
	}
	if mustChange != 1 || !auth.Verify(hash, "temporary-pass") {
		t.Fatal("temporary credential was not applied")
	}
	if err = db.QueryRow("SELECT COUNT(*) FROM sessions WHERE profile_id='profile-a'").Scan(&sessions); err != nil || sessions != 0 {
		t.Fatal("existing sessions were not revoked", sessions, err)
	}
}

func TestOperatorServerRestoresBackupLive(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dir := t.TempDir()
	c := config.Config{DataDir: dir, Timezone: "UTC", PasswordMin: 4, PasswordMax: 128}
	db, err := database.Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	backups := &backup.Service{DB: db, DataDir: dir, Path: filepath.Join(dir, "backups"), Timezone: "UTC"}
	if _, err = db.Exec("INSERT INTO profiles(id,display_name,avatar,created_at) VALUES('profile-a','Before backup','violet',1)"); err != nil {
		t.Fatal(err)
	}
	filename, err := backups.Create(ctx, "manual")
	if err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(backups.Path, filename)
	if _, err = db.Exec("UPDATE profiles SET display_name='Changed later' WHERE id='profile-a'"); err != nil {
		t.Fatal(err)
	}

	server, err := startOperatorServer(ctx, c, db, backups, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	handled, err := callRunningOperator(ctx, c, operatorRequest{Action: "restore", Archive: archive})
	if err != nil {
		t.Fatal(err)
	}
	if !handled {
		t.Fatal("running operator server was not detected")
	}
	var name string
	if err = db.QueryRow("SELECT display_name FROM profiles WHERE id='profile-a'").Scan(&name); err != nil {
		t.Fatal(err)
	}
	if name != "Before backup" {
		t.Fatalf("live restore did not replace state: %q", name)
	}
}

func TestOperatorClientFallsBackWhenServerIsNotRunning(t *testing.T) {
	c := config.Config{DataDir: t.TempDir()}
	handled, err := callRunningOperator(context.Background(), c, operatorRequest{Action: "restore", Archive: "missing.zip"})
	if err != nil {
		t.Fatal(err)
	}
	if handled {
		t.Fatal("missing operator socket should allow offline fallback")
	}
}

func TestCommandArgumentsAreValidatedBeforeRuntimeWork(t *testing.T) {
	for _, test := range []struct {
		command string
		args    []string
	}{
		{"reset-password", nil},
		{"restore", nil},
		{"verify-backup", nil},
		{"delete-backup", nil},
		{"healthcheck", []string{"unexpected"}},
	} {
		if err := validateCommandArgs(test.command, test.args); err == nil {
			t.Fatalf("%s accepted invalid arguments", test.command)
		}
	}
}
