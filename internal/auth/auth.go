package auth

import (
	"sync"

	"github.com/CarlFlo/mediaManager/internal/config"
	"github.com/CarlFlo/mediaManager/internal/database"
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
