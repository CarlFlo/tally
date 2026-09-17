package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

func Load() (Config, error) {
	c := Config{}
	var errors []string
	s := func(key, fallback string) string {
		if v, ok := os.LookupEnv(key); ok {
			return v
		}
		return fallback
	}
	i := func(key string, fallback, min, max int) int {
		v, err := strconv.Atoi(s(key, strconv.Itoa(fallback)))
		if err != nil || v < min || v > max {
			errors = append(errors, fmt.Sprintf("%s must be %d–%d", key, min, max))
		}
		return v
	}
	d := func(key, fallback string) time.Duration {
		raw := s(key, fallback)
		if strings.HasSuffix(raw, "d") {
			n, err := strconv.Atoi(strings.TrimSuffix(raw, "d"))
			if err == nil {
				raw = strconv.Itoa(n*24) + "h"
			}
		}
		v, err := time.ParseDuration(raw)
		if err != nil || v <= 0 {
			errors = append(errors, key+" must be a positive duration")
		}
		return v
	}
	c.Addr = s("APP_ADDR", ":8080")
	c.DataDir = s("APP_DATA_DIR", "/config")
	c.PublicURL = s("APP_PUBLIC_URL", "")
	c.Timezone = s("TZ", "UTC")
	c.Theme = s("APP_THEME_DEFAULT", "system")
	c.MaxProfiles = i("APP_MAX_PROFILES", 8, 1, 100)
	c.PasswordMin = i("LOCAL_PASSWORD_MIN_LENGTH", 6, 1, 128)
	c.PasswordMax = i("LOCAL_PASSWORD_MAX_LENGTH", 128, 4, 1024)
	c.ResetCooldown = d("LOCAL_PASSWORD_RESET_COOLDOWN", "60s")
	c.SessionIdle = d("SESSION_IDLE_TIMEOUT", "30d")
	c.SessionAbsolute = d("SESSION_ABSOLUTE_TIMEOUT", "180d")
	c.JobConcurrency = i("JOB_MAX_CONCURRENCY", 4, 1, 32)
	c.JobRetries = i("JOB_MAX_RETRIES", 3, 0, 5)
	c.BatchSize = i("JOB_MAX_BATCH_SIZE", 50, 1, 500)
	c.ProviderConcurrency = i("PROVIDER_MAX_CONCURRENCY", 2, 1, 16)
	c.JobRuntime = d("JOB_MAX_RUNTIME", "5m")
	c.RawRetention = i("STATS_RAW_RETENTION_DAYS", 30, 1, 365)
	c.AggregateRetention = i("STATS_AGGREGATE_RETENTION_DAYS", 365, 30, 3650)
	if c.Theme != "system" && c.Theme != "light" && c.Theme != "dark" {
		errors = append(errors, "APP_THEME_DEFAULT must be system, light, or dark")
	}
	if c.PasswordMax < c.PasswordMin {
		errors = append(errors, "password maximum is below minimum")
	}
	if c.SessionIdle > c.SessionAbsolute {
		errors = append(errors, "idle session timeout exceeds absolute timeout")
	}
	if _, err := time.LoadLocation(c.Timezone); err != nil {
		errors = append(errors, "TZ must be an IANA timezone")
	}
	if c.PublicURL != "" {
		if err := ValidateURL(c.PublicURL); err != nil {
			errors = append(errors, "APP_PUBLIC_URL: "+err.Error())
		} else {
			u, err := url.Parse(c.PublicURL)
			if err == nil && ((u.Path != "" && u.Path != "/") || u.RawQuery != "") {
				errors = append(errors, "APP_PUBLIC_URL must be the root HTTP(S) origin")
			}
		}
	}
	if len(errors) > 0 {
		return c, fmt.Errorf("invalid configuration: %s", strings.Join(errors, "; "))
	}
	return c, nil
}
