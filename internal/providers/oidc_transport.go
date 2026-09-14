package providers

import (
	"bytes"
	"io"
	"net/http"
)

// Transport allows standards-based OIDC clients to use the same outbound controls.
type Transport struct{ Coordinator Requester }

func (t Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	var body []byte
	var e error
	if req.Body != nil {
		body, e = io.ReadAll(io.LimitReader(req.Body, 1<<20))
		if e != nil {
			return nil, e
		}
	}
	r, e := t.Coordinator.Do(req.Context(), Request{Provider: "oidc", Method: req.Method, URL: req.URL.String(), Header: req.Header, Body: body, Trigger: "authentication", NoRetry: req.Method != "GET"})
	if e != nil {
		return nil, e
	}
	return &http.Response{StatusCode: r.Status, Header: r.Header, Body: io.NopCloser(bytes.NewReader(r.Body)), Request: req}, nil
}
