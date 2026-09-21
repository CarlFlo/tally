package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CarlFlo/tally/internal/config"
	"github.com/CarlFlo/tally/internal/database"
	"golang.org/x/crypto/argon2"
)

func testAuth(t *testing.T) *Service {
	db, e := database.Open(context.Background(), t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close() })
	return New(db, config.Config{PasswordMin: 4, PasswordMax: 128, ResetCooldown: time.Minute, SessionIdle: time.Hour, SessionAbsolute: 24 * time.Hour, MaxProfiles: 3})
}

func legacyPasswordHash(password string) string {
	salt := []byte("0123456789abcdef")
	key := argon2.IDKey([]byte(password), salt, 2, 64*1024, 2, 32)
	return "$argon2id$v=19$m=65536,t=2,p=2$" + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(key)
}

func TestOpaquePasswordHashAndPolicy(t *testing.T) {
	s := testAuth(t)
	for _, password := range []string{"1234", "  pass  ", "🤍🤍🤍🤍"} {
		if e := s.Policy(password); e != nil {
			t.Fatal(e)
		}
		hash, e := Hash(password)
		if e != nil || !Verify(hash, password) {
			t.Fatal("password verification failed")
		}
		if NeedsRehash(hash) {
			t.Fatal("new password hash does not use current parameters")
		}
		other, _ := Hash(password)
		if hash == other {
			t.Fatal("salt was reused")
		}
		if Verify(hash, password+" ") {
			t.Fatal("password was normalized")
		}
	}
	if e := s.Policy("123"); e == nil {
		t.Fatal("short password accepted")
	}
	if s.Policy("Mixed_CASE123! ") != nil || s.Policy("bad\x00password") == nil {
		t.Fatal("password character validation failed")
	}
	if Verify("$argon2id$v=19$m=999999999,t=999999,p=255$a$b", "1234") {
		t.Fatal("unbounded hash parameters accepted")
	}
}

func TestLegacyPasswordHashMigratesOnLogin(t *testing.T) {
	s := testAuth(t)
	legacy := legacyPasswordHash("legacy-password")
	if !Verify(legacy, "legacy-password") || !NeedsRehash(legacy) {
		t.Fatal("legacy hash compatibility is broken")
	}
	if _, err := s.DB.Exec("INSERT INTO profiles(id,display_name,avatar,created_at,auth_method) VALUES('legacy-profile','Legacy','violet',1,'password')"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec("INSERT INTO local_credentials VALUES('legacy-profile',?,0)", legacy); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "http://example.com", nil)
	if err := s.Login(context.Background(), w, r, "legacy-profile", "legacy-password"); err != nil {
		t.Fatal(err)
	}
	var stored string
	if err := s.DB.QueryRow("SELECT hash FROM local_credentials WHERE profile_id='legacy-profile'").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored == legacy || NeedsRehash(stored) || !Verify(stored, "legacy-password") {
		t.Fatal("legacy hash was not migrated to current parameters")
	}
}

func TestPasswordOnlyAuthenticationErrors(t *testing.T) {
	s := testAuth(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "http://example.com", nil)
	if err := s.Login(context.Background(), w, r, "missing-profile", "wrong"); err == nil || err.Error() != "incorrect password" {
		t.Fatalf("unexpected login error: %v", err)
	}
	if err := s.Reauthenticate(context.Background(), "missing-profile", "wrong"); err == nil || err.Error() != "incorrect password" {
		t.Fatalf("unexpected reauthentication error: %v", err)
	}
}

func TestPasswordChangeReplacesCredentialAndRevokesOldSession(t *testing.T) {
	s := testAuth(t)
	hash, _ := Hash("original")
	if _, err := s.DB.Exec("INSERT INTO profiles(id,display_name,avatar,created_at,auth_method) VALUES('profile-fixture','Fixture','violet',1,'password')"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec("INSERT INTO local_credentials VALUES('profile-fixture',?,0)", hash); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	loginWriter := httptest.NewRecorder()
	loginRequest := httptest.NewRequest(http.MethodPost, "http://example.com", nil)
	if err := s.Login(ctx, loginWriter, loginRequest, "profile-fixture", "original"); err != nil {
		t.Fatal(err)
	}
	cookies := loginWriter.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("login did not create a session cookie")
	}
	resolveRequest := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	resolveRequest.AddCookie(cookies[0])
	session, err := s.Resolve(resolveRequest)
	if err != nil {
		t.Fatal(err)
	}
	changeWriter := httptest.NewRecorder()
	if err = s.Change(ctx, changeWriter, resolveRequest, session, "original", "changed"); err != nil {
		t.Fatal(err)
	}
	var stored string
	if err = s.DB.QueryRow("SELECT hash FROM local_credentials WHERE profile_id='profile-fixture'").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if !Verify(stored, "changed") || Verify(stored, "original") {
		t.Fatal("password replacement failed")
	}
	var oldSessions int
	if err = s.DB.QueryRow("SELECT COUNT(*) FROM sessions WHERE id=?", session.ID).Scan(&oldSessions); err != nil {
		t.Fatal(err)
	}
	if oldSessions != 0 {
		t.Fatal("old session survived password change")
	}
}

func TestResolveDistinguishesSessionStateFromStorageFailure(t *testing.T) {
	s := testAuth(t)
	if _, err := s.DB.Exec("INSERT INTO profiles(id,display_name,avatar,created_at,auth_method) VALUES('session-profile','Session','violet',1,'none')"); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	if err := s.NewSession(context.Background(), w, r, "session-profile", false); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	request.AddCookie(w.Result().Cookies()[0])

	if err := s.DB.Close(); err != nil {
		t.Fatal(err)
	}
	_, err := s.Resolve(request)
	if err == nil {
		t.Fatal("closed database did not fail session resolution")
	}
	if errors.Is(err, ErrSignInRequired) || errors.Is(err, ErrSessionExpired) {
		t.Fatalf("storage failure was misclassified as an invalid session: %v", err)
	}
}
