package auth

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CarlFlo/mediaManager/internal/config"
	"github.com/CarlFlo/mediaManager/internal/database"
)

func testAuth(t *testing.T) *Service {
	db, e := database.Open(context.Background(), t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close() })
	return New(db, config.Config{AuthMode: "local", PasswordMin: 4, PasswordMax: 128, ResetCooldown: time.Minute, SessionIdle: time.Hour, SessionAbsolute: 24 * time.Hour, MaxProfiles: 3, OIDCAutoCreate: true})
}
func TestOpaquePasswordHashAndPolicy(t *testing.T) {
	s := testAuth(t)
	for _, password := range []string{"1234", "  PIN  ", "🤍🤍🤍🤍"} {
		if e := s.Policy(password); e != nil {
			t.Fatal(e)
		}
		hash, e := Hash(password)
		if e != nil || !Verify(hash, password) {
			t.Fatal("password verification failed")
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
func TestRecoveryExpiryRestartAndForcedReplacement(t *testing.T) {
	s := testAuth(t)
	hash, _ := Hash("original")
	_, _ = s.DB.Exec("INSERT INTO local_credentials VALUES('user0',?,0)", hash)
	ctx := context.Background()
	if e := s.Recover(ctx, "user0"); e != nil {
		t.Fatal(e)
	}
	if e := s.Recover(ctx, "user0"); e == nil {
		t.Fatal("cooldown not enforced")
	}
	var permanent string
	_ = s.DB.QueryRow("SELECT hash FROM local_credentials WHERE profile_id='user0'").Scan(&permanent)
	if !Verify(permanent, "original") {
		t.Fatal("recovery invalidated permanent password")
	}
	s.recovery["user0"] = recovery{Digest("temporary"), time.Now().Add(time.Minute), time.Now()}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "http://example.com", nil)
	if e := s.Login(ctx, w, r, "user0", "temporary"); e != nil {
		t.Fatal(e)
	}
	request := httptest.NewRequest("GET", "http://example.com", nil)
	request.AddCookie(w.Result().Cookies()[0])
	session, e := s.Resolve(request)
	if e != nil || !session.Restricted {
		t.Fatal("recovery did not restrict session")
	}
	if _, ok := s.recovery["user0"]; ok {
		t.Fatal("temporary credential not consumed")
	}
	s.recovery["user0"] = recovery{Digest("expired"), time.Now().Add(-time.Second), time.Now()}
	if e = s.Login(ctx, httptest.NewRecorder(), r, "user0", "expired"); e == nil {
		t.Fatal("expired credential accepted")
	}
	restarted := New(s.DB, s.Config)
	if len(restarted.recovery) != 0 {
		t.Fatal("recovery survived restart")
	}
	if e = restarted.Change(ctx, httptest.NewRecorder(), r, session, "", "changed"); e != nil {
		t.Fatal(e)
	}
	_ = s.DB.QueryRow("SELECT hash FROM local_credentials WHERE profile_id='user0'").Scan(&permanent)
	if !Verify(permanent, "changed") || Verify(permanent, "original") {
		t.Fatal("permanent replacement failed")
	}
}
func TestOIDCIdentityIsIssuerAndSubject(t *testing.T) {
	s := testAuth(t)
	o := NewOIDC(s, nil)
	ctx := context.Background()
	first, e := o.MapIdentity(ctx, "https://issuer.example", "abc", "Same Name")
	if e != nil || first != "user0" {
		t.Fatalf("first identity: %s %v", first, e)
	}
	same, e := o.MapIdentity(ctx, "https://issuer.example", "abc", "Changed Name")
	if e != nil || same != first {
		t.Fatal("mutable display name changed identity")
	}
	second, e := o.MapIdentity(ctx, "https://another.example", "abc", "Same Name")
	if e != nil || second == first {
		t.Fatal("issuer was ignored")
	}
	third, e := o.MapIdentity(ctx, "https://issuer.example", "xyz", "Same Name")
	if e != nil || third == first || third == second {
		t.Fatal("subject was ignored")
	}
	if _, e = o.MapIdentity(ctx, "https://issuer.example", "over-limit", "Name"); e == nil {
		t.Fatal("OIDC ignored profile maximum")
	}
}
