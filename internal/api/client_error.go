package api

import (
	"errors"

	"github.com/CarlFlo/tally/internal/torrent"
)

func clientInputError(err error) error {
	if errors.Is(err, torrent.ErrClientConflict) {
		return apiError{409, err.Error()}
	}
	var validation torrent.ClientValidationError
	if errors.As(err, &validation) || errors.Is(err, torrent.ErrNoClient) {
		return bad(err.Error())
	}
	return err
}
