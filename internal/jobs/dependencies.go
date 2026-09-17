package jobs

import (
	"context"

	"github.com/CarlFlo/tally/internal/providers"
)

type MetadataSource interface {
	Sync(context.Context, string) (string, error)
	ProcessNext(context.Context) (bool, error)
}

type ProviderControl interface {
	providers.Requester
	Avoid(provider, trigger, entity, reason string)
	Maintenance(context.Context) error
}

type BackupCreator interface {
	Create(context.Context, string) (string, error)
}

type TorrentAutomator interface {
	Run(context.Context) (int, error)
}
