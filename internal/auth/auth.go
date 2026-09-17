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
	recovery map[string]recovery
	attempts map[string]throttle
	hashes   chan struct{}
}

func New(db *database.Store, c config.Config) *Service {
	return &Service{DB: db, Config: c, recovery: map[string]recovery{}, attempts: map[string]throttle{}, hashes: make(chan struct{}, 1)}
}

func (s *Service) acquireHashMemory(ctx context.Context) (func(), error) {
	select {
	case s.hashes <- struct{}{}:
		return func() {
			// Argon2id intentionally uses a 64 MiB working set. Authentication is
			// infrequent, so return those temporary heap pages to the OS promptly
			// instead of letting one sign-in raise the process RSS high-water mark.
			debug.FreeOSMemory()
			<-s.hashes
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
