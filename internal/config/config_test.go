package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultsAndValidation(t *testing.T) {
	t.Setenv("TZ", "UTC")
	c, e := Load()
	if e != nil {
		t.Fatal(e)
	}
	if c.MaxProfiles != 8 || c.AuthMode != "disabled" || c.SessionIdle != 30*24*time.Hour || c.JobRetries != 3 {
		t.Fatal("unexpected defaults")
	}
	for _, test := range []struct{ key, value string }{{"APP_MAX_PROFILES", "0"}, {"TZ", "Not/AZone"}, {"APP_AUTH_MODE", "invalid"}, {"OIDC_REDIRECT_URL", "https://example.com/wrong"}, {"APP_PUBLIC_URL", "https://example.com/subpath"}} {
		t.Run(test.key, func(t *testing.T) {
			t.Setenv(test.key, test.value)
			if _, e := Load(); e == nil {
				t.Fatal("invalid configuration accepted")
			}
		})
	}
}

func TestRemovedUIEnvironmentDoesNotBlockStartup(t *testing.T) {
	t.Setenv("JOB_METADATA_CRON", "invalid")
	t.Setenv("WEBHOOK_URL", "invalid")
	t.Setenv("TORZNAB_PROVIDERS", "invalid")
	t.Setenv("BACKUP_ENABLED", "invalid")
	t.Setenv("BACKUP_KEEP", "invalid")
	t.Setenv("BACKUP_PATH", "obsolete")
	t.Setenv("LOCAL_PASSWORD_ALLOW_NUMERIC_ONLY", "invalid")
	t.Setenv("TZ", "UTC")
	if _, e := Load(); e != nil {
		t.Fatal("removed UI settings still validated as infrastructure", e)
	}
}

func TestBackupLocationAndMetadataDefaults(t *testing.T) {
	t.Setenv("TZ", "UTC")
	t.Setenv("APP_DATA_DIR", t.TempDir())
	t.Setenv("BACKUP_PATH", "ignored-custom-directory")
	t.Setenv("BACKUP_KEEP", "100")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.BackupPath != filepath.Join(c.DataDir, "backups") || c.BackupKeep != 10 || c.MetadataCron != "0 * * * *" {
		t.Fatal("wrong application defaults")
	}
}
