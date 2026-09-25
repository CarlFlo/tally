package backup

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/CarlFlo/tally/internal/database"
	"github.com/CarlFlo/tally/internal/flows"
)

func TestAdvancedFlowRevisionAndRunSurviveArchiveRestore(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, err := database.Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := flows.Store{DB: db}
	if err := store.EnsureDefaults(ctx); err != nil {
		t.Fatal(err)
	}
	live, err := store.Get(ctx, "default-live")
	if err != nil {
		t.Fatal(err)
	}
	live.Definition.Blocks[2].Config["min_seeders"] = "12"
	if _, err = store.Save(ctx, live, live.Revision); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO shows(id,name) VALUES('show-chain','Chain Show')"); err != nil {
		t.Fatal(err)
	}
	definition := flows.DefaultDefinition()
	flow, err := store.Save(ctx, flows.Flow{Name: "Archive fixture", Definition: definition}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Assign(ctx, "show-chain", flow.ID); err != nil {
		t.Fatal(err)
	}
	run := flows.Execute(ctx, flow, flows.Event{ShowName: "Example Show", Season: 1, Episode: 2}, nil)
	run, err = store.Record(ctx, run)
	if err != nil {
		t.Fatal(err)
	}
	service := Service{DB: db, DataDir: dir, Path: filepath.Join(dir, "backups"), Keep: 1}
	archive, err := service.Create(ctx, "manual")
	if err != nil {
		t.Fatal(err)
	}
	restoredDir := t.TempDir()
	if err = Restore(ctx, filepath.Join(service.Path, archive), restoredDir); err != nil {
		t.Fatal(err)
	}
	restored, err := database.Open(ctx, restoredDir)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	flowAfter, err := (flows.Store{DB: restored}).Get(ctx, flow.ID)
	if err != nil {
		t.Fatal(err)
	}
	liveAfter, err := (flows.Store{DB: restored}).Get(ctx, "default-live")
	if err != nil || liveAfter.Definition.Blocks[2].Config["min_seeders"] != "12" {
		t.Fatalf("default chain edit missing after restore: %+v %v", liveAfter, err)
	}
	assigned, err := (flows.Store{DB: restored}).Assigned(ctx, "show-chain")
	if err != nil || assigned != flow.ID {
		t.Fatalf("chain assignment missing after restore: %q %v", assigned, err)
	}
	runAfter, err := (flows.Store{DB: restored}).GetRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if flowAfter.Revision != 1 || runAfter.FlowRevision != 1 || len(runAfter.Steps) != 5 || runAfter.Event.ShowName != "Example Show" {
		t.Fatalf("advanced flow archive incomplete: %+v %+v", flowAfter, runAfter)
	}
}
