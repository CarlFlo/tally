package api

type apiError struct {
	Status  int
	Message string
}

func (e apiError) Error() string { return e.Message }

func bad(message string) error { return apiError{400, message} }

func remote(e error) error { return apiError{502, e.Error()} }

type codedAPIError struct {
	Status  int
	Message string
	Code    string
}

func (e codedAPIError) Error() string { return e.Message }

func badCode(code, message string) error {
	return codedAPIError{Status: 400, Message: message, Code: code}
}
