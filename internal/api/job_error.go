package api

import (
	"errors"

	"github.com/CarlFlo/tally/internal/jobs"
)

func jobRequestError(err error) error {
	var requestErr jobs.RequestError
	if errors.As(err, &requestErr) {
		return bad(requestErr.Error())
	}
	return err
}
