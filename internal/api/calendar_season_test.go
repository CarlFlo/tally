package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestCalendarSeasonSizeUsesSharedDeclaredMetadata(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	if _, err := s.DB.Exec(`INSERT INTO profiles VALUES('profile-member','Alex','mint',1);
 INSERT INTO shows(id,name) VALUES('known','Known season'),('unknown','Unknown season');
 INSERT INTO seasons(id,show_id,number,episode_count) VALUES('season','known',2,8);
 INSERT INTO episodes(id,show_id,season,number,name,airdate) VALUES
 ('one','known',2,1,'First','2026-09-12'),('two','known',2,2,'Second','2026-09-12'),
 ('unknown-one','unknown',1,1,'Pilot','2026-09-12');
 INSERT INTO profile_shows(profile_id,show_id,added_at) VALUES('profile-admin','known',1),('profile-admin','unknown',1);`); err != nil {
		t.Fatal(err)
	}
	owner := &http.Cookie{Name: "tally_profile", Value: "profile-admin"}
	member := &http.Cookie{Name: "tally_profile", Value: "profile-member"}
	path := "/api/calendar?from=2026-09-01&to=2026-10-01"
	result := request(t, h, "GET", path, nil, owner)
	expect(t, result, 200)
	var episodes []struct {
		ShowID string `json:"show_id"`
		Count  int    `json:"season_episode_count"`
	}
	if err := json.Unmarshal(result.Body.Bytes(), &episodes); err != nil {
		t.Fatal(err)
	}
	if len(episodes) != 3 {
		t.Fatal("missing episodes")
	}
	for _, episode := range episodes {
		if episode.ShowID == "known" && episode.Count != 8 {
			t.Fatal("incomplete imported episodes were treated as a complete season")
		}
		if episode.ShowID == "unknown" && episode.Count != 0 {
			t.Fatal("invented full-season size")
		}
	}
	result = request(t, h, "GET", path, nil, member)
	expect(t, result, 200)
	if result.Body.String() != "[]\n" {
		t.Fatal("calendar exposed another profile's follows")
	}
}
