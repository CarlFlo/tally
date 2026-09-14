package auth

import (
	"fmt"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

func (o *OIDC) Start(w http.ResponseWriter, r *http.Request) error {
	cfg, _, e := o.config(r.Context())
	if e != nil {
		return e
	}
	state, nonce, verifier := Token(), Token(), oauth2.GenerateVerifier()
	o.mu.Lock()
	for k, v := range o.flows {
		if time.Now().After(v.expires) {
			delete(o.flows, k)
		}
	}
	if len(o.flows) >= 1000 {
		o.mu.Unlock()
		return fmt.Errorf("too many pending sign-ins; try later")
	}
	o.flows[Digest(state)] = oidcFlow{nonce, verifier, time.Now().Add(10 * time.Minute)}
	o.mu.Unlock()
	o.Auth.Cookie(w, r, "tally_oidc", state, 600)
	http.Redirect(w, r, cfg.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier)), http.StatusSeeOther)
	return nil
}
