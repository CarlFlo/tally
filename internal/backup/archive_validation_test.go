package backup

import (
	"archive/zip"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/CarlFlo/mediaManager/internal/database"
)

func TestRejectsTraversalAndCorruptedArchives(t *testing.T) {
	for _, name := range []string{"../escape", "avatars/../../escape", "avatars\\escape.png", "app.db"} {
		t.Run(name, func(t *testing.T) {
			archive := filepath.Join(t.TempDir(), "invalid.zip")
			f, _ := os.Create(archive)
			z := zip.NewWriter(f)
			w, _ := z.Create(name)
			w.Write([]byte("not a database"))
			z.Close()
			f.Close()
			if _, e := Extract(context.Background(), archive, t.TempDir()); e == nil {
				t.Fatal("invalid archive accepted")
			}
		})
	}
}

func TestRestoreValidationPreservesExistingData(t *testing.T) {
	dir := t.TempDir()
	db, e := database.Open(context.Background(), dir)
	if e != nil {
		t.Fatal(e)
	}
	_, _ = db.Exec("INSERT INTO profiles(id,display_name,avatar,created_at) VALUES('user0','Keep me','violet',1)")
	db.Close()
	archive := filepath.Join(t.TempDir(), "invalid.zip")
	_ = os.WriteFile(archive, []byte("bad"), 0600)
	if e = Restore(context.Background(), archive, dir); e == nil {
		t.Fatal("invalid restore accepted")
	}
	raw, e := sql.Open("sqlite", filepath.ToSlash(filepath.Join(dir, "app.db")))
	if e != nil {
		t.Fatal(e)
	}
	defer raw.Close()
	var name string
	_ = raw.QueryRow("SELECT display_name FROM profiles WHERE id='user0'").Scan(&name)
	if name != "Keep me" {
		t.Fatal("failed restore changed original database")
	}
}
