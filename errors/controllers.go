package errors

import "errors"

// ErrParse is the error returned when encountering issues with parsing some config items.
var ErrParse = errors.New("errParse")

// ErrInvalidData is the error returned when encountering issues with some resources data.
var ErrInvalidData = errors.New("errInvalidData")

// ErrReadiness is the error returned if/when the controller is not ready -- for example, in the
// readiness endpoint.
var ErrReadiness = errors.New("errReadiness")
