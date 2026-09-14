package backup

import (
	"github.com/CarlFlo/mediaManager/internal/database"
)

type Service struct {
	DB            *database.Store
	DataDir, Path string
	Keep          int
}
