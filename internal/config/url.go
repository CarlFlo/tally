package config

import (
	"fmt"
	"net/url"
)

func ValidateURL(raw string) error {
	u, e := url.Parse(raw)
	if e != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.Fragment != "" {
		return fmt.Errorf("expected an HTTP(S) URL without credentials or fragment")
	}
	return nil
}
