package api

import (
	"io/fs"

	"github.com/CarlFlo/tally/internal/auth"
	"github.com/CarlFlo/tally/internal/backup"
	"github.com/CarlFlo/tally/internal/config"
	"github.com/CarlFlo/tally/internal/database"
	"github.com/CarlFlo/tally/internal/jobs"
	"github.com/CarlFlo/tally/internal/live"
	"github.com/CarlFlo/tally/internal/localization"
	"github.com/CarlFlo/tally/internal/metadata"
	"github.com/CarlFlo/tally/internal/providers"
	"github.com/CarlFlo/tally/internal/torrent"
)

type Server struct {
	DB         *database.Store
	Backup     *backup.Service
	Config     config.Config
	Auth       *auth.Service
	OIDC       *auth.OIDC
	Metadata   *metadata.Service
	Control    *providers.Coordinator
	Jobs       *jobs.Service
	Events     *live.Hub
	Locales    *localization.Registry
	Clients    *torrent.ClientStore
	Assets     fs.FS
	selections selectionStore
}
