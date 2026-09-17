package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestVersionSevenUpgradeAddsPerProfileAuthentication(t *testing.T) {
	dir := t.TempDir()
	raw, err := sql.Open("sqlite", filepath.ToSlash(filepath.Join(dir, "app.db")))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = raw.Exec(schema + "\n" + migration002 + "\n" + migration003 + "\n" + migration004 + "\n" + migration005 + "\n" + migration006 + "\n" + migration007 + "\nPRAGMA user_version=7;"); err != nil {
		t.Fatal(err)
	}
	if _, err = raw.Exec("INSERT INTO profiles(id,display_name,avatar,created_at,locale) VALUES('password-profile','Password','violet',1,'en'),('open-profile','Open','mint',2,'en')"); err != nil {
		t.Fatal(err)
	}
	if _, err = raw.Exec("INSERT INTO local_credentials(profile_id,hash,must_change) VALUES('password-profile','hash',0)"); err != nil {
		t.Fatal(err)
	}
	if _, err = raw.Exec("INSERT INTO profile_identities(issuer,subject,profile_id) VALUES('https://unused.example','subject','open-profile')"); err != nil {
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

	var passwordMethod, openMethod string
	if err = db.QueryRow("SELECT auth_method FROM profiles WHERE id='password-profile'").Scan(&passwordMethod); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow("SELECT auth_method FROM profiles WHERE id='open-profile'").Scan(&openMethod); err != nil {
		t.Fatal(err)
	}
	if passwordMethod != "password" || openMethod != "none" {
		t.Fatalf("unexpected auth methods: password=%q open=%q", passwordMethod, openMethod)
	}
	var identities int
	if err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='profile_identities'").Scan(&identities); err != nil {
		t.Fatal(err)
	}
	if identities != 0 {
		t.Fatal("unused OIDC identity table survived schema 8")
	}
}
