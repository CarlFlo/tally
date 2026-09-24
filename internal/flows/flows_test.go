package flows

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/CarlFlo/tally/internal/database"
	"github.com/CarlFlo/tally/internal/torrent"
)

func testDefinition() Definition {
	return Definition{
		Nodes: []Node{
			{ID: "trigger", Type: "trigger.episode"},
			{ID: "query", Type: "query.build"},
			{ID: "search", Type: "jackett.search"},
			{ID: "filter", Type: "torrent.filter"},
			{ID: "branch", Type: "logic.has_results"},
			{ID: "best", Type: "torrent.best"},
			{ID: "download", Type: "action.download"},
			{ID: "stop", Type: "action.stop"},
		},
		Edges: []Edge{
			{ID: "a", Source: "trigger", SourcePort: "event", Target: "query", TargetPort: "event"},
			{ID: "b", Source: "query", SourcePort: "query", Target: "search", TargetPort: "query"},
			{ID: "c", Source: "search", SourcePort: "results", Target: "filter", TargetPort: "results"},
			{ID: "d", Source: "filter", SourcePort: "results", Target: "branch", TargetPort: "results"},
			{ID: "e", Source: "branch", SourcePort: "true", Target: "best", TargetPort: "results"},
			{ID: "f", Source: "branch", SourcePort: "false", Target: "stop", TargetPort: "results"},
			{ID: "g", Source: "best", SourcePort: "candidate", Target: "download", TargetPort: "candidate"},
		},
	}
}

func TestValidationRejectsUntrustedGraphs(t *testing.T) {
	base := testDefinition()
	if err := Validate("Test", base); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		change func(*Definition)
	}{
		{"unknown node", func(d *Definition) { d.Nodes[1].Type = "shell.execute" }},
		{"wrong port type", func(d *Definition) { d.Edges[1].SourcePort = "event" }},
		{"missing input", func(d *Definition) { d.Edges = d.Edges[:len(d.Edges)-1] }},
		{"duplicate node", func(d *Definition) { d.Nodes[1].ID = "trigger" }},
		{"second trigger", func(d *Definition) { d.Nodes = append(d.Nodes, Node{ID: "other", Type: "trigger.manual"}) }},
		{"invalid config", func(d *Definition) {
			d.Nodes[3].Config = map[string]json.RawMessage{"min_seeders": json.RawMessage(`"-1"`)}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			d := testDefinition()
			test.change(&d)
			if err := Validate("Test", d); err == nil {
				t.Fatal("invalid graph accepted")
			}
		})
	}
	if err := validateConfig(registry["text.replace"], nil); err == nil {
		t.Fatal("replace node accepted missing find text")
	}
}

func TestReplayBranchesAndSuppressesActions(t *testing.T) {
	flow := Flow{ID: "flow", Name: "Test", Revision: 3, Definition: testDefinition()}
	event := Event{ShowName: "Example Show", Season: 1, Episode: 2}
	called := 0
	search := func(_ context.Context, q string) ([]torrent.SearchResult, error) {
		called++
		if q != "Example Show S01E02" {
			t.Fatalf("unexpected query %q", q)
		}
		return []torrent.SearchResult{{Name: "Example.Show.S01E02.1080p.WEB-DL-GROUP", Seeders: 8, Magnet: "magnet:?xt=urn:btih:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, {Name: "Other.Show.S01E02", Seeders: 9}}, nil
	}
	run := Execute(context.Background(), flow, event, search)
	if run.Status != "completed" || called != 1 {
		t.Fatalf("unexpected run: %+v", run)
	}
	var downloaded, stopped, filtered bool
	for _, step := range run.Steps {
		if step.NodeID == "download" {
			downloaded = strings.Contains(step.Summary, "Would inspect and submit")
		}
		if step.NodeID == "stop" {
			stopped = step.Status == "skipped"
		}
		if step.NodeID == "filter" {
			filtered = strings.Contains(step.Summary, "1 of 2")
			if output, ok := step.Output.(Results); !ok || output.RejectionCounts["confidence_too_low"] != 1 {
				t.Fatalf("missing rejection detail: %+v", step.Output)
			}
		}
	}
	if !downloaded || !stopped || !filtered {
		t.Fatalf("wrong path: %+v", run.Steps)
	}
	if strings.Contains(string(mustJSON(t, run)), "magnet:?") {
		t.Fatal("secret-bearing transport persisted in trace")
	}
	empty := Execute(context.Background(), flow, event, func(context.Context, string) ([]torrent.SearchResult, error) {
		return []torrent.SearchResult{{Name: "Wrong.Show.S01E02", Seeders: 0}}, nil
	})
	if empty.Status != "stopped" {
		t.Fatalf("false branch not followed: %+v", empty)
	}
	failure := Execute(context.Background(), flow, event, nil)
	if failure.Status != "failed" {
		t.Fatalf("missing search provider did not fail: %+v", failure)
	}
}
func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	return b
}

func TestPersistenceRevisionsAndHistoricalSnapshot(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := Store{DB: db}
	flow, err := store.Save(ctx, Flow{Name: "First", Definition: testDefinition()}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if flow.Revision != 1 {
		t.Fatal(flow)
	}
	run := Execute(ctx, flow, Event{ShowName: "Example Show", Season: 1, Episode: 2}, func(context.Context, string) ([]torrent.SearchResult, error) { return nil, nil })
	run, err = store.Record(ctx, run)
	if err != nil {
		t.Fatal(err)
	}
	flow.Name = "Second"
	flow.Definition.Nodes[1].Config = map[string]json.RawMessage{"prefix": json.RawMessage(`"New"`)}
	updated, err := store.Save(ctx, flow, flow.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Revision != 2 {
		t.Fatal(updated)
	}
	if _, err = store.Save(ctx, flow, flow.Revision); err == nil {
		t.Fatal("stale revision accepted")
	}
	old, err := store.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if old.FlowRevision != 1 || old.Definition.Nodes[1].Config != nil {
		t.Fatalf("historical snapshot changed: %+v", old)
	}
	listed, err := store.ListRuns(ctx)
	if err != nil || len(listed) != 1 {
		t.Fatalf("run list: %v %+v", err, listed)
	}
}
