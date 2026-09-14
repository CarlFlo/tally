package torrent

import (
	"net/url"
	"strconv"
	"strings"
)

func ValidMagnet(raw string) bool {
	if len(raw) > 16384 {
		return false
	}
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "magnet" {
		return false
	}
	for _, xt := range u.Query()["xt"] {
		if strings.HasPrefix(xt, "urn:btih:") {
			v := strings.TrimPrefix(xt, "urn:btih:")
			if len(v) == 40 {
				if _, e := strconv.ParseUint(v[:16], 16, 64); e == nil {
					valid := true
					for _, r := range v {
						if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
							valid = false
						}
					}
					if valid {
						return true
					}
				}
			}
			if len(v) == 32 {
				valid := true
				for _, r := range strings.ToUpper(v) {
					if !strings.ContainsRune("ABCDEFGHIJKLMNOPQRSTUVWXYZ234567", r) {
						valid = false
					}
				}
				if valid {
					return true
				}
			}
		}
		if strings.HasPrefix(xt, "urn:btmh:1220") && len(xt) == 77 {
			return true
		}
	}
	return false
}
