package torrent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/CarlFlo/tally/internal/database"
	"github.com/CarlFlo/tally/internal/providers"
)

var ErrClientConflict = errors.New("torrent client settings changed in another tab; reload the settings before saving or testing")

var ErrNoClient = errors.New("choose and save a torrent client in Settings first")

type ClientStore struct {
	DB      *database.Store
	Control providers.Requester
}

func (s *ClientStore) Load(ctx context.Context) (ClientConfig, error) {
	c := ClientConfig{Fields: map[string]string{}}
	var raw string
	if e := s.DB.QueryRowContext(ctx, "SELECT adapter,fields,revision FROM download_client_settings WHERE id=1").Scan(&c.Adapter, &raw, &c.Revision); e != nil {
		return c, e
	}
	if e := json.Unmarshal([]byte(raw), &c.Fields); e != nil {
		return c, fmt.Errorf("stored torrent client settings are invalid")
	}
	return c, nil
}

func (s *ClientStore) Save(ctx context.Context, in ClientUpdate) (ClientConfig, error) {
	c, e := s.Prepare(ctx, in)
	if e != nil {
		return c, e
	}
	raw, e := json.Marshal(c.Fields)
	if e != nil {
		return c, e
	}
	result, e := s.DB.ExecContext(ctx, "UPDATE download_client_settings SET adapter=?,fields=?,revision=revision+1,updated_at=? WHERE id=1 AND revision=?", c.Adapter, string(raw), time.Now().Unix(), c.Revision)
	if e != nil {
		return c, e
	}
	n, e := result.RowsAffected()
	if e != nil {
		return c, e
	}
	if n != 1 {
		return c, ErrClientConflict
	}
	c.Revision++
	return c, nil
}
