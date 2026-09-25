package flows

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/CarlFlo/tally/internal/database"
	"github.com/CarlFlo/tally/internal/settings"
	"github.com/CarlFlo/tally/internal/torrent"
)

func TestChainValidationAndDryRun(t *testing.T) {
	d := DefaultDefinition()
	if err := Validate("Preset", d); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*Definition){
		func(d *Definition) { d.Blocks[1].Type = "action.download" },
		func(d *Definition) { d.Blocks[2].Config["min_seeders"] = "-1" },
		func(d *Definition) { d.Blocks[1].Config["url"] = "https://secret" },
		func(d *Definition) { d.Blocks = d.Blocks[:4] },
	} {
		invalid := DefaultDefinition()
		change(&invalid)
		if Validate("Preset", invalid) == nil {
			t.Fatalf("accepted invalid chain: %+v", invalid)
		}
	}
	d.Blocks[0].Config["prefix"] = "WEB-DL"
	d.Blocks[0].Config["title_override"] = "Alternate Show Title"
	d.Blocks[2].Config["min_seeders"] = "5"
	flow := Flow{ID: "flow", Name: "Preset", Revision: 1, Definition: d}
	called := 0
	run := Execute(context.Background(), flow, Event{ShowName: "Example Show", Season: 1, Episode: 2}, func(_ context.Context, query string) ([]torrent.SearchResult, error) {
		called++
		if query != "WEB-DL Alternate Show Title S01E02" {
			t.Fatal(query)
		}
		return []torrent.SearchResult{{Name: "Alternate.Show.Title.S01E02.1080p.WEB-DL-GROUP", Seeders: 8, Magnet: "magnet:?xt=urn:btih:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}, nil
	})
	if called != 1 || run.Status != "completed" || len(run.Steps) != 5 || !strings.Contains(run.Steps[4].Summary, "Would inspect") {
		t.Fatalf("bad dry run: %+v", run)
	}
	raw, _ := json.Marshal(run)
	if strings.Contains(string(raw), "magnet:?") {
		t.Fatal("transport leaked")
	}
	empty := Execute(context.Background(), flow, Event{ShowName: "Example Show", Season: 1, Episode: 2}, func(context.Context, string) ([]torrent.SearchResult, error) { return nil, nil })
	if empty.Status != "no_candidate" || empty.Steps[4].Status != "skipped" {
		t.Fatalf("bad empty result: %+v", empty)
	}
}

func TestChainPersistenceAssignmentAndSnapshot(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec("INSERT INTO shows(id,name) VALUES('one','One'),('two','Two')"); err != nil {
		t.Fatal(err)
	}
	store := Store{DB: db}
	flow, err := store.Save(ctx, Flow{Name: "General", Definition: DefaultDefinition()}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Assign(ctx, "one", flow.ID); err != nil {
		t.Fatal(err)
	}
	if id, err := store.Assigned(ctx, "one"); err != nil || id != flow.ID {
		t.Fatalf("assignment: %q %v", id, err)
	}
	run := Execute(ctx, flow, Event{ShowName: "One", Season: 1, Episode: 1}, nil)
	run, err = store.Record(ctx, run)
	if err != nil {
		t.Fatal(err)
	}
	flow.Definition.Blocks[0].Config["prefix"] = "New"
	updated, err := store.Save(ctx, flow, flow.Revision)
	if err != nil || updated.Revision != 2 {
		t.Fatalf("update: %+v %v", updated, err)
	}
	if _, err = store.Save(ctx, flow, flow.Revision); err == nil {
		t.Fatal("accepted stale revision")
	}
	saved, err := store.GetRun(ctx, run.ID)
	if err != nil || saved.Definition.Blocks[0].Config["prefix"] != "" {
		t.Fatalf("snapshot changed: %+v %v", saved, err)
	}
	custom, err := store.Save(ctx, Flow{Name: "One custom", ShowID: "one", Definition: DefaultDefinition()}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Assign(ctx, "two", custom.ID); err == nil {
		t.Fatal("assigned show-specific chain to another show")
	}
	if err = store.Assign(ctx, "one", custom.ID); err != nil {
		t.Fatal(err)
	}
	if err = store.Delete(ctx, custom.ID); err != nil {
		t.Fatal(err)
	}
	if id, err := store.Assigned(ctx, "one"); err != nil || id != "" {
		t.Fatalf("deleted chain still assigned: %q %v", id, err)
	}
}

func TestProfileDefaultsAreEditableAndResettable(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := Store{DB: db}
	if err := store.EnsureDefaults(ctx); err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureDefaults(ctx); err != nil {
		t.Fatal(err)
	}
	live, err := store.Get(ctx, "default-live")
	if err != nil {
		t.Fatal(err)
	}
	animated, err := store.Get(ctx, "default-animated")
	if err != nil {
		t.Fatal(err)
	}
	if live.ShowID != "" || animated.ShowID != "" || live.Definition.Blocks[2].Config["min_mb_per_minute"] == animated.Definition.Blocks[2].Config["min_mb_per_minute"] {
		t.Fatalf("incorrect profile defaults: %+v %+v", live, animated)
	}
	live.Definition.Blocks[2].Config["min_mb_per_minute"] = "12"
	live, err = store.Save(ctx, live, live.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ResetDefault(ctx, live.ID, live.Revision-1); err == nil {
		t.Fatal("stale reset accepted")
	}
	live, err = store.ResetDefault(ctx, live.ID, live.Revision)
	if err != nil || live.Definition.Blocks[2].Config["min_mb_per_minute"] != "8" {
		t.Fatalf("reset failed: %+v %v", live, err)
	}
	if err := store.Delete(ctx, live.ID); err == nil {
		t.Fatal("deleted default chain")
	}
}

func TestProfileDefaultsCaptureExistingAutomationSettings(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	configuration := settings.Store{DB: db}
	if err := configuration.Ensure(ctx); err != nil {
		t.Fatal(err)
	}
	var existing settings.TorrentAutomation
	revision, err := configuration.Load(ctx, "torrent_automation", &existing)
	if err != nil {
		t.Fatal(err)
	}
	existing.MinSeeders = 19
	existing.LiveMinMBPerMinute = 12
	existing.AnimatedMaxMBPerMinute = 110
	existing.AllowedGroups = []string{"FLUX"}
	if _, err := configuration.Save(ctx, "torrent_automation", existing, revision); err != nil {
		t.Fatal(err)
	}
	store := Store{DB: db}
	if err := store.EnsureDefaults(ctx); err != nil {
		t.Fatal(err)
	}
	live, err := store.Get(ctx, "default-live")
	if err != nil {
		t.Fatal(err)
	}
	animated, err := store.Get(ctx, "default-animated")
	if err != nil {
		t.Fatal(err)
	}
	if live.Definition.Blocks[2].Config["min_seeders"] != "19" || live.Definition.Blocks[2].Config["min_mb_per_minute"] != "12" || live.Definition.Blocks[2].Config["allowed_groups"] != "FLUX" || animated.Definition.Blocks[2].Config["max_mb_per_minute"] != "110" {
		t.Fatalf("existing policy was not preserved: %+v %+v", live, animated)
	}
}
