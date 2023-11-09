package errors

import "errors"

// ErrUtil is the error returned when if we need to have more descriptive custom errors from util
// functions.
var ErrUtil = errors.New("errUtil")
