package backup

import (
	"errors"
	"sync"

	"github.com/CarlFlo/tally/internal/database"
)

var ErrNotFound = errors.New("backup not found")

type inspectionCache struct {
	size       int64
	modifiedAt int64
	manifest   Manifest
	err        error
}

type Service struct {
	DB                      *database.Store
	DataDir, Path, Timezone string
	Keep                    int
	mu                      sync.Mutex
	inspections             map[string]inspectionCache
}
