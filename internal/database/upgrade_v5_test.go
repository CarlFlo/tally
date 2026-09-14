package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestVersionFourUpgradeChangesOnlyUntouchedMetadataDefault(t *testing.T) {
	for _, fixture := range []struct {
		name, schedule string
		revision       int
		want           string
	}{
		{"old default", "*/15 * * * *", 1, "0 * * * *"},
		{"edited default", "*/15 * * * *", 2, "*/15 * * * *"},
		{"custom schedule", "15 8 * * *", 1, "15 8 * * *"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			dir := t.TempDir()
			raw, err := sql.Open("sqlite", filepath.ToSlash(filepath.Join(dir, "app.db")))
			if err != nil {
				t.Fatal(err)
			}
			if _, err = raw.Exec(schema + "\n" + migration002 + "\n" + migration003 + "\n" + migration004 + "\nPRAGMA user_version=4;"); err != nil {
				t.Fatal(err)
			}
			if _, err = raw.Exec("INSERT INTO jobs(key,type,schedule,revision) VALUES('metadata','metadata',?,?)", fixture.schedule, fixture.revision); err != nil {
				t.Fatal(err)
			}
			raw.Close()
			db, err := Open(context.Background(), dir)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			var schedule string
			if err = db.QueryRow("SELECT schedule FROM jobs WHERE key='metadata'").Scan(&schedule); err != nil || schedule != fixture.want {
				t.Fatal("schedule not preserved/migrated", schedule, err)
			}
			var changes int
			if err = db.QueryRow("SELECT COUNT(*) FROM activity_log WHERE action='schedule_updated'").Scan(&changes); err != nil {
				t.Fatal(err)
			}
			if (changes == 1) != (fixture.name == "old default") {
				t.Fatal("incorrect migration activity")
			}
			if err = Validate(context.Background(), db.DB); err != nil {
				t.Fatal(err)
			}
			paths, _ := filepath.Glob(filepath.Join(dir, "pre-upgrade-v4-*.db"))
			if len(paths) != 1 {
				t.Fatal("missing pre-upgrade snapshot")
			}
			snapshot, err := sql.Open("sqlite", filepath.ToSlash(paths[0]))
			if err != nil {
				t.Fatal(err)
			}
			defer snapshot.Close()
			if err = Validate(context.Background(), snapshot); err != nil {
				t.Fatal(err)
			}
			var version int
			_ = snapshot.QueryRow("PRAGMA user_version").Scan(&version)
			if version != 4 {
				t.Fatal("snapshot was taken after upgrade")
			}
		})
	}
}
