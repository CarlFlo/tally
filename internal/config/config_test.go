package config

import (
	"testing"
	"time"
)

func TestDefaultsAndValidation(t *testing.T) {
	t.Setenv("TZ", "UTC")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.MaxProfiles != 8 || c.AuthMode != "disabled" || c.PasswordMin != 6 || c.SessionIdle != 30*24*time.Hour || c.JobRetries != 3 {
		t.Fatal("unexpected defaults")
	}
	for _, test := range []struct{ key, value string }{{"APP_MAX_PROFILES", "0"}, {"TZ", "Not/AZone"}, {"APP_AUTH_MODE", "invalid"}, {"OIDC_REDIRECT_URL", "https://example.com/wrong"}, {"APP_PUBLIC_URL", "https://example.com/subpath"}} {
		t.Run(test.key, func(t *testing.T) {
			t.Setenv(test.key, test.value)
			if _, err := Load(); err == nil {
				t.Fatal("invalid configuration accepted")
			}
		})
	}
}
