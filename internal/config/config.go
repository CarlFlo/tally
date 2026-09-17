package config

import "time"

type Config struct {
	Addr, DataDir, PublicURL, Timezone, Theme                     string
	MaxProfiles, PasswordMin, PasswordMax                          int
	ResetCooldown, SessionIdle, SessionAbsolute                    time.Duration
	JobConcurrency, JobRetries, BatchSize, ProviderConcurrency     int
	JobRuntime                                                     time.Duration
	RawRetention, AggregateRetention                               int
}
