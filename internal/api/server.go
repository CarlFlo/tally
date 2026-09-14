package api

import (
	"io/fs"

	"github.com/CarlFlo/mediaManager/internal/auth"
	"github.com/CarlFlo/mediaManager/internal/config"
	"github.com/CarlFlo/mediaManager/internal/database"
	"github.com/CarlFlo/mediaManager/internal/jobs"
	"github.com/CarlFlo/mediaManager/internal/live"
	"github.com/CarlFlo/mediaManager/internal/metadata"
	"github.com/CarlFlo/mediaManager/internal/providers"
	"github.com/CarlFlo/mediaManager/internal/torrent"
)

type Server struct {
	DB         *database.Store
	Config     config.Config
	Auth       *auth.Service
	OIDC       *auth.OIDC
	Metadata   *metadata.Service
	Control    *providers.Coordinator
	Jobs       *jobs.Service
	Events     *live.Hub
	Clients    *torrent.ClientStore
	Assets     fs.FS
	selections selectionStore
}
