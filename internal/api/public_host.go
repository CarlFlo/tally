package api

import (
	"net"
	"net/url"
	"strings"
)

func publicHost(raw string) bool {
	u, e := url.Parse(raw)
	if e != nil {
		return false
	}
	host := u.Hostname()
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") || strings.HasSuffix(host, ".ts.net") {
		return false
	}
	if ip := net.ParseIP(host); ip != nil {
		return !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsLinkLocalUnicast() && !ip.IsUnspecified()
	}
	return host != "" && strings.Contains(host, ".")
}
