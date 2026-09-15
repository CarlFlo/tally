package providers

import (
	"net/http"
	"time"
)

type Request struct {
	Provider, Method, URL, Trigger, Entity, JobID string
	Header                                        http.Header
	Body                                          []byte
	TTL                                           time.Duration
	Force, NoRetry, Coalesce                      bool
	MaxBytes                                      int64
}

type Response struct {
	Body   []byte
	Status int
	Header http.Header
}
