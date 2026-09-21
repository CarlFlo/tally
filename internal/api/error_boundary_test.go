package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CarlFlo/tally/internal/auth"
)

func TestRemoteErrorDoesNotLeakCause(t *testing.T) {
	s := &Server{}
	h := s.wrap(func(http.ResponseWriter, *http.Request, auth.Session) error {
		return remote(errors.New("upstream secret detail"))
	}, publicRoute)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/test", nil))

	expect(t, w, http.StatusBadGateway)
	if strings.Contains(w.Body.String(), "upstream secret detail") {
		t.Fatal("external error cause leaked to the client")
	}
	if !strings.Contains(w.Body.String(), remoteErrorMessage) {
		t.Fatalf("safe external error message missing: %s", w.Body.String())
	}
}

func TestAPIPanicReturnsSafeJSONError(t *testing.T) {
	s := &Server{}
	h := s.security(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("sensitive panic detail")
	}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/test", nil))

	expect(t, w, http.StatusInternalServerError)
	if got := w.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("API panic response content type = %q", got)
	}
	if strings.Contains(w.Body.String(), "sensitive panic detail") {
		t.Fatal("panic detail leaked to the client")
	}
	if !strings.Contains(w.Body.String(), internalErrorMessage) {
		t.Fatalf("safe internal error message missing: %s", w.Body.String())
	}
}
