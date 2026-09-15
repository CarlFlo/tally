package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestProfileIDsAreOpaqueAndLastAdminIsProtected(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	admin := &http.Cookie{Name: "tally_profile", Value: "user0"}

	first := request(t, h, "POST", "/api/profiles", map[string]any{"name": "First"}, admin)
	expect(t, first, 201)
	firstID := value(t, first, "id")
	second := request(t, h, "POST", "/api/profiles", map[string]any{"name": "Second"}, admin)
	expect(t, second, 201)
	secondID := value(t, second, "id")
	if len(firstID) != 32 || len(secondID) != 32 || firstID == secondID {
		t.Fatal("profile IDs must be opaque and unique")
	}
	expect(t, request(t, h, "POST", "/api/profiles", map[string]any{"name": "Over limit"}, admin), 400)

	// Existing profiles must never be stranded without an administrator.
	expect(t, request(t, h, "PATCH", "/api/profiles/user0/admin", map[string]any{"is_admin": false}, admin), 400)
	expect(t, request(t, h, "DELETE", "/api/profiles/user0", nil, admin), 400)

	expect(t, request(t, h, "PATCH", "/api/profiles/"+secondID+"/admin", map[string]any{"is_admin": true}, admin), 200)
	expect(t, request(t, h, "DELETE", "/api/profiles/user0", nil, admin), 200)
	var oldAdmin int
	_ = s.DB.QueryRow("SELECT COUNT(*) FROM profiles WHERE id='user0'").Scan(&oldAdmin)
	if oldAdmin != 0 {
		t.Fatal("former administrator was not removable")
	}

	expect(t, request(t, h, "DELETE", "/api/profiles/"+firstID, nil, &http.Cookie{Name: "tally_profile", Value: secondID}), 200)
	replacement := request(t, h, "POST", "/api/profiles", map[string]any{"name": "Replacement"}, &http.Cookie{Name: "tally_profile", Value: secondID})
	expect(t, replacement, 201)
	replacementID := value(t, replacement, "id")
	if replacementID == firstID || replacementID == secondID {
		t.Fatal("deleted profile ID was reused")
	}
	boot := request(t, h, "GET", "/api/bootstrap", nil, &http.Cookie{Name: "tally_profile", Value: firstID})
	if !strings.Contains(boot.Body.String(), `"profile":null`) {
		t.Fatal("deleted remembered profile was trusted")
	}
}

func TestDeletingOnlyProfileAllowsCleanFirstAdminBootstrap(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	admin := &http.Cookie{Name: "tally_profile", Value: "user0"}
	deleted := request(t, h, "DELETE", "/api/profiles/user0", nil, admin)
	expect(t, deleted, 200)
	var count int
	if err := s.DB.QueryRow("SELECT COUNT(*) FROM profiles").Scan(&count); err != nil || count != 0 {
		t.Fatal("last profile was not deleted", err)
	}

	signedOut := profileCookie(t, deleted)
	created := request(t, h, "POST", "/api/auth/register", map[string]any{"name": "New owner"}, signedOut)
	expect(t, created, 201)
	var body map[string]any
	if err := json.Unmarshal(created.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["is_admin"] != true || body["id"] == "user0" {
		t.Fatal("first profile after an empty state must become a generated administrator")
	}
}
