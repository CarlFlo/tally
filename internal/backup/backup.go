package backup

import (
	"errors"
	"sync"

	"github.com/CarlFlo/mediaManager/internal/database"
)

var (
	ErrNotFound   = errors.New("backup not found")
	ErrUnverified = errors.New("backup is not verified")
)

type Service struct {
	DB                      *database.Store
	DataDir, Path, Timezone string
	Keep                    int
	mu                      sync.Mutex
}
