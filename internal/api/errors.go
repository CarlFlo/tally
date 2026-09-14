package api

type apiError struct {
	Status  int
	Message string
}

func (e apiError) Error() string { return e.Message }

func bad(message string) error { return apiError{400, message} }

func remote(e error) error { return apiError{502, e.Error()} }
