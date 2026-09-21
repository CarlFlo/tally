package auth

import (
	"context"
	"runtime/debug"
	"sync"

	"github.com/CarlFlo/tally/internal/config"
	"github.com/CarlFlo/tally/internal/database"
)

type Service struct {
	DB       *database.Store
	Config   config.Config
	mu       sync.Mutex
	attempts map[string]throttle
	hashes   chan struct{}
}

func New(db *database.Store, c config.Config) *Service {
	return &Service{DB: db, Config: c, attempts: map[string]throttle{}, hashes: make(chan struct{}, 1)}
}

func (s *Service) acquireHashMemory(ctx context.Context) (func(), error) {
	select {
	case s.hashes <- struct{}{}:
		return func() {
			// Password hashing is intentionally memory-hard. Authentication is
			// infrequent, so return the temporary Argon2 heap pages to the OS
			// promptly instead of retaining the process RSS high-water mark.
			debug.FreeOSMemory()
			<-s.hashes
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
