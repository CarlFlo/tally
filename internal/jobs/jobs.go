package jobs

import (
	"context"
	"sync"
	"time"

	"github.com/CarlFlo/tally/internal/config"
	"github.com/CarlFlo/tally/internal/database"
)

type Service struct {
	DB                *database.Store
	Config            config.Config
	Metadata          MetadataSource
	Control           ProviderControl
	Backup            BackupCreator
	TorrentAutomation TorrentAutomator
	OnChange          func(string, ...string)
	ctx               context.Context
	cancel            context.CancelFunc
	mu                sync.Mutex
	running           map[string]context.CancelFunc
	sem               chan struct{}
	wg                sync.WaitGroup
	stopped           bool
	notifications     chan struct{}
	scheduleWake      chan struct{}
}

func New(ctx context.Context, db *database.Store, c config.Config, m MetadataSource, p ProviderControl, b BackupCreator) *Service {
	ctx, cancel := context.WithCancel(ctx)
	if c.JobRuntime <= 0 {
		c.JobRuntime = 5 * time.Minute
	}
	if c.JobConcurrency < 1 {
		c.JobConcurrency = 1
	}
	return &Service{
		DB:            db,
		Config:        c,
		Metadata:      m,
		Control:       p,
		Backup:        b,
		ctx:           ctx,
		cancel:        cancel,
		running:       map[string]context.CancelFunc{},
		sem:           make(chan struct{}, c.JobConcurrency),
		notifications: make(chan struct{}, 1),
		scheduleWake:  make(chan struct{}, 1),
	}
}
