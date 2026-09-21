package metadata

import "fmt"

type QueueError struct {
	Message string
}

func (e QueueError) Error() string { return e.Message }

func queueError(format string, args ...any) error {
	return QueueError{Message: fmt.Sprintf(format, args...)}
}
