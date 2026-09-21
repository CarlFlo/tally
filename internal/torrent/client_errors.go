package torrent

import "fmt"

type ClientValidationError struct {
	Message string
}

func (e ClientValidationError) Error() string { return e.Message }

func clientValidation(format string, args ...any) error {
	return ClientValidationError{Message: fmt.Sprintf(format, args...)}
}
