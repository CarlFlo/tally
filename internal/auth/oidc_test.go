package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"

	"github.com/CarlFlo/mediaManager/internal/providers"
)

func TestOIDCProtocolStateNoncePKCEAndReplay(t *testing.T) {
	s := testAuth(t)
	control, e := providers.New(context.Background(), s.DB, t.TempDir(), 2, 0)
	if e != nil {
		t.Fatal(e)
	}
	defer control.Close()
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	var issuer, nonce, challenge string
	var wrongNonce bool
	idp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			json.NewEncoder(w).Encode(map[string]any{"issuer": issuer, "authorization_endpoint": issuer + "/authorize", "token_endpoint": issuer + "/token", "jwks_uri": issuer + "/keys", "response_types_supported": []string{"code"}, "subject_types_supported": []string{"public"}, "id_token_signing_alg_values_supported": []string{"RS256"}})
		case "/keys":
			json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: "test", Algorithm: "RS256", Use: "sig"}}})
		case "/token":
			r.ParseForm()
			hash := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
			if base64.RawURLEncoding.EncodeToString(hash[:]) != challenge {
				t.Error("PKCE verifier does not match authorization challenge")
			}
			signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, (&jose.SignerOptions{}).WithHeader("kid", "test"))
			if err != nil {
				t.Error(err)
				http.Error(w, "signing failed", 500)
				return
			}
			n := nonce
			if wrongNonce {
				n = "wrong"
			}
			raw, err := jwt.Signed(signer).Claims(jwt.Claims{Issuer: issuer, Subject: "immutable-subject", Audience: jwt.Audience{"tally"}, Expiry: jwt.NewNumericDate(time.Now().Add(time.Minute)), IssuedAt: jwt.NewNumericDate(time.Now())}).Claims(map[string]any{"nonce": n, "name": "Viewer"}).Serialize()
			if err != nil {
				t.Error(err)
			}
			json.NewEncoder(w).Encode(map[string]any{"access_token": "opaque", "token_type": "Bearer", "expires_in": 60, "id_token": raw})
		default:
			http.NotFound(w, r)
		}
	}))
	defer idp.Close()
	issuer = idp.URL
	s.Config.OIDCIssuer = issuer
	s.Config.OIDCClientID = "tally"
	s.Config.OIDCRedirect = "http://app.example/auth/oidc/callback"
	s.Config.OIDCScopes = "openid,profile"
	o := NewOIDC(s, control)
	start := func() (*httptest.ResponseRecorder, *http.Request) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "http://app.example/auth/oidc/start", nil)
		if e = o.Start(w, r); e != nil {
			t.Fatal(e)
		}
		u, _ := url.Parse(w.Header().Get("Location"))
		nonce = u.Query().Get("nonce")
		challenge = u.Query().Get("code_challenge")
		if nonce == "" || challenge == "" || u.Query().Get("code_challenge_method") != "S256" {
			t.Fatal("state/nonce/PKCE not present")
		}
		callback := httptest.NewRequest("GET", "http://app.example/auth/oidc/callback?state="+url.QueryEscape(u.Query().Get("state"))+"&code=example", nil)
		callback.AddCookie(w.Result().Cookies()[0])
		return httptest.NewRecorder(), callback
	}
	w, r := start()
	if e = o.Callback(w, r); e != nil {
		t.Fatal(e)
	}
	if w.Code != 303 || len(w.Result().Cookies()) < 2 {
		t.Fatal("OIDC did not establish a session")
	}
	if e = o.Callback(httptest.NewRecorder(), r); e == nil {
		t.Fatal("callback replay accepted")
	}
	wrongNonce = true
	w, r = start()
	if e = o.Callback(w, r); e == nil {
		t.Fatal("invalid nonce accepted")
	}
	w, r = start()
	r.URL.RawQuery = "state=bad&code=example"
	if e = o.Callback(w, r); e == nil {
		t.Fatal("invalid state accepted")
	}
}
