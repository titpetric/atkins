package api

import (
	"encoding/json"
	"errors"
	"net/http"
)

// RequestError is an error carrying the HTTP status to report it with.
type RequestError struct {
	StatusCode int

	Err error
}

// Error returns the underlying error message.
func (r *RequestError) Error() string {
	return r.Err.Error()
}

// Unwrap exposes the wrapped error to errors.Is/As.
func (r *RequestError) Unwrap() error {
	return r.Err
}

// requestError is a shorthand for building a *RequestError.
func requestError(status int, err error) error {
	return &RequestError{StatusCode: status, Err: err}
}

// errInvalidBody is returned for anything that isn't decodable JSON.
var errInvalidBody = errors.New("invalid request body")

// decode reads a JSON request body into target.
func decode(r *http.Request, target any) error {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		return requestError(http.StatusBadRequest, errInvalidBody)
	}
	return nil
}
