package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/CarlFlo/tally/internal/providers"
)

type oidcFlow struct {
	nonce, verifier string
	expires         time.Time
}

type OIDC struct {
	Auth     *Service
	Control  providers.Requester
	mu       sync.Mutex
	flows    map[string]oidcFlow
	provider *oidc.Provider
}

func NewOIDC(a *Service, c providers.Requester) *OIDC {
	return &OIDC{Auth: a, Control: c, flows: map[string]oidcFlow{}}
}

func (o *OIDC) clientContext(ctx context.Context) context.Context {
	return oidc.ClientContext(ctx, &http.Client{Transport: providers.Transport{Coordinator: o.Control}, Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
}

func (o *OIDC) config(ctx context.Context) (*oauth2.Config, *oidc.Provider, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.provider == nil {
		p, e := oidc.NewProvider(o.clientContext(ctx), o.Auth.Config.OIDCIssuer)
		if e != nil {
			return nil, nil, fmt.Errorf("OIDC discovery failed; check issuer configuration and provider availability")
		}
		o.provider = p
	}
	scopes := strings.Split(o.Auth.Config.OIDCScopes, ",")
	found := false
	for _, s := range scopes {
		if s == "openid" {
			found = true
		}
	}
	if !found {
		scopes = append(scopes, "openid")
	}
	return &oauth2.Config{ClientID: o.Auth.Config.OIDCClientID, ClientSecret: o.Auth.Config.OIDCSecret, RedirectURL: o.Auth.Config.OIDCRedirect, Scopes: scopes, Endpoint: o.provider.Endpoint()}, o.provider, nil
}
