package auth

import (
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
	return &Service{DB: db, Config: c, recovery: map[string]recovery{}, attempts: map[string]throttle{}, hashes: make(chan struct{}, 2)}
}
