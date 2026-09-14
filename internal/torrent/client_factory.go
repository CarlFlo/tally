package torrent

import (
	"context"
	"fmt"
)

func (s *ClientStore) Build(c ClientConfig) (DownloadClient, error) {
	if c.Adapter == "" {
		return nil, ErrNoClient
	}
	adapter := findClient(c.Adapter)
	if adapter == nil {
		return nil, fmt.Errorf("the saved torrent client is no longer supported; choose a client in Settings")
	}
	if e := c.validate(); e != nil {
		return nil, e
	}
	return adapter.create(s.Control, c.Fields), nil
}

func (s *ClientStore) Current(ctx context.Context) (DownloadClient, error) {
	c, e := s.Load(ctx)
	if e != nil {
		return nil, e
	}
	return s.Build(c)
}
