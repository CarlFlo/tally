package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CarlFlo/mediaManager/internal/auth"
)

func TestAdministratorRoleIsTransferableWithBackendGuard(t *testing.T) {
	s, h, _ := testServer(t, "disabled")
	if _, err := s.DB.Exec("INSERT INTO profiles VALUES('user1','Alex','mint',1); INSERT INTO profiles VALUES('user2','Sam','mint',2)"); err != nil {
		t.Fatal(err)
	}
	user1 := &http.Cookie{Name: "tally_profile", Value: "user1"}
	admin := &http.Cookie{Name: "tally_profile", Value: "user0"}

	expect(t, request(t, h, "GET", "/api/settings", nil, user1), 403)
	expect(t, request(t, h, "DELETE", "/api/profiles/user2", nil, user1), 403)

	expect(t, request(t, h, "PATCH", "/api/profiles/user1/admin", map[string]any{"is_admin": true}, admin), 200)
	expect(t, request(t, h, "GET", "/api/settings", nil, user1), 200)
	expect(t, request(t, h, "PATCH", "/api/profiles/user0/admin", map[string]any{"is_admin": false}, user1), 200)
	expect(t, request(t, h, "GET", "/api/settings", nil, admin), 403)

	// The only remaining admin cannot demote themselves while other profiles remain.
	expect(t, request(t, h, "PATCH", "/api/profiles/user1/admin", map[string]any{"is_admin": false}, user1), 400)

	expect(t, request(t, h, "PATCH", "/api/profiles/user2/admin", map[string]any{"is_admin": true}, user1), 200)
	expect(t, request(t, h, "PATCH", "/api/profiles/user1/admin", map[string]any{"is_admin": false}, &http.Cookie{Name: "tally_profile", Value: "user2"}), 200)

	var events int
	if err := s.DB.QueryRow("SELECT COUNT(*) FROM activity_log WHERE action IN ('admin_granted','admin_revoked')").Scan(&events); err != nil || events < 4 {
		t.Fatal("administrator changes were not audited", events, err)
	}
}

func TestLocalAdminDemotionAndDeletionRequireActingAdminsPassword(t *testing.T) {
	s, h, _ := testServer(t, "local")
	adminHash, err := auth.Hash("admin-pass")
	if err != nil {
		t.Fatal(err)
	}
	userHash, err := auth.Hash("alex-pass")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec("INSERT INTO local_credentials VALUES('user0',?,0)", adminHash); err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec("INSERT INTO profiles VALUES('user1','Alex','mint',1)"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec("INSERT INTO local_credentials VALUES('user1',?,0)", userHash); err != nil {
		t.Fatal(err)
	}

	adminRecorder := httptest.NewRecorder()
	if err = s.Auth.NewSession(context.Background(), adminRecorder, httptest.NewRequest("GET", "/", nil), "user0", false); err != nil {
		t.Fatal(err)
	}
	adminCookies := adminRecorder.Result().Cookies()
	expect(t, request(t, h, "PATCH", "/api/profiles/user1/admin", map[string]any{"is_admin": true}, adminCookies...), 200)

	userRecorder := httptest.NewRecorder()
	if err = s.Auth.NewSession(context.Background(), userRecorder, httptest.NewRequest("GET", "/", nil), "user1", false); err != nil {
		t.Fatal(err)
	}
	userCookies := userRecorder.Result().Cookies()
	// Admin B must authenticate as B. Admin A's password must not authorize B's action.
	expect(t, request(t, h, "PATCH", "/api/profiles/user0/admin", map[string]any{"is_admin": false, "password": "admin-pass"}, userCookies...), 401)
	time.Sleep(1100 * time.Millisecond)
	expect(t, request(t, h, "PATCH", "/api/profiles/user0/admin", map[string]any{"is_admin": false, "password": "alex-pass"}, userCookies...), 200)
	expect(t, request(t, h, "GET", "/api/settings", nil, adminCookies...), 403)

	expect(t, request(t, h, "PATCH", "/api/profiles/user0/admin", map[string]any{"is_admin": true}, userCookies...), 200)
	expect(t, request(t, h, "DELETE", "/api/profiles/user0", map[string]any{"password": "admin-pass"}, userCookies...), 401)
	time.Sleep(1100 * time.Millisecond)
	expect(t, request(t, h, "DELETE", "/api/profiles/user0", map[string]any{"password": "alex-pass"}, userCookies...), 200)

	var message string
	if err := s.DB.QueryRow("SELECT message FROM activity_log WHERE action='profile_deleted' ORDER BY id DESC LIMIT 1").Scan(&message); err != nil || !strings.Contains(message, "My profile") {
		t.Fatal("administrator deletion activity lost", err)
	}
}
