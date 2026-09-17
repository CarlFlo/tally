package auth

import (
	"context"
	"encoding/base64"
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

func TestRecoveryExpiryRestartAndForcedReplacement(t *testing.T) {
	s := testAuth(t)
	hash, _ := Hash("original")
	if _, err := s.DB.Exec("INSERT INTO profiles(id,display_name,avatar,created_at,auth_method) VALUES('profile-fixture','Fixture','violet',1,'password')"); err != nil {
		t.Fatal(err)
	}
	_, _ = s.DB.Exec("INSERT INTO local_credentials VALUES('profile-fixture',?,0)", hash)
	ctx := context.Background()
	if e := s.Recover(ctx, "profile-fixture"); e != nil {
		t.Fatal(e)
	}
	if e := s.Recover(ctx, "profile-fixture"); e == nil {
		t.Fatal("cooldown not enforced")
	}
	var stored string
	_ = s.DB.QueryRow("SELECT hash FROM local_credentials WHERE profile_id='profile-fixture'").Scan(&stored)
	if !Verify(stored, "original") {
		t.Fatal("recovery invalidated stored password")
	}
	s.recovery["profile-fixture"] = recovery{Digest("temporary"), time.Now().Add(time.Minute), time.Now()}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "http://example.com", nil)
	if e := s.Login(ctx, w, r, "profile-fixture", "temporary"); e != nil {
		t.Fatal(e)
	}
	request := httptest.NewRequest("GET", "http://example.com", nil)
	request.AddCookie(w.Result().Cookies()[0])
	session, e := s.Resolve(request)
	if e != nil || !session.Restricted {
		t.Fatal("recovery did not restrict session")
	}
	if _, ok := s.recovery["profile-fixture"]; ok {
		t.Fatal("temporary credential not consumed")
	}
	s.recovery["profile-fixture"] = recovery{Digest("expired"), time.Now().Add(-time.Second), time.Now()}
	if e = s.Login(ctx, httptest.NewRecorder(), r, "profile-fixture", "expired"); e == nil {
		t.Fatal("expired credential accepted")
	}
	restarted := New(s.DB, s.Config)
	if len(restarted.recovery) != 0 {
		t.Fatal("recovery survived restart")
	}
	if e = restarted.Change(ctx, httptest.NewRecorder(), r, session, "", "changed"); e != nil {
		t.Fatal(e)
	}
	_ = s.DB.QueryRow("SELECT hash FROM local_credentials WHERE profile_id='profile-fixture'").Scan(&stored)
	if !Verify(stored, "changed") || Verify(stored, "original") {
		t.Fatal("password replacement failed")
	}
}
