package errors

import "errors"

// ErrConnectivity is the error returned when encountering issues with clabernetes connectivity.
var ErrConnectivity = errors.New("errConnectivity")

// ErrLaunch is the error returned when encountering issues with launching things in a
// clabernetes pod.
var ErrLaunch = errors.New("errLaunch")
