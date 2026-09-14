package config

import (
	"time"
)

type Config struct {
	Addr, DataDir, AuthMode, PublicURL, Timezone, Language, Theme  string
	MaxProfiles, PasswordMin, PasswordMax                          int
	ResetCooldown, SessionIdle, SessionAbsolute                    time.Duration
	JobConcurrency, JobRetries, BatchSize, ProviderConcurrency     int
	JobRuntime                                                     time.Duration
	OIDCIssuer, OIDCClientID, OIDCSecret, OIDCRedirect, OIDCScopes string
	OIDCAutoCreate                                                 bool
	RawRetention, AggregateRetention                               int
}
