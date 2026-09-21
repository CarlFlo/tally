package jobs

import "fmt"

// RequestError marks an expected request/state error that is safe to return to
// an API caller. Infrastructure and persistence failures remain unwrapped.
type RequestError struct {
	Message string
}

func (e RequestError) Error() string { return e.Message }

func requestError(format string, args ...any) error {
	return RequestError{Message: fmt.Sprintf(format, args...)}
}
