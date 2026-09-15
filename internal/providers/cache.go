package providers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"
)

func cacheKey(r Request) string {
	h := sha256.Sum256([]byte(r.Provider + "\x00" + r.URL))
	return hex.EncodeToString(h[:])
}

func (c *Coordinator) Maintenance(ctx context.Context) error {
	if _, err := c.cache.ExecContext(ctx, "DELETE FROM responses WHERE expires<?", time.Now().Add(-7*24*time.Hour).Unix()); err != nil {
		return err
	}
	_, err := c.cache.ExecContext(ctx, "PRAGMA wal_checkpoint(PASSIVE)")
	return err
}
