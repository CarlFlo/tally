package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/CarlFlo/tally/internal/flows"
	"github.com/CarlFlo/tally/internal/torrent"
)

func TestAdvancedFlowAPIIsAdminOnlyAndReplaysHistoricalTrigger(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	if _, err := s.DB.Exec("INSERT INTO profiles(id,display_name,avatar,created_at) VALUES('member','Member','mint',1)"); err != nil {
		t.Fatal(err)
	}
	member := &http.Cookie{Name: "tally_profile", Value: "member"}
	definition := flows.DefaultDefinition()
	if response := request(t, h, "POST", "/api/flows", flows.Flow{Name: "Test", Definition: definition}, member); response.Code != http.StatusForbidden {
		t.Fatalf("member created flow: %d", response.Code)
	}
	created := request(t, h, "POST", "/api/flows", flows.Flow{Name: "Test", Definition: definition})
	expect(t, created, http.StatusCreated)
	var flow flows.Flow
	if err := json.Unmarshal(created.Body.Bytes(), &flow); err != nil {
		t.Fatal(err)
	}
	if response := request(t, h, "GET", "/api/flows", nil, member); response.Code != http.StatusForbidden {
		t.Fatalf("member listed flows: %d", response.Code)
	}
	if response := request(t, h, "GET", "/api/flows/runs", nil, member); response.Code != http.StatusForbidden {
		t.Fatalf("member listed runs: %d", response.Code)
	}
	if response := request(t, h, "PUT", "/api/flows/"+flow.ID, flow, member); response.Code != http.StatusForbidden {
		t.Fatalf("member edited flow: %d", response.Code)
	}
	if response := request(t, h, "POST", "/api/flows/"+flow.ID+"/replay", map[string]string{"source_run_id": "x"}, member); response.Code != http.StatusForbidden {
		t.Fatalf("member replayed flow: %d", response.Code)
	}
	invalid := flow
	invalid.Definition.Blocks[1].Type = "shell.execute"
	expect(t, request(t, h, "PUT", "/api/flows/"+flow.ID, invalid), http.StatusBadRequest)
	sourceID, err := (torrent.AutomationStore{DB: s.DB}).StartRun(t.Context(), torrent.AutomationRun{ShowID: "show", EpisodeID: "episode", ShowName: "Example Show", Season: 1, Episode: 2, Query: "Example Show S01E02"})
	if err != nil {
		t.Fatal(err)
	}
	if err = (torrent.AutomationStore{DB: s.DB}).FinishRun(t.Context(), sourceID, torrent.RunSkipped, torrent.ReleaseAssessment{}, ""); err != nil {
		t.Fatal(err)
	}
	response := request(t, h, "POST", "/api/flows/"+flow.ID+"/replay", map[string]string{"source_run_id": sourceID})
	expect(t, response, http.StatusCreated)
	var run flows.Run
	if err = json.Unmarshal(response.Body.Bytes(), &run); err != nil {
		t.Fatal(err)
	}
	if run.SourceRunID != sourceID || run.FlowRevision != 1 || run.Event.ShowName != "Example Show" || run.Status != "failed" {
		t.Fatalf("bad replay: %+v", run)
	}
	if strings.Contains(response.Body.String(), "api_key") || strings.Contains(response.Body.String(), "magnet:") {
		t.Fatal("replay leaked transport or credential")
	}
	expect(t, request(t, h, "GET", "/api/flows/runs/"+run.ID, nil), http.StatusOK)
	expect(t, request(t, h, "GET", "/api/flows/runs/"+run.ID, nil, member), http.StatusForbidden)
	custom := request(t, h, "POST", "/api/flows/"+flow.ID+"/replay", map[string]any{"event": map[string]any{"kind": "show_available", "show_name": "  Example Show  ", "season": 1, "episode": 3}})
	expect(t, custom, http.StatusCreated)
	var customRun flows.Run
	if err = json.Unmarshal(custom.Body.Bytes(), &customRun); err != nil {
		t.Fatal(err)
	}
	if customRun.SourceRunID != "" || customRun.Event.Kind != "show_available" || customRun.Event.ShowName != "Example Show" || customRun.Event.ShowID == "" || customRun.Event.EpisodeID == "" || customRun.Event.TriggeredAt == 0 || customRun.Event.Episode != 3 {
		t.Fatalf("bad custom event: %+v", customRun.Event)
	}
	if len(customRun.Steps) == 0 || customRun.Steps[0].Output == nil {
		t.Fatalf("custom trigger was not inspectable: %+v", customRun.Steps)
	}
	loaded := request(t, h, "GET", "/api/flows/runs/"+customRun.ID, nil)
	expect(t, loaded, http.StatusOK)
	if !strings.Contains(loaded.Body.String(), `"kind":"show_available"`) || !strings.Contains(loaded.Body.String(), `"episode_id":"custom-`) {
		t.Fatal("recorded custom event lacks trigger details")
	}
	for _, payload := range []map[string]any{
		{"event": map[string]any{"kind": "show_available", "show_name": "", "season": 1, "episode": 3}},
		{"event": map[string]any{"kind": "show_available", "show_name": "Example", "season": 1, "episode": 0}},
		{"event": map[string]any{"kind": "unknown", "show_name": "Example", "season": 1, "episode": 3}},
		{"source_run_id": sourceID, "event": map[string]any{"kind": "show_available", "show_name": "Example", "season": 1, "episode": 3}},
	} {
		expect(t, request(t, h, "POST", "/api/flows/"+flow.ID+"/replay", payload), http.StatusBadRequest)
	}
}

func TestDefaultChainResetRequiresAdminAndRejectsStaleRevision(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	if _, err := s.DB.Exec("INSERT INTO profiles(id,display_name,avatar,created_at) VALUES('member','Member','mint',1)"); err != nil {
		t.Fatal(err)
	}
	member := &http.Cookie{Name: "tally_profile", Value: "member"}
	var live flows.Flow
	expect(t, request(t, h, "GET", "/api/flows", nil), http.StatusOK)
	if err := json.Unmarshal(request(t, h, "GET", "/api/flows/default-live", nil).Body.Bytes(), &live); err != nil {
		t.Fatal(err)
	}
	if response := request(t, h, "POST", "/api/flows/default-live/reset", map[string]int{"revision": live.Revision}, member); response.Code != http.StatusForbidden {
		t.Fatalf("member reset default: %d", response.Code)
	}
	live.Definition.Blocks[2].Config["min_seeders"] = "7"
	updated := request(t, h, "PUT", "/api/flows/default-live", live)
	expect(t, updated, http.StatusOK)
	if response := request(t, h, "POST", "/api/flows/default-live/reset", map[string]int{"revision": live.Revision}); response.Code != http.StatusConflict {
		t.Fatalf("stale reset: %d", response.Code)
	}
	var changed flows.Flow
	if err := json.Unmarshal(updated.Body.Bytes(), &changed); err != nil {
		t.Fatal(err)
	}
	expect(t, request(t, h, "POST", "/api/flows/default-live/reset", map[string]int{"revision": changed.Revision}), http.StatusOK)
	if response := request(t, h, "DELETE", "/api/flows/default-live", nil); response.Code != http.StatusBadRequest {
		t.Fatalf("deleted default: %d", response.Code)
	}
}
