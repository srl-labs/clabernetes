package errors

import "errors"

// ErrPrepare is the error returned when encountering issues with prepare steps (steps that
// happen prior to clabernetes start/startLeading).
var ErrPrepare = errors.New("errPrepare")
