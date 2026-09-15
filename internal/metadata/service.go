package metadata

import (
	"context"
	"fmt"
	"strconv"

	"golang.org/x/sync/singleflight"

	"github.com/CarlFlo/tally/internal/database"
)

type Service struct {
	DB       *database.Store
	Provider TVProvider
	OnChange func(string, ...string)
	flights  singleflight.Group
}

func (s *Service) Sync(ctx context.Context, external string) (string, error) {
	if n, e := strconv.Atoi(external); e != nil || n <= 0 {
		return "", fmt.Errorf("invalid TVmaze show ID")
	}
	v, e, _ := s.flights.Do(external, func() (any, error) { return s.sync(ctx, external) })
	if e != nil {
		return "", e
	}
	return v.(string), nil
}
