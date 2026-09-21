package api

const (
	internalErrorMessage = "The local operation failed. Check server logs for details."
	remoteErrorMessage   = "The external service request failed. Check server logs for details."
)

type apiError struct {
	Status  int
	Message string
}

func (e apiError) Error() string { return e.Message }

func bad(message string) error { return apiError{400, message} }

type remoteAPIError struct {
	Cause error
}

func (e remoteAPIError) Error() string { return remoteErrorMessage }

func (e remoteAPIError) Unwrap() error { return e.Cause }

func remote(err error) error { return remoteAPIError{Cause: err} }

type codedAPIError struct {
	Status  int
	Message string
	Code    string
}

func (e codedAPIError) Error() string { return e.Message }

func badCode(code, message string) error {
	return codedAPIError{Status: 400, Message: message, Code: code}
}
