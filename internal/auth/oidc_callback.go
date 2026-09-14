package auth

import (
	"fmt"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

func (o *OIDC) Callback(w http.ResponseWriter, r *http.Request) error {
	cookie, e := r.Cookie("tally_oidc")
	state := r.URL.Query().Get("state")
	if e != nil || state == "" || cookie.Value != state {
		return fmt.Errorf("invalid sign-in state; start again")
	}
	o.mu.Lock()
	flow, ok := o.flows[Digest(state)]
	delete(o.flows, Digest(state))
	o.mu.Unlock()
	o.Auth.Cookie(w, r, "tally_oidc", "", -1)
	if !ok || time.Now().After(flow.expires) {
		return fmt.Errorf("sign-in expired; start again")
	}
	cfg, p, e := o.config(r.Context())
	if e != nil {
		return e
	}
	token, e := cfg.Exchange(o.clientContext(r.Context()), r.URL.Query().Get("code"), oauth2.VerifierOption(flow.verifier))
	if e != nil {
		return fmt.Errorf("OIDC token exchange failed")
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok {
		return fmt.Errorf("identity provider did not return an ID token")
	}
	id, e := p.Verifier(&oidc.Config{ClientID: cfg.ClientID}).Verify(o.clientContext(r.Context()), raw)
	if e != nil || id.Nonce != flow.nonce {
		return fmt.Errorf("identity token validation failed")
	}
	var claims struct {
		Name string `json:"name"`
	}
	if e = id.Claims(&claims); e != nil {
		return fmt.Errorf("invalid identity claims")
	}
	profile, e := o.MapIdentity(r.Context(), id.Issuer, id.Subject, claims.Name)
	if e != nil {
		return e
	}
	if e = o.Auth.NewSession(r.Context(), w, r, profile, false); e != nil {
		return e
	}
	http.Redirect(w, r, "/calendar", http.StatusSeeOther)
	return nil
}
