package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/CarlFlo/tally/internal/flows"
	"github.com/CarlFlo/tally/internal/torrent"
)

func TestTorrentAutomationShowEnrollmentIsGlobalButListIsProfileScoped(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	if _, err := s.DB.Exec(`INSERT INTO shows(id,name,status) VALUES
		('show-active','Active Show','Running'),
		('show-inactive','Ended Show','Ended'),
		('show-other','Other Show','Running');
	INSERT INTO episodes(id,show_id,season,number,name,airdate) VALUES
		('episode-active','show-active',1,1,'Future','2099-01-01'),
		('episode-inactive','show-inactive',1,1,'Past','2020-01-01'),
		('episode-other','show-other',1,1,'Future','2099-01-01');
	INSERT INTO profile_shows(profile_id,show_id,added_at) VALUES
		('profile-admin','show-active',1),
		('profile-admin','show-inactive',1);
	INSERT INTO profiles(id,display_name,avatar,created_at) VALUES('profile-member','Member','mint',2);
	INSERT INTO profile_shows(profile_id,show_id,added_at) VALUES('profile-member','show-other',1);`); err != nil {
		t.Fatal(err)
	}
	if err := (torrent.AutomationStore{DB: s.DB}).SetShowPolicy(t.Context(), "show-active", "auto"); err != nil {
		t.Fatal(err)
	}

	response := request(t, h, "GET", "/api/torrents/automation/shows", nil)
	expect(t, response, http.StatusOK)
	var out struct {
		Shows []struct {
			ID                string `json:"id"`
			Name              string `json:"name"`
			Active            int    `json:"active"`
			AutomationEnabled int    `json:"automation_enabled"`
			MediaProfile      struct {
				Effective string `json:"effective"`
			} `json:"media_profile"`
		} `json:"shows"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Shows) != 2 {
		t.Fatalf("admin list returned %d shows: %s", len(out.Shows), response.Body.String())
	}
	found := map[string]struct{ active, enabled int }{}
	for _, show := range out.Shows {
		found[show.ID] = struct{ active, enabled int }{show.Active, show.AutomationEnabled}
	}
	if found["show-active"].active != 1 || found["show-active"].enabled != 1 {
		t.Fatalf("active enrollment missing: %+v", found["show-active"])
	}
	if found["show-inactive"].active != 0 || found["show-inactive"].enabled != 0 {
		t.Fatalf("inactive show state unexpected: %+v", found["show-inactive"])
	}
	for _, show := range out.Shows {
		if show.ID == "show-active" && show.MediaProfile.Effective != torrent.MediaProfileLive {
			t.Fatalf("live show default profile = %q", show.MediaProfile.Effective)
		}
	}
	if _, ok := found["show-other"]; ok {
		t.Fatal("another profile's My Shows leaked into the enrollment list")
	}

	response = request(t, h, "PUT", "/api/torrents/automation/shows/show-inactive", map[string]any{"enabled": true})
	expect(t, response, http.StatusOK)
	if !strings.Contains(response.Body.String(), `"enabled":true`) {
		t.Fatal(response.Body.String())
	}
	policy, err := (torrent.AutomationStore{DB: s.DB}).ShowPolicy(t.Context(), "show-inactive")
	if err != nil || policy != "auto" {
		t.Fatalf("global enrollment was not saved: policy=%q err=%v", policy, err)
	}

	member := &http.Cookie{Name: "tally_profile", Value: "profile-member"}
	response = request(t, h, "GET", "/api/torrents/automation/shows", nil, member)
	expect(t, response, http.StatusOK)
	if strings.Contains(response.Body.String(), "Active Show") || strings.Contains(response.Body.String(), "Ended Show") || !strings.Contains(response.Body.String(), "Other Show") {
		t.Fatalf("member list was not profile scoped: %s", response.Body.String())
	}
	expect(t, request(t, h, "PUT", "/api/torrents/automation/shows/show-other", map[string]any{"enabled": true}, member), http.StatusForbidden)
}

func TestAutomationChainAssignmentRequiresAdminAndMatchingShow(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	if _, err := s.DB.Exec(`INSERT INTO shows(id,name) VALUES('one','One'),('two','Two');
		INSERT INTO profile_shows(profile_id,show_id,added_at) VALUES('profile-admin','one',1),('profile-admin','two',1);
		INSERT INTO profiles(id,display_name,avatar,created_at) VALUES('member','Member','mint',1);
		INSERT INTO profile_shows(profile_id,show_id,added_at) VALUES('member','one',1);`); err != nil {
		t.Fatal(err)
	}
	store := flows.Store{DB: s.DB}
	general, err := store.Save(t.Context(), flows.Flow{Name: "General", Definition: flows.DefaultDefinition()}, 0)
	if err != nil {
		t.Fatal(err)
	}
	custom, err := store.Save(t.Context(), flows.Flow{Name: "One only", ShowID: "one", Definition: flows.DefaultDefinition()}, 0)
	if err != nil {
		t.Fatal(err)
	}
	member := &http.Cookie{Name: "tally_profile", Value: "member"}
	expect(t, request(t, h, "PUT", "/api/torrents/automation/shows/one/chain", map[string]string{"flow_id": general.ID}, member), http.StatusForbidden)
	expect(t, request(t, h, "PUT", "/api/torrents/automation/shows/two/chain", map[string]string{"flow_id": custom.ID}), http.StatusBadRequest)
	expect(t, request(t, h, "PUT", "/api/torrents/automation/shows/one/chain", map[string]string{"flow_id": custom.ID}), http.StatusOK)
	current := request(t, h, "GET", "/api/torrents/automation/shows/one", nil)
	if !strings.Contains(current.Body.String(), custom.ID) {
		t.Fatal(current.Body.String())
	}
	expect(t, request(t, h, "PUT", "/api/torrents/automation/shows/one/chain", map[string]string{"flow_id": ""}), http.StatusOK)
	if id, err := store.Assigned(t.Context(), "one"); err != nil || id != "" {
		t.Fatalf("assignment not cleared: %q %v", id, err)
	}
}

func TestTorrentAutomationBooleanEnrollmentKeepsLegacyPolicyCompatibility(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	if _, err := s.DB.Exec(`INSERT INTO shows(id,name) VALUES('show-policy','Policy Show');
	INSERT INTO profile_shows(profile_id,show_id,added_at) VALUES('profile-admin','show-policy',1);`); err != nil {
		t.Fatal(err)
	}

	expect(t, request(t, h, "PUT", "/api/torrents/automation/shows/show-policy", map[string]any{"policy": "never"}), http.StatusOK)
	current := request(t, h, "GET", "/api/torrents/automation/shows/show-policy", nil)
	expect(t, current, http.StatusOK)
	if !strings.Contains(current.Body.String(), `"policy":"never"`) || !strings.Contains(current.Body.String(), `"enabled":false`) {
		t.Fatal(current.Body.String())
	}

	expect(t, request(t, h, "PUT", "/api/torrents/automation/shows/show-policy", map[string]any{"enabled": true}), http.StatusOK)
	current = request(t, h, "GET", "/api/torrents/automation/shows/show-policy", nil)
	if !strings.Contains(current.Body.String(), `"policy":"auto"`) || !strings.Contains(current.Body.String(), `"enabled":true`) {
		t.Fatal(current.Body.String())
	}
}
