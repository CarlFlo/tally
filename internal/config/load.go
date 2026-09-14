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
		v, e := strconv.Atoi(s(key, strconv.Itoa(fallback)))
		if e != nil || v < min || v > max {
			errors = append(errors, fmt.Sprintf("%s must be %d–%d", key, min, max))
		}
		return v
	}
	b := func(key string, fallback bool) bool {
		v, e := strconv.ParseBool(s(key, strconv.FormatBool(fallback)))
		if e != nil {
			errors = append(errors, key+" must be true or false")
		}
		return v
	}
	d := func(key, fallback string) time.Duration {
		raw := s(key, fallback)
		if strings.HasSuffix(raw, "d") {
			n, e := strconv.Atoi(strings.TrimSuffix(raw, "d"))
			if e == nil {
				raw = strconv.Itoa(n*24) + "h"
			}
		}
		v, e := time.ParseDuration(raw)
		if e != nil || v <= 0 {
			errors = append(errors, key+" must be a positive duration")
		}
		return v
	}
	c.Addr = s("APP_ADDR", ":8080")
	c.DataDir = s("APP_DATA_DIR", "/config")
	c.AuthMode = s("APP_AUTH_MODE", "disabled")
	c.PublicURL = s("APP_PUBLIC_URL", "")
	c.Timezone = s("TZ", "UTC")
	c.Language = s("APP_LANGUAGE", "en")
	c.Theme = s("APP_THEME_DEFAULT", "system")
	c.MaxProfiles = i("APP_MAX_PROFILES", 8, 1, 100)
	c.PasswordMin = i("LOCAL_PASSWORD_MIN_LENGTH", 4, 1, 128)
	c.PasswordMax = i("LOCAL_PASSWORD_MAX_LENGTH", 128, 4, 1024)
	c.ResetCooldown = d("LOCAL_PASSWORD_RESET_COOLDOWN", "60s")
	c.SessionIdle = d("SESSION_IDLE_TIMEOUT", "30d")
	c.SessionAbsolute = d("SESSION_ABSOLUTE_TIMEOUT", "180d")
	c.JobConcurrency = i("JOB_MAX_CONCURRENCY", 4, 1, 32)
	c.JobRetries = i("JOB_MAX_RETRIES", 3, 0, 5)
	c.BatchSize = i("JOB_MAX_BATCH_SIZE", 50, 1, 500)
	c.ProviderConcurrency = i("PROVIDER_MAX_CONCURRENCY", 2, 1, 16)
	c.JobRuntime = d("JOB_MAX_RUNTIME", "5m")
	c.OIDCIssuer = s("OIDC_ISSUER_URL", "")
	c.OIDCClientID = s("OIDC_CLIENT_ID", "")
	c.OIDCSecret = s("OIDC_CLIENT_SECRET", "")
	c.OIDCRedirect = s("OIDC_REDIRECT_URL", "")
	c.OIDCScopes = s("OIDC_SCOPES", "openid,profile,email")
	c.OIDCAutoCreate = b("OIDC_AUTO_CREATE_USERS", true)
	c.RawRetention = i("STATS_RAW_RETENTION_DAYS", 30, 1, 365)
	c.AggregateRetention = i("STATS_AGGREGATE_RETENTION_DAYS", 365, 30, 3650)
	if c.AuthMode != "disabled" && c.AuthMode != "local" && c.AuthMode != "oidc" {
		errors = append(errors, "APP_AUTH_MODE must be disabled, local, or oidc")
	}
	if c.Theme != "system" && c.Theme != "light" && c.Theme != "dark" {
		errors = append(errors, "APP_THEME_DEFAULT must be system, light, or dark")
	}
	if c.PasswordMax < c.PasswordMin {
		errors = append(errors, "password maximum is below minimum")
	}
	if c.SessionIdle > c.SessionAbsolute {
		errors = append(errors, "idle session timeout exceeds absolute timeout")
	}
	if _, e := time.LoadLocation(c.Timezone); e != nil {
		errors = append(errors, "TZ must be an IANA timezone")
	}
	for key, v := range map[string]string{"APP_PUBLIC_URL": c.PublicURL, "OIDC_ISSUER_URL": c.OIDCIssuer, "OIDC_REDIRECT_URL": c.OIDCRedirect} {
		if v != "" {
			if e := ValidateURL(v); e != nil {
				errors = append(errors, key+": "+e.Error())
			}
		}
	}
	if c.AuthMode == "oidc" && (c.OIDCIssuer == "" || c.OIDCClientID == "" || c.OIDCRedirect == "") {
		errors = append(errors, "OIDC issuer, client ID, and redirect URL are required")
	}
	if c.OIDCRedirect != "" {
		u, e := url.Parse(c.OIDCRedirect)
		if e == nil && (u.Path != "/auth/oidc/callback" || u.RawQuery != "") {
			errors = append(errors, "OIDC_REDIRECT_URL must end in /auth/oidc/callback with no query")
		}
	}
	if c.PublicURL != "" {
		u, e := url.Parse(c.PublicURL)
		if e == nil && ((u.Path != "" && u.Path != "/") || u.RawQuery != "") {
			errors = append(errors, "APP_PUBLIC_URL must be the root HTTP(S) origin")
		}
	}
	if c.Language != "en" {
		errors = append(errors, "only APP_LANGUAGE=en is currently supported")
	}
	if len(errors) > 0 {
		return c, fmt.Errorf("invalid configuration: %s", strings.Join(errors, "; "))
	}
	return c, nil
}
