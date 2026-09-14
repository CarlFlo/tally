package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/CarlFlo/mediaManager/internal/config"
)

func healthcheck(c config.Config) error {
	addr := c.Addr
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	client := http.Client{Timeout: 3 * time.Second}
	r, e := client.Get("http://" + addr + "/readyz")
	if e != nil {
		return fmt.Errorf("readiness check failed")
	}
	r.Body.Close()
	if r.StatusCode != 200 {
		return fmt.Errorf("not ready")
	}
	return nil
}
