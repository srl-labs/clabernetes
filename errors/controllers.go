package errors

import "errors"

// ErrParse is the error returned when encountering issues with parsing some config items.
var ErrParse = errors.New("errParse")

// ErrInvalidData is the error returned when encountering issues with some resources data.
var ErrInvalidData = errors.New("errInvalidData")
