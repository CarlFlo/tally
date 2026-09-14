package providers

import "context"

// Requester is the outbound boundary used by adapters. Production callers
// supply a Coordinator so caching, limits and telemetry cannot be bypassed.
type Requester interface {
	Do(context.Context, Request) (Response, error)
}
