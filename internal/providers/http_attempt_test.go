package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestSizeLimitNoRedirectAndErrorsRedactSecrets(t *testing.T) {
	c := control(t)
	var targetCalls atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { targetCalls.Add(1) }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, target.URL, 302)
			return
		}
		w.Write([]byte(strings.Repeat("x", 100)))
	}))
	defer server.Close()
	if _, e := c.Do(context.Background(), Request{Provider: "size", URL: server.URL + "?apikey=TOPSECRET", MaxBytes: 5}); e == nil || strings.Contains(e.Error(), "TOPSECRET") {
		t.Fatal("unsafe size error")
	}
	if _, e := c.Do(context.Background(), Request{Provider: "redirect", URL: server.URL + "/redirect"}); e == nil {
		t.Fatal("redirect accepted")
	}
	if targetCalls.Load() != 0 {
		t.Fatal("redirect followed")
	}
}

func TestInvalidRequestDoesNotEnterRetryOrFailureAccounting(t *testing.T) {
	c := control(t)
	c.retries = 3
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1) }))
	defer server.Close()
	if _, err := c.Do(context.Background(), Request{Provider: "invalid", URL: server.URL, Method: "bad\nmethod"}); err == nil {
		t.Fatal("invalid request was accepted")
	}
	var records int
	if err := c.db.QueryRow("SELECT COUNT(*) FROM provider_requests WHERE provider='invalid'").Scan(&records); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 0 || records != 0 || c.State("invalid").failures != 0 {
		t.Fatal("request construction error was retried or counted as a provider failure")
	}
}
