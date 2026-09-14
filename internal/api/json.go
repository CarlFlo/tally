package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

func jsonResponse(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func decode(r *http.Request, v any) error {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return apiError{415, "send JSON with Content-Type application/json"}
	}
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return bad("invalid request body")
	}
	if e := d.Decode(new(any)); e != io.EOF {
		return bad("request must contain one JSON value")
	}
	return nil
}
