package torrent

import (
	"encoding/base32"
	"encoding/hex"
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

func MagnetInfoHash(raw string) string {
	if len(raw) > 16384 {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "magnet" {
		return ""
	}
	for _, xt := range u.Query()["xt"] {
		if strings.HasPrefix(xt, "urn:btih:") {
			value := strings.TrimPrefix(xt, "urn:btih:")
			if len(value) == 40 {
				for _, r := range value {
					if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
						return ""
					}
				}
				return strings.ToLower(value)
			}
			if len(value) == 32 {
				decoded, decodeErr := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(value))
				if decodeErr == nil && len(decoded) == 20 {
					return hex.EncodeToString(decoded)
				}
			}
		}
		if strings.HasPrefix(xt, "urn:btmh:1220") && len(xt) == 77 {
			value := strings.TrimPrefix(xt, "urn:btmh:1220")
			if _, decodeErr := hex.DecodeString(value); decodeErr == nil {
				return strings.ToLower(value)
			}
		}
	}
	return ""
}
