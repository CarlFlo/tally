package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestVersionSixUpgradeAddsProfileLocale(t *testing.T) {
	dir := t.TempDir()
	raw, err := sql.Open("sqlite", filepath.ToSlash(filepath.Join(dir, "app.db")))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = raw.Exec(schema + "\n" + migration002 + "\n" + migration003 + "\n" + migration004 + "\n" + migration005 + "\n" + migration006 + "\nPRAGMA user_version=6;"); err != nil {
		t.Fatal(err)
	}
	if _, err = raw.Exec("INSERT INTO profiles(id,display_name,avatar,created_at) VALUES('profile','Profile','violet',1)"); err != nil {
		t.Fatal(err)
	}
	if err = raw.Close(); err != nil {
		t.Fatal(err)
	}

	db, err := Open(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var locale string
	if err = db.QueryRow("SELECT locale FROM profiles WHERE id='profile'").Scan(&locale); err != nil {
		t.Fatal(err)
	}
	if locale != "en" {
		t.Fatalf("locale=%q, want en", locale)
	}
}
