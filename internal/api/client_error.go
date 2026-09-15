package api

import (
	"errors"

	"github.com/CarlFlo/tally/internal/torrent"
)

func clientInputError(e error) error {
	if errors.Is(e, torrent.ErrClientConflict) {
		return apiError{409, e.Error()}
	}
	return bad(e.Error())
}
