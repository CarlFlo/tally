package metadata

import (
	"context"
)

type triggerKey struct{}

type Info struct {
	Trigger, JobID string
	Force          bool
}

func WithInfo(ctx context.Context, i Info) context.Context {
	return context.WithValue(ctx, triggerKey{}, i)
}
