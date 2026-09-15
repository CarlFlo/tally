package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/CarlFlo/mediaManager/internal/inbox"
)

func TestInboxMarkersDismissalsAndLogsAreProfileScoped(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	if _, err := s.DB.Exec(`INSERT INTO profiles VALUES('profile-member','Alex','mint',1);
 INSERT INTO activity_log(id,action,profile_id,message,created_at) VALUES
 (1,'show_added','profile-member','Added personal show',1),
 (2,'settings_updated','profile-admin','Private operator action',2),
 (3,'job_failed','','System failure',3),
 (4,'episode_progress_updated','profile-member','Noisy progress',4),
 (5,'show_removed','profile-member','Removed personal show',5);`); err != nil {
		t.Fatal(err)
	}
	owner := &http.Cookie{Name: "tally_profile", Value: "profile-admin"}
	member := &http.Cookie{Name: "tally_profile", Value: "profile-member"}
	read := func(cookie *http.Cookie) inbox.Page {
		t.Helper()
		result := request(t, h, "GET", "/api/inbox", nil, cookie)
		expect(t, result, 200)
		var page inbox.Page
		if err := json.Unmarshal(result.Body.Bytes(), &page); err != nil {
			t.Fatal(err)
		}
		return page
	}
	first := read(member)
	if first.Unread != 0 || len(first.Entries) != 0 || first.Latest != 0 {
		t.Fatal("incorrect personal inbox", first)
	}
	all := read(owner)
	if all.Unread != 1 || len(all.Entries) != 1 || all.Entries[0]["status"] != "failed" {
		t.Fatal("incorrect operator inbox", all)
	}
	logs := request(t, h, "GET", "/api/logs", nil, member)
	expect(t, logs, 200)
	if strings.Contains(logs.Body.String(), "Private operator") || strings.Contains(logs.Body.String(), "job_failed") {
		t.Fatal("other activity exposed")
	}
	expect(t, request(t, h, "POST", "/api/inbox/seen", map[string]any{"through": 5}, member), 200)
	if got := read(member); got.Unread != 0 || len(got.Entries) != 0 {
		t.Fatal("seen removed entries", got)
	}
	expect(t, request(t, h, "DELETE", "/api/inbox/2", nil, member), 200)
	var dismissed int
	_ = s.DB.QueryRow("SELECT COUNT(*) FROM inbox_dismissals").Scan(&dismissed)
	if dismissed != 0 {
		t.Fatal("dismissed another profile event")
	}
	expect(t, request(t, h, "DELETE", "/api/inbox/1", nil, member), 200)
	if len(read(member).Entries) != 0 || len(read(owner).Entries) != 1 {
		t.Fatal("dismiss leaked across profiles")
	}
	expect(t, request(t, h, "POST", "/api/inbox/clear", map[string]any{"through": 999999}, member), 200)
	if got := read(member); got.Unread != 0 || len(got.Entries) != 0 {
		t.Fatal("clear did not persist", got)
	}
	if _, err := s.DB.Exec("INSERT INTO activity_log(action,profile_id,message,created_at) VALUES('show_added','profile-member','New arrival',6)"); err != nil {
		t.Fatal(err)
	}
	if got := read(member); got.Unread != 0 || len(got.Entries) != 0 {
		t.Fatal("clear swallowed a future arrival", got)
	}
	expect(t, request(t, h, "POST", "/api/inbox/seen", map[string]any{"through": -1}, member), 400)
	if read(owner).Unread != 1 {
		t.Fatal("member markers changed operator inbox")
	}
}

func TestInboxUsesBellCategoriesInsteadOfActivityMessages(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	if _, err := s.DB.Exec(`INSERT INTO activity_log(action,message,created_at) VALUES
 ('job_succeeded','Completed backup job',1),
 ('job_succeeded','Completed metadata job',2),
 ('job_failed','backup: disk unavailable',3),
 ('settings_updated','Updated scheduling settings',4);`); err != nil {
		t.Fatal(err)
	}
	owner := &http.Cookie{Name: "tally_profile", Value: "profile-admin"}
	expect(t, request(t, h, "PATCH", "/api/preferences", map[string]any{"bell_categories": []string{"backup_successes", "routine_background"}}, owner), 200)
	result := request(t, h, "GET", "/api/inbox", nil, owner)
	expect(t, result, 200)
	var page inbox.Page
	if err := json.Unmarshal(result.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.Unread != 2 || len(page.Entries) != 2 || strings.Contains(result.Body.String(), "disk unavailable") || strings.Contains(result.Body.String(), "scheduling") {
		t.Fatal("bell categories did not filter activity", result.Body.String())
	}
}
