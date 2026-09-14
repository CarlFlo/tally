package config

import (
	"time"
)

type Config struct {
	Addr, DataDir, AuthMode, PublicURL, Timezone, Language, Theme  string
	MaxProfiles, PasswordMin, PasswordMax                          int
	ResetCooldown, SessionIdle, SessionAbsolute                    time.Duration
	MetadataCron, MaintenanceCron, BackupCron                      string
	JobConcurrency, JobRetries, BatchSize, ProviderConcurrency     int
	JobRuntime                                                     time.Duration
	BackupEnabled                                                  bool
	BackupKeep                                                     int
	BackupPath                                                     string
	TorznabURL, TorznabKey, TorznabName                            string
	TorznabProviders                                               []TorznabConfig
	WebhookURL                                                     string
	OIDCIssuer, OIDCClientID, OIDCSecret, OIDCRedirect, OIDCScopes string
	OIDCAutoCreate                                                 bool
	RawRetention, AggregateRetention                               int
}

type TorznabConfig struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	URL    string `json:"url"`
	APIKey string `json:"api_key"`
}
